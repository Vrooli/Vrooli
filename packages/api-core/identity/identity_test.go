package identity

import (
	"errors"
	"strings"
	"testing"
)

func TestPrincipalStatesAreExplicit(t *testing.T) {
	cases := []struct {
		name   string
		kind   ActorKind
		verify bool
		human  bool
	}{
		{name: "unknown", kind: ActorUnknown},
		{name: "human", kind: ActorHuman, verify: true, human: true},
		{name: "agent", kind: ActorAgent, verify: true},
		{name: "service", kind: ActorService, verify: true},
		{name: "conflict", kind: ActorConflict, verify: true},
		{name: "missing subject", kind: ActorHuman, verify: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			principal := Principal{Kind: tc.kind, Subject: "subject", Verified: tc.verify}
			if tc.name == "missing subject" {
				principal.Subject = ""
			}
			expectedVerified := tc.verify && tc.name != "missing subject" && tc.kind != ActorConflict
			if got := principal.IsVerified(); got != expectedVerified {
				t.Fatalf("IsVerified()=%t", got)
			}
			if got := principal.IsHuman(); got != tc.human {
				t.Fatalf("IsHuman()=%t", got)
			}
		})
	}
}

func TestFailureFromErrorHandlesWrappingWithoutCredentialDetails(t *testing.T) {
	failure := NewFailure(FailureInvalid, SourceCloudflareAccess)
	wrapped := errors.New("outer: " + failure.Error())
	if got, ok := FailureFromError(wrapped); ok || got.Class != "" {
		t.Fatalf("plain error unexpectedly classified: %#v, %t", got, ok)
	}
	wrappedFailure := fmtWrap{err: failure}
	got, ok := FailureFromError(wrappedFailure)
	if !ok || got.Class != FailureInvalid || got.Source != SourceCloudflareAccess {
		t.Fatalf("failure=%#v ok=%t", got, ok)
	}
	if strings.Contains(got.Error(), "token") {
		t.Fatal("failure classification contains credential terminology")
	}
}

type fmtWrap struct{ err error }

func (w fmtWrap) Error() string { return "wrapped failure" }
func (w fmtWrap) Unwrap() error { return w.err }
