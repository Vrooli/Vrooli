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
		}},
		RunCtx: func(rc cliapp.RunContext) error {
			request, err := buildPreviewScreenshotRequest(previewScreenshotFlagsFromContext(rc))
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

func previewScreenshotFlagsFromContext(rc cliapp.RunContext) previewScreenshotFlags {
	f := previewScreenshotFlags{url: strings.TrimSpace(rc.Flag("url")), device: strings.ToLower(strings.TrimSpace(rc.Flag("device")))}
	if value := rc.Flag("width"); value != "" {
		f.width, _ = strconv.Atoi(value)
	}
	if value := rc.Flag("height"); value != "" {
		f.height, _ = strconv.Atoi(value)
	}
	if value := rc.Flag("device-scale-factor"); value != "" {
		f.deviceScale, _ = strconv.ParseFloat(value, 64)
		f.hasDeviceScale = true
	}
	return f
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
	if width != 0 || flags.hasDeviceScale {
		request.Viewport = &aiv1.Viewport{Width: int32(width), Height: int32(height)}
		if flags.hasDeviceScale {
			scale := flags.deviceScale
			request.Viewport.DeviceScaleFactor = &scale
		}
	}
	return request, nil
}
