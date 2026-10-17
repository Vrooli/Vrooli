package domains

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/vrooli/cli-core/cliapp"
)

func TestStyleLyricsFilePreservesTextForCreateAndImport(t *testing.T) {
	const lyrics = "[Verse]\nOur own café lights\n\n[Chorus]\nWe rise again\n"
	for _, operation := range []string{"create", "import"} {
		t.Run(operation, func(t *testing.T) {
			dir := t.TempDir()
			file := filepath.Join(dir, "style.json")
			text := filepath.Join(dir, "lyrics.txt")
			require.NoError(t, os.WriteFile(file, []byte(`{"id":"song","name":"Song","caption":"bright pop","lyrics":"old text","bpm":150,"duration":30}`), 0600))
			require.NoError(t, os.WriteFile(text, []byte(lyrics), 0600))
			requests := make(chan map[string]any, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				wantPath := "/api/v1/styles"
				if operation == "import" {
					wantPath += "/import"
				}
				if r.Method != http.MethodPost || r.URL.Path != wantPath {
					t.Errorf("request=%s %s", r.Method, r.URL.Path)
				}
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				requests <- body
				w.WriteHeader(http.StatusCreated)
				fmt.Fprint(w, `{"id":"song"}`)
			}))
			defer server.Close()
			app, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{Name: "lyrics-file-test", DefaultAPIBase: server.URL, APIPrefix: "/api/v1", AllowAnonymous: true})
			require.NoError(t, err)
			groups, err := SubcommandGroups(app, readManifestForTest(t))
			require.NoError(t, err)
			var run func([]string) error
			for _, group := range groups {
				if group.Name == "styles" {
					for _, command := range group.Subcommands {
						if command.Name == operation {
							run = command.Run
						}
					}
				}
			}
			require.NotNil(t, run)
			require.NoError(t, run([]string{"--file", file, "--lyrics-file", text}))
			got := <-requests
			require.Equal(t, lyrics, got["lyrics"])
			require.Equal(t, float64(150), got["bpm"])
		})
	}
}

func TestStyleLyricsFileErrorsAndEmptyInputAreExplicit(t *testing.T) {
	for name, test := range map[string]struct {
		content []byte
		reason  string
	}{"empty": {[]byte{}, "empty"}, "whitespace": {[]byte(" \n\t"), "empty"}, "invalid UTF8": {[]byte{0xff}, "UTF-8"}} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			file := filepath.Join(dir, "style.json")
			text := filepath.Join(dir, "lyrics.txt")
			require.NoError(t, os.WriteFile(file, []byte(`{"id":"song"}`), 0600))
			require.NoError(t, os.WriteFile(text, test.content, 0600))
			_, err := styleBody([]string{"--file", file, "--lyrics-file", text})
			require.Error(t, err)
			require.Contains(t, err.Error(), test.reason)
		})
	}
	file := filepath.Join(t.TempDir(), "style.json")
	require.NoError(t, os.WriteFile(file, []byte(`{"id":"song"}`), 0600))
	_, err := styleBody([]string{"--file", file, "--lyrics-file", filepath.Join(t.TempDir(), "missing.txt")})
	require.Error(t, err)
	require.Contains(t, err.Error(), "read lyrics")
}

func TestComposeOptionalSeedAndShiftWire(t *testing.T) {
	for name, args := range map[string][]string{"default": {}, "explicit zero": {"--seed", "0"}, "paired shift": {"--seed", "424242", "--shift", "3"}, "negative base": {"--seed", "-7", "--shift", "1"}} {
		t.Run(name, func(t *testing.T) {
			requests := make(chan map[string]any, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				requests <- body
				fmt.Fprint(w, `{"job_id":"fake"}`)
			}))
			defer server.Close()
			app, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{Name: "compose-seed-test", DefaultAPIBase: server.URL, APIPrefix: "/api/v1", AllowAnonymous: true})
			require.NoError(t, err)
			require.NoError(t, compose(app, append([]string{"--style", "song", "--takes", "1"}, args...)))
			body := <-requests
			switch name {
			case "default":
				require.NotContains(t, body, "seed")
				require.NotContains(t, body, "shift")
			case "explicit zero":
				require.Equal(t, float64(0), body["seed"])
			case "paired shift":
				require.Equal(t, float64(424242), body["seed"])
				require.Equal(t, float64(3), body["shift"])
			case "negative base":
				require.Equal(t, float64(-7), body["seed"])
				require.Equal(t, float64(1), body["shift"])
			}
		})
	}
	for _, value := range []string{"NaN", "Inf", "0", "-1", "6"} {
		err := compose(nil, []string{"--style", "song", "--shift", value})
		require.Error(t, err)
		require.Contains(t, err.Error(), "between 1 and 5")
	}
}

func TestWaitForJobUsesServerOwnedWaitClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/jobs/job-1/wait" {
			t.Fatalf("path=%q", r.URL.Path)
		}
		time.Sleep(80 * time.Millisecond)
		fmt.Fprint(w, `{"state":"succeeded"}`)
	}))
	defer server.Close()

	app, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{
		Name:               "music-tools-wait-test",
		DefaultAPIBase:     server.URL,
		APIPrefix:          "/api/v1",
		DefaultHTTPTimeout: 10 * time.Millisecond,
		AllowAnonymous:     true,
	})
	require.NoError(t, err)
	start := time.Now()
	require.NoError(t, waitForJob(app, []string{"job-1"}))
	require.GreaterOrEqual(t, time.Since(start), 80*time.Millisecond)
}

// TestCommandGroups exercises the flat-commands aggregator. The
// template ships zero flat commands, so the contract is "returns nil";
// the test exists so a future scenario that adds CommandGroups gets
// caller-side wiring for free (the call goes through unchanged).
func TestCommandGroups(t *testing.T) {
	got := CommandGroups(&cliapp.ScenarioApp{})
	require.Nil(t, got, "CommandGroups should return nil until a domain registers a flat group")
}

// TestSubcommandGroups proves the aggregator returns whatever domains
// are wired in domains.go without panicking, and that every registered
// group has the load-bearing fields (Name + Subcommands) populated.
//
// Deliberately flexible on count and name: scenarios add and remove
// domain packages over time, and pinning "exactly 1 group named X"
// breaks the moment a scenario swaps the canonical reference for its
// own first domain. The catch-the-typo failure mode (a domain registers
// but forgets to set Name or has no
// subcommands) still fails this test loudly.
func TestSubcommandGroups(t *testing.T) {
	manifest := readManifestForTest(t)
	got, err := SubcommandGroups(&cliapp.ScenarioApp{}, manifest)
	require.NoError(t, err, "SubcommandGroups must build cleanly from cli/manifest.json")
	require.NotNil(t, got, "SubcommandGroups must return a slice (possibly empty), not nil")
	for i, g := range got {
		require.NotEmpty(t, g.Name, "group[%d].Name must be set", i)
		require.NotEmpty(t, g.Subcommands, "group[%d] (%s) must register at least one subcommand", i, g.Name)
	}
}

// readManifestForTest reads cli/manifest.json from the parent cli/ directory
// (this test runs in cli/domains/). Mirrors what the embed in main does,
// but at test time the embed isn't available since this package is not main.
func readManifestForTest(t *testing.T) []byte {
	t.Helper()
	bytes, err := os.ReadFile(filepath.Join("..", "manifest.json"))
	require.NoError(t, err, "read cli/manifest.json")
	return bytes
}
