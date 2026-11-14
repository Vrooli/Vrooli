package themes_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"react-component-library/internal/themes"
)

const flowVerifierDesignMD = `---
id: vrooli-default
version: 0.2.0
name: Vrooli Operational Console
description: Dense operational UI.
colors:
  primary: "#2563eb"
  secondary: "#0891b2"
  surface: "#ffffff"
  on-surface: "#0f172a"
typography:
  body-md:
    fontFamily: Inter
    fontSize: 16px
  label-md:
    fontFamily: Inter
    fontSize: 14px
rounded:
  sm: 0.375rem
  md: 0.5rem
spacing:
  unit: 0.25rem
  touch: 44px
---

# DESIGN

body goes here
`

func TestExtractFrontMatter(t *testing.T) {
	front, ok := themes.ExtractFrontMatter([]byte(flowVerifierDesignMD))
	require.True(t, ok)
	require.True(t, strings.Contains(front, "primary: \"#2563eb\""))
	require.False(t, strings.Contains(front, "DESIGN"))
}

func TestExtractFrontMatter_NoMarker(t *testing.T) {
	_, ok := themes.ExtractFrontMatter([]byte("# no front matter\n"))
	require.False(t, ok)
}

func TestExtractFrontMatter_NotClosed(t *testing.T) {
	_, ok := themes.ExtractFrontMatter([]byte("---\nfoo: bar\n"))
	require.False(t, ok)
}

func TestParseDesignMDToTheme(t *testing.T) {
	theme, err := themes.ParseDesignMDToTheme([]byte(flowVerifierDesignMD), "flow-verifier")
	require.NoError(t, err)
	require.Equal(t, "scenario:flow-verifier", theme.ID)
	require.Equal(t, "scenario:flow-verifier", theme.Source)
	require.Equal(t, "#2563eb", theme.Tokens["--color-primary"])
	require.Equal(t, "0.375rem", theme.Tokens["--radius-sm"])
	require.NotContains(t, theme.Tokens, "--rounded-sm")
	require.Equal(t, "0.25rem", theme.Tokens["--spacing-unit"])
	require.Equal(t, "Inter", theme.Tokens["--typography-body-md-fontfamily"])
}

func TestParseDesignMDToTheme_MissingFrontMatter(t *testing.T) {
	_, err := themes.ParseDesignMDToTheme([]byte("# nope"), "scn")
	require.Error(t, err)
	var sentinel themes.ErrInvalidDesignMD
	require.ErrorAs(t, err, &sentinel)
}

func TestParseDesignMDToTheme_EmptyShape(t *testing.T) {
	_, err := themes.ParseDesignMDToTheme([]byte("---\nid: foo\n---\n"), "scn")
	require.Error(t, err)
	var sentinel themes.ErrInvalidDesignMD
	require.ErrorAs(t, err, &sentinel)
}

func TestParseDesignMDToTheme_BadYAML(t *testing.T) {
	_, err := themes.ParseDesignMDToTheme([]byte("---\ncolors: { broken\n---\n"), "scn")
	require.Error(t, err)
}
