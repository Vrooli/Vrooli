package devicegraph

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// smartToolName is the command the SMART reader is declared under in
// internal/tools/smartctl/tool.json. Nothing here installs it; the tool
// manifest is what `vrooli setup` acts on.
const smartToolName = "smartctl"

const smartProbeTimeout = 10 * time.Second

// smartReading is the outcome of one SMART interrogation. Blocked is the case
// that matters most: the tool ran, the device exists, and the host refused the
// read. That is reported as unmeasurable with Reason, never as zero errors and
// never as a healthy drive.
type smartReading struct {
	// ToolPresent is false when no SMART reader is installed at all.
	ToolPresent bool
	// Blocked is true when the reader could not open the device.
	Blocked bool
	// PermissionDenied narrows Blocked to the access-control case, which is
	// the one a commissioning-time capability grant can fix.
	PermissionDenied bool
	// Reason explains ToolPresent=false or Blocked=true. Empty on success.
	Reason string
	// Protocol is the SMART dialect the device answered in.
	Protocol string
	// Mechanism names the exact probe used, for provenance.
	Mechanism string

	HealthPassed         *bool
	PowerOnHours         *int64
	TemperatureCelsius   *int64
	ReallocatedSectors   *int64
	PendingSectors       *int64
	UncorrectableSectors *int64
	// WearPercentUsed is the fraction of rated endurance consumed, 0..100+.
	WearPercentUsed *int64
	MediaErrors     *int64
	UnsafeShutdowns *int64
	CriticalWarning *int64
	AvailableSpare  *int64
}

// hasAttributes reports whether the device answered with any usable attribute.
func (r smartReading) hasAttributes() bool {
	return r.PowerOnHours != nil || r.ReallocatedSectors != nil || r.PendingSectors != nil ||
		r.UncorrectableSectors != nil || r.WearPercentUsed != nil || r.MediaErrors != nil ||
		r.UnsafeShutdowns != nil || r.CriticalWarning != nil || r.HealthPassed != nil
}

// readSMART interrogates one device node. It never returns an error: a SMART
// read that could not happen is a graded state, not a failure of collection.
func readSMART(ctx context.Context, env Env, devicePath string) smartReading {
	binary, err := env.LookPath(smartToolName)
	if err != nil {
		return smartReading{
			Reason:    fmt.Sprintf("no SMART reader on PATH: %s is not installed", smartToolName),
			Mechanism: smartToolName + " -j -H -A",
		}
	}
	output, runErr := env.Run(ctx, smartProbeTimeout, binary, "-j", "-H", "-A", devicePath)
	reading := parseSMART(output)
	reading.ToolPresent = true
	reading.Mechanism = smartToolName + " -j -H -A " + devicePath
	if !reading.Blocked && !reading.hasAttributes() {
		reading.Blocked = true
		if reading.Reason == "" {
			if runErr != nil {
				reading.Reason = fmt.Sprintf("%s produced no SMART attributes: %v", smartToolName, runErr)
			} else {
				reading.Reason = fmt.Sprintf("%s produced no SMART attributes for this device", smartToolName)
			}
		}
	}
	return reading
}

type smartctlDocument struct {
	Smartctl struct {
		ExitStatus int `json:"exit_status"`
		Messages   []struct {
			String   string `json:"string"`
			Severity string `json:"severity"`
		} `json:"messages"`
	} `json:"smartctl"`
	Device struct {
		Name     string `json:"name"`
		Type     string `json:"type"`
		Protocol string `json:"protocol"`
	} `json:"device"`
	SmartStatus *struct {
		Passed bool `json:"passed"`
	} `json:"smart_status"`
	PowerOnTime *struct {
		Hours int64 `json:"hours"`
	} `json:"power_on_time"`
	Temperature *struct {
		Current int64 `json:"current"`
	} `json:"temperature"`
	NVMeLog *struct {
		CriticalWarning int64 `json:"critical_warning"`
		Temperature     int64 `json:"temperature"`
		AvailableSpare  int64 `json:"available_spare"`
		PercentageUsed  int64 `json:"percentage_used"`
		MediaErrors     int64 `json:"media_errors"`
		UnsafeShutdowns int64 `json:"unsafe_shutdowns"`
		PowerOnHours    int64 `json:"power_on_hours"`
	} `json:"nvme_smart_health_information_log"`
	ATAAttributes *struct {
		Table []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
			Raw  struct {
				Value int64 `json:"value"`
			} `json:"raw"`
			Value int `json:"value"`
		} `json:"table"`
	} `json:"ata_smart_attributes"`
}

