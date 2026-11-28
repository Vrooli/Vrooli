package support

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

func TestHelpers(t *testing.T) {
	if got := ShortID("123456789"); got != "12345678" {
		t.Fatalf("ShortID = %q", got)
	}
	if got := ShortID(" short "); got != "short" {
		t.Fatalf("ShortID trim = %q", got)
	}
	if got := FormatTime(""); got != "unknown" {
		t.Fatalf("empty FormatTime = %q", got)
	}
	if got := FormatTime("not-a-time"); got != "not-a-time" {
		t.Fatalf("invalid FormatTime = %q", got)
	}
	if got := FormatTime("2024-01-02T03:04:05-05:00"); got != "2024-01-02T08:04:05Z" {
		t.Fatalf("valid FormatTime = %q", got)
	}

	fs := NewFlagSet("test")
	fs.String("name", "", "")
	if err := ParseFlags(fs, []string{"value", "--name", "x"}); err != nil {
		t.Fatal(err)
	}
	if fs.Arg(0) != "value" || fs.Lookup("name").Value.String() != "x" {
		t.Fatalf("ParseFlags did not preserve positional and flag values")
	}

	raw, err := ReadJSONFile("", false)
	if err != nil || raw != nil {
		t.Fatalf("optional empty body = %s, %v", raw, err)
	}
	if _, err := ReadJSONFile("", true); err == nil {
		t.Fatal("required empty body unexpectedly succeeded")
	}
	file := filepath.Join(t.TempDir(), "body.json")
	if err := os.WriteFile(file, []byte(`{"ok":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	raw, err = ReadJSONFile(file, true)
	if err != nil || !json.Valid(raw) {
		t.Fatalf("valid body = %s, %v", raw, err)
	}
	if _, err := ReadJSONFile(filepath.Join(t.TempDir(), "missing"), true); err == nil {
		t.Fatal("missing body unexpectedly succeeded")
	}
	bad := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(bad, []byte("nope"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadJSONFile(bad, true); err == nil {
		t.Fatal("invalid body unexpectedly succeeded")
	}

	output := filepath.Join(t.TempDir(), "nested", "out.txt")
	if err := WriteOutput(output, []byte("ok")); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(output)
	if err != nil || string(data) != "ok" {
		t.Fatalf("WriteOutput = %q, %v", data, err)
	}
	if !strings.Contains(CLIName, "web-console") {
		t.Fatal("CLIName not configured")
	}
}

func TestApplyAliases(t *testing.T) {
	commands := []cliapp.Command{{Name: "list"}, {Name: "other", Aliases: []string{"old"}}}
	ApplyAliases(commands, map[string][]string{"list": {"ls"}, "other": {"new"}})
	if len(commands[0].Aliases) != 1 || commands[0].Aliases[0] != "ls" {
		t.Fatalf("list aliases = %#v", commands[0].Aliases)
	}
	if len(commands[1].Aliases) != 2 {
		t.Fatalf("other aliases = %#v", commands[1].Aliases)
	}
}
