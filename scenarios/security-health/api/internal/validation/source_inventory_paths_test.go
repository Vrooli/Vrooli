package validation

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type whitespacePathGitleaksCommander struct {
	git     Commander
	content string
}

func (c *whitespacePathGitleaksCommander) LookPath(name string) (string, error) {
	if name == "git" {
		return c.git.LookPath(name)
	}
	return "/logical/gitleaks", nil
}

func (c *whitespacePathGitleaksCommander) Run(ctx context.Context, dir, name string, args ...string) ([]byte, []byte, int, error) {
	if name == "git" {
		return c.git.Run(ctx, dir, name, args...)
	}
	const fileName = " leading secret.env "
	content, err := os.ReadFile(filepath.Join(dir, fileName))
	if errors.Is(err, os.ErrNotExist) {
		return []byte("[]"), nil, 0, nil
	}
	if err != nil {
		return nil, nil, 1, err
	}
	c.content = string(content)
	if !strings.Contains(c.content, "OUTCOME-SENSITIVE-SECRET") {
		return []byte("[]"), nil, 0, nil
	}
	return []byte(`[{"Description":"detected secret","StartLine":1,"File":" leading secret.env ","RuleID":"generic-api-key","Secret":"OUTCOME-SENSITIVE-SECRET"}]`), nil, 1, nil
}

func TestGitleaksScansGitPathWithLeadingAndTrailingSpaces(t *testing.T) {
	if _, err := NewExecCommander().LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	const fileName = " leading secret.env "
	const secret = "OUTCOME-SENSITIVE-SECRET"
	dir := t.TempDir()
	runGit := func(args ...string) {
		t.Helper()
		if _, stderr, code, err := NewExecCommander().Run(context.Background(), dir, "git", args...); err != nil || code != 0 {
			t.Fatalf("git %v: code=%d err=%v stderr=%s", args, code, err, stderr)
		}
	}
	runGit("init", "-q")
	writeFile(t, filepath.Join(dir, fileName), "TOKEN="+secret+"\n")
	runGit("add", "--", fileName)

	cmd := &whitespacePathGitleaksCommander{git: NewExecCommander()}
	findings, err := newGitleaksScanner(cmd).Scan(context.Background(), dir, Substrate{})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.content != "TOKEN="+secret+"\n" {
		t.Fatalf("Gitleaks input for %q = %q, want exact committed bytes", fileName, cmd.content)
	}
	if len(findings) != 1 || findings[0].FilePath != fileName+":1" || findings[0].Severity != SeverityError {
		t.Fatalf("secret finding = %+v, want redacted ERROR at exact whitespace path", findings)
	}
	encoded, err := json.Marshal(findings)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), secret) {
		t.Fatalf("finding leaked raw secret: %s", encoded)
	}
}
