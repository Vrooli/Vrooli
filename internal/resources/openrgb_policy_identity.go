package resources

import (
	"errors"
	"os"
)

// CurrentOpenRGBDevicePolicyID reads the selected platform policy without
// enumerating hardware, acquiring artifacts or changing permissions.
func (c *Controller) CurrentOpenRGBDevicePolicyID() (string, error) {
	if c == nil {
		return "", errors.New("OpenRGB resource owner unavailable")
	}
	manifest, err := c.managedServiceArtifactManifest(openRGBResourceName)
	if err != nil {
		return "", err
	}
	record, err := c.loadOpenRGBDevicePolicy(manifest)
	if err != nil {
		return "", err
	}
	if record == nil {
		return "", os.ErrNotExist
	}
	return record.ID()
}
