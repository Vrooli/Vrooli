package credentialclient

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/securestore"
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
	value, err := client.Resolve(context.Background(), "vrooli/test", "api-key")
	if err != nil || value != "value-not-output" {
		t.Fatalf("Resolve() = %q, %v", value, err)
	}
}

func TestInProcessHydrateLabelsExplicitRuntimeInjection(t *testing.T) {
	store := &testStore{}
	authority, err := credentialauthority.NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	client, err := NewInProcess(InProcessOptions{Authority: authority})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Provision(context.Background(), ProvisionRequest{Identity: "vrooli/test", Field: "api-key", Value: "one-process-secret"}); err != nil {
		t.Fatal(err)
	}
	hydrator, ok := client.(HydrationProvider)
	if !ok {
		t.Fatal("in-process client must expose the explicit hydration seam")
	}
	target := map[string]string{}
	result, err := hydrator.Hydrate(context.Background(), HydrationRequest{Identity: "vrooli/test", Field: "api-key", Env: "TEST_API_KEY", Target: target})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExposureMode != ExposureRuntimeInjection || !result.Injected || result.LeaseID == "" || result.ExpiresAt.Before(time.Now()) || target["TEST_API_KEY"] != "one-process-secret" {
		t.Fatalf("hydration result = %+v target = %#v", result, target)
	}
}

func TestInProcessHydrationRevocationSeparatesFutureDeliveryFromProcessExposure(t *testing.T) {
	store := &testStore{value: "one-process-secret"}
	authority, err := credentialauthority.NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	client, err := NewInProcess(InProcessOptions{Authority: authority})
	if err != nil {
		t.Fatal(err)
	}
	hydrator := client.(HydrationProvider)
	revoker := client.(HydrationRevocationProvider)
	target := map[string]string{}
	hydrated, err := hydrator.Hydrate(context.Background(), HydrationRequest{Identity: "vrooli/test", Field: "api-key", Env: "TEST_API_KEY", Target: target})
	if err != nil {
		t.Fatal(err)
	}
	result, err := revoker.RevokeHydration(context.Background(), HydrationRevocationRequest{LeaseID: hydrated.LeaseID, ProcessRunning: true})
	if err != nil {
		t.Fatal(err)
	}
	if !result.FutureDeliveryStopped || result.ProcessExposure != "already_running_process" || target["TEST_API_KEY"] != "" {
		t.Fatalf("revocation result = %+v target = %#v", result, target)
	}
}

func TestInProcessResolvePreservesUnconfiguredTaxonomy(t *testing.T) {
	authority, err := credentialauthority.NewAuthority(&testStore{})
	if err != nil {
		t.Fatal(err)
	}
	client, err := NewInProcess(InProcessOptions{Authority: authority})
	if err != nil {
		t.Fatal(err)
	}
	value, err := client.Resolve(context.Background(), "vrooli/test", "api-key")
	if value != "" || !errors.Is(err, credentialauthority.ErrUnconfigured) {
		t.Fatalf("Resolve() = %q, %v, want empty and ErrUnconfigured", value, err)
	}
	var resolutionErr *credentialauthority.ResolutionError
	if !errors.As(err, &resolutionErr) {
		t.Fatalf("Resolve() error type = %T, want *ResolutionError", err)
	}
}

func TestUnavailableTransportReturnsTypedError(t *testing.T) {
	_, err := NewClient(ClientOptions{})
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
