package vroolicli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/vrooli/internal/config"
	cleanupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/storage-manager/v1/cleanup"
	cleanupconnect "github.com/vrooli/vrooli/packages/proto/gen/go/storage-manager/v1/cleanup/cleanup_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	hostWatchdogDefaultFloorBytes     = uint64(10 * 1024 * 1024 * 1024)
	hostWatchdogDefaultSustainSeconds = int64(120)
)

type hostWatchdogCommandContext struct {
	Stdout io.Writer
	Stderr io.Writer
	JSON   bool
}

type hostWatchdogConfig struct {
	Mount          string
	FloorBytes     uint64
	Sustain        time.Duration
	StatePath      string
	Now            func() time.Time
	FreeSpace      func(string) (uint64, float64, error)
	ReportPressure func(context.Context, hostWatchdogReport) error
}

type hostWatchdogReport struct {
	Mount          string  `json:"mount"`
	AvailableBytes uint64  `json:"available_bytes"`
	UsedPercent    float64 `json:"used_percent"`
	BelowFloor     bool    `json:"below_floor"`
	Sustained      bool    `json:"sustained"`
}

type hostWatchdogState struct {
	FirstBelow time.Time `json:"first_below"`
}

func runHostWatchdog(ctx context.Context, commandCtx hostWatchdogCommandContext, args []string) error {
	if len(args) == 0 || (len(args) == 1 && (args[0] == "--help" || args[0] == "-h")) {
		_, err := fmt.Fprintln(commandCtx.Stdout, "Usage: vrooli host-watchdog <tick|report-pressure> [options]")
		return err
	}
	switch args[0] {
	case "tick":
		return runHostWatchdogTick(ctx, commandCtx, args[1:])
	case "report-pressure":
		return runHostWatchdogReportPressure(ctx, commandCtx, args[1:])
	default:
		return fmt.Errorf("unknown host-watchdog command %q", args[0])
	}
}

func runHostWatchdogTick(ctx context.Context, commandCtx hostWatchdogCommandContext, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("host-watchdog tick accepts no arguments")
	}
	statePath := strings.TrimSpace(os.Getenv("VROOLI_WATCHDOG_STATE_PATH"))
	if statePath == "" {
		home, err := config.HomeDir()
		if err != nil {
			return fmt.Errorf("resolve home: %w", err)
		}
		statePath = filepath.Join(home, ".vrooli", "state", "host-watchdog", "state.json")
	}
	floor := hostWatchdogDefaultFloorBytes
	if raw := strings.TrimSpace(os.Getenv("VROOLI_WATCHDOG_FLOOR_BYTES")); raw != "" {
		value, err := strconv.ParseUint(raw, 10, 64)
		if err != nil || value == 0 {
			return fmt.Errorf("invalid VROOLI_WATCHDOG_FLOOR_BYTES %q", raw)
		}
		floor = value
	}
	report, err := tickHostWatchdog(ctx, hostWatchdogConfig{
		Mount:      "/",
		FloorBytes: floor,
		Sustain:    hostWatchdogTimeDurationFromEnv(),
		StatePath:  statePath,
		ReportPressure: func(ctx context.Context, report hostWatchdogReport) error {
			_, err := sendHostWatchdogPressure(ctx, report, "floor")
			return err
		},
	})
	if err != nil {
		return err
	}
	return writeHostWatchdogReport(commandCtx, report)
}

func hostWatchdogTimeDurationFromEnv() time.Duration {
	seconds := hostWatchdogDefaultSustainSeconds
	if raw := strings.TrimSpace(os.Getenv("VROOLI_WATCHDOG_FLOOR_SUSTAIN_SECONDS")); raw != "" {
		if value, err := strconv.ParseInt(raw, 10, 64); err == nil && value > 0 {
			seconds = value
		}
	}
	return time.Duration(seconds) * time.Second
}

func runHostWatchdogReportPressure(ctx context.Context, commandCtx hostWatchdogCommandContext, args []string) error {
	values := map[string]string{}
	for i := 0; i < len(args); i++ {
		if !strings.HasPrefix(args[i], "--") || i+1 >= len(args) {
			return fmt.Errorf("expected --name value, got %q", args[i])
		}
		values[strings.TrimPrefix(args[i], "--")] = args[i+1]
		i++
	}
	partition := values["partition"]
	if partition == "" {
		partition = "/"
	}
	band := strings.ToLower(values["band"])
	used, err := strconv.ParseFloat(values["used-percent"], 64)
	if err != nil {
		return fmt.Errorf("invalid --used-percent: %w", err)
	}
	available, err := strconv.ParseUint(values["available-bytes"], 10, 63)
	if err != nil {
		return fmt.Errorf("invalid --available-bytes: %w", err)
	}
	trigger := values["trigger"]
	if trigger == "" {
		trigger = "floor"
	}
	report := hostWatchdogReport{Mount: partition, UsedPercent: used, AvailableBytes: available}
	response, err := sendHostWatchdogPressure(ctx, report, band, trigger)
	if err != nil {
		return err
	}
	if commandCtx.JSON {
		data, marshalErr := protojson.MarshalOptions{Indent: "  "}.Marshal(response)
		if marshalErr != nil {
			return marshalErr
		}
		_, writeErr := fmt.Fprintln(commandCtx.Stdout, string(data))
		return writeErr
	}
	_, err = fmt.Fprintln(commandCtx.Stdout, "host-watchdog pressure reported")
	return err
}

