package capabilities

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	clitest "github.com/vrooli/cli-core/cliapptest"
)

func TestRegisterExposesCapabilityCommands(t *testing.T) {
	group := Register(&cliapp.ScenarioApp{})
	if group.Name != "capabilities" || len(group.Subcommands) != 3 {
		t.Fatalf("group = %+v", group)
	}
}

func TestCapabilitySecretFlagIsRejected(t *testing.T) {
	core := clitest.NewTestApp(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	if err := action(core, []string{"--id", "demo", "--secret", "value"}, false); err == nil || !strings.Contains(err.Error(), "standard input") {
		t.Fatalf("error = %v, want standard-input rejection", err)
	}
}

func TestCapabilitySecretComesFromStandardInput(t *testing.T) {
	var previewBody string
	core := clitest.NewTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/v2/capabilities":
			_, _ = w.Write([]byte(`{"capabilities":[{"descriptor":{"id":"demo","inputs":[{"id":"token","kind":"secret"}]}}]}`))
		case "/api/v1/v2/capabilities/preview":
			body, _ := io.ReadAll(r.Body)
			previewBody = string(body)
			_, _ = w.Write([]byte(`{"capability_id":"demo","mutations":[]}`))
		default:
			_, _ = w.Write([]byte(`{"state":"ready"}`))
		}
	}))
	oldStdin := os.Stdin
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_, _ = writer.WriteString("secret-from-stdin\n")
	_ = writer.Close()
	os.Stdin = reader
	t.Cleanup(func() { os.Stdin = oldStdin; _ = reader.Close() })
	if err := action(core, []string{"--id", "demo", "--json"}, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(previewBody, `"token":"secret-from-stdin"`) {
		t.Fatalf("preview body = %q", previewBody)
	}
}

func TestCapabilityApplyPreviewsBeforeApplying(t *testing.T) {
	var calls []string
	core := clitest.NewTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/v2/capabilities":
			_, _ = w.Write([]byte(`{"capabilities":[{"descriptor":{"id":"demo","policy":{"requires_confirmation":true},"inputs":[]}}]}`))
		case "/api/v1/v2/capabilities/preview":
			calls = append(calls, "preview")
			_, _ = w.Write([]byte(`{"capability_id":"demo","mutations":[]}`))
		case "/api/v1/v2/capabilities/apply":
			calls = append(calls, "apply")
			_, _ = w.Write([]byte(`{"state":"ready","outcome":"applied"}`))
		default:
			_, _ = w.Write([]byte(`{"state":"ready"}`))
		}
	}))
	if err := action(core, []string{"--id", "demo", "--confirm", "--json"}, true); err != nil {
		t.Fatal(err)
	}
	if strings.Join(calls, ",") != "preview,apply" {
		t.Fatalf("capability calls = %v, want preview before apply", calls)
	}
}
