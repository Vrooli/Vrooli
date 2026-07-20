package gates

import (
	"context"
	"io"
	"testing"

	"swarm-manager/internal/agentsessions"
)

type proposalStore struct{ sessions []agentsessions.Session }

func (s proposalStore) CreateSession(agentsessions.Session) error { return nil }
func (s proposalStore) SaveSession(agentsessions.Session) error   { return nil }
func (s proposalStore) DeleteSession(string) error                { return nil }
func (s proposalStore) LoadSession(string) (agentsessions.Session, error) {
	return agentsessions.Session{}, agentsessions.ErrNotFound
}

func (s proposalStore) ListSessions(agentsessions.ListFilters) ([]agentsessions.Session, error) {
	return s.sessions, nil
}
func (s proposalStore) AppendMessage(string, agentsessions.Message) error      { return nil }
func (s proposalStore) SaveProposal(string, agentsessions.Proposal) error      { return nil }
func (s proposalStore) AppendArtifacts(string, []agentsessions.Artifact) error { return nil }
func (s proposalStore) ListArtifacts(string) ([]agentsessions.Artifact, error) { return nil, nil }
func (s proposalStore) ListArtifactsByEntity(agentsessions.ArtifactType, string) ([]agentsessions.Artifact, error) {
	return nil, nil
}
func (s proposalStore) SaveAttachment(string, agentsessions.Attachment, io.Reader) error { return nil }
func (s proposalStore) AttachmentPath(string, string) (string, agentsessions.Attachment, error) {
	return "", agentsessions.Attachment{}, agentsessions.ErrNotFound
}

func TestProposalSourceAggregatesReadyBacklogTargets(t *testing.T) {
	target := &agentsessions.ProposalTarget{Type: agentsessions.ContextBacklogItem, Ref: "execute/refresh", Name: "Refresh"}
	source := ProposalSource{Store: proposalStore{sessions: []agentsessions.Session{{Proposals: []agentsessions.Proposal{{Kind: agentsessions.ProposalMutationList, Status: agentsessions.ProposalStatusReady, Target: target}, {Kind: agentsessions.ProposalMutationList, Status: agentsessions.ProposalStatusReady, Target: target}}}}}}
	gates, err := source.Enumerate(context.Background())
	if err != nil || len(gates) != 1 {
		t.Fatalf("Enumerate() = %#v, %v", gates, err)
	}
	if gates[0].Kind != KindProposal || gates[0].Count != 2 || gates[0].OwnerName != "refresh" {
		t.Fatalf("unexpected proposal gate: %#v", gates[0])
	}
}

func TestProposalSourceAggregatesReadyInitiativeTargets(t *testing.T) {
	target := &agentsessions.ProposalTarget{Type: agentsessions.ContextInitiative, Ref: "refresh-program", Name: "Refresh program"}
	source := ProposalSource{Store: proposalStore{sessions: []agentsessions.Session{{Proposals: []agentsessions.Proposal{{Kind: agentsessions.ProposalMutationList, Status: agentsessions.ProposalStatusReady, Target: target}}}}}}
	got, err := source.Enumerate(context.Background())
	if err != nil || len(got) != 1 {
		t.Fatalf("Enumerate() = %#v, %v", got, err)
	}
	if got[0].OwnerType != "initiative" || got[0].OwnerName != "refresh-program" || got[0].Count != 1 {
		t.Fatalf("unexpected initiative proposal gate: %#v", got[0])
	}
}
