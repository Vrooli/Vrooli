package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type ControlledCommandResolver interface {
	ResolveCommand(ctx context.Context, argv []string) (CommandResolution, error)
}

type ManifestCommandResolver struct {
	repoRoot     string
	cliManifests *cliManifestCache
}

func NewManifestCommandResolver(configDir string) *ManifestCommandResolver {
	repoRoot := inferRepoRoot(configDir)
	return &ManifestCommandResolver{
		repoRoot:     repoRoot,
		cliManifests: newCLIManifestCache(repoRoot),
	}
}

func (r *ManifestCommandResolver) ResolveCommand(ctx context.Context, argv []string) (CommandResolution, error) {
	if len(argv) == 0 {
		return CommandResolution{Certainty: CertaintyNone, Message: "command argv is empty"}, nil
	}
	target := argv[0]
	switch target {
	case "vrooli":
		return r.resolveVrooli(argv), nil
	case "prompt-manager":
		return resolvePromptManager(argv), nil
	default:
		owner, ok := r.manifestOwner(ctx, target)
		if !ok {
			return CommandResolution{Certainty: CertaintyNone, Target: target, Message: "command target is not Vrooli-controlled"}, nil
		}
		// Owner identified via .vrooli/service.json. Now attempt manifest-
		// derived resolution before falling back to the Phase 0 unvalidated
		// (CertaintyOwnerOnly) path. Three outcomes (per plan
		// cli-manifest-language-agnostic-single-source-of-truth, §8):
		//   - schema-invalid → CertaintyNone (rejected)
		//   - missing manifest → CertaintyOwnerOnly (allowed, unvalidated)
		//   - hit + run_eligible=false → CertaintyNone (rejected)
		//   - hit + run_eligible=true → CertaintyCommand with governance
		if owner.Type == "scenario" && r.cliManifests != nil {
			result := r.cliManifests.load(owner.ID)
			switch {
			case result.Err != nil:
				return CommandResolution{
					Certainty: CertaintyNone,
					Owner:     CommandOwner(owner),
					Target:    target,
					Message:   "cli/manifest.json is invalid: " + result.Err.Error(),
				}, nil
			case result.Manifest != nil:
				if cmd, group, matched := resolveManifestCommand(result.Manifest, argv); matched {
					return buildManifestResolution(owner, target, group, cmd), nil
				}
				// Manifest exists but doesn't catalogue this command path —
				// treat as unvalidated rather than rejecting (the manifest
				// may not yet cover every subcommand during incremental
				// adoption).
			}
		}
		return CommandResolution{
			Certainty: CertaintyOwnerOnly,
			Owner:     CommandOwner(owner),
			Target:    target,
			Message:   "binary is Vrooli-owned, but command path is not yet cataloged",
		}, nil
	}
}

// buildManifestResolution converts a manifest hit into a CommandResolution.
// run_eligible == false is rejected here (per plan §8 certainty matrix):
// the command is documented for human/CLI use only and prompt-manager must
// not auto-invoke it.
func buildManifestResolution(owner manifestOwner, target string, group cliManifestGroup, cmd cliManifestCommand) CommandResolution {
	commandPath := []string{group.Name, cmd.Name}
	commandDescription := target + " " + group.Name + " " + cmd.Name
	if group.Flat {
		commandPath = []string{cmd.Name}
		commandDescription = target + " " + cmd.Name
	}
	if !cmd.Governance.RunEligible {
		return CommandResolution{
			Certainty:   CertaintyNone,
			Owner:       CommandOwner(owner),
			Target:      target,
			CommandPath: commandPath,
			Message:     "command is declared run_eligible=false in cli/manifest.json; not invokable as an action",
		}
	}
	effect, _ := manifestEffectToCommandEffect(cmd.Governance.Effect)
	requiresConfirmation := false
	if cmd.Governance.HasRequiresConfirmation {
		requiresConfirmation = cmd.Governance.RequiresConfirmationValue
	} else if effect == EffectDestructive {
		requiresConfirmation = true
	}
	return CommandResolution{
		Certainty:            CertaintyCommand,
		Owner:                CommandOwner(owner),
		Target:               target,
		CommandPath:          commandPath,
		Effect:               effect,
		Permissions:          append([]string(nil), cmd.Governance.Permissions...),
		RunSurfaces:          []string{"cli", "api", "action"},
		RequiresConfirmation: requiresConfirmation,
		Message:              "manifest-bound: " + commandDescription,
	}
}

