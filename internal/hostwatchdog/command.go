package hostwatchdog

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
	cleanupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/storage-manager/v1/cleanup"
	cleanupconnect "github.com/vrooli/vrooli/packages/proto/gen/go/storage-manager/v1/cleanup/cleanup_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	defaultFloorBytes     = uint64(10 * 1024 * 1024 * 1024)
	defaultSustainSeconds = int64(120)
)

// CommandContext contains the small, testable boundary needed by the root CLI.
type CommandContext struct {
	Stdout io.Writer
	Stderr io.Writer
	JSON   bool
}

// Run dispatches the control-plane watchdog commands. It never shells out to
// storage-manager; pressure is sent through its typed Connect client.
func Run(ctx context.Context, commandCtx CommandContext, args []string) error {
	if len(args) == 0 || (len(args) == 1 && (args[0] == "--help" || args[0] == "-h")) {
		_, err := fmt.Fprintln(commandCtx.Stdout, "Usage: vrooli host-watchdog <tick|report-pressure> [options]")
		return err
	}
	switch args[0] {
	case "tick":
		return runTick(ctx, commandCtx, args[1:])
	case "report-pressure":
		return runReportPressure(ctx, commandCtx, args[1:])
	default:
		return fmt.Errorf("unknown host-watchdog command %q", args[0])
	}
}

func runTick(ctx context.Context, commandCtx CommandContext, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("host-watchdog tick accepts no arguments")
	}
	statePath := strings.TrimSpace(os.Getenv("VROOLI_WATCHDOG_STATE_PATH"))
	if statePath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("resolve home: %w", err)
		}
		statePath = filepath.Join(home, ".vrooli", "state", "host-watchdog", "state.json")
	}
	floor := defaultFloorBytes
	if raw := strings.TrimSpace(os.Getenv("VROOLI_WATCHDOG_FLOOR_BYTES")); raw != "" {
		value, err := strconv.ParseUint(raw, 10, 64)
		if err != nil || value == 0 {
			return fmt.Errorf("invalid VROOLI_WATCHDOG_FLOOR_BYTES %q", raw)
		}
		floor = value
	}
	report, err := Tick(ctx, Config{
		Mount:      "/",
		FloorBytes: floor,
		Sustain:    timeDurationFromEnv(),
		StatePath:  statePath,
		ReportPressure: func(ctx context.Context, report Report) error {
			_, err := sendPressure(ctx, report, "floor")
			return err
		},
	})
	if err != nil {
		return err
	}
	return writeReport(commandCtx, report)
}

func timeDurationFromEnv() time.Duration {
	seconds := defaultSustainSeconds
	if raw := strings.TrimSpace(os.Getenv("VROOLI_WATCHDOG_FLOOR_SUSTAIN_SECONDS")); raw != "" {
		if value, err := strconv.ParseInt(raw, 10, 64); err == nil && value > 0 {
			seconds = value
		}
	}
	return time.Duration(seconds) * time.Second
}

func runReportPressure(ctx context.Context, commandCtx CommandContext, args []string) error {
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
	report := Report{Mount: partition, UsedPercent: used, AvailableBytes: available}
	response, err := sendPressure(ctx, report, band, trigger)
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

func sendPressure(ctx context.Context, report Report, parts ...string) (*cleanupv1.ReportPressureResponse, error) {
	band, trigger := "", "floor"
	if len(parts) == 1 {
		trigger = parts[0]
	}
	if len(parts) > 1 {
		band, trigger = parts[0], parts[1]
	}
	if band == "" {
		// A floor signal is an absolute trigger. The receiver re-checks the
		// available bytes and applies the shared classifier; critical is the
		// conservative wire value for this bypass path.
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

func writeReport(commandCtx CommandContext, report Report) error {
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
