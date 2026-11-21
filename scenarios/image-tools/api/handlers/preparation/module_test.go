package preparation

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/blobstore"

	preparationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/preparation"
	"image-tools/internal/sceneprep"
)

// fakeRepository is the immutable-store seam in memory. The handler boundary is
// what these tests exercise: error mapping and proto pass-through, not the SQL
// encoding (covered by internal/sceneprep).
type fakeRepository struct {
	items     map[string]*preparationv1.StoredPhotograph
	putErr    error
	getErr    error
	listErr   error
	reviewErr error
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{items: map[string]*preparationv1.StoredPhotograph{}}
}

func (r *fakeRepository) Put(_ context.Context, p *preparationv1.StoredPhotograph) error {
	if r.putErr != nil {
		return r.putErr
	}
	r.items[p.GetId()] = p
	return nil
}

func (r *fakeRepository) Get(_ context.Context, id string) (*preparationv1.StoredPhotograph, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	p, ok := r.items[id]
	if !ok {
		return nil, sceneprep.ErrNotFound
	}
	return p, nil
}

func (r *fakeRepository) List(_ context.Context, _ int) ([]*preparationv1.StoredPhotograph, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	out := make([]*preparationv1.StoredPhotograph, 0, len(r.items))
	for _, p := range r.items {
		out = append(out, p)
	}
	return out, nil
}

func (r *fakeRepository) ReviewOperation(context.Context, string) (string, *preparationv1.StoredPhotograph, error) {
	if r.reviewErr != nil {
		return "", nil, r.reviewErr
	}
	return "", nil, nil
}

func (r *fakeRepository) PutReviewOperation(_ context.Context, _, _ string, p *preparationv1.StoredPhotograph) (*preparationv1.StoredPhotograph, error) {
	if r.putErr != nil {
		return nil, r.putErr
	}
	r.items[p.GetId()] = p
	return p, nil
}

type fakeBlobs struct {
	objects map[string][]byte
	getErr  error
}

func (b *fakeBlobs) Put(_ context.Context, key string, r io.Reader, _ string) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	if b.objects == nil {
		b.objects = map[string][]byte{}
	}
	b.objects[key] = data
	return nil
}

func (b *fakeBlobs) Get(_ context.Context, key string) (io.ReadCloser, string, error) {
	if b.getErr != nil {
		return nil, "", b.getErr
	}
	data, ok := b.objects[key]
	if !ok {
		return nil, "", errors.New("missing object")
	}
	return io.NopCloser(bytes.NewReader(data)), "application/octet-stream", nil
}

func (b *fakeBlobs) Delete(_ context.Context, key string) error {
	delete(b.objects, key)
	return nil
}

var _ blobstore.BlobStore = (*fakeBlobs)(nil)

func newTestHandler(repo *fakeRepository, blobs *fakeBlobs) *Handler {
	return &Handler{Service: &sceneprep.Service{Repository: repo, Blobs: blobs}}
}

func TestGetPhotographPassesThroughAndMapsMissing(t *testing.T) {
	repo := newFakeRepository()
	id := strings.Repeat("a", 64)
	repo.items[id] = &preparationv1.StoredPhotograph{Id: id, Label: "kitchen"}
	h := newTestHandler(repo, &fakeBlobs{})

	resp, err := h.GetPhotograph(context.Background(), connect.NewRequest(&preparationv1.GetPhotographRequest{Id: id}))
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got := resp.Msg.GetPhotograph().GetLabel(); got != "kitchen" {
		t.Fatalf("label = %q", got)
	}

	_, err = h.GetPhotograph(context.Background(), connect.NewRequest(&preparationv1.GetPhotographRequest{Id: strings.Repeat("b", 64)}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("missing id code = %v, want NotFound", connect.CodeOf(err))
	}
}

func TestListPhotographsPassesThroughRepositoryError(t *testing.T) {
	repo := newFakeRepository()
	repo.listErr = errors.New("ledger unavailable")
	h := newTestHandler(repo, &fakeBlobs{})

	_, err := h.ListPhotographs(context.Background(), connect.NewRequest(&preparationv1.ListPhotographsRequest{Limit: 10}))
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("code = %v, want Internal", connect.CodeOf(err))
	}
}

func TestCreatePhotographRejectsInvalidBeforeTouchingBlobs(t *testing.T) {
	h := newTestHandler(newFakeRepository(), &fakeBlobs{})
	_, err := h.CreatePhotograph(context.Background(), connect.NewRequest(&preparationv1.CreatePhotographRequest{Label: "  "}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want InvalidArgument", connect.CodeOf(err))
	}
}

func TestReviewPhotographRequiresOperationIdentity(t *testing.T) {
	h := newTestHandler(newFakeRepository(), &fakeBlobs{})
	_, err := h.ReviewPhotograph(context.Background(), connect.NewRequest(&preparationv1.ReviewPhotographRequest{
		Id:     "abc",
		Review: &preparationv1.VisualReview{Decision: "accepted"},
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want InvalidArgument", connect.CodeOf(err))
	}
}

func TestAssessRepairRejectsUnownedRefAndMissingObject(t *testing.T) {
	h := newTestHandler(newFakeRepository(), &fakeBlobs{})

	_, err := h.AssessRepair(context.Background(), connect.NewRequest(&preparationv1.AssessRepairRequest{MaskRef: "/etc/passwd"}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("unowned ref code = %v, want InvalidArgument", connect.CodeOf(err))
	}

	_, err = h.AssessRepair(context.Background(), connect.NewRequest(&preparationv1.AssessRepairRequest{MaskRef: "mask/retained.png"}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("missing mask code = %v, want NotFound", connect.CodeOf(err))
	}
}

func TestMapErrorIsExhaustiveOverTheSentinelSet(t *testing.T) {
	cases := []struct {
		err  error
		want connect.Code
	}{
		{sceneprep.ErrInvalid, connect.CodeInvalidArgument},
		{sceneprep.ErrNotFound, connect.CodeNotFound},
		{errors.New("boom"), connect.CodeInternal},
	}
	for _, tc := range cases {
		if got := connect.CodeOf(mapError(tc.err)); got != tc.want {
			t.Errorf("mapError(%v) code = %v, want %v", tc.err, got, tc.want)
		}
	}
}
