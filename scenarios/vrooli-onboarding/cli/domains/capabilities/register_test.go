package capabilities

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	clitest "github.com/vrooli/cli-core/cliapptest"
	capabilitiesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/capabilities/capabilitiesv1connect"
)

func TestRegisterExposesCapabilityCommands(t *testing.T) {
	group := Register(&cliapp.ScenarioApp{})
	if group.Name != "capabilities" || len(group.Subcommands) != 4 {
		t.Fatalf("group = %+v", group)
	}
	for _, command := range group.Subcommands {
		if command.Name == "verify" {
			return
		}
	}
	t.Fatal("capabilities group does not expose verify")
}

func TestVerifyBuildsBoundedContextRequest(t *testing.T) {
	var body string
	core := clitest.NewTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == capabilitiesconnect.CapabilitiesServiceVerifyCapabilityProcedure {
			data, _ := io.ReadAll(r.Body)
			body = string(data)
			_, _ = w.Write([]byte(`{"capabilityId":"demo","evidence":[]}`))
			return
		}
		_, _ = w.Write([]byte(`{"state":"ready"}`))
	}))
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Core: core,
		Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{
			{Name: "capability-id"},
			{Name: "target"},
			{Name: "operation"},
			{Name: "environment"},
			{Name: "account-identity"},
		}},
		Flags: map[string]string{"capability-id": "demo", "target": "remote", "operation": "read"},
		JSON:  true,
	})
	if err := verify(core, ctx); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"capabilityId":"demo"`, `"targetId":"remote"`, `"operation":"read"`, `"effectClass":"read_only"`, `"timeoutSeconds":"30"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("verify request = %q, want %q", body, want)
		}
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
		case "/vrooli.vrooli_onboarding.v1.capabilities.CapabilitiesService/ListCapabilities":
			_, _ = w.Write([]byte(`{"capabilities":[{"descriptor":{"id":"demo","inputs":[{"id":"token","kind":"secret"}]}}]}`))
		case "/vrooli.vrooli_onboarding.v1.capabilities.CapabilitiesService/PreviewCapability":
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
		case "/vrooli.vrooli_onboarding.v1.capabilities.CapabilitiesService/ListCapabilities":
			_, _ = w.Write([]byte(`{"capabilities":[{"descriptor":{"id":"demo","policy":{"requires_confirmation":true},"inputs":[]}}]}`))
		case "/vrooli.vrooli_onboarding.v1.capabilities.CapabilitiesService/PreviewCapability":
			calls = append(calls, "preview")
			_, _ = w.Write([]byte(`{"capability_id":"demo","mutations":[]}`))
		case "/vrooli.vrooli_onboarding.v1.capabilities.CapabilitiesService/ApplyCapability":
			calls = append(calls, "apply")
			_, _ = w.Write([]byte(`{"state":"CAPABILITY_STATE_READY","outcome":"applied"}`))
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
