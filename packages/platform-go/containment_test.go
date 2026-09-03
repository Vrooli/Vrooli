package platform

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func agentSliceDefinition() ServiceDefinition {
	return ServiceDefinition{
		Name:        "vrooli-agents",
		Description: "Vrooli coding-agent sessions",
		Kind:        KindSlice,
		Scope:       ScopeUser,
		Protections: Protections{Containment: Containment{CPUWeight: 50, MemoryHigh: "50%", MemoryMax: "60%", TasksMax: 4096}},
	}
}

func TestContainmentRendersCeilingsInSystemd(t *testing.T) {
	d := coreDefinitions(t, "linux", "/home/alice")[0]
	d.Protections.Containment = Containment{CPUWeight: 400, MemoryHigh: "1G", MemoryMax: "2G", TasksMax: 512, Slice: "vrooli-agents.slice"}
	artifact, err := RenderSystemd(d)
	if err != nil {
		t.Fatal(err)
	}
	content := artifact.Primary().Content
	for _, want := range []string{"CPUWeight=400\n", "Slice=vrooli-agents.slice\n", "MemoryHigh=1G\n", "MemoryMax=2G\n", "TasksMax=512\n", "MemoryMin=128M\n"} {
		if !strings.Contains(content, want) {
			t.Errorf("service lacks %q:\n%s", want, content)
		}
	}
}

func TestSliceRendersUnderSystemd(t *testing.T) {
	artifact, err := RenderSystemd(agentSliceDefinition())
	if err != nil {
		t.Fatal(err)
	}
	file := artifact.Primary()
	if file.Name != "vrooli-agents.slice" {
		t.Fatalf("slice file name = %q", file.Name)
	}
	for _, want := range []string{"[Unit]\nDescription=Vrooli coding-agent sessions\n", "[Slice]\n", "CPUWeight=50\n", "MemoryHigh=50%\n", "MemoryMax=60%\n", "TasksMax=4096\n", "ManagedOOMMemoryPressure=kill\n"} {
		if !strings.Contains(file.Content, want) {
			t.Errorf("slice lacks %q:\n%s", want, file.Content)
		}
	}
	if strings.Contains(file.Content, "ExecStart") || strings.Contains(file.Content, "[Service]") {
		t.Fatalf("slice carries service directives:\n%s", file.Content)
	}
	if _, err := RenderLaunchd(agentSliceDefinition()); err == nil {
		t.Fatal("launchd rendered a slice")
	}
	if _, err := RenderWindowsTaskXML(agentSliceDefinition()); err == nil {
		t.Fatal("windows task rendered a slice")
	}
}

func TestLaunchdRendersNiceAndResourceLimits(t *testing.T) {
	d := coreDefinitions(t, "darwin", "/Users/alice")[0]
	d.Protections.Containment = Containment{CPUWeight: 50, MemoryMax: "2G", TasksMax: 64}
	artifact, err := RenderLaunchd(d)
	if err != nil {
		t.Fatal(err)
	}
	content := artifact.Primary().Content
	for _, want := range []string{"<key>Nice</key>\n  <integer>10</integer>", "<key>SoftResourceLimits</key>", "<key>NumberOfProcesses</key>\n    <integer>64</integer>", "<key>ResidentSetSize</key>\n    <integer>2147483648</integer>"} {
		if !strings.Contains(content, want) {
			t.Errorf("plist lacks %q:\n%s", want, content)
		}
	}
	d.Protections.Containment = Containment{CPUWeight: 400, MemoryMax: "60%"}
	artifact, _ = RenderLaunchd(d)
	if !strings.Contains(artifact.Primary().Content, "<key>Nice</key>\n  <integer>-5</integer>") || strings.Contains(artifact.Primary().Content, "ResidentSetSize") {
		t.Fatalf("weight 400 / percent ceiling rendered wrong:\n%s", artifact.Primary().Content)
	}
}

func TestWindowsTaskRendersPriority(t *testing.T) {
	d := coreDefinitions(t, "windows", `C:\Users\alice`)[0]
	for weight, want := range map[int]string{400: "<Priority>4</Priority>", 100: "<Priority>7</Priority>", 50: "<Priority>9</Priority>"} {
		d.Protections.Containment = Containment{CPUWeight: weight}
		artifact, err := RenderWindowsTaskXML(d)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(artifact.Primary().Content, want) {
			t.Errorf("weight %d: task lacks %s:\n%s", weight, want, artifact.Primary().Content)
		}
	}
}

