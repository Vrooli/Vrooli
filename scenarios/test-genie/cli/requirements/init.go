package requirements

import (
	"bytes"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

//go:embed templates/modular/*
var templateFS embed.FS

func runInit(args []string) error {
	fs := flag.NewFlagSet("requirements init", flag.ContinueOnError)
	dirFlag, scenarioFlag := parseCommonFlags(fs)
	force := fs.Bool("force", false, "Overwrite existing requirements directory")
	templateName := fs.String("template", "modular", "Template to use (only 'modular' supported)")
	owner := fs.String("owner", "", "Owner contact (defaults to engineering@<scenario>.local)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	dir, err := resolveDir(*dirFlag)
	if err != nil {
		return err
	}
	if err := ensureDir(dir); err != nil {
		return err
	}

	if strings.ToLower(strings.TrimSpace(*templateName)) != "modular" {
		return fmt.Errorf("unsupported template: %s (only 'modular' is available)", *templateName)
	}

	scenario := scenarioNameFromDir(dir, *scenarioFlag)
	if scenario == "" {
		return fmt.Errorf("unable to determine scenario name; pass --scenario")
	}

	targetDir := filepath.Join(dir, "requirements")
	if _, err := os.Stat(targetDir); err == nil && !*force {
		return fmt.Errorf("%s already exists; re-run with --force to overwrite", targetDir)
	}

	if err := os.RemoveAll(targetDir); err != nil {
		return fmt.Errorf("failed to clear existing requirements dir: %w", err)
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("failed to create requirements dir: %w", err)
	}

	if err := copyTemplate(targetDir, scenario, *owner); err != nil {
		return err
	}

	fmt.Printf("✅ Initialized requirements/ registry for '%s' using '%s' template\n", scenario, *templateName)
	fmt.Println("Next steps:")
	fmt.Println("  • Tag tests with [REQ:ID] and run your suite so auto-sync can update statuses")
	fmt.Println("  • See scenarios/test-genie/docs/phases/business/requirements-sync.md for schema details")
	return nil
}

func copyTemplate(targetDir, scenario, owner string) error {
	generated := time.Now().Format("2006-01-02")
	contact := owner
	if strings.TrimSpace(contact) == "" {
		contact = fmt.Sprintf("engineering@%s.local", scenario)
	}

	return fs.WalkDir(templateFS, "templates/modular", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel("templates/modular", path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		dest := filepath.Join(targetDir, rel)
		if d.IsDir() {
			return os.MkdirAll(dest, 0o755)
		}

		content, err := fs.ReadFile(templateFS, path)
		if err != nil {
			return fmt.Errorf("read template %s: %w", path, err)
		}

		updated := applyPlaceholders(content, scenario, generated, contact)
		if err := os.WriteFile(dest, updated, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", dest, err)
		}
		return nil
	})
}

func applyPlaceholders(content []byte, scenario, generated, contact string) []byte {
	replacer := strings.NewReplacer(
		"__SCENARIO_NAME__", scenario,
		"__GENERATED_DATE__", generated,
		"__CONTACT__", contact,
	)
	return []byte(replacer.Replace(string(bytes.TrimSpace(content)) + "\n"))
}
