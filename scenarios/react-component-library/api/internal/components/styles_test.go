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
		"vrooli-command-display",
		"vrooli-conversion-landing",
		"vrooli-default",
	}, ids)
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
