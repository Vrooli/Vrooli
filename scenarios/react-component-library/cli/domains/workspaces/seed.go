package workspaces

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace"

	"github.com/vrooli/cli-core/cliapp"

	"react-component-library/cli/internal/scenariopath"
)

// seedFile mirrors library/workspace-seeds.json. Entries stay raw so an export
// rewrites one entry and leaves the reviewed content of every other template
// byte-identical.
type seedFile struct {
	SchemaVersion int               `json:"schemaVersion"`
	SeedVersion   int               `json:"seedVersion"`
	Description   string            `json:"description"`
	Workspaces    []json.RawMessage `json:"workspaces"`
}

// exportSeedCall promotes one workspace into the reviewed template shelf.
//
// A template is not a separate authoring model: it is an ordinary workspace
// aggregate that was built in the studio, then marked curated at review time.
// The command only changes the four fields that make a snapshot curated (id,
// provenance, revision and timestamps) and writes it into the seed file. The
// loader takes it from there, so the same composition renders in the studio
// and on the shelf.
func (h *handlers) exportSeedCall(ctx cliapp.RunContext) error {
	id := strings.TrimSpace(ctx.Positional("id"))
	if id == "" {
		return errors.New("workspace id is required")
	}
	resp, err := h.client.GetWorkspace(context.Background(), connect.NewRequest(&workspacev1.GetWorkspaceRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get workspace %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetWorkspace() == nil {
		return fmt.Errorf("server returned no workspace for %q", id)
	}
	workspace := resp.Msg.GetWorkspace()
	if workspace.GetProvenance() == workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_CURATED {
		return fmt.Errorf("%q is already a curated template; fork it, edit the fork, then export that", id)
	}
	if err := validateSeedSnapshot(workspace); err != nil {
		return fmt.Errorf("%q cannot become a template: %w", id, err)
	}

	seedID := strings.TrimSpace(ctx.Flag("seed-id"))
	if seedID == "" {
		if strings.HasPrefix(workspace.GetId(), "ws_curated_") {
			seedID = workspace.GetId()
		} else {
			return errors.New("--seed-id is required for an authored or forked workspace (for example --seed-id ws_curated_harbor)")
		}
	}
	if !strings.HasPrefix(seedID, "ws_curated_") {
		return errors.New("--seed-id must start with ws_curated_ so the shelf's templates stay recognisable")
	}

	path, err := seedPath(ctx.Flag("file"))
	if err != nil {
		return err
	}
	rawFile, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	var payload seedFile
	if err := json.Unmarshal(rawFile, &payload); err != nil {
		return fmt.Errorf("%s is not valid JSON: %w", path, err)
	}
	if payload.SchemaVersion != 1 {
		return fmt.Errorf("%s declares schemaVersion %d; this command writes schemaVersion 1", path, payload.SchemaVersion)
	}

	snapshot := proto.Clone(workspace).(*workspacev1.Workspace)
	snapshot.Id = seedID
	snapshot.Provenance = workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_CURATED
	snapshot.Revision = 1
	snapshot.OriginWorkspaceId = ""
	snapshot.BindingScenario = ""
	snapshot.BindingPage = ""
	now := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	snapshot.CreatedAt = now
	snapshot.UpdatedAt = now
	encoded, err := protojson.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("encode workspace snapshot: %w", err)
	}

	replaced := false
	for index, entry := range payload.Workspaces {
		var identity struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(entry, &identity); err != nil {
			return fmt.Errorf("%s entry %d has no readable id: %w", path, index, err)
		}
		if identity.ID == seedID {
			payload.Workspaces[index] = json.RawMessage(encoded)
			replaced = true
			break
		}
	}
	if !replaced {
		payload.Workspaces = append(payload.Workspaces, json.RawMessage(encoded))
	}
	if payload.SeedVersion < 1 {
		payload.SeedVersion = 1
	}
	payload.SeedVersion++

	out, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("encode seed file: %w", err)
	}
	out = append(out, '\n')
	if err := writeAtomic(path, out); err != nil {
		return err
	}

	verb := "Added"
	if replaced {
		verb = "Updated"
	}
	return ctx.RenderMutation(cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("%s %q as %s at seedVersion %d.", verb, workspace.GetName(), seedID, payload.SeedVersion),
			fmt.Sprintf("Wrote %d templates to %s. Nothing was restarted.", len(payload.Workspaces), path),
		},
		Changes: []string{
			"provenance → curated; revision → 1; binding cleared",
			"the composition, pages, displays and states are unchanged",
		},
		NextCommand: []string{
			"review the diff and commit library/workspace-seeds.json",
			"restart the scenario to apply the reviewed seed version",
		},
	})
}

