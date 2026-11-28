package ai

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"browser-automation-studio/cli/internal/appctx"

	"connectrpc.com/connect"
	"github.com/vrooli/browser-automation-studio/viewport"
	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/ai"
	aiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/ai/aiconnect"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/vrooli/cli-core/cliapp"
)

type previewScreenshotFlags struct {
	url            string
	width, height  int
	device         string
	deviceScale    float64
	hasDeviceScale bool
	waitFor        string
	waitUntil      string
	settleMs       int
}

func previewScreenshotCommand(core *cliapp.ScenarioApp) cliapp.Command {
	ctx := &appctx.Context{Core: core}
	return cliapp.Command{
		Name: "preview-screenshot", NeedsAPI: true,
		Description: "Navigate to a URL and capture a full-page screenshot at an explicit viewport or device preset.",
		Args: cliapp.ArgSchema{Flags: []cliapp.Flag{
			{Name: "url", Required: true, Description: "Target URL to capture"},
			{Name: "width", Description: "Viewport width in CSS pixels"},
			{Name: "height", Description: "Viewport height in CSS pixels"},
			{Name: "device", Description: "Viewport preset: mobile (390x844), tablet (768x1024), desktop (1440x900)"},
			{Name: "device-scale-factor", Description: "Browser device scale factor (0.5-4.0)"},
			{Name: "wait-for", Description: "CSS selector that must exist before capture"},
			{Name: "wait-until", Description: "Navigation readiness: load|domcontentloaded|networkidle"},
			{Name: "settle-ms", Description: "Fixed delay after readiness (0-15000 ms)"},
		}},
		RunCtx: func(rc cliapp.RunContext) error {
			flags, err := previewScreenshotFlagsFromContext(rc)
			if err != nil {
				return err
			}
			request, err := buildPreviewScreenshotRequest(flags)
			if err != nil {
				return err
			}
			httpClient, baseURL := cliapp.NewConnectHTTPClient(ctx.Core)
			response, err := aiconnect.NewAIServiceClient(httpClient, baseURL).TakePreviewScreenshot(context.Background(), connect.NewRequest(request))
			if err != nil {
				return cliapp.WrapAPIError("preview screenshot", err, nil)
			}
			if response == nil || response.Msg == nil {
				return fmt.Errorf("server returned no preview screenshot")
			}
			if rc.JSON() {
				encoded, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(response.Msg)
				if err != nil {
					return err
				}
				fmt.Println(string(encoded))
				return nil
			}
			return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{Result: []string{fmt.Sprintf("Captured preview at %dx%d.", response.Msg.ViewportWidth, response.Msg.ViewportHeight)}})
		},
	}
}

func previewScreenshotFlagsFromContext(rc cliapp.RunContext) (previewScreenshotFlags, error) {
	f := previewScreenshotFlags{url: strings.TrimSpace(rc.Flag("url")), device: strings.ToLower(strings.TrimSpace(rc.Flag("device"))), waitFor: strings.TrimSpace(rc.Flag("wait-for")), waitUntil: strings.ToLower(strings.TrimSpace(rc.Flag("wait-until")))}
	if f.waitUntil != "" && f.waitUntil != "load" && f.waitUntil != "domcontentloaded" && f.waitUntil != "networkidle" {
		return f, fmt.Errorf("--wait-until must be load, domcontentloaded, or networkidle")
	}
	if value := rc.Flag("settle-ms"); value != "" {
		settle, err := strconv.Atoi(value)
		if err != nil || settle < 0 || settle > 15000 {
			return f, fmt.Errorf("--settle-ms must be between 0 and 15000")
		}
		f.settleMs = settle
	}
	if value := rc.Flag("width"); value != "" {
		width, err := strconv.Atoi(value)
		if err != nil {
			return f, fmt.Errorf("--width: %w", err)
		}
		f.width = width
	}
	if value := rc.Flag("height"); value != "" {
		height, err := strconv.Atoi(value)
		if err != nil {
			return f, fmt.Errorf("--height: %w", err)
		}
		f.height = height
	}
	if value := rc.Flag("device-scale-factor"); value != "" {
		deviceScale, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return f, fmt.Errorf("--device-scale-factor: %w", err)
		}
		f.deviceScale = deviceScale
		f.hasDeviceScale = true
	}
	return f, nil
}

func buildPreviewScreenshotRequest(flags previewScreenshotFlags) (*aiv1.TakePreviewScreenshotRequest, error) {
	if flags.url == "" {
		return nil, fmt.Errorf("--url is required")
	}
	if flags.device != "" && (flags.width != 0 || flags.height != 0) {
		return nil, fmt.Errorf("--device cannot be combined with --width or --height")
	}
	width, height := flags.width, flags.height
	if flags.device != "" {
		preset, err := viewport.Resolve(flags.device)
		if err != nil {
			return nil, err
		}
		width, height = int(preset.Width), int(preset.Height)
	}
	if (width == 0) != (height == 0) {
		return nil, fmt.Errorf("--width and --height must be set together")
	}
	if flags.hasDeviceScale && (flags.deviceScale < 0.5 || flags.deviceScale > 4.0) {
		return nil, fmt.Errorf("--device-scale-factor must be between 0.5 and 4.0")
	}
	request := &aiv1.TakePreviewScreenshotRequest{Url: flags.url}
	request.WaitFor = flags.waitFor
	request.SettleMs = int32(flags.settleMs)
	switch flags.waitUntil {
	case "domcontentloaded":
		request.WaitUntil = aiv1.WaitUntil_WAIT_UNTIL_DOMCONTENTLOADED
	case "networkidle":
		request.WaitUntil = aiv1.WaitUntil_WAIT_UNTIL_NETWORKIDLE
	default:
		request.WaitUntil = aiv1.WaitUntil_WAIT_UNTIL_LOAD
	}
	if width != 0 || flags.hasDeviceScale {
		request.Viewport = &aiv1.Viewport{Width: int32(width), Height: int32(height)}
		if flags.hasDeviceScale {
			scale := flags.deviceScale
			request.Viewport.DeviceScaleFactor = &scale
		}
	}
	return request, nil
}
