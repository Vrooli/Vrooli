package credentialauthority

import (
	"context"
	"path/filepath"

	"github.com/vrooli/vrooli/internal/artifactlock"
	"github.com/vrooli/vrooli/internal/config"
)

// AcquireCloudflareConnectorLock serializes connector credential mutation with
// managed connector launch. All participants must use the same operator home.
// The returned release must be called after mutation or launch completes.
func AcquireCloudflareConnectorLock(ctx context.Context) (func(), error) {
	return artifactlock.AcquireWithPathContext(ctx, "cloudflare-connector", func(string) (string, error) {
		home, err := config.VrooliHome()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "state", "locks", "cloudflare-connector.lock"), nil
	})
}
