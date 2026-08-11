package imageengine

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClientApplyRunsTheOrderedImageToolsChain(t *testing.T) {
	var operations []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		operations = append(operations, r.URL.Path)
		require.Equal(t, "bytes", r.URL.Query().Get("output"))
		require.NoError(t, r.ParseMultipartForm(1<<20))
		file, _, err := r.FormFile("file")
		require.NoError(t, err)
		defer func() { _ = file.Close() }()
		input, err := io.ReadAll(file)
		require.NoError(t, err)
		if len(operations) == 1 {
			require.Equal(t, []byte{1, 2}, input)
		} else {
			require.Equal(t, []byte{1, 2, 1}, input)
		}
		params := r.FormValue("params")
		if strings.HasSuffix(r.URL.Path, "/duotone") {
			require.Contains(t, params, "#123456")
		}
		_, _ = w.Write(append(input, byte(len(operations))))
	}))
	defer server.Close()

	client := &Client{HTTPClient: server.Client(), Resolve: func(context.Context) (string, error) { return server.URL, nil }}
	out, err := client.Apply(context.Background(), []byte{1, 2}, []string{"duotone", "grain"}, map[string]string{"$brand.primary": "#123456"})
	require.NoError(t, err)
	require.Equal(t, []byte{1, 2, 1, 2}, out)
	require.Equal(t, []string{"/api/v1/ops/duotone", "/api/v1/ops/grain"}, operations)
}

func TestClientApplyRefusesMissingInputsAndUnresolvedTreatmentNames(t *testing.T) {
	client := &Client{Resolve: func(context.Context) (string, error) { return "http://example.test", nil }}
	_, err := client.Apply(context.Background(), nil, []string{"grain"}, nil)
	require.ErrorContains(t, err, "input image is empty")
	_, err = client.Apply(context.Background(), []byte{1}, []string{"$brand.primary"}, nil)
	require.ErrorContains(t, err, "invalid treatment")
}