func (r *ManifestCommandResolver) resolveVrooli(argv []string) CommandResolution {
	if len(argv) < 2 {
		return CommandResolution{Certainty: CertaintyNone, Target: "vrooli", Message: "vrooli subcommand is required"}
	}
	switch argv[1] {
	case "scenario":
		return resolveCatalogedCommand("vrooli", "scenario", argv[2:], scenarioCommandCatalog())
	case "resource":
		return resolveCatalogedCommand("vrooli", "resource", argv[2:], resourceCommandCatalog())
	case "setup", "develop", "build", "clean", "status", "stop", "backup", "restore", "cleanup", "doctor", "orphans", "locks", "diagnose-port", "contract":
		entry := rootCommandCatalog()[argv[1]]
		return CommandResolution{
			Certainty:   CertaintyCommand,
			Owner:       CommandOwner{Type: "project", ID: "vrooli"},
			Target:      "vrooli",
			CommandPath: []string{argv[1]},
			Effect:      entry.Effect,
			Permissions: entry.Permissions,
			RunSurfaces: []string{"cli", "api", "action"},
			Message:     "cataloged Vrooli project command",
		}
	default:
		return CommandResolution{Certainty: CertaintyNone, Target: "vrooli", CommandPath: []string{argv[1]}, Message: "unknown vrooli subcommand"}
	}
}

func resolvePromptManager(argv []string) CommandResolution {
	if len(argv) < 2 {
		return CommandResolution{Certainty: CertaintyNone, Target: "prompt-manager", Message: "prompt-manager subcommand is required"}
	}
	group, ok := promptManagerCommandCatalog()[argv[1]]
	if !ok {
		return CommandResolution{Certainty: CertaintyNone, Target: "prompt-manager", CommandPath: []string{argv[1]}, Message: "unknown prompt-manager command group"}
	}
	if group.Direct != nil {
		entry := *group.Direct
		return CommandResolution{
			Certainty:   CertaintyCommand,
			Owner:       CommandOwner{Type: "scenario", ID: "prompt-manager"},
			Target:      "prompt-manager",
			CommandPath: []string{group.Canonical},
			Effect:      entry.Effect,
			Permissions: entry.Permissions,
			RunSurfaces: []string{"cli", "api", "action"},
			Message:     "cataloged prompt-manager command",
		}
	}
	if len(argv) < 3 || strings.HasPrefix(argv[2], "-") {
		return CommandResolution{Certainty: CertaintyNone, Target: "prompt-manager", CommandPath: []string{group.Canonical}, Message: "prompt-manager " + group.Canonical + " subcommand is required"}
	}
	subcommand := argv[2]
	entry, ok := group.Subcommands[subcommand]
	if !ok {
		return CommandResolution{Certainty: CertaintyNone, Target: "prompt-manager", CommandPath: []string{group.Canonical, subcommand}, Message: "unknown prompt-manager " + group.Canonical + " subcommand"}
	}
	return CommandResolution{
		Certainty:   CertaintyCommand,
		Owner:       CommandOwner{Type: "scenario", ID: "prompt-manager"},
		Target:      "prompt-manager",
		CommandPath: []string{group.Canonical, entry.Canonical},
		Effect:      entry.Effect,
		Permissions: entry.Permissions,
		RunSurfaces: []string{"cli", "api", "action"},
		Message:     "cataloged prompt-manager command",
	}
}

type commandCatalogEntry struct {
	Canonical   string
	Effect      CommandEffect
	Permissions []string
}

type promptManagerCommandGroup struct {
	Canonical   string
	Direct      *commandCatalogEntry
	Subcommands map[string]commandCatalogEntry
}

func resolveCatalogedCommand(target, group string, args []string, catalog map[string]commandCatalogEntry) CommandResolution {
	if len(args) == 0 {
		return CommandResolution{Certainty: CertaintyNone, Target: target, CommandPath: []string{group}, Message: "subcommand is required"}
	}
	sub := args[0]
	entry, ok := catalog[sub]
	if !ok {
		return CommandResolution{Certainty: CertaintyNone, Target: target, CommandPath: []string{group, sub}, Message: fmt.Sprintf("unknown %s subcommand", group)}
	}
	return CommandResolution{
		Certainty:   CertaintyCommand,
		Owner:       CommandOwner{Type: "project", ID: "vrooli"},
		Target:      target,
		CommandPath: []string{group, sub},
		Effect:      entry.Effect,
		Permissions: entry.Permissions,
		RunSurfaces: []string{"cli", "api", "action"},
		Message:     "cataloged Vrooli " + group + " command",
	}
}

