package credentialclient

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/securestore"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

type testStore struct {
	value  string
	values map[string]string
}

func (s *testStore) Put(service, key, value string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[service+"\x00"+key] = value
	if !strings.HasPrefix(key, "candidate/") {
		s.value = value
	}
	return nil
}
func (s *testStore) Get(service, key string) (string, error) {
	if value, ok := s.values[service+"\x00"+key]; ok {
		return value, nil
	}
	if s.value == "" {
		return "", securestore.ErrNotFound
	}
	return s.value, nil
}
func (s *testStore) Delete(service, key string) error {
	if s.values != nil {
		delete(s.values, service+"\x00"+key)
	}
	if !strings.HasPrefix(key, "candidate/") {
		s.value = ""
	}
	return nil
}

func TestInProcessProvisionAndStatusNeverNeedsSubprocess(t *testing.T) {
	authority, err := credentialauthority.NewAuthority(&testStore{})
	if err != nil {
		t.Fatal(err)
	}
	client, err := NewInProcess(InProcessOptions{Authority: authority})
	if err != nil {
		t.Fatal(err)
	}
	provisioned, err := client.Provision(context.Background(), ProvisionRequest{Identity: "vrooli/test", Field: "api-key", Value: "value-not-output"})
	if err != nil {
		t.Fatal(err)
	}
	status, err := client.Status(context.Background(), "vrooli/test", "api-key")
	if err != nil {
		t.Fatal(err)
	}
	if !status.Configured || status.ProviderState != "available" {
		t.Fatalf("status = %+v", status)
	}
	if provisioned.Version == "" || provisioned.Version != status.Version {
		t.Fatalf("provision response version = %q, status version = %q", provisioned.Version, status.Version)
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
	status, err := client.Status(context.Background(), "vrooli/test", "api-key")
	if err != nil {
		t.Fatal(err)
	}
	result, err := hydrator.Hydrate(context.Background(), HydrationRequest{Identity: "vrooli/test", Field: "api-key", TargetID: "local", ExpectedVersion: status.Version, Env: "TEST_API_KEY", Target: target})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExposureMode != ExposureRuntimeInjection || !result.Injected || result.RestartRequired || result.NextAction != "start-consumer" || result.TargetID != "local" || result.Version != status.Version || result.LeaseID == "" || result.ExpiresAt.Before(time.Now()) || target["TEST_API_KEY"] != "one-process-secret" {
		t.Fatalf("hydration result = %+v target = %#v", result, target)
	}
}

func TestInProcessHydrateReportsRestartForRunningConsumer(t *testing.T) {
	store := &testStore{value: "running-process-secret"}
	authority, err := credentialauthority.NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	client, err := NewInProcess(InProcessOptions{Authority: authority})
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.(HydrationProvider).Hydrate(context.Background(), HydrationRequest{
		Identity: "vrooli/test", Field: "api-key", TargetID: "local", Env: "TEST_API_KEY", Target: map[string]string{}, ProcessRunning: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.RestartRequired || result.NextAction != "restart-consumer" {
		t.Fatalf("hydration result = %+v, want an explicit restart requirement", result)
	}
}

func TestInProcessHydrateRejectsChangedVersionAndUnboundedLease(t *testing.T) {
	store := &testStore{}
	authority, err := credentialauthority.NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	client, err := NewInProcess(InProcessOptions{Authority: authority})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Provision(context.Background(), ProvisionRequest{Identity: "vrooli/test", Field: "api-key", Value: "version-bound-secret"}); err != nil {
		t.Fatal(err)
	}
	hydrator := client.(HydrationProvider)
	target := map[string]string{}
	if _, err := hydrator.Hydrate(context.Background(), HydrationRequest{Identity: "vrooli/test", Field: "api-key", TargetID: "local", ExpectedVersion: "stale-version", Env: "TEST_API_KEY", Target: target}); err == nil || len(target) != 0 {
		t.Fatalf("changed-version hydration = %v target = %#v, want rejection without injection", err, target)
	}
	target["TEST_API_KEY"] = "caller-owned-value"
	if _, err := hydrator.Hydrate(context.Background(), HydrationRequest{Identity: "vrooli/test", Field: "api-key", TargetID: "local", LeaseTTL: maxHydrationLeaseTTL + time.Second, Env: "TEST_API_KEY", Target: target}); err == nil || target["TEST_API_KEY"] != "caller-owned-value" {
		t.Fatalf("unbounded hydration = %v target = %#v, want rejection without injection", err, target)
	}
}

func TestInProcessHydrateDoesNotOverwriteCallerOwnedEnvironment(t *testing.T) {
	store := &testStore{}
	authority, err := credentialauthority.NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	client, err := NewInProcess(InProcessOptions{Authority: authority})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Provision(context.Background(), ProvisionRequest{Identity: "vrooli/test", Field: "api-key", Value: "secret-value"}); err != nil {
		t.Fatal(err)
	}
	target := map[string]string{"TEST_API_KEY": "caller-owned"}
	_, err = client.(HydrationProvider).Hydrate(context.Background(), HydrationRequest{
		Identity: "vrooli/test", Field: "api-key", TargetID: "local", Env: "TEST_API_KEY", Target: target,
	})
	if err == nil || !strings.Contains(err.Error(), "already contains") {
		t.Fatalf("Hydrate error = %v, want caller-owned target rejection", err)
	}
	if target["TEST_API_KEY"] != "caller-owned" {
		t.Fatalf("target = %#v, caller-owned value was changed", target)
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
	hydrated, err := hydrator.Hydrate(context.Background(), HydrationRequest{Identity: "vrooli/test", Field: "api-key", TargetID: "local", Env: "TEST_API_KEY", Target: target})
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

func TestInProcessHydrationExpiryClearsEphemeralTarget(t *testing.T) {
	store := &testStore{}
	authority, err := credentialauthority.NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	client, err := NewInProcess(InProcessOptions{Authority: authority})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Provision(context.Background(), ProvisionRequest{Identity: "vrooli/test", Field: "api-key", Value: "expires-secret"}); err != nil {
		t.Fatal(err)
	}
	hydrator := client.(HydrationProvider)
	target := map[string]string{}
	hydrated, err := hydrator.Hydrate(context.Background(), HydrationRequest{Identity: "vrooli/test", Field: "api-key", TargetID: "local", LeaseTTL: 10 * time.Millisecond, Env: "TEST_API_KEY", Target: target})
	if err != nil {
		t.Fatal(err)
	}
	// Target is caller-owned and must not be read while the expiry callback can
	// delete from it. Wait past the bounded lease and acquire the lifecycle
	// mutex through revocation before inspecting the cleanup.
	time.Sleep(50 * time.Millisecond)
	_, _ = client.(HydrationRevocationProvider).RevokeHydration(context.Background(), HydrationRevocationRequest{LeaseID: hydrated.LeaseID})
	if target["TEST_API_KEY"] != "" {
		t.Fatalf("expired hydration retained target value %q", target["TEST_API_KEY"])
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

func TestInProcessDoctorClassifiesRecoveryFreshnessAndCoverage(t *testing.T) {
	const identity = "vrooli/test"
	const field = "api-key"

	tests := []struct {
		name          string
		receipt       bool
		receiptAge    time.Duration
		covers        bool
		wantStatus    string
		wantReason    string
		wantUncovered bool
	}{
		{name: "no receipt", wantStatus: "incomplete", wantReason: "no verified recovery receipt exists", wantUncovered: true},
		{name: "covered and fresh", receipt: true, covers: true, wantStatus: "protected", wantReason: "covers the current configured credential inventory"},
		{name: "covered but stale", receipt: true, receiptAge: recoveryFreshnessWindow + time.Hour, covers: true, wantStatus: "stale", wantReason: "older than the supported freshness window"},
		{name: "receipt misses configured credential", receipt: true, wantStatus: "incomplete", wantReason: "does not cover every current required or configured credential", wantUncovered: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := &testStore{value: "secret"}
			authority, err := credentialauthority.NewAuthority(store)
			if err != nil {
				t.Fatal(err)
			}
			stateDir := t.TempDir()
			client, err := NewInProcess(InProcessOptions{
				Authority: authority,
				StateDir:  stateDir,
				Descriptors: func() ([]CredentialRef, error) {
					return []CredentialRef{{LogicalID: identity, Field: field, Required: true}}, nil
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			if test.receipt {
				entries := []credentialauthority.RecoveryEntry{}
				if test.covers {
					entries = append(entries, credentialauthority.RecoveryEntry{Identity: credentialauthority.Identity(identity), Field: field})
				}
				exportedAt := time.Now().UTC().Add(-test.receiptAge)
				if err := credentialauthority.WriteRecoveryReceipt(stateDir, filepath.Join(stateDir, "recovery.bundle"), entries, exportedAt); err != nil {
					t.Fatal(err)
				}
			}

			response, err := client.Doctor(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if response.Recovery.Status != test.wantStatus || !strings.Contains(response.Recovery.FreshnessReason, test.wantReason) {
				t.Fatalf("recovery = %+v, want status %q and reason containing %q", response.Recovery, test.wantStatus, test.wantReason)
			}
			if test.wantUncovered != (len(response.Recovery.Uncovered) == 1) {
				t.Fatalf("uncovered = %v, want one entry: %v", response.Recovery.Uncovered, test.wantUncovered)
			}
		})
	}
}

func TestInProcessRecoveryExportRecordsVerifiedReceipt(t *testing.T) {
	authority, err := credentialauthority.NewAuthority(&testStore{value: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	stateDir := t.TempDir()
	client, err := NewInProcess(InProcessOptions{Authority: authority, StateDir: stateDir})
	if err != nil {
		t.Fatal(err)
	}
	outputPath := filepath.Join(t.TempDir(), "recovery.bundle")
	ref := CredentialRef{LogicalID: "vrooli/test", Field: "api-key"}
	if _, err := client.RecoveryExport(context.Background(), RecoveryExportRequest{Entries: []CredentialRef{ref}, Passphrase: "correct horse battery staple", OutputPath: outputPath}); err != nil {
		t.Fatal(err)
	}
	receipt, found, err := credentialauthority.ReadRecoveryReceipt(stateDir)
	if err != nil || !found {
		t.Fatalf("receipt = %+v, found = %v, err = %v", receipt, found, err)
	}
	if receipt.Verification != "decrypt-readback" || receipt.VerifiedAt.IsZero() || !receipt.Covers(credentialauthority.Identity("vrooli/test"), "api-key") {
		t.Fatalf("receipt = %+v, want verified coverage", receipt)
	}
	if _, err := credentialauthority.InspectRecovery(mustReadFile(t, outputPath), "correct horse battery staple"); err != nil {
		t.Fatalf("exported bundle does not decrypt after receipt: %v", err)
	}
	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != credentialBundleFileMode {
		t.Fatalf("recovery bundle permissions = %04o, want %04o", info.Mode().Perm(), credentialBundleFileMode)
	}
}

func TestInProcessRecoveryImportRejectsBroadPermissionsAndSymlinks(t *testing.T) {
	authority, err := credentialauthority.NewAuthority(&testStore{})
	if err != nil {
		t.Fatal(err)
	}
	client, err := NewInProcess(InProcessOptions{Authority: authority})
	if err != nil {
		t.Fatal(err)
	}
	broadPath := filepath.Join(t.TempDir(), "broad.bundle")
	if err := os.WriteFile(broadPath, []byte("bundle"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(broadPath, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := client.RecoveryVerify(context.Background(), RecoveryVerifyRequest{InputPath: broadPath, Passphrase: "passphrase"}); err == nil || !strings.Contains(err.Error(), "permissions") {
		t.Fatalf("broad-permission import error = %v, want permission rejection", err)
	}

	symlinkPath := filepath.Join(t.TempDir(), "linked.bundle")
	if err := os.Symlink(broadPath, symlinkPath); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := client.RecoveryVerify(context.Background(), RecoveryVerifyRequest{InputPath: symlinkPath, Passphrase: "passphrase"}); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("symlink import error = %v, want symlink rejection", err)
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
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
	if stdin != nil {
		data, err := io.ReadAll(stdin)
		if err != nil {
			return nil, err
		}
		r.input = string(data)
	}
	if len(args) >= 3 && args[0] == "vrooli" && args[1] == "credentials" && args[2] == "status" {
		return []byte(`{"version":"remote-version","configured":true,"provider_state":"available"}`), nil
	}
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
