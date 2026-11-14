package preparation

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	ai "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ai"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/preparation"
	pc "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/preparation/preparation_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	httpClient, url := cliapp.NewConnectHTTPClient(core)
	c := pc.NewPreparationServiceClient(httpClient, url)
	bindings := map[string]func(cliapp.RunContext) error{
		"PreparationService.AssessRepair": func(ctx cliapp.RunContext) error {
			contextPixels, maxEdge := uint64(0), uint64(0)
			var err error
			if value := ctx.Flag("context-pixels"); value != "" {
				contextPixels, err = strconv.ParseUint(value, 10, 32)
				if err != nil {
					return fmt.Errorf("invalid --context-pixels: %w", err)
				}
			}
			if value := ctx.Flag("max-edge-px"); value != "" {
				maxEdge, err = strconv.ParseUint(value, 10, 32)
				if err != nil {
					return fmt.Errorf("invalid --max-edge-px: %w", err)
				}
			}
			resp, err := c.AssessRepair(context.Background(), connect.NewRequest(&v1.AssessRepairRequest{MaskRef: ctx.Positional("mask_ref"), Inpaint: &ai.InpaintParams{CropToMask: ctx.BoolFlag("crop-to-mask"), ContextPixels: uint32(contextPixels), MaxEdgePx: uint32(maxEdge)}}))
			if err != nil {
				return cliapp.WrapAPIError("assess repair context", err, nil)
			}
			a := resp.Msg.GetAssessment()
			return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Removal covers %.2f%% of source; advisory risk %s", a.GetMaskAreaFraction()*100, a.GetRiskLevel())}, ResultsHeading: "Measured context", Results: a.GetRiskSignals()})
		},
		"PreparationService.CreatePhotograph": func(ctx cliapp.RunContext) error {
			req := &v1.CreatePhotographRequest{}
			if err := readInput(ctx, req); err != nil {
				return err
			}
			resp, err := c.CreatePhotograph(context.Background(), connect.NewRequest(req))
			if err != nil {
				return cliapp.WrapAPIError("create photograph preparation", err, nil)
			}
			return render(ctx, resp.Msg)
		},
		"PreparationService.GetPhotograph": func(ctx cliapp.RunContext) error {
			resp, err := c.GetPhotograph(context.Background(), connect.NewRequest(&v1.GetPhotographRequest{Id: ctx.Positional("id")}))
			if err != nil {
				return cliapp.WrapAPIError("get photograph preparation", err, nil)
			}
			return render(ctx, resp.Msg)
		},
		"PreparationService.ListPhotographs": func(ctx cliapp.RunContext) error {
			limit := uint64(30)
			var err error
			if ctx.Flag("limit") != "" {
				limit, err = strconv.ParseUint(ctx.Flag("limit"), 10, 32)
				if err != nil {
					return fmt.Errorf("invalid --limit: %w", err)
				}
			}
			resp, err := c.ListPhotographs(context.Background(), connect.NewRequest(&v1.ListPhotographsRequest{Limit: uint32(limit)}))
			if err != nil {
				return cliapp.WrapAPIError("list photograph preparations", err, nil)
			}
			results := []string{}
			for _, p := range resp.Msg.Photographs {
				results = append(results, fmt.Sprintf("%s  %s  [%s]", p.Id, p.Label, p.Manifest.GetReview().GetDecision()))
			}
			return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("%d immutable preparation revision(s)", len(results))}, ResultsHeading: "Preparations", Results: results})
		},
		"PreparationService.ReviewPhotograph": func(ctx cliapp.RunContext) error {
			req := &v1.ReviewPhotographRequest{}
			if err := readInput(ctx, req); err != nil {
				return err
			}
			resp, err := c.ReviewPhotograph(context.Background(), connect.NewRequest(req))
			if err != nil {
				return cliapp.WrapAPIError("review photograph preparation", err, nil)
			}
			return render(ctx, resp.Msg)
		},
	}
	return cliapp.LoadFromManifest(manifest, "preparation", bindings)
}

func render(ctx cliapp.RunContext, resp *v1.PhotographResponse) error {
	p := resp.GetPhotograph()
	return cliapp.RenderProtoList(ctx, resp, cliapp.ListReport{Summary: []string{p.GetLabel() + " [" + p.GetManifest().GetReview().GetDecision() + "]"}, ResultsHeading: "Immutable preparation", Results: []string{"id: " + p.GetId(), "bundle: " + p.GetBundleRef(), "parent: " + p.GetParentId()}})
}
func readInput(ctx cliapp.RunContext, m proto.Message) error {
	raw := ctx.Flag("request")
	if raw == "" {
		return readRequest(ctx.Flag("file"), m)
	}
	if ctx.Flag("file") != "" {
		return fmt.Errorf("use exactly one of --file or --request")
	}
	if len(raw) > 256<<10 {
		return fmt.Errorf("preparation request exceeds 256 KiB")
	}
	if err := protojson.Unmarshal([]byte(raw), m); err != nil {
		return fmt.Errorf("preparation request: %w", err)
	}
	return nil
}

func readRequest(path string, m proto.Message) error {
	if path == "" {
		return fmt.Errorf("--file is required")
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, (256<<10)+1))
	if err != nil {
		return err
	}
	if len(raw) > 256<<10 {
		return fmt.Errorf("preparation request exceeds 256 KiB")
	}
	if err := protojson.Unmarshal(raw, m); err != nil {
		return fmt.Errorf("preparation request: %w", err)
	}
	return nil
}
