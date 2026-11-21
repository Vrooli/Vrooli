package stagedmessages

import (
	"context"
	"fmt"
	"strings"
	"testing"

	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/advisory"
)

func TestSearchDraftContextsMatchesWorkflowInputBound(t *testing.T) {
	s, _, _, provider := serviceFixture(t)
	for index := 0; index < MaxSelectedContexts+3; index++ {
		provider.contexts = append(provider.contexts, &v1.StagedMessageContext{Id: fmt.Sprintf("source-%02d", index)})
	}

	contexts, omissions := s.searchDraftContexts(context.Background(), &v1.StagedMessageSnapshot{Digest: "sha256:test"})
	if len(contexts) != MaxSelectedContexts {
		t.Fatalf("context count = %d, want workflow limit %d", len(contexts), MaxSelectedContexts)
	}
	if len(omissions) != 1 || !strings.Contains(omissions[0].GetReason(), fmt.Sprintf("%d-context workflow input bound", MaxSelectedContexts)) {
		t.Fatalf("context truncation was not disclosed: %+v", omissions)
	}
}
