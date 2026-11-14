package resourcehandlers

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cli/resourcecli"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/resources"
	"github.com/vrooli/vrooli/internal/runtimesupervisor"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

func resourceUpdateByOwnerHandler[C any](deps rootcli.HandlerDeps[C], readOnly bool) rootcli.ResourceHandler[C] {
	return func(ctx C, controller *resources.Controller, args []string) error {
		var spec commandtree.Spec[resourcecli.CommandID]
		for _, s := range resourcecli.CommandSpecs() {
			if s.Handler == resourcecli.CommandUpdate {
				spec = s
				break
			}
		}
		commandID := resourcecli.CommandUpdate
		if readOnly {
			commandID = resourcecli.CommandUpdateCheck
			for _, s := range resourcecli.CommandSpecs() {
				if s.Handler == commandID {
					spec = s
					break
				}
			}
		}
		p, err := commandtree.ParseArgs("resource "+string(commandID), resourcecli.CommandHelpText(commandID), spec.Args, args)
		if err != nil {
			if commandtree.HandleHelp(deps.Stdout(ctx), err) {
				return nil
			}
			return err
		}
		if len(p.Positionals) != 1 {
			return rootcli.UsageErrorf("resource update", "requires one resource name")
		}
		name := p.Positionals[0]
		managed := false
		if request := p.FlagValue("--request"); request != "" {
			_, e := resources.LookupManagedServiceUpdate(name, request)
			if e == nil {
				managed = true
			} else if !errors.Is(e, resources.ErrManagedUpdateNotFound) {
				return e
			}
		}
		if !managed && filepath.Base(name) == name && name != "." && name != ".." {
			path := filepath.Join(controller.Root, "resources", name, "resource.json")
			if _, e := os.Stat(path); e == nil {
				m, e := controller.LoadManifest(path)
				if e != nil {
					return e
				}
				managed = m.Driver == "managed-service"
			} else if !os.IsNotExist(e) {
				return e
			}
		}
		if !managed {
			parse := resourcecli.ParseUpdateRequest
			if readOnly {
				parse = resourcecli.ParseUpdateCheckRequest
			}
			return resourceUpdateHandler(deps, parse)(ctx, controller, args)
		}
		if p.HasFlag("--on-behalf-of") || p.HasFlag("--cleanup-plan") {
			return rootcli.UsageErrorf("resource update", "harness actor and cleanup flags do not apply to managed-service updates")
		}
		action := p.FlagValue("--action")
		if action == "" {
			action = "prepare"
		}
		if readOnly || p.HasFlag("--check") {
			if p.HasFlag("--action") {
				return rootcli.UsageErrorf("resource update", "--check and --action cannot be combined")
			}
			action = "status"
		}
		request := p.FlagValue("--request")
		if request == "" && action != "status" {
			return rootcli.UsageErrorf("resource update", "requires --request")
		}
		if action != "prepare" && action != "rollback" && (p.HasFlag("--retry") || p.HasFlag("--version") || p.HasFlag("--sha256")) {
			return rootcli.UsageErrorf("resource update", "release identity and --retry apply only to prepare")
		}
		if action != "rollback" && p.HasFlag("--rollback-of") {
			return rootcli.UsageErrorf("resource update", "--rollback-of applies only to rollback")
		}
		operation, cancel := context.WithTimeout(rootcli.ResolveOperationContext(deps, ctx), 4*time.Minute)
		defer cancel()
		var result resources.ManagedServiceUpdate
		switch action {
		case "status":
			if request == "" {
				inspection, e := controller.InspectManagedServiceUpdate(operation, name)
				if e != nil {
					return e
				}
				return cliout.WriteJSON(deps.Stdout(ctx), inspection)
			}
			result, err = resources.LookupManagedServiceUpdate(name, request)
		case "prepare", "rollback":
			if p.FlagValue("--version") == "" || p.FlagValue("--sha256") == "" {
				return rootcli.UsageErrorf("resource update", "prepare requires --version and --sha256")
			}
			if action == "rollback" && p.FlagValue("--rollback-of") == "" {
				return rootcli.UsageErrorf("resource update", "rollback requires --rollback-of and a new --request")
			}
			if err := requireManagedUpdateSupervisor(operation, controller.Home); err != nil {
				return err
			}
			result, _, err = controller.AdmitManagedServiceUpdate(operation, p.Positionals[0], resources.ManagedServiceUpdateAdmission{RequestID: request, ExpectedVersion: p.FlagValue("--version"), ExpectedSHA256: p.FlagValue("--sha256"), ExplicitRetry: p.HasFlag("--retry"), RollbackOf: p.FlagValue("--rollback-of")})
		case "resume", "abort":
			result, err = controller.AdvanceManagedServiceUpdate(operation, p.Positionals[0], request, action == "abort")
		default:
			return rootcli.UsageErrorf("resource update", "unknown action %q", action)
		}
		if result.ID != "" {
			return writeManagedUpdateCommandResult(deps.Stdout(ctx), action, result, err)
		}
		return err
	}
}

func writeManagedUpdateCommandResult(w io.Writer, action string, result resources.ManagedServiceUpdate, operationErr error) error {
	writeErr := cliout.WriteJSON(w, result)
	if action != "status" && result.Phase.Terminal() && result.Phase != resources.UpdateSucceeded && !(action == "abort" && result.Phase == resources.UpdateAborted) {
		return errors.Join(operationErr, writeErr, rootcli.ExitCodeError{Code: 1, Silent_: true})
	}
	return errors.Join(operationErr, writeErr)
}

func requireManagedUpdateSupervisor(ctx context.Context, home string) error {
	if runtimesupervisor.ModeFromEnv() == runtimesupervisor.ModeOff {
		return fmt.Errorf("managed update requires the native runtime supervisor; supervision is disabled")
	}
	cfg := runtimesupervisor.EnvConfig()
	cfg.HomeDir = home
	svc := runtimesupervisor.New(cfg)
	defer svc.Close()
	r, err := svc.Status(ctx)
	if err != nil {
		return err
	}
	if r.Status != scenarioruntime.SupervisorStatusRunning || !r.HeartbeatDeadlineAt.After(time.Now()) || buildinfo.Fingerprint == "" || r.BuildIdentity != buildinfo.Fingerprint {
		return fmt.Errorf("managed update recovery owner is absent, stale or running a different control-plane build; run vrooli runtime supervisor install --user")
	}
	if r.PID == nil || *r.PID <= 0 {
		return fmt.Errorf("managed update supervisor has no live process identity")
	}
	incarnation, err := process.Incarnation(*r.PID)
	if err != nil {
		return err
	}
	current, err := platform.OpenProcessExecutable(os.Getpid())
	if err != nil {
		return err
	}
	defer current.Close()
	owner, err := platform.OpenProcessExecutable(*r.PID)
	if err != nil {
		return err
	}
	defer owner.Close()
	a, b := sha256.New(), sha256.New()
	if _, err := io.Copy(a, current); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := io.Copy(b, owner); err != nil {
		return err
	}
	if fmt.Sprintf("%x", a.Sum(nil)) != fmt.Sprintf("%x", b.Sum(nil)) {
		return fmt.Errorf("managed update supervisor is executing a different control-plane image; run vrooli runtime supervisor install --user")
	}
	if after, err := process.Incarnation(*r.PID); err != nil || after != incarnation {
		return fmt.Errorf("managed update supervisor changed during image verification")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}
