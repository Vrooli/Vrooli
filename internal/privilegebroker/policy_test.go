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
