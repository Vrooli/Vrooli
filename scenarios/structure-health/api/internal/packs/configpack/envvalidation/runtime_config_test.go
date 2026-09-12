package envvalidation

import "testing"

func TestCheckViteRuntimeConfig(t *testing.T) {
	violations := CheckViteRuntimeConfig("const url = import.meta.env.VITE_API_URL", "ui/src/api.ts")
	if len(violations) != 1 || violations[0].LineNumber != 1 {
		t.Fatalf("expected one runtime-config violation, got %#v", violations)
	}
}

func TestCheckViteRuntimeConfigAllowsViteConfig(t *testing.T) {
	if violations := CheckViteRuntimeConfig("const url = import.meta.env.VITE_API_URL", "vite.config.ts"); len(violations) != 0 {
		t.Fatalf("expected vite config exemption, got %#v", violations)
	}
}
