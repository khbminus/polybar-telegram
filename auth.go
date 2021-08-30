package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
	"github.com/mitchellh/go-homedir"
	"golang.org/x/crypto/ssh/terminal"
	"io/fs"
	"io/ioutil"
	"os"
	"strings"
	"sync"
)

type noSignUp struct{}
type AuthData struct {
	noSignUp

	phone string
}
type NotAuthorizedError struct{}

func (m *NotAuthorizedError) Error() string {
	return "Not Authorized"
}

func (a AuthData) AcceptTermsOfService(ctx context.Context, tos tg.HelpTermsOfService) error {
	panic("implement me")
}

func (a AuthData) SignUp(ctx context.Context) (auth.UserInfo, error) {
	panic("implement me")
}

func (a AuthData) Phone(_ context.Context) (string, error) {
	return a.phone, nil
}

func (a AuthData) Password(_ context.Context) (string, error) {
	fmt.Print("Enter 2FA password: ")
	bytePwd, err := terminal.ReadPassword(0)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(bytePwd)), nil
}

func (a AuthData) Code(_ context.Context, _ *tg.AuthSentCode) (string, error) {
	fmt.Print("Enter code: ")
	code, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(code), nil
}

func InvokeAuth(client *telegram.Client, ctx context.Context, needAuth bool) error {
	flow := auth.NewFlow(AuthData{
		phone: os.Getenv("PHONE"),
	}, auth.SendCodeOptions{})
	if !needAuth {
		if status, err := client.Auth().Status(ctx); err != nil {
			return err
		} else if !status.Authorized {
			return &NotAuthorizedError{}
		}
		return nil
	}
	return client.Auth().IfNecessary(ctx, flow)
}

type MemorySession struct {
	Mux  sync.RWMutex `json:"mux"`
	Data []byte       `json:"data"`
}

func (s *MemorySession) LoadSession(context.Context) ([]byte, error) {
	if s == nil {
		return nil, session.ErrNotFound
	}

	s.Mux.RLock()
	defer s.Mux.RUnlock()
	path, err := homedir.Expand(os.Getenv("AUTH_FILE"))
	if err != nil {
		return nil, err
	}
	file, err := ioutil.ReadFile(path)
	switch err.(type) {
	case *fs.PathError:
		file = []byte("{}")
		err = ioutil.WriteFile(path, file, 0644)
	case nil:
		break
	default:
		return nil, err
	}
	err = json.Unmarshal(file, s)
	if err != nil {
		return nil, err
	}
	if len(s.Data) == 0 {
		return nil, session.ErrNotFound
	}

	cpy := append([]byte(nil), s.Data...)

	return cpy, nil
}

func (s *MemorySession) StoreSession(ctx context.Context, data []byte) error {
	s.Mux.Lock()
	defer s.Mux.Unlock()
	s.Data = data

	file, err := json.MarshalIndent(s, "", " ")
	if err != nil {
		return err
	}
	path, err := homedir.Expand(os.Getenv("AUTH_FILE"))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path, file, 0644)
	return err
}
