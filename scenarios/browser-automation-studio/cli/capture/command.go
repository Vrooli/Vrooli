// Package capture is the CLI surface for the BAS CaptureService
// Connect-RPC method. One verb, composable flags — agents wrap it with
// fixed flag combinations as prompt-manager actions.
//
// The handler builds a typed CaptureRequest from flags and dispatches
// through cli-core's Connect HTTP client. Output mirrors the wire shape
// when --json is set; otherwise it renders a Mutation Contract report.
package capture

import (
	"browser-automation-studio/cli/internal/appctx"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"connectrpc.com/connect"

	capturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture"
	captureconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture/captureconnect"

	"github.com/vrooli/cli-core/cliapp"
)

// Commands returns the `capture` CommandGroup. The shape mirrors the
// existing workflow/recordings/etc. domains so registration in
// cli/domains/domains.go is one line.
func Commands(ctx *appctx.Context) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Capture",
		Commands: []cliapp.Command{
			{
				Name:        "capture",
				NeedsAPI:    true,
				Description: "Capture screenshot/console/network/video/DOM/performance artifacts from a single page load",
				Run: func(args []string) error {
					return runCapture(ctx, args)
				},
			},
		},
	}
}

type captureFlags struct {
	url               string
	captures          []string
	dimensions        string // "mobile" | "tablet" | "desktop" | ""
	width             int
	height            int
	deviceScaleFactor float64
	hasWidth          bool
	hasHeight         bool
	hasDeviceScale    bool
	waitFor           string
	outDir            string
	label             string
	json              bool
	dryRun            bool
}

func runCapture(ctx *appctx.Context, args []string) error {
	for _, a := range args {
		if a == "--help" || a == "-h" {
			printCaptureHelp(ctx.Name)
			return nil
		}
	}

	f, err := parseCaptureFlags(args)
	if err != nil {
		return err
	}
	if strings.TrimSpace(f.url) == "" {
		return fmt.Errorf("--url is required")
	}

	req, err := buildCaptureRequest(f)
	if err != nil {
		return err
	}

	httpClient, baseURL := cliapp.NewConnectHTTPClient(ctx.Core)
	client := captureconnect.NewCaptureServiceClient(httpClient, baseURL)

	connectReq := connect.NewRequest(req)
	if f.dryRun {
		connectReq.Header().Set("X-Dry-Run", "true")
	}

	resp, err := client.Capture(context.Background(), connectReq)
	if err != nil {
		return cliapp.WrapAPIError("capture", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no capture response")
	}

	return renderCaptureResponse(resp.Msg, f.json)
}

func parseCaptureFlags(args []string) (captureFlags, error) {
	f := captureFlags{}
	needsValue := func(i int, name string) (string, error) {
		if i+1 >= len(args) {
			return "", fmt.Errorf("%s requires a value", name)
		}
		return args[i+1], nil
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--url":
			v, err := needsValue(i, "--url")
			if err != nil {
				return f, err
			}
			f.url = v
			i++
		case "--capture":
			v, err := needsValue(i, "--capture")
			if err != nil {
				return f, err
			}
			for _, tok := range strings.Split(v, ",") {
				tok = strings.TrimSpace(tok)
				if tok != "" {
					f.captures = append(f.captures, tok)
				}
			}
			i++
		case "--dimensions":
			v, err := needsValue(i, "--dimensions")
			if err != nil {
				return f, err
			}
			f.dimensions = strings.ToLower(strings.TrimSpace(v))
			i++
		case "--width":
			v, err := needsValue(i, "--width")
			if err != nil {
				return f, err
			}
			n, err := strconv.Atoi(v)
			if err != nil {
				return f, fmt.Errorf("--width: %w", err)
			}
			f.width = n
			f.hasWidth = true
			i++
		case "--height":
			v, err := needsValue(i, "--height")
			if err != nil {
				return f, err
			}
			n, err := strconv.Atoi(v)
			if err != nil {
				return f, fmt.Errorf("--height: %w", err)
			}
			f.height = n
			f.hasHeight = true
			i++
		case "--device-scale-factor":
			v, err := needsValue(i, "--device-scale-factor")
			if err != nil {
				return f, err
			}
			n, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return f, fmt.Errorf("--device-scale-factor: %w", err)
			}
			f.deviceScaleFactor = n
			f.hasDeviceScale = true
			i++
		case "--wait-for":
			v, err := needsValue(i, "--wait-for")
			if err != nil {
				return f, err
			}
			f.waitFor = v
			i++
		case "--out":
			v, err := needsValue(i, "--out")
			if err != nil {
				return f, err
			}
			f.outDir = v
			i++
		case "--label":
			v, err := needsValue(i, "--label")
			if err != nil {
				return f, err
			}
			f.label = v
			i++
		case "--json":
			f.json = true
		case "--dry-run":
			f.dryRun = true
		default:
			return f, fmt.Errorf("unknown option: %s", args[i])
		}
	}
	return f, nil
}

