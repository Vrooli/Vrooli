package playbooks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"browser-automation-studio/cli/internal/appctx"
	internalplaybooks "browser-automation-studio/cli/internal/playbooks"
	"browser-automation-studio/cli/internal/util"
)

func runScaffold(ctx *appctx.Context, args []string) error {
	scenarioDir := ctx.ScenarioRoot
	resetMode := "project"
	description := ""
	positionals := []string{}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--scenario":
			if i+1 >= len(args) {
				return fmt.Errorf("--scenario requires a path")
			}
			scenarioDir = args[i+1]
			i++
		case "--reset":
			if i+1 >= len(args) {
				return fmt.Errorf("--reset requires a value")
			}
			resetMode = args[i+1]
			i++
		case "--description":
			if i+1 >= len(args) {
				return fmt.Errorf("--description requires a value")
			}
			description = args[i+1]
			i++
		case "--help", "-h":
			fmt.Println("Usage: browser-automation-studio playbooks scaffold <folder> <name> [--reset <mode>] [--description <text>]")
			fmt.Println("")
			fmt.Println("Positional arguments:")
			fmt.Println("  folder   Path under test/playbooks (e.g., capabilities/01-foundation/projects)")
			fmt.Println("  name     Human-friendly workflow name used for metadata/slug")
			fmt.Println("")
			fmt.Println("Options:")
			fmt.Println("  --reset <mode>       Reset mode (none|project|global). Defaults to project.")
			fmt.Println("  --description <text> Optional description, defaults to the workflow name.")
			fmt.Println("  --scenario <dir>     Override scenario directory (defaults to CLI root).")
			return nil
		case "--":
			positionals = append(positionals, args[i+1:]...)
			i = len(args)
		default:
			if strings.HasPrefix(args[i], "--") {
				return fmt.Errorf("unknown option: %s", args[i])
			}
			positionals = append(positionals, args[i])
		}
	}

	if len(positionals) < 2 {
		return fmt.Errorf("provide a folder and workflow name")
	}

	if scenarioDir == "" {
		return fmt.Errorf("scenario root not resolved")
	}

	folder, err := util.CleanPlaybookFolder(positionals[0])
	if err != nil {
		return err
	}
	workflowName := strings.TrimSpace(positionals[1])
	if workflowName == "" {
		return fmt.Errorf("workflow name must not be empty")
	}

	normalizedReset := strings.ToLower(strings.TrimSpace(resetMode))
	switch normalizedReset {
	case "none", "project", "global":
		resetMode = normalizedReset
	default:
		fmt.Printf("Warning: unknown reset mode '%s', defaulting to project\n", resetMode)
		resetMode = "project"
	}

	slug := util.Slugify(workflowName)
	baseDir := filepath.Join(scenarioDir, "test", "playbooks")
	targetDir := filepath.Join(baseDir, filepath.FromSlash(folder))
	targetFile := filepath.Join(targetDir, slug+".json")

	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("create playbook folder: %w", err)
	}
	if _, err := os.Stat(targetFile); err == nil {
		return fmt.Errorf("%s already exists", targetFile)
	}

	templateDesc := description
	if strings.TrimSpace(templateDesc) == "" {
		templateDesc = workflowName
	}

	payload, err := internalplaybooks.BuildScaffoldTemplate(workflowName, templateDesc, resetMode)
	if err != nil {
		return fmt.Errorf("build workflow template: %w", err)
	}

	if err := os.WriteFile(targetFile, payload, 0o644); err != nil {
		return fmt.Errorf("write workflow template: %w", err)
	}

	fmt.Printf("OK: Created %s\n", targetFile)
	fmt.Println("Next steps:")
	fmt.Println("  * Edit the JSON to add real nodes and selectors")
	fmt.Printf("  * Link the workflow from requirements/*.json\n")
	fmt.Printf("  * Regenerate the registry (node scripts/.../build-registry.mjs --scenario %s)\n", scenarioDir)
	return nil
}
