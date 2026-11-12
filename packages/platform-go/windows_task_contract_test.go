package platform

import (
	"os"
	"strings"
	"testing"
)

func TestWindowsTaskUsesOperatorS4ULeastPrivilegeAndUnlimitedDaemon(t *testing.T) {
	raw, e := os.ReadFile("testdata/servicedef/windows.vrooli-autoheal.xml")
	if e != nil {
		t.Fatal(e)
	}
	d, e := parseWindowsTaskDefinition(string(raw))
	if e != nil {
		t.Fatal(e)
	}
	if d.Principals[0].UserID != "alice" || d.Principals[0].RunLevel != "LeastPrivilege" || d.Settings.Limit != "PT0S" || d.Settings.Multiple != "IgnoreNew" || len(d.Boot) != 1 || d.Boot[0].Enabled != "true" {
		t.Fatalf("unexpected boot policy: %+v", d)
	}
}
func TestWindowsTaskRefusesWideningBeforeNativeRegistration(t *testing.T) {
	raw, e := os.ReadFile("testdata/servicedef/windows.vrooli-autoheal.xml")
	if e != nil {
		t.Fatal(e)
	}
	base := string(raw)
	for name, content := range map[string]string{
		"elevation":           strings.Replace(base, "LeastPrivilege", "HighestAvailable", 1),
		"interactive":         strings.Replace(base, "S4U", "InteractiveToken", 1),
		"password":            strings.Replace(base, "S4U", "Password", 1),
		"additional-action":   strings.Replace(base, "</Actions>", "<ComHandler><ClassId>any</ClassId></ComHandler></Actions>", 1),
		"duplicate-action":    strings.Replace(base, "</Actions>", "<Exec><Command>evil.exe</Command></Exec></Actions>", 1),
		"duplicate-principal": strings.Replace(base, "</Principals>", "<Principal><UserId>SYSTEM</UserId><LogonType>ServiceAccount</LogonType></Principal></Principals>", 1),
		"foreign-namespace":   strings.Replace(base, "<Command>", "<Command xmlns=\"urn:other\">", 1),
		"trailing-xml":        base + "<Task/>",
		"entity-declaration":  "<!DOCTYPE Task [<!ENTITY test 'bad'>]>" + base,
	} {
		t.Run(name, func(t *testing.T) {
			if _, e := parseWindowsTaskDefinition(content); e == nil {
				t.Fatal("unsafe definition accepted")
			}
		})
	}
}

