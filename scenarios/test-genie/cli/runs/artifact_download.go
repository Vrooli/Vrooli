package runs

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

type artifactDownloadResult struct {
	Scenario         string `json:"scenario"`
	RunID            string `json:"run_id"`
	ArtifactID       string `json:"artifact_id"`
	Kind             string `json:"kind"`
	OutputPath       string `json:"output_path"`
	Bytes            int64  `json:"bytes"`
	SHA256           string `json:"sha256"`
	LegacyDiscovered bool   `json:"legacy_discovered,omitempty"`
}

func downloadArtifactCall(apiClient *cliutil.APIClient) func(cliapp.OperationContext) (artifactDownloadResult, error) {
	return func(ctx cliapp.OperationContext) (artifactDownloadResult, error) {
		scenario := strings.TrimSpace(ctx.Flag("scenario"))
		runID := strings.TrimSpace(ctx.Positional("run_id"))
		artifactID := strings.TrimSpace(ctx.Positional("artifact_id"))
		output := strings.TrimSpace(ctx.Flag("out"))
		if scenario == "" || runID == "" || artifactID == "" || output == "" {
			return artifactDownloadResult{}, fmt.Errorf("scenario, run id, artifact id, and --out are required")
		}
		if apiClient == nil {
			return artifactDownloadResult{}, fmt.Errorf("test-genie API client is unavailable")
		}

		cl, err := client(apiClient)
		if err != nil {
			return artifactDownloadResult{}, err
		}
		metadata, err := cl.GetRunArtifact(context.Background(), connect.NewRequest(&runspb.GetRunArtifactRequest{
			Target: scenario, RunId: runID, ArtifactId: artifactID,
		}))
		if err != nil {
			return artifactDownloadResult{}, err
		}
		if metadata == nil || metadata.Msg == nil || metadata.Msg.GetArtifact() == nil {
			return artifactDownloadResult{}, fmt.Errorf("server returned no artifact metadata")
		}
		artifact := metadata.Msg.GetArtifact()
		if artifact.GetAccessCapability() != runspb.ArtifactAccessCapability_ARTIFACT_ACCESS_CAPABILITY_STREAM {
			return artifactDownloadResult{}, fmt.Errorf("artifact does not grant stream access")
		}
		accessPath := "/api/v1/scenarios/" + url.PathEscape(scenario) + "/runs/" +
			url.PathEscape(runID) + "/artifacts/" + url.PathEscape(artifactID)
		if artifact.GetId() != artifactID || artifact.GetAccessPath() != accessPath {
			return artifactDownloadResult{}, fmt.Errorf("server returned an unexpected artifact access path")
		}

		outputPath, err := filepath.Abs(output)
		if err != nil {
			return artifactDownloadResult{}, fmt.Errorf("resolve output path: %w", err)
		}
		tmp, err := os.CreateTemp(filepath.Dir(outputPath), ".test-genie-artifact-*")
		if err != nil {
			return artifactDownloadResult{}, fmt.Errorf("create temporary output beside destination: %w", err)
		}
		tmpPath := tmp.Name()
		defer os.Remove(tmpPath)

		hash := sha256.New()
		bytesWritten, err := apiClient.WithoutTimeout().WithoutRedirects().DownloadTo(context.Background(), accessPath, io.MultiWriter(tmp, hash))
		if err != nil {
			_ = tmp.Close()
			return artifactDownloadResult{}, fmt.Errorf("download artifact: %w", err)
		}
		if artifact.GetSizeBytes() < 0 || bytesWritten != artifact.GetSizeBytes() {
			_ = tmp.Close()
			return artifactDownloadResult{}, fmt.Errorf("downloaded artifact size %d does not match metadata size %d", bytesWritten, artifact.GetSizeBytes())
		}
		if err := tmp.Sync(); err != nil {
			_ = tmp.Close()
			return artifactDownloadResult{}, fmt.Errorf("sync downloaded artifact: %w", err)
		}
		if err := tmp.Close(); err != nil {
			return artifactDownloadResult{}, fmt.Errorf("close downloaded artifact: %w", err)
		}
		if ctx.BoolFlag("overwrite") {
			if err := os.Rename(tmpPath, outputPath); err != nil {
				return artifactDownloadResult{}, fmt.Errorf("publish downloaded artifact: %w", err)
			}
		} else if err := os.Link(tmpPath, outputPath); err != nil {
			return artifactDownloadResult{}, fmt.Errorf("publish downloaded artifact without overwrite (use --overwrite to replace): %w", err)
		}

		return artifactDownloadResult{
			Scenario: scenario, RunID: runID, ArtifactID: artifactID,
			Kind: artifact.GetKind(), OutputPath: outputPath,
			Bytes: bytesWritten, SHA256: fmt.Sprintf("%x", hash.Sum(nil)),
			LegacyDiscovered: metadata.Msg.GetLegacyDiscovered(),
		}, nil
	}
}

func downloadArtifactReport(_ cliapp.OperationContext, result artifactDownloadResult) cliapp.MutationReport {
	return cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Downloaded %s (%d bytes)", result.ArtifactID, result.Bytes)},
		Changes: []string{"output=" + result.OutputPath, "sha256=" + result.SHA256, "kind=" + result.Kind},
	}
}
