package sign

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// GenerateSignature returns HMAC-SHA256(secret, payload) as a lowercase hex string.
// For replay protection, callers should sign the concatenation of timestamp + payload.
func GenerateSignature(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}
