package host

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/hostinventory"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
)

type inventoryCheck struct {
	id          string
	title       string
	description string
	importance  string
	run         func(hostinventory.HostInventory) checks.Result
	collector   hostinventory.Collector
}

func (c *inventoryCheck) ID() string                 { return c.id }
func (c *inventoryCheck) Title() string              { return c.title }
func (c *inventoryCheck) Description() string        { return c.description }
func (c *inventoryCheck) Importance() string         { return c.importance }
func (c *inventoryCheck) Category() checks.Category  { return checks.CategorySystem }
func (c *inventoryCheck) IntervalSeconds() int       { return 300 }
func (c *inventoryCheck) Platforms() []platform.Type { return nil }

func (c *inventoryCheck) Run(ctx context.Context) checks.Result {
	start := time.Now()
	inv, err := c.collector.Collect(ctx)
	if err != nil {
		return checks.Result{
			CheckID: c.id,
			Status:  checks.StatusWarning,
			Message: "Host inventory collection was degraded",
			Details: map[string]any{
				"probeStatus":     inv.ProbeStatus,
				"error":           err.Error(),
				"recommendations": []string{"Inspect host inventory probe errors before relying on this check."},
			},
			Timestamp: time.Now().UTC(),
			Duration:  time.Since(start),
		}
	}
	result := c.run(inv)
	result.CheckID = c.id
	result.Timestamp = time.Now().UTC()
	result.Duration = time.Since(start)
	if result.Details == nil {
		result.Details = map[string]any{}
	}
	result.Details["probeStatus"] = inv.ProbeStatus
	result.Details["inventoryFingerprint"] = inv.Fingerprint
	return result
}

func NewChecks(collector hostinventory.Collector) []checks.Check {
	if collector == nil {
		collector = hostinventory.NewCachedCollector(hostinventory.NewCollector(hostinventory.CollectorOptions{}), 30*time.Second)
	}
	return []checks.Check{
		NewKernelModuleDriftCheck(collector),
		NewDeviceDriverBindingCheck(collector),
		NewRuntimeIntegrityCheck(collector),
		NewPackageStateCheck(collector),
		NewKernelErrorSignalsCheck(collector),
		NewCapabilityDriftCheck(collector),
	}
}

func baseDetails(inv hostinventory.HostInventory, evidence []map[string]any, recommendations []string) map[string]any {
	return map[string]any{
		"evidence":                evidence,
		"recommendations":         recommendations,
		"unsupportedCapabilities": inv.Unsupported,
		"kernel":                  inv.Kernel,
		"bootId":                  inv.BootID,
		"devices":                 inv.Devices,
		"runtimes":                inv.Runtimes,
		"packages":                inv.Packages,
		"secureBoot":              inv.SecureBoot,
		"resetReasons":            inv.ResetReasons,
		"crashEvidence":           inv.CrashEvidence,
	}
}

func okResult(message string, inv hostinventory.HostInventory) checks.Result {
	return checks.Result{
		Status:  checks.StatusOK,
		Message: message,
		Details: baseDetails(inv, nil, nil),
	}
}

func statusFromCounts(critical, warning int) checks.Status {
	if critical > 0 {
		return checks.StatusCritical
	}
	if warning > 0 {
		return checks.StatusWarning
	}
	return checks.StatusOK
}

func summarizeEvidence(prefix string, critical, warning int) string {
	switch {
	case critical > 0:
		return fmt.Sprintf("%s: %d critical finding(s), %d warning(s)", prefix, critical, warning)
	case warning > 0:
		return fmt.Sprintf("%s: %d warning(s)", prefix, warning)
	default:
		return prefix + ": no drift detected"
	}
}

func capabilityDevice(device hostinventory.DeviceInfo) bool {
	class := strings.ToLower(device.Class)
	name := strings.ToLower(device.DeviceName)
	return strings.Contains(class, "vga") ||
		strings.Contains(class, "3d") ||
		strings.Contains(class, "network") ||
		strings.Contains(class, "storage") ||
		strings.Contains(name, "accelerator") ||
		strings.Contains(name, "controller")
}
