package securestore

import (
	"os"
	"strings"
	"testing"
)

// copyScheduledVerb is the CLI path the scheduled refresh must invoke. The
// credentials CLI dispatches `credentials <group> <command>` and nothing
// deeper, so the refresh has to be one command name, not "copy" followed by a
// stray "scheduled" positional.
const copyScheduledVerb = "credentials store copy-scheduled"

// Every platform's schedule emitter, and the operator remediation string, must
// name a command the CLI can actually route. All four emitted
// "credentials store copy scheduled" — four tokens — which the dispatcher
// cannot reach, so the installed launchd agent, systemd timer and scheduled
// task failed on every fire for as long as they existed, and no host ever had
// a working credential-store copy (2026-09-16). The emitters agreed with each
// other and with nothing else, which is why this pins the text.
func TestScheduleEmittersInvokeARoutableCopyCommand(t *testing.T) {
	for _, file := range []string{
		"schedule_darwin.go",
		"schedule_linux.go",
		"schedule_windows.go",
		"schedule.go",
	} {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		source := string(data)

		// The dead four-token form must not come back, in argv or in prose.
		if strings.Contains(source, "credentials store copy scheduled") {
			t.Fatalf("%s still invokes the unroutable four-token form", file)
		}
		// The darwin plist splits argv into separate XML elements.
		if strings.Contains(source, "<string>copy</string><string>scheduled</string>") {
			t.Fatalf("%s still splits copy and scheduled into separate arguments", file)
		}
	}
}

// The darwin plist and the linux unit both have to carry the routable verb,
// each in its own encoding.
func TestPlatformSchedulesCarryTheRoutableVerb(t *testing.T) {
	darwin, err := os.ReadFile("schedule_darwin.go")
	if err != nil {
		t.Fatalf("read schedule_darwin.go: %v", err)
	}
	if !strings.Contains(string(darwin), "<string>copy-scheduled</string>") {
		t.Fatal("the launchd plist must invoke copy-scheduled as one argument")
	}

	linux, err := os.ReadFile("schedule_linux.go")
	if err != nil {
		t.Fatalf("read schedule_linux.go: %v", err)
	}
	if !strings.Contains(string(linux), copyScheduledVerb) {
		t.Fatalf("the systemd unit must invoke %q", copyScheduledVerb)
	}
}
