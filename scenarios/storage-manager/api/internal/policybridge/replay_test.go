package policybridge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// This is the proof the operator asked for: the 28 deletion questions Storage
// Manager raised between 23 October and 14 November (their path sets are in
// testdata/historical-asks.json, with the user's name replaced) are replayed
// through the real agent policy runner with the real declarations, and each
// question is classified as allowed without asking or still asking.
//
// Nothing here is deleted, and no path is mocked: each requested path is laid
// out under a throwaway home with the contents its producer makes, the real
// `Declarations` read the repository contract and the leftovers declaration, and
// the runner is built from the current source and given each `rm -rf`.

type historicalAsk struct {
	N     int      `json:"n"`
	Group string   `json:"group"`
	Count int      `json:"count"`
	Paths []string `json:"paths"`
	State string   `json:"state"`
}

// repoRootForTest is the Vrooli checkout this test file lives in.
func repoRootForTest(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for ; dir != "/"; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, ".vrooli", "repo-contract.json")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, "cmd", "vrooli-policy-runner")); err == nil {
				return dir
			}
		}
	}
	t.Skip("not inside a Vrooli checkout")
	return ""
}

// laidOut creates a requested path with the contents its producer makes.
func laidOut(t *testing.T, path string) {
	t.Helper()
	name := filepath.Base(path)
	write := func(rel string) {
		full := filepath.Join(path, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	switch {
	case strings.Contains(name, "_cf2-") || strings.Contains(name, "_cf5-"):
		write("variant.db")
		write("variant.db-wal")
		write("variant.db-shm")
	case strings.Contains(path, "/.vrooli/state/vrooli/") && (strings.Contains(name, "fixture") || name == "demo"):
		write("retention/enforcement-receipt.json")
	case strings.Contains(path, "/.claude/projects/"):
		write("session.jsonl")
	case strings.Contains(path, "/vrooli-onboarding/apply-runs/"):
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
	case strings.HasPrefix(name, ".proto-gen-"):
		write("go/x.pb.go")
	case strings.HasPrefix(name, "rcl-fixture-"):
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	case strings.Contains(name, ".") && !strings.HasPrefix(name, ".") && filepath.Ext(name) != "":
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	default:
		write("content.txt")
	}
}

func TestReplayOfTheHistoricalDeletionQuestions(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain unavailable")
	}
	repo := repoRootForTest(t)
	base := t.TempDir()
	for _, shared := range []string{"/tmp", "/var/tmp"} {
		if strings.HasPrefix(base, shared+"/") {
			t.Skip("the test home would sit inside a shared temporary directory, which the guard always allows")
		}
	}
	// The runner, from the current source.
	runner := filepath.Join(base, "vrooli-policy-runner")
	build := exec.Command("go", "build", "-o", runner, ".")
	build.Dir = filepath.Join(repo, "cmd", "vrooli-policy-runner")
	if out, err := build.CombinedOutput(); err != nil {
		t.Skipf("cannot build the policy runner: %v\n%s", err, out)
	}

	// A throwaway home holding a throwaway checkout with the real contract.
	home := filepath.Join(base, "home", "user")
	checkout := filepath.Join(home, "Vrooli")
	for _, dir := range []string{filepath.Join(checkout, ".git"), filepath.Join(checkout, ".vrooli"), filepath.Join(home, ".vrooli"), filepath.Join(base, "othertmp")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"repo-contract.json", "disposable-leftovers.json"} {
		data, err := os.ReadFile(filepath.Join(repo, ".vrooli", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(checkout, ".vrooli", name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	policyHome := filepath.Join(base, "policy")
	env := append(os.Environ(), "HOME="+home, "VROOLI_HOME="+filepath.Join(home, ".vrooli"), "VROOLI_AGENT_POLICY_HOME="+policyHome,
		"TMPDIR="+filepath.Join(base, "othertmp"), "VROOLI_ROOT="+checkout, "CLAUDE_CODE_SESSION_ID=", "VROOLI_AGENT_SESSION=", "CODEX_HOME=")
	t.Setenv("HOME", home)
	t.Setenv("VROOLI_HOME", filepath.Join(home, ".vrooli"))
	t.Setenv("VROOLI_ROOT", checkout)

	// The real declarations, published the way storage-manager publishes them.
	rules, warnings, err := Declarations(checkout)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := BuildSnapshot(time.Now(), rules, warnings, RemovalApprovalCommand)
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(base, "snapshot.json")
	if err := os.WriteFile(file, snapshot, 0o644); err != nil {
		t.Fatal(err)
	}
	publish := exec.Command(runner, "publish", "--snapshot-file", file)
	publish.Env = env
	if out, err := publish.CombinedOutput(); err != nil {
		t.Fatalf("publish: %v\n%s", err, out)
	}

	// decide asks the runner about one command, optionally as a session whose
	// transcript says it created the listed files.
	reasons := map[string]string{}
	decide := func(command string, transcript string) string {
		event := map[string]any{"runner": "claude-code", "tool": "Bash", "shell": command, "working_directory": checkout}
		if transcript != "" {
			event["context"] = map[string]string{"transcript_path": transcript, "session_id": "replay"}
		}
		payload, _ := json.Marshal(event)
		cmd := exec.Command(runner, "canary", "--runner", "claude-code")
		cmd.Env = env
		cmd.Stdin = bytes.NewReader(payload)
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("canary: %v\n%s", err, out)
		}
		var decision struct {
			Action string `json:"action"`
			Reason string `json:"reason"`
		}
		if err := json.Unmarshal(out, &decision); err != nil {
			t.Fatalf("decision: %v\n%s", err, out)
		}
		reasons[command] = decision.Reason
		return decision.Action
	}

	data, err := os.ReadFile(filepath.Join("testdata", "historical-asks.json"))
	if err != nil {
		t.Fatal(err)
	}
	var asks []historicalAsk
	if err := json.Unmarshal(data, &asks); err != nil {
		t.Fatal(err)
	}
	mapPath := func(p string) string {
		switch {
		case strings.HasPrefix(p, "/home/user"):
			return home + strings.TrimPrefix(p, "/home/user")
		case strings.HasPrefix(p, "/tmp/"):
			return filepath.Join(base, "othertmp", strings.TrimPrefix(p, "/tmp/"))
		}
		return p
	}

	// What the guard does for each question, by what the asker's session made.
	// A group-B question is the agent's own scratch, so the session's
	// transcript says it created those paths a moment before.
	sort.Slice(asks, func(i, j int) bool { return asks[i].N < asks[j].N })
	results := map[int]string{}
	why := map[int]string{}
	for _, ask := range asks {
		var paths []string
		for _, p := range ask.Paths {
			full := mapPath(p)
			if !strings.HasPrefix(full, filepath.Join(base, "othertmp")) {
				laidOut(t, full)
			} else if err := os.MkdirAll(full, 0o755); err != nil {
				t.Fatal(err)
			}
			paths = append(paths, full)
		}
		transcript := ""
		if ask.Group == "B" {
			started := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339Nano)
			var lines []string
			// The session made each folder with mkdir and wrote each file
			// in it with its Write tool, so the transcript lists them all.
			var made []string
			for _, p := range paths {
				_ = filepath.WalkDir(p, func(q string, d fs.DirEntry, err error) error {
					if err == nil && !d.IsDir() {
						made = append(made, q)
					}
					return nil
				})
			}
			for _, f := range made {
				lines = append(lines, fmt.Sprintf(`{"type":"user","timestamp":%q,"toolUseResult":{"type":"create","filePath":%q},"message":{"role":"user","content":[]}}`, started, f))
			}
			for _, p := range paths {
				lines = append(lines, fmt.Sprintf(`{"type":"assistant","timestamp":%q,"cwd":%q,"message":{"role":"assistant","content":[{"type":"tool_use","name":"Bash","input":{"command":"mkdir -p %s"}}]}}`, started, checkout, p))
			}
			transcript = filepath.Join(base, fmt.Sprintf("session-%d.jsonl", ask.N))
			if err := os.WriteFile(transcript, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		quoted := make([]string, len(paths))
		for i, p := range paths {
			quoted[i] = "'" + p + "'"
		}
		command := "rm -rf " + strings.Join(quoted, " ")
		results[ask.N] = decide(command, transcript)
		why[ask.N] = reasons[command]
	}

	// The same paths asked by a session whose transcript starts an hour ago but
	// lists none of them: the file system says they are new, yet nothing says
	// this session made them, so birth time alone must not loosen anything.
	idle := filepath.Join(base, "idle-session.jsonl")
	idleLine := fmt.Sprintf(`{"type":"user","timestamp":%q,"message":{"role":"user","content":"go"}}`, time.Now().UTC().Add(-time.Hour).Format(time.RFC3339Nano))
	if err := os.WriteFile(idle, []byte(idleLine+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, ask := range asks {
		if ask.Group != "B" {
			continue
		}
		var quoted []string
		for _, p := range ask.Paths {
			quoted = append(quoted, "'"+mapPath(p)+"'")
		}
		if got := decide("rm -rf "+strings.Join(quoted, " "), idle); got == "allow" {
			t.Errorf("ask %d: birth time alone allowed the removal of paths the session never listed", ask.N)
		}
	}

	allowed := map[string]int{}
	total := map[string]int{}
	for _, ask := range asks {
		total[ask.Group]++
		if results[ask.N] == "allow" {
			allowed[ask.Group]++
		}
		t.Logf("ask %2d group %s paths %4d -> %s: %s", ask.N, ask.Group, ask.Count, results[ask.N], why[ask.N])
	}
	t.Logf("allowed without asking: A %d/%d, B %d/%d, C %d/%d, D %d/%d, E %d/%d", allowed["A"], total["A"], allowed["B"], total["B"], allowed["C"], total["C"], allowed["D"], total["D"], allowed["E"], total["E"])

	// Desired behaviour. Everything that needs the operator's judgement still
	// asks: retiring content (D), real data (E) and the Trash (C, n=1).
	for _, ask := range asks {
		if (ask.Group == "D" || ask.Group == "E" || ask.N == 1) && results[ask.N] == "allow" {
			t.Errorf("ask %d (group %s) was allowed without asking; it needs the operator", ask.N, ask.Group)
		}
	}
	// The agent's own scratch is allowed, except inside data an owner protects:
	// ask 24 is a scratch file in the research home, which is declared protected,
	// so it keeps asking however sure the guard is that the session made it.
	for _, ask := range asks {
		want := "allow"
		if ask.N == 24 {
			want = "not-allow"
		}
		if ask.Group == "B" && (results[ask.N] == "allow") != (want == "allow") {
			t.Errorf("ask %d (own scratch): %s, want %s", ask.N, results[ask.N], want)
		}
	}
	// The declared leftovers are allowed: proto staging and verification
	// copies, the CF2 and CF5 rehearsal data folders, retention fixture state,
	// the onboarding test run records and the live-compaction test transcripts.
	for _, n := range []int{8, 9, 10, 13, 16, 25, 26} {
		if results[n] != "allow" {
			t.Errorf("ask %d is a declared leftover but still asks: %s", n, results[n])
		}
	}
	// Known gaps: leftovers that no tool names, so no declaration can open them
	// without opening real data. Each is a producer bug to fix at the source:
	// the 4,700 session folders (7), a stray empty .git (11), dead generator
	// copies (12, 15), stray non-scenario folders (14), a fixture data folder
	// named 001 (17) and stray test output in a component library (19). They keep
	// asking until their producers stop making them.
	for _, n := range []int{7, 11, 12, 14, 15, 17, 19} {
		if results[n] == "allow" {
			t.Errorf("ask %d has no declaration and must still ask, but was allowed", n)
		}
	}
}
