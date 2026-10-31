package cloudtargethandlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cloudtarget"
	"github.com/vrooli/vrooli/internal/credentialauthority"
)

type memoryAuthority struct {
	values map[string]string
	locked bool
}

func (m *memoryAuthority) key(identity credentialauthority.Identity, field string) string {
	return string(identity) + "/" + field
}

func (m *memoryAuthority) Availability() error {
	if m.locked {
		return fmt.Errorf("%w: locked", credentialauthority.ErrProviderUnavailable)
	}
	return nil
}

func (m *memoryAuthority) Put(identity credentialauthority.Identity, field, value string) error {
	if err := m.Availability(); err != nil {
		return err
	}
	m.values[m.key(identity, field)] = value
	return nil
}

func (m *memoryAuthority) Delete(identity credentialauthority.Identity, field string) error {
	delete(m.values, m.key(identity, field))
	return nil
}

func (m *memoryAuthority) Status(identity credentialauthority.Identity, field string) credentialauthority.Status {
	_, ok := m.values[m.key(identity, field)]
	return credentialauthority.Status{Identity: identity, Field: field, Configured: ok, ProviderState: credentialauthority.ProviderAvailable}
}

// runCredentialVerb runs one cloud-target credential verb with a scripted
// standard input and returns the printed JSON, the exit code and the stderr.
func runCredentialVerb(t *testing.T, store *cloudtarget.Store, authority *memoryAuthority, stdin string, args ...string) (map[string]any, int, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	ctx := &rootcli.CommandContext{Root: t.TempDir(), Stdout: &stdout, Stderr: &stderr, Stdin: strings.NewReader(stdin), Context: context.Background()}
	deps := Deps{
		Store:       func() (*cloudtarget.Store, error) { return store, nil },
		Credentials: cloudtarget.CredentialDeps{Authority: func() (cloudtarget.CredentialAuthority, error) { return authority, nil }},
	}
	err := Run(ctx, deps, args)
	exit := 0
	if err != nil {
		var exitErr rootcli.ExitCodeError
		if !errors.As(err, &exitErr) {
			t.Fatalf("Run(%q) returned an untyped error: %v (stderr %s)", args, err, stderr.String())
		}
		exit = exitErr.Code
	}
	var value map[string]any
	if strings.TrimSpace(stdout.String()) != "" {
		if err := json.Unmarshal(stdout.Bytes(), &value); err != nil {
			t.Fatalf("stdout is not JSON: %s", stdout.String())
		}
	}
	return value, exit, stderr.String()
}

// [REQ:STC-P0-032] SECRET-01/SECRET-05: the CLI reads the sealed payload
// from standard input only, prints a metadata receipt, and the value never
// appears in argv, stdout or stderr.
func TestCredentialVerbsIngestAcknowledgeRevokeThroughStdin(t *testing.T) {
	store := cloudtarget.NewStore(filepath.Join(t.TempDir(), "deployments"))
	authority := &memoryAuthority{values: map[string]string{}}
	const canary = "canary-secret-3b1d7e"
	const bindingID = "dep-1/vrooli/app:db-password"
	payload, _ := json.Marshal(cloudtarget.IngestPayload{SchemaVersion: cloudtarget.IngestSchemaVersion, DeploymentID: "dep-1", BindingID: bindingID, LogicalID: "vrooli/app", Field: "db-password", Version: 1, ContentRef: "cref-1", Value: canary})
	args := []string{"credential", "ingest", "--deployment", "dep-1", "--binding", bindingID, "--version", "1", "--operation", "op-1", "--step", "ingest-v1", "--fence", "4"}
	if strings.Contains(strings.Join(args, " "), canary) {
		t.Fatal("test argv carries the value")
	}
	value, exit, stderr := runCredentialVerb(t, store, authority, string(payload), args...)
	if exit != 0 {
		t.Fatalf("ingest exit=%d value=%v stderr=%s", exit, value, stderr)
	}
	receipt, _ := value["receipt"].(map[string]any)
	if receipt["outcome"] != string(cloudtarget.OutcomeSucceeded) || receipt["fence"] != float64(4) {
		t.Fatalf("receipt = %v", receipt)
	}
	printed, _ := json.Marshal(value)
	if strings.Contains(string(printed), canary) || strings.Contains(stderr, canary) {
		t.Fatal("the value leaked into stdout or stderr")
	}
	if authority.values["vrooli/app/db-password"] != canary {
		t.Fatalf("authority holds %q", authority.values["vrooli/app/db-password"])
	}

	value, exit, _ = runCredentialVerb(t, store, authority, "", "credential", "acknowledge", "--deployment", "dep-1", "--binding", bindingID, "--version", "1", "--consumer", "scenario:app", "--operation", "op-1", "--step", "ack-app", "--fence", "4")
	if exit != 0 || value["receipt"].(map[string]any)["details"].(map[string]any)["verified"] != true {
		t.Fatalf("acknowledge exit=%d value=%v", exit, value)
	}

	value, exit, _ = runCredentialVerb(t, store, authority, "", "credential", "revoke", "--deployment", "dep-1", "--binding", bindingID, "--version", "1", "--operation", "op-2", "--step", "revoke-v1", "--fence", "5")
	if exit != 0 {
		t.Fatalf("revoke exit=%d value=%v", exit, value)
	}
	details := value["receipt"].(map[string]any)["details"].(map[string]any)
	if details["store_purged"] != true || details["limitations"] == nil {
		t.Fatalf("revoke details = %v", details)
	}
	if _, held := authority.values["vrooli/app/db-password"]; held {
		t.Fatal("value still held after revoke")
	}

	// A stale fence is refused before the store is touched.
	value, exit, _ = runCredentialVerb(t, store, authority, string(payload), "credential", "ingest", "--deployment", "dep-1", "--binding", bindingID, "--version", "1", "--operation", "op-0", "--step", "ingest-v1", "--fence", "1")
	if exit != cloudtarget.ExitRefused || value["error"].(map[string]any)["code"] != cloudtarget.CodeFenceStale {
		t.Fatalf("stale fence exit=%d value=%v", exit, value)
	}
}

// [REQ:STC-P0-032] SECRET-07: a locked node store is a typed refusal with a
// next action and no write.
func TestCredentialIngestRefusesLockedStore(t *testing.T) {
	store := cloudtarget.NewStore(filepath.Join(t.TempDir(), "deployments"))
	authority := &memoryAuthority{values: map[string]string{}, locked: true}
	const bindingID = "dep-1/vrooli/app:api-token"
	payload, _ := json.Marshal(cloudtarget.IngestPayload{SchemaVersion: cloudtarget.IngestSchemaVersion, DeploymentID: "dep-1", BindingID: bindingID, LogicalID: "vrooli/app", Field: "api-token", Version: 1, ContentRef: "cref-1", Value: "canary-secret-locked"})
	value, exit, _ := runCredentialVerb(t, store, authority, string(payload), "credential", "ingest", "--deployment", "dep-1", "--binding", bindingID, "--version", "1", "--operation", "op-1", "--step", "ingest-v1", "--fence", "1")
	if exit != cloudtarget.ExitRefused {
		t.Fatalf("exit=%d value=%v", exit, value)
	}
	errValue, _ := value["error"].(map[string]any)
	if errValue["code"] != cloudtarget.CodeCredentialStoreLocked {
		t.Fatalf("error = %v", errValue)
	}
	if len(authority.values) != 0 {
		t.Fatal("locked store received a write")
	}
	if strings.Contains(fmt.Sprint(value), "canary-secret-locked") {
		t.Fatal("the value leaked into the refusal")
	}
}
