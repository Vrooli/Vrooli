package handlers_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"workspace-sandbox/internal/handlers"
)

type n8nCompanionOwnerFake struct {
	startReceipt                                   handlers.N8NCompanionReceipt
	startErr                                       error
	stopErr                                        error
	starts                                         int
	stops                                          int
	gotSandbox                                     uuid.UUID
	gotName, gotVersion, gotDatabase, gotRuntimeID string
}

func (f *n8nCompanionOwnerFake) StartPostgres(_ context.Context, sandboxID uuid.UUID, name, version, database string) (handlers.N8NCompanionReceipt, error) {
	f.starts++
	f.gotSandbox, f.gotName, f.gotVersion, f.gotDatabase = sandboxID, name, version, database
	return f.startReceipt, f.startErr
}

func (f *n8nCompanionOwnerFake) StopCompanion(_ context.Context, sandboxID uuid.UUID, runtimeID string) error {
	f.stops++
	f.gotSandbox, f.gotRuntimeID = sandboxID, runtimeID
	return f.stopErr
}

func TestN8NCompanionProviderStartsOnlyAttestedPostgres(t *testing.T) {
	sandboxID := uuid.New()
	owner := &n8nCompanionOwnerFake{startReceipt: handlers.N8NCompanionReceipt{
		SandboxID: sandboxID.String(), CompanionRuntimeID: "sandbox.postgres", NetworkAlias: "companion-12345678-abcdef012345", Status: "attached", NoEgressVerified: true,
	}}
	provider := handlers.N8NCompanionServiceProvider{Owner: owner}
	runtime, err := provider.Start(context.Background(), sandboxID, handlers.CompanionServiceSpec{Name: "pg", Kind: "postgres", Version: "16.4", Options: map[string]string{"database": "workflow"}})
	if err != nil {
		t.Fatal(err)
	}
	if owner.starts != 1 || owner.gotSandbox != sandboxID || owner.gotName != "pg" || owner.gotVersion != "16.4" || owner.gotDatabase != "workflow" {
		t.Fatalf("owner start args = sandbox:%s name:%q version:%q database:%q calls:%d", owner.gotSandbox, owner.gotName, owner.gotVersion, owner.gotDatabase, owner.starts)
	}
	if runtime.ID != "sandbox.postgres" || runtime.Address != "companion-12345678-abcdef012345:5432" {
		t.Fatalf("runtime = %+v", runtime)
	}
	if _, err := provider.Start(context.Background(), sandboxID, handlers.CompanionServiceSpec{Name: "cache", Kind: "redis", Version: "7.2.0"}); err == nil {
		t.Fatal("expected unsupported service kind to fail closed")
	}
	if owner.starts != 1 {
		t.Fatalf("unsupported kind reached owner: starts=%d", owner.starts)
	}
}

func TestN8NCompanionProviderUsesOwnerFixedDatabaseByDefault(t *testing.T) {
	sandboxID := uuid.New()
	owner := &n8nCompanionOwnerFake{startReceipt: handlers.N8NCompanionReceipt{
		SandboxID: sandboxID.String(), CompanionRuntimeID: "sandbox.postgres", NetworkAlias: "companion-12345678-abcdef012345", Status: "attached", NoEgressVerified: true,
	}}
	provider := handlers.N8NCompanionServiceProvider{Owner: owner}
	if _, err := provider.Start(context.Background(), sandboxID, handlers.CompanionServiceSpec{Name: "pg", Kind: "postgres", Version: "16.4"}); err != nil {
		t.Fatal(err)
	}
	if owner.gotDatabase != "workflow" {
		t.Fatalf("default database = %q, want owner-pinned database %q", owner.gotDatabase, "workflow")
	}
}

func TestN8NCompanionProviderRejectsUnattachedOrForeignReceiptAndCleansUp(t *testing.T) {
	sandboxID := uuid.New()
	owner := &n8nCompanionOwnerFake{startReceipt: handlers.N8NCompanionReceipt{
		SandboxID: sandboxID.String(), CompanionRuntimeID: "sandbox.postgres", NetworkAlias: "public.example.com", Status: "attached", NoEgressVerified: true,
	}}
	provider := handlers.N8NCompanionServiceProvider{Owner: owner}
	if _, err := provider.Start(context.Background(), sandboxID, handlers.CompanionServiceSpec{Name: "pg", Kind: "postgres", Version: "16.4"}); err == nil {
		t.Fatal("expected untrusted alias to fail closed")
	}
	if owner.stops != 1 || owner.gotRuntimeID != "sandbox.postgres" {
		t.Fatalf("invalid owner receipt cleanup: %+v", owner)
	}
	owner.startReceipt.NetworkAlias = "companion-12345678-abcdef012345"
	owner.startReceipt.NoEgressVerified = false
	if _, err := provider.Start(context.Background(), sandboxID, handlers.CompanionServiceSpec{Name: "pg", Kind: "postgres", Version: "16.4"}); err == nil {
		t.Fatal("expected missing no-egress attestation to fail closed")
	}
	owner.startReceipt.NoEgressVerified = true
	owner.startReceipt.SandboxID = uuid.NewString()
	if _, err := provider.Start(context.Background(), sandboxID, handlers.CompanionServiceSpec{Name: "pg", Kind: "postgres", Version: "16.4"}); err == nil {
		t.Fatal("expected foreign receipt to fail closed")
	}
}

