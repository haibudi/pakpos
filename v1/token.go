package pakpos

import (
	"github.com/eaciit/toolkit"
	"time"
)

type Token struct {
	TokenType, Secret, Reference1 string
	Created, Expiry               time.Time
}

var _defaultTokenExpiry time.Duration

func SetTokenDefaultExpiry(d time.Duration) time.Duration {
	_defaultTokenExpiry = d
	return _defaultTokenExpiry
}

func TokenDefaultExpiry() time.Duration {
	if int(_defaultTokenExpiry) == 0 {
		_defaultTokenExpiry = 8 * time.Hour
	}
	return _defaultTokenExpiry
}

func NewToken(tokentype, reference1 string) *Token {
	t := new(Token)
	t.Secret = toolkit.RandomString(32)
	t.Reference1 = reference1
	t.TokenType = tokentype
	t.Created = time.Now()
	t.Expiry = t.Created.Add(TokenDefaultExpiry())
	return t
}

func (t *Token) IsExpired() bool {
	return t.Expiry.Before(time.Now())
}
