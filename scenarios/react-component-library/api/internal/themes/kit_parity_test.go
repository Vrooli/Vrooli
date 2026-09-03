package themes

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSharedRampNamesMatchAcrossDesignKitsAndLibraryTheme(t *testing.T) {
	root := findRepositoryRootForTest(t)
	files := []string{
		filepath.Join(root, "templates/design/vrooli-default/adapters/react-vite-tailwind/tailwind.theme.json"),
		filepath.Join(root, "templates/design/vrooli-command-display/adapters/react-vite-tailwind/tailwind.theme.json"),
		filepath.Join(root, "templates/design/vrooli-conversion-landing/adapters/react-vite-tailwind/tailwind.theme.json"),
		filepath.Join(root, "scenarios/react-component-library/ui/src/theme/tailwind.theme.json"),
	}
	want := map[string][]string{
		"spacing":                  {"space-3xs", "space-2xs", "space-xs", "space-sm", "space-md", "space-lg", "space-xl", "space-2xl"},
		"fontSize":                 {"display", "title", "heading", "subheading", "body", "body-sm", "label", "caption"},
		"boxShadow":                {"flat", "raised", "overlay", "modal"},
		"zIndex":                   {"base", "sticky", "dropdown", "overlay", "modal", "toast", "tooltip"},
		"borderWidth":              {"hairline", "strong"},
		"opacity":                  {"disabled", "muted", "scrim"},
		"transitionDuration":       {"instant", "quick", "moderate", "deliberate"},
		"transitionTimingFunction": {"standard", "enter", "exit", "spring-subtle", "spring-expressive"},
	}
	for _, path := range files {
		raw, err := os.ReadFile(path)
		require.NoError(t, err, path)
		var theme map[string]map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(raw, &theme), path)
		for section, names := range want {
			for _, name := range names {
				require.Contains(t, theme[section], name, "%s must expose %s.%s", path, section, name)
			}
		}
	}

	for _, path := range []string{
		filepath.Join(root, "templates/design/vrooli-default/adapters/react-vite-tailwind/tokens.css"),
		filepath.Join(root, "templates/design/vrooli-command-display/adapters/react-vite-tailwind/tokens.css"),
		filepath.Join(root, "templates/design/vrooli-conversion-landing/adapters/react-vite-tailwind/tokens.css"),
		filepath.Join(root, "scenarios/react-component-library/ui/src/design-tokens.css"),
	} {
		raw, err := os.ReadFile(path)
		require.NoError(t, err, path)
		require.NotContains(t, string(raw), "--spacing-unit", path)
	}
}

func findRepositoryRootForTest(t *testing.T) string {
	t.Helper()
	for _, candidate := range []string{".", "../..", "../../..", "../../../..", "../../../../.."} {
		if _, err := os.Stat(filepath.Join(candidate, "templates/design/vrooli-default/adapters/react-vite-tailwind/tokens.css")); err == nil {
			root, err := filepath.Abs(candidate)
			require.NoError(t, err)
			return root
		}
	}
	t.Fatal("repository root not found")
	return ""
}
