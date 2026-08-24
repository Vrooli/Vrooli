// Package targetmodel resolves the repository's first-class validation target
// vocabulary. It has no phase knowledge: providers and orchestration consume
// the same resolved {kind,id,root} value.
package targetmodel

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

type Target struct {
	Kind commonv1.ValidationTargetKind
	ID   string
	Root string
	Path string
}

// ArtifactRoot returns the durable evidence owner for a target. Scenario
// artifacts retain their historical coverage location; every other target is
// isolated below the contract-owned runtime state root.
func ArtifactRoot(repoRoot string, target Target) (string, error) {
	if target.HasRuntime() {
		return target.Path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home: %w", err)
	}
	contract, err := repocontract.LoadDefault(repoRoot)
	if err != nil {
		return "", fmt.Errorf("load repository contract: %w", err)
	}
	state, err := contract.RuntimeHomeEntry(home, repocontract.HomeKeyState)
	if err != nil {
		return "", fmt.Errorf("resolve runtime state: %w", err)
	}
	kind := strings.TrimPrefix(strings.ToLower(target.Kind.String()), "validation_target_kind_")
	return filepath.Join(state.AbsPath, "test-genie", "targets", kind, target.ID), nil
}

func (t Target) Proto() *commonv1.ValidationTarget {
	return &commonv1.ValidationTarget{Kind: t.Kind, Id: t.ID, Root: t.Root}
}

func (t Target) HasRuntime() bool {
	return t.Kind == commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SCENARIO
}

// Resolve accepts a supported bare scenario-slug alias or the generalized kind:id
// notation. An explicit path under scenarios/ is promoted to a scenario target
// so it receives the full scenario contract rather than a generic tree gate.
func Resolve(repoRoot, expression string) (Target, error) {
	repoRoot = filepath.Clean(strings.TrimSpace(repoRoot))
	expression = strings.TrimSpace(expression)
	if repoRoot == "." || repoRoot == "" {
		return Target{}, fmt.Errorf("repository root is required")
	}
	if expression == "" {
		return Target{}, fmt.Errorf("target is required")
	}
	contract, err := repocontract.LoadDefault(repoRoot)
	if err != nil {
		return Target{}, fmt.Errorf("load repository contract: %w", err)
	}
	if !strings.Contains(expression, ":") {
		return resolveScenario(contract, repoRoot, expression)
	}
	parts := strings.SplitN(expression, ":", 2)
	kind, id := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	if kind == "" || id == "" {
		return Target{}, fmt.Errorf("target must use kind:id")
	}
	if kind == "scenario" {
		return resolveScenario(contract, repoRoot, id)
	}
	// code-facts and older operator docs call a shared package a module. Keep
	// that input as a compatibility alias while storing the repository target
	// vocabulary as package.
	if kind == "module" {
		kind = "package"
		id = strings.TrimPrefix(filepath.ToSlash(filepath.Clean(id)), "packages/")
	}
	// Paths are accepted as an explicit operator-facing alias. The contract
	// still supplies the canonical owner ID used for durable identity.
	if strings.Contains(id, "/") {
		if target, lookupErr := contract.Target(repoRoot, repocontract.TargetKind(kind), filepath.ToSlash(filepath.Clean(id))); lookupErr == nil {
			return fromContractTarget(repoRoot, target), nil
		}
	}
	target, err := contract.Target(repoRoot, repocontract.TargetKind(kind), id)
	if err != nil {
		return Target{}, err
	}
	return fromContractTarget(repoRoot, target), nil
}

func resolveScenario(contract *repocontract.Contract, repoRoot, value string) (Target, error) {
	value = filepath.Clean(strings.TrimSpace(value))
	if value == "." || value == ".." || strings.ContainsAny(value, `\\`) {
		return Target{}, fmt.Errorf("invalid scenario target %q", value)
	}
	if strings.HasPrefix(filepath.ToSlash(value), "scenarios/") {
		rel := filepath.ToSlash(value)
		parts := strings.Split(strings.TrimPrefix(rel, "scenarios/"), "/")
		if len(parts) == 1 && parts[0] != "" {
			value = parts[0]
		}
	}
	target, err := contract.Target(repoRoot, repocontract.TargetKindScenario, value)
	if err != nil {
		return Target{}, err
	}
	return fromContractTarget(repoRoot, target), nil
}

func fromContractTarget(repoRoot string, target repocontract.Target) Target {
	return Target{Kind: targetKind(target.Kind), ID: target.ID, Root: target.Root, Path: filepath.Join(repoRoot, filepath.FromSlash(target.Root))}
}

func targetKind(kind repocontract.TargetKind) commonv1.ValidationTargetKind {
	switch kind {
	case repocontract.TargetKindScenario:
		return commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SCENARIO
	case repocontract.TargetKindResource:
		return commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_RESOURCE
	case repocontract.TargetKindTool:
		return commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_TOOL
	case repocontract.TargetKindSafeguard:
		return commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SAFEGUARD
	case repocontract.TargetKindTeam:
		return commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_TEAM
	case repocontract.TargetKindPackage:
		return commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_PACKAGE
	case repocontract.TargetKindControlPlane:
		return commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_CONTROL_PLANE
	case repocontract.TargetKindDocs:
		return commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_DOCS
	case repocontract.TargetKindProject:
		return commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_PROJECT
	default:
		return commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_UNSPECIFIED
	}
}
