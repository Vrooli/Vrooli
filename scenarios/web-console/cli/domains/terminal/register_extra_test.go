package terminal

import "testing"

func TestParseKeySpec(t *testing.T) {
	key, err := parseKeySpec("Ctrl+Alt+Enter")
	if err != nil || key.Name != "Enter" || !key.Ctrl || !key.Alt || key.Shift {
		t.Fatalf("parsed key = %#v, %v", key, err)
	}
	for _, spec := range []string{"Ctrl+", "Bogus+Enter", "+Enter"} {
		if _, err := parseKeySpec(spec); err == nil {
			t.Fatalf("parseKeySpec(%q) unexpectedly succeeded", spec)
		}
	}
}

func TestTerminalArgumentValidation(t *testing.T) {
	if len(Register(nil).Subcommands) != 5 {
		t.Fatal("terminal command group is incomplete")
	}
	if err := runTargets([]string{"--json"}); err != nil {
		t.Fatal(err)
	}
	if err := runTargets(nil); err != nil {
		t.Fatal(err)
	}
	if err := runScreen(nil, nil); err == nil {
		t.Fatal("screen without session unexpectedly succeeded")
	}
	if err := runSendText(nil, []string{"session"}); err == nil {
		t.Fatal("send-text without text unexpectedly succeeded")
	}
	if err := runSendKeys(nil, []string{"session"}); err == nil {
		t.Fatal("send-keys without key unexpectedly succeeded")
	}
	if err := runWaitIdle(nil, nil); err == nil {
		t.Fatal("wait-idle without session unexpectedly succeeded")
	}
}
