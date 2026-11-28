package setup

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/vrooli/vrooli/internal/operatorstate"
	"github.com/vrooli/vrooli/internal/resources"
	resourcecatalog "github.com/vrooli/vrooli/internal/resources/catalog"
	vrooliruntime "github.com/vrooli/vrooli/internal/runtime"
)

const (
	resourcesEnabled = "enabled"
	resourcesNone    = "none"
)

// resourceSelection keeps the planned groups with their names so later setup
// phases can reuse the catalog snapshot that produced them.
type resourceSelection struct {
	names    []string
	required []string
	optional []string
}

func (s *setupService) maybeInstallResources(root, home string, opts Options, stdout, stderr io.Writer, onOperation ...func(string)) ([]string, error) {
	operation := func(label string) {}
	if len(onOperation) > 0 && onOperation[0] != nil {
		operation = onOperation[0]
	}
	selection := strings.TrimSpace(opts.Resources)
	if selection == "" {
		resolved, err := defaultSetupResourceSelection(opts.Context, root, home)
		if err != nil {
			return nil, err
		}
		opts.resolvedResources = &resolved
		selection = strings.Join(resolved.names, ",")
		if selection == "" {
			selection = resourcesNone
		}
	}
	if selection == resourcesNone {
		return nil, nil
	}

	controller := s.deps.resourceController(root, home)
	var names, requiredNames, optionalNames []string
	if selection == resourcesEnabled {
		var err error
		names, err = enabledResourceNames(root, home)
		if err != nil {
			return nil, err
		}
		requiredNames, optionalNames, err = partitionResourceNames(root, names)
		if err != nil {
			return nil, err
		}
	} else if opts.resolvedResources != nil {
		names = opts.resolvedResources.names
		requiredNames = opts.resolvedResources.required
		optionalNames = opts.resolvedResources.optional
	} else {
		for _, raw := range strings.Split(selection, ",") {
			name := strings.TrimSpace(raw)
			if name == "" {
				continue
			}
			names = append(names, name)
		}
		var err error
		requiredNames, optionalNames, err = partitionResourceNames(root, names)
		if err != nil {
			return nil, err
		}
	}
	if err := preflightDockerResources(root, home, requiredNames, opts); err != nil {
		return nil, err
	}
	return installSelectedResources(controller, names, optionalNames, stdout, stderr, operation, root, home)
}

// defaultSetupResourceSelection keeps raw setup focused on enabled resources
// that the project marks required. Optional capabilities remain available
// through explicit resource selection or `--resources=enabled`.
func defaultSetupResourceSelection(ctx context.Context, root, home string) (resourceSelection, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return resourceSelection{}, err
	}
	state, err := operatorstate.New(operatorstate.ConfigForOperator(root, home)).Load(ctx)
	if err != nil {
		return resourceSelection{}, fmt.Errorf("read existing operator resource choices: %w", err)
	}
	entries, err := resourcecatalog.New(root).ReadConfigEntriesWithState(state)
	if err != nil {
		return resourceSelection{}, fmt.Errorf("read resource choices: %w", err)
	}
	// Only resources explicitly enabled in operator state count as optional
	// selections. An absent choice retains the lightweight required-only default;
	// a stored false remains an explicit opt-out.
	selection := resourceSelection{names: make([]string, 0, len(entries))}
	for name, entry := range entries {
		choice, hasChoice := state.Resources[name]
		if hasChoice && choice.Enabled != nil {
			if !*choice.Enabled {
				continue
			}
		} else if !entry.Enabled || !entry.Required {
			continue
		}
		selection.names = append(selection.names, name)
	}
	slices.Sort(selection.names)
	selection.required, selection.optional = partitionResourceEntries(entries, selection.names)
	return selection, nil
}

func partitionResourceNames(root string, names []string) (required, optional []string, err error) {
	entries, err := resourcecatalog.New(root).ReadConfigEntries()
	if err != nil {
		return nil, nil, fmt.Errorf("read resource requirements: %w", err)
	}
	required, optional = partitionResourceEntries(entries, names)
	return required, optional, nil
}

func partitionResourceEntries(entries map[string]resourcecatalog.ConfigEntry, names []string) (required, optional []string) {
	for _, name := range names {
		if entries[name].Required {
			required = append(required, name)
		} else {
			optional = append(optional, name)
		}
	}
	return required, optional
}

func installSelectedResources(controller resourceRunner, names, optionalNames []string, stdout, stderr io.Writer, operation func(string), root, home string) ([]string, error) {
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
	ensureController := resources.NewController(root, home)
	for _, name := range names {
		manifest, err := ensureController.ResourceManifest(name)
		if err != nil || !manifest.Capabilities.SupportsEnsure {
			continue
		}
		operation("Ensuring resource " + name)
		if err := ensureController.RunResourceCLI(name, []string{"ensure"}, stdout, stderr); err != nil {
			_, _ = fmt.Fprintf(stderr, "[WARN]    Resource %s ensure failed: %v\n", name, err)
		}
	}
	return degraded, nil
}

func preflightDockerResources(root, home string, names []string, opts Options) error {
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
		case "managed-service":
			needsDocker = append(needsDocker, name)
		}
	}
	if len(needsDocker) == 0 {
		return nil
	}
	status, err := ensureContainerRuntimeFn("docker", vrooliruntime.EnsureOptions{
		Context: opts.Context, Environment: opts.Environment, SudoMode: opts.SudoMode, DryRun: opts.DryRun,
		AutoInstall: true, IncludeOptional: opts.IncludeOptional, MaintenanceWindow: opts.MaintenanceWindow,
	})
	if err != nil {
		return fmt.Errorf("selected resources require Docker (%s), but container-runtime setup failed: %w", strings.Join(needsDocker, ", "), err)
	}
	if status.Installed || opts.DryRun {
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

func enabledResourceNames(root, home string) ([]string, error) {
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
