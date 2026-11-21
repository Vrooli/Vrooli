package searchcontrol

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	aisearchpkg "github.com/vrooli/ai-go/search"
	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/control"
)

type migrationContractFake struct{ calls int }

func (f *migrationContractFake) Reindex(context.Context, string, bool) (string, int, int, error) {
	f.calls++
	return "job", 0, 0, nil
}
func (*migrationContractFake) ReindexStatus(string) (string, int, int, string, bool) {
	return "succeeded", 0, 0, "", true
}
func (*migrationContractFake) ReindexCancel(string) bool { return false }
func (*migrationContractFake) ApplyTuning(context.Context, aisearchpkg.TuningConfig) (string, int, int, error) {
	return "", 0, 0, nil
}

func TestReindexRejectsEmbeddingActionsBeforeLiveReconcile(t *testing.T) {
	fake := &migrationContractFake{}
	handler := NewConnectHandler(Deps{Reindexer: fake, Gate: &Gate{Token: func() string { return "secret" }}})
	for _, request := range []*controlv1.ReindexRequest{
		{ControlToken: "secret", Action: "shadow", ShadowCollection: "candidate", EmbeddingModel: "m", EmbeddingRole: "r", EmbeddingDimensions: 3, EmbeddingPolicySchemaVersion: "v1"},
		{ControlToken: "secret", EmbeddingModel: "m"},
		{ControlToken: "secret", Action: "abort"},
	} {
		if _, err := handler.Reindex(context.Background(), connect.NewRequest(request)); connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Fatalf("request %+v: code=%s err=%v, want invalid_argument", request, connect.CodeOf(err), err)
		}
	}
	if fake.calls != 0 {
		t.Fatalf("unsupported migration requests reached live reindex %d times", fake.calls)
	}
}
