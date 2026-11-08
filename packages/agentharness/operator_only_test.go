package agentharness

import "testing"

func TestOperatorOnlyCommandsAskTheOperatorUnderEveryProfile(t *testing.T) {
	for _, shell := range []string{
		"make rebaseline",
		"cd /home/op/Vrooli && make rebaseline",
		"make -C /home/op/Vrooli rebaseline",
		"bash -c 'make rebaseline'",
		"env FOO=1 make rebaseline",
		"vrooli runtime update rebaseline /repo/.vrooli/build/vrooli",
		"/home/op/.vrooli/bin/vrooli runtime update disable",
		"vrooli runtime update enroll --json",
		"vrooli runtime update retry ./candidate",
		"go run ./cmd/vrooli runtime update rebaseline .vrooli/build/vrooli",
		"go run ./cmd/vrooli/ runtime update disable",
		// Home-relative and variable paths: the last path segment decides.
		"~/.vrooli/bin/vrooli runtime update disable",
		"$HOME/.vrooli/bin/vrooli runtime update disable",
		`"$HOME/.vrooli/bin/vrooli" runtime update enroll`,
		"~/Vrooli/.vrooli/build/vrooli runtime update rebaseline x",
		"$UNSET_ROOT/build/vrooli runtime update retry ./c",
		// A name only known at run time fails closed on its arguments.
		"$(command -v vrooli) runtime update disable",
		// The installed watchdog's own enrollment and retry modes.
		"~/.vrooli/libexec/vrooli-watchdog --disable-core-updates",
		"/home/op/.vrooli/libexec/vrooli-watchdog --enroll-core-updates",
		"vrooli-watchdog --retry-core-update ./candidate /repo",
		// Wrappers and relaunchers.
		"setsid make rebaseline",
		"setsid -f vrooli runtime update disable",
		"script -qec 'make rebaseline' /dev/null",
		"script --command='vrooli runtime update enroll' /dev/null",
		"flock /tmp/rb.lock make rebaseline",
		"flock -w 5 /tmp/rb.lock -c 'make rebaseline'",
		"tmux send-keys -t ops 'make rebaseline' Enter",
		"tmux new-window 'vrooli runtime update disable'",
		"systemd-run --user --scope make rebaseline",
		"watch -n 60 make rebaseline",
		"ssh localhost make rebaseline",
		"parallel make ::: rebaseline",
		"echo rebaseline | xargs make",
		"echo disable | xargs vrooli runtime update",
		"echo | xargs -I{} vrooli runtime update disable",
	} {
		for _, profile := range []RolloutProfile{ProfileAdvisory, ProfileGuided, ProfileGuarded, ProfileEnforcing} {
			decision, err := Runtime{Profile: profile}.Evaluate(ToolEvent{Runner: "claude-code", Tool: "Bash", Shell: shell, WorkingDirectory: "/home/op/Vrooli"})
			if err != nil {
				t.Fatal(err)
			}
			if decision.Action != ActionAsk || decision.Risk != RiskOperatorOnly || decision.Reason == "" {
				t.Fatalf("%q under %s = %s/%s, want an operator-only ask", shell, profile, decision.Action, decision.Risk)
			}
		}
	}
}

func TestOrdinaryCommandsAreNotOperatorOnly(t *testing.T) {
	for _, shell := range []string{
		"make install",
		"make help",
		`grep -n "make rebaseline" docs/resources/deployment-contract.md`,
		"vrooli runtime update status --json",
		"vrooli runtime update apply ./candidate",
		"vrooli version --state-manifest",
		"echo rebaseline",
		"~/.vrooli/bin/vrooli runtime update status",
		"$(command -v vrooli) scenario status",
		"vrooli-watchdog --submit-enrolled-core-update ./candidate /repo",
		"ls scenarios | xargs -I{} make -C scenarios/{} test",
		"git diff --name-only | xargs make lint",
		"tmux new-window htop",
		"ssh host 'journalctl -u vrooli'",
		"setsid make install",
		"script -qec 'make test' /dev/null",
	} {
		decision, err := Runtime{Profile: ProfileAdvisory}.Evaluate(ToolEvent{Runner: "claude-code", Tool: "Bash", Shell: shell, WorkingDirectory: "/home/op/Vrooli"})
		if err != nil {
			t.Fatal(err)
		}
		if decision.Risk == RiskOperatorOnly {
			t.Fatalf("%q was classified operator-only", shell)
		}
	}
}
