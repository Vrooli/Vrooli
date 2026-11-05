package depsapproved

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
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

func runRemove(_ *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("deps remove")
	var req removeGoRequirementRequest
	var jsonOutput bool
	fs.StringVar(&req.Scenario, "scenario", "", "Target scenario (vrooli only)")
	fs.StringVar(&req.Surface, "surface", "", "Target surface (control-plane only)")
	fs.StringVar(&req.Ecosystem, "ecosystem", "", "Dependency ecosystem (go only)")
	fs.StringVar(&req.Module, "module", "", "Exact Go module path to remove from root go.mod")
	fs.BoolVar(&req.Apply, "apply", false, "Write the change (default is a dry run)")
	fs.StringVar(&req.ExpectedSHA256, "expected-sha256", "", "Required with --apply; copy before_sha256 from the preview")
	fs.BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 0 {
		return fmt.Errorf("usage: %s deps remove --scenario vrooli --surface control-plane --ecosystem go --module <module-path> [--apply --expected-sha256 <before-sha256>] [--json]", support.AppName)
	}
	result, err := removeRootGoRequirement(cliutil.ResolveRepoRoot(), req)
	if err != nil {
		return err
	}
	if jsonOutput {
		return support.PrintReportJSON(result)
	}
	state := "Dry run"
	if result.Applied {
		state = "Applied"
	}
	fmt.Printf("%s: remove require %s from %s\nBefore SHA-256: %s\nAfter SHA-256:  %s\n", state, result.Module, result.Target, result.BeforeSHA, result.AfterSHA)
	return nil
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