func TestWindowsTaskArgumentsPreserveTrailingSlashesAndQuotes(t *testing.T) {
	cases := []struct{ arg, want string }{
		{"", `""`},
		{`C:\plain\`, `C:\plain\`},
		{`C:\with space\`, `"C:\with space\\"`},
		{`C:\before\"quote`, `"C:\before\\\"quote"`},
	}
	for _, c := range cases {
		if got := windowsArguments([]string{c.arg}); got != c.want {
			t.Errorf("%q: got %q want %q", c.arg, got, c.want)
		}
	}
}

func TestWindowsTaskActionIdentityDoesNotAssumeEngineIsAction(t *testing.T) {
	instance := WindowsTaskInstance{GUID: "native-instance-a", EnginePID: 40, State: 4}
	if !windowsTaskActionRelation(41, 40, 101, instance, 100) {
		t.Fatal("birth-ordered native action relation refused")
	}
	if !windowsTaskActionRelation(40, 10, 100, instance, 100) {
		t.Fatal("separately verified action/engine identity refused")
	}
	for _, facts := range []struct {
		pid, parent        uint32
		birth, engineBirth uint64
	}{
		{0, 40, 101, 100}, {41, 39, 101, 100}, {41, 40, 99, 100}, {40, 10, 101, 100}, {41, 40, 0, 100}, {41, 40, 101, 0},
	} {
		if windowsTaskActionRelation(facts.pid, facts.parent, facts.birth, instance, facts.engineBirth) {
			t.Fatalf("unproven action relationship accepted: %+v", facts)
		}
	}
}
func TestWindowsTaskInstanceReplacementInvalidatesObservation(t *testing.T) {
	before := WindowsTaskSnapshot{Name: "task", XML: "protected-definition", SecurityDescriptor: "protected-dacl", State: 4, Instances: []WindowsTaskInstance{{GUID: "one", EnginePID: 40, State: 4}}}
	if !windowsTaskSameIncarnation(before, before) {
		t.Fatal("stable instance refused")
	}
	for _, change := range []func(*WindowsTaskSnapshot){
		func(s *WindowsTaskSnapshot) { s.Name = "replacement" }, func(s *WindowsTaskSnapshot) { s.XML = "other-image" }, func(s *WindowsTaskSnapshot) { s.SecurityDescriptor = "widened" }, func(s *WindowsTaskSnapshot) { s.State = 3 }, func(s *WindowsTaskSnapshot) {
			s.Instances = []WindowsTaskInstance{{GUID: "two", EnginePID: 40, State: 4}}
		}, func(s *WindowsTaskSnapshot) { s.Instances = nil }, func(s *WindowsTaskSnapshot) {
			s.Instances = append(append([]WindowsTaskInstance(nil), s.Instances...), s.Instances[0])
		},
	} {
		after := before
		change(&after)
		if windowsTaskSameIncarnation(before, after) {
			t.Fatal("changed manager proof accepted")
		}
	}
}

func TestWindowsOperatorBootPolicyRejectsContinuousOperationGaps(t *testing.T) {
	raw, e := os.ReadFile("testdata/servicedef/windows.vrooli-autoheal.xml")
	if e != nil {
		t.Fatal(e)
	}
	base := strings.Replace(string(raw), "<Count>5</Count>", "<Count>999</Count>", 1)
	for name, content := range map[string]string{
		"baseline":           base,
		"remote-app-gate":    strings.Replace(base, "</Settings>", "<DisallowStartOnRemoteAppSession>true</DisallowStartOnRemoteAppSession></Settings>", 1),
		"restart-absent":     strings.Replace(base, "<Count>999</Count>", "<Count>1</Count>", 1),
		"missed-start":       strings.Replace(base, "<StartWhenAvailable>true", "<StartWhenAvailable>false", 1),
		"network-gate":       strings.Replace(base, "</Settings>", "<RunOnlyIfNetworkAvailable>true</RunOnlyIfNetworkAvailable></Settings>", 1),
		"idle-gate":          strings.Replace(base, "</Settings>", "<RunOnlyIfIdle>true</RunOnlyIfIdle></Settings>", 1),
		"volatile":           strings.Replace(base, "</Settings>", "<Volatile>true</Volatile></Settings>", 1),
		"battery-start":      strings.Replace(base, "<DisallowStartIfOnBatteries>false", "<DisallowStartIfOnBatteries>true", 1),
		"battery-stop":       strings.Replace(base, "<StopIfGoingOnBatteries>false", "<StopIfGoingOnBatteries>true", 1),
		"demand-disabled":    strings.Replace(base, "<AllowStartOnDemand>true", "<AllowStartOnDemand>false", 1),
		"boot-delayed":       strings.Replace(base, "PT30S", "P365D", 1),
		"boot-disabled":      strings.Replace(base, "<Enabled>true", "<Enabled>false", 1),
		"extra-time-trigger": strings.Replace(base, "</Triggers>", "<TimeTrigger><Enabled>true</Enabled><StartBoundary>2026-01-01T00:00:00</StartBoundary></TimeTrigger></Triggers>", 1),
	} {
		t.Run(name, func(t *testing.T) {
			d, e := parseWindowsTaskDefinition(content)
			if e != nil {
				t.Fatal(e)
			}
			if got := windowsOperatorBootPolicy(d); got != (name == "baseline") {
				t.Fatalf("boot policy accepted=%v", got)
			}
		})
	}
}

func TestWindowsTaskRejectsForeignAttributeNamespace(t *testing.T) {
	raw, e := os.ReadFile("testdata/servicedef/windows.vrooli-autoheal.xml")
	if e != nil {
		t.Fatal(e)
	}
	base := string(raw)
	for _, content := range []string{
		strings.Replace(base, `id="Author"`, `x:id="Author" xmlns:x="urn:foreign"`, 1),
		strings.Replace(base, `Context="Author"`, `x:Context="Author" xmlns:x="urn:foreign"`, 1),
		strings.Replace(base, `version="1.4"`, `x:version="1.4" xmlns:x="urn:foreign"`, 1),
	} {
		if _, e := parseWindowsTaskDefinition(content); e == nil {
			t.Fatal("foreign attribute namespace accepted")
		}
	}
}

func TestWindowsTaskAbsenceRejectsCredentialAndManagerFailures(t *testing.T) {
	if !windowsTaskHRESULTMissing(0x80070002) {
		t.Fatal("documented missing task refused")
	}
	for _, hr := range []uint32{0x8004130f, 0x80070005, 0x80041315, 0x80041321, 0x8004131f, 0x8004130e, 0} {
		if windowsTaskHRESULTMissing(hr) {
			t.Fatalf("existing-task/manager error %#x treated as absence", hr)
		}
	}
}

func TestWindowsTaskStopRequiresNativeSOK(t *testing.T) {
	if !windowsTaskStopAcknowledged(0) {
		t.Fatal("native S_OK refused")
	}
	for _, hr := range []uint32{1, 2, 0x41300, 0x80070005, 0x8004130f, 0x80070002} {
		if windowsTaskStopAcknowledged(hr) {
			t.Fatalf("unacknowledged Stop accepted: %#x", hr)
		}
	}
}
