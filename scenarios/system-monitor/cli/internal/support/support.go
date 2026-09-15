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

	metricspb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/metrics"
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

type Alert struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

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

func DeriveAlerts(metrics *metricspb.MetricsResponse, settings *settingspb.SystemSettings) []Alert {
	if metrics == nil {
		return []Alert{{Severity: "critical", Message: "No metrics snapshot is available."}}
	}

	cpuThreshold, memoryThreshold, _ := MetricThresholds(settings)
	alerts := make([]Alert, 0, 4)
	if metrics.GetCpuUsage() >= cpuThreshold+10 {
		alerts = append(alerts, Alert{Severity: "critical", Message: fmt.Sprintf("CPU usage is %s against a %.1f%% threshold.", FormatPercent(metrics.GetCpuUsage()), cpuThreshold)})
	} else if metrics.GetCpuUsage() >= cpuThreshold {
		alerts = append(alerts, Alert{Severity: "warning", Message: fmt.Sprintf("CPU usage is %s against a %.1f%% threshold.", FormatPercent(metrics.GetCpuUsage()), cpuThreshold)})
	}
	if metrics.GetMemoryUsage() >= memoryThreshold+5 {
		alerts = append(alerts, Alert{Severity: "critical", Message: fmt.Sprintf("Memory usage is %s against a %.1f%% threshold.", FormatPercent(metrics.GetMemoryUsage()), memoryThreshold)})
	} else if metrics.GetMemoryUsage() >= memoryThreshold {
		alerts = append(alerts, Alert{Severity: "warning", Message: fmt.Sprintf("Memory usage is %s against a %.1f%% threshold.", FormatPercent(metrics.GetMemoryUsage()), memoryThreshold)})
	}
	if metrics.GetTcpConnections() >= 300 {
		alerts = append(alerts, Alert{Severity: "warning", Message: fmt.Sprintf("TCP connections are elevated at %d.", metrics.GetTcpConnections())})
	}
	if metrics.GpuUsage != nil && metrics.GetGpuUsage() >= 90 {
		alerts = append(alerts, Alert{Severity: "warning", Message: fmt.Sprintf("GPU usage is %s.", FormatPercent(metrics.GetGpuUsage()))})
	}
	return alerts
}

func OverallStatus(metrics *metricspb.MetricsResponse, settings *settingspb.SystemSettings, maintenance string) string {
	maintenance = strings.ToLower(strings.TrimSpace(maintenance))
	if maintenance == "active" {
		return "MAINTENANCE"
	}
	if settings != nil && !settings.GetActive() {
		return "INACTIVE"
	}
	alerts := DeriveAlerts(metrics, settings)
	for _, alert := range alerts {
		switch strings.ToLower(strings.TrimSpace(alert.Severity)) {
		case "critical":
			return "CRITICAL"
		case "warning":
			return "WARNING"
		}
	}
	return "HEALTHY"
}

func AlertLines(alerts []Alert) []string {
	if len(alerts) == 0 {
		return []string{"No active alerts from the current snapshot."}
	}
	lines := make([]string, 0, len(alerts))
	for _, alert := range alerts {
		lines = append(lines, fmt.Sprintf("[%s] %s", strings.ToUpper(alert.Severity), alert.Message))
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
