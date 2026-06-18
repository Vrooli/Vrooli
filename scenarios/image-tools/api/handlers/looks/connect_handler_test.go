package looks

import (
	"context"
	"testing"

	"github.com/vrooli/api-core/blobstore"
	apidb "github.com/vrooli/api-core/database"

	internallooks "image-tools/internal/looks"
	"image-tools/internal/storage"
	"image-tools/internal/testutil/db"

	"connectrpc.com/connect"
	looksv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/looks"
)

func newHandler(t *testing.T) *connectHandler {
	t.Helper()
	d := db.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(internallooks.Schema)); err != nil {
		t.Fatalf("ensure schemas: %v", err)
	}
	store := internallooks.NewStore(d)
	blobs := storage.NewWithBlobStore(blobstore.NewMemoryBlobStore(), t.TempDir())
	return NewConnectHandler(Deps{Store: store, Blobs: blobs})
}

func TestHandlerListAndGet(t *testing.T) {
	ctx := context.Background()
	h := newHandler(t)

	list, err := h.ListLooks(ctx, connect.NewRequest(&looksv1.ListLooksRequest{}))
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list.Msg.GetLooks()) == 0 {
		t.Fatal("expected built-in Looks")
	}

	got, err := h.GetLook(ctx, connect.NewRequest(&looksv1.GetLookRequest{Id: "noir"}))
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Msg.GetLook().GetName() != "Noir" {
		t.Errorf("unexpected Look %q", got.Msg.GetLook().GetName())
	}

	_, err = h.GetLook(ctx, connect.NewRequest(&looksv1.GetLookRequest{Id: "nope"}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("want NotFound for unknown id, got %v", err)
	}
}

func TestHandlerCreateCompileRender(t *testing.T) {
	ctx := context.Background()
	h := newHandler(t)

	created, err := h.CreateLook(ctx, connect.NewRequest(&looksv1.CreateLookRequest{Look: &looksv1.Look{
		Name: "Cyan Pop",
		Kind: looksv1.LookKind_LOOK_KIND_CAMERA,
		Steps: []*looksv1.LookStep{
			{Operation: "adjust", Kind: looksv1.StepKind_STEP_KIND_DETERMINISTIC, Params: map[string]string{"saturation": "20"}},
		},
	}}))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	id := created.Msg.GetLook().GetId()
	if id != "cyan-pop" {
		t.Errorf("slug id = %q", id)
	}

	// Compile resolves the deterministic step + flags requires_image.
	comp, err := h.CompileLook(ctx, connect.NewRequest(&looksv1.CompileLookRequest{LookId: id, HasInput: true}))
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if !comp.Msg.GetRequiresImage() || len(comp.Msg.GetSteps()) != 1 {
		t.Errorf("unexpected compile result %+v", comp.Msg)
	}

	// RenderPreview stores a blob and persists the ref on the custom Look.
	rp, err := h.RenderPreview(ctx, connect.NewRequest(&looksv1.RenderPreviewRequest{LookId: id}))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if rp.Msg.GetThumbnailRef() == "" {
		t.Fatal("expected a thumbnail ref")
	}
	got, _ := h.GetLook(ctx, connect.NewRequest(&looksv1.GetLookRequest{Id: id}))
	if got.Msg.GetLook().GetThumbnailRef() != rp.Msg.GetThumbnailRef() {
		t.Errorf("thumbnail ref not persisted on custom Look: %q vs %q", got.Msg.GetLook().GetThumbnailRef(), rp.Msg.GetThumbnailRef())
	}
}

func TestHandlerCreateCollisionIsAlreadyExists(t *testing.T) {
	ctx := context.Background()
	h := newHandler(t)
	_, err := h.CreateLook(ctx, connect.NewRequest(&looksv1.CreateLookRequest{Look: &looksv1.Look{
		Name:  "noir",
		Steps: []*looksv1.LookStep{{Operation: "filter", Kind: looksv1.StepKind_STEP_KIND_DETERMINISTIC, Params: map[string]string{"filter": "grayscale"}}},
	}}))
	if connect.CodeOf(err) != connect.CodeAlreadyExists {
		t.Fatalf("want AlreadyExists for built-in collision, got %v", err)
	}
}

func TestHandlerRenderBuiltinDoesNotPersist(t *testing.T) {
	ctx := context.Background()
	h := newHandler(t)
	rp, err := h.RenderPreview(ctx, connect.NewRequest(&looksv1.RenderPreviewRequest{LookId: "polaroid-600"}))
	if err != nil {
		t.Fatalf("render built-in: %v", err)
	}
	if rp.Msg.GetThumbnailRef() == "" {
		t.Fatal("expected a rendered ref even for a built-in")
	}
	// The built-in seed entry stays read-only (no persisted thumbnail).
	got, _ := h.GetLook(ctx, connect.NewRequest(&looksv1.GetLookRequest{Id: "polaroid-600"}))
	if got.Msg.GetLook().GetThumbnailRef() != "" {
		t.Errorf("built-in Look must stay read-only, got thumbnail %q", got.Msg.GetLook().GetThumbnailRef())
	}
}
