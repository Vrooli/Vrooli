package privilegebroker

import (
	"reflect"
	"testing"
)

func validRequest(action string) Request {
	return Request{Version: ProtocolVersion, RequestID: "request-1", Action: action, Subject: Subject{Scenario: BridgeScenario, CandidateIP: "192.168.1.176", Port: BridgePort}}
}

func TestUFWPolicyOnlyBuildsImmutableAdmissionArgv(t *testing.T) {
	got, err := UFWArgs(validRequest(ActionBridgeUFWAllow))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"allow", "from", "192.168.1.176", "to", "any", "port", "18767", "proto", "tcp", "comment", RuleComment}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("args=%q want=%q", got, want)
	}
}

func TestUFWPolicyRejectsArbitraryCommandShapes(t *testing.T) {
	for name, mutate := range map[string]func(*Request){
		"unknown action": func(r *Request) { r.Action = "shell" },
		"wrong scenario": func(r *Request) { r.Subject.Scenario = "anything" },
		"wrong port":     func(r *Request) { r.Subject.Port = 22 },
		"cidr":           func(r *Request) { r.Subject.CandidateIP = "192.168.1.0/24" },
		"loopback":       func(r *Request) { r.Subject.CandidateIP = "127.0.0.1" },
		"missing id":     func(r *Request) { r.RequestID = "" },
	} {
		t.Run(name, func(t *testing.T) {
			req := validRequest(ActionBridgeUFWAllow)
			mutate(&req)
			if _, err := UFWArgs(req); err == nil {
				t.Fatal("expected policy rejection")
			}
		})
	}
}

func TestRuntimeHomePolicyAcceptsOnlyApprovedClassAndCallerIdentity(t *testing.T) {
	req := Request{Version: ProtocolVersion, RequestID: "runtime-1", Action: ActionRuntimeHomeOwnershipRepair, RuntimeHome: &RuntimeHomeSubject{Class: "cache", ExpectedUID: 1000, ExpectedGID: 1000}}
	if err := Validate(req); err != nil {
		t.Fatalf("Validate(runtime home): %v", err)
	}
	req.RuntimeHome.Class = "secrets_enc"
	if err := Validate(req); err != nil {
		t.Fatalf("Validate(secrets_enc): %v", err)
	}
	for _, class := range []string{"/tmp", "../cache", "secrets"} {
		req.RuntimeHome.Class = class
		if err := Validate(req); err == nil {
			t.Fatalf("Validate accepted unapproved class %q", class)
		}
	}
}

func TestStorageActionPolicyRejectsUnboundedSubjects(t *testing.T) {
	tests := []struct {
		name string
		req  Request
	}{
		{"unknown log stanza", Request{Version: ProtocolVersion, RequestID: "1", Action: ActionLogRotateForce, Log: &LogSubject{Stanza: "all"}}},
		{"negative journal size", Request{Version: ProtocolVersion, RequestID: "2", Action: ActionJournaldVacuum, Journal: &JournalSubject{MaxUseBytes: -1}}},
		{"wildcard volume", Request{Version: ProtocolVersion, RequestID: "3", Action: ActionDockerPruneUnusedVolumes, Docker: &DockerSubject{VolumeNames: []string{"*"}}}},
		{"empty volume list", Request{Version: ProtocolVersion, RequestID: "4", Action: ActionDockerPruneUnusedVolumes, Docker: &DockerSubject{}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := Validate(test.req); err == nil {
				t.Fatal("Validate accepted an unsafe storage action")
			}
		})
	}
}

func TestStorageActionPolicyAcceptsOnlyBoundedSubjects(t *testing.T) {
	for _, req := range []Request{
		{Version: ProtocolVersion, RequestID: "log", Action: ActionLogRotateForce, Log: &LogSubject{Stanza: managedLogStanza}},
		{Version: ProtocolVersion, RequestID: "journal", Action: ActionJournaldVacuum, Journal: &JournalSubject{MaxUseBytes: 1024}},
		{Version: ProtocolVersion, RequestID: "image", Action: ActionDockerPruneUnusedImages, Docker: &DockerSubject{}},
		{Version: ProtocolVersion, RequestID: "volume", Action: ActionDockerPruneUnusedVolumes, Docker: &DockerSubject{VolumeNames: []string{"unused-volume"}}},
	} {
		if err := Validate(req); err != nil {
			t.Fatalf("Validate(%s) = %v", req.Action, err)
		}
	}
}