func buildCaptureRequest(f captureFlags) (*capturev1.CaptureRequest, error) {
	req := &capturev1.CaptureRequest{
		Url:    f.url,
		OutDir: f.outDir,
		Label:  f.label,
	}

	for _, tok := range f.captures {
		ct, err := parseCaptureType(tok)
		if err != nil {
			return nil, err
		}
		req.Captures = append(req.Captures, ct)
	}

	if f.dimensions != "" || f.hasWidth || f.hasHeight || f.hasDeviceScale {
		d := &capturev1.Dimensions{}
		// Explicit width/height wins over preset (mirrors server).
		if f.hasWidth || f.hasHeight {
			if f.hasWidth {
				w := int32(f.width)
				d.Width = &w
			}
			if f.hasHeight {
				h := int32(f.height)
				d.Height = &h
			}
		} else if f.dimensions != "" {
			p, err := parseDimensionsPreset(f.dimensions)
			if err != nil {
				return nil, err
			}
			d.Preset = p
		}
		if f.hasDeviceScale {
			s := f.deviceScaleFactor
			d.DeviceScaleFactor = &s
		}
		req.Dimensions = d
	}

	if strings.TrimSpace(f.waitFor) != "" {
		wf, err := parseWaitFor(f.waitFor)
		if err != nil {
			return nil, err
		}
		req.WaitFor = wf
	}

	return req, nil
}

func parseCaptureType(tok string) (capturev1.CaptureType, error) {
	switch strings.ToLower(strings.ReplaceAll(tok, "_", "-")) {
	case "screenshot":
		return capturev1.CaptureType_CAPTURE_TYPE_SCREENSHOT, nil
	case "console-logs", "console", "logs":
		return capturev1.CaptureType_CAPTURE_TYPE_CONSOLE_LOGS, nil
	case "network":
		return capturev1.CaptureType_CAPTURE_TYPE_NETWORK, nil
	case "video":
		return capturev1.CaptureType_CAPTURE_TYPE_VIDEO, nil
	case "dom":
		return capturev1.CaptureType_CAPTURE_TYPE_DOM, nil
	case "performance", "perf":
		return capturev1.CaptureType_CAPTURE_TYPE_PERFORMANCE, nil
	}
	return capturev1.CaptureType_CAPTURE_TYPE_UNSPECIFIED, fmt.Errorf("unknown capture type %q (want one of: screenshot,console-logs,network,video,dom,performance)", tok)
}

func parseDimensionsPreset(s string) (capturev1.DimensionsPreset, error) {
	switch s {
	case "mobile":
		return capturev1.DimensionsPreset_DIMENSIONS_PRESET_MOBILE, nil
	case "tablet":
		return capturev1.DimensionsPreset_DIMENSIONS_PRESET_TABLET, nil
	case "desktop":
		return capturev1.DimensionsPreset_DIMENSIONS_PRESET_DESKTOP, nil
	}
	return capturev1.DimensionsPreset_DIMENSIONS_PRESET_UNSPECIFIED, fmt.Errorf("unknown dimensions preset %q (want mobile|tablet|desktop)", s)
}

func parseWaitFor(s string) (*capturev1.WaitFor, error) {
	s = strings.TrimSpace(s)
	// Numeric → timeout_ms.
	if n, err := strconv.Atoi(s); err == nil {
		return &capturev1.WaitFor{Spec: &capturev1.WaitFor_TimeoutMs{TimeoutMs: int32(n)}}, nil
	}
	if strings.EqualFold(s, "networkidle") {
		return &capturev1.WaitFor{Spec: &capturev1.WaitFor_Networkidle{Networkidle: true}}, nil
	}
	return &capturev1.WaitFor{Spec: &capturev1.WaitFor_Selector{Selector: s}}, nil
}