func TestN8NCompanionProviderStopUsesOwnerAndRequiresRuntimeID(t *testing.T) {
	sandboxID := uuid.New()
	owner := &n8nCompanionOwnerFake{}
	provider := handlers.N8NCompanionServiceProvider{Owner: owner}
	state := handlers.CompanionServiceState{Spec: handlers.CompanionServiceSpec{Name: "pg", Kind: "postgres", Version: "16.4"}, RuntimeID: "sandbox.postgres"}
	if err := provider.Stop(context.Background(), sandboxID, state); err != nil {
		t.Fatal(err)
	}
	if owner.stops != 1 || owner.gotSandbox != sandboxID || owner.gotRuntimeID != state.RuntimeID {
		t.Fatalf("stop args mismatch: %+v", owner)
	}
	state.RuntimeID = ""
	if err := provider.Stop(context.Background(), sandboxID, state); err == nil {
		t.Fatal("expected empty runtime ID to be rejected")
	}
}

func TestN8NCompanionCLIUsesFixedArgvAndBoundedOutput(t *testing.T) {
	root := t.TempDir()
	argsFile := filepath.Join(root, "args")
	binary := filepath.Join(root, "fake-vrooli")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > '" + argsFile + "'\nprintf '%s\\n' '{\"sandboxId\":\"" + uuid.NewString() + "\",\"companionRuntimeId\":\"sandbox.postgres\",\"networkAlias\":\"companion-12345678-abcdef012345\",\"status\":\"attached\",\"noEgressVerified\":true}'\n"
	if err := os.WriteFile(binary, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	provider := handlers.N8NCompanionCLI{Binary: binary}
	sandboxID := uuid.New()
	// The fake reports an intentionally wrong ID: decoding is tested here;
	// provider-level identity validation is covered in its own test.
	receipt, err := provider.StartPostgres(context.Background(), sandboxID, "pg", "16.4", "workflow")
	if err != nil {
		t.Fatal(err)
	}
	if receipt.CompanionRuntimeID != "sandbox.postgres" {
		t.Fatalf("receipt = %+v", receipt)
	}
	args, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	argv := string(args)
	for _, expected := range []string{"resource\nn8n\nruntime\ncompanion-start\n--sandbox-id\n" + sandboxID.String(), "--companion-name\npg", "--service-version\n16.4", "--database\nworkflow", "--json"} {
		if !strings.Contains(argv, expected) {
			t.Fatalf("argv missing %q: %s", expected, argv)
		}
	}
	if strings.Contains(argv, "sh -c") {
		t.Fatalf("command must not invoke a shell: %s", argv)
	}
	if err := provider.StopCompanion(context.Background(), sandboxID, "sandbox.postgres"); err != nil {
		t.Fatal(err)
	}
	args, err = os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(args), "runtime\ncompanion-stop\n--sandbox-id\n"+sandboxID.String()) || !strings.Contains(string(args), "--companion-runtime-id\nsandbox.postgres") {
		t.Fatalf("stop argv did not use bounded owner operation: %s", args)
	}
	if _, err := (handlers.N8NCompanionCLI{}).StartPostgres(context.Background(), sandboxID, "pg", "16.4", "workflow"); err == nil {
		t.Fatal("expected missing binary to fail closed")
	}
}

func TestN8NCompanionCLIRejectsOversizedOutput(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "fake-vrooli")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\ndd if=/dev/zero bs=4096 count=10 2>/dev/null\n"), 0700); err != nil {
		t.Fatal(err)
	}
	provider := handlers.N8NCompanionCLI{Binary: binary}
	if _, err := provider.StartPostgres(context.Background(), uuid.New(), "pg", "16.4", "workflow"); err == nil {
		t.Fatal("expected oversized owner output to fail")
	}
}
