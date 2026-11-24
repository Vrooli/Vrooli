package downloads

import (
	"bytes"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	"landing-page-business-suite/cli/internal/support"
	testutil "landing-page-business-suite/cli/internal/testutil"
)

func TestRegisterExposesValidCommandGroup(t *testing.T) {
	group := Register(support.Dependencies{})
	if err := testutil.ValidateCommandGroup(group); err != nil {
		t.Fatalf("ValidateCommandGroup() error = %v", err)
	}
}

func TestRunUploadManagedCommitsDigestsFromPresignedPut(t *testing.T) {
	payload := []byte("the exact managed release artifact")
	var uploaded []byte
	var commit map[string]any
	var applied map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/admin/download-artifacts/presign-upload":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"upload_url":%q,"bucket":"local-test","object_key":"release/artifact.zip"}`, "http://"+r.Host+"/upload")
		case "/upload":
			if r.Method != http.MethodPut {
				t.Errorf("upload method = %s, want PUT", r.Method)
			}
			uploaded, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusOK)
		case "/api/v1/admin/download-artifacts/commit":
			if err := json.NewDecoder(r.Body).Decode(&commit); err != nil {
				t.Errorf("decode commit payload: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, `{"id":41}`)
		case "/api/v1/admin/download-assets/apply":
			if err := json.NewDecoder(r.Body).Decode(&applied); err != nil {
				t.Errorf("decode apply payload: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, `{}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	configDir := t.TempDir()
	t.Setenv("LPBS_DOWNLOADS_TEST_CONFIG_DIR", configDir)
	core, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{
		Name:             "landing-page-business-suite-test",
		DefaultAPIBase:   server.URL,
		APIPrefix:        "/api/v1",
		ConfigDirEnvVars: []string{"LPBS_DOWNLOADS_TEST_CONFIG_DIR"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(core.ConfigFile.Path), "admin_session.json"), []byte(fmt.Sprintf(`{"sessions":{%q:{"api_base":%q,"session":"test-session"}}}`, server.URL, server.URL)), 0o600); err != nil {
		t.Fatal(err)
	}
	artifactPath := filepath.Join(configDir, "artifact.zip")
	if err := os.WriteFile(artifactPath, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	deps := support.Dependencies{Core: func() *cliapp.ScenarioApp { return core }}
	if err := runUploadManaged(deps, []string{
		"--file", artifactPath, "--app-key", "desktop", "--platform", "linux", "--release-version", "1.2.3", "--json",
	}); err != nil {
		t.Fatalf("runUploadManaged: %v", err)
	}
	if !bytes.Equal(uploaded, payload) {
		t.Fatalf("uploaded bytes = %q, want %q", uploaded, payload)
	}
	expectedSHA256 := sha256.Sum256(payload)
	expectedSHA512 := sha512.Sum512(payload)
	if got := commit["sha256"]; got != fmt.Sprintf("%x", expectedSHA256) {
		t.Fatalf("commit sha256 = %v", got)
	}
	if got := commit["sha512"]; got != base64.StdEncoding.EncodeToString(expectedSHA512[:]) {
		t.Fatalf("commit sha512 = %v", got)
	}
	if got := applied["artifact_id"]; got != float64(41) {
		t.Fatalf("applied artifact id = %v, want 41", got)
	}
}

func TestArtifactDigestReaderHashesExactlyTheStreamedBytes(t *testing.T) {
	payload := []byte("managed artifact bytes")
	sha256Hash, sha512Hash := sha256.New(), sha512.New()
	stream := &artifactDigestReader{reader: bytes.NewReader(payload), digest: io.MultiWriter(sha256Hash, sha512Hash)}
	uploaded, err := io.ReadAll(stream)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(uploaded, payload) || stream.bytesRead != int64(len(payload)) {
		t.Fatalf("streamed %q (%d bytes), want %q (%d bytes)", uploaded, stream.bytesRead, payload, len(payload))
	}
	expectedSHA256 := sha256.Sum256(payload)
	if got, want := fmt.Sprintf("%x", sha256Hash.Sum(nil)), fmt.Sprintf("%x", expectedSHA256); got != want {
		t.Fatalf("sha256 = %s, want %s", got, want)
	}
	expectedSHA512 := sha512.Sum512(payload)
	if got, want := base64.StdEncoding.EncodeToString(sha512Hash.Sum(nil)), base64.StdEncoding.EncodeToString(expectedSHA512[:]); got != want {
		t.Fatalf("sha512 = %s, want %s", got, want)
	}
}

func TestDownloadAppCommandsUseGeneratedConnectPrimitives(t *testing.T) {
	commands := make(map[string]cliapp.Command)
	for _, command := range Register(support.Dependencies{}).Commands {
		commands[command.Name] = command
	}

	tests := []struct {
		name      string
		primitive cliapp.PrimitiveClass
		body      bool
		appKey    bool
	}{
		{name: "admin-download-apps-list", primitive: cliapp.PrimitiveProtoList},
		{name: "admin-download-apps-create", primitive: cliapp.PrimitiveProtoMutation, body: true},
		{name: "admin-download-apps-save", primitive: cliapp.PrimitiveProtoMutation, body: true, appKey: true},
		{name: "admin-download-apps-delete", primitive: cliapp.PrimitiveProtoMutation, appKey: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			command, ok := commands[tt.name]
			if !ok {
				t.Fatalf("command %q is not registered", tt.name)
			}
			if !command.NeedsAPI || command.Architecture.Primitive != tt.primitive || command.PrimitiveEvidence() != tt.primitive {
				t.Fatalf("command transport primitive = %q/%q, want %q", command.Architecture.Primitive, command.PrimitiveEvidence(), tt.primitive)
			}
			if tt.body {
				if len(command.Args.Flags) != 1 || command.Args.Flags[0].Name != "body" || !command.Args.Flags[0].Required {
					t.Fatalf("command body contract = %#v", command.Args.Flags)
				}
			}
			if tt.appKey {
				if len(command.Args.Positionals) != 1 || command.Args.Positionals[0].Name != "app_key" || !command.Args.Positionals[0].Required {
					t.Fatalf("command app_key contract = %#v", command.Args.Positionals)
				}
			}
		})
	}
}

func TestRunUploadManagedRejectsIncompleteInvocationBeforeNetwork(t *testing.T) {
	err := runUploadManaged(support.Dependencies{}, []string{"--file", "artifact.zip"})
	if err == nil || !strings.Contains(err.Error(), "platform is required") {
		t.Fatalf("error = %v, want platform validation error", err)
	}
}

func TestRunUploadManagedRejectsInvalidPlatformBeforeNetwork(t *testing.T) {
	err := runUploadManaged(support.Dependencies{}, []string{
		"--file", "artifact.zip", "--app-key", "desktop", "--platform", "ios", "--release-version", "1.2.3",
	})
	if err == nil || !strings.Contains(err.Error(), "platform") {
		t.Fatalf("error = %v, want platform validation error", err)
	}
}

func TestRunUploadManagedRejectsMissingArtifactBeforeNetwork(t *testing.T) {
	err := runUploadManaged(support.Dependencies{}, []string{
		"--file", "does-not-exist.zip", "--app-key", "desktop", "--platform", "windows", "--release-version", "1.2.3",
	})
	if err == nil || !strings.Contains(err.Error(), "artifact file not found") {
		t.Fatalf("error = %v, want missing artifact error", err)
	}
}
