package test

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"testing"
)

// GitHubSignature returns a string suitable for using in the
// X-Hub-Signature header for requests that are testing responses to GitHub hook
// events.
func GitHubSignature(t *testing.T, secret, payload []byte) string {
	t.Helper()

	mac := hmac.New(sha1.New, secret)
	_, err := mac.Write(payload)
	if err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("sha1=%s", hex.EncodeToString(mac.Sum(nil)))
}
