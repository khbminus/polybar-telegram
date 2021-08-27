package main

import (
	"context"
	"fmt"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
	"github.com/gotd/td/tgerr"
	"golang.org/x/xerrors"
	"os"
	"strconv"
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

//need TG_APPID TG_APPHASH PHONE AUTH_FILE
func main() {
	sessionStorage := &MemorySession{}
	appId, err := strconv.Atoi(os.Getenv("TG_APPID"))
	if err != nil {
		panic(err)
	}
	client := telegram.NewClient(appId, os.Getenv("TG_APPHASH"), telegram.Options{
		SessionStorage: sessionStorage,
	})
	if err := client.Run(context.Background(), func(ctx context.Context) error {
		err := InvokeAuth(client, ctx)
		if err != nil {
			return err
		}
		api := client.API()
		params := tg.MessagesGetDialogsRequest{
			OffsetPeer: &tg.InputPeerEmpty{},
			OffsetID: 0,
			Limit: DialogsLimit,
		}
		sum := 0
		was := make(map[tg.PeerClass]bool)
		for flag := true; flag; {
			dialogs, err := api.MessagesGetDialogs(ctx, &params)
			for err != nil {
				switch e := xerrors.Unwrap(err).(type) {
				case *tgerr.Error:
					if !e.IsCode(420) {
						return err
					} else {
						time.Sleep(30 * time.Second)
					}
				default:
					return err
				}
				dialogs, err = api.MessagesGetDialogs(ctx, &params)
			}
			switch v := dialogs.(type) {
			case *tg.MessagesDialogsSlice:
				m := make(map[MessageMapper]*tg.Message)
				for _, t := range v.Messages {
					switch message := t.(type) {
					case *tg.Message:
						m[getMapper(message)] = message
					}
				}
				if len(v.Dialogs) < DialogsLimit {
					flag = false
					break
				}
				var lastMessage *tg.Message = nil
				for _, dd := range v.Dialogs {
					d := dd.(*tg.Dialog)
					if d.FolderID != 0 {
						continue
					}
					_, ok := was[d.Peer]
					if ok  {
						fmt.Printf("was: %v\n", d)
					}
					was[d.Peer] = true
					_, ok = m[getMapperDialog(d)]
					if (d.UnreadMark || d.UnreadCount > 0) {
						sum++
					}
				}
				for i := len(v.Dialogs) - 1; i >= 0; i-- {
					d := v.Dialogs[i].(*tg.Dialog)
					k, ok := m[getMapperDialog(d)]
					if ok {
						lastMessage = k
						break
					}
				}
				params.ExcludePinned = true
				if lastMessage != nil {
					params.OffsetID = lastMessage.ID
					params.OffsetDate = lastMessage.Date
				}
			default:
				flag = false
			}
		}
		fmt.Println(sum)

		return nil
	}); err != nil {
		panic(err)
	}
}
