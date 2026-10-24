package sceneprep

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/vrooli/api-core/blobstore"
	database "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/preparation"
	"google.golang.org/protobuf/proto"
)

func preparationService(t *testing.T) (*Service, *v1.CreatePhotographRequest) {
	t.Helper()
	db := databasetest.NewSQLite(t)
	if err := database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(Schema)); err != nil {
		t.Fatal(err)
	}
	blobs := blobstore.NewMemoryBlobStore()
	m, files := fixture()
	req := &v1.CreatePhotographRequest{Manifest: m, Label: "Desk background"}
	for name, raw := range files {
		ref := "out/" + name
		if err := blobs.Put(context.Background(), ref, bytes.NewReader(raw), "application/octet-stream"); err != nil {
			t.Fatal(err)
		}
		req.Artifacts = append(req.Artifacts, &v1.ArtifactBinding{ArtifactPath: name, StoredRef: ref})
	}
	return &Service{Repository: NewSQLRepository(db), Blobs: blobs}, req
}

func TestPreparationRepairsMissingContentOnExactRetry(t *testing.T) {
	s, req := preparationService(t)
	ctx := context.Background()
	first, err := s.Create(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	for _, ref := range []string{first.BundleRef, first.SourcePreviewRef, first.CleanPlatePreviewRef} {
		if err := s.Blobs.Delete(ctx, ref); err != nil {
			t.Fatal(err)
		}
	}
	again, err := s.Create(ctx, req)
	if err != nil || !proto.Equal(first, again) {
		t.Fatalf("retry lost immutable identity: %v %v", again, err)
	}
	for _, ref := range []string{first.BundleRef, first.SourcePreviewRef, first.CleanPlatePreviewRef} {
		r, _, err := s.Blobs.Get(ctx, ref)
		if err != nil {
			t.Fatalf("retry did not restore %s: %v", ref, err)
		}
		r.Close()
	}
}

func TestPreparationMeasuresActualRemovalMaskAndRetainsInlineProof(t *testing.T) {
	s, req := preparationService(t)
	ctx := context.Background()
	img := image.NewGray(image.Rect(0, 0, 3, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 3; x++ {
			img.Set(x, y, color.White)
		}
	}
	var raw bytes.Buffer
	if err := png.Encode(&raw, img); err != nil {
		t.Fatal(err)
	}
	mask := artifact("removal.png", "image/png", raw.Bytes())
	if err := s.Blobs.Put(ctx, "mask/removal", bytes.NewReader(raw.Bytes()), "image/png"); err != nil {
		t.Fatal(err)
	}
	proof := `{"inpaint_crop":{"crop_to_mask":true,"context_pixels":160,"max_edge_px":1024}}`
	req.Manifest.Route = v1.PhotographRoute_PHOTOGRAPH_ROUTE_DEVICE_REMOVAL
	req.Manifest.Background.DeviceMask = mask
	req.Manifest.Background.RemovalMask = mask
	req.Manifest.Background.RepairProvenance = artifact("repair.json", "application/json", []byte(proof))
	req.Artifacts = append(req.Artifacts, &v1.ArtifactBinding{ArtifactPath: "removal.png", StoredRef: "mask/removal"}, &v1.ArtifactBinding{ArtifactPath: "repair.json", InlineJson: proof})
	p, err := s.Create(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if p.RepairAssessment == nil || p.RepairAssessment.MaskAreaFraction != 1 || p.RepairAssessment.RiskLevel != "high" || len(p.RepairAssessment.ContextClippedSides) != 4 {
		t.Fatalf("actual mask context not measured: %v", p.RepairAssessment)
	}
	if p.Manifest.Review.Decision != "pending" {
		t.Fatal("advisory measurement made a visual decision")
	}
	req.Artifacts[len(req.Artifacts)-1].StoredRef = "out/both-forms"
	if _, err := s.Create(ctx, req); !errors.Is(err, ErrInvalid) {
		t.Fatalf("two sources for one proof accepted: %v", err)
	}
}

func TestPreparationRetainsInputsAndReviewsAsImmutableChild(t *testing.T) {
	ctx := context.Background()
	s, req := preparationService(t)
	first, err := s.Create(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	// Reordered bindings and retry do not create a replacement identity.
	for i, j := 0, len(req.Artifacts)-1; i < j; i, j = i+1, j-1 {
		req.Artifacts[i], req.Artifacts[j] = req.Artifacts[j], req.Artifacts[i]
	}
	second, err := s.Create(ctx, req)
	if err != nil || !proto.Equal(first, second) {
		t.Fatalf("non-idempotent create: %v %v", second, err)
	}
	// Native stage output may be disposable; preparation bytes must survive it.
	for _, b := range req.Artifacts {
		if err := s.Blobs.Delete(ctx, b.StoredRef); err != nil {
			t.Fatal(err)
		}
	}
	firstReview := &v1.ReviewPhotographRequest{Id: first.Id, OperationId: "11111111-1111-4111-8111-111111111111", Review: &v1.VisualReview{Decision: "rejected", ReviewedBy: "visual reviewer", Rationale: "visible fill boundary", EvidenceRefs: []string{"retained/sample.mp4"}}}
	reviewed, err := s.Review(ctx, firstReview)
	if err != nil {
		t.Fatal(err)
	}
	if reviewed.Manifest.Review.ReviewedAtUnixMs <= 0 || reviewed.CreatedAtUnixMs != reviewed.Manifest.Review.ReviewedAtUnixMs {
		t.Fatal("owner review timestamp was not retained on the immutable child")
	}
	retry, err := s.Review(ctx, proto.Clone(firstReview).(*v1.ReviewPhotographRequest))
	if err != nil || !proto.Equal(reviewed, retry) {
		t.Fatalf("same operation retry did not return same child: %v %v", retry, err)
	}
	conflict := proto.Clone(firstReview).(*v1.ReviewPhotographRequest)
	conflict.Review.Rationale = "different review evidence"
	if _, err := s.Review(ctx, conflict); !errors.Is(err, ErrInvalid) {
		t.Fatalf("operation key reuse with changed review must fail: %v", err)
	}
	forged, err := s.Review(ctx, &v1.ReviewPhotographRequest{Id: first.Id, OperationId: "22222222-2222-4222-8222-222222222222", Review: &v1.VisualReview{Decision: "rejected", ReviewedBy: "visual reviewer", Rationale: "visible fill boundary", EvidenceRefs: []string{"retained/sample.mp4"}, ReviewedAtUnixMs: 1}})
	if err != nil {
		t.Fatal(err)
	}
	if forged.Manifest.Review.ReviewedAtUnixMs == 1 {
		t.Fatal("caller-controlled review timestamp was retained")
	}
	if reviewed.Id == first.Id || reviewed.ParentId != first.Id || reviewed.Manifest.Review.Decision != "rejected" {
		t.Fatalf("review did not create child: %v", reviewed)
	}
	original, err := s.Get(ctx, first.Id)
	if err != nil || original.Manifest.Review.Decision != "pending" {
		t.Fatalf("review mutated original: %v %v", original, err)
	}
	list, err := s.List(ctx, 30)
	if err != nil || len(list) != 3 {
		t.Fatalf("retained revisions lost: %v %v", list, err)
	}
	if list[0].Id != forged.Id || list[1].Id != reviewed.Id {
		t.Fatalf("review operation insertion order was not retained: parent=%d child=%d rows=%s,%s,%s", first.CreatedAtUnixMs, reviewed.CreatedAtUnixMs, list[0].Id, list[1].Id, list[2].Id)
	}
}

func TestPreparationRejectsUnownedSubstitutedAndPreapprovedSources(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Service, *v1.CreatePhotographRequest)
	}{
		{"host-path", func(_ *Service, r *v1.CreatePhotographRequest) { r.Artifacts[0].StoredRef = "/etc/passwd" }},
		{"duplicate-binding", func(_ *Service, r *v1.CreatePhotographRequest) { r.Artifacts[0] = r.Artifacts[1] }},
		{"changed-actual-bytes", func(s *Service, r *v1.CreatePhotographRequest) {
			_ = s.Blobs.Put(context.Background(), r.Artifacts[0].StoredRef, bytes.NewReader([]byte("changed")), "image/png")
		}},
		{"preapproved-source", func(_ *Service, r *v1.CreatePhotographRequest) {
			r.Manifest.Review = &v1.VisualReview{Decision: "accepted", ReviewedBy: "old reviewer", Rationale: "old unrelated shot", EvidenceRefs: []string{"old.mp4"}}
		}},
		{"preapproved-overscan", func(_ *Service, r *v1.CreatePhotographRequest) { r.Manifest.CameraEnvelope.OverscanReviewed = true }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, req := preparationService(t)
			tc.mutate(s, req)
			if _, err := s.Create(context.Background(), req); !errors.Is(err, ErrInvalid) {
				t.Fatalf("bad admission: %v", err)
			}
			list, err := s.List(context.Background(), 30)
			if err != nil || len(list) != 0 {
				t.Fatalf("invalid preparation retained: %v %v", list, err)
			}
		})
	}
}

func TestPreparationReviewRequiresRetainedExactBundle(t *testing.T) {
	s, req := preparationService(t)
	p, err := s.Create(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Blobs.Put(context.Background(), p.BundleRef, bytes.NewReader([]byte("substituted")), "application/zip"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Review(context.Background(), &v1.ReviewPhotographRequest{Id: p.Id, OperationId: "33333333-3333-4333-8333-333333333333", Review: &v1.VisualReview{Decision: "accepted", ReviewedBy: "reviewer", Rationale: "looks good", EvidenceRefs: []string{"shot.mp4"}}}); err == nil {
		t.Fatal("substituted retained bundle accepted")
	}
}
