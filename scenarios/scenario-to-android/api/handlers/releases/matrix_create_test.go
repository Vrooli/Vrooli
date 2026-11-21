package ramp

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	validationmatrix "github.com/vrooli/vrooli/packages/delivery-ramp-go/validationmatrix"
	buildsh "scenario-to-android/handlers/builds"
	buildsdomain "scenario-to-android/internal/builds"
)

type matrixCreatorFunc func(validationmatrix.MatrixSelection) (*validationmatrix.MatrixRun, error)

func (f matrixCreatorFunc) Create(selection validationmatrix.MatrixSelection) (*validationmatrix.MatrixRun, error) {
	return f(selection)
}

func TestMatrixCreateResolvesArtifactInternallyAndRedactsResponsePath(t *testing.T) {
	const digest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	const ref = "android-debug:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	path := "/producer/private/builds/artifacts/app.apk"
	descriptor := buildsdomain.APKDescriptor{
		ImmutableRef: ref, Checksum: digest, PackageName: "com.example.real",
		VersionName: "1.2.3", VersionCode: 12, Signing: "debug-only",
	}
	file, err := os.CreateTemp(t.TempDir(), "verified.apk")
	require.NoError(t, err)
	defer file.Close()
	var received validationmatrix.MatrixSelection
	handler := matrixCreateHandler(
		buildsh.Surface{ArtifactRoot: func(context.Context) (string, error) { return "/producer/private", nil }},
		matrixCreatorFunc(func(selection validationmatrix.MatrixSelection) (*validationmatrix.MatrixRun, error) {
			received = selection
			require.Equal(t, "com.example.real", selection.Metadata["package_name"])
			require.Equal(t, "1.2.3", selection.Metadata["version_name"])
			require.Equal(t, "12", selection.Metadata["version_code"])
			require.Equal(t, "debug-only", selection.Metadata["signing"])
			return &validationmatrix.MatrixRun{RunID: "run-1", Selection: selection}, nil
		}),
		func(root, immutableRef string) (buildsdomain.ResolvedAPK, error) {
			require.Equal(t, "/producer/private", root)
			require.Equal(t, ref, immutableRef)
			return buildsdomain.ResolvedAPK{Descriptor: descriptor, File: file, Path: path}, nil
		},
	)
	requestBody := `{"artifact_ref":"` + ref + `","selection":{"scenario_name":"demo","artifact_digest":"` + digest + `","metadata":{"immutable_ref":"` + ref + `"}}}`
	response := httptest.NewRecorder()
	handler(response, httptest.NewRequest("POST", "/api/v1/android/matrices", strings.NewReader(requestBody)))
	require.Equal(t, 201, response.Code, response.Body.String())
	require.Equal(t, path, received.ArtifactPath)
	require.Contains(t, response.Body.String(), "run-1")
	require.NotContains(t, response.Body.String(), path)
	require.NotContains(t, response.Body.String(), "artifact_path")
	var payload map[string]any
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &payload))
}

func TestMatrixCreateRejectsUntrustedArtifactMetadata(t *testing.T) {
	const digest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	const ref = "android-debug:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	descriptor := buildsdomain.APKDescriptor{
		ImmutableRef: ref, Checksum: digest, PackageName: "com.example.real",
		VersionName: "1.2.3", VersionCode: 12, Signing: "debug-only",
	}
	for _, field := range []string{"package_name", "version_name", "version_code", "signing", "signing_identity", "apk_path"} {
		t.Run(field, func(t *testing.T) {
			file, err := os.CreateTemp(t.TempDir(), "verified.apk")
			require.NoError(t, err)
			defer file.Close()
			called := false
			handler := matrixCreateHandler(
				buildsh.Surface{ArtifactRoot: func(context.Context) (string, error) { return "/producer", nil }},
				matrixCreatorFunc(func(validationmatrix.MatrixSelection) (*validationmatrix.MatrixRun, error) {
					called = true
					return nil, nil
				}),
				func(string, string) (buildsdomain.ResolvedAPK, error) {
					return buildsdomain.ResolvedAPK{Descriptor: descriptor, File: file, Path: "/producer/app.apk"}, nil
				},
			)
			requestBody, err := json.Marshal(map[string]any{
				"artifact_ref": ref,
				"selection": map[string]any{
					"artifact_digest": digest,
					"metadata":        map[string]string{"immutable_ref": ref, field: "spoofed-value"},
				},
			})
			require.NoError(t, err)
			response := httptest.NewRecorder()
			handler(response, httptest.NewRequest("POST", "/api/v1/android/matrices", strings.NewReader(string(requestBody))))
			require.Equal(t, 400, response.Code, response.Body.String())
			require.False(t, called, "creator must not receive spoofed artifact identity")
		})
	}
}

func TestMatrixCreateRejectsCallerSuppliedArtifactPath(t *testing.T) {
	handler := MatrixCreateHandler(buildsh.Surface{}, matrixCreatorFunc(func(validationmatrix.MatrixSelection) (*validationmatrix.MatrixRun, error) {
		t.Fatal("creator should not receive caller-supplied paths")
		return nil, nil
	}))
	request := httptest.NewRequest("POST", "/api/v1/android/matrices", strings.NewReader(`{"artifact_ref":"android-debug:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","selection":{"artifact_path":"/outside/app.apk"}}`))
	response := httptest.NewRecorder()
	handler(response, request)
	require.Equal(t, 400, response.Code)
}
