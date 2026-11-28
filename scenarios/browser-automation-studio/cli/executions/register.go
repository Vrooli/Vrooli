// Package executions wires the BAS `executions` CLI group to the proto
// ExecutionsService via the generic protodispatch dispatcher.
//
// ExecutionsService lives in the browser_automation_studio.v1 namespace
// alongside WorkflowsService (api/service.proto). Multipart export and
// HAR/video streaming endpoints remain on REST per
// docs/internal/REST_EXCEPTIONS.md; the JSON metadata methods migrate to
// Connect via this register.
//
// Part of Phase 10 of the BAS proto+Connect migration
// (plans:bas-migration-to-proto-connect-rpc). The prior hand-coded
// REST-driven execution handlers (list.go, watch.go, render.go,
// render_video.go, etc.) have been deleted; the chi routes they targeted
// were removed in earlier phases.
package executions

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"browser-automation-studio/cli/internal/api"
	"browser-automation-studio/cli/internal/appctx"
	apiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this register package owns.
const GroupName = "executions"

// Register builds the executions subcommand group by binding every
// manifest command to a generic protodispatch handler resolved against
// the generated ExecutionsService descriptor.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	svc := apiv1.File_browser_automation_studio_v1_api_service_proto.Services().ByName("ExecutionsService")
	if svc == nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("%s: ExecutionsService descriptor not found", GroupName)
	}
	bindings, err := cliapp.ProtoBindings(core, svc.FullName(), cliapp.ProtoBindingOptions{})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("%s: %w", GroupName, err)
	}
	for _, command := range purgeCommands(core) {
		bindings[command.Name] = command.RunCtx
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("%s: load from manifest: %w", GroupName, err)
	}
	return group, nil
}

var uuidPattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

type purgePreview struct {
	ExecutionID        string `json:"execution_id"`
	ArtifactFiles      int64  `json:"artifact_files"`
	ArtifactBytes      int64  `json:"artifact_bytes"`
	ArtifactSHA256     string `json:"artifact_sha256"`
	ArtifactPathSHA256 string `json:"artifact_path_sha256"`
	Status             string `json:"status"`
	Ready              bool   `json:"ready"`
}

func purgeCommands(core *cliapp.ScenarioApp) []cliapp.Command {
	ctx := &appctx.Context{Core: core}
	return []cliapp.Command{
		{
			Name: "purge-preview", NeedsAPI: true,
			Description: "Preview metadata for purging one execution by UUID",
			Args:        cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "execution-id", Required: true, Description: "Execution UUID"}}},
			RunCtx: func(rc cliapp.RunContext) error {
				id, err := parseExecutionUUID(rc.Positional("execution-id"))
				if err != nil {
					return err
				}
				preview, err := requestPurgePreview(ctx, id)
				if err != nil {
					return err
				}
				return printPurgeMetadata(rc.Stdout(), preview, nil)
			},
		},
		{
			Name: "purge", NeedsAPI: true,
			Description: "Purge one execution by UUID after an explicit matching confirmation",
			Args:        cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "execution-id", Required: true, Description: "Execution UUID"}}, Flags: []cliapp.Flag{{Name: "confirm", Required: true, Description: "Must exactly match the execution UUID"}}},
			RunCtx: func(rc cliapp.RunContext) error {
				id, err := parseExecutionUUID(rc.Positional("execution-id"))
				if err != nil {
					return err
				}
				confirmation, err := parseExecutionUUID(rc.Flag("confirm"))
				if err != nil {
					return fmt.Errorf("--confirm: %w", err)
				}
				if confirmation != id {
					return fmt.Errorf("--confirm must exactly match execution UUID %s", id)
				}
				preview, err := requestPurgePreview(ctx, id)
				if err != nil {
					return err
				}
				if !preview.Ready {
					return fmt.Errorf("execution %s is not ready to purge", id)
				}
				payload := struct {
					Confirmation string       `json:"confirmation"`
					Preview      purgePreview `json:"preview"`
				}{Confirmation: id, Preview: preview}
				body, err := json.Marshal(payload)
				if err != nil {
					return err
				}
				status, response, err := api.Do(ctx, http.MethodPost, "/executions/"+id+"/purge", nil, body, nil)
				if err != nil {
					return err
				}
				if status < 200 || status >= 300 {
					return fmt.Errorf("execution purge failed (HTTP %d)", status)
				}
				var result struct {
					purgePreview
					Deleted bool `json:"deleted"`
				}
				if err := json.Unmarshal(response, &result); err != nil {
					return fmt.Errorf("decode purge metadata: %w", err)
				}
				if !result.Deleted {
					return fmt.Errorf("purge response did not confirm deletion")
				}
				return printPurgeMetadata(rc.Stdout(), result.purgePreview, &result.Deleted)
			},
		},
	}
}

func parseExecutionUUID(value string) (string, error) {
	id := strings.TrimSpace(value)
	if !uuidPattern.MatchString(id) {
		return "", fmt.Errorf("execution ID must be a UUID")
	}
	return strings.ToLower(id), nil
}

func requestPurgePreview(ctx *appctx.Context, id string) (purgePreview, error) {
	var preview purgePreview
	status, response, err := api.Do(ctx, http.MethodPost, "/executions/"+id+"/purge-preview", nil, []byte(`{}`), nil)
	if err != nil {
		return preview, err
	}
	if status < 200 || status >= 300 {
		return preview, fmt.Errorf("execution purge preview failed (HTTP %d)", status)
	}
	if err := json.Unmarshal(response, &preview); err != nil {
		return preview, fmt.Errorf("decode purge preview metadata: %w", err)
	}
	if preview.ExecutionID != id {
		return preview, fmt.Errorf("purge preview returned a different execution UUID")
	}
	return preview, nil
}

func printPurgeMetadata(out io.Writer, preview purgePreview, deleted *bool) error {
	metadata := map[string]any{
		"execution_id":         preview.ExecutionID,
		"artifact_files":       preview.ArtifactFiles,
		"artifact_bytes":       preview.ArtifactBytes,
		"artifact_sha256":      preview.ArtifactSHA256,
		"artifact_path_sha256": preview.ArtifactPathSHA256,
		"status":               preview.Status,
		"ready":                preview.Ready,
	}
	if deleted != nil {
		metadata["deleted"] = *deleted
	}
	encoded, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, string(encoded))
	return err
}
