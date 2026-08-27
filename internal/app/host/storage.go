package hostapp

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/internal/resources/securestore"
	kopiaregistry "github.com/vrooli/vrooli/packages/kopiaregistry-go"
)

func hostStorageCandidatesSpec() commandtree.Spec[string] {
	return commandtree.Spec[string]{
		Name:    "storage",
		Summary: "Discover metadata-safe escrow storage candidates",
		Help: commandtree.Help{
			Description: "Enumerates existing mounted storage and validates containment, write access, and physical-device independence. It never selects, formats, repairs, or adopts a destination.",
			Usage:       "vrooli host storage candidates [--json] [--allow-unknown-physical-device]",
			Options: []commandtree.OptionArg{
				commandtree.JSONOption(),
				{Name: "--allow-unknown-physical-device", Description: "Report writable candidates with degraded physical independence instead of requiring a known device identity"},
			},
			Examples: []string{
				"vrooli host storage candidates --json",
				"vrooli host storage candidates --allow-unknown-physical-device",
			},
		},
		Args: commandtree.ArgSchema{
			Positionals: []commandtree.PositionalArg{
				{Name: "action", Required: true, Description: "candidates"},
			},
			Options: []commandtree.OptionArg{
				commandtree.JSONOption(),
				{Name: "--allow-unknown-physical-device", Description: "Allow degraded physical identity"},
			},
		},
		Handler: "storage",
	}
}

func (app *App) runHostStorageCommand(ctx *CommandContext, args []string) error {
	spec := hostStorageCandidatesSpec()
	parsed, err := commandtree.ParseArgs("host storage", commandtree.SpecHelpText("", "vrooli host storage", spec), spec.Args, args)
	if err != nil {
		if rootcli.HandleHelp(ctx.Stdout, err) {
			return nil
		}
		return rootcli.UsageErrorf("host storage", "%s", err.Error())
	}
	if len(parsed.Positionals) != 1 || strings.ToLower(strings.TrimSpace(parsed.Positionals[0])) != "candidates" {
		return rootcli.UsageErrorf("host storage", "the action must be candidates")
	}
	home, err := config.VrooliHome()
	if err != nil {
		return fmt.Errorf("resolve Vrooli runtime home: %w", err)
	}
	protected := []string{home}
	if store, storeErr := securestore.DescribeStore(); storeErr == nil && store.Path != "" {
		protected = append(protected, filepath.Dir(store.Path))
	}
	repositoryRoots := []string{}
	registry := kopiaregistry.New(kopiaregistry.RegistryPath())
	if entries, loadErr := registry.Load(); loadErr == nil {
		for _, entry := range entries {
			if entry.Backend == kopiaregistry.BackendFilesystem && strings.TrimSpace(entry.Path) != "" {
				repositoryRoots = append(repositoryRoots, entry.Path)
			}
		}
	}
	candidates, err := hostinventory.DiscoverStorageCandidates(hostinventory.StoragePolicy{
		ProtectedRoots: protected, RepositoryRoots: repositoryRoots,
		RequirePhysicalSeparation: !parsed.HasFlag("--allow-unknown-physical-device"),
	})
	if err != nil {
		return err
	}
	if ctx.Globals.JSON || parsed.HasFlag("--json") {
		return cliout.WriteJSONValue(ctx.Stdout, struct {
			Candidates      []hostinventory.StorageCandidate `json:"candidates"`
			ProtectedRoots  []string                         `json:"protected_roots"`
			RepositoryRoots []string                         `json:"repository_roots"`
		}{Candidates: candidates, ProtectedRoots: protected, RepositoryRoots: repositoryRoots})
	}
	rows := make([][]string, 0, len(candidates))
	for _, candidate := range candidates {
		rows = append(rows, []string{candidate.Status, candidate.Kind, candidate.Location, candidate.Remediation})
	}
	return cliout.WriteSection(ctx.Stdout, cliout.Section{Rows: rows})
}