func TestValidateRejectsInvertedMemoryCeilingsAndTinyTasksMax(t *testing.T) {
	d := agentSliceDefinition()
	d.Protections.Containment.MemoryHigh = "70%"
	if err := d.Validate(); err == nil || !strings.Contains(err.Error(), "MemoryHigh") {
		t.Fatalf("inverted ceilings accepted: %v", err)
	}
	d = agentSliceDefinition()
	d.Protections.Containment.TasksMax = 8
	if err := d.Validate(); err == nil || !strings.Contains(err.Error(), "TasksMax") {
		t.Fatalf("tiny TasksMax accepted: %v", err)
	}
	if err := agentSliceDefinition().Validate(); err != nil {
		t.Fatalf("valid slice rejected: %v", err)
	}
}

func TestScopeRefRoundTrips(t *testing.T) {
	for _, ref := range []ScopeRef{
		{Name: "vrooli-agent-1", Kind: ScopeKindCgroup, Path: "/user.slice/user-1000.slice/user@1000.service/vrooli.slice/vrooli-agents.slice/vrooli-agent-1.scope"},
		{Name: "vrooli-agent-2", Kind: ScopeKindProcessGroup, PID: 4242},
		{Name: "vrooli-agent-3", Kind: ScopeKindJob, PID: 77},
		{Kind: ScopeKindNone},
	} {
		parsed, err := ParseScopeRef(ref.String())
		if err != nil {
			t.Fatalf("%s: %v", ref.String(), err)
		}
		if parsed.Kind != ref.Kind || parsed.Path != ref.Path || parsed.PID != ref.PID {
			t.Fatalf("round trip %q → %+v", ref.String(), parsed)
		}
	}
	if _, err := ParseScopeRef("bogus:1"); err == nil {
		t.Fatal("bogus kind parsed")
	}
}

func TestWeightMappingIsMonotone(t *testing.T) {
	previousNice, previousPriority := -100, -1
	for _, weight := range []int{1, 50, 100, 200, 400, 10000} {
		nice := niceForWeight(weight)
		priority := windowsPriorityForWeight(weight)
		if nice > previousNice && previousNice != -100 {
			t.Fatalf("nice not monotone at weight %d: %d after %d", weight, nice, previousNice)
		}
		if priority > previousPriority && previousPriority != -1 {
			t.Fatalf("priority not monotone at weight %d: %d after %d", weight, priority, previousPriority)
		}
		previousNice, previousPriority = nice, priority
	}
	if niceForWeight(50) != 10 || niceForWeight(100) != 0 || niceForWeight(400) != -5 {
		t.Fatalf("documented nice anchors drifted: %d %d %d", niceForWeight(50), niceForWeight(100), niceForWeight(400))
	}
}

// The slice fixture pins the rendered shape of vrooli-agents.slice the way
// the darwin and windows fixtures pin theirs; systemd-analyze accepts it in
// TestSystemdRenderPassesSystemdAnalyze on any host with systemd.
func TestAgentSliceMatchesFixture(t *testing.T) {
	artifact, err := RenderSystemd(agentSliceDefinition())
	if err != nil {
		t.Fatal(err)
	}
	fixture := filepath.Join("testdata", "servicedef", "linux.vrooli-agents.slice")
	header := "# Fixture derived 2026-09-02 on swarminator (Linux, systemd 255) from platform-go RenderSystemdSlice; host-verified by systemd-analyze --user verify. Regenerate: PLATFORM_GO_UPDATE_FIXTURES=1 go test -run TestAgentSliceMatchesFixture\n"
	if os.Getenv("PLATFORM_GO_UPDATE_FIXTURES") == "1" {
		if err := os.WriteFile(fixture, []byte(header+artifact.Primary().Content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	raw, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatalf("%s: %v (set PLATFORM_GO_UPDATE_FIXTURES=1 to regenerate)", fixture, err)
	}
	if want := strings.TrimPrefix(string(raw), header); want != artifact.Primary().Content {
		t.Fatalf("slice drifted from its fixture; rendered:\n%s\nfixture:\n%s", artifact.Primary().Content, want)
	}
}
