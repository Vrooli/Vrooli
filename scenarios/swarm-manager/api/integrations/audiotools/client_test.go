package audiotools

import (
	"testing"
	"time"
)

type staticResolver string

func (r staticResolver) Resolve() (string, error) {
	return string(r), nil
}

func TestNewDefaultsPerCallTimeoutForLongAudioToolsCalls(t *testing.T) {
	client, err := New(staticResolver("http://127.0.0.1:1"), Policy{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if client.policy.PerCallTimeout != 150*time.Second {
		t.Fatalf("PerCallTimeout = %v, want 150s", client.policy.PerCallTimeout)
	}
	if client.http.Timeout != 150*time.Second {
		t.Fatalf("http timeout = %v, want 150s", client.http.Timeout)
	}
}
