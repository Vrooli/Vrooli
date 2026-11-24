package relationships

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	internal "content-desk/internal/relationships"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/relationships"
)

type fakeRepository struct {
	relation internal.Relation
	impact   internal.Impact
	err      error
}

func (f *fakeRepository) CreateRelation(_ context.Context, rel internal.Relation) (internal.Relation, error) {
	f.relation = rel
	if f.err != nil {
		return internal.Relation{}, f.err
	}
	rel.ID = "relation-1"
	return rel, nil
}
func (*fakeRepository) ListRelations(context.Context, string) ([]internal.Relation, error) {
	return nil, nil
}
func (*fakeRepository) CreateCollection(context.Context, string, string) (internal.Collection, error) {
	return internal.Collection{}, nil
}
func (*fakeRepository) ListCollections(context.Context, string) ([]internal.Collection, error) {
	return []internal.Collection{{ID: "c1", ProjectID: "p1", Name: "Reading list"}}, nil
}
func (*fakeRepository) AddCollectionItem(context.Context, internal.CollectionItem) error { return nil }
func (*fakeRepository) RemoveCollectionItem(context.Context, internal.CollectionItem) error { return nil }
func (*fakeRepository) ListCollectionItems(context.Context, string) ([]internal.CollectionItem, error) {
	return nil, nil
}
func (f *fakeRepository) PreviewImpact(context.Context, string, string) (internal.Impact, error) {
	if f.err != nil {
		return internal.Impact{}, f.err
	}
	return f.impact, nil
}

func TestCreateRelationMapsExactRevisionPins(t *testing.T) {
	repo := &fakeRepository{}
	h := handler{repo: repo}
	resp, err := h.CreateRelation(context.Background(), connect.NewRequest(&v1.CreateRelationRequest{Relation: &v1.Relation{SourceDraftId: "source", SourceRevisionId: "source-r3", TargetDraftId: "target", TargetRevisionId: "target-r8", ProjectId: "project", Kind: "family", Primary: true}}))
	if err != nil {
		t.Fatal(err)
	}
	if repo.relation.SourceRevisionID != "source-r3" || repo.relation.TargetRevisionID != "target-r8" || !repo.relation.Primary {
		t.Fatalf("revision pins were not mapped: %#v", repo.relation)
	}
	if resp.Msg.Relation == nil || resp.Msg.Relation.Id != "relation-1" || resp.Msg.Relation.TargetRevisionId != "target-r8" {
		t.Fatalf("bad response mapping: %#v", resp.Msg)
	}
}

func TestCreateRelationRejectsMissingRequestAndMapsConflicts(t *testing.T) {
	h := handler{repo: &fakeRepository{}}
	_, err := h.CreateRelation(context.Background(), connect.NewRequest(&v1.CreateRelationRequest{}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("missing relation returned %v", err)
	}
	h.repo = &fakeRepository{err: internal.ErrConflict}
	_, err = h.CreateRelation(context.Background(), connect.NewRequest(&v1.CreateRelationRequest{Relation: &v1.Relation{}}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("conflict returned %v", err)
	}
}

func TestPreviewImpactIsExplicitlyNonMutating(t *testing.T) {
	repo := &fakeRepository{impact: internal.Impact{DraftID: "d1", Operation: "archive", Relationships: 2, CollectionMemberships: 1, Attachments: 3, HasDraftSlot: true, HasApproval: true}}
	h := handler{repo: repo}
	resp, err := h.PreviewImpact(context.Background(), connect.NewRequest(&v1.PreviewImpactRequest{DraftId: "d1", Operation: "archive"}))
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Msg.PreviewOnly || resp.Msg.CascadesApplied || resp.Msg.Relationships != 2 || resp.Msg.CollectionMemberships != 1 || !resp.Msg.HasApproval {
		t.Fatalf("preview claimed mutation or lost impact data: %#v", resp.Msg)
	}
	repo.err = errors.New("storage unavailable")
	_, err = h.PreviewImpact(context.Background(), connect.NewRequest(&v1.PreviewImpactRequest{DraftId: "d1", Operation: "archive"}))
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("storage error returned %v", err)
	}
}

func TestListCollectionsMapsProjectScopedResults(t *testing.T) {
	h := handler{repo: &fakeRepository{}}
	resp, err := h.ListCollections(context.Background(), connect.NewRequest(&v1.ListCollectionsRequest{ProjectId: "p1"}))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Msg.Collections) != 1 || resp.Msg.Collections[0].Id != "c1" || resp.Msg.Collections[0].ProjectId != "p1" {
		t.Fatalf("unexpected collection mapping: %#v", resp.Msg.Collections)
	}
}

func TestRemoveCollectionItemRequiresExactItemAndMapsResult(t *testing.T) {
	h := handler{repo: &fakeRepository{}}
	_, err := h.RemoveCollectionItem(context.Background(), connect.NewRequest(&v1.RemoveCollectionItemRequest{}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("missing item returned %v", err)
	}
	resp, err := h.RemoveCollectionItem(context.Background(), connect.NewRequest(&v1.RemoveCollectionItemRequest{Item: &v1.CollectionItem{CollectionId: "c1", DraftId: "d1", RevisionId: "r2"}}))
	if err != nil || resp.Msg.Item.GetRevisionId() != "r2" {
		t.Fatalf("exact membership response failed: %#v, %v", resp, err)
	}
}
