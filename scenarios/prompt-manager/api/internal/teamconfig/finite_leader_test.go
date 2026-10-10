package teamconfig

import "testing"

func TestFiniteLeaderRequiresExactBoundedConfiguration(t *testing.T) {
	for _, field := range []string{"effort", "revision", "prompt", "source", "profile", "source-limit", "whitespace"} {
		t.Run(field, func(t *testing.T) {
			b := &FiniteLeader{EffortRef: "any:effort", AcceptedRevision: "r7", CoordinatorPromptRef: "pm:coordinator", SourceRefs: []string{"owner:assignment"}}
			profile := "qualified"
			switch field {
			case "effort":
				b.EffortRef = ""
			case "revision":
				b.AcceptedRevision = ""
			case "prompt":
				b.CoordinatorPromptRef = ""
			case "source":
				b.SourceRefs = nil
			case "profile":
				profile = ""
			case "source-limit":
				b.SourceRefs = make([]string, 9)
			case "whitespace":
				b.AcceptedRevision = " r7"
			}
			if err := b.Validate(profile); err == nil {
				t.Fatal("incomplete or conflicting binding accepted")
			}
		})
	}
}

func TestFiniteLeaderCheckinPolicyDefaultsToParkedAndValidates(t *testing.T) {
	b := &FiniteLeader{EffortRef: "any:effort", AcceptedRevision: "r7", CoordinatorPromptRef: "pm:coordinator", SourceRefs: []string{"owner:assignment"}, KeepAlive: true}
	if b.EventCheckins() || b.CheckinPollInterval() != DefaultCheckinPollInterval || b.CheckinBackstop() != DefaultCheckinBackstop {
		t.Fatalf("absent checkin policy is not parked with defaults")
	}
	b.Checkin = &CheckinPolicy{Mode: CheckinModeEvent, PollInterval: "5m", Backstop: "12h"}
	if err := b.Validate("qualified"); err != nil || !b.EventCheckins() || b.CheckinPollInterval().String() != "5m0s" || b.CheckinBackstop().String() != "12h0m0s" {
		t.Fatalf("event policy not honoured: %v", err)
	}
	for name, policy := range map[string]*CheckinPolicy{
		"mode":     {Mode: "wake"},
		"poll":     {Mode: CheckinModeEvent, PollInterval: "10s"},
		"backstop": {Mode: CheckinModeEvent, Backstop: "soon"},
	} {
		b.Checkin = policy
		if err := b.Validate("qualified"); err == nil {
			t.Fatalf("%s: invalid checkin policy accepted", name)
		}
	}
	b.KeepAlive, b.Checkin = false, &CheckinPolicy{Mode: CheckinModeEvent}
	if err := b.Validate("qualified"); err == nil || b.EventCheckins() {
		t.Fatal("event check-ins accepted without keepAlive")
	}
}
