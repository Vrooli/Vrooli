package contextcapture

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"image"
	"image/png"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/blobstore"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/owneridentity"
	"github.com/vrooli/api-core/targetmodel"
	wire "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/contextcapture"
	rpc "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/contextcapture/contextcapture_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
	domain "portal/internal/contextcapture"
)

type identities struct{ now time.Time }

func (v identities) Validate(_ context.Context, token string) (owneridentity.Identity, error) {
	if token != "alice" && token != "bob" {
		return owneridentity.Identity{}, owneridentity.ErrUnauthenticated
	}
	return owneridentity.Identity{Subject: token, ExpiresAt: v.now.Add(time.Hour)}, nil
}

func authorized[T any](value *T, token string) *connect.Request[T] {
	r := connect.NewRequest(value)
	r.Header().Set("Authorization", "Bearer "+token)
	return r
}

func TestContextHTTPRequiresOwnerAndPreservesImage(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	db := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(Schema)))
	svc := domain.NewService(domain.NewSQLiteRepository(db), blobstore.NewMemoryBlobStore(), func() time.Time { return now })
	handler := NewHandler(svc, identities{now: now}, func() time.Time { return now })
	router := mux.NewRouter()
	Module(handler).Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()
	client := rpc.NewContextCaptureServiceClient(server.Client(), server.URL)
	var encoded bytes.Buffer
	require.NoError(t, png.Encode(&encoded, image.NewRGBA(image.Rect(0, 0, 2, 2))))
	input := &wire.ImportRequest{RequestId: uuid.NewString(), Source: &wire.Source{Surface: (targetmodel.SurfaceRef{Target: targetmodel.TargetRef{OwnerScenario: "vrooli-bridge", ResourceID: "host", HostNodeID: "host"}, OwnerScenario: "device-control", SurfaceID: "desktop"}).Proto(), CaptureId: uuid.NewString(), DisplayId: "d", GeometryRevision: "g", CapturedAt: timestamppb.New(now), Bounds: &wire.Bounds{X: -100, Width: 2, Height: 2}}, Region: &wire.Region{Width: 1, Height: 1}, Png: encoded.Bytes(), RetentionSeconds: 60}
	_, err := client.Import(ctx, connect.NewRequest(input))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	var rows int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM context_capsules").Scan(&rows))
	require.Zero(t, rows)
	cancelID := uuid.NewString()
	_, err = client.CancelImport(ctx, connect.NewRequest(&wire.ReconcileImportRequest{RequestId: cancelID}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	_, err = client.CancelImport(ctx, authorized(&wire.ReconcileImportRequest{RequestId: cancelID}, "alice"))
	require.NoError(t, err)
	cancelled := *input
	cancelled.RequestId = cancelID
	_, err = client.Import(ctx, authorized(&cancelled, "alice"))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	statusRequest := &wire.ReconcileImportRequest{RequestId: input.RequestId}
	status, err := client.ReconcileImport(ctx, authorized(statusRequest, "alice"))
	require.NoError(t, err)
	require.Equal(t, wire.ImportState_IMPORT_STATE_ABSENT, status.Msg.State)
	imported, err := client.Import(ctx, authorized(input, "alice"))
	require.NoError(t, err)
	require.Equal(t, "no-store", imported.Header().Get("Cache-Control"))
	retry, err := client.Import(ctx, authorized(input, "alice"))
	require.NoError(t, err)
	require.Equal(t, imported.Msg.Id, retry.Msg.Id)
	require.Equal(t, imported.Msg.ExpiresAt.AsTime(), retry.Msg.ExpiresAt.AsTime())
	status, err = client.ReconcileImport(ctx, authorized(statusRequest, "alice"))
	require.NoError(t, err)
	require.Equal(t, wire.ImportState_IMPORT_STATE_READY, status.Msg.State)
	require.Equal(t, imported.Msg.Id, status.Msg.Document.Id)
	require.Equal(t, "no-store", status.Header().Get("Cache-Control"))
	status, err = client.ReconcileImport(ctx, authorized(statusRequest, "bob"))
	require.NoError(t, err)
	require.Equal(t, wire.ImportState_IMPORT_STATE_ABSENT, status.Msg.State)
	require.Nil(t, status.Msg.Document)
	ref := &wire.ReferenceRequest{Id: imported.Msg.Id}
	read, err := client.Read(ctx, authorized(ref, "alice"))
	require.NoError(t, err)
	require.Equal(t, imported.Msg.OriginalSha256, read.Msg.Document.OriginalSha256)
	require.NotEmpty(t, read.Msg.Png)
	rendered, err := client.Render(ctx, authorized(ref, "alice"))
	require.NoError(t, err)
	picture, err := png.Decode(bytes.NewReader(rendered.Msg.Png))
	require.NoError(t, err)
	require.Equal(t, image.Rect(0, 0, 1, 1), picture.Bounds())
	digest := sha256.Sum256(rendered.Msg.Png)
	require.Equal(t, hex.EncodeToString(digest[:]), rendered.Msg.RenderedSha256)
	require.Equal(t, imported.Msg.OriginalSha256, rendered.Msg.Document.OriginalSha256)
	_, err = client.Render(ctx, authorized(ref, "bob"))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))

	_, err = client.Read(ctx, authorized(ref, "bob"))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	_, err = client.Delete(ctx, authorized(ref, "bob"))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	_, err = client.Read(ctx, authorized(ref, "invalid"))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	_, err = client.CancelImport(ctx, authorized(statusRequest, "alice"))
	require.NoError(t, err)
	status, err = client.ReconcileImport(ctx, authorized(statusRequest, "alice"))
	require.NoError(t, err)
	require.Equal(t, wire.ImportState_IMPORT_STATE_UNAVAILABLE, status.Msg.State)
	require.Nil(t, status.Msg.Document)
	_, err = client.Read(ctx, authorized(ref, "alice"))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}
