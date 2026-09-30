package pathredact

import (
	"bytes"
	"encoding/json"
	"testing"
)

func testRedactor() Redactor {
	return Redactor{
		RepoRoots:     []string{"/home/alice/Vrooli"},
		HomeDirs:      []string{"/home/alice"},
		Usernames:     []string{"alice"},
		IdentityTerms: []string{"Alice Example"},
	}
}

func TestRedactStringPortablePathTokens(t *testing.T) {
	r := testRedactor()

	input := "/home/alice/Vrooli/scenarios/swarm-manager/ideas/demo/plan.md:12"
	got := r.RedactString(input)
	want := "path:scenarios/swarm-manager/ideas/demo/plan.md:12"
	if got != want {
		t.Fatalf("RedactString() = %q, want %q", got, want)
	}
}

func TestRedactStringHomeAndUsername(t *testing.T) {
	r := testRedactor()

	input := "read /home/alice/.vrooli/secrets.json as alice; owner Alice Example"
	got := r.RedactString(input)
	want := "read <vrooli-home>/secrets.json as <user>; owner <user>"
	if got != want {
		t.Fatalf("RedactString() = %q, want %q", got, want)
	}
}

func TestRedactBytesSkipsBinary(t *testing.T) {
	r := testRedactor()
	input := []byte{'a', 0, 'b', '/', 'h', 'o', 'm', 'e'}
	got, changed := r.RedactBytes("capture.bin", input)
	if changed {
		t.Fatal("RedactBytes() changed binary data")
	}
	if !bytes.Equal(got, input) {
		t.Fatal("RedactBytes() returned different binary bytes")
	}
}

func TestRedactJSONValue(t *testing.T) {
	r := testRedactor()
	value := map[string]any{
		"path": "/home/alice/Vrooli/scenarios/swarm-manager/review/round-001.json",
		"items": []any{
			map[string]any{"output": "/home/alice/tmp/out.txt"},
		},
	}

	got, changed, err := r.RedactJSONValue(value)
	if err != nil {
		t.Fatalf("RedactJSONValue() error = %v", err)
	}
	if !changed {
		t.Fatal("RedactJSONValue() did not report a change")
	}
	data, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("Marshal redacted value: %v", err)
	}
	text := string(data)
	for _, forbidden := range []string{"/home/alice", "alice"} {
		if bytes.Contains(data, []byte(forbidden)) {
			t.Fatalf("redacted JSON still contains %q: %s", forbidden, text)
		}
	}
}

func TestRedactStringIsIdempotentForCaptureOutput(t *testing.T) {
	r := testRedactor()
	input := "-rwxrwxr-x 1 alice alice 8891595 Apr  7 21:39 /home/alice/Vrooli/scenarios/prompt-manager/cli/prompt-manager\n\nls: cannot access '/home/alice/Vrooli/scenarios/prompt-manager/cli/pm': No such file or directory\n"
	once := r.RedactString(input)
	twice := r.RedactString(once)
	if once != twice {
		t.Fatalf("RedactString is not idempotent:\nonce: %q\ntwice: %q", once, twice)
	}
}
