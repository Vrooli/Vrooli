package runsignal

import (
	"fmt"
	"strings"
	"testing"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// deriveOne runs one tool call (and optionally a result with an exit code)
// through the production derivation.
func deriveOne(t *testing.T, tool string, input map[string]any, exit *int) []InvocationFact {
	t.Helper()
	runID := uuid.New()
	events := []*domain.RunEvent{domain.NewToolCallEvent(runID, tool, "c1", input)}
	if exit != nil {
		result := domain.NewToolResultEvent(runID, tool, "c1", "out", nil)
		data := result.Data.(*domain.ToolResultEventData)
		data.ExitCode, data.Success = exit, *exit == 0
		events = append(events, result)
	}
	return DeriveInvocationFacts(events)
}

func intp(v int) *int { return &v }

func paths(facts []InvocationFact) string {
	out := make([]string, len(facts))
	for i, f := range facts {
		out[i] = f.CommandPath
	}
	return strings.Join(out, " | ")
}

func TestCommandPathAndIntentFromRealShapedShell(t *testing.T) {
	heredoc := "cat > /tmp/note.md <<'EOF'\nline with $VAR and `ticks` and && ; |\nEOF\nrg -n foo scenarios/ | head -20"
	for _, tc := range []struct {
		name    string
		command string
		want    string
	}{
		{"plain search", `rg -n "foo bar" scenarios/agent-manager`, "rg"},
		{"cd prefix is not a fact", `cd /home/x/Vrooli && vrooli scenario logs agent-manager --tail 50`, "vrooli scenario logs"},
		{"env assignment and timeout wrapper", `FOO=1 timeout 30 sudo -u bob git -C /repo status --short`, "git status"},
		{"env wrapper", `env -i PATH=/bin make -C scenarios/x restart`, "make restart"},
		{"heredoc body is ignored", heredoc, "cat | rg | head"},
		{"command substitution and quotes", `echo "$(date +%s) a && b" ; ls -la "my dir"`, "echo | ls"},
		{"newline separated script", "set -e\nfor f in a b; do\n  rg -n x \"$f\" || true\ndone\nwc -l a", "rg | wc"},
		{"bash -lc wrapper", `bash -lc 'cd /x && go test ./... 2>&1 | tail -5'`, "go test | tail"},
		{"redirects are not words", `go build ./... 2>&1 >/dev/null`, "go build"},
		{"non-subcommand program stays bare", `python3 scripts/run.py --flag`, "python3"},
		{"absolute program path", `/usr/bin/rg --files`, "rg"},
		{"comment", "# explain\nls", "ls"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			facts := deriveOne(t, "Bash", map[string]any{"command": tc.command}, nil)
			if got := paths(facts); got != tc.want {
				t.Fatalf("command_path=%q want %q", got, tc.want)
			}
			for _, f := range facts {
				if f.IntentClass != f.CommandPath || strings.Contains(f.IntentClass, "unparseable") {
					t.Fatalf("intent not derived from command path: %+v", f)
				}
				if f.Ownership == OwnershipUnparseable {
					t.Fatalf("tolerant lexer rejected %q: %+v", tc.command, f)
				}
			}
		})
	}
}

func TestSameCommandHasSameIntentRegardlessOfArguments(t *testing.T) {
	a := deriveOne(t, "Bash", map[string]any{"command": `rg -n "alpha" a/`}, nil)[0]
	b := deriveOne(t, "Bash", map[string]any{"command": "rg --files-with-matches 'beta gamma' b/ c/"}, nil)[0]
	if a.IntentClass != b.IntentClass || a.Fingerprint == b.Fingerprint {
		t.Fatalf("intent must match and fingerprint differ: %+v %+v", a, b)
	}
}

func TestCodexExecJavaScriptYieldsShellFactsAndCodeFacts(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input string
		paths string
		code  bool
	}{
		{"single cmd", `text(await tools.exec_command({cmd:"rg -n foo scenarios", workdir:"/x"}))`, "rg", false},
		{"quoted keys and escapes", `const r = await tools.exec_command({"cmd":"sed -n '1,20p' a.md && echo \"done\"","yield_time_ms":1000}); text(r.output);`, "sed | echo", false},
		{"two calls", "const a = await tools.exec_command({cmd: 'git status --short'});\nconst b = await tools.exec_command({cmd: `ls -la`});\ntext(a.output + b.output)", "git status | ls", false},
		{"argv form", `await tools.exec_command({cmd:["bash","-lc","cd /x && cargo test"]})`, "cargo test", false},
		{"pure code", `const rows = [1,2,3].map(x => x*2); text(JSON.stringify(rows));`, "", true},
		{"variable command is code", `const c = "rg x"; await tools.exec_command({cmd: c})`, "", true},
		{"command plus other tool call", `await tools.exec_command({cmd:"ls"}); await tools.apply_patch({patch: "x"})`, "ls", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			facts := deriveOne(t, "exec", map[string]any{"input": tc.input}, nil)
			var shell []InvocationFact
			sawCode := false
			for _, f := range facts {
				if f.Capability == "code" {
					sawCode = true
					if f.Executable != "" || f.CommandPath != "" || f.Ownership != OwnershipNotACommand {
						t.Fatalf("code fact carries shell identity: %+v", f)
					}
					continue
				}
				shell = append(shell, f)
			}
			if got := paths(shell); got != tc.paths || sawCode != tc.code {
				t.Fatalf("paths=%q code=%v want %q %v", got, sawCode, tc.paths, tc.code)
			}
			for _, f := range facts {
				switch f.Executable {
				case "await", "const", "let", "})", "text":
					t.Fatalf("JavaScript parsed as shell: %+v", f)
				}
			}
		})
	}
}