func rootCommandCatalog() map[string]commandCatalogEntry {
	return map[string]commandCatalogEntry{
		"setup":         {Effect: EffectAdmin, Permissions: []string{"filesystem:write", "host:configure", "process:start"}},
		"develop":       {Effect: EffectWrite, Permissions: []string{"process:start", "network:localhost"}},
		"build":         {Effect: EffectWrite, Permissions: []string{"filesystem:write", "process:start"}},
		"clean":         {Effect: EffectDestructive, Permissions: []string{"filesystem:write"}},
		"status":        {Effect: EffectRead, Permissions: []string{"filesystem:read", "process:start"}},
		"stop":          {Effect: EffectDestructive, Permissions: []string{"process:stop"}},
		"backup":        {Effect: EffectWrite, Permissions: []string{"filesystem:read", "filesystem:write"}},
		"restore":       {Effect: EffectDestructive, Permissions: []string{"filesystem:write"}},
		"cleanup":       {Effect: EffectDestructive, Permissions: []string{"filesystem:write", "process:stop"}},
		"doctor":        {Effect: EffectRead, Permissions: []string{"filesystem:read", "process:start"}},
		"orphans":       {Effect: EffectRead, Permissions: []string{"process:start"}},
		"locks":         {Effect: EffectRead, Permissions: []string{"filesystem:read"}},
		"diagnose-port": {Effect: EffectRead, Permissions: []string{"process:start", "network:localhost"}},
		"contract":      {Effect: EffectRead, Permissions: []string{"filesystem:read"}},
	}
}

func scenarioCommandCatalog() map[string]commandCatalogEntry {
	return map[string]commandCatalogEntry{
		"list":              {Effect: EffectRead, Permissions: []string{"filesystem:read"}},
		"info":              {Effect: EffectRead, Permissions: []string{"filesystem:read"}},
		"status":            {Effect: EffectRead, Permissions: []string{"filesystem:read", "process:start"}},
		"validate-env":      {Effect: EffectRead, Permissions: []string{"filesystem:read"}},
		"run":               {Effect: EffectWrite, Permissions: []string{"process:start", "network:localhost"}},
		"start":             {Effect: EffectWrite, Permissions: []string{"process:start", "network:localhost"}},
		"start-all":         {Effect: EffectWrite, Permissions: []string{"process:start", "network:localhost"}},
		"setup":             {Effect: EffectAdmin, Permissions: []string{"filesystem:write", "process:start"}},
		"restart":           {Effect: EffectDestructive, Permissions: []string{"process:stop", "process:start", "network:localhost"}},
		"stop":              {Effect: EffectDestructive, Permissions: []string{"process:stop"}},
		"stop-all":          {Effect: EffectDestructive, Permissions: []string{"process:stop"}},
		"test":              {Effect: EffectWrite, Permissions: []string{"filesystem:write", "process:start"}},
		"logs":              {Effect: EffectRead, Permissions: []string{"filesystem:read"}},
		"open":              {Effect: EffectRead, Permissions: []string{"network:localhost"}},
		"port":              {Effect: EffectRead, Permissions: []string{"filesystem:read"}},
		"ui-smoke":          {Effect: EffectWrite, Permissions: []string{"process:start", "network:localhost"}},
		"requirements":      {Effect: EffectWrite, Permissions: []string{"filesystem:read", "filesystem:write"}},
		"template":          {Effect: EffectWrite, Permissions: []string{"filesystem:read", "filesystem:write"}},
		"generate":          {Effect: EffectWrite, Permissions: []string{"filesystem:write"}},
		"completeness":      {Effect: EffectRead, Permissions: []string{"filesystem:read"}},
		"heal-from-sandbox": {Effect: EffectAdmin, Permissions: []string{"process:stop", "process:start"}},
	}
}

