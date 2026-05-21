package manifest

import "context"

// Repository is the persistence seam for parsed manifests. Phase 2
// ships an in-memory implementation (no sqlite yet); Phase 4 adds the
// production sqlite cache + content-hash invalidation.
type Repository interface {
	SaveManifest(ctx context.Context, m ManifestDefinition) (ManifestDefinition, error)
	GetManifest(ctx context.Context, scenario string) (ManifestDefinition, error)
}