func renderCaptureResponse(msg *capturev1.CaptureResponse, asJSON bool) error {
	if asJSON {
		return cliapp.PrintReportJSON(os.Stdout, protoToJSON(msg))
	}

	result := []string{
		fmt.Sprintf("Captured %d artifact(s) in %dms.", len(msg.Artifacts), msg.DurationMs),
	}
	if msg.DryRun {
		result = append(result, "(dry-run — no artifacts written)")
	}
	if strings.TrimSpace(msg.ExecutionId) != "" {
		result = append(result, fmt.Sprintf("Execution: %s", msg.ExecutionId))
	}
	if strings.TrimSpace(msg.OutDir) != "" {
		result = append(result, fmt.Sprintf("Output: %s", msg.OutDir))
	}

	changes := make([]string, 0, len(msg.Artifacts))
	for _, a := range msg.Artifacts {
		changes = append(changes, fmt.Sprintf("%s — %s (%d bytes)", captureTypeLabel(a.Type), a.Path, a.SizeBytes))
	}

	nextCmd := []string{
		"`browser-automation-studio capture --url <...> --capture screenshot,console-logs,network --out <dir>` — full audit",
		"`browser-automation-studio execution show <id>` — inspect this execution",
	}

	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{
		Result:      result,
		Changes:     changes,
		NextCommand: nextCmd,
	})
}

// protoToJSON returns a flat map ready for PrintReportJSON. We avoid
// pulling protojson here — keeping the dependency surface minimal —
// because every field is plain-typed.
func protoToJSON(m *capturev1.CaptureResponse) map[string]interface{} {
	arts := make([]map[string]interface{}, 0, len(m.Artifacts))
	for _, a := range m.Artifacts {
		arts = append(arts, map[string]interface{}{
			"type":       captureTypeLabel(a.Type),
			"path":       a.Path,
			"size_bytes": a.SizeBytes,
			"metadata":   a.Metadata,
		})
	}
	return map[string]interface{}{
		"execution_id": m.ExecutionId,
		"out_dir":      m.OutDir,
		"duration_ms":  m.DurationMs,
		"dry_run":      m.DryRun,
		"artifacts":    arts,
	}
}

func captureTypeLabel(t capturev1.CaptureType) string {
	switch t {
	case capturev1.CaptureType_CAPTURE_TYPE_SCREENSHOT:
		return "screenshot"
	case capturev1.CaptureType_CAPTURE_TYPE_CONSOLE_LOGS:
		return "console-logs"
	case capturev1.CaptureType_CAPTURE_TYPE_NETWORK:
		return "network"
	case capturev1.CaptureType_CAPTURE_TYPE_VIDEO:
		return "video"
	case capturev1.CaptureType_CAPTURE_TYPE_DOM:
		return "dom"
	case capturev1.CaptureType_CAPTURE_TYPE_PERFORMANCE:
		return "performance"
	}
	return "unspecified"
}

func printCaptureHelp(cliName string) {
	fmt.Printf("Usage: %s capture --url <url-or-shorthand> [options]\n\n", cliName)
	fmt.Println("Capture artifacts from a single page load.")
	fmt.Println()
	fmt.Println("Required:")
	fmt.Println("  --url <url>             http(s) URL OR `scenario=<slug>,path=<path>` shorthand")
	fmt.Println()
	fmt.Println("Artifacts:")
	fmt.Println("  --capture <csv>         comma-separated: screenshot,console-logs,network,video,dom,performance")
	fmt.Println("                          (default: screenshot)")
	fmt.Println()
	fmt.Println("Viewport:")
	fmt.Println("  --dimensions <preset>   mobile (390x844) | tablet (768x1024) | desktop (1440x900)")
	fmt.Println("  --width <int>           explicit width (overrides preset)")
	fmt.Println("  --height <int>          explicit height (overrides preset)")
	fmt.Println("  --device-scale-factor <f> CSS pixel ratio (0.5-4.0)")
	fmt.Println()
	fmt.Println("Readiness:")
	fmt.Println("  --wait-for <spec>       CSS selector | 'networkidle' | timeout in ms")
	fmt.Println()
	fmt.Println("Output:")
	fmt.Println("  --out <dir>             server-relative output directory")
	fmt.Println("  --label <s>             echoed into the artifact bundle")
	fmt.Println("  --json                  emit proto wire shape")
	fmt.Println("  --dry-run               validate without producing artifacts")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Printf("  %s capture --url scenario=app-monitor,path=/ --capture screenshot\n", cliName)
	fmt.Printf("  %s capture --url https://example.com --capture screenshot,console-logs --dimensions mobile\n", cliName)
	fmt.Printf("  %s capture --url scenario=swarm-manager,path=/backlog --capture screenshot,console-logs,network --out /tmp/audit\n", cliName)
}
