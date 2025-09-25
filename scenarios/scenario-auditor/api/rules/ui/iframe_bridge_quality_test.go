package ui

import (
	"testing"
)

func TestValidateEntry_StaticMissingCall(t *testing.T) {
	content := []byte(`(function () {
  if (typeof window === 'undefined') return;
  if (typeof window.initIframeBridgeChild !== 'function') return;
  if (window.parent === window) return;
  console.log('Bridge ready but not invoked');
})();`)

	violations := CheckIframeBridgeQuality(content, "ui/app.js")
	if len(violations) == 0 {
		t.Fatalf("expected violation for missing bridge call, got none")
	}
}
