package themes

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveKitTokensInheritsTierAndOverridesValue(t *testing.T) {
	root := t.TempDir()
	writeThemeFixture(t, root, "templates/design/_base/tokens.css", ":root {\n  /* @tier Expression */\n  --color-surface: white;\n  /* @tier Contract */\n  --layer-modal: 400;\n}\n")
	writeThemeFixture(t, root, "templates/design/dark/adapters/react-vite-tailwind/tokens.css", ":root {\n  --color-surface: black;\n}\n")
	tokens, err := ResolveKitTokens(root, "dark")
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]DesignToken{}
	for _, token := range tokens {
		byName[token.Name] = token
	}
	if got := byName["--color-surface"]; got.Value != "black" || got.Tier != TokenTierExpression {
		t.Fatalf("surface = %#v, want black Expression", got)
	}
	if got := byName["--layer-modal"]; got.Value != "400" || got.Tier != TokenTierContract {
		t.Fatalf("layer = %#v, want 400 Contract", got)
	}
}

func writeThemeFixture(t *testing.T, root, relative, content string) {
	t.Helper()
	path := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
