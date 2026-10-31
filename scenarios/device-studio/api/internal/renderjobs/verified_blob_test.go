package renderjobs

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/vrooli/api-core/blobstore"
)

type countedBlobWrites struct {
	blobstore.BlobStore
	writes int
	onGet  func()
}

func (b *countedBlobWrites) Put(ctx context.Context, key string, reader io.Reader, mime string) error {
	b.writes++
	return b.BlobStore.Put(ctx, key, reader, mime)
}

func (b *countedBlobWrites) Get(ctx context.Context, key string) (io.ReadCloser, string, error) {
	reader, mime, err := b.BlobStore.Get(ctx, key)
	if b.onGet != nil {
		b.onGet()
	}
	return reader, mime, err
}

func TestPutVerifiedReusesOnlyMatchingImmutableInputs(t *testing.T) {
	data := []byte("worker-v1")
	digest := digestBytes(data)
	for _, tc := range []struct {
		name, key, existing, mime string
		wantWrites                int
	}{
		{"worker", workerBlobPrefix + digest + "/device-studio-api", string(data), "application/octet-stream", 0},
		{"source", inputBlobPrefix + digest + "/screen.png", string(data), "application/octet-stream", 0},
		{"closure", closureBlobPrefix + digest + ".json", string(data), "application/octet-stream", 0},
		{"missing", workerBlobPrefix + digest + "/device-studio-api", "", "", 1},
		{"changed", workerBlobPrefix + digest + "/device-studio-api", "worker-v2", "application/octet-stream", 1},
		{"trailing bytes", workerBlobPrefix + digest + "/device-studio-api", string(data) + "!", "application/octet-stream", 1},
		{"wrong MIME", workerBlobPrefix + digest + "/device-studio-api", string(data), "text/plain", 1},
		{"intent", intentBlobPrefix + digest + "/intent.json", string(data), "application/octet-stream", 1},
		{"receipt", receiptBlobPrefix + digest + "/receipt.json", string(data), "application/octet-stream", 1},
		{"unbound key", workerBlobPrefix + "different/worker", string(data), "application/octet-stream", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, ctx, _ := integrationCoordinator(t, &integrationOps{}, false)
			store := &countedBlobWrites{BlobStore: c.blobs}
			c.blobs = store
			if tc.existing != "" {
				if err := store.BlobStore.Put(ctx, tc.key, bytes.NewBufferString(tc.existing), tc.mime); err != nil {
					t.Fatal(err)
				}
			}
			route, err := c.route(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if err := c.putVerified(ctx, route, tc.key, data, "application/octet-stream", digest); err != nil {
				t.Fatal(err)
			}
			if store.writes != tc.wantWrites {
				t.Fatalf("blob writes = %d; want %d", store.writes, tc.wantWrites)
			}
			reader, mime, err := store.BlobStore.Get(ctx, tc.key)
			if err != nil {
				t.Fatal(err)
			}
			got, err := io.ReadAll(reader)
			_ = reader.Close()
			if err != nil || !bytes.Equal(got, data) || mime != "application/octet-stream" {
				t.Fatalf("stored immutable input differs: %q, %q, %v", got, mime, err)
			}
		})
	}
}

func TestPutVerifiedRejectsChangedRouteOrCallerPayloadBeforeWriting(t *testing.T) {
	for _, changeRoute := range []bool{false, true} {
		c, ctx, _ := integrationCoordinator(t, &integrationOps{}, false)
		store := &countedBlobWrites{BlobStore: c.blobs}
		c.blobs = store
		data := []byte("worker-v1")
		digest := digestBytes(data)
		key := workerBlobPrefix + digest + "/device-studio-api"
		route, err := c.route(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if changeRoute {
			store.onGet = func() {
				if err := c.roots.ClearTestRoots("integration-lease"); err != nil {
					t.Fatal(err)
				}
			}
		} else {
			data = []byte("worker-v2")
		}
		if err := c.putVerified(ctx, route, key, data, "application/octet-stream", digest); err == nil {
			t.Fatal("changed binding was accepted")
		}
		if store.writes != 0 {
			t.Fatalf("wrote %d blobs after binding changed", store.writes)
		}
	}
}
