package agentmanager

import (
	"testing"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// TestApplyAcceptanceOverride_NoOpWhenEmpty verifies that an empty
// allow/deny pair leaves runReq.InlineConfig nil. This matters because
// agent-manager's resolveSandboxConfig backfills Mode/NetworkMode from
// DefaultSandboxConfig() — emitting a non-nil but empty SandboxConfig
// would cause the orchestrator to clone the override (which is empty)
// rather than start from defaults, silently regressing protected mode.
func TestApplyAcceptanceOverride_NoOpWhenEmpty(t *testing.T) {
	runReq := &apipb.CreateRunRequest{}
	applyAcceptanceOverride(runReq, nil, nil)
	if runReq.InlineConfig != nil {
		t.Errorf("expected InlineConfig to remain nil for empty inputs; got %+v", runReq.InlineConfig)
	}

	applyAcceptanceOverride(runReq, []string{}, []string{})
	if runReq.InlineConfig != nil {
		t.Errorf("expected InlineConfig to remain nil for empty slice inputs; got %+v", runReq.InlineConfig)
	}
}

// TestApplyAcceptanceOverride_AllowOnly verifies allow paths are wired
// onto Acceptance.Allow without populating Deny.
func TestApplyAcceptanceOverride_AllowOnly(t *testing.T) {
	runReq := &apipb.CreateRunRequest{}
	applyAcceptanceOverride(runReq, []string{"src/**", "tests/**"}, nil)

	if runReq.InlineConfig == nil || runReq.InlineConfig.SandboxConfig == nil {
		t.Fatal("expected InlineConfig.SandboxConfig populated")
	}
	cfg := runReq.InlineConfig.SandboxConfig
	if cfg.Acceptance == nil {
		t.Fatal("Acceptance should be set")
	}
	if cfg.Acceptance.Mode != domainpb.SandboxAcceptanceMode_SANDBOX_ACCEPTANCE_MODE_ALLOWLIST {
		t.Errorf("Mode = %v; want ALLOWLIST", cfg.Acceptance.Mode)
	}
	if cfg.Acceptance.Allow == nil || len(cfg.Acceptance.Allow.PathGlobs) != 2 {
		t.Fatalf("Allow.PathGlobs = %+v; want 2 entries", cfg.Acceptance.Allow)
	}
	if cfg.Acceptance.Deny != nil {
		t.Errorf("Deny should be nil when no deny inputs; got %+v", cfg.Acceptance.Deny)
	}
}

// TestApplyAcceptanceOverride_OnlySetsAcceptance verifies the helper
// leaves SandboxConfig.Mode and SandboxConfig.NetworkMode at proto-zero
// so agent-manager's resolveSandboxConfig can backfill them. This is
// the load-bearing contract documented in PROTECTED_MODE_RUNNERS.md.
func TestApplyAcceptanceOverride_OnlySetsAcceptance(t *testing.T) {
	runReq := &apipb.CreateRunRequest{}
	applyAcceptanceOverride(runReq, []string{"docs/**"}, []string{"vendor/**"})

	cfg := runReq.InlineConfig.SandboxConfig
	if cfg.Mode != domainpb.SandboxMode_SANDBOX_MODE_UNSPECIFIED {
		t.Errorf("SandboxConfig.Mode = %v; want SANDBOX_MODE_UNSPECIFIED so resolver can backfill", cfg.Mode)
	}
	if cfg.NetworkMode != domainpb.NetworkAccess_NETWORK_ACCESS_UNSPECIFIED {
		t.Errorf("SandboxConfig.NetworkMode = %v; want UNSPECIFIED so resolver can backfill", cfg.NetworkMode)
	}
}

// TestApplyAcceptanceOverride_DenyOnly verifies deny paths are wired
// without populating Allow.
func TestApplyAcceptanceOverride_DenyOnly(t *testing.T) {
	runReq := &apipb.CreateRunRequest{}
	applyAcceptanceOverride(runReq, nil, []string{"vendor/**"})

	cfg := runReq.InlineConfig.SandboxConfig
	if cfg.Acceptance.Allow != nil {
		t.Errorf("Allow should be nil when no allow inputs; got %+v", cfg.Acceptance.Allow)
	}
	if cfg.Acceptance.Deny == nil || len(cfg.Acceptance.Deny.PathGlobs) != 1 {
		t.Fatalf("Deny.PathGlobs = %+v; want 1 entry", cfg.Acceptance.Deny)
	}
}
