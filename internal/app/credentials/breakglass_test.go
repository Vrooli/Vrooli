package credentials

import (
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/trustposture"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
)

func TestBreakGlassCommandRoundTripUsesRealIssuerAndVerifier(t *testing.T) {
	t.Setenv("VROOLI_BREAK_GLASS_DIR", t.TempDir())
	ctx := &CommandContext{Globals: rootcli.GlobalOptions{JSON: true}}
	var provisionOutput bytes.Buffer
	ctx.Stdout = &provisionOutput
	if err := (&App{}).runBreakGlassCommandWithInput(ctx, []string{
		"provision", "--account-id", "operator-1", "--audience", "vrooli:uninstall",
		"--target", "host-a", "--scopes", "vrooli:uninstall",
	}, strings.NewReader("correct horse\n")); err != nil {
		t.Fatal(err)
	}
	var status trustposture.KeyStatus
	paths, err := trustposture.ResolveKeyPaths()
	if err != nil {
		t.Fatal(err)
	}
	status, err = trustposture.Status(paths)
	if err != nil || !status.Complete || status.AccountID != "operator-1" || status.Audience != "vrooli:uninstall" || status.Target != "host-a" || len(status.Scopes) != 1 {
		t.Fatalf("status = %+v, err=%v", status, err)
	}

	var issueOutput bytes.Buffer
	ctx.Stdout = &issueOutput
	if err := (&App{}).runBreakGlassCommandWithInput(ctx, []string{
		"issue", "--purpose", "vrooli:uninstall", "--target", "host-a",
		"--scopes", "vrooli:uninstall", "--ttl", "10m",
	}, strings.NewReader("correct horse\n")); err != nil {
		t.Fatal(err)
	}
	var issued breakGlassCredentialOutput
	if err := json.Unmarshal(issueOutput.Bytes(), &issued); err != nil {
		t.Fatalf("issue output = %q: %v", issueOutput.String(), err)
	}
	credentialRaw, err := os.ReadFile(issued.Path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(issueOutput.String(), strings.TrimSpace(string(credentialRaw))) {
		t.Fatal("issue command printed the credential instead of writing it")
	}
	credentialInfo, err := os.Stat(issued.Path)
	if err != nil {
		t.Fatal(err)
	}
	if credentialInfo.Mode().Perm() != 0o600 {
		t.Fatalf("credential mode = %o, want 600", credentialInfo.Mode().Perm())
	}
	publicRaw, err := os.ReadFile(paths.Public)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := trustposture.Verify(ed25519.PublicKey(publicRaw), strings.TrimSpace(string(credentialRaw)), "vrooli:uninstall", "host-a", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != "operator-1" || claims.Target != "host-a" || len(claims.Scopes) != 1 {
		t.Fatalf("claims = %+v", claims)
	}
}

func TestBreakGlassCommandRequiresPipedPassphrase(t *testing.T) {
	if _, err := readBreakGlassPassphrase(strings.NewReader(""), "issue"); err == nil {
		t.Fatal("empty passphrase accepted")
	}
	if got, err := readBreakGlassPassphrase(strings.NewReader("passphrase\n"), "issue"); err != nil || got != "passphrase" {
		t.Fatalf("passphrase = %q, err=%v", got, err)
	}
}

func TestBreakGlassPassphraseInputModes(t *testing.T) {
	device, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer device.Close()
	if runtime.GOOS != "windows" {
		if _, err := readBreakGlassPassphrase(device, "issue"); err == nil || !strings.Contains(err.Error(), "pipe it in") {
			t.Fatalf("character-device input error = %v, want printed pipe instruction", err)
		}
	}

	pipeReader, pipeWriter, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pipeWriter.WriteString("pipe secret\n"); err != nil {
		t.Fatal(err)
	}
	if err := pipeWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if got, err := readBreakGlassPassphrase(pipeReader, "issue"); err != nil || got != "pipe secret" {
		t.Fatalf("pipe input = %q, err=%v", got, err)
	}
	if err := pipeReader.Close(); err != nil {
		t.Fatal(err)
	}

	file, err := os.CreateTemp(t.TempDir(), "passphrase-")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString("file secret\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := file.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	if got, err := readBreakGlassPassphrase(file, "issue"); err != nil || got != "file secret" {
		t.Fatalf("file input = %q, err=%v", got, err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestBreakGlassStatusWithoutMaterial(t *testing.T) {
	t.Setenv("VROOLI_BREAK_GLASS_DIR", t.TempDir())
	ctx := &CommandContext{Globals: rootcli.GlobalOptions{JSON: true}}
	var output bytes.Buffer
	ctx.Stdout = &output
	if err := (&App{}).runBreakGlassCommandWithInput(ctx, []string{"status"}, strings.NewReader("")); err != nil {
		t.Fatal(err)
	}
	var status trustposture.KeyStatus
	if err := json.Unmarshal(output.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if status.Complete || status.WrappedPrivate || status.Public || status.Metadata {
		t.Fatalf("empty status = %+v", status)
	}
}
