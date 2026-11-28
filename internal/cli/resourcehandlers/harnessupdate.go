package resourcehandlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/internal/cli/resourcecli"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/resources"
	"github.com/vrooli/vrooli/packages/harness-lifecycle"
)

// harnessActionTimeout covers a governed npm install (the gateway allows three
// minutes), its approval, and the checks before and after it.
const harnessActionTimeout = 6 * time.Minute

func resourceUpdateHandler[C any](deps rootcli.HandlerDeps[C], parse func([]string) (resourcecli.UpdateRequest, error)) rootcli.ResourceHandler[C] {
	return func(ctx C, controller *resources.Controller, args []string) error {
		format, err := deps.OutputFormat(ctx)
		if err != nil {
			return err
		}
		parseMapped := func(args []string) (resourcecli.UpdateRequest, error) {
			req, err := parse(args)
			return req, mapResourceParseError("resource update", err)
		}
		return rootcli.BindResourceCommand(deps.Stdout, deps.OutputFormat, parseMapped,
			func(ctx C, controller *resources.Controller, req resourcecli.UpdateRequest) (agentinstall.HarnessActionResult, error) {
				if _, err := cliutil.CodingAgentBinary(req.Name); err != nil {
					return agentinstall.HarnessActionResult{}, rootcli.UsageErrorf("resource update", "unknown coding-agent resource %q (known: %s)", req.Name, knownCodingAgentResourceNames())
				}
				operation, cancel := context.WithTimeout(rootcli.ResolveOperationContext(deps, ctx), harnessActionTimeout)
				defer cancel()
				return runHarnessUpdate(operation, controller.Root, req)
			},
			func(w io.Writer, _ cliout.Format, result agentinstall.HarnessActionResult) error {
				if err := resourcecli.WriteHarnessUpdate(w, format, result); err != nil {
					return err
				}
				if !result.Success {
					// The result is already rendered; only the exit status is left to say.
					return rootcli.ExitCodeError{Code: 1, Silent_: true}
				}
				return nil
			},
		)(ctx, controller, args)
	}
}

func runHarnessUpdate(ctx context.Context, root string, req resourcecli.UpdateRequest) (agentinstall.HarnessActionResult, error) {
	channel := req.Channel
	if req.Major > 0 {
		resolved, err := agentinstall.ChannelForMajor(root, req.Name, req.Major)
		if err != nil {
			return agentinstall.HarnessActionResult{Action: req.Action, Message: err.Error()}, nil
		}
		channel = resolved
	}
	result, err := agentinstall.ManageHarnessWithChannel(ctx, root, req.Name, channel, req.Action, cliutil.ResolveCallerActor(req.OnBehalfOf), req.CleanupPlanID)
	if err != nil || !result.Success {
		return result, err
	}
	if req.Action == agentinstall.ActionUpdate || req.Action == agentinstall.ActionInstall {
		if warning := reconcileAfterHarnessUpdate(ctx, root, req.Name); warning != "" {
			result.Success = false
			result.Message = strings.TrimSpace(result.Message + " " + warning)
		}
	}
	return result, nil
}

// reconcileAfterHarnessUpdate runs the resource's idempotent ensure step after
// a successful install/update. The binary is already selected; ensure is what
// projects config, credentials, and the major-appropriate policy plugin. A
// failure is surfaced as a failed command rather than hidden, because leaving
// the previous major's plugin in place would misrepresent enforcement.
func reconcileAfterHarnessUpdate(ctx context.Context, root, name string) string {
	data, err := os.ReadFile(filepath.Join(root, "resources", name, "resource.json"))
	if err != nil {
		return ""
	}
	var manifest struct {
		Capabilities struct {
			SupportsEnsure bool `json:"supports_ensure"`
		} `json:"capabilities"`
		CLI struct {
			Command string `json:"command"`
		} `json:"cli"`
	}
	if json.Unmarshal(data, &manifest) != nil || !manifest.Capabilities.SupportsEnsure {
		return ""
	}
	command := strings.TrimSpace(manifest.CLI.Command)
	if command == "" {
		return ""
	}
	path, err := exec.LookPath(command)
	if err != nil {
		return fmt.Sprintf("Updated, but %s is not on PATH; configuration and policy-plugin reconciliation did not run.", command)
	}
	var output bytes.Buffer
	cmd := exec.CommandContext(ctx, path, "ensure")
	cmd.Stdout, cmd.Stderr = &output, &output
	if err := cmd.Run(); err != nil {
		return fmt.Sprintf("Updated, but %s ensure failed: %v: %s", command, err, strings.TrimSpace(output.String()))
	}
	return ""
}
