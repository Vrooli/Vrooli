package privilegebroker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const smartProbeTimeout = 30 * time.Second

const maxSMARTAttributes = 32

type smartctlResponse struct {
	JSONFormatVersion []int `json:"json_format_version"`
	Smartctl          struct {
		Version    []int `json:"version"`
		ExitStatus *int  `json:"exit_status"`
	} `json:"smartctl"`
	LocalTime json.RawMessage `json:"local_time"`
	Device    struct {
		Name         string `json:"name"`
		InfoName     string `json:"info_name"`
		ModelName    string `json:"model_name"`
		SerialNumber string `json:"serial_number"`
		Type         string `json:"type"`
		Protocol     string `json:"protocol"`
	} `json:"device"`
	SmartStatus struct {
		Passed *bool `json:"passed"`
	} `json:"smart_status"`
	Temperature struct {
		Current int `json:"current"`
	} `json:"temperature"`
	ATAAttributes struct {
		Table []struct {
			ID         int    `json:"id"`
			Name       string `json:"name"`
			Value      int    `json:"value"`
			Worst      int    `json:"worst"`
			Threshold  int    `json:"thresh"`
			WhenFailed string `json:"when_failed"`
			Raw        struct {
				String string `json:"string"`
			} `json:"raw"`
		} `json:"table"`
	} `json:"ata_smart_attributes"`
	ATAErrorLog struct {
		Summary json.RawMessage `json:"summary"`
	} `json:"ata_smart_error_log"`
	ATASelfTestLog struct {
		Standard json.RawMessage `json:"standard"`
	} `json:"ata_smart_self_test_log"`
}

type smartAttributeSummary struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Value      int    `json:"value"`
	Worst      int    `json:"worst"`
	Threshold  int    `json:"threshold"`
	WhenFailed string `json:"when_failed,omitempty"`
	Raw        string `json:"raw"`
}

type smartEvidenceSummary struct {
	LocalTime          string                  `json:"local_time,omitempty"`
	Device             smartDeviceSummary      `json:"device"`
	ExpectedSerial     string                  `json:"expected_serial"`
	Health             string                  `json:"health"`
	TemperatureC       *int                    `json:"temperature_c,omitempty"`
	ExitStatus         int                     `json:"smartctl_exit_status"`
	Attributes         []smartAttributeSummary `json:"ata_smart_attributes"`
	ErrorLogSummary    json.RawMessage         `json:"ata_smart_error_log_summary,omitempty"`
	SelfTestLogSummary json.RawMessage         `json:"ata_smart_self_test_log_summary,omitempty"`
}

type smartDeviceSummary struct {
	Name     string `json:"name,omitempty"`
	Model    string `json:"model,omitempty"`
	Type     string `json:"type,omitempty"`
	Protocol string `json:"protocol,omitempty"`
}

// executeSMART verifies the whole-disk serial again inside the broker, then
// invokes smartctl with its fixed read-only health/attribute arguments.
func executeSMART(ctx context.Context, executor Executor, req Request) Result {
	fail := func(code string) Result { return NewFailure(req.RequestID, req.Action, code) }
	device := strings.TrimSpace(req.SMART.Device)

	identity, err := executor.Run(ctx, "lsblk", "-dnro", "SERIAL,TYPE", "--", device)
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return fail("device_inventory_unavailable")
		}
		return fail("device_identity_unavailable")
	}
	fields := strings.Fields(string(identity))
	if len(fields) != 2 || fields[0] != strings.TrimSpace(req.SMART.Serial) || fields[1] != "disk" {
		return fail("device_identity_mismatch")
	}

	tool, args, err := SMARTArgs(req)
	if err != nil {
		return fail("action_not_allowed")
	}
	out, runErr := executor.Run(ctx, tool, args...)
	if runErr != nil {
		var exitErr *exec.ExitError
		if !errors.As(runErr, &exitErr) {
			if errors.Is(runErr, exec.ErrNotFound) {
				return fail("smartctl_unavailable")
			}
			return fail("smartctl_failed")
		}
	}
	detail, err := summarizeSMART(out, device, strings.TrimSpace(req.SMART.Serial))
	if err != nil {
		return Result{
			Version: ProtocolVersion, RequestID: req.RequestID, Action: req.Action,
			Status: "failed", Code: "smartctl_output_invalid",
			Evidence: Evidence{
				Available: true, IdentityVerified: true, ExitCode: volumeExitCode(runErr),
				Detail: boundedDetail([]byte(err.Error())),
			},
		}
	}

	return Result{
		Version: ProtocolVersion, RequestID: req.RequestID, Action: req.Action,
		Status: "verified",
		Evidence: Evidence{
			Available: true, IdentityVerified: true,
			ExitCode: volumeExitCode(runErr),
			Detail:   detail,
		},
	}
}