func resourceCommandCatalog() map[string]commandCatalogEntry {
	return map[string]commandCatalogEntry{
		"list":         {Effect: EffectRead, Permissions: []string{"filesystem:read"}},
		"info":         {Effect: EffectRead, Permissions: []string{"filesystem:read"}},
		"status":       {Effect: EffectRead, Permissions: []string{"filesystem:read", "process:start"}},
		"start":        {Effect: EffectWrite, Permissions: []string{"process:start", "network:localhost"}},
		"stop":         {Effect: EffectDestructive, Permissions: []string{"process:stop"}},
		"restart":      {Effect: EffectDestructive, Permissions: []string{"process:stop", "process:start"}},
		"setup":        {Effect: EffectAdmin, Permissions: []string{"filesystem:write", "process:start"}},
		"logs":         {Effect: EffectRead, Permissions: []string{"filesystem:read"}},
		"health":       {Effect: EffectRead, Permissions: []string{"network:localhost"}},
		"validate-env": {Effect: EffectRead, Permissions: []string{"filesystem:read"}},
	}
}

func promptManagerCommandCatalog() map[string]promptManagerCommandGroup {
	catalog := map[string]promptManagerCommandGroup{}
	addGroup := func(canonical string, aliases []string, entries map[string]commandCatalogEntry) {
		expanded := map[string]commandCatalogEntry{}
		for alias, entry := range entries {
			if entry.Canonical == "" {
				entry.Canonical = alias
			}
			expanded[alias] = entry
		}
		group := promptManagerCommandGroup{Canonical: canonical, Subcommands: expanded}
		for _, name := range append([]string{canonical}, aliases...) {
			catalog[name] = group
		}
	}
	addDirect := func(canonical string, aliases []string, entry commandCatalogEntry) {
		if entry.Canonical == "" {
			entry.Canonical = canonical
		}
		group := promptManagerCommandGroup{Canonical: canonical, Direct: &entry}
		for _, name := range append([]string{canonical}, aliases...) {
			catalog[name] = group
		}
	}
	read := commandCatalogEntry{Effect: EffectRead, Permissions: []string{"api:read"}}
	write := commandCatalogEntry{Effect: EffectWrite, Permissions: []string{"api:write"}}
	writeProcess := commandCatalogEntry{Effect: EffectWrite, Permissions: []string{"api:write", "process:start"}}
	destructive := commandCatalogEntry{Effect: EffectDestructive, Permissions: []string{"api:write"}}
	addGroup("skill", []string{"skills", "s"}, commandEntries(map[string]commandCatalogEntry{
		"list": read, "ls": {Canonical: "list", Effect: EffectRead, Permissions: []string{"api:read"}},
		"show": read, "get": {Canonical: "show", Effect: EffectRead, Permissions: []string{"api:read"}},
		"read": read, "cat": {Canonical: "read", Effect: EffectRead, Permissions: []string{"api:read"}},
		"sync": read, "versions": read, "history": {Canonical: "versions", Effect: EffectRead, Permissions: []string{"api:read"}},
		"variants": read, "variant": {Canonical: "variants", Effect: EffectRead, Permissions: []string{"api:read"}},
		"add": write, "create": {Canonical: "add", Effect: EffectWrite, Permissions: []string{"api:write"}},
		"update": write, "edit": {Canonical: "update", Effect: EffectWrite, Permissions: []string{"api:write"}},
		"use": write, "copy": {Canonical: "use", Effect: EffectWrite, Permissions: []string{"api:write"}},
		"rate": write, "revert": write, "restore": {Canonical: "revert", Effect: EffectWrite, Permissions: []string{"api:write"}},
		"add-variant": write,
		"delete":      destructive, "rm": {Canonical: "delete", Effect: EffectDestructive, Permissions: []string{"api:write"}},
		"rm-variant": destructive,
	}))
	addGroup("action", []string{"actions"}, commandEntries(map[string]commandCatalogEntry{
		"list": read, "ls": {Canonical: "list", Effect: EffectRead, Permissions: []string{"api:read"}},
		"show": read, "get": {Canonical: "show", Effect: EffectRead, Permissions: []string{"api:read"}},
		"validate": read,
		"create":   write, "add": {Canonical: "create", Effect: EffectWrite, Permissions: []string{"api:write"}},
		"update": write, "edit": {Canonical: "update", Effect: EffectWrite, Permissions: []string{"api:write"}},
		"run":    {Effect: EffectWrite, Permissions: []string{"api:write", "process:start"}},
		"delete": destructive, "rm": {Canonical: "delete", Effect: EffectDestructive, Permissions: []string{"api:write"}},
	}))
	addGroup("experiment", []string{"experiments"}, commandEntries(map[string]commandCatalogEntry{
		"list": read, "ls": {Canonical: "list", Effect: EffectRead, Permissions: []string{"api:read"}},
		"show": read, "get": {Canonical: "show", Effect: EffectRead, Permissions: []string{"api:read"}},
		"outcomes": read,
		"create":   write, "add": {Canonical: "create", Effect: EffectWrite, Permissions: []string{"api:write"}},
		"start": write, "conclude": write,
		"delete": destructive, "rm": {Canonical: "delete", Effect: EffectDestructive, Permissions: []string{"api:write"}},
	}))
	addGroup("tag", []string{"tags"}, commandEntries(map[string]commandCatalogEntry{
		"list": read, "ls": {Canonical: "list", Effect: EffectRead, Permissions: []string{"api:read"}},
		"create": write, "add": {Canonical: "create", Effect: EffectWrite, Permissions: []string{"api:write"}},
	}))
	addGroup("member", []string{"members"}, basicCRUDEntries(read, write, destructive))
	addGroup("agent", []string{"agents"}, commandEntries(map[string]commandCatalogEntry{
		"list": read, "ls": {Canonical: "list", Effect: EffectRead, Permissions: []string{"api:read"}},
		"show": read, "get": {Canonical: "show", Effect: EffectRead, Permissions: []string{"api:read"}},
		"soul": read, "search": read, "find": {Canonical: "search", Effect: EffectRead, Permissions: []string{"api:read"}},
		"create": write, "add": {Canonical: "create", Effect: EffectWrite, Permissions: []string{"api:write"}},
		"update": write, "edit": {Canonical: "update", Effect: EffectWrite, Permissions: []string{"api:write"}},
		"delete": destructive, "rm": {Canonical: "delete", Effect: EffectDestructive, Permissions: []string{"api:write"}},
	}))
	addGroup("topic", []string{"topics"}, commandEntries(map[string]commandCatalogEntry{
		"list": read, "ls": {Canonical: "list", Effect: EffectRead, Permissions: []string{"api:read"}},
		"show": read, "get": {Canonical: "show", Effect: EffectRead, Permissions: []string{"api:read"}},
		"skills": read, "search": read, "tree": read,
		"create": write, "add": {Canonical: "create", Effect: EffectWrite, Permissions: []string{"api:write"}},
		"update": write, "edit": {Canonical: "update", Effect: EffectWrite, Permissions: []string{"api:write"}},
		"delete": destructive, "rm": {Canonical: "delete", Effect: EffectDestructive, Permissions: []string{"api:write"}},
	}))
	addGroup("team", []string{"teams"}, teamCommandEntries(read, write, writeProcess, destructive))
	addGroup("test", []string{"testing"}, commandEntries(map[string]commandCatalogEntry{
		"run": writeProcess, "execute": {Canonical: "run", Effect: EffectWrite, Permissions: []string{"api:write", "process:start"}},
		"history": read, "results": {Canonical: "history", Effect: EffectRead, Permissions: []string{"api:read"}},
	}))
	addGroup("metadata", nil, commandEntries(map[string]commandCatalogEntry{
		"fetch": read, "get": {Canonical: "fetch", Effect: EffectRead, Permissions: []string{"api:read"}},
	}))
	addGroup("graph", nil, commandEntries(map[string]commandCatalogEntry{
		"show": read, "dump": read, "node": read,
		"orphaned-skills": read, "orphans": {Canonical: "orphaned-skills", Effect: EffectRead, Permissions: []string{"api:read"}},
		"skillless-agents": read, "skillless": {Canonical: "skillless-agents", Effect: EffectRead, Permissions: []string{"api:read"}},
		"empty-teams": read, "unaffiliated-agents": read, "unaffiliated": {Canonical: "unaffiliated-agents", Effect: EffectRead, Permissions: []string{"api:read"}},
		"cliless-skills": read, "cliless": {Canonical: "cliless-skills", Effect: EffectRead, Permissions: []string{"api:read"}},
		"popular": read, "circular-refs": read, "cycles": {Canonical: "circular-refs", Effect: EffectRead, Permissions: []string{"api:read"}},
		"health":     read,
		"regenerate": write, "regen": {Canonical: "regenerate", Effect: EffectWrite, Permissions: []string{"api:write"}},
	}))
	addDirect("discover", nil, read)
	addDirect("search", nil, read)
	addDirect("search-status", nil, read)
	addDirect("search-reindex", nil, write)
	addDirect("search-reindex-status", nil, read)
	addDirect("search-reindex-cancel", nil, write)
	return catalog
}