func seedPath(flag string) (string, error) {
	root, err := scenariopath.Find()
	if err != nil {
		return "", err
	}
	if value := strings.TrimSpace(flag); value != "" {
		if filepath.IsAbs(value) {
			return value, nil
		}
		return filepath.Join(root, value), nil
	}
	return filepath.Join(root, "library", "workspace-seeds.json"), nil
}

// writeAtomic writes beside the target and renames, so an interrupted export
// never leaves a half-written reviewed file.
func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, ".workspace-seeds-*.json")
	if err != nil {
		return fmt.Errorf("create temporary seed file: %w", err)
	}
	name := temp.Name()
	defer func() { _ = os.Remove(name) }()
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return fmt.Errorf("write temporary seed file: %w", err)
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return fmt.Errorf("sync temporary seed file: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close temporary seed file: %w", err)
	}
	if err := os.Chmod(name, 0o644); err != nil {
		return fmt.Errorf("mode temporary seed file: %w", err)
	}
	if err := os.Rename(name, path); err != nil {
		return fmt.Errorf("replace %s: %w", path, err)
	}
	return nil
}

// validateSeedSnapshot enforces the two composition rules a reviewed template
// must satisfy: it has a real composition, and every layer id is unique across
// the workspace. The server stores the spec as opaque JSON, so this is the
// point where the promoted snapshot is checked against the studio's own
// address rule. A template the studio cannot address is not a template.
func validateSeedSnapshot(workspace *workspacev1.Workspace) error {
	if len(workspace.GetPages()) == 0 {
		return errors.New("it has no pages")
	}
	seen := make(map[string]string)
	composed := false
	for pageIndex, page := range workspace.GetPages() {
		for displayIndex, display := range page.GetDisplays() {
			if len(display.GetStates()) == 0 {
				return fmt.Errorf("page %d display %d has no states", pageIndex, displayIndex)
			}
			for stateIndex, state := range display.GetStates() {
				raw := strings.TrimSpace(state.GetArgsJson())
				if raw == "" || raw == "{}" || raw == "null" {
					continue
				}
				var payload struct {
					Spec json.RawMessage `json:"spec"`
				}
				if err := json.Unmarshal([]byte(raw), &payload); err != nil {
					return fmt.Errorf("page %d display %d state %d has unreadable args: %w", pageIndex, displayIndex, stateIndex, err)
				}
				if len(payload.Spec) == 0 {
					continue
				}
				path := fmt.Sprintf("%s/%s/%s", page.GetId(), display.GetId(), state.GetId())
				if err := collectLayerIDs(payload.Spec, path, seen); err != nil {
					return err
				}
				composed = true
			}
		}
	}
	if !composed {
		return errors.New("none of its states has a composition")
	}
	return nil
}

func collectLayerIDs(raw json.RawMessage, path string, seen map[string]string) error {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return fmt.Errorf("%s has unreadable composition JSON: %w", path, err)
	}
	var walk func(any) error
	walk = func(node any) error {
		switch typed := node.(type) {
		case map[string]any:
			// Only a node-shaped object owns a layer address. A prop that
			// happens to be named "id" is data, not a layer.
			if id, ok := typed["id"].(string); ok && strings.TrimSpace(id) != "" {
				if _, nodeShaped := typed["type"].(string); nodeShaped {
					if first, exists := seen[id]; exists {
						return fmt.Errorf("layer id %q is used at both %s and %s", id, first, path)
					}
					seen[id] = path
				}
			}
			for _, child := range typed {
				if err := walk(child); err != nil {
					return err
				}
			}
		case []any:
			for _, child := range typed {
				if err := walk(child); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return walk(value)
}
