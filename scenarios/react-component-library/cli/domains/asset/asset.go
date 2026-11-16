package asset

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	catalogv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/catalog"
	catalogconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/catalog/catalog_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
)

const GroupName = "asset"

func shortRevision(revision string) string {
	revision = strings.TrimSpace(revision)
	if revision == "" {
		return "unknown"
	}
	if len(revision) > 8 {
		return revision[:8]
	}
	return revision
}

type handlers struct {
	catalog      catalogClient
	buildCatalog func(string) error
}

type catalogClient interface {
	CheckAsset(context.Context, *connect.Request[catalogv1.CheckAssetRequest]) (*connect.Response[catalogv1.CheckAssetResponse], error)
	RecordGate(context.Context, *connect.Request[catalogv1.RunGateRequest]) (*connect.Response[catalogv1.RunGateResponse], error)
}

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	// Catalog build and scoped gates can legitimately take longer than the
	// ordinary CLI request budget. The server owns cancellation for the job;
	// the client must not disconnect while a valid asset verdict is finishing.
	httpClient, baseURL := cliapp.NewConnectHTTPClientWithTimeout(core, 20*time.Minute)
	h := &handlers{catalog: catalogconnect.NewCatalogServiceClient(httpClient, baseURL), buildCatalog: buildDerivedCatalog}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"check":        h.check,
		"record-check": h.recordCheck,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("asset: load from manifest: %w", err)
	}
	return group, nil
}

func (h *handlers) check(ctx cliapp.RunContext) error {
	assetID := strings.TrimSpace(ctx.Positional("asset-id"))
	if assetID == "" {
		return exitError{code: 2, message: "usage: asset check <asset-id>"}
	}
	profile := strings.TrimSpace(ctx.Flag("profile"))
	if profile == "" {
		profile = "publish"
	}
	if profile != "publish" && profile != "edit" {
		return exitError{code: 2, message: "profile must be publish or edit"}
	}
	resp, err := h.catalog.CheckAsset(context.Background(), connect.NewRequest(&catalogv1.CheckAssetRequest{AssetId: assetID, Version: strings.TrimSpace(ctx.Flag("version")), Profile: profile}))
	if err != nil {
		return exitError{code: 2, message: cliapp.WrapAPIError("check asset", err, nil).Error()}
	}
	if resp == nil || resp.Msg == nil {
		return exitError{code: 2, message: "server returned no asset check verdict"}
	}
	if ctx.JSON() || ctx.BoolFlag("json") {
		data, marshalErr := protojson.MarshalOptions{UseProtoNames: true, Indent: "  "}.Marshal(resp.Msg)
		if marshalErr != nil {
			return exitError{code: 2, message: marshalErr.Error()}
		}
		fmt.Println(string(data))
	} else {
		for _, stage := range resp.Msg.Stages {
			fmt.Printf("%-9s %-7s %s (%.1f s)\n", stage.Name, stage.Status, stage.Detail, stage.Seconds)
		}
		fmt.Printf("verdict   %-11s (%.1f s)\n", resp.Msg.Verdict, totalStageSeconds(resp.Msg.Stages))
	}
	switch resp.Msg.Verdict {
	case "PUBLISHABLE":
		return nil
	case "BLOCKED", "STALE_TESTS":
		return exitError{code: 1, message: fmt.Sprintf("asset %s: %s", assetID, resp.Msg.Verdict)}
	case "STALE_CATALOG":
		return exitError{code: 1, message: fmt.Sprintf("asset %s: %s; run `react-component-library asset record-check %s` to refresh and record evidence", assetID, resp.Msg.Verdict, assetID)}
	default:
		return exitError{code: 2, message: fmt.Sprintf("asset %s: %s", assetID, resp.Msg.Verdict)}
	}
}

// recordCheck is the explicit write path paired with check. It refreshes the
// derived catalog projections and records scoped gate evidence (including
// browser-backed appearance evidence when the configured executor is ready),
// then delegates the final verdict to the read-only API.
func (h *handlers) recordCheck(ctx cliapp.RunContext) error {
	assetID := strings.TrimSpace(ctx.Positional("asset-id"))
	if assetID == "" {
		return exitError{code: 2, message: "usage: asset record-check <asset-id>"}
	}
	root, err := findScenarioRoot()
	if err != nil {
		return err
	}
	return h.recordCheckAsset(assetID, root, ctx.JSON() || ctx.BoolFlag("json"))
}

func (h *handlers) recordCheckAsset(assetID, root string, jsonOutput bool) error {
	build := h.buildCatalog
	if build == nil {
		build = buildDerivedCatalog
	}
	if err := build(root); err != nil {
		return exitError{code: 2, message: fmt.Sprintf("refresh derived catalog files: %v", err)}
	}
	resp, err := h.catalog.RecordGate(context.Background(), connect.NewRequest(&catalogv1.RunGateRequest{All: true, AssetId: assetID}))
	if err != nil {
		return exitError{code: 2, message: cliapp.WrapAPIError("record asset gate evidence", err, nil).Error()}
	}
	if resp == nil || resp.Msg == nil {
		return exitError{code: 2, message: "server returned no recorded gate result"}
	}
	checked, err := h.catalog.CheckAsset(context.Background(), connect.NewRequest(&catalogv1.CheckAssetRequest{AssetId: assetID, Profile: "publish"}))
	if err != nil {
		return exitError{code: 2, message: cliapp.WrapAPIError("check asset after recording evidence", err, nil).Error()}
	}
	if checked == nil || checked.Msg == nil {
		return exitError{code: 2, message: "server returned no asset check verdict"}
	}
	if jsonOutput {
		data, marshalErr := protojson.MarshalOptions{UseProtoNames: true, Indent: "  "}.Marshal(checked.Msg)
		if marshalErr != nil {
			return exitError{code: 2, message: marshalErr.Error()}
		}
		fmt.Println(string(data))
	} else {
		fmt.Printf("recorded %d gate evidence row(s) for %s\n", resp.Msg.GetEvidenceRowsWritten(), assetID)
		for _, stage := range checked.Msg.Stages {
			fmt.Printf("%-9s %-7s %s (%.1f s)\n", stage.Name, stage.Status, stage.Detail, stage.Seconds)
		}
		fmt.Printf("verdict   %-11s (%.1f s)\n", checked.Msg.Verdict, totalStageSeconds(checked.Msg.Stages))
	}
	if checked.Msg.GetVerdict() == "PUBLISHABLE" {
		return nil
	}
	return exitError{code: 1, message: fmt.Sprintf("asset %s: %s", assetID, checked.Msg.GetVerdict())}
}

func buildDerivedCatalog(root string) error {
	cmd := exec.CommandContext(context.Background(), "node", "tooling/catalog-build.mjs")
	cmd.Dir = filepath.Join(root, "..", "..", "packages", "react-component-library")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}

func findScenarioRoot() (string, error) {
	start, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for current := filepath.Clean(start); ; current = filepath.Dir(current) {
		if _, apiErr := os.Stat(filepath.Join(current, "api", "go.mod")); apiErr == nil {
			if _, catalogErr := os.Stat(filepath.Join(current, "catalog", "config.json")); catalogErr == nil {
				return current, nil
			}
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
	}
	return "", fmt.Errorf("cannot locate react-component-library scenario root")
}

func totalStageSeconds(stages []*catalogv1.AssetCheckStage) float64 {
	var total float64
	for _, stage := range stages {
		total += stage.Seconds
	}
	return total
}

type exitError struct {
	code    int
	message string
}

func (e exitError) Error() string { return e.message }
func (e exitError) ExitCode() int { return e.code }
