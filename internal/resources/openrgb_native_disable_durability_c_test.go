package resources

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/operatorstate"
)

// Exercise the actual native operator-state writer and actual persistDisabled
// boundary, without a release, service, device, SDK or platform-manager effect.
// In particular, this must run on Windows: Linux-shaped headless fixtures skip
// there and cannot catch a read-only directory FlushFileBuffers call.
func TestAdversarialCNativeDisableDurabilityPreservesSourceAcrossRetry(t *testing.T) {
	home := t.TempDir()
	root := t.TempDir()
	copyOperatorStateSchemaForTest(t, root)
	c := NewController(root, home)
	c.operatorState = operatorstate.New(operatorstate.Config{RepoRoot: root, LogicalOperatorHome: home, StateRoot: filepath.Join(home, "private-state")})
	if _, err := c.operatorStateService().Apply(t.Context(), []byte(`{"resources":{"openrgb":{"enabled":true,"source":{"mode":"managed"}}}}`)); err != nil {
		t.Fatal(err)
	}
	owner := NewOpenRGBHeadless(c, nil)
	for range 2 {
		if err := owner.persistDisabled(t.Context()); err != nil {
			t.Fatalf("native disable must confirm its owner's durable publication and permit retirement: %v", err)
		}
		got, err := c.operatorStateService().Load(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		choice := got.Resources[openRGBResourceName]
		if choice.Enabled == nil || *choice.Enabled || choice.Source == nil || choice.Source.Mode != "managed" {
			t.Fatalf("retry lost opt-out or selected source: %+v", choice)
		}
	}
}

// Use production composition rather than the injected test operator service.
// This native-only case must never resolve or modify the shared runtime home.
func TestAdversarialCDefaultNativeOperatorDisableUsesLogicalPrivateHome(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		t.Skip("default native operator composition")
	}
	home, root := t.TempDir(), t.TempDir()
	copyOperatorStateSchemaForTest(t, root)
	c := NewController(root, home)
	path, err := c.operatorStateService().Path(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(path, home+string(filepath.Separator)) {
		t.Fatalf("native operator state escaped private logical home: %s", path)
	}
	if _, err = c.operatorStateService().Apply(t.Context(), []byte(`{"resources":{"openrgb":{"enabled":true,"source":{"mode":"managed"}}}}`)); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err = NewOpenRGBHeadless(c, nil).persistDisabled(t.Context()); err != nil {
			t.Fatalf("default native owner prevented retirement: %v", err)
		}
	}
}