func commandEntries(entries map[string]commandCatalogEntry) map[string]commandCatalogEntry {
	return entries
}

func basicCRUDEntries(read, write, destructive commandCatalogEntry) map[string]commandCatalogEntry {
	return commandEntries(map[string]commandCatalogEntry{
		"list": read, "ls": {Canonical: "list", Effect: EffectRead, Permissions: []string{"api:read"}},
		"show": read, "get": {Canonical: "show", Effect: EffectRead, Permissions: []string{"api:read"}},
		"create": write, "add": {Canonical: "create", Effect: EffectWrite, Permissions: []string{"api:write"}},
		"update": write, "edit": {Canonical: "update", Effect: EffectWrite, Permissions: []string{"api:write"}},
		"delete": destructive, "rm": {Canonical: "delete", Effect: EffectDestructive, Permissions: []string{"api:write"}},
	})
}

func teamCommandEntries(read, write, writeProcess, destructive commandCatalogEntry) map[string]commandCatalogEntry {
	return commandEntries(map[string]commandCatalogEntry{
		"list": read, "ls": {Canonical: "list", Effect: EffectRead, Permissions: []string{"api:read"}},
		"show": read, "get": {Canonical: "show", Effect: EffectRead, Permissions: []string{"api:read"}},
		"roles": read, "org-list": read, "message-list": read, "heartbeat-list": read, "heartbeat": read,
		"heartbeat-logs": read, "responsibilities": read, "heartbeat-instructions": read, "export-cc": read,
		"member-context": read, "search": read, "find": {Canonical: "search", Effect: EffectRead, Permissions: []string{"api:read"}},
		"handoff-latest": read, "handoff-history": read, "task-list": read, "retention": read,
		"create": write, "add": {Canonical: "create", Effect: EffectWrite, Permissions: []string{"api:write"}},
		"update": write, "edit": {Canonical: "update", Effect: EffectWrite, Permissions: []string{"api:write"}},
		"add-member": write, "update-member": write, "remove-member": write,
		"org-set": write, "org-remove": write, "message-send": write,
		"heartbeat-enable": write, "heartbeat-disable": write, "heartbeat-trigger": write,
		"import-cc": write, "trigger": writeProcess, "task-add": write, "task-update": write,
		"prune": write,
		"delete": destructive, "rm": {Canonical: "delete", Effect: EffectDestructive, Permissions: []string{"api:write"}},
		"message-delete": destructive, "message-clear": destructive, "task-delete": destructive,
	})
}

