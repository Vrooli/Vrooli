package depsapproved

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	governancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_governance"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
	"scenario-dependency-analyzer/cli/internal/support"
)

type removeGoRequirementRequest struct {
	Scenario, Surface, Ecosystem, Module, ExpectedSHA256 string
	Apply                                                bool
}

type removeGoRequirementResult struct {
	Module    string `json:"module"`
	Target    string `json:"target"`
	Applied   bool   `json:"applied"`
	BeforeSHA string `json:"before_sha256"`
	AfterSHA  string `json:"after_sha256"`
}

const removeUsage = "usage: %s deps remove <ecosystem>/<package>... --scenario <name> --surface <surface> [--apply] [--json]\n   or: %s deps remove --scenario vrooli --surface control-plane --ecosystem go --module <module-path> [--apply --expected-sha256 <before-sha256>] [--json]"

func removeArgumentSchema() cliapp.ArgSchema {
	return cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{
			Name:        "dependency",
			Repeated:    true,
			Description: "JavaScript dependency to remove, as npm/<package>; name several to remove them with one resolution",
		}},
		Flags: []cliapp.Flag{
			{Name: "scenario", Description: "Target scenario (vrooli for the Go root module)"},
			{Name: "surface", Description: "Target surface: ui, api, cli, tools/<package>, ... (control-plane for the Go root module)"},
			{Name: "apply", Bool: true, Description: "Perform the removal; default is a dry-run preview"},
			{Name: "ecosystem", Description: "Go root-module removal only: go"},
			{Name: "module", Description: "Go root-module removal only: exact Go module path to drop from the root go.mod"},
			{Name: "expected-sha256", Description: "Go root-module removal only: required with --apply; copy before_sha256 from the preview"},
		},
	}
}

// runRemove takes dependencies off a surface. Named npm packages go through the
// governed gateway (the same staged resolution, store and rollback as install);
// without a package it is the Go root-module requirement removal, which edits
// one file locally and so needs no running API.
func runRemove(core *cliapp.ScenarioApp, args cliapp.RunContext) error {
	if packages := args.Positionals("dependency"); len(packages) > 0 {
		if args.Flag("module") != "" || args.Flag("ecosystem") != "" || args.Flag("expected-sha256") != "" {
			return fmt.Errorf("--module, --ecosystem and --expected-sha256 belong to the Go root-module removal; do not combine them with <ecosystem>/<package> arguments")
		}
		return runRemoveDependencies(core, args, packages)
	}
	if args.Flag("module") == "" {
		return fmt.Errorf(removeUsage, support.AppName, support.AppName)
	}
	req := removeGoRequirementRequest{
		Scenario: args.Flag("scenario"), Surface: args.Flag("surface"), Ecosystem: args.Flag("ecosystem"),
		Module: args.Flag("module"), ExpectedSHA256: args.Flag("expected-sha256"), Apply: args.BoolFlag("apply"),
	}
	result, err := removeRootGoRequirement(cliutil.ResolveRepoRoot(), req)
	if err != nil {
		return err
	}
	if args.JSON() {
		return support.PrintReportJSON(result)
	}
	state := "Dry run"
	if result.Applied {
		state = "Applied"
	}
	fmt.Printf("%s: remove require %s from %s\nBefore SHA-256: %s\nAfter SHA-256:  %s\n", state, result.Module, result.Target, result.BeforeSHA, result.AfterSHA)
	return nil
}

