package agentsessions

import (
	"testing"

	"buf.build/go/protovalidate"
)

func TestPublicProposalContractAcceptsEveryPersistableProposalKind(t *testing.T) {
	validator, err := protovalidate.New()
	if err != nil {
		t.Fatalf("protovalidate.New() error = %v", err)
	}
	for _, kind := range KnownProposalKinds() {
		proposal := Proposal{
			ID:          "prop_contract",
			Kind:        kind,
			Status:      ProposalStatusReady,
			Summary:     "Contract proposal",
			PayloadJSON: `{}`,
			CreatedAt:   testTimestamp,
			UpdatedAt:   testTimestamp,
		}
		if kind == ProposalMutationList {
			proposal.Target = &ProposalTarget{Type: ContextInitiative, Ref: "contract", Name: "Contract"}
		}
		if err := validator.Validate(proposalToProto(proposal)); err != nil {
			t.Errorf("public proposal contract rejects %q: %v", kind, err)
		}
	}
}
