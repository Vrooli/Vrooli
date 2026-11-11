package facts

import (
	"testing"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

func TestGraphProjectPathUsesProjectConfigForTypeScriptAndRootForJavaScript(t *testing.T) {
	unit := &factsv1.ParseUnit{RootPath: "/repo/ui", ConfigPath: "/repo/ui/package.json"}
	if got := graphProjectPath("typescript", unit); got != unit.GetConfigPath() {
		t.Fatalf("TypeScript project path = %q, want config %q", got, unit.GetConfigPath())
	}
	if got := graphProjectPath("javascript", unit); got != unit.GetRootPath() {
		t.Fatalf("JavaScript project path = %q, want root %q", got, unit.GetRootPath())
	}
}
