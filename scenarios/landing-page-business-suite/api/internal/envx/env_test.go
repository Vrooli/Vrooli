package envx

import (
	"testing"
)

func TestSystemGet(t *testing.T) {
	t.Setenv("LPBS_ENVX_TEST_VALUE", "configured")
	if got := (System{}).Get("LPBS_ENVX_TEST_VALUE"); got != "configured" {
		t.Fatalf("Get() = %q, want configured", got)
	}
}
