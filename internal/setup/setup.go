package setup

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/lifecycle"
	vrooliruntime "github.com/vrooli/vrooli/internal/runtime"
	"github.com/vrooli/vrooli/internal/scenario"
)

const (
	defaultEnvironment = "development"
	defaultTarget      = "docker"
	defaultLocation    = "Local"
)

type options struct {
	DryRun      bool
	SudoMode    string
	Environment string
	Resources   string
	Yes         string
	Help        bool
}

func RunSetup(root, home string, args []string, stdout, stderr io.Writer) error {
	opts, err := parseOptions("setup", args)
	if err != nil {
		return err
	}
	if opts.Help {
		showSetupHelp(stdout)
		return nil
	}
	if err := vrooliruntime.Current().ValidateSetup(); err != nil {
		return err
	}

	project, err := loadProject(root)
	if err != nil {
		return err
	}

	restoreEnv, err := applyEnvironment(root, project.ServicePath, opts)
	if err != nil {
		return err
	}
	defer restoreEnv()

	runner, err := lifecycle.NewRunner(root, home, stdout, stderr)
	if err != nil {
		return err
	}
	if err := runner.RunPhase(project.Slug, "setup", lifecycle.PhaseOptions{CustomPath: root, ProjectMode: true}); err != nil {
		return err
	}
	return markComplete(root, project.Manifest)
}

func RunDevelop(root, home string, args []string, stdout, stderr io.Writer) error {
	opts, err := parseOptions("develop", args)
	if err != nil {
		return err
	}
	if opts.Help {
		showDevelopHelp(stdout)
		return nil
	}
	if err := vrooliruntime.Current().ValidateDevelop(); err != nil {
		return err
	}

	project, err := loadProject(root)
	if err != nil {
		return err
	}

	restoreEnv, err := applyEnvironment(root, project.ServicePath, opts)
	if err != nil {
		return err
	}
	defer restoreEnv()

	runner, err := lifecycle.NewRunner(root, home, stdout, stderr)
	if err != nil {
		return err
	}

	setupNeeded, reasons, err := runner.SetupNeeded(project, forceSetupApplies(project.Slug))
	if err != nil {
		return err
	}
	if setupNeeded {
		if len(reasons) > 0 {
			_, _ = fmt.Fprintf(stdout, "[INFO]    Running setup before develop (%s)\n", strings.Join(reasons, ", "))
		} else {
			_, _ = fmt.Fprintln(stdout, "[INFO]    Running setup before develop")
		}
		if err := runner.RunPhase(project.Slug, "setup", lifecycle.PhaseOptions{CustomPath: root, ProjectMode: true}); err != nil {
			return err
		}
		if err := markComplete(root, project.Manifest); err != nil {
			return err
		}
	}

	return runner.RunPhase(project.Slug, "develop", lifecycle.PhaseOptions{CustomPath: root, ProjectMode: true})
}

func parseOptions(command string, args []string) (options, error) {
	opts := options{}
	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch {
		case arg == "--help" || arg == "-h":
			opts.Help = true
		case arg == "--dry-run":
			opts.DryRun = true
		case arg == "--sudo-mode":
			value, next, err := requireValue(command, arg, args, index)
			if err != nil {
				return options{}, err
			}
			index = next
			value = strings.ToLower(strings.TrimSpace(value))
			switch value {
			case "ask", "skip", "error":
				opts.SudoMode = value
			default:
				return options{}, fmt.Errorf("invalid value for --sudo-mode: %s", value)
			}
		case strings.HasPrefix(arg, "--sudo-mode="):
			value := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(arg, "--sudo-mode=")))
			switch value {
			case "ask", "skip", "error":
				opts.SudoMode = value
			default:
				return options{}, fmt.Errorf("invalid value for --sudo-mode: %s", value)
			}
		case arg == "--environment" || arg == "--env":
			value, next, err := requireValue(command, arg, args, index)
			if err != nil {
				return options{}, err
			}
			index = next
			value = strings.ToLower(strings.TrimSpace(value))
			switch value {
			case "development", "production", "minimal":
				opts.Environment = value
			default:
				return options{}, fmt.Errorf("invalid value for --environment: %s", value)
			}
		case strings.HasPrefix(arg, "--environment="):
			value := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(arg, "--environment=")))
			switch value {
			case "development", "production", "minimal":
				opts.Environment = value
			default:
				return options{}, fmt.Errorf("invalid value for --environment: %s", value)
			}
		case strings.HasPrefix(arg, "--env="):
			value := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(arg, "--env=")))
			switch value {
			case "development", "production", "minimal":
				opts.Environment = value
			default:
				return options{}, fmt.Errorf("invalid value for --environment: %s", value)
			}
		case arg == "--resources":
			value, next, err := requireValue(command, arg, args, index)
			if err != nil {
				return options{}, err
			}
			index = next
			opts.Resources = strings.ToLower(strings.TrimSpace(value))
		case strings.HasPrefix(arg, "--resources="):
			opts.Resources = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(arg, "--resources=")))
		case arg == "--yes" || arg == "-y":
			value, next, err := requireValue(command, arg, args, index)
			if err != nil {
				return options{}, err
			}
			index = next
			opts.Yes = strings.ToLower(strings.TrimSpace(value))
		case strings.HasPrefix(arg, "--yes="):
			opts.Yes = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(arg, "--yes=")))
		default:
			return options{}, fmt.Errorf("unknown option for %s: %s", command, arg)
		}
	}
	return opts, nil
}

