package agenttranscript

import "testing"

func TestGeminiCLIResumeSelectionSupportsNativeSessionIDs(t *testing.T) {
	format, ok := FormatFor("gemini-cli")
	if !ok || format.ResumeFlag != "--resume" || format.ResumeSelection != "session-id-or-index-or-latest" {
		t.Fatalf("Gemini CLI continuation metadata = %+v, found=%v", format, ok)
	}
}

func TestGeminiCLINativeResumeInvocationUsesSharedFormat(t *testing.T) {
	got, err := BuildNativeResumeInvocation("gemini-cli", "native-session", "continue this work")
	if err != nil {
		t.Fatal(err)
	}
	if got.Command != "gemini" || len(got.Args) != 4 || got.Args[0] != "--resume" || got.Args[1] != "native-session" || got.Args[2] != "-p" || got.Args[3] != "continue this work" {
		t.Fatalf("Gemini CLI resume invocation = %+v", got)
	}
}

func TestCursorDescriptorIdentifiesNativeKVStoreWithoutInventingResumeCLI(t *testing.T) {
	format, ok := FormatFor("cursor")
	if !ok || format.MessageDialect != "cursor-composer-kv" || !format.ArchiveSnapshot || format.ResumeFlag != "" || format.ResumeSelection != "" {
		t.Fatalf("Cursor continuation metadata = %+v, found=%v", format, ok)
	}
	if len(format.HomeRoots) != 2 || format.HomeRoots[0] != ".config/Cursor/User/workspaceStorage" || format.HomeRoots[1] != ".cursor/User/workspaceStorage" {
		t.Fatalf("Cursor home roots = %#v", format.HomeRoots)
	}
}

func TestAntigravityNativeResumeInvocationUsesInstalledCLIContract(t *testing.T) {
	prompt := `continue "this work"; $() stays literal`
	got, err := BuildNativeResumeInvocation("antigravity", "native-session", prompt)
	if err != nil {
		t.Fatal(err)
	}
	want := ResumeInvocation{Command: "agy", Args: []string{"--conversation", "native-session", "--print", prompt}}
	if got.Command != want.Command || len(got.Args) != len(want.Args) {
		t.Fatalf("Antigravity resume invocation = %+v, want %+v", got, want)
	}
	for i := range want.Args {
		if got.Args[i] != want.Args[i] {
			t.Fatalf("Antigravity resume args = %#v, want %#v", got.Args, want.Args)
		}
	}
}

func TestGrokResumeInvocationUsesSharedPromptFirstFormat(t *testing.T) {
	prompt := `continue "the ACP session"; $() stays literal`
	got, err := BuildNativeResumeInvocation(HarnessGrok, "native-grok-session", prompt)
	if err != nil {
		t.Fatal(err)
	}
	want := ResumeInvocation{Command: "grok", Args: []string{"-p", prompt, "--resume", "native-grok-session"}}
	if got.Command != want.Command || len(got.Args) != len(want.Args) {
		t.Fatalf("Grok resume invocation = %+v, want %+v", got, want)
	}
	for i := range want.Args {
		if got.Args[i] != want.Args[i] {
			t.Fatalf("Grok resume args = %#v, want %#v", got.Args, want.Args)
		}
	}
}

func TestAntigravityArchiveDescriptorRequiresNativeArchivePath(t *testing.T) {
	format, ok := FormatFor("antigravity")
	if !ok || !format.ArchiveOnly || !RequiresNativeArchive("antigravity") || RequiresNativeArchive("codex") {
		t.Fatalf("Antigravity archive discovery policy = %+v, found=%v", format, ok)
	}
	if !IsNativeArchiveFile("antigravity", "/home/operator/.gemini/antigravity-cli/brain/thread/.system_generated/logs/transcript_full.jsonl") {
		t.Fatal("native full transcript path should match shared archive descriptor")
	}
	if IsNativeArchiveFile("antigravity", "/home/operator/.gemini/antigravity-cli/brain/thread/.system_generated/logs/transcript.jsonl") {
		t.Fatal("reduced transcript sibling should not match shared archive descriptor")
	}
}
