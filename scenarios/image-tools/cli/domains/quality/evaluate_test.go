package quality

import (
	"io"
	"mime"
	"mime/multipart"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	qualityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/quality"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestUploadResolvesRelativePathsAndDeduplicatesAssets(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "source.png"), []byte("image fixture"), 0o600))
	path := filepath.Join(dir, "manifest.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"schema_version":1,"cases":[{"id":"one","input":"source.png","reference":"source.png","output":"source.png"}]}`), 0o600))
	f, ct, err := buildUpload(path, "")
	require.NoError(t, err)
	defer f.Close()
	defer os.Remove(f.Name())
	_, params, err := mime.ParseMediaType(ct)
	require.NoError(t, err)
	reader := multipart.NewReader(f, params["boundary"])
	parts := map[string][]byte{}
	for {
		p, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		parts[p.FormName()], err = io.ReadAll(p)
		require.NoError(t, err)
	}
	require.Len(t, parts, 2)
	require.Equal(t, []byte("image fixture"), parts["asset_0"])
	m := &qualityv1.EvaluationManifest{}
	require.NoError(t, protojson.Unmarshal(parts["manifest"], m))
	require.Equal(t, "asset_0", m.Cases[0].Input)
	require.Equal(t, m.Cases[0].Reference, m.Cases[0].Output)
}

func TestUploadRejectsMissingLocalAsset(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"schema_version":1,"cases":[{"id":"missing","output":"absent.png"}]}`), 0o600))
	f, _, err := buildUpload(path, "")
	require.Error(t, err)
	require.Nil(t, f)
}
