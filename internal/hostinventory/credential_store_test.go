package hostinventory

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/shell/shelltest"
)

type credentialStoreCommandFixture struct {
	ownerOutput      string
	collections      string
	collectionsError error
	loginOutput      string
	loginError       error
}

func credentialStoreFixture(f credentialStoreCommandFixture) Collector {
	return Collector{Commands: &shelltest.Fake{
		LookPathFunc: func(file string) (string, error) {
			if file == "gdbus" || file == "id" {
				return "/usr/bin/gdbus", nil
			}
			return "", errors.New("not found")
		},
		RunFunc: func(_ context.Context, name string, args ...string) ([]byte, error) {
			if name == "id" {
				return []byte("1000\n"), nil
			}
			joined := strings.Join(args, " ")
			switch {
			case strings.Contains(joined, "NameHasOwner"):
				return []byte(f.ownerOutput), nil
			case strings.Contains(joined, "org.freedesktop.Secret.Service Collections"):
				return []byte(f.collections), f.collectionsError
			case strings.Contains(joined, "/collection/login"):
				return []byte(f.loginOutput), f.loginError
			default:
				return nil, errors.New("unexpected credential-store probe: " + joined)
			}
		},
	}, GOOS: "linux"}
}

func TestProbeCredentialStoreStates(t *testing.T) {
	tests := []struct {
		name  string
		fix   credentialStoreCommandFixture
		state string
	}{
		{
			name:  "owner absent is unavailable",
			fix:   credentialStoreCommandFixture{ownerOutput: "(false,)"},
			state: "unavailable",
		},
		{
			name:  "readable login collection is ready",
			fix:   credentialStoreCommandFixture{ownerOutput: "(true,)", collections: "/org/freedesktop/secrets/collection/login"},
			state: "ready",
		},
		{
			name:  "owner with no login collection is empty",
			fix:   credentialStoreCommandFixture{ownerOutput: "(true,)", collections: "(@ao, [])"},
			state: "empty",
		},
		{
			name: "owner with unreadable login collection is locked",
			fix: credentialStoreCommandFixture{
				ownerOutput: "(true,)", collectionsError: errors.New("Collections: access denied"),
				loginError: errors.New("GDBus.Error: org.freedesktop.DBus.Error.AccessDenied"),
			},
			state: "locked",
		},
		{
			name: "timed out collection is unresponsive",
			fix: credentialStoreCommandFixture{
				ownerOutput: "(true,)", collectionsError: context.DeadlineExceeded,
			},
			state: "unresponsive",
		},
		{
			name: "missing login object is empty",
			fix: credentialStoreCommandFixture{
				ownerOutput: "(true,)", collectionsError: errors.New("Collections failed"),
				loginOutput: "UnknownObject: no such object path",
				loginError:  errors.New("object missing"),
			},
			state: "empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := probeCredentialStore(context.Background(), credentialStoreFixture(tt.fix), "alice")
			if got.State != tt.state {
				t.Fatalf("state = %q, want %q; capability = %+v", got.State, tt.state, got)
			}
			if strings.Contains(got.Reason, "passphrase") && tt.state != "locked" {
				t.Fatalf("non-locked state leaked a locked-store remedy: %q", got.Reason)
			}
		})
	}
}
