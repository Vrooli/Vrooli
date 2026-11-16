package hub

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

const maxRenderedKitPageBytes = 3 << 20

var kitTemplateNameRule = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// RenderKitTemplate validates a data object against the published kit schema,
// then embeds it in the template's single inert JSON slot. Kit files remain
// owned by the Notification Hub skill and are read from the scenario install.
func RenderKitTemplate(name string, data map[string]any) (string, error) {
	if !kitTemplateNameRule.MatchString(name) {
		return "", errors.New("template name must be lowercase and hyphenated")
	}
	if data == nil {
		return "", errors.New("template data must be a JSON object")
	}
	kitDir, err := notificationHubKitDir()
	if err != nil {
		return "", err
	}
	templatePath := filepath.Join(kitDir, "templates", name+".html")
	schemaPath := filepath.Join(kitDir, "schemas", name+".schema.json")
	template, err := os.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("kit template %q is unavailable: %w", name, err)
	}
	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		return "", fmt.Errorf("kit schema %q is unavailable: %w", name, err)
	}
	encoded, err := json.Marshal(data)
	if err != nil || len(encoded) > 2<<20 {
		return "", errors.New("template data must be JSON and no larger than 2 MiB")
	}
	compiler := jsonschema.NewCompiler()
	schemaURL := "urn:notification-hub:kit:" + name
	if err := compiler.AddResource(schemaURL, bytes.NewReader(schemaBytes)); err != nil {
		return "", fmt.Errorf("load kit schema %q: %w", name, err)
	}
	schema, err := compiler.Compile(schemaURL)
	if err != nil {
		return "", fmt.Errorf("compile kit schema %q: %w", name, err)
	}
	var instance any
	if err := json.Unmarshal(encoded, &instance); err != nil {
		return "", fmt.Errorf("decode template data: %w", err)
	}
	if err := schema.Validate(instance); err != nil {
		return "", fmt.Errorf("template data does not match %q: %w", name, err)
	}
	open := []byte(`<script type="application/json" id="hub-data">`)
	close := []byte(`</script>`)
	if bytes.Count(template, open) != 1 {
		return "", fmt.Errorf("kit template %q must have exactly one data slot", name)
	}
	start := bytes.Index(template, open) + len(open)
	endRelative := bytes.Index(template[start:], close)
	if endRelative < 0 {
		return "", fmt.Errorf("kit template %q has an incomplete data slot", name)
	}
	end := start + endRelative
	page := make([]byte, 0, len(template)-(end-start)+len(encoded))
	page = append(page, template[:start]...)
	page = append(page, encoded...)
	page = append(page, template[end:]...)
	if len(page) > maxRenderedKitPageBytes {
		return "", fmt.Errorf("rendered kit template %q exceeds %d MiB", name, maxRenderedKitPageBytes>>20)
	}
	return string(page), nil
}

func notificationHubKitDir() (string, error) {
	if scenarioDir := strings.TrimSpace(os.Getenv("VROOLI_SCENARIO_DIR")); filepath.IsAbs(scenarioDir) {
		candidate := filepath.Join(scenarioDir, "skills", "notification-hub", "kit")
		if isKitDir(candidate) {
			return candidate, nil
		}
	}
	workingDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve Notification Hub kit: %w", err)
	}
	for current := workingDir; ; current = filepath.Dir(current) {
		for _, candidate := range []string{
			filepath.Join(current, "skills", "notification-hub", "kit"),
			filepath.Join(current, "scenarios", "notification-hub", "skills", "notification-hub", "kit"),
		} {
			if isKitDir(candidate) {
				return candidate, nil
			}
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
	}
	return "", errors.New("Notification Hub kit directory was not found in the scenario install or repository")
}

func isKitDir(path string) bool {
	info, err := os.Stat(filepath.Join(path, "templates"))
	return err == nil && info.IsDir()
}
