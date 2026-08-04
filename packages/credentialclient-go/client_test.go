package credentialclient

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/resources/securestore"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

type testStore struct{ value string }

func (s *testStore) Put(_, _ string, value string) error { s.value = value; return nil }
func (s *testStore) Get(_, _ string) (string, error) {
	if s.value == "" {
		return "", securestore.ErrNotFound
	}
	return s.value, nil
}
func (s *testStore) Delete(_, _ string) error { s.value = ""; return nil }

func TestInProcessProvisionAndStatusNeverNeedsSubprocess(t *testing.T) {
	authority, err := credentialauthority.NewAuthority(&testStore{})
	if err != nil {
		t.Fatal(err)
	}
	client, err := NewInProcess(InProcessOptions{Authority: authority})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Provision(context.Background(), ProvisionRequest{Identity: "vrooli/test", Field: "api-key", Value: "value-not-output"}); err != nil {
		t.Fatal(err)
	}
	status, err := client.Status(context.Background(), "vrooli/test", "api-key")
	if err != nil {
		t.Fatal(err)
	}
	if !status.Configured || status.ProviderState != "available" {
		t.Fatalf("status = %+v", status)
	}
}

func TestIPCTransportReturnsTypedUnavailableError(t *testing.T) {
	_, err := NewClient(ClientOptions{BundlePortFile: t.TempDir() + "/port", BundleTokenFile: t.TempDir() + "/token"})
	if err == nil {
		t.Fatal("expected no transport when IPC files are absent")
	}
	var unavailable ErrTransportUnavailable
	if !errors.As(err, &unavailable) {
		t.Fatalf("error = %v, want ErrTransportUnavailable", err)
	}
}

type recordingSSHRunner struct {
	args  []string
	input string
}

func (r *recordingSSHRunner) Run(_ context.Context, _ string, args []string, stdin io.Reader) ([]byte, error) {
	r.args = append([]string(nil), args...)
	data, err := io.ReadAll(stdin)
	if err != nil {
		return nil, err
	}
	r.input = string(data)
	return nil, nil
}

func TestSSHProvisionKeepsValueOutOfCommand(t *testing.T) {
	runner := &recordingSSHRunner{}
	client, err := NewClient(ClientOptions{RemoteTarget: "operator@example.test", RemoteRunner: runner})
	if err != nil {
		t.Fatal(err)
	}
	const secret = "ssh-value-must-stay-on-stdin"
	if _, err := client.Provision(context.Background(), ProvisionRequest{Identity: "vrooli/test", Field: "api-key", Value: secret}); err != nil {
		t.Fatal(err)
	}
	if strings.Join(runner.args, " ") == "" || strings.Contains(strings.Join(runner.args, " "), secret) {
		t.Fatalf("SSH command contains credential value: %q", runner.args)
	}
	if runner.input != secret {
		t.Fatalf("SSH stdin = %q, want value", runner.input)
	}
}