func sendHostWatchdogPressure(ctx context.Context, report hostWatchdogReport, parts ...string) (*cleanupv1.ReportPressureResponse, error) {
	band, trigger := "", "floor"
	if len(parts) == 1 {
		trigger = parts[0]
	}
	if len(parts) > 1 {
		band, trigger = parts[0], parts[1]
	}
	if band == "" {
		band = "critical"
	}
	bandValue := map[string]cleanupv1.PressureBand{"warning": cleanupv1.PressureBand_PRESSURE_BAND_WARNING, "high": cleanupv1.PressureBand_PRESSURE_BAND_HIGH, "critical": cleanupv1.PressureBand_PRESSURE_BAND_CRITICAL}[band]
	if bandValue == 0 {
		return nil, fmt.Errorf("unknown pressure band %q", band)
	}
	triggerValue := map[string]cleanupv1.PressureTrigger{"band": cleanupv1.PressureTrigger_PRESSURE_TRIGGER_BAND, "floor": cleanupv1.PressureTrigger_PRESSURE_TRIGGER_FLOOR, "rate": cleanupv1.PressureTrigger_PRESSURE_TRIGGER_RATE, "manual": cleanupv1.PressureTrigger_PRESSURE_TRIGGER_MANUAL}[trigger]
	if triggerValue == 0 {
		return nil, fmt.Errorf("unknown pressure trigger %q", trigger)
	}
	base, err := discovery.ResolveScenarioURLDefault(ctx, "storage-manager")
	if err != nil {
		return nil, fmt.Errorf("resolve storage-manager: %w", err)
	}
	client := cleanupconnect.NewCleanupServiceClient(http.DefaultClient, base)
	response, err := client.ReportPressure(ctx, connect.NewRequest(&cleanupv1.ReportPressureRequest{
		SourceScenario: "host-watchdog", Partition: report.Mount, UsedPercent: report.UsedPercent,
		Band: bandValue, AvailableBytes: int64(report.AvailableBytes), Trigger: triggerValue,
	}))
	if err != nil {
		return nil, fmt.Errorf("report pressure: %w", err)
	}
	return response.Msg, nil
}

func writeHostWatchdogReport(commandCtx hostWatchdogCommandContext, report hostWatchdogReport) error {
	if commandCtx.JSON {
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(commandCtx.Stdout, string(data))
		return err
	}
	_, err := fmt.Fprintf(commandCtx.Stdout, "mount=%s available_bytes=%d used_percent=%.2f below_floor=%t sustained=%t\n", report.Mount, report.AvailableBytes, report.UsedPercent, report.BelowFloor, report.Sustained)
	return err
}

func tickHostWatchdog(ctx context.Context, cfg hostWatchdogConfig) (hostWatchdogReport, error) {
	if strings.TrimSpace(cfg.Mount) == "" {
		cfg.Mount = "/"
	}
	if cfg.FloorBytes == 0 {
		return hostWatchdogReport{}, fmt.Errorf("floor bytes must be positive")
	}
	if cfg.Sustain <= 0 {
		cfg.Sustain = 120 * time.Second
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	spaceReader := cfg.FreeSpace
	if spaceReader == nil {
		spaceReader = freeHostWatchdogSpace
	}
	available, used, err := spaceReader(cfg.Mount)
	if err != nil {
		return hostWatchdogReport{}, err
	}
	report := hostWatchdogReport{Mount: cfg.Mount, AvailableBytes: available, UsedPercent: used, BelowFloor: available < cfg.FloorBytes}
	var state hostWatchdogState
	if cfg.StatePath != "" {
		if raw, readErr := os.ReadFile(cfg.StatePath); readErr == nil {
			_ = json.Unmarshal(raw, &state)
		}
	}
	if report.BelowFloor {
		if state.FirstBelow.IsZero() {
			state.FirstBelow = cfg.Now().UTC()
		}
		report.Sustained = !cfg.Now().UTC().Before(state.FirstBelow.Add(cfg.Sustain))
	} else {
		state.FirstBelow = time.Time{}
	}
	if cfg.StatePath != "" {
		if err := os.MkdirAll(filepath.Dir(cfg.StatePath), 0o700); err != nil {
			return report, err
		}
		data, _ := json.Marshal(state)
		if err := os.WriteFile(cfg.StatePath, data, 0o600); err != nil {
			return report, err
		}
	}
	if report.Sustained && cfg.ReportPressure != nil {
		if err := cfg.ReportPressure(ctx, report); err != nil {
			return report, err
		}
	}
	return report, nil
}
