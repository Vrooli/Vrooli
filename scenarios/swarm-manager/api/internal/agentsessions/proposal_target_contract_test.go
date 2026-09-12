package agentsessions

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// The client validates every response against the proto's buf.validate rules,
// and protovalidate rejects the whole message on one violation. So a proposal
// target type the server happily persists but the proto does not list does not
// degrade one row — it fails the entire agent-sessions list response with
// "Invalid agent sessions response", blanking the Sessions view.
//
// That is exactly what happened: nine live sessions targeted captures, which
// ProposalTarget.Validate accepts and sessionDecisionProvider counts, while the
// proto allowed only backlog_item and goal.

const proposalTargetProtoPath = "../../../../../packages/proto/schemas/swarm-manager/v1/domain/agent_session.proto"

var proposalTargetBlockPattern = regexp.MustCompile(
	`(?s)message AgentSessionProposalTarget \{.*?string type = 1 \[\(buf\.validate\.field\)\.string = \{(.*?)\}\];`,
)

var quotedValuePattern = regexp.MustCompile(`"([a-z_]+)"`)

func protoProposalTargetTypes(t *testing.T) []string {
	t.Helper()
	source, err := os.ReadFile(filepath.Clean(proposalTargetProtoPath))
	if err != nil {
		t.Fatalf("read proposal target proto: %v", err)
	}
	block := proposalTargetBlockPattern.FindSubmatch(source)
	if block == nil {
		t.Fatal("AgentSessionProposalTarget.type constraint not found; update the pattern if the proto was restructured")
	}
	matches := quotedValuePattern.FindAllStringSubmatch(string(block[1]), -1)
	values := make([]string, 0, len(matches))
	for _, match := range matches {
		values = append(values, match[1])
	}
	if len(values) == 0 {
		t.Fatal("AgentSessionProposalTarget.type declares no allowed values")
	}
	sort.Strings(values)
	return values
}

func TestProposalTargetTypesMatchProtoContract(t *testing.T) {
	server := make([]string, 0, len(ProposalTargetTypes))
	for _, targetType := range ProposalTargetTypes {
		server = append(server, string(targetType))
	}
	sort.Strings(server)

	proto := protoProposalTargetTypes(t)
	if strings.Join(server, ",") != strings.Join(proto, ",") {
		t.Fatalf("proposal target types disagree across the wire contract:\n  server accepts: %v\n  proto allows:   %v\n"+
			"A type the server accepts but the proto omits fails the whole list response for every client.",
			server, proto)
	}
}

func TestProposalTargetValidateAcceptsEveryDeclaredType(t *testing.T) {
	for _, targetType := range ProposalTargetTypes {
		target := ProposalTarget{Type: targetType, Ref: "ref", Name: "Name"}
		if err := target.Validate(); err != nil {
			t.Errorf("Validate(%q) error = %v, want nil", targetType, err)
		}
	}
}

func TestProposalTargetValidateRejectsUndeclaredTypes(t *testing.T) {
	// ContextExecution is a legitimate *context* type but never a proposal
	// target; the two sets are deliberately different sizes.
	target := ProposalTarget{Type: ContextExecution, Ref: "ref", Name: "Name"}
	if err := target.Validate(); err == nil {
		t.Fatal("Validate(execution) error = nil, want a validation error")
	}
}
