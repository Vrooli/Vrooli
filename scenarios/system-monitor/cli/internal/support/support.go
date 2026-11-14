package support

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os/exec"
	"strings"

	alertspb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/alerts"
	settingspb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/settings"

	"github.com/vrooli/cli-core/cliutil"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	DefaultCPUThreshold    = 85.0
	DefaultMemoryThreshold = 90.0
	DefaultDiskThreshold   = 85.0
	DefaultUIPort          = "36232"
)

type DashboardResult struct {
	URL    string `json:"url"`
	Opened bool   `json:"opened"`
}

func NewFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

func ParseFlags(fs *flag.FlagSet, args []string) error {
	return cliutil.ParseInterspersed(fs, args)
}

func PrettyPrintJSON(data []byte) error {
	var out bytes.Buffer
	if err := json.Indent(&out, data, "", "  "); err == nil {
		_, err = fmt.Println(out.String())
		return err
	}
	_, err := fmt.Println(string(data))
	return err
}

func DecodeProto(data []byte, msg proto.Message) error {
	return protojson.UnmarshalOptions{DiscardUnknown: false}.Unmarshal(data, msg)
}

func DecodeJSON(data []byte, dest interface{}) error {
	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}
	return nil
}

func FormatTimestamp(ts *timestamppb.Timestamp) string {
	if ts == nil {
		return "unknown"
	}
	return ts.AsTime().Format("2006-01-02 15:04:05 MST")
}

func FormatPercent(value float64) string {
	return fmt.Sprintf("%.1f%%", value)
}

func FormatMaybePercent(value *float64) string {
	if value == nil {
		return "n/a"
	}
	return FormatPercent(*value)
}

func FormatMaybeString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func ResolveUIPort() string {
	if port := strings.TrimSpace(cliutil.DetectPortFromVrooli("system-monitor", "UI_PORT")()); port != "" {
		return port
	}
	return DefaultUIPort
}

func DashboardURL() string {
	return fmt.Sprintf("http://localhost:%s", ResolveUIPort())
}

func OpenBrowser(target string) (bool, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return false, fmt.Errorf("url is required")
	}

	candidates := [][]string{
		{"xdg-open", target},
		{"open", target},
		{"cmd", "/c", "start", target},
	}
	for _, candidate := range candidates {
		if _, err := exec.LookPath(candidate[0]); err != nil {
			continue
		}
		cmd := exec.Command(candidate[0], candidate[1:]...)
		if err := cmd.Start(); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func MetricThresholds(settings *settingspb.SystemSettings) (cpu float64, memory float64, disk float64) {
	cpu = DefaultCPUThreshold
	memory = DefaultMemoryThreshold
	disk = DefaultDiskThreshold
	if settings == nil {
		return cpu, memory, disk
	}
	if settings.GetCpuThreshold() > 0 {
		cpu = settings.GetCpuThreshold()
	}
	if settings.GetMemoryThreshold() > 0 {
		memory = settings.GetMemoryThreshold()
	}
	if settings.GetDiskThreshold() > 0 {
		disk = settings.GetDiskThreshold()
	}
	return cpu, memory, disk
}

// MonitoringActive reports whether monitoring is running. The
// maintenance_state wire value is the platform maintenance-orchestrator
// contract: "active" means monitoring is running normally and "inactive"
// means it has been paused. It never means "in maintenance".
func MonitoringActive(settings *settingspb.SystemSettings, maintenanceState string) bool {
	if settings != nil && !settings.GetActive() {
		return false
	}
	return !strings.EqualFold(strings.TrimSpace(maintenanceState), "inactive")
}

// MonitoringStateLabel renders the maintenance-orchestrator state in words
// that cannot be misread as "under maintenance".
func MonitoringStateLabel(active bool) string {
	if active {
		return "running"
	}
	return "paused"
}

// OverallStatus summarizes server-owned alert state. A paused monitor is
// INACTIVE (its alert list is not a statement about the host); a monitor
// whose evaluator has not completed a pass is UNKNOWN; otherwise the most
// severe active alert decides.
func OverallStatus(monitoringActive bool, alerts *alertspb.ListAlertsResponse) string {
	if !monitoringActive {
		return "INACTIVE"
	}
	if alerts == nil || alerts.GetEvaluatedAt() == nil {
		return "UNKNOWN"
	}
	status := "HEALTHY"
	for _, alert := range alerts.GetAlerts() {
		if alert.GetResolvedAt() != nil {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(alert.GetSeverity())) {
		case "critical":
			return "CRITICAL"
		case "warning", "high", "medium":
			status = "WARNING"
		}
	}
	return status
}

// AlertLines renders server alerts for terminal output.
func AlertLines(alerts []*alertspb.Alert) []string {
	if len(alerts) == 0 {
		return []string{"No active alerts."}
	}
	lines := make([]string, 0, len(alerts))
	for _, alert := range alerts {
		line := fmt.Sprintf("[%s] %s", strings.ToUpper(alert.GetSeverity()), alert.GetMessage())
		if rule := alert.GetRuleId(); rule != "" {
			line = fmt.Sprintf("[%s] %s: %s", strings.ToUpper(alert.GetSeverity()), rule, alert.GetMessage())
		}
		if opened := alert.GetOpenedAt(); opened != nil {
			line += " (since " + FormatTimestamp(opened) + ")"
		}
		if alert.GetEscalated() {
			line += " [escalated: " + alert.GetAnomalyId() + "]"
		}
		lines = append(lines, line)
	}
	return lines
}

// RuleLines reports rules that could not evaluate, so an unsupported rule is
// visible rather than silently counted as healthy.
func RuleLines(rules []*alertspb.RuleState) []string {
	lines := make([]string, 0)
	for _, rule := range rules {
		if rule.GetStatus() == "supported" {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s: %s (%s)", rule.GetRuleId(), rule.GetStatus(), rule.GetReason()))
	}
	return lines
}

func BoolString(value bool, yes string, no string) string {
	if value {
		return yes
	}
	return no
}

func ParseOptionalBool(value string) (*bool, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	switch strings.ToLower(value) {
	case "true", "1", "yes", "on":
		v := true
		return &v, nil
	case "false", "0", "no", "off":
		v := false
		return &v, nil
	default:
		return nil, errors.New("must be one of true,false,yes,no,1,0")
	}
}
