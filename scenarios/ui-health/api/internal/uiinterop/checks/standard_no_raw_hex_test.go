package checks

import (
	"strings"
	"testing"

	"ui-health/internal/uiinterop"
)

func TestNoRawHexExemptsOnlyNativeColorInputValues(t *testing.T) {
	ctx := uiinterop.CheckContext{Sources: []uiinterop.SourceFile{
		{
			RelPath: "ui/src/components/Branding.tsx",
			Content: `export const Branding = () => <input type="color" value="#0f172a" />;`,
		},
	}}

	result := checkNoRawHex(ctx)
	if !result.Passed {
		t.Fatalf("native color input result = %+v, want pass", result)
	}

	ctx.Sources[0].Content = `export const Branding = () => <><input type="color" value="#0f172a" /><div style={{ color: "#0f172a" }} /></>;`
	result = checkNoRawHex(ctx)
	if result.Passed || len(result.Violations) != 1 {
		t.Fatalf("mixed native color input result = %+v, want one violation", result)
	}
	if got := result.Violations[0].Description; got == "" || !strings.Contains(got, "#0f172a") {
		t.Fatalf("violation description = %q, want remaining inline color", got)
	}
}
