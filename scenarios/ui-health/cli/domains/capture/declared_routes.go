package capture

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

//go:embed declared_routes.cjs
var declaredRoutesScript string

// declaredRoute resolves only an exact page name in an explicit React Router
// declaration. Unsupported expressions and multiple routes remain unresolved.
func declaredRoute(ctx context.Context, root, scenario, subject string) (string, error) {
	if filepath.Base(scenario) != scenario || scenario == "." || scenario == ".." {
		return "", fmt.Errorf("invalid scenario id %q", scenario)
	}
	sourceRoot := filepath.Join(root, "scenarios", scenario, "ui", "src")
	app, err := os.ReadFile(filepath.Join(sourceRoot, "App.tsx"))
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	routes, err := os.ReadFile(filepath.Join(sourceRoot, "routes.ts"))
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	if len(app)+len(routes) > 1<<20 {
		return "", fmt.Errorf("route declarations exceed 1 MiB")
	}
	payload, err := json.Marshal(map[string]string{"app": string(app), "routes": string(routes)})
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	compiler := filepath.Join(root, "scenarios", "ui-health", "ui", "node_modules", "typescript")
	cmd := exec.CommandContext(ctx, "node", "-e", declaredRoutesScript, compiler)
	cmd.Stdin = bytes.NewReader(payload)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("parse declared routes: %w", err)
	}
	var declarations []struct{ Subject, Route string }
	if err := json.Unmarshal(output, &declarations); err != nil {
		return "", err
	}
	name := strings.TrimSuffix(filepath.Base(subject), filepath.Ext(subject))
	found := ""
	for _, declaration := range declarations {
		if declaration.Subject != name {
			continue
		}
		if found != "" && found != declaration.Route {
			return "", fmt.Errorf("page %s has multiple declared routes; supply its exact URL", name)
		}
		found = declaration.Route
	}
	return found, nil
}
