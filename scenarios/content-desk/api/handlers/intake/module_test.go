package intake

import (
	"context"
	"encoding/json"
	"testing"

	"connectrpc.com/connect"
	internal "content-desk/internal/intake"
	intakev1 "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/intake"
)

type fakeService struct{ got internal.Request }

func (f *fakeService) Preview(_ context.Context, r internal.Request) (internal.Preview, error) {
	f.got = r
	return internal.Preview{Source: r.Source, Assessments: []internal.Assessment{{ExternalID: r.Items[0].ExternalID, Status: "admissible"}}}, nil
}
func (f *fakeService) Admit(_ context.Context, r internal.Request) ([]internal.StoredItem, error) {
	f.got = r
	return nil, nil
}
func (f *fakeService) List(context.Context, string) ([]internal.StoredItem, error) { return nil, nil }
func (f *fakeService) Reconcile(context.Context, string, internal.Request) ([]internal.StoredItem, error) {
	return nil, nil
}
func (f *fakeService) Promote(context.Context, internal.PromoteRequest) ([]internal.PromotedDraft, error) {
	return nil, nil
}

func TestPreviewPreservesUnknownSourceFieldsAndReturnsAssessment(t *testing.T) {
	svc := &fakeService{}
	h := handler{service: svc}
	resp, err := h.Preview(context.Background(), connect.NewRequest(&intakev1.PreviewRequest{
		Source:       &intakev1.Source{ArtifactId: "r1", RevisionId: "rev-7", Producer: "fixture", ImportIntent: "adapt"},
		RawItemsJson: []string{`{"external_id":"article-1","kind":"article","title":"Example","body":"Useful text","future_field":{"keep":true}}`},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if svc.got.Source.ArtifactID != "r1" || svc.got.Source.RevisionID != "rev-7" || len(svc.got.Items) != 1 {
		t.Fatalf("request lost source/item: %#v", svc.got)
	}
	var raw map[string]any
	if err := json.Unmarshal(svc.got.Items[0].RawPayload, &raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw["future_field"]; !ok {
		t.Fatalf("unknown source field was lost: %s", svc.got.Items[0].RawPayload)
	}
	var result internal.Preview
	if err := json.Unmarshal([]byte(resp.Msg.ResultJson), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Assessments) != 1 || result.Assessments[0].ExternalID != "article-1" {
		t.Fatalf("bad result: %#v", result)
	}
}

func TestPreviewRejectsMalformedRawItem(t *testing.T) {
	h := handler{service: &fakeService{}}
	_, err := h.Preview(context.Background(), connect.NewRequest(&intakev1.PreviewRequest{RawItemsJson: []string{`{"external_id":`}}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("got %v, want InvalidArgument", err)
	}
}
