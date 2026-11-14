package hostreqkit

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

// OpenRGBPermissionRequest contains only qualified physical selectors, never a
// caller-selected destination, command, transient device number or username.
type OpenRGBPermissionRequest struct {
	ArtifactSHA256 string
	DetectorSHA256 string
	OwnerUID       uint32
	Devices        []OpenRGBLinuxDevice
}

// OpenRGBPermissionPlan is the exact reviewable transaction. Its ID binds the
// expected prior root-owned rule and expiry as well as the physical grant.
// A host owner may persist this in operator state and expose only ID to clients.
type OpenRGBPermissionPlan struct {
	ID                 string
	Request            OpenRGBPermissionRequest
	Policy             OpenRGBLinuxPolicy
	RulePath           string
	RuleContent        string
	PreviousRuleSHA256 string
	CreatedAt          time.Time
	ExpiresAt          time.Time
	Nodes              []OpenRGBLinuxNode
}

// OpenRGBPermissionResult describes host metadata, not hardware output or a
// logged-out boot. A nonnil error may have a durable pending receipt.
type OpenRGBPermissionResult struct {
	PlanID       string
	Policy       OpenRGBLinuxPolicy
	ActivePolicy *OpenRGBLinuxPolicy
	ActivePlanID string
	Applied      bool
	RolledBack   bool
	Pending      bool
	Nodes        []string
}

const (
	openRGBPermissionHeader   = "# Managed by Vrooli selected OpenRGB device access v1\n"
	openRGBPermissionLifetime = 15 * time.Minute
)

func openRGBPermissionRules(p OpenRGBLinuxPolicy) (string, error) {
	if p.OwnerUID == 0 || p.GroupID != 0 {
		return "", errors.New("selected-node permissions require an unprivileged numeric owner, without group provisioning")
	}
	if err := ValidateOpenRGBLinuxDetectorScope(p); err != nil {
		return "", err
	}
	devices := slices.Clone(p.Devices)
	slices.SortFunc(devices, func(a, b OpenRGBLinuxDevice) int {
		return strings.Compare(a.Topology+"\x00"+a.Adapter+"\x00"+a.HardwareID, b.Topology+"\x00"+b.Adapter+"\x00"+b.HardwareID)
	})
	content := openRGBPermissionHeader + "SUBSYSTEM==\"i2c-dev\", ENV{VROOLI_OPENRGB_ROUTE}=\"\"\nSUBSYSTEM==\"hidraw\", ENV{VROOLI_OPENRGB_ROUTE}=\"\"\n"
	for _, d := range devices {
		// Udev quoted strings are a separate language. Reject its metacharacters
		// rather than escaping input into a pattern or an assignment.
		for _, v := range []string{d.Topology, d.HardwareID, d.Adapter} {
			if strings.ContainsAny(v, "\"\\$%*?[]\r\n\x00") {
				return "", errors.New("OpenRGB selector cannot be represented as a literal udev match")
			}
		}
		token := fmt.Sprintf("%x", sha256.Sum256([]byte(d.Detector+"\x00"+d.Topology+"\x00"+d.Adapter+"\x00"+d.HardwareID)))
		parts := strings.Split(d.HardwareID, ":")
		switch d.Kind {
		case "i2c":
			if len(parts) != 4 || d.Adapter == "" {
				return "", errors.New("invalid I2C physical selector")
			}
			path := "/devices/" + d.Topology + "/i2c-*/i2c-dev/i2c-*"
			content += fmt.Sprintf("SUBSYSTEM==\"i2c-dev\", ENV{DEVPATH}==\"%s\", ATTRS{vendor}==\"0x%s\", ATTRS{device}==\"0x%s\", ATTRS{subsystem_vendor}==\"0x%s\", ATTRS{subsystem_device}==\"0x%s\", ENV{VROOLI_OPENRGB_ROUTE}=\"%s\"\n", path, parts[0], parts[1], parts[2], parts[3], token)
			content += fmt.Sprintf("SUBSYSTEM==\"i2c-dev\", ENV{VROOLI_OPENRGB_ROUTE}==\"%s\", ATTRS{name}==\"%s\", OWNER:=\"%d\", MODE:=\"0600\"\n", token, d.Adapter, p.OwnerUID)
		case "hidraw":
			if len(parts) != 3 {
				return "", errors.New("invalid USB interface selector")
			}
			path := "/devices/" + d.Topology + "/*/hidraw/hidraw*"
			content += fmt.Sprintf("SUBSYSTEM==\"hidraw\", ENV{DEVPATH}==\"%s\", ATTRS{idVendor}==\"%s\", ATTRS{idProduct}==\"%s\", ENV{VROOLI_OPENRGB_ROUTE}=\"%s\"\n", path, parts[0], parts[1], token)
			content += fmt.Sprintf("SUBSYSTEM==\"hidraw\", ENV{VROOLI_OPENRGB_ROUTE}==\"%s\", ATTRS{bInterfaceNumber}==\"%s\", OWNER:=\"%d\", MODE:=\"0600\"\n", token, parts[2], p.OwnerUID)
		default:
			return "", errors.New("unsupported OpenRGB permission device kind")
		}
	}
	return content, nil
}

func (p OpenRGBPermissionPlan) identity() (string, error) {
	p.ID = ""
	p.Nodes = nil // Transient node numbers are observations, never authority.
	data, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(data)), nil
}
