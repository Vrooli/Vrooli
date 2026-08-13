package deployment

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
	"scenario-dependency-analyzer/internal/config"
	types "scenario-dependency-analyzer/internal/types"
)

// BuildTargetDAG resolves a repository target through repo-contract-go and
// exports the dependency surface appropriate to that target kind.
func BuildTargetDAG(repoRoot, expression string, recursive, refresh bool) (*types.TargetDAGResponse, error) {
	contract, err := repocontract.LoadDefault(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("load repository contract: %w", err)
	}
	kind, id := splitTargetExpression(expression)
	if kind == "" {
		kind = string(repocontract.TargetKindScenario)
	}
	target, err := contract.Target(repoRoot, repocontract.TargetKind(kind), id)
	if err != nil {
		return nil, err
	}
	var nodes []types.DeploymentDependencyNode
	switch target.Kind {
	case repocontract.TargetKindScenario:
		cfg, loadErr := config.LoadServiceConfig(filepath.Join(repoRoot, filepath.FromSlash(target.Root)))
		if loadErr != nil {
			return nil, loadErr
		}
		nodes = BuildDependencyNodeList(filepath.Join(repoRoot, "scenarios"), target.ID, cfg, map[string]struct{}{config.NormalizeName(target.ID): {}})
	case repocontract.TargetKindPackage:
		nodes, err = buildModuleDependencies(filepath.Join(repoRoot, filepath.FromSlash(target.Root)), repoRoot)
		if err != nil {
			return nil, err
		}
	case repocontract.TargetKindResource:
		nodes, err = buildResourceDependencies(filepath.Join(repoRoot, filepath.FromSlash(target.Root)), repoRoot)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("target kind %q has no dependency DAG adapter", target.Kind)
	}
	if !recursive {
		for i := range nodes {
			nodes[i].Children = nil
		}
	}
	return &types.TargetDAGResponse{
		TargetKind: string(target.Kind), TargetID: target.ID, TargetRoot: target.Root,
		Recursive: recursive, GeneratedAt: time.Now().UTC(), DAG: nodes,
	}, nil
}

func splitTargetExpression(expression string) (string, string) {
	parts := strings.SplitN(strings.TrimSpace(expression), ":", 2)
	if len(parts) == 1 {
		return "", parts[0]
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}

func buildModuleDependencies(moduleRoot, repoRoot string) ([]types.DeploymentDependencyNode, error) {
	file, err := os.Open(filepath.Join(moduleRoot, "go.mod"))
	if err != nil {
		return nil, fmt.Errorf("open package go.mod: %w", err)
	}
	defer file.Close()
	var nodes []types.DeploymentDependencyNode
	inRequire := false
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(strings.SplitN(scanner.Text(), "//", 2)[0])
		if strings.HasPrefix(line, "require (") {
			inRequire = true
			continue
		}
		if inRequire && line == ")" {
			inRequire = false
			continue
		}
		if strings.HasPrefix(line, "require ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "require "))
		}
		if !inRequire && !strings.HasPrefix(line, "require ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 || strings.HasPrefix(fields[0], "(") {
			continue
		}
		nodes = append(nodes, types.DeploymentDependencyNode{Name: fields[0], Type: "module", Source: "go.mod"})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read package go.mod: %w", err)
	}
	return nodes, nil
}

func buildResourceDependencies(resourceRoot, repoRoot string) ([]types.DeploymentDependencyNode, error) {
	var manifest struct {
		Dependencies []struct {
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"dependencies"`
	}
	data, err := os.ReadFile(filepath.Join(resourceRoot, "resource.json"))
	if err != nil {
		return nil, fmt.Errorf("read resource manifest: %w", err)
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parse resource manifest: %w", err)
	}
	nodes := make([]types.DeploymentDependencyNode, 0, len(manifest.Dependencies))
	for _, dependency := range manifest.Dependencies {
		kind := dependency.Type
		if kind == "" {
			kind = "resource"
		}
		nodes = append(nodes, types.DeploymentDependencyNode{Name: dependency.Name, Type: kind, Source: "resource.json"})
	}
	return nodes, nil
}
