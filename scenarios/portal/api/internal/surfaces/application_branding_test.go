package surfaces

import (
	"context"
	"encoding/json"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func brandingFixture(t *testing.T) (*ApplicationBranding, string) {
	t.Helper()
	root := t.TempDir()
	b := &ApplicationBranding{ScenarioRoot: func(name string) (string, error) { return filepath.Join(root, name), nil }, ServicePath: func(name string) (string, error) { return filepath.Join(root, name, ".vrooli", "service.json"), nil }}
	return b, root
}
func writeBranding(t *testing.T, root, name, label string) string {
	t.Helper()
	dir := filepath.Join(root, name)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".vrooli"), 0700))
	payload, _ := json.Marshal(map[string]any{"service": map[string]any{"displayName": label}})
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".vrooli", "service.json"), payload, 0600))
	public := filepath.Join(dir, "ui", "public", "public")
	require.NoError(t, os.MkdirAll(public, 0700))
	path := filepath.Join(public, "icon-192.png")
	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, png.Encode(f, image.NewRGBA(image.Rect(0, 0, 192, 192))))
	require.NoError(t, f.Close())
	return path
}
func TestApplicationBrandingPreservesIdentityAndAppliedProductName(t *testing.T) {
	b, root := brandingFixture(t)
	writeBranding(t, root, "web-console", "Aquila")
	writeBranding(t, root, "nutrition-planner", "Nooch")
	apps := &Applications{Branding: b, Run: func(context.Context) ([]byte, error) {
		return []byte(`{"success":true,"scenarios":[{"name":"web-console","status":"running"},{"name":"nutrition-planner","status":"running"},{"name":"unbranded","status":"running"}]}`), nil
	}}
	values, err := apps.List(context.Background())
	require.NoError(t, err)
	require.Equal(t, "web-console", values[0].Scenario)
	require.Equal(t, "Aquila", values[0].Label)
	require.Equal(t, "/api/v1/applications/web-console/icon", values[0].IconPath)
	require.Equal(t, "Nooch", values[1].Label)
	require.Equal(t, "unbranded", values[2].Label)
	require.Empty(t, values[2].IconPath, "missing artwork must not fabricate branding")
	data, digest, err := b.Icon(context.Background(), "web-console")
	require.NoError(t, err)
	require.NotEmpty(t, data)
	require.Len(t, digest, 64)
}
func TestApplicationBrandingRejectsUntrustedOrUnboundedArtwork(t *testing.T) {
	b, root := brandingFixture(t)
	path := writeBranding(t, root, "example", "Example")
	for _, name := range []string{"../example", "example/../../private", "Example", ""} {
		_, _, err := b.Icon(context.Background(), name)
		require.Error(t, err)
	}
	require.NoError(t, os.WriteFile(path, []byte("<html>sign in</html>"), 0600))
	_, _, err := b.Icon(context.Background(), "example")
	require.Error(t, err)
	require.NoError(t, os.WriteFile(path, make([]byte, 513<<10), 0600))
	_, _, err = b.Icon(context.Background(), "example")
	require.Error(t, err)
	external := filepath.Join(t.TempDir(), "private.png")
	require.NoError(t, os.WriteFile(external, []byte("private"), 0600))
	require.NoError(t, os.Remove(path))
	require.NoError(t, os.Symlink(external, path))
	_, _, err = b.Icon(context.Background(), "example")
	require.Error(t, err)
}
func TestApplicationBrandingHonorsCancelledReadAndPixelBound(t *testing.T) {
	b, root := brandingFixture(t)
	path := writeBranding(t, root, "example", "Example")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := b.Metadata(ctx, "example")
	require.ErrorIs(t, err, context.Canceled)
	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, png.Encode(f, image.NewRGBA(image.Rect(0, 0, 1025, 1))))
	require.NoError(t, f.Close())
	_, _, err = b.Icon(context.Background(), "example")
	require.Error(t, err)
}

func TestApplicationBrandingDoesNotOfferTemplatePlaceholderAsProductIcon(t *testing.T) {
	b, root := brandingFixture(t)
	writeBranding(t, root, "nutrition-planner", "Nooch")
	require.NoError(t, os.WriteFile(filepath.Join(root, "nutrition-planner", "ui", "public", "public", "logo.svg"), []byte(`<svg aria-label="Placeholder logo"><rect fill="#808080"/></svg>`), 0600))
	metadata, err := b.Metadata(context.Background(), "nutrition-planner")
	require.NoError(t, err)
	require.Equal(t, "Nooch", metadata.Label)
	require.Empty(t, metadata.IconPath)
	_, _, err = b.Icon(context.Background(), "nutrition-planner")
	require.Error(t, err)
}