func requireValue(command, flag string, args []string, index int) (string, int, error) {
	if index+1 >= len(args) {
		return "", index, fmt.Errorf("%s requires a value for %s", command, flag)
	}
	return args[index+1], index + 1, nil
}

func showSetupHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Usage: vrooli setup [options]")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Options:")
	_, _ = fmt.Fprintln(w, "  --environment, --env <name>   Set environment profile (development|production|minimal)")
	_, _ = fmt.Fprintln(w, "  --resources <value>           Resource selection (enabled|none|comma,list)")
	_, _ = fmt.Fprintln(w, "  --sudo-mode <mode>            Sudo policy (ask|skip|error)")
	_, _ = fmt.Fprintln(w, "  --yes <value>                 Confirmation policy forwarded to setup steps")
	_, _ = fmt.Fprintln(w, "  --dry-run                     Export DRY_RUN=true for setup steps")
}

func showDevelopHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Usage: vrooli develop [options]")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Options:")
	_, _ = fmt.Fprintln(w, "  --environment, --env <name>   Set environment profile for auto-setup (development|production|minimal)")
	_, _ = fmt.Fprintln(w, "  --resources <value>           Resource selection for auto-setup (enabled|none|comma,list)")
	_, _ = fmt.Fprintln(w, "  --sudo-mode <mode>            Sudo policy for auto-setup (ask|skip|error)")
	_, _ = fmt.Fprintln(w, "  --yes <value>                 Confirmation policy forwarded to auto-setup")
	_, _ = fmt.Fprintln(w, "  --dry-run                     Export DRY_RUN=true for auto-setup")
}

func loadProject(root string) (scenario.Scenario, error) {
	servicePath := filepath.Join(root, ".vrooli", "service.json")
	manifest, err := scenario.ReadService(servicePath)
	if err != nil {
		return scenario.Scenario{}, fmt.Errorf("read project service manifest: %w", err)
	}
	slug := strings.TrimSpace(manifest.Service.Name)
	if slug == "" {
		slug = filepath.Base(root)
	}
	if slug == "" || slug == "." {
		slug = "vrooli-dev"
	}
	return scenario.Scenario{
		Slug:        slug,
		Path:        root,
		ServicePath: servicePath,
		Manifest:    manifest,
	}, nil
}

type envSnapshot struct {
	value   string
	existed bool
}

func applyEnvironment(root, servicePath string, opts options) (func(), error) {
	changes := map[string]envSnapshot{}
	set := func(key, value string, onlyIfUnset bool) error {
		current, existed := os.LookupEnv(key)
		if onlyIfUnset && existed && strings.TrimSpace(current) != "" {
			return nil
		}
		if _, tracked := changes[key]; !tracked {
			changes[key] = envSnapshot{value: current, existed: existed}
		}
		return os.Setenv(key, value)
	}

	if err := set("APP_ROOT", root, false); err != nil {
		return nil, err
	}
	if err := set("SERVICE_JSON_PATH", servicePath, false); err != nil {
		return nil, err
	}
	if err := set("TARGET", defaultTarget, true); err != nil {
		return nil, err
	}
	if err := set("LOCATION", defaultLocation, true); err != nil {
		return nil, err
	}
	if opts.Environment != "" {
		if err := set("ENVIRONMENT", opts.Environment, false); err != nil {
			return nil, err
		}
	} else if err := set("ENVIRONMENT", defaultEnvironment, true); err != nil {
		return nil, err
	}
	if opts.Resources != "" {
		if err := set("RESOURCES", opts.Resources, false); err != nil {
			return nil, err
		}
	}
	if opts.Yes != "" {
		if err := set("YES", opts.Yes, false); err != nil {
			return nil, err
		}
	}
	if opts.SudoMode != "" {
		if err := set("SUDO_MODE", opts.SudoMode, false); err != nil {
			return nil, err
		}
		if err := set("SUDO_MODE_EXPLICIT", opts.SudoMode, false); err != nil {
			return nil, err
		}
	}
	if opts.DryRun {
		if err := set("DRY_RUN", "true", false); err != nil {
			return nil, err
		}
	}

	return func() {
		for key, snapshot := range changes {
			if snapshot.existed {
				_ = os.Setenv(key, snapshot.value)
				continue
			}
			_ = os.Unsetenv(key)
		}
	}, nil
}

func forceSetupApplies(slug string) bool {
	if strings.ToLower(strings.TrimSpace(os.Getenv("FORCE_SETUP"))) != "true" {
		return false
	}
	target := strings.TrimSpace(os.Getenv("FORCE_SETUP_SCENARIO"))
	return target == "" || target == slug
}

func markComplete(root string, manifest scenario.ServiceManifest) error {
	dataDir := filepath.Join(root, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}

	stepNames := make([]string, 0, len(manifest.Lifecycle.Setup.Steps))
	resourceMarker := false
	for _, step := range manifest.Lifecycle.Setup.Steps {
		name := strings.TrimSpace(step.Name)
		if name == "" {
			name = "unnamed"
		}
		stepNames = append(stepNames, name)
		if name == "populate-resources" || name == "add-data" {
			resourceMarker = true
		}
	}

	payload := map[string]any{
		"setup_version":   "2.0.0",
		"completed_at":    time.Now().Format(time.RFC3339),
		"steps_completed": stepNames,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(dataDir, ".setup-complete"), data, 0o644); err != nil {
		return err
	}
	if resourceMarker {
		if err := os.WriteFile(filepath.Join(dataDir, ".resources-populated"), []byte{}, 0o644); err != nil {
			return err
		}
	}
	return nil
}
