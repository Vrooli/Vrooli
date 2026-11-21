package authcli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStatusForUsesOnlyCredentialSourcePresence(t *testing.T) {
	home := t.TempDir()
	lstat := os.Lstat
	if got := statusFor(home, false, lstat); got.SignInState != StateSignedOut {
		t.Fatalf("no configured credential source = %+v, want signed_out", got)
	}

	if err := os.MkdirAll(filepath.Join(home, ".grok"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".grok", "auth.json"), []byte("opaque"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := statusFor(home, false, lstat); got.SignInState != StateUnknown {
		t.Fatalf("credential file presence cannot prove validity: %+v", got)
	}
	if got := statusFor(home, true, lstat); got.SignInState != StateUnknown {
		t.Fatalf("deployment key presence cannot prove validity: %+v", got)
	}
}

func TestStatusForMapsInspectionFailureToUnknown(t *testing.T) {
	got := statusFor(t.TempDir(), false, func(string) (os.FileInfo, error) {
		return nil, errors.New("permission denied")
	})
	if got.SignInState != StateUnknown {
		t.Fatalf("inspection failure = %+v, want unknown", got)
	}
}

func TestStatusForMapsMissingHomeToUnknown(t *testing.T) {
	got := statusFor(filepath.Join(t.TempDir(), "missing-home"), false, os.Lstat)
	if got.SignInState != StateUnknown {
		t.Fatalf("missing home = %+v, want unknown", got)
	}
}

func TestStatusCommandDoesNotEmitDeploymentKey(t *testing.T) {
	home := t.TempDir()
	const secret = "do-not-return-this"
	var out bytes.Buffer
	err := runStatusCommand([]string{"--json"}, &out,
		func() (string, error) { return home, nil },
		func(name string) (string, bool) {
			if name == "GROK_DEPLOYMENT_KEY" {
				return secret, true
			}
			return "", false
		}, os.Lstat,
	)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), secret) {
		t.Fatal("status output contains the deployment key")
	}
	var got Status
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if !got.Success || got.SignInState != StateUnknown {
		t.Fatalf("deployment key status = %+v, want unknown", got)
	}
}

func TestStatusCommandReturnsUnknownWhenHomeCannotBeResolved(t *testing.T) {
	var out bytes.Buffer
	err := runStatusCommand([]string{"--json"}, &out,
		func() (string, error) { return "", errors.New("home unavailable") },
		func(string) (string, bool) { return "", false }, os.Lstat,
	)
	if err != nil {
		t.Fatal(err)
	}
	var got Status
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if got.SignInState != StateUnknown {
		t.Fatalf("home resolution failure = %+v, want unknown", got)
	}
}
