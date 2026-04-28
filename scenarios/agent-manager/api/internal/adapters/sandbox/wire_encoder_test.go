// Tests for the wire-encoder that translates agent-manager domain
// SandboxConfig into the JSON shape workspace-sandbox decodes as
// SandboxBehavior. The protected-mode git allowlist is materialized here
// so workspace-sandbox can enforce it on /exec.
//
// See execute/protected-sandbox-git-and-network-guardrails.

package sandbox

import (
	"reflect"
	"testing"

	"agent-manager/internal/domain"
)

func TestEncodeBehaviorForWire_NilReturnsNil(t *testing.T) {
	if got := encodeBehaviorForWire(nil); got != nil {
		t.Fatalf("encodeBehaviorForWire(nil) = %v, want nil", got)
	}
}

func TestEncodeBehaviorForWire_TrackingModeOmitsProtectedBlock(t *testing.T) {
	cfg := domain.DefaultSandboxConfig()
	cfg.Mode = domain.SandboxModeTracking
	wire := encodeBehaviorForWire(cfg)
	if _, ok := wire["protected"]; ok {
		t.Fatalf("tracking-mode behavior should not emit a protected block; got %v", wire["protected"])
	}
	if mr, ok := wire["manualReview"].(bool); !ok || mr {
		t.Errorf("expected manualReview=false on default cfg, got %v", wire["manualReview"])
	}
}

func TestEncodeBehaviorForWire_ProtectedModeEmitsDefaultGitAllowlist(t *testing.T) {
	cfg := domain.DefaultSandboxConfig()
	cfg.Mode = domain.SandboxModeProtected
	wire := encodeBehaviorForWire(cfg)
	prot, ok := wire["protected"].(map[string]interface{})
	if !ok {
		t.Fatalf("protected-mode behavior should emit a protected block; got %T", wire["protected"])
	}
	got, ok := prot["gitAllowlist"].([]string)
	if !ok {
		t.Fatalf("protected.gitAllowlist should be []string; got %T", prot["gitAllowlist"])
	}
	want := []string{"status", "diff", "log", "show", "rev-parse"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("gitAllowlist = %v, want %v", got, want)
	}
}

func TestEncodeBehaviorForWire_UnspecifiedModeTreatedAsTracking(t *testing.T) {
	cfg := domain.DefaultSandboxConfig()
	cfg.Mode = ""
	wire := encodeBehaviorForWire(cfg)
	if _, ok := wire["protected"]; ok {
		t.Fatalf("unspecified mode should not emit protected block")
	}
}
