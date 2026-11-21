package runs

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

// commandLogKind is the artifact kind a phase's command output is cataloged as.
const commandLogKind = "command.log"

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

// artifactAccessPath is the run-scoped stream route the API must report for an
// artifact; anything else is refused so a server can never redirect the read.
func artifactAccessPath(scenario, runID, artifactID string) string {
	return "/api/v1/scenarios/" + url.PathEscape(scenario) + "/runs/" +
		url.PathEscape(runID) + "/artifacts/" + url.PathEscape(artifactID)
}

// streamArtifact resolves one cataloged artifact's metadata, verifies it grants
// stream access on its run-scoped route, and copies exactly its declared bytes
// to w. It returns the metadata response and the number of bytes written.
func streamArtifact(apiClient *cliutil.APIClient, scenario, runID, artifactID string, w io.Writer) (*runspb.GetRunArtifactResponse, int64, error) {
	if apiClient == nil {
		return nil, 0, fmt.Errorf("test-genie API client is unavailable")
	}
	cl, err := client(apiClient)
	if err != nil {
		return nil, 0, err
	}
	metadata, err := cl.GetRunArtifact(context.Background(), connect.NewRequest(&runspb.GetRunArtifactRequest{
		Target: scenario, RunId: runID, ArtifactId: artifactID,
	}))
	if err != nil {
		return nil, 0, err
	}
	if metadata == nil || metadata.Msg == nil || metadata.Msg.GetArtifact() == nil {
		return nil, 0, fmt.Errorf("server returned no artifact metadata")
	}
	artifact := metadata.Msg.GetArtifact()
	if artifact.GetAccessCapability() != runspb.ArtifactAccessCapability_ARTIFACT_ACCESS_CAPABILITY_STREAM {
		return nil, 0, fmt.Errorf("artifact does not grant stream access")
	}
	accessPath := artifactAccessPath(scenario, runID, artifactID)
	if artifact.GetId() != artifactID || artifact.GetAccessPath() != accessPath {
		return nil, 0, fmt.Errorf("server returned an unexpected artifact access path")
	}
	written, err := apiClient.WithoutTimeout().WithoutRedirects().DownloadTo(context.Background(), accessPath, w)
	if err != nil {
		return nil, written, fmt.Errorf("download artifact: %w", err)
	}
	if artifact.GetSizeBytes() < 0 || written != artifact.GetSizeBytes() {
		return nil, written, fmt.Errorf("downloaded artifact size %d does not match metadata size %d", written, artifact.GetSizeBytes())
	}
	return metadata.Msg, written, nil
}

// writesToStdout reports whether --out selects standard output (absent or "-").
func writesToStdout(output string) bool {
	output = strings.TrimSpace(output)
	return output == "" || output == "-"
}

// downloadArtifactHandler prints an artifact's bytes to stdout by default, the
// read agents pipe into rg or jq; --out <path> publishes it atomically with a
// byte-count and SHA-256 receipt instead.
func downloadArtifactHandler(apiClient *cliutil.APIClient) cliapp.PrimitiveHandler {
	toFile := cliapp.Action(downloadArtifactCall(apiClient), downloadArtifactReport).Run
	return cliapp.ContentAction(func(ctx cliapp.RunContext) error {
		if !writesToStdout(ctx.Flag("out")) {
			return toFile(ctx)
		}
		scenario, runID, artifactID := artifactArgs(ctx)
		if scenario == "" || runID == "" || artifactID == "" {
			return fmt.Errorf("scenario, run id, and artifact id are required")
		}
		_, _, err := streamArtifact(apiClient, scenario, runID, artifactID, ctx.Stdout())
		return err
	})
}

func artifactArgs(ctx cliapp.OperationContext) (scenario, runID, artifactID string) {
	return strings.TrimSpace(ctx.Positional("scenario")),
		strings.TrimSpace(ctx.Positional("run_id")),
		strings.TrimSpace(ctx.Positional("artifact_id"))
}

