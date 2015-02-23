package cli

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
)

// GetHmac256 will generate a HMAC hash encoded to base64
func GetHmac256(message string, secret string) string {
	key := []byte(secret)
	h := hmac.New(sha256.New, key)
	h.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}
