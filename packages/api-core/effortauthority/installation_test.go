package effortauthority

import (
	"bytes"
	"testing"
	"time"
)

func readinessInstallationFixture() Installation {
	now := time.Now().UTC().Truncate(time.Second)
	p := Policy{ID: "installed-policy", Owner: "owner", Client: "client", ClientKey: bytes.Repeat([]byte{1}, 32), Epoch: 1, Repository: "/approved/repository", Effort: "effort", Revision: "revision", ContentDigest: Digest("content"), Team: "team", TeamDigest: Digest("team"), BindingDigest: Digest("binding"), Members: []string{"leader"}, Profiles: map[string]string{"bounded": Digest("profile")}, Scopes: []string{"agent-manager:write"}, Effects: []string{"run.create"}, NotBefore: now, Deadline: now.Add(8 * time.Hour), MaxStarts: 8, MaxConcurrent: 1, MaxTurns: 32, MaxToolCalls: 100, MaxRunSeconds: 3600, TotalTurns: 256, TotalToolCalls: 800, TotalRunSeconds: 28800, SurviveLogout: true}
	h := Digest("installed-build")
	return Installation{Enabled: true, Policy: p, Origin: "https://owner.example", Builds: map[string]string{"issuer": h, "pm": h, "am": h}, ServerPins: map[string]string{"issuer": h, "pm": h, "am": h}}
}
func TestReadinessInstallationFreezeCopiesEveryMutableBinding(t *testing.T) {
	original := readinessInstallationFixture()
	got, e := original.Freeze()
	if e != nil {
		t.Fatal(e)
	}
	original.Policy.ClientKey[0] = 2
	original.Policy.Members[0] = "foreign"
	original.Policy.Scopes[0] = "foreign"
	original.Policy.Effects[0] = "foreign"
	original.Policy.Profiles["bounded"] = "foreign"
	original.Builds["am"] = "foreign"
	original.ServerPins["pm"] = "foreign"
	if got.Policy.ClientKey[0] != 1 || got.Policy.Members[0] != "leader" || got.Policy.Scopes[0] != "agent-manager:write" || got.Policy.Effects[0] != "run.create" || got.Policy.Profiles["bounded"] != Digest("profile") || got.Builds["am"] != Digest("installed-build") || got.ServerPins["pm"] != Digest("installed-build") {
		t.Fatal("frozen installation aliases offered source")
	}
	disabled := Installation{Policy: Policy{Members: []string{"old"}}}
	copy, e := disabled.Freeze()
	if e != nil || copy.Enabled {
		t.Fatal("disabled default rejected", e)
	}
	disabled.Policy.Members[0] = "changed"
	if copy.Policy.Members[0] != "old" {
		t.Fatal("disabled copy aliases source")
	}
}
func TestReadinessInstallationFreezeRejectsBroaderTopologyAndBounds(t *testing.T) {
	cases := map[string]func(*Installation){
		"http": func(s *Installation) { s.Origin = "http://owner.example" }, "userinfo": func(s *Installation) { s.Origin = "https://user@owner.example" }, "path": func(s *Installation) { s.Origin = "https://owner.example/path" }, "query": func(s *Installation) { s.Origin = "https://owner.example?x=1" }, "fragment": func(s *Installation) { s.Origin = "https://owner.example#x" }, "missing-build": func(s *Installation) { delete(s.Builds, "am") }, "wrong-pin": func(s *Installation) { s.ServerPins["pm"] = "invalid" }, "starts": func(s *Installation) { s.Policy.MaxStarts = 9 }, "concurrent": func(s *Installation) { s.Policy.MaxConcurrent = 2 }, "deadline": func(s *Installation) { s.Policy.Deadline = s.Policy.NotBefore.Add(8*time.Hour + time.Second) }, "turns": func(s *Installation) { s.Policy.MaxTurns = 33 }, "calls": func(s *Installation) { s.Policy.MaxToolCalls = 101 }, "run-seconds": func(s *Installation) { s.Policy.MaxRunSeconds = 3601 }, "total-turns": func(s *Installation) { s.Policy.TotalTurns = 257 }, "total-calls": func(s *Installation) { s.Policy.TotalToolCalls = 801 }, "total-seconds": func(s *Installation) { s.Policy.TotalRunSeconds = 28801 }, "policy": func(s *Installation) { s.Policy.Owner = "" },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			s := readinessInstallationFixture()
			change(&s)
			if _, e := s.Freeze(); e == nil {
				t.Fatal("invalid installation accepted")
			}
		})
	}
}
func TestReadinessMainGateNeverActivatesWithoutSuppliedOwner(t *testing.T) {
	for _, value := range []string{"", "0", "false", " FALSE "} {
		for _, supplied := range []bool{false, true} {
			if MainInstallationGate(value, supplied) != nil {
				t.Fatalf("disabled value refused %q", value)
			}
		}
	}
	for _, value := range []string{"1", "true", " TRUE "} {
		if MainInstallationGate(value, false) == nil {
			t.Fatalf("activation lacked installation %q", value)
		}
		if MainInstallationGate(value, true) != nil {
			t.Fatalf("supplied typed installation rejected %q", value)
		}
	}
	for _, value := range []string{"yes", "on", "2", "true false"} {
		if MainInstallationGate(value, true) == nil {
			t.Fatalf("ambiguous activation accepted %q", value)
		}
	}
}
