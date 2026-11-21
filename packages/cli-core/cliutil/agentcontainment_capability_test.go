package cliutil

import (
	"context"
	"errors"
	"testing"
)

// capabilityContainer is a container that declares whether the host has a
// ceiling primitive. Run always refuses, so each test isolates the one thing
// under test: how the launcher classifies an uncontained session.
type capabilityContainer struct{ supported bool }

func (c capabilityContainer) Run(context.Context, string, SessionContainment, SessionProcess) (ContainedSession, error) {
	return ContainedSession{Method: ContainmentMethodNone}, &UncontainedError{Err: errors.New("no ceiling here")}
}

func (c capabilityContainer) ContainSelf(string, SessionContainment) (ContainedSession, error) {
	return ContainedSession{Method: ContainmentMethodNone}, errors.New("no ceiling here")
}

func (c capabilityContainer) SupportsContainment() bool { return c.supported }

// TestContainmentSupportedSeparatesAbsentPrimitiveFromFailure is the whole
// point of the capability seam. Both cases end with Method "none", so a
// reader that cannot tell them apart must alarm on a platform that can never
// comply or stay silent when a platform that should have contained a session
// did not.
func TestContainmentSupportedSeparatesAbsentPrimitiveFromFailure(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		container SessionContainer
		want      bool
	}{
		{"platform has a primitive", capabilityContainer{supported: true}, true},
		{"platform has no primitive", capabilityContainer{supported: false}, false},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if got := containmentSupported(testCase.container); got != testCase.want {
				t.Fatalf("containmentSupported() = %v, want %v", got, testCase.want)
			}
		})
	}
}

// A container predating the capability interface must keep the behaviour it
// has today: assumed capable, so an undeclared capability can never quietly
// downgrade a real containment regression into an expected limit.
func TestContainmentSupportedAssumesCapableWithoutDeclaration(t *testing.T) {
	if !containmentSupported(&fakeContainer{}) {
		t.Fatal("a container that does not implement ContainmentCapability must be assumed capable")
	}
}

// The classification has to reach the report, because the report is what the
// editor lease row stores. A log line alone is not evidence a later reader
// can query.
func TestContainSelfRecordsUnsupportedInTheReport(t *testing.T) {
	for _, testCase := range []struct {
		name            string
		supported       bool
		wantUnsupported bool
	}{
		{"supported host that failed is a regression", true, false},
		{"unsupported host is an expected limit", false, true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			previous, previousFn := DefaultSessionContainer, agentContainmentFn
			DefaultSessionContainer = capabilityContainer{supported: testCase.supported}
			agentContainmentFn = testContainment
			t.Cleanup(func() { DefaultSessionContainer, agentContainmentFn = previous, previousFn })

			var report containmentReport
			containSelf(AgentLaunchRequest{Agent: "codex"}, "vrooli-agent-test", SessionContainment{}, ContainmentSourceDefaults, &report)

			if report.Failure == "" {
				t.Fatal("a refused ceiling must still record why it was refused")
			}
			if report.Unsupported != testCase.wantUnsupported {
				t.Fatalf("report.Unsupported = %v, want %v", report.Unsupported, testCase.wantUnsupported)
			}
		})
	}
}
