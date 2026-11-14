package components_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"react-component-library/internal/components"
)

func TestLoadDesignStylesSkipsReservedSharedDirectories(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "_base"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "demo"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "_base", "tokens.css"), []byte(":root {}"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "demo", "metadata.json"), []byte(`{"id":"demo","name":"Demo","adapters":{}}`), 0o644))

	styles, err := components.LoadDesignStyles(context.Background(), root)
	require.NoError(t, err)
	require.Len(t, styles, 1)
	require.Equal(t, "demo", styles[0].ID)
}

func TestLoadDesignStylesRejectsMalformedMetadata(t *testing.T) {
	root := t.TempDir()
	designDirectory := filepath.Join(root, "broken")
	require.NoError(t, os.MkdirAll(designDirectory, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(designDirectory, "metadata.json"), []byte("{"), 0o644))

	_, err := components.LoadDesignStyles(context.Background(), root)
	require.ErrorContains(t, err, `parse design metadata "broken"`)
}

func TestLoadDesignStylesLoadsCanonicalTemplates(t *testing.T) {
	styles, err := components.LoadDesignStyles(context.Background(), "../../../../../templates/design")
	require.NoError(t, err)

	ids := make([]string, 0, len(styles))
	for _, style := range styles {
		ids = append(ids, style.ID)
		require.NotEmpty(t, style.Name)
		require.NotEmpty(t, style.Supports)
	}
	require.ElementsMatch(t, []string{
		"switchboard-console",
		"vrooli-command-display",
		"vrooli-conversion-landing",
		"vrooli-default",
	}, ids)
}

func TestListDesignStylesFindsCanonicalTemplatesFromNestedWorkingDirectory(t *testing.T) {
	originalWorkingDirectory, err := os.Getwd()
	require.NoError(t, err)
	temporaryDirectory, err := os.MkdirTemp(originalWorkingDirectory, ".trimpath-design-root-*")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(temporaryDirectory))
	})
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(originalWorkingDirectory))
	})
	nestedWorkingDirectory := filepath.Join(temporaryDirectory, "one", "two", "three", "four", "five")
	require.NoError(t, os.MkdirAll(nestedWorkingDirectory, 0o755))
	require.NoError(t, os.Chdir(nestedWorkingDirectory))

	styles, err := components.NewService(nil).ListDesignStyles(context.Background())
	require.NoError(t, err)
	var hasDefaultStyle bool
	for _, style := range styles {
		if style.ID == "vrooli-default" {
			hasDefaultStyle = true
			break
		}
	}
	require.True(t, hasDefaultStyle)
}

func TestServiceValidateDesignStyleRejectsUnknownID(t *testing.T) {
	repo, _ := newComponentsDB(t)
	svc := components.NewService(repo)

	require.NoError(t, svc.ValidateDesignStyle(context.Background(), "vrooli-default"))
	err := svc.ValidateDesignStyle(context.Background(), "missing-style")
	require.Error(t, err)
	var invalid components.ErrInvalidHeader
	require.ErrorAs(t, err, &invalid)
	require.Equal(t, "designStyles", invalid.Field)
}
