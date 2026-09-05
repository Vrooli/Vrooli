package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
)

// [REQ:ONB-CRED-SCOPE-PARITY]
// Three fields decide what the credential client can answer. An empty Root
// makes the inventory empty, an empty StateDir makes the whole recovery half of
// Doctor unreachable, and a nil Descriptors makes the manifest fallback return
// nothing. Assert all three explicitly so none can be dropped silently again.
func TestCredentialClientOptionsCarryRootStateDirAndDescriptors(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("BUNDLE_ROOT", "")
	t.Setenv("VROOLI_STORAGE_ROOT", "")

	options, err := onboardingCredentialClientOptions()
	if err != nil {
		t.Fatalf("build client options: %v", err)
	}
	if options.Root != root {
		t.Fatalf("Root = %q, want %q", options.Root, root)
	}
	if options.StateDir == "" {
		t.Fatal("StateDir is empty; Doctor would skip its entire recovery block")
	}
	if options.Descriptors == nil {
		t.Fatal("Descriptors is nil; the manifest fallback would return nothing")
	}
}

// A desktop bundle has no repository and no control-plane runtime home. Leaving
// both paths empty keeps today's honest degradation instead of reading paths
// that do not exist.
func TestCredentialClientOptionsStayEmptyInBundleMode(t *testing.T) {
	bundle := t.TempDir()
	t.Setenv("VROOLI_ROOT", "")
	t.Setenv("VROOLI_STORAGE_ROOT", "")
	t.Setenv("BUNDLE_ROOT", bundle)

	options, err := onboardingCredentialClientOptions()
	if err != nil {
		t.Fatalf("build client options: %v", err)
	}
	if options.Root != "" || options.StateDir != "" || options.Descriptors != nil {
		t.Fatalf("bundle-mode options = {Root:%q StateDir:%q DescriptorsSet:%t}, want all empty", options.Root, options.StateDir, options.Descriptors != nil)
	}
}

// The Descriptors fallback must carry the project scope, because the project
// manifest is the authoritative owner of host-owned credentials that have no
// scenario directory.
func TestCredentialClientDescriptorsIncludeProjectScope(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile(t, filepath.Join(root, ".vrooli", "service.json"), `{
  "service": {"name": "vrooli", "description": "Project scope"},
  "credentials": {"descriptors": [{"logical_id": "vrooli/remote-desktop", "field": "username", "label": "Remote desktop username", "required": false}]}
}`)
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("BUNDLE_ROOT", "")
	t.Setenv("VROOLI_STORAGE_ROOT", "")

	options, err := onboardingCredentialClientOptions()
	if err != nil {
		t.Fatalf("build client options: %v", err)
	}
	refs, err := options.Descriptors()
	if err != nil {
		t.Fatalf("descriptors: %v", err)
	}
	found := false
	for _, ref := range refs {
		if ref.LogicalID == "vrooli/remote-desktop" && ref.Field == "username" {
			found = true
			if ref.Resource != credentialclient.ProjectScopeOwner {
				t.Fatalf("owner = %q, want %q", ref.Resource, credentialclient.ProjectScopeOwner)
			}
		}
	}
	if !found {
		t.Fatalf("descriptors = %v, want the project-scope address", refs)
	}
}

// [REQ:ONB-CRED-RECOVERY-VISIBLE]
// A state directory is what turns the recovery half of Doctor on. Prove both
// directions from one receipt fixture so the difference is unambiguous.
func TestDoctorRecoveryBlockNeedsAStateDirectory(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile(t, filepath.Join(root, ".vrooli", "service.json"), `{
  "service": {"name": "vrooli", "description": "Project scope"},
  "credentials": {"descriptors": [{"logical_id": "vrooli/remote-desktop", "field": "username", "required": false}]}
}`)
	stateDir := t.TempDir()
	identity, err := credentialauthority.ParseIdentity("vrooli/remote-desktop")
	if err != nil {
		t.Fatal(err)
	}
	entries := []credentialauthority.RecoveryEntry{{Identity: identity, Field: "username"}}
	if err := credentialauthority.WriteRecoveryReceipt(stateDir, filepath.Join(stateDir, "bundle.age"), entries, time.Now()); err != nil {
		t.Fatalf("write receipt: %v", err)
	}
	// A store that reports no backend is enough and is deterministic: the
	// recovery block is read from the receipt on disk, and an absent store only
	// means every credential reads as unconfigured, which is what makes the
	// uncovered list empty rather than host-dependent.
	authority, err := credentialauthority.Unavailable("recovery block test")
	if err != nil {
		t.Fatal(err)
	}

	withState, err := credentialclient.NewInProcess(credentialclient.InProcessOptions{Authority: authority, Root: root, StateDir: stateDir})
	if err != nil {
		t.Fatal(err)
	}
	report, err := withState.Doctor(context.Background())
	if err != nil {
		t.Fatalf("doctor with state directory: %v", err)
	}
	if !report.Recovery.ReceiptExists || report.Recovery.EntryCount != 1 || report.Recovery.ExportedAt == "" {
		encoded, _ := json.Marshal(report.Recovery)
		t.Fatalf("recovery with a state directory = %s, want a populated receipt", encoded)
	}

	withoutState, err := credentialclient.NewInProcess(credentialclient.InProcessOptions{Authority: authority, Root: root})
	if err != nil {
		t.Fatal(err)
	}
	blind, err := withoutState.Doctor(context.Background())
	if err != nil {
		t.Fatalf("doctor without state directory: %v", err)
	}
	if blind.Recovery.ReceiptExists || blind.Recovery.EntryCount != 0 {
		encoded, _ := json.Marshal(blind.Recovery)
		t.Fatalf("recovery without a state directory = %s, want an empty block", encoded)
	}
}