func runRemoveDependencies(core *cliapp.ScenarioApp, args cliapp.RunContext, specs []string) error {
	scenario, surface := strings.TrimSpace(args.Flag("scenario")), strings.TrimSpace(args.Flag("surface"))
	if scenario == "" || surface == "" {
		return fmt.Errorf("--scenario and --surface are required; %s", fmt.Sprintf(removeUsage, support.AppName, support.AppName))
	}
	ecosystem, names, err := parseRemoveSpecs(specs)
	if err != nil {
		return err
	}
	// Resolving the removal, relinking node_modules and rolling back on failure
	// share the install's three-minute budget.
	timeout := 3*time.Minute + 15*time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	resp, err := governanceClientWithTimeout(core, timeout).RemoveDependency(ctx, connect.NewRequest(&governancev1.RemoveDependencyRequest{
		Scenario: scenario, Surface: surface, Ecosystem: ecosystem, PackageNames: names, Apply: args.BoolFlag("apply"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("remove dependency", err, nil)
	}
	if args.JSON() {
		if err := printProto(resp.Msg); err != nil {
			return err
		}
		return removeOutcome(resp.Msg)
	}
	msg := resp.Msg
	summary := []string{msg.GetMessage(), fmt.Sprintf("Verdict: %s", msg.GetVerdict()), fmt.Sprintf("Command: %s", msg.GetCommand())}
	if msg.GetManifestPath() != "" {
		summary = append(summary, fmt.Sprintf("Manifest: %s", msg.GetManifestPath()))
	}
	summary = append(summary, msg.GetNotes()...)
	if err := support.PrintList(false, cliapp.ListReport{Summary: summary, ResultsHeading: "Dependency Removal", Results: msg.GetNextSteps()}, nil); err != nil {
		return err
	}
	return removeOutcome(msg)
}

// parseRemoveSpecs splits npm/<package> arguments. All must name one ecosystem.
func parseRemoveSpecs(specs []string) (ecosystem string, names []string, err error) {
	for _, spec := range specs {
		eco, name, ok := strings.Cut(strings.TrimSpace(spec), "/")
		if !ok || strings.TrimSpace(eco) == "" || strings.TrimSpace(name) == "" {
			return "", nil, fmt.Errorf("%q must be formatted as <ecosystem>/<package>, for example npm/three", spec)
		}
		if ecosystem != "" && eco != ecosystem {
			return "", nil, fmt.Errorf("one removal covers one ecosystem; got %q and %q", ecosystem, eco)
		}
		ecosystem = eco
		names = append(names, name)
	}
	return ecosystem, names, nil
}

func removeOutcome(msg *governancev1.RemoveDependencyResponse) error {
	if !msg.GetBlocked() && (msg.GetDryRun() || msg.GetRemoved() || msg.GetSourceUpdated()) {
		return nil
	}
	return fmt.Errorf("dependency removal did not succeed (verdict=%s, blocked=%t, removed=%t)", msg.GetVerdict(), msg.GetBlocked(), msg.GetRemoved())
}

func removeRootGoRequirement(root string, req removeGoRequirementRequest) (removeGoRequirementResult, error) {
	if req.Scenario != "vrooli" || req.Surface != "control-plane" || req.Ecosystem != "go" {
		return removeGoRequirementResult{}, errors.New("deps remove is limited to --scenario vrooli --surface control-plane --ecosystem go")
	}
	if req.Module == "" || strings.TrimSpace(req.Module) != req.Module {
		return removeGoRequirementResult{}, errors.New("--module must be one exact Go module path")
	}
	if err := module.CheckPath(req.Module); err != nil {
		return removeGoRequirementResult{}, fmt.Errorf("invalid Go module path: %w", err)
	}
	if req.Apply && !validSHA256(req.ExpectedSHA256) {
		return removeGoRequirementResult{}, errors.New("--apply requires --expected-sha256 from the preview")
	}
	if !req.Apply && req.ExpectedSHA256 != "" {
		return removeGoRequirementResult{}, errors.New("--expected-sha256 requires --apply")
	}
	path := filepath.Join(root, "go.mod")
	before, info, err := readRegularGoMod(path)
	if err != nil {
		return removeGoRequirementResult{}, err
	}
	beforeSHA := sha256Hex(before)
	if req.Apply && !strings.EqualFold(beforeSHA, req.ExpectedSHA256) {
		return removeGoRequirementResult{}, errors.New("root go.mod changed since preview; no write performed")
	}
	parsed, err := modfile.Parse(path, before, nil)
	if err != nil {
		return removeGoRequirementResult{}, fmt.Errorf("parse root go.mod: %w", err)
	}
	count := 0
	for _, item := range parsed.Require {
		if item.Mod.Path == req.Module {
			count++
		}
	}
	if count != 1 {
		return removeGoRequirementResult{}, fmt.Errorf("root go.mod must contain exactly one require for %s; found %d", req.Module, count)
	}
	if err := parsed.DropRequire(req.Module); err != nil {
		return removeGoRequirementResult{}, err
	}
	after, err := parsed.Format()
	if err != nil {
		return removeGoRequirementResult{}, fmt.Errorf("format root go.mod: %w", err)
	}
	result := removeGoRequirementResult{Module: req.Module, Target: path, BeforeSHA: beforeSHA, AfterSHA: sha256Hex(after)}
	if !req.Apply {
		return result, nil
	}
	if err := writeGoModIfUnchanged(root, path, after, info, req.ExpectedSHA256); err != nil {
		return removeGoRequirementResult{}, err
	}
	result.Applied = true
	return result, nil
}

func readRegularGoMod(path string) ([]byte, os.FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return nil, nil, errors.New("root go.mod must be a regular non-symlink file")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	latest, err := os.Lstat(path)
	if err != nil || !os.SameFile(info, latest) {
		return nil, nil, errors.New("root go.mod changed during read")
	}
	return data, info, nil
}

func writeGoModIfUnchanged(root, path string, after []byte, info os.FileInfo, expected string) error {
	tmp, err := os.CreateTemp(root, ".go.mod-remove-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err = tmp.Close(); err != nil {
		return err
	}
	if err = os.WriteFile(tmpPath, after, info.Mode().Perm()); err != nil {
		return err
	}
	if err = os.Chmod(tmpPath, info.Mode().Perm()); err != nil {
		return err
	}
	current, currentInfo, err := readRegularGoMod(path)
	if err != nil {
		return err
	}
	if !os.SameFile(info, currentInfo) || !strings.EqualFold(sha256Hex(current), expected) {
		return errors.New("root go.mod changed before apply; no write performed")
	}
	return os.Rename(tmpPath, path)
}

func sha256Hex(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }

func validSHA256(value string) bool {
	_, err := hex.DecodeString(value)
	return len(value) == sha256.Size*2 && err == nil
}
