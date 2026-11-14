package searchcontrol

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	aisearch "github.com/vrooli/ai-go/search"
	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/control"
)

type migrationFake struct {
	calls      int
	shadowArgs []any
}

func (f *migrationFake) Reindex(context.Context, string, bool) (string, int, int, error) {
	f.calls++
	return "legacy-job", 1, 0, nil
}
func (*migrationFake) ReindexStatus(string) (string, int, int, string, bool) {
	return "succeeded", 1, 1, "", true
}
func (*migrationFake) ReindexCancel(string) bool { return false }
func (*migrationFake) ApplyTuning(context.Context, aisearch.TuningConfig) (string, int, int, error) {
	return "", 0, 0, nil
}
func (f *migrationFake) ReindexShadow(_ context.Context, collection, model, role string, dimensions int, policySchemaVersion, scope string, dryRun bool) (string, int, int, error) {
	f.calls++
	f.shadowArgs = []any{collection, model, role, dimensions, policySchemaVersion, scope, dryRun}
	return "shadow-job", 1, 0, nil
}
func (f *migrationFake) ActivateShadow(context.Context, string) error { f.calls++; return nil }
func (f *migrationFake) RollbackShadow(context.Context, string) error { f.calls++; return nil }

func TestReindexRejectsUnsupportedAndIncompleteMigrationBeforeProviderCalls(t *testing.T) {
	fake := &migrationFake{}
	handler := NewConnectHandler(Deps{Reindexer: fake, Gate: &Gate{Token: func() string { return "secret" }}})
	for _, request := range []*controlv1.ReindexRequest{
		{ControlToken: "secret", Action: "abort"},
		{ControlToken: "secret", Action: "shadow", ShadowCollection: "candidate"},
		{ControlToken: "secret", Action: "shadow", ShadowCollection: "../live", EmbeddingModel: "m", EmbeddingRole: "r", EmbeddingDimensions: 3, EmbeddingPolicySchemaVersion: "v1"},
		{ControlToken: "secret", Action: "rollback"},
	} {
		if _, err := handler.Reindex(context.Background(), connect.NewRequest(request)); connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Fatalf("request %+v: code=%s err=%v, want invalid_argument", request, connect.CodeOf(err), err)
		}
	}
	if fake.calls != 0 {
		t.Fatalf("invalid migration requests reached provider %d times", fake.calls)
	}
}

func TestReindexPreservesLegacyAndCompletedShadowRequest(t *testing.T) {
	fake := &migrationFake{}
	handler := NewConnectHandler(Deps{Reindexer: fake, Gate: &Gate{Token: func() string { return "secret" }}})
	legacy, err := handler.Reindex(context.Background(), connect.NewRequest(&controlv1.ReindexRequest{ControlToken: "secret"}))
	if err != nil || legacy.Msg.GetJobId() != "legacy-job" {
		t.Fatalf("legacy reindex response=%+v err=%v", legacy, err)
	}
	shadow, err := handler.Reindex(context.Background(), connect.NewRequest(&controlv1.ReindexRequest{
		ControlToken: "secret", Action: "shadow", ShadowCollection: "candidate-a", EmbeddingModel: "model@sha256:abc",
		EmbeddingRole: "embedding.default", EmbeddingDimensions: 768, EmbeddingPolicySchemaVersion: "v1",
	}))
	if err != nil || shadow.Msg.GetJobId() != "shadow-job" {
		t.Fatalf("shadow response=%+v err=%v", shadow, err)
	}
	if fake.calls != 2 {
		t.Fatalf("provider calls=%d, want legacy reindex plus shadow build", fake.calls)
	}
	want := []any{"candidate-a", "model@sha256:abc", "embedding.default", 768, "v1", "", false}
	if len(fake.shadowArgs) != len(want) {
		t.Fatalf("shadow adapter args=%v, want %v", fake.shadowArgs, want)
	}
	for index := range want {
		if fake.shadowArgs[index] != want[index] {
			t.Fatalf("shadow adapter argument %d=%#v, want %#v; request fields may have been dropped", index, fake.shadowArgs[index], want[index])
		}
	}
}

func TestReindexNegotiatesCapabilitiesAndNormalizesPromote(t *testing.T) {
	fake := &migrationFake{}
	handler := NewConnectHandler(Deps{Reindexer: fake, Gate: &Gate{Token: func() string { return "secret" }}})
	capabilities, err := handler.Reindex(context.Background(), connect.NewRequest(&controlv1.ReindexRequest{ControlToken: "secret", Action: "capabilities"}))
	if err != nil {
		t.Fatalf("capabilities negotiation: %v", err)
	}
	if capabilities.Msg.GetMigrationProtocolVersion() != aisearch.ReindexMigrationProtocolVersion {
		t.Fatalf("protocol version=%d", capabilities.Msg.GetMigrationProtocolVersion())
	}
	if got := capabilities.Msg.GetSupportedMigrationActions(); len(got) != 3 || got[0] != "shadow" || got[1] != "cutover" || got[2] != "rollback" {
		t.Fatalf("supported actions=%v", got)
	}
	if fake.calls != 0 {
		t.Fatalf("capability negotiation mutated provider %d times", fake.calls)
	}
	promote, err := handler.Reindex(context.Background(), connect.NewRequest(&controlv1.ReindexRequest{ControlToken: "secret", Action: "promote", ShadowCollection: "candidate"}))
	if err != nil || promote == nil {
		t.Fatalf("promote alias did not dispatch as cutover: response=%+v err=%v", promote, err)
	}
	if fake.calls != 1 {
		t.Fatalf("promote alias provider calls=%d, want one canonical cutover", fake.calls)
	}
}
