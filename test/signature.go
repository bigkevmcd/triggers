package test

import (
	"crypto/hmac"
	"crypto/sha1" //nolint
	"encoding/hex"
	"fmt"
	"hash"
	"testing"
)

// HMACHeader generates a X-Hub-Signature header given a secret token and the request body
// See https://developer.github.com/webhooks/securing/#validating-payloads-from-github
func HMACHeader(t testing.TB, secret string, body []byte) string {
	t.Helper()
	return HMACString(t, "sha1", []byte(secret), body, sha1.New)
}

// HMACString generates a signature with a prefix.
func HMACString(t testing.TB, prefix string, secret, body []byte, hasher func() hash.Hash) string {
	t.Helper()
	h := hmac.New(hasher, secret)
	_, err := h.Write(body)
	if err != nil {
		t.Fatalf("HMACString fail: %s", err)
	}
	return fmt.Sprintf("%s=%s", prefix, hex.EncodeToString(h.Sum(nil)))
}
