package setup

import (
	"fmt"
	"io"
	"strings"

	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/dockerhost"
	"github.com/vrooli/vrooli/internal/resources"
	vrooliruntime "github.com/vrooli/vrooli/internal/runtime"
)

func (s *setupService) maybeInstallResources(root, home string, opts Options, stdout, stderr io.Writer, onOperation ...func(string)) ([]string, error) {
	operation := func(label string) {}
	if len(onOperation) > 0 && onOperation[0] != nil {
		operation = onOperation[0]
	}
	selection := strings.TrimSpace(opts.Resources)
	if selection == "" {
		selection = "enabled"
	}
	if selection == "none" {
		return nil, nil
	}

	controller := s.deps.resourceController(root, home)
	if selection == "enabled" {
		names, err := enabledResourceNames(root)
		if err != nil {
			return nil, err
		}
		requiredNames, optionalNames, err := partitionResourceNames(root, home, names)
		if err != nil {
			return nil, err
		}
		if err := preflightDockerResources(root, home, requiredNames, opts); err != nil {
			return nil, err
		}
		return installSelectedResources(controller, names, optionalNames, stdout, stderr, operation)
	}

	names := []string{}
	for _, raw := range strings.Split(selection, ",") {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		names = append(names, name)
	}
	requiredNames, optionalNames, err := partitionResourceNames(root, home, names)
	if err != nil {
		return nil, err
	}
	if err := preflightDockerResources(root, home, requiredNames, opts); err != nil {
		return nil, err
	}
	return installSelectedResources(controller, names, optionalNames, stdout, stderr, operation)
}

func partitionResourceNames(root, home string, names []string) (required, optional []string, err error) {
	config := resources.NewController(root, home)
	for _, name := range names {
		isRequired, err := config.ResourceRequired(name)
		if err != nil {
			return nil, nil, fmt.Errorf("read resource requirement %s: %w", name, err)
		}
		if isRequired {
			required = append(required, name)
		} else {
			optional = append(optional, name)
		}
	}
	return required, optional, nil
}

func installSelectedResources(controller resourceRunner, names, optionalNames []string, stdout, stderr io.Writer, operation func(string)) ([]string, error) {
	optional := make(map[string]struct{}, len(optionalNames))
	for _, name := range optionalNames {
		optional[name] = struct{}{}
	}
	degraded := []string{}
	for _, name := range names {
		operation("Installing resource " + name)
		if err := controller.Run(name, []string{"install"}, stdout, stderr); err != nil {
			if _, isOptional := optional[name]; isOptional {
				degraded = append(degraded, name)
				_, _ = fmt.Fprintf(stderr, "[WARN]    Optional resource %s unavailable; continuing setup: %v\n", name, err)
				continue
			}
			return degraded, err
		}
	}
	return degraded, nil
}

func preflightDockerResources(root, home string, names []string, setupOpts ...Options) error {
	if len(names) == 0 {
		return nil
	}
	controller := resources.NewController(root, home)
	needsDocker := []string{}
	for _, name := range names {
		manifest, err := controller.ResourceManifest(name)
		if err != nil {
			continue
		}
		switch strings.TrimSpace(manifest.Driver) {
		case "docker-service", "compose-service":
			needsDocker = append(needsDocker, name)
		}
	}
	if len(needsDocker) == 0 {
		return nil
	}
	if len(setupOpts) == 0 {
		// Keep the narrow helper seam used by callers that only want to inspect
		// readiness. The setup apply path below supplies Options and performs
		// the provider ladder's repair/provisioning step.
		health := inspectDockerHealthFn()
		if health.InfoOK {
			return nil
		}
		detail := strings.TrimSpace(health.Detail)
		if detail == "" {
			detail = "Docker daemon is not reachable"
		}
		return fmt.Errorf("selected resources require Docker (%s), but Docker is not healthy: %s", strings.Join(needsDocker, ", "), dockerhost.DiagnosticLine(detail))
	}
	applyOpts := setupOpts[0]
	status, err := ensureContainerRuntimeFn("docker", vrooliruntime.EnsureOptions{
		Environment: applyOpts.Environment, SudoMode: applyOpts.SudoMode, DryRun: applyOpts.DryRun,
		AutoInstall: true, IncludeOptional: applyOpts.IncludeOptional, MaintenanceWindow: applyOpts.MaintenanceWindow,
	})
	if err != nil {
		return fmt.Errorf("selected resources require Docker (%s), but container-runtime setup failed: %w", strings.Join(needsDocker, ", "), err)
	}
	if status.Installed || applyOpts.DryRun {
		return nil
	}
	detail := "container runtime is not ready"
	if len(status.Notes) > 0 {
		detail = status.Notes[len(status.Notes)-1]
	}
	return fmt.Errorf("selected resources require Docker (%s), but container-runtime setup did not complete (provider=%s, state=%s): %s", strings.Join(needsDocker, ", "), status.SelectedProvider, status.ExecutionState, detail)
}

type resourceRunner interface {
	Run(name string, args []string, stdout, stderr io.Writer) error
}

func enabledResourceNames(root string) ([]string, error) {
	home, err := config.HomeDir()
	if err != nil {
		return nil, err
	}
	return resources.NewController(root, home).EnabledResourceNames()
}

func syncResourceSchemaArtifacts(root string) error {
	report, err := resources.SyncSchemaArtifacts(root)
	if err != nil {
		return err
	}
	if report.Passed {
		return nil
	}
	if len(report.MissingReferences) > 0 {
		first := report.MissingReferences[0]
		return fmt.Errorf("resource schema sync failed: scenario %s references missing resource %s", first.Scenario, first.Resource)
	}
	return fmt.Errorf("resource schema sync failed")
}