// ATA SMART attribute ids the graph reads. These are the SMART specification's
// own identifiers, not a per-vendor or per-machine assumption.
const (
	ataAttrReallocatedSectorCount = 5
	ataAttrPowerOnHours           = 9
	ataAttrWearLevelingCount      = 177
	ataAttrCurrentPendingSector   = 197
	ataAttrOfflineUncorrectable   = 198
	ataAttrSSDLifeLeft            = 231
	ataAttrMediaWearoutIndicator  = 233
)

// parseSMART reads smartctl's JSON output. It handles both the NVMe health-log
// shape and the ATA attribute-table shape, and it treats a non-zero exit status
// with an open failure as a blocked read rather than an empty result.
func parseSMART(output []byte) smartReading {
	reading := smartReading{}
	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		reading.Blocked = true
		reading.Reason = "SMART reader produced no output"
		return reading
	}
	var document smartctlDocument
	if err := json.Unmarshal([]byte(trimmed), &document); err != nil {
		reading.Blocked = true
		reading.Reason = fmt.Sprintf("SMART reader output was not JSON: %v", err)
		return reading
	}
	reading.Protocol = strings.ToLower(strings.TrimSpace(document.Device.Protocol))

	messages := make([]string, 0, len(document.Smartctl.Messages))
	for _, message := range document.Smartctl.Messages {
		if text := strings.TrimSpace(message.String); text != "" {
			messages = append(messages, text)
		}
	}
	joined := strings.Join(messages, "; ")
	lowered := strings.ToLower(joined)

	switch {
	case strings.Contains(lowered, "permission denied"):
		reading.Blocked = true
		reading.PermissionDenied = true
		reading.Reason = "permission denied: " + joined
		return reading
	case strings.Contains(lowered, "operation not permitted"):
		reading.Blocked = true
		reading.PermissionDenied = true
		reading.Reason = "operation not permitted: " + joined
		return reading
	// smartctl exit-status bit 1 (value 2) means the device could not be
	// opened at all, which is a blocked read no matter what the text says.
	case document.Smartctl.ExitStatus&0x02 != 0:
		reading.Blocked = true
		reading.Reason = "SMART device open failed"
		if joined != "" {
			reading.Reason += ": " + joined
		}
		return reading
	// Bit 0 (value 1) means smartctl rejected the command line.
	case document.Smartctl.ExitStatus&0x01 != 0:
		reading.Blocked = true
		reading.Reason = "SMART reader rejected the request"
		if joined != "" {
			reading.Reason += ": " + joined
		}
		return reading
	}

	if document.SmartStatus != nil {
		passed := document.SmartStatus.Passed
		reading.HealthPassed = &passed
	}
	if document.PowerOnTime != nil {
		reading.PowerOnHours = int64Pointer(document.PowerOnTime.Hours)
	}
	if document.Temperature != nil && document.Temperature.Current != 0 {
		reading.TemperatureCelsius = int64Pointer(document.Temperature.Current)
	}

	if log := document.NVMeLog; log != nil {
		if reading.Protocol == "" {
			reading.Protocol = "nvme"
		}
		reading.CriticalWarning = int64Pointer(log.CriticalWarning)
		reading.WearPercentUsed = int64Pointer(log.PercentageUsed)
		reading.MediaErrors = int64Pointer(log.MediaErrors)
		reading.UnsafeShutdowns = int64Pointer(log.UnsafeShutdowns)
		reading.AvailableSpare = int64Pointer(log.AvailableSpare)
		if log.PowerOnHours != 0 || reading.PowerOnHours == nil {
			reading.PowerOnHours = int64Pointer(log.PowerOnHours)
		}
	}

	if attributes := document.ATAAttributes; attributes != nil && len(attributes.Table) > 0 {
		if reading.Protocol == "" {
			reading.Protocol = "ata"
		}
		for _, row := range attributes.Table {
			switch row.ID {
			case ataAttrReallocatedSectorCount:
				reading.ReallocatedSectors = int64Pointer(row.Raw.Value)
			case ataAttrPowerOnHours:
				reading.PowerOnHours = int64Pointer(row.Raw.Value)
			case ataAttrCurrentPendingSector:
				reading.PendingSectors = int64Pointer(row.Raw.Value)
			case ataAttrOfflineUncorrectable:
				reading.UncorrectableSectors = int64Pointer(row.Raw.Value)
			case ataAttrWearLevelingCount, ataAttrSSDLifeLeft, ataAttrMediaWearoutIndicator:
				// These attributes normalize to "life remaining"; the graph
				// reports consumed endurance so NVMe and ATA agree on units.
				if reading.WearPercentUsed == nil && row.Value >= 0 && row.Value <= 100 {
					reading.WearPercentUsed = int64Pointer(int64(100 - row.Value))
				}
			}
		}
	}

	return reading
}

func int64Pointer(value int64) *int64 {
	copied := value
	return &copied
}
