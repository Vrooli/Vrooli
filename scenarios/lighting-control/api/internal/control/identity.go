package control

import (
	"fmt"
	"regexp"
	"strings"
)

// SDKHardwareIdentity holds controller-data evidence independent of the
// server-assigned device ID. Version is informational and can change on update.
type SDKHardwareIdentity struct {
	Type        uint32
	Name        string
	Vendor      string
	Description string
	Version     string
	Serial      string
	Location    string
}

// decodeSDKHardwareIdentityV6 reads the controller-data header immediately
// after the full body length and before the controller mode count.
func decodeSDKHardwareIdentityV6(c *cursor) (SDKHardwareIdentity, error) {
	var identity SDKHardwareIdentity
	var err error
	identity.Type, err = c.u32()
	if err != nil {
		return SDKHardwareIdentity{}, err
	}
	for _, field := range []*string{&identity.Name, &identity.Vendor, &identity.Description, &identity.Version, &identity.Serial, &identity.Location} {
		*field, err = c.str16()
		if err != nil {
			return SDKHardwareIdentity{}, err
		}
	}
	return identity, nil
}

// OpenRGB's Linux PCI I2C bus description includes an adapter label, a PCI
// address, and the allocated /dev/i2c number. The ENE driver adds its address
// on that bus. Keep the adapter label to distinguish channels on the same PCI
// device and drop the allocated device node.
var sdkLinuxPCII2CLocation = regexp.MustCompile(`(?i)^I2C: ([^\r\n]+) at ((?:[a-f0-9]{4}:)?[a-f0-9]{1,2}:[a-f0-9]{1,2}\.[0-7])\s+\(/dev/i2c-[0-9]+\), address (0x[a-f0-9]{2})$`)

// stableControllerIdentity returns a candidate identity and its provenance.
// The caller must require a unique live match before enrollment or replay.
// A blank identity leaves the device available for discovery only.
func stableControllerIdentity(identity SDKHardwareIdentity) (id, source string) {
	serial := strings.TrimSpace(identity.Serial)
	switch strings.ToLower(serial) {
	case "n/a", "none", "unknown", "not specified", "default string", "to be filled by o.e.m.":
		serial = ""
	}
	if strings.Trim(serial, "0") == "" || strings.ContainsRune(serial, 0) {
		serial = ""
	}
	hardware := fmt.Sprintf("%d|%q|%q|%q", identity.Type, identity.Vendor, identity.Name, identity.Description)
	if serial != "" {
		return stableID("controller-", fmt.Sprintf("openrgb|serial|%s|%q", hardware, serial)), "openrgb_sdk_serial"
	}
	location := sdkLinuxPCII2CLocation.FindStringSubmatch(strings.TrimSpace(identity.Location))
	if location == nil {
		return "", ""
	}
	topology := fmt.Sprintf("%q|%q|%q", location[1], strings.ToLower(location[2]), strings.ToLower(location[3]))
	return stableID("controller-", "openrgb|pci_i2c|"+hardware+"|"+topology), "openrgb_linux_pci_i2c_location"
}
