package credentials

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	clitest "github.com/vrooli/cli-core/cliapptest"
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

func TestListDoctorAndProvisionUseSafeTransport(t *testing.T) {
	var provisionBody string
	core := clitest.NewTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/v2/credentials/provision" {
			data, _ := io.ReadAll(r.Body)
			provisionBody = string(data)
		}
		_, _ = w.Write([]byte(`{"status":"configured","credentials":[]}`))
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