// summarizeSMART preserves the useful health and attribute readings as valid,
// bounded JSON. Returning a prefix of smartctl's raw JSON would produce
// unparseable evidence and could silently drop the attributes needed to assess
// the device.
func summarizeSMART(out []byte, expectedDevice, expectedSerial string) (string, error) {
	var response smartctlResponse
	if err := json.Unmarshal(out, &response); err != nil {
		return "", err
	}
	if len(response.JSONFormatVersion) == 0 || len(response.Smartctl.Version) == 0 || response.Smartctl.ExitStatus == nil {
		return "", errors.New("SMART output is missing its format or tool identity")
	}
	if strings.TrimSpace(response.Device.Name) != strings.TrimSpace(expectedDevice) {
		return "", errors.New("SMART output device does not match the requested device")
	}
	attrs := response.ATAAttributes.Table
	if len(attrs) > maxSMARTAttributes {
		attrs = attrs[:maxSMARTAttributes]
	}
	summary := smartEvidenceSummary{
		LocalTime: smartctlLocalTime(response.LocalTime),
		Device: smartDeviceSummary{
			Name: response.Device.Name, Model: response.Device.ModelName,
			Type: response.Device.Type, Protocol: response.Device.Protocol,
		},
		ExpectedSerial: expectedSerial, Health: "unknown", ExitStatus: *response.Smartctl.ExitStatus,
		Attributes:         make([]smartAttributeSummary, 0, len(attrs)),
		ErrorLogSummary:    response.ATAErrorLog.Summary,
		SelfTestLogSummary: response.ATASelfTestLog.Standard,
	}
	if response.SmartStatus.Passed != nil {
		if *response.SmartStatus.Passed {
			summary.Health = "passed"
		} else {
			summary.Health = "failed"
		}
	}
	if response.Temperature.Current != 0 {
		temperature := response.Temperature.Current
		summary.TemperatureC = &temperature
	}
	for _, attr := range attrs {
		summary.Attributes = append(summary.Attributes, smartAttributeSummary{
			ID: attr.ID, Name: attr.Name, Value: attr.Value, Worst: attr.Worst,
			Threshold: attr.Threshold, WhenFailed: attr.WhenFailed, Raw: attr.Raw.String,
		})
	}
	for {
		encoded, err := json.Marshal(summary)
		if err != nil {
			return "", fmt.Errorf("marshal SMART evidence summary: %w", err)
		}
		if len(encoded) <= 8192 {
			return string(encoded), nil
		}
		if len(summary.SelfTestLogSummary) > 0 {
			summary.SelfTestLogSummary = nil
			continue
		}
		if len(summary.ErrorLogSummary) > 0 {
			summary.ErrorLogSummary = nil
			continue
		}
		if len(summary.Attributes) > 0 {
			summary.Attributes = summary.Attributes[:len(summary.Attributes)-1]
			continue
		}
		return "", errors.New("SMART evidence summary exceeds broker limit")
	}
	return "", errors.New("SMART evidence summary could not be bounded")
}

func smartctlLocalTime(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var object struct {
		Asctime string `json:"asctime"`
	}
	if err := json.Unmarshal(raw, &object); err == nil && object.Asctime != "" {
		return object.Asctime
	}
	var value string
	if err := json.Unmarshal(raw, &value); err == nil {
		return value
	}
	return ""
}