func downloadArtifactCall(apiClient *cliutil.APIClient) func(cliapp.OperationContext) (artifactDownloadResult, error) {
	return func(ctx cliapp.OperationContext) (artifactDownloadResult, error) {
		scenario, runID, artifactID := artifactArgs(ctx)
		output := strings.TrimSpace(ctx.Flag("out"))
		if scenario == "" || runID == "" || artifactID == "" || writesToStdout(output) {
			return artifactDownloadResult{}, fmt.Errorf("scenario, run id, artifact id, and an --out file path are required")
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
		metadata, bytesWritten, err := streamArtifact(apiClient, scenario, runID, artifactID, io.MultiWriter(tmp, hash))
		if err != nil {
			_ = tmp.Close()
			return artifactDownloadResult{}, err
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
			Kind: metadata.GetArtifact().GetKind(), OutputPath: outputPath,
			Bytes: bytesWritten, SHA256: fmt.Sprintf("%x", hash.Sum(nil)),
			LegacyDiscovered: metadata.GetLegacyDiscovered(),
		}, nil
	}
}

func downloadArtifactReport(_ cliapp.OperationContext, result artifactDownloadResult) cliapp.MutationReport {
	return cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Downloaded %s (%d bytes)", result.ArtifactID, result.Bytes)},
		Changes: []string{"output=" + result.OutputPath, "sha256=" + result.SHA256, "kind=" + result.Kind},
	}
}

// runLogsHandler prints the command logs a run's phases recorded, one phase
// after another under a header, so reading a failed phase's output is one
// command instead of a catalog lookup plus a download per phase.
func runLogsHandler(apiClient *cliutil.APIClient) cliapp.PrimitiveHandler {
	return cliapp.ContentAction(func(ctx cliapp.RunContext) error {
		return printRunLogs(apiClient, ctx, ctx.Stdout())
	})
}

func printRunLogs(apiClient *cliutil.APIClient, ctx cliapp.OperationContext, w io.Writer) error {
	scenario := strings.TrimSpace(ctx.Positional("scenario"))
	runID := strings.TrimSpace(ctx.Positional("run_id"))
	phase := strings.TrimSpace(ctx.Flag("phase"))
	if scenario == "" || runID == "" {
		return fmt.Errorf("scenario and run id are required")
	}
	cl, err := client(apiClient)
	if err != nil {
		return err
	}
	catalog, err := cl.ListRunArtifacts(context.Background(), connect.NewRequest(&runspb.ListRunArtifactsRequest{
		Target: scenario, RunId: runID, Kinds: []string{commandLogKind}, ProducingPhase: phase,
	}))
	if err != nil {
		return err
	}
	logs := make([]*runspb.ArtifactRef, 0, len(catalog.Msg.GetArtifacts()))
	for _, artifact := range catalog.Msg.GetArtifacts() {
		if artifact != nil && artifact.GetKind() == commandLogKind && (phase == "" || artifact.GetProducingPhase() == phase) {
			logs = append(logs, artifact)
		}
	}
	if len(logs) == 0 {
		scope := "any phase"
		if phase != "" {
			scope = "phase " + phase
		}
		return fmt.Errorf("run %s/%s has no %s artifact for %s; list what it recorded with: test-genie runs artifacts %s %s", scenario, runID, commandLogKind, scope, scenario, runID)
	}
	sort.SliceStable(logs, func(i, j int) bool { return logs[i].GetCreatedAt() < logs[j].GetCreatedAt() })
	for i, artifact := range logs {
		if len(logs) > 1 {
			if i > 0 {
				fmt.Fprintln(w)
			}
			fmt.Fprintf(w, "==> phase %s: %s (%s, %d bytes) <==\n", artifact.GetProducingPhase(), artifact.GetLabel(), artifact.GetId(), artifact.GetSizeBytes())
		}
		if _, _, err := streamArtifact(apiClient, scenario, runID, artifact.GetId(), w); err != nil {
			return fmt.Errorf("phase %s log %s: %w", artifact.GetProducingPhase(), artifact.GetId(), err)
		}
	}
	return nil
}
