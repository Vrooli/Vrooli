package asset

import (
	"context"
	"fmt"
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
	catalog catalogconnect.CatalogServiceClient
}

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	// Catalog build and scoped gates can legitimately take longer than the
	// ordinary CLI request budget. The server owns cancellation for the job;
	// the client must not disconnect while a valid asset verdict is finishing.
	httpClient, baseURL := cliapp.NewConnectHTTPClientWithTimeout(core, 20*time.Minute)
	h := &handlers{catalog: catalogconnect.NewCatalogServiceClient(httpClient, baseURL)}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"check": h.check,
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
	resp, err := h.catalog.CheckAsset(context.Background(), connect.NewRequest(&catalogv1.CheckAssetRequest{AssetId: assetID, Version: strings.TrimSpace(ctx.Flag("version")), RunTests: ctx.BoolFlag("run-tests")}))
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
	default:
		return exitError{code: 2, message: fmt.Sprintf("asset %s: %s", assetID, resp.Msg.Verdict)}
	}
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
