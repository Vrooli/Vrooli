package settings

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vrooli/cli-core/cliapp"

	apisettings "flow-verifier/internal/settings"
)

// TestRegisterReturnsNonEmptyGroup pins the public shape: the group
// registers under "settings" and exposes the two canonical
// subcommands. Mirrors runs_test.go::TestRegisterReturnsNonEmptyRunsGroup.
func TestRegisterReturnsNonEmptyGroup(t *testing.T) {
	g := Register(&cliapp.ScenarioApp{})
	require.Equal(t, "settings", g.Name)
	require.NotEmpty(t, g.Description)
	require.NotEmpty(t, g.Subcommands)

	names := map[string]bool{}
	for _, c := range g.Subcommands {
		require.NotEmpty(t, c.Name)
		require.False(t, names[c.Name], "duplicate subcommand %q", c.Name)
		names[c.Name] = true
	}
	for _, expected := range []string{"get", "set"} {
		require.True(t, names[expected], "missing subcommand %q", expected)
	}
}

// TestAllowedKeysAlphabetical pins the contract: AllowedKeys returns a
// sorted slice so help / error text is stable across builds. A future
// addition that breaks the sort fails here.
func TestAllowedKeysAlphabetical(t *testing.T) {
	keys := AllowedKeys()
	require.NotEmpty(t, keys)
	for i := 1; i < len(keys); i++ {
		require.LessOrEqual(t, keys[i-1], keys[i], "AllowedKeys must be sorted; %s > %s at index %d", keys[i-1], keys[i], i)
	}
}

// TestParseAssignments_HappyPath drives every supported key through the
// parser and asserts the resulting Patch has the right pointer set.
func TestParseAssignments_HappyPath(t *testing.T) {
	cases := []struct {
		name   string
		pairs  []string
		assert func(t *testing.T, p apisettings.Patch)
	}{
		{
			name:  "theme",
			pairs: []string{"theme=dark"},
			assert: func(t *testing.T, p apisettings.Patch) {
				require.NotNil(t, p.Theme)
				require.Equal(t, apisettings.ThemeDark, *p.Theme)
			},
		},
		{
			name:  "fontScale",
			pairs: []string{"fontScale=lg"},
			assert: func(t *testing.T, p apisettings.Patch) {
				require.NotNil(t, p.FontScale)
				require.Equal(t, apisettings.FontScaleLg, *p.FontScale)
			},
		},
		{
			name:  "density",
			pairs: []string{"density=compact"},
			assert: func(t *testing.T, p apisettings.Patch) {
				require.NotNil(t, p.Density)
				require.Equal(t, apisettings.DensityCompact, *p.Density)
			},
		},
		{
			name:  "defaultRoot",
			pairs: []string{"defaultRoot=./scenarios"},
			assert: func(t *testing.T, p apisettings.Patch) {
				require.NotNil(t, p.DefaultRoot)
				require.Equal(t, "./scenarios", *p.DefaultRoot)
			},
		},
		{
			name:  "sidebarWidth",
			pairs: []string{"sidebarWidth=400"},
			assert: func(t *testing.T, p apisettings.Patch) {
				require.NotNil(t, p.SidebarWidth)
				require.Equal(t, 400, *p.SidebarWidth)
			},
		},
		{
			name:  "multiple",
			pairs: []string{"theme=light", "fontScale=sm", "density=compact"},
			assert: func(t *testing.T, p apisettings.Patch) {
				require.NotNil(t, p.Theme)
				require.Equal(t, apisettings.ThemeLight, *p.Theme)
				require.NotNil(t, p.FontScale)
				require.NotNil(t, p.Density)
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseAssignments(tc.pairs)
			require.NoError(t, err)
			tc.assert(t, got)
		})
	}
}

// TestParseAssignments_BooleanCoercion pins the vocabulary the plan
// promises: true|false|1|0|on|off (case-insensitive).
func TestParseAssignments_BooleanCoercion(t *testing.T) {
	truthy := []string{"true", "TRUE", "1", "on", "ON"}
	falsy := []string{"false", "FALSE", "0", "off", "OFF"}

	for _, v := range truthy {
		t.Run("reducedMotion="+v, func(t *testing.T) {
			p, err := ParseAssignments([]string{"reducedMotion=" + v})
			require.NoError(t, err)
			require.NotNil(t, p.ReducedMotion)
			require.True(t, *p.ReducedMotion, "value %q must coerce to true", v)
		})
	}
	for _, v := range falsy {
		t.Run("rtl="+v, func(t *testing.T) {
			p, err := ParseAssignments([]string{"rtl=" + v})
			require.NoError(t, err)
			require.NotNil(t, p.RTL)
			require.False(t, *p.RTL, "value %q must coerce to false", v)
		})
	}
}

// TestParseAssignments_UnknownKey: the typed error includes the
// offending key and lists the allowed set.
func TestParseAssignments_UnknownKey(t *testing.T) {
	_, err := ParseAssignments([]string{"bananas=yellow"})
	require.Error(t, err)
	var uk UnknownKeyError
	require.True(t, errors.As(err, &uk), "expected UnknownKeyError, got %T", err)
	require.Equal(t, "bananas", uk.Key)
	require.Contains(t, uk.Error(), "theme", "error must list allowed keys")
}

// TestParseAssignments_BadShape: a value without '=' is rejected
// up-front so a malformed argv never reaches the typed-value layer.
func TestParseAssignments_BadShape(t *testing.T) {
	for _, in := range []string{"justKey", "=noKey", " "} {
		t.Run(in, func(t *testing.T) {
			_, err := ParseAssignments([]string{in})
			if in == " " {
				require.NoError(t, err, "whitespace-only entries are skipped")
				return
			}
			require.Error(t, err)
		})
	}
}

// TestParseAssignments_BadInt: sidebarWidth must be a non-negative
// integer. Negative values and non-numeric strings both fail.
func TestParseAssignments_BadInt(t *testing.T) {
	for _, v := range []string{"abc", "-1"} {
		t.Run(v, func(t *testing.T) {
			_, err := ParseAssignments([]string{"sidebarWidth=" + v})
			require.Error(t, err)
		})
	}
}

// TestRenderText_TwoColumnLayout exercises the human-text render path.
// Asserts every documented row is present and aligned. The exact width
// is not pinned (it depends on the longest key); the property checked
// is that "theme" and "sidebarWidth" both appear and at least one
// value column is present.
func TestRenderText_TwoColumnLayout(t *testing.T) {
	s := apisettings.DefaultSettings()
	s.Theme = apisettings.ThemeDark
	s.SidebarWidth = 400

	var buf bytes.Buffer
	require.NoError(t, renderText(&buf, s))
	out := buf.String()
	require.Contains(t, out, "theme")
	require.Contains(t, out, "dark")
	require.Contains(t, out, "sidebarWidth")
	require.Contains(t, out, "400")
	require.Contains(t, out, "inventoryFilters.sort.key")
}
