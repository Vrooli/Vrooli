package workloadowner

import "testing"

func TestPostureChangesReportingNotClassification(t *testing.T) {
	observed := []Workload{{Kind: "container", Name: "airbyte-abctl-control-plane", CommandLine: "/sbin/init", Running: true}, {Kind: "container", Name: "operator-container", CommandLine: "private command", Running: true}, {Kind: "container", Name: "personal-container", CommandLine: "secret command", Running: true}}
	decl := []Declaration{
		{Kind: "container", Name: "airbyte-abctl-control-plane", Live: false, Evidence: []string{"historical manifest: agent experiment"}},
		{Kind: "container", Name: "operator-container", Live: true},
	}
	whole := Classify(observed, decl, WholeHost, 10)
	shared := Classify(observed, decl, VrooliOnly, 10)
	if len(whole.Findings) != 2 || len(shared.Findings) != 1 || whole.Findings[0].Class != Abandoned || shared.Findings[0].Class != Abandoned {
		t.Fatalf("posture reports diverged incorrectly: whole=%+v shared=%+v", whole, shared)
	}
	if len(shared.Informational) != 1 || shared.Informational[0].CommandLine != "" {
		t.Fatalf("shared posture leaked unmanaged command line: %+v", shared.Informational)
	}
}

func TestCrashLoopIsADeclaredFinding(t *testing.T) {
	r := Classify([]Workload{{Kind: "unit", Name: "kubelet", WindowHours: 72, RestartCount: 191985}}, []Declaration{{Kind: "unit", Name: "kubelet", Live: true}}, WholeHost, 10)
	if len(r.Findings) != 1 || !r.Findings[0].CrashLoop {
		t.Fatalf("expected crash-loop finding: %+v", r)
	}
}

func TestHistoricalNamingRuleAndDeclaredResource(t *testing.T) {
	observed := []Workload{
		{Kind: "container", Name: "airbyte-abctl-control-plane", Image: "kindest/node:v1.32.2", Running: true, RestartCount: 191985, WindowHours: 72},
		{Kind: "container", Name: "postgis-main", Image: "vrooli/postgis-routing:16-3.4", Running: true},
		{Kind: "container", Name: "operator-container", Image: "postgres:16", CommandLine: "private", Running: true},
	}
	r := Classify(observed, []Declaration{{Kind: "container", Name: "postgis-main", Live: true, Evidence: []string{"enabled resource manifest: resources/postgis/resource.json"}}}, VrooliOnly, 10)
	if len(r.Findings) != 1 || r.Findings[0].Class != Abandoned || r.Findings[0].Name != "airbyte-abctl-control-plane" {
		t.Fatalf("expected Airbyte abandoned finding: %#v", r)
	}
	if !r.Findings[0].CrashLoop || r.Findings[0].ProposedAction == "" {
		t.Fatalf("expected crash-loop and disposal proposal: %#v", r.Findings[0])
	}
	if len(r.Informational) != 1 || r.Informational[0].Class != Unmanaged || r.Informational[0].Name != "operator-container" {
		t.Fatalf("expected operator workload informational row: %#v", r)
	}
	if r.Informational[0].CommandLine != "" {
		t.Fatalf("unmanaged command leaked: %#v", r.Informational[0])
	}
}

func TestUnmanagedRowsCarryEvidenceAndPostureOnlyChangesReporting(t *testing.T) {
	observed := []Workload{{Kind: "container", Name: "operator-owned", CommandLine: "private command"}}
	whole := Classify(observed, nil, WholeHost, 10)
	shared := Classify(observed, nil, VrooliOnly, 10)
	if len(whole.Findings) != 1 || whole.Findings[0].Class != Unmanaged || len(whole.Findings[0].Evidence) == 0 {
		t.Fatalf("whole-host unmanaged evidence missing: %#v", whole)
	}
	if len(shared.Informational) != 1 || shared.Informational[0].CommandLine != "" || len(shared.Informational[0].Evidence) == 0 {
		t.Fatalf("vrooli-only unmanaged reporting invalid: %#v", shared)
	}
	if whole.Findings[0].Class != shared.Informational[0].Class || whole.Findings[0].Reason != shared.Informational[0].Reason || whole.Findings[0].Evidence[0] != shared.Informational[0].Evidence[0] {
		t.Fatalf("posture changed classification evidence: whole=%#v shared=%#v", whole, shared)
	}
}

func TestParseDockerInspectRestartCounts(t *testing.T) {
	counts := ParseDockerInspectRestartCounts([]byte("/airbyte-abctl-control-plane\t191985\n/postgis-main\t0\n"))
	if counts["airbyte-abctl-control-plane"] != 191985 || counts["postgis-main"] != 0 {
		t.Fatalf("unexpected restart counts: %#v", counts)
	}
}

func TestParseDockerInspectJSONRestartCounts(t *testing.T) {
	counts := ParseDockerInspectJSON([]byte(`[{"Name":"/airbyte-abctl-control-plane","State":{"RestartCount":191985}},{"Name":"/postgis-main","RestartCount":0}]`))
	if counts["airbyte-abctl-control-plane"] != 191985 || counts["postgis-main"] != 0 {
		t.Fatalf("unexpected JSON restart counts: %#v", counts)
	}
}
