package approveddependencies

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/permissions"
)

func writeRegistry(t *testing.T, root, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(SourcePath))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestPermissionProviderExposesPrincipalBoundReadOnlySnapshot(t *testing.T) {
	root := t.TempDir()
	writeRegistry(t, root, validRegistry)
	provider := NewPermissionProvider(New(root))
	registry, err := permissions.NewRegistry(provider)
	if err != nil {
		t.Fatal(err)
	}
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "operator:alice", Verified: true})
	entries, err := registry.Inventory(ctx)
	if err != nil || len(entries) != 1 || entries[0].CanWrite || entries[0].Owner != PermissionOwner || entries[0].PrincipalID != "operator:alice" {
		t.Fatalf("inventory=%+v err=%v", entries, err)
	}
	read, err := registry.Read(ctx, PermissionID, permissions.ReadRequest{ActorID: "operator:alice", Owner: PermissionOwner})
	if err != nil {
		t.Fatal(err)
	}
	var snapshot Snapshot
	if err := json.Unmarshal(read.Value, &snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.Principal != "operator:alice" || len(snapshot.Records) != 1 || snapshot.SourceDigest == "" || read.Revision == "" {
		t.Fatalf("read snapshot=%+v result=%+v", snapshot, read)
	}
	_, err = registry.Write(ctx, PermissionID, permissions.WriteRequest{ActorID: "operator:alice", Owner: PermissionOwner, ExpectedRevision: read.Revision, IdempotencyKey: "forbidden", Value: []byte(`{}`)})
	if err == nil {
		t.Fatal("read-only write was accepted")
	}
}

const validRegistry = `{"schema_version":"1","version":"2026-06-16","guidance":"review with evidence","policy":{"mode":"advisory"},"records":[{"ecosystem":"npm","package_name":"example","version_range":"^1.2.0","state":"approved","rationale":"needed","approved_by":"operator","security_notes":"scanned"}]}`

func TestReadReturnsPrincipalBoundReviewedSnapshot(t *testing.T) {
	root := t.TempDir()
	writeRegistry(t, root, validRegistry)
	s, err := New(root).Read(context.Background(), "operator:alice")
	if err != nil {
		t.Fatal(err)
	}
	if s.Principal != "operator:alice" || s.Source != SourcePath || !strings.HasPrefix(s.SourceDigest, "sha256:") || len(s.Records) != 1 || s.Records[0].ApprovedBy != "operator" {
		t.Fatalf("snapshot lost principal/source/review metadata: %+v", s)
	}
	if err := New(root).RefuseMutation(context.Background(), s.Principal, []byte(`{}`)); !errors.Is(err, ErrReadOnly) {
		t.Fatalf("mutation refusal = %v", err)
	}
}

func TestReadFailsClosed(t *testing.T) {
	for name, body := range map[string]string{"malformed": "{", "unsupported": strings.Replace(validRegistry, `"1"`, `"2"`, 1), "incomplete-record": `{"schema_version":"1","version":"v","policy":{"mode":"advisory"},"records":[{}]}`} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeRegistry(t, root, body)
			if _, err := New(root).Read(context.Background(), "operator"); err == nil {
				t.Fatal("expected fail-closed error")
			}
		})
	}
	if _, err := New(t.TempDir()).Read(context.Background(), " "); err == nil {
		t.Fatal("empty principal accepted")
	}
}