type manifestOwner struct {
	Type string
	ID   string
}

func (r *ManifestCommandResolver) manifestOwner(ctx context.Context, command string) (manifestOwner, bool) {
	owners := r.manifestOwners(ctx)
	owner, ok := owners[command]
	return owner, ok
}

func (r *ManifestCommandResolver) manifestOwners(ctx context.Context) map[string]manifestOwner {
	owners := map[string]manifestOwner{}
	add := func(root, pattern, ownerType string) {
		matches, err := filepath.Glob(filepath.Join(root, pattern))
		if err != nil {
			return
		}
		sort.Strings(matches)
		for _, path := range matches {
			select {
			case <-ctx.Done():
				return
			default:
			}
			command := readManifestCommand(path)
			if command == "" {
				continue
			}
			id := filepath.Base(filepath.Dir(filepath.Dir(path)))
			if ownerType == "resource" {
				id = filepath.Base(filepath.Dir(path))
			}
			owners[command] = manifestOwner{Type: ownerType, ID: id}
		}
	}
	add(r.repoRoot, "scenarios/*/.vrooli/service.json", "scenario")
	add(r.repoRoot, "resources/*/resource.json", "resource")
	return owners
}

func readManifestCommand(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var raw struct {
		CLI struct {
			Enabled bool   `json:"enabled"`
			Command string `json:"command"`
			Invoke  struct {
				Command string `json:"command"`
			} `json:"invoke"`
		} `json:"cli"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return ""
	}
	if !raw.CLI.Enabled {
		return ""
	}
	if raw.CLI.Invoke.Command != "" {
		return raw.CLI.Invoke.Command
	}
	return raw.CLI.Command
}

func inferRepoRoot(configDir string) string {
	if configDir == "" {
		if wd, err := os.Getwd(); err == nil {
			return filepath.Clean(filepath.Join(wd, "..", "..", ".."))
		}
		return "."
	}
	abs, err := filepath.Abs(configDir)
	if err != nil {
		abs = configDir
	}
	// scenarios/prompt-manager/store -> repo root
	return filepath.Clean(filepath.Join(abs, "..", "..", ".."))
}
