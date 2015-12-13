package pakpos

import (
	//	"fmt"
	"strings"
	"time"
)

var _tokenSessions map[string]*TokenSession

type TokenSession struct {
	UserID  string
	Token   string
	Created time.Time
	Expiry  time.Time
}

func defaultTokenExpiry() time.Duration {
	return 8 * time.Hour
}

func tokenSessions() map[string]*TokenSession {
	if _tokenSessions == nil {
		_tokenSessions = map[string]*TokenSession{}
	}
	return _tokenSessions
}

func NewTokenSession(userid, password string) *TokenSession {
	userid = strings.ToLower(userid)
	if _, exist := tokenSessions()[userid]; exist == false {
		return nil
	}

	t := new(TokenSession)
	t.Created = time.Now()
	t.Expiry = time.Now().Add(defaultTokenExpiry())
	tokenSessions()[userid] = t
	return t
}

func GetToken(userid, token string) *TokenSession {
	userid = strings.ToLower(userid)
	if ts, exist := tokenSessions()[userid]; !exist {
		return nil
	} else {
		if ts.Token == token {
			ts.Expiry = time.Now().Add(defaultTokenExpiry())
			return ts
		}
	}
	return nil
}
