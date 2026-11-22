package review

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	internal "content-desk/internal/review"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/review"
)

type fakeService struct {
	createdID string
	pkg       internal.EditorialPackage
	err       error
}

func (*fakeService) Record(context.Context, string, []internal.Verdict) (internal.Run, error) {
	return internal.Run{}, nil
}
func (*fakeService) List(context.Context) ([]internal.Run, error) { return nil, nil }
func (f *fakeService) CreateEditorialPackage(_ context.Context, draft, revision, body, key string) (internal.EditorialPackage, error) {
	f.createdID = draft + "|" + revision + "|" + body + "|" + key
	if f.err != nil {
		return internal.EditorialPackage{}, f.err
	}
	return f.pkg, nil
}
func (f *fakeService) GetEditorialPackage(_ context.Context, id string) (internal.EditorialPackage, error) {
	if f.err != nil {
		return internal.EditorialPackage{}, f.err
	}
	f.createdID = id
	return f.pkg, nil
}

func TestCreateEditorialPackageMapsExactWordingScope(t *testing.T) {
	service := &fakeService{pkg: internal.EditorialPackage{ID: "pkg-1", DraftID: "draft-1", SourceRevisionID: "rev-2", Scope: internal.EditorialPackageScope, Body: "Exact wording", BodySHA256: "body-hash", PackageSHA256: "package-hash", ManifestJSON: `{"context":{"channel":"linkedin"}}`, CreatedAt: time.Date(2026, 11, 21, 0, 0, 0, 0, time.UTC)}}
	h := handler{service: service}
	resp, err := h.CreateEditorialPackage(context.Background(), connect.NewRequest(&v1.CreateEditorialPackageRequest{DraftId: "draft-1", ExpectedRevisionId: "rev-2", ExpectedBody: "Exact wording", IdempotencyKey: "op-1"}))
	if err != nil {
		t.Fatal(err)
	}
	if service.createdID != "draft-1|rev-2|Exact wording|op-1" {
		t.Fatalf("request mapping lost exact basis: %q", service.createdID)
	}
	p := resp.Msg.EditorialPackage
	if p == nil || p.Scope != internal.EditorialPackageScope || p.Body != "Exact wording" || p.ManifestJson != service.pkg.ManifestJSON || p.PackageSha256 != "package-hash" {
		t.Fatalf("bad package response: %#v", p)
	}
}

func TestReviewHandlerRejectsStaleCreateAndMissingPackage(t *testing.T) {
	service := &fakeService{err: errors.New("draft revision changed; refresh the exact wording")}
	h := handler{service: service}
	_, err := h.CreateEditorialPackage(context.Background(), connect.NewRequest(&v1.CreateEditorialPackageRequest{DraftId: "draft-1", ExpectedRevisionId: "old", ExpectedBody: "Old text", IdempotencyKey: "op-1"}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("stale create returned %v", err)
	}
	_, err = h.GetEditorialPackage(context.Background(), connect.NewRequest(&v1.GetEditorialPackageRequest{Id: "missing"}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("missing package returned %v", err)
	}
}

func TestReviewHandlerRejectsNilVerdict(t *testing.T) {
	h := handler{service: &fakeService{}}
	_, err := h.RecordReviewRun(context.Background(), connect.NewRequest(&v1.RecordReviewRunRequest{DraftId: "draft-1", Verdicts: []*v1.Verdict{nil}}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("nil verdict returned %v", err)
	}
}
