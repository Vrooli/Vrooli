package credentials

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	clitest "github.com/vrooli/cli-core/cliapptest"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

func TestRegisterExposesSafeCredentialCommands(t *testing.T) {
	group := Register(nil)
	if len(group.Subcommands) != 3 {
		t.Fatalf("credential commands = %+v", group.Subcommands)
	}
	for _, command := range group.Subcommands {
		if command.Name == "provision" {
			if err := command.Run([]string{"--logical-id", "demo", "--value", "secret"}); err == nil || !strings.Contains(err.Error(), "standard input") {
				t.Fatalf("value-bearing provision flag was not rejected: %v", err)
			}
			return
		}
	}
	t.Fatal("provision command was not registered")
}

type fakeResolver struct{}

func (fakeResolver) Resolve(credentialauthority.Identity, string) (string, error) {
	return "revealed-test-value", nil
}

func TestRevealRequiresExplicitConfirmationAndWritesOnlyToProvidedTerminal(t *testing.T) {
	previous := revealAuthority
	revealAuthority = func() (credentialResolver, error) { return fakeResolver{}, nil }
	t.Cleanup(func() { revealAuthority = previous })

	var output bytes.Buffer
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{
			{Name: "logical-id"},
			{Name: "field"},
			{Name: "confirm-reveal", Bool: true},
		}},
		Flags:     map[string]string{"logical-id": "vrooli/demo", "field": "token"},
		Stdout:    &output,
		BoolFlags: map[string]bool{"confirm-reveal": true},
	})
	if err := reveal(ctx); err != nil {
		t.Fatal(err)
	}
	if got := output.String(); got != "revealed-test-value\n" {
		t.Fatalf("reveal output = %q", got)
	}
}

func TestRevealRejectsJSONAndMissingConfirmation(t *testing.T) {
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{
			{Name: "logical-id"},
			{Name: "field"},
			{Name: "confirm-reveal", Bool: true},
		}},
		Flags: map[string]string{"logical-id": "vrooli/demo", "field": "token"},
		JSON:  true,
	})
	if err := reveal(ctx); err == nil || !strings.Contains(err.Error(), "--json") {
		t.Fatalf("JSON reveal error = %v", err)
	}

	ctx = cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{
			{Name: "logical-id"},
			{Name: "field"},
			{Name: "confirm-reveal", Bool: true},
		}},
		Flags: map[string]string{"logical-id": "vrooli/demo", "field": "token"},
	})
	if err := reveal(ctx); err == nil || !strings.Contains(err.Error(), "--confirm-reveal") {
		t.Fatalf("confirmation error = %v", err)
	}
}

func TestListDoctorAndProvisionUseSafeTransport(t *testing.T) {
	var provisionBody string
	core := clitest.NewTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/vrooli.vrooli_onboarding.v1.credentials.CredentialsService/ProvisionCredential":
			data, _ := io.ReadAll(r.Body)
			provisionBody = string(data)
			_, _ = w.Write([]byte(`{"status":"provisioned","logicalId":"demo","field":"key"}`))
		case "/vrooli.vrooli_onboarding.v1.credentials.CredentialsService/DiagnoseCredentials":
			_, _ = w.Write([]byte(`{"provider":{"condition":"available"}}`))
		default:
			_, _ = w.Write([]byte(`{"credentials":[]}`))
		}
	}))
	group := Register(core)
	if err := group.Subcommands[0].Run([]string{"--json"}); err != nil {
		t.Fatal(err)
	}
	if err := group.Subcommands[2].Run([]string{"--json"}); err != nil {
		t.Fatal(err)
	}
	oldStdin := os.Stdin
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_, _ = writer.WriteString("secret-from-stdin\n")
	_ = writer.Close()
	os.Stdin = reader
	t.Cleanup(func() { os.Stdin = oldStdin; _ = reader.Close() })
	if err := group.Subcommands[1].Run([]string{"--logical-id", "demo", "--json"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(provisionBody, "secret-from-stdin") {
		t.Fatalf("stdin value was not submitted: %q", provisionBody)
	}
}
