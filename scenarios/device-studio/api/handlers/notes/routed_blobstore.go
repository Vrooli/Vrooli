package notes

import (
	"device-studio/internal/routedblobstore"
	"fmt"

	"github.com/vrooli/api-core/blobstore"
	"github.com/vrooli/api-core/filerouting"
)

func newRoutedFilesystemBlobStore(roots *filerouting.RoutedRoots) (blobstore.BlobStore, error) {
	store, err := routedblobstore.New(roots, "attachments")
	if err != nil {
		if roots == nil {
			return nil, fmt.Errorf("notes attachment file roots are required")
		}
		return nil, err
	}
	return store, nil
}