func TestSegmentOutcomes(t *testing.T) {
	for _, tc := range []struct {
		name    string
		command string
		exit    int
		want    []string
	}{
		{"single success", `ls`, 0, []string{"success"}},
		{"rg exit 1 is no-match", `rg -n zzz .`, 1, []string{"no-match"}},
		{"grep exit 1 after cd", `cd /x; grep -r foo .`, 1, []string{"no-match"}},
		{"diff exit 1", `diff a b`, 1, []string{"no-match"}},
		{"test exit 1", `[ -f x ]`, 1, []string{"no-match"}},
		{"rg exit 2 is failure", `rg -n ( .`, 2, []string{"failure"}},
		{"other exit 1 is failure", `git commit -m x`, 1, []string{"failure"}},
		{"and-chain success proves all", `go build ./... && go vet ./...`, 0, []string{"success", "success"}},
		{"and-chain failure blames neither", `go build ./... && go vet ./...`, 1, []string{"unknown", "unknown"}},
		{"and-chain after test is unknown", `test -f x && vrooli scenario logs foo`, 1, []string{"unknown", "unknown"}},
		{"or-chain success is unknown", `a || b`, 0, []string{"unknown", "unknown"}},
		{"semicolon failure blames the last", `a; b`, 1, []string{"unknown", "failure"}},
		{"pipeline failure blames the tail", `a | b`, 1, []string{"unknown", "failure"}},
		{"single command inherits failure", `a`, 1, []string{"failure"}},
		{"pipeline success proves only the last", `rg foo | head -3`, 0, []string{"unknown", "success"}},
		{"semicolon list", `make a; make b`, 0, []string{"unknown", "success"}},
		{"pipeline whose rg finds nothing", `rg zzz . | head`, 0, []string{"unknown", "success"}},
		{"last rg no-match", `ls; rg zzz .`, 1, []string{"unknown", "no-match"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			facts := deriveOne(t, "Bash", map[string]any{"command": tc.command}, intp(tc.exit))
			got := make([]string, len(facts))
			for i, f := range facts {
				got[i] = f.Outcome
				if f.Outcome == OutcomeNoMatch && f.FailureSignature != "" {
					t.Fatalf("no-match carries a failure signature: %+v", f)
				}
			}
			if fmt.Sprint(got) != fmt.Sprint(tc.want) {
				t.Fatalf("outcomes=%v want %v", got, tc.want)
			}
		})
	}
}

func TestWaitCapabilityMatchesWholeCommandToken(t *testing.T) {
	for path, want := range map[string]string{
		"program-runtime programs wait": "wait",
		"await":                         "shell",
		"awaiter run":                   "shell",
		"vrooli scenario status":        "shell",
	} {
		if got := CapabilityForCommand(path, "shell"); got != want {
			t.Fatalf("%q capability=%q want %q", path, got, want)
		}
	}
}

// A heredoc message inside $(...) with an apostrophe still parses, and a
// non-literal argv element makes a Codex exec_command unresolved, not a guess.
func TestParserEdgeCasesFromReview(t *testing.T) {
	segments, ok := lexShell("git commit -m \"$(cat <<'EOF'\nIt's fixed\nEOF\n)\"")
	if !ok || len(segments) != 1 || len(segments[0].Tokens) < 2 || segments[0].Tokens[0] != "git" || segments[0].Tokens[1] != "commit" {
		t.Fatalf("segments = %+v ok=%v", segments, ok)
	}
	if _, _, ok := parseJSCommandValue(`["rg", pattern, "."]`, 0); ok {
		t.Fatal("an argv with a variable element must not parse as a literal command")
	}
}

// One heredoc parser serves the lexer and $(...): any delimiter word the
// shell accepts ends its body in both places, so an apostrophe in the body
// never opens a quote.
func TestHeredocDelimitersParseAlikeInsideAndOutsideSubstitution(t *testing.T) {
	for _, delim := range []string{"EOF", "'END.TXT'", `"MSG v2"`, "PY:3"} {
		end := strings.Trim(delim, `'"`)
		for name, script := range map[string]string{
			"top level":    "cat <<" + delim + "\nIt's fixed\n" + end + "\nrg -n x .",
			"substitution": "git commit -m \"$(cat <<" + delim + "\nIt's fixed\n" + end + "\n)\" && rg -n x .",
		} {
			segments, ok := lexShell(script)
			if !ok || len(segments) != 2 || segments[1].Tokens[0] != "rg" {
				t.Errorf("%s %s: segments=%+v ok=%v", name, delim, segments, ok)
			}
		}
	}
}

// A probe for whether a program exists does not run it.
func TestExistenceProbesAreNotCalls(t *testing.T) {
	for _, words := range [][]string{{"command", "-v", "vrooli"}, {"command", "-V", "vrooli"}} {
		if got := stripWrappers(words); got != nil {
			t.Fatalf("stripWrappers(%v) = %v, want nil", words, got)
		}
	}
	if !noiseCommands["which"] || !noiseCommands["type"] {
		t.Fatal("which and type are probes, not calls")
	}
}
