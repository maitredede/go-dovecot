package doveadm

import (
	"encoding/base64"
	"fmt"
	"net/http"
)

type Auth interface {
	apply(req *http.Request) error
}

type authAPIKey struct {
	key string
}

var _ Auth = (*authAPIKey)(nil)

func (a *authAPIKey) apply(req *http.Request) error {
	req.Header.Add("Authorization", fmt.Sprintf("X-Dovecot-API %v", base64.StdEncoding.EncodeToString([]byte(a.key))))
	return nil
}

type authBasic struct {
	password string
}

var _ Auth = (*authBasic)(nil)

func (a *authBasic) apply(req *http.Request) error {
	req.Header.Add("Authorization", fmt.Sprintf("Basic %v", base64.StdEncoding.EncodeToString([]byte("doveadm:"+a.password))))
	return nil
}

func AuthWithAPIKey(key string) Auth {
	return &authAPIKey{
		key: key,
	}
}

func AuthWithPassword(password string) Auth {
	return &authBasic{
		password: password,
	}
}
