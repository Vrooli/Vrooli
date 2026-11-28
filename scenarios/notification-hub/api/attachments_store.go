package main

import (
	"context"

	"github.com/vrooli/api-core/blobstore"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"

	"notification-hub/internal/hub"
)

// attachmentsDir is the kind directory of the decision-attachments storage
// entry in .vrooli/service.json, below the scenario data root.
const attachmentsDir = "attachments"

// configureAttachmentStore points decision-packet attachments at the scenario
// data root. The root is chosen per request, so a test-mode request writes to
// its lease-owned root and never to the live tree.
func configureAttachmentStore(service *hub.Service, fileRoots *filerouting.RoutedRoots) {
	service.SetAttachmentStore(func(ctx context.Context) (blobstore.BlobStore, error) {
		path, err := fileRootPath(ctx, fileRoots, storage.ClassData, attachmentsDir)
		if err != nil {
			return nil, err
		}
		return blobstore.NewFilesystemBlobStore(path), nil
	})
}
