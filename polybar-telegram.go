// Package main: Application that parses telegram dialogs and get numbers of unread
// You should set some environment variables:
// TG_APPID: your app ID from https://my.telegram.org/apps
// TG_APPHASH: your app hash from https://my.telegram.org/apps
// PHONE: phone number of account that you use
// AUTH_FILE: file to save authentication data
package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
	"github.com/gotd/td/tgerr"
	"golang.org/x/xerrors"
	"os"
	"strconv"
	"text/template"
	"time"
)

type MessageMapper struct {
	PeerID    int
	MessageID int
}

func getMapper(message *tg.Message) MessageMapper {
	switch v := message.PeerID.(type) {
	case *tg.PeerChannel:
		return MessageMapper{PeerID: v.ChannelID, MessageID: message.ID}
	default:
		return MessageMapper{PeerID: -1, MessageID: message.ID}
	}
}

func getMapperDialog(dialog *tg.Dialog) MessageMapper {
	switch v := dialog.Peer.(type) {
	case *tg.PeerChannel:
		return MessageMapper{PeerID: v.ChannelID, MessageID: dialog.TopMessage}
	default:
		return MessageMapper{PeerID: -1, MessageID: dialog.TopMessage}
	}
}

const DialogsLimit = 100

func main() {
	firstAuth := flag.Bool("auth", false, "perform authorization")
	onlyUnmuted := flag.Bool("onlyUnmuted", false, "count only unmuted dialogs")
	outputFormat := flag.String("format", "{{.unread}}/{{.mentions}}", "output format")
	flag.Parse()

	sessionStorage := &MemorySession{}
	appId, err := strconv.Atoi(os.Getenv("TG_APPID"))
	if err != nil {
		panic(err)
	}
	client := telegram.NewClient(appId, os.Getenv("TG_APPHASH"), telegram.Options{
		SessionStorage: sessionStorage,
	})
	nowTime := time.Now()

	if err := client.Run(context.Background(), func(ctx context.Context) error {
		err := InvokeAuth(client, ctx, *firstAuth)
		if err != nil {
			return err
		}
		if *firstAuth {
			fmt.Println("Successfully logged in.")
			return nil
		}
		api := client.API()
		params := tg.MessagesGetDialogsRequest{
			OffsetPeer: &tg.InputPeerEmpty{},
			OffsetID:   0,
			Limit:      DialogsLimit,
		}
		sumUnread, sumMentions := 0, 0
		for isEnded := true; isEnded; {
			_dialogs, err := api.MessagesGetDialogs(ctx, &params)
			for err != nil {
				if e, ok := xerrors.Unwrap(err).(*tgerr.Error); ok {
					if !e.IsCode(420) {
						return err
					} else {
						time.Sleep(3 * time.Second)
					}
				} else {
					return err
				}

				_dialogs, err = api.MessagesGetDialogs(ctx, &params)
			}

			if dialogs, ok := _dialogs.(*tg.MessagesDialogsSlice); ok {
				messageByDialog := make(map[MessageMapper]*tg.Message)
				for _, _message := range dialogs.Messages {
					if message, ok := _message.(*tg.Message); ok {
						messageByDialog[getMapper(message)] = message
					}
				}

				if len(dialogs.Dialogs) < DialogsLimit {
					isEnded = false
					break
				}

				var lastMessage *tg.Message = nil
				for _, _dialog := range dialogs.Dialogs {
					if dialog, ok := _dialog.(*tg.Dialog); ok {
						if dialog.FolderID != 0 {
							continue
						}
						if (!*onlyUnmuted ||
							!time.Unix(int64(dialog.NotifySettings.MuteUntil), 0).After(nowTime)) &&
							(dialog.UnreadMark || dialog.UnreadCount > 0) {
							sumUnread++
						}
						if dialog.UnreadMentionsCount > 0 { // If you have mentioned, that is something important, so you need to read this
							sumMentions++
						}
					}
				}

				for i := len(dialogs.Dialogs) - 1; i >= 0; i-- {
					if dialog, ok := dialogs.Dialogs[i].(*tg.Dialog); ok {
						message, ok := messageByDialog[getMapperDialog(dialog)]
						if ok {
							lastMessage = message
							break
						}
					}
				}

				params.ExcludePinned = true
				if lastMessage != nil {
					params.OffsetID = lastMessage.ID
					params.OffsetDate = lastMessage.Date
				}
			} else {
				isEnded = false
			}
		}
		t := template.Must(template.New("").Parse(*outputFormat))
		err = t.Execute(os.Stdout, map[string]interface{}{
			"unread":   sumUnread,
			"mentions": sumMentions,
		})
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		panic(err)
	}
}
