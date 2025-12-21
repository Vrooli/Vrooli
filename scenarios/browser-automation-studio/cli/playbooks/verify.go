package playbooks

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"browser-automation-studio/cli/internal/appctx"
)

var prefixPattern = regexp.MustCompile(`^[0-9]{2}-`)

func runVerify(ctx *appctx.Context, args []string) error {
	scenarioDir := ctx.ScenarioRoot

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--scenario":
			if i+1 >= len(args) {
				return fmt.Errorf("--scenario requires a path")
			}
			scenarioDir = args[i+1]
			i++
		case "--help", "-h":
			fmt.Println("Usage: browser-automation-studio playbooks verify [--scenario <dir>]")
			fmt.Println("Checks that every directory under bas/cases (excluding __*) begins with a two-digit prefix")
			return nil
		default:
			return fmt.Errorf("unknown option: %s", args[i])
		}
	}

	if scenarioDir == "" {
		return fmt.Errorf("scenario root not resolved")
	}

	baseDir := filepath.Join(scenarioDir, "bas", "cases")
	if info, err := os.Stat(baseDir); err != nil || !info.IsDir() {
		return fmt.Errorf("cases directory not found at %s", baseDir)
	}

	issues := 0
	var verifyTree func(root, rel string) error
	verifyTree = func(root, rel string) error {
		entries, err := os.ReadDir(root)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			name := entry.Name()
			if len(name) >= 2 && name[:2] == "__" {
				continue
			}
			fullPath := filepath.Join(root, name)
			if !prefixPattern.MatchString(name) {
				fmt.Printf("WARN %s%s is missing a two-digit prefix\n", rel, name)
				issues++
			}
			if err := verifyTree(fullPath, rel+name+"/"); err != nil {
				return err
			}
		}
		return nil
	}

	if err := verifyTree(baseDir, ""); err != nil {
		return err
	}

	if issues == 0 {
		fmt.Println("OK: All playbook folders include numeric prefixes")
		return nil
	}

	return fmt.Errorf("%d folder(s) missing numeric prefixes", issues)
}
