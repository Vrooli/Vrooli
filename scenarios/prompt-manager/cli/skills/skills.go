// Package skills provides CLI commands for skill management.
//
// DOC: docs/reference/cli-commands.md#skills
// DOC: docs/concepts/ARCHITECTURE.md#cli-architecture
package skills

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"prompt-manager/cli/internal/appctx"
	"prompt-manager/cli/internal/clipboard"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// SkillResponse matches the API response for skills
type SkillResponse struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	Description         string   `json:"description"`
	Content             string   `json:"content"`
	Modes               []string `json:"modes"`
	Tags                []string `json:"tags"`
	Icon                string   `json:"icon,omitempty"`
	ProgrammaticHome    *string  `json:"programmaticHome,omitempty"`
	Draft               bool     `json:"draft"`
	Folder              string   `json:"folder"`
	CreatedAt           string   `json:"createdAt"`
	UpdatedAt           string   `json:"updatedAt"`
	UsageCount          int      `json:"usageCount"`
	EffectivenessRating *int     `json:"effectivenessRating,omitempty"`
}

// CreateSkillRequest matches the API request for creating skills
type CreateSkillRequest struct {
	ID          string   `json:"id,omitempty"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Content     string   `json:"content"`
	Modes       []string `json:"modes,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Icon        string   `json:"icon,omitempty"`
	Draft       bool     `json:"draft"`
	Folder      string   `json:"folder"`
}

// UpdateSkillRequest matches the API request for updating skills
type UpdateSkillRequest struct {
	Name                  *string  `json:"name,omitempty"`
	Description           *string  `json:"description,omitempty"`
	Content               *string  `json:"content,omitempty"`
	Modes                 []string `json:"modes,omitempty"`
	Tags                  []string `json:"tags,omitempty"`
	Icon                  *string  `json:"icon,omitempty"`
	ProgrammaticHome      *string  `json:"programmaticHome,omitempty"`
	ClearProgrammaticHome bool     `json:"clearProgrammaticHome,omitempty"`
	Draft                 *bool    `json:"draft,omitempty"`
}

// SyncResponse matches the API response for sync
type SyncResponse struct {
	Skills      []SkillResponse `json:"skills"`
	LastUpdated string          `json:"lastUpdated"`
	Hash        string          `json:"hash"`
}

// ReadRequest matches the API request for reading multiple skills
type ReadRequest struct {
	Identifiers  []string `json:"identifiers"`
	Resolve      string   `json:"resolve,omitempty"`
	AllowMissing *bool    `json:"allowMissing,omitempty"`
	Output       string   `json:"output,omitempty"`
	Format       string   `json:"format,omitempty"`
	WithScope    bool     `json:"withScope,omitempty"`
	Scope        string   `json:"scope,omitempty"`
}

// ReadIssue captures missing identifiers
type ReadIssue struct {
	Identifier string `json:"identifier"`
	Reason     string `json:"reason"`
}

// ReadCandidate is a minimal skill representation for ambiguity reporting
type ReadCandidate struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	File   string `json:"file"`
	Folder string `json:"folder"`
}

// ReadAmbiguous captures ambiguous identifiers
type ReadAmbiguous struct {
	Identifier string          `json:"identifier"`
	Candidates []ReadCandidate `json:"candidates"`
}

// ReadResponse matches the API response for reading multiple skills
type ReadResponse struct {
	Skills      []SkillResponse `json:"skills,omitempty"`
	Combined    string          `json:"combined,omitempty"`
	SkillCount  int             `json:"skillCount,omitempty"`
	TotalTokens int             `json:"totalTokens,omitempty"`
	Format      string          `json:"format,omitempty"`
	Missing     []ReadIssue     `json:"missing,omitempty"`
	Ambiguous   []ReadAmbiguous `json:"ambiguous,omitempty"`
	Resolve     string          `json:"resolve"`
	Output      string          `json:"output,omitempty"`
}

// VersionResponse matches the API response for versions
type VersionResponse struct {
	SkillID  string            `json:"skillId"`
	Current  int               `json:"current"`
	Versions []SkillVersionDef `json:"versions"`
}

// SkillVersionDef represents a skill version
type SkillVersionDef struct {
	Version   int    `json:"version"`
	Content   string `json:"content"`
	Name      string `json:"name"`
	UpdatedAt string `json:"updatedAt"`
}

// Commands returns the skill command groups using noun-verb pattern.
func Commands(ctx appctx.Context) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		{
			Title: "Skills",
			Commands: []cliapp.Command{
				{
					Name:        "skill",
					Aliases:     []string{"skills", "s"},
					NeedsAPI:    true,
					Description: "Manage skills (list|topology|show|read|add|update|delete|sync|rate|versions|revert|variants|import|review-import|import-staleness)",
					Usage:       "prompt-manager skill <subcommand> [args]",
					HelpText:    usageText(),
					Run: func(args []string) error {
						return route(ctx, args)
					},
				},
			},
		},
	}
}

// route dispatches to the appropriate subcommand.
func route(ctx appctx.Context, args []string) error {
	if len(args) == 0 {
		return printUsage()
	}

	subcommand := args[0]
	subArgs := args[1:]

	switch subcommand {
	case "list", "ls":
		return cmdList(ctx, subArgs)
	case "topology":
		return cmdTopology(ctx, subArgs)
	case "show", "get":
		return cmdShow(ctx, subArgs)
	case "read", "cat":
		return cmdRead(ctx, subArgs)
	case "add", "create":
		return cmdAdd(ctx, subArgs)
	case "update", "edit":
		return cmdUpdate(ctx, subArgs)
	case "delete", "rm":
		return cmdDelete(ctx, subArgs)
	case "use", "copy":
		return fmt.Errorf("skill use/copy was removed; use `skill read <id>` — it records the read and takes --copy for the clipboard")
	case "sync":
		return cmdSync(ctx, subArgs)
	case "rate":
		return cmdRate(ctx, subArgs)
	case "versions", "history":
		return cmdVersions(ctx, subArgs)
	case "revert", "restore":
		return cmdRevert(ctx, subArgs)
	case "variants", "variant":
		return cmdVariants(ctx, subArgs)
	case "add-variant":
		return cmdAddVariant(ctx, subArgs)
	case "rm-variant":
		return cmdRmVariant(ctx, subArgs)
	case "import":
		return cmdImport(ctx, subArgs)
	case "review-import":
		return cmdReviewImport(ctx, subArgs)
	case "import-staleness":
		return cmdImportStaleness(ctx, subArgs)
	case "activation-hook":
		return cmdActivationHook(ctx)
	default:
		return fmt.Errorf("unknown subcommand: %s\n\n%s", subcommand, usageText())
	}
}

func printUsage() error {
	fmt.Println(usageText())
	return nil
}

func usageText() string {
	return `Usage: prompt-manager skill <subcommand> [args]

Subcommands:
  list, ls              List all skills
  topology              Report each skill's pack and generated projection status
  show, get <id>        Show skill details
  read <identifier>...  Read skills (content or combined output)
  add, create <name>    Create a new skill
  update, edit <id>     Update an existing skill
  delete, rm <id>       Delete a skill
  sync                  Sync skills with hash-based change detection
  rate <id> <1-5>       Rate skill effectiveness
  versions, history <id> Show version history
  revert, restore <id> <version>  Revert to a specific version
  variants <id>         List variants for a skill
  add-variant <id>      Create a new variant
  rm-variant <id> <vid> Delete a variant
  import --source-dir <dir> --source-url <url> --commit <sha> --license <SPDX> --checksum <sha256:...> --imported-by <id> [--id <id>]
                       Import pinned local skill material into quarantined vendor storage
  review-import <id> --reviewer <id> --verdict passed|rejected
                       Record an independent review verdict
  import-staleness <id> --upstream-version <version>
                       Compare the recorded upstream version with a current version
  activation-hook       Read a native tool-activation event from stdin and report it fail-open`
}

// cmdActivationHook is designed for native PreToolUse/skill hooks. It never
// turns telemetry failure into an agent failure and bounds the local request.
func cmdActivationHook(ctx appctx.Context) error {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil
	}
	var event struct {
		ToolName  string `json:"tool_name"`
		ToolInput struct {
			SkillName string `json:"skill_name"`
			Skill     string `json:"skill"`
			Name      string `json:"name"`
		} `json:"tool_input"`
		SkillName string `json:"skill_name"`
	}
	if json.Unmarshal(data, &event) != nil {
		return nil
	}
	id := event.ToolInput.SkillName
	if id == "" {
		id = event.ToolInput.Skill
	}
	if id == "" {
		id = event.ToolInput.Name
	}
	if id == "" {
		id = event.SkillName
	}
	if id == "" {
		return nil
	}
	done := make(chan struct{})
	go func() {
		var ignored struct{}
		_ = ctx.Post("/skills/"+url.PathEscape(id)+"/use", nil, &ignored)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(750 * time.Millisecond):
	}
	return nil
}

func cmdImport(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("import", flag.ContinueOnError)
	sourceDir := fs.String("source-dir", "", "Pinned local directory containing SKILL.md")
	sourceURL := fs.String("source-url", "", "Immutable source URL")
	commit := fs.String("commit", "", "Pinned commit")
	license := fs.String("license", "", "SPDX license identifier")
	checksum := fs.String("checksum", "", "sha256:<64 hex> checksum of SKILL.md")
	importedBy := fs.String("imported-by", "", "Importer identity")
	version := fs.String("upstream-version", "", "Upstream version")
	id := fs.String("id", "", "Skill ID (must match frontmatter name)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *sourceDir == "" || *sourceURL == "" || *commit == "" || *license == "" || *checksum == "" || *importedBy == "" || *id == "" {
		return fmt.Errorf("usage: skill import --source-dir <dir> --source-url <url> --commit <sha> --license <SPDX> --checksum <sha256:...> --imported-by <id> --id <id>")
	}
	var response struct {
		ID            string `json:"id"`
		Pack          string `json:"pack"`
		Status        string `json:"status"`
		Checksum      string `json:"checksum"`
		ReviewVerdict string `json:"reviewVerdict"`
		ImportedAt    string `json:"importedAt"`
	}
	if err := ctx.Post("/skills/import", map[string]string{"sourceDir": *sourceDir, "sourceUrl": *sourceURL, "commit": *commit, "license": *license, "checksum": *checksum, "importedBy": *importedBy, "upstreamVersion": *version, "id": *id}, &response); err != nil {
		return fmt.Errorf("import skill: %w", err)
	}
	fmt.Printf("Imported %s into %s pack: status=%s review=%s checksum=%s\n", response.ID, response.Pack, response.Status, response.ReviewVerdict, response.Checksum)
	return nil
}

func cmdReviewImport(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("review-import", flag.ContinueOnError)
	reviewer := fs.String("reviewer", "", "Reviewer identity")
	verdict := fs.String("verdict", "", "passed or rejected")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 || *reviewer == "" || *verdict == "" {
		return fmt.Errorf("usage: skill review-import <id> --reviewer <id> --verdict passed|rejected")
	}
	var response struct {
		ID         string `json:"id"`
		Status     string `json:"status"`
		Verdict    string `json:"verdict"`
		Reviewer   string `json:"reviewer"`
		ReviewedAt string `json:"reviewedAt"`
	}
	if err := ctx.Post("/skills/"+url.PathEscape(fs.Arg(0))+"/review", map[string]string{"reviewer": *reviewer, "verdict": *verdict}, &response); err != nil {
		return fmt.Errorf("review imported skill: %w", err)
	}
	fmt.Printf("Reviewed %s: verdict=%s reviewer=%s\n", response.ID, response.Verdict, response.Reviewer)
	return nil
}

func cmdImportStaleness(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("import-staleness", flag.ContinueOnError)
	version := fs.String("upstream-version", "", "Current upstream version")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 || *version == "" {
		return fmt.Errorf("usage: skill import-staleness <id> --upstream-version <version>")
	}
	var response struct {
		ID              string `json:"id"`
		RecordedVersion string `json:"recordedVersion"`
		CurrentVersion  string `json:"currentVersion"`
		Stale           bool   `json:"stale"`
	}
	if err := ctx.Post("/skills/"+url.PathEscape(fs.Arg(0))+"/staleness", map[string]string{"upstreamVersion": *version}, &response); err != nil {
		return fmt.Errorf("report import staleness: %w", err)
	}
	fmt.Printf("%s: recorded=%s current=%s stale=%t\n", response.ID, response.RecordedVersion, response.CurrentVersion, response.Stale)
	return nil
}

func cmdList(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	folder := fs.String("folder", "", "Filter by folder (core|local|drafts)")
	tag := fs.String("tag", "", "Filter by tag")
	mode := fs.String("mode", "", "Filter by mode (e.g. steer, search)")
	withoutProgrammaticHome := fs.Bool("without-programmatic-home", false, "Keep only skills whose detection has NOT graduated to a programmatic engine (programmaticHome unset). Combine with --mode steer for the quality-auditor frontier rotation.")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	query := url.Values{}
	if *folder != "" {
		query.Set("folder", *folder)
	}
	if *tag != "" {
		query.Set("tag", *tag)
	}
	if *mode != "" {
		query.Set("modes", *mode)
	}
	if *withoutProgrammaticHome {
		query.Set("withoutProgrammaticHome", "true")
	}

	var skills []SkillResponse
	if err := ctx.GetWithQuery("/skills", query, &skills); err != nil {
		return fmt.Errorf("failed to list skills: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(skills)
	}

	if len(skills) == 0 {
		fmt.Println("No skills found")
		return nil
	}

	fmt.Println("Skills:")
	for _, p := range skills {
		tags := ""
		if len(p.Tags) > 0 {
			tags = " [" + strings.Join(p.Tags, ", ") + "]"
		}
		rating := ""
		if p.EffectivenessRating != nil {
			rating = fmt.Sprintf(" ★%d", *p.EffectivenessRating)
		}
		fmt.Printf("  %s - %s (used %d times)%s%s [%s]\n", p.Name, p.Folder, p.UsageCount, rating, tags, p.ID)
	}
	return nil
}

// topologyRow is the operator-facing view of the one skill registry. Pack is
// returned by the registry; projection is derived from the generated target
// and therefore cannot accidentally report a merely configured target as
// resident.
type topologyRow struct {
	ID               string   `json:"id"`
	Pack             string   `json:"pack"`
	Projected        bool     `json:"projected"`
	ProjectedTargets []string `json:"projectedTargets,omitempty"`
	ProjectedBytes   int      `json:"projectedBytes,omitempty"`
	ProjectedTokens  int      `json:"projectedTokens,omitempty"`
}

func projectionDirs(value string) []string {
	seen := map[string]bool{}
	var dirs []string
	for _, raw := range strings.Split(value, string(os.PathListSeparator)) {
		dir := strings.TrimSpace(raw)
		if dir == "" || seen[dir] {
			continue
		}
		seen[dir] = true
		dirs = append(dirs, dir)
	}
	return dirs
}

// declaredProjectionDirs reads the skills directory each coding-agent resource
// declares, so an operator who passes no flag still gets the truth instead of a
// confident "nothing is projected". This mirrors the api module's
// projection.LoadTargets; the two modules cannot share the package, so the read
// is deliberately limited to the one declared field.
func declaredProjectionDirs() []string {
	root := cliutil.ResolveRepoRoot()
	if root == "" {
		return nil
	}
	resourcesDir := strings.TrimSpace(os.Getenv("VROOLI_RESOURCES_DIR"))
	if resourcesDir == "" {
		resourcesDir = filepath.Join(root, "resources")
	}
	entries, err := os.ReadDir(resourcesDir)
	if err != nil {
		return nil
	}
	home, homeErr := os.UserHomeDir()
	seen := map[string]bool{}
	var dirs []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(resourcesDir, entry.Name(), "resource.json"))
		if readErr != nil {
			continue
		}
		var doc struct {
			Storage struct {
				Entries struct {
					Skills struct {
						Path string `json:"path"`
					} `json:"skills"`
				} `json:"entries"`
			} `json:"storage"`
		}
		if json.Unmarshal(data, &doc) != nil {
			continue
		}
		path := strings.TrimSpace(doc.Storage.Entries.Skills.Path)
		if path == "" {
			continue
		}
		if strings.Contains(path, "$USER_HOME") {
			if homeErr != nil {
				continue
			}
			path = strings.ReplaceAll(path, "$USER_HOME", home)
		}
		if seen[path] {
			continue
		}
		seen[path] = true
		dirs = append(dirs, path)
	}
	sort.Strings(dirs)
	return dirs
}

func residentProjectionTokens(content []byte) int {
	residentBytes := 0
	inFrontmatter := false
	for _, raw := range strings.Split(string(content), "\n") {
		line := strings.TrimSpace(raw)
		if line == "---" {
			if inFrontmatter {
				break
			}
			inFrontmatter = true
			continue
		}
		if !inFrontmatter {
			continue
		}
		for _, key := range []string{"name:", "description:"} {
			if strings.HasPrefix(line, key) {
				residentBytes += len(strings.TrimSpace(strings.TrimPrefix(line, key)))
				break
			}
		}
	}
	return (residentBytes + 3) / 4
}

func cmdTopology(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("topology", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	projectionDir := fs.String("projection-dir", os.Getenv("VROOLI_SKILL_PROJECTION_DIR"), "Generated projection directory or path-list")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	configured := strings.TrimSpace(*projectionDir)
	if configured == "" {
		configured = os.Getenv("VROOLI_SKILL_PROJECTION_DIRS")
	}
	dirs := projectionDirs(configured)
	// With nothing configured, fall back to what the resources declare. An empty
	// list would otherwise report every skill as not-projected, which reads as a
	// finding rather than as the absence of one.
	declared := false
	if len(dirs) == 0 {
		dirs = declaredProjectionDirs()
		declared = len(dirs) > 0
	}

	var catalog []SkillResponse
	if err := ctx.GetWithQuery("/skills", nil, &catalog); err != nil {
		return fmt.Errorf("failed to load skill registry: %w", err)
	}
	rows := make([]topologyRow, 0, len(catalog))
	for _, skill := range catalog {
		row := topologyRow{ID: skill.ID, Pack: skill.Folder}
		for _, dir := range dirs {
			path := filepath.Join(dir, skill.ID, "SKILL.md")
			if data, err := os.ReadFile(path); err == nil {
				row.Projected = true
				row.ProjectedTargets = append(row.ProjectedTargets, dir)
				if row.ProjectedBytes == 0 {
					row.ProjectedBytes = len(data)
					row.ProjectedTokens = residentProjectionTokens(data)
				}
			}
		}
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	if *jsonOut {
		return json.NewEncoder(os.Stdout).Encode(rows)
	}
	fmt.Printf("Skills: %d\n", len(rows))
	switch {
	case len(dirs) == 0:
		fmt.Println("Projection targets: none — no directory configured and no resource declares one.")
	case declared:
		fmt.Printf("Projection targets (declared by resources): %s\n", strings.Join(dirs, ", "))
	default:
		fmt.Printf("Projection targets (configured): %s\n", strings.Join(dirs, ", "))
	}
	projected := 0
	for _, row := range rows {
		if row.Projected {
			projected++
		}
	}
	fmt.Printf("Projected: %d of %d\n", projected, len(rows))
	for _, row := range rows {
		status := "not-projected"
		if row.Projected {
			status = fmt.Sprintf("projected in %d target(s) (%d tokens)", len(row.ProjectedTargets), row.ProjectedTokens)
		}
		fmt.Printf("  %-48s pack=%-10s %s\n", row.ID, row.Pack, status)
	}
	return nil
}

func cmdShow(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: skill show <id>")
	}
	skillID := fs.Arg(0)

	var skill SkillResponse
	if err := ctx.Get(fmt.Sprintf("/skills/%s", skillID), &skill); err != nil {
		return fmt.Errorf("failed to get skill: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(skill)
	}

	fmt.Printf("Name: %s\n", skill.Name)
	fmt.Printf("ID: %s\n", skill.ID)
	fmt.Printf("Folder: %s\n", skill.Folder)
	if skill.Description != "" {
		fmt.Printf("Description: %s\n", skill.Description)
	}
	fmt.Printf("Usage Count: %d\n", skill.UsageCount)
	if skill.EffectivenessRating != nil {
		fmt.Printf("Rating: %d/5\n", *skill.EffectivenessRating)
	}
	fmt.Printf("Draft: %v\n", skill.Draft)
	fmt.Printf("Created: %s\n", skill.CreatedAt)
	fmt.Printf("Updated: %s\n", skill.UpdatedAt)
	if len(skill.Modes) > 0 {
		fmt.Printf("Modes: %s\n", strings.Join(skill.Modes, ", "))
	}
	if len(skill.Tags) > 0 {
		fmt.Printf("Tags: %s\n", strings.Join(skill.Tags, ", "))
	}
	if skill.ProgrammaticHome != nil && *skill.ProgrammaticHome != "" {
		fmt.Printf("Programmatic Home: %s\n", *skill.ProgrammaticHome)
	}
	fmt.Printf("\nContent:\n%s\n", skill.Content)
	return nil
}

// cmdRead outputs skill content or combined formatted output.
func cmdRead(ctx appctx.Context, args []string) error {
	args = reorderFlagArgs(args, map[string]bool{
		"resolve": true,
		"output":  true,
		"format":  true,
		"sep":     true,
		"scope":   true,
		"variant": true,
	})
	fs := flag.NewFlagSet("read", flag.ContinueOnError)
	resolve := fs.String("resolve", "auto", "Resolution mode (auto|id|file|name)")
	variant := fs.String("variant", "", "Read one skill's named variant content instead of the current SKILL.md")
	jsonOut := fs.Bool("json", false, "Output full JSON response")
	strict := fs.Bool("strict", false, "Fail if any identifier is missing or ambiguous")
	separator := fs.String("sep", "\n\n---\n\n", "Separator between skills")
	output := fs.String("output", "auto", "Output mode (skills|combined|both|auto)")
	format := fs.String("format", "xml", "Combined output format (xml|markdown|json)")
	copyOut := fs.Bool("copy", false, "Copy combined output to clipboard")
	withScope := fs.Bool("with-scope", false, "Include default scope skill from first skill")
	scope := fs.String("scope", "", "Explicit scope skill to include")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: skill read <identifier> [identifier...] [--variant=<variant-id>] [--resolve=auto|id|file|name] [--output=skills|combined|both|auto] [--format=xml|markdown|json] [--with-scope] [--scope=<scope-id>] [--strict] [--copy] [--json]")
	}

	if *variant != "" {
		if fs.NArg() != 1 {
			return fmt.Errorf("--variant reads exactly one skill")
		}
		var v struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Content string `json:"content"`
		}
		if err := ctx.Get(fmt.Sprintf("/skills/%s/variants/%s", fs.Arg(0), *variant), &v); err != nil {
			return fmt.Errorf("failed to read variant: %w", err)
		}
		if *jsonOut {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(v)
		}
		fmt.Println(v.Content)
		return nil
	}

	req := ReadRequest{
		Identifiers: fs.Args(),
		Resolve:     *resolve,
		Output:      strings.ToLower(strings.TrimSpace(*output)),
		Format:      strings.ToLower(strings.TrimSpace(*format)),
		WithScope:   *withScope,
		Scope:       *scope,
	}

	var resp ReadResponse
	if err := ctx.Post("/skills/read", req, &resp); err != nil {
		return fmt.Errorf("failed to read skills: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	effectiveOutput := resp.Output
	if effectiveOutput == "" {
		effectiveOutput = req.Output
	}
	printCombined := outputIncludesCombined(effectiveOutput)
	printSkills := outputIncludesSkills(effectiveOutput)

	printed := false
	if printCombined && resp.Combined != "" {
		fmt.Print(resp.Combined)
		printed = true
	}

	if printSkills {
		for i, skill := range resp.Skills {
			if printed || i > 0 {
				fmt.Print(*separator)
			}
			fmt.Print(skill.Content)
			printed = true
		}
	}

	if len(resp.Missing) > 0 {
		var ids []string
		for _, miss := range resp.Missing {
			ids = append(ids, miss.Identifier)
		}
		fmt.Fprintf(os.Stderr, "\nMissing skills: %s\n", strings.Join(ids, ", "))
	}
	if len(resp.Ambiguous) > 0 {
		var ids []string
		for _, amb := range resp.Ambiguous {
			ids = append(ids, amb.Identifier)
		}
		fmt.Fprintf(os.Stderr, "\nAmbiguous skills: %s\n", strings.Join(ids, ", "))
	}

	if *strict && (len(resp.Missing) > 0 || len(resp.Ambiguous) > 0) {
		return fmt.Errorf("one or more skills were missing or ambiguous")
	}

	if printCombined && *copyOut && resp.Combined != "" && clipboard.IsAvailable() {
		if errMsg := clipboard.Copy(resp.Combined); errMsg == "" {
			fmt.Printf("\n(Copied to clipboard via %s)\n", clipboard.ToolName())
		} else {
			fmt.Printf("\n(%s)\n", errMsg)
		}
	}

	return nil
}

func cmdAdd(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	folder := fs.String("folder", "local", "Folder to create skill in (local|drafts|core). 'local' is the default for new skills; 'core' is reserved for foundational skills that have proven their value and should be opted into deliberately.")
	description := fs.String("description", "", "Skill description")
	draft := fs.Bool("draft", false, "Mark as draft")
	tags := fs.String("tags", "", "Comma-separated tags")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: skill add <name> [--folder=local|drafts|core] [--description=...] [--tags=...] [--draft]")
	}
	name := fs.Arg(0)

	if *folder != "local" && *folder != "drafts" && *folder != "core" {
		return fmt.Errorf("folder must be 'local', 'drafts', or 'core'")
	}

	// Get content from stdin
	fmt.Println("Enter skill content (end with Ctrl+D on a new line):")
	reader := bufio.NewReader(os.Stdin)
	var lines []string
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		lines = append(lines, line)
	}
	content := strings.TrimSpace(strings.Join(lines, ""))
	if content == "" {
		return fmt.Errorf("skill content is required")
	}

	var tagList []string
	if *tags != "" {
		tagList = strings.Split(*tags, ",")
		for i, t := range tagList {
			tagList[i] = strings.TrimSpace(t)
		}
	}

	req := CreateSkillRequest{
		Name:        name,
		Description: *description,
		Content:     content,
		Folder:      *folder,
		Draft:       *draft,
		Tags:        tagList,
		Modes:       []string{},
	}

	var skill SkillResponse
	if err := ctx.Post("/skills", req, &skill); err != nil {
		return fmt.Errorf("failed to create skill: %w", err)
	}

	fmt.Printf("Created skill: %s [%s] in %s/\n", skill.Name, skill.ID, skill.Folder)
	return nil
}

func cmdUpdate(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	name := fs.String("name", "", "New name")
	description := fs.String("description", "", "New description")
	content := fs.String("content", "", "New content (or use stdin)")
	tags := fs.String("tags", "", "Comma-separated tags (replaces existing)")
	draft := fs.Bool("draft", false, "Mark as draft")
	undraft := fs.Bool("undraft", false, "Unmark as draft")
	programmaticHome := fs.String("programmatic-home", "", "Record-of-fact pointer (format 'engine:identifier', e.g. 'test-genie:architecture') marking that this skill's detection has graduated to a programmatic engine")
	clearProgrammaticHome := fs.Bool("clear-programmatic-home", false, "Clear the programmatic-home pointer (detection is agentic again / never graduated)")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: skill update <id> [--name=...] [--description=...] [--content=...] [--tags=...] [--draft|--undraft] [--programmatic-home=engine:id|--clear-programmatic-home]")
	}
	skillID := fs.Arg(0)

	req := UpdateSkillRequest{}
	if *name != "" {
		req.Name = name
	}
	if *description != "" {
		req.Description = description
	}
	if *content != "" {
		req.Content = content
	}
	if *tags != "" {
		tagList := strings.Split(*tags, ",")
		for i, t := range tagList {
			tagList[i] = strings.TrimSpace(t)
		}
		req.Tags = tagList
	}
	if *draft {
		d := true
		req.Draft = &d
	}
	if *undraft {
		d := false
		req.Draft = &d
	}
	if *clearProgrammaticHome {
		req.ClearProgrammaticHome = true
	} else if *programmaticHome != "" {
		req.ProgrammaticHome = programmaticHome
	}

	var skill SkillResponse
	if err := ctx.Put(fmt.Sprintf("/skills/%s", skillID), req, &skill); err != nil {
		return fmt.Errorf("failed to update skill: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(skill)
	}

	fmt.Printf("Updated skill: %s [%s]\n", skill.Name, skill.ID)
	return nil
}

func cmdDelete(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	force := fs.Bool("force", false, "Skip confirmation prompt")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: skill delete <id> [--force]")
	}
	skillID := fs.Arg(0)

	// Get skill info first for confirmation
	var skill SkillResponse
	if err := ctx.Get(fmt.Sprintf("/skills/%s", skillID), &skill); err != nil {
		return fmt.Errorf("failed to get skill: %w", err)
	}

	if !*force {
		fmt.Printf("Delete skill %q (%s)? [y/N]: ", skill.Name, skillID)
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Cancelled")
			return nil
		}
	}

	if err := ctx.Delete(fmt.Sprintf("/skills/%s", skillID)); err != nil {
		return fmt.Errorf("failed to delete skill: %w", err)
	}

	fmt.Printf("Deleted skill: %s\n", skill.Name)
	return nil
}

func cmdSync(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("sync", flag.ContinueOnError)
	tag := fs.String("tag", "", "Filter by tag")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	query := url.Values{}
	if *tag != "" {
		query.Set("tag", *tag)
	}

	var resp SyncResponse
	if err := ctx.GetWithQuery("/skills/sync", query, &resp); err != nil {
		return fmt.Errorf("failed to sync: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	fmt.Printf("Synced %d skills\n", len(resp.Skills))
	fmt.Printf("Last Updated: %s\n", resp.LastUpdated)
	fmt.Printf("Hash: %s\n", resp.Hash)
	return nil
}

func cmdRate(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("rate", flag.ContinueOnError)
	notes := fs.String("notes", "", "Optional notes about the rating")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 2 {
		return fmt.Errorf("usage: skill rate <id> <1-5> [--notes=...]")
	}

	skillID := fs.Arg(0)
	ratingStr := fs.Arg(1)

	rating, err := strconv.Atoi(ratingStr)
	if err != nil || rating < 1 || rating > 5 {
		return fmt.Errorf("rating must be between 1 and 5")
	}

	req := struct {
		Rating int     `json:"rating"`
		Notes  *string `json:"notes,omitempty"`
	}{
		Rating: rating,
	}
	if *notes != "" {
		req.Notes = notes
	}

	if err := ctx.Put(fmt.Sprintf("/skills/%s/rating", skillID), req, nil); err != nil {
		return fmt.Errorf("failed to set rating: %w", err)
	}

	fmt.Printf("Rated skill %s: %d/5\n", skillID, rating)
	return nil
}

func cmdVersions(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("versions", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: skill versions <id>")
	}
	skillID := fs.Arg(0)

	var resp VersionResponse
	if err := ctx.Get(fmt.Sprintf("/skills/%s/versions", skillID), &resp); err != nil {
		return fmt.Errorf("failed to get versions: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	fmt.Printf("Version History for %s (current: v%d):\n", resp.SkillID, resp.Current)
	for _, v := range resp.Versions {
		current := ""
		if v.Version == resp.Current {
			current = " (current)"
		}
		fmt.Printf("  v%d - %s - %s%s\n", v.Version, v.UpdatedAt, v.Name, current)
	}
	return nil
}

func cmdRevert(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("revert", flag.ContinueOnError)
	force := fs.Bool("force", false, "Skip confirmation prompt")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 2 {
		return fmt.Errorf("usage: skill revert <id> <version> [--force]")
	}

	skillID := fs.Arg(0)
	versionStr := fs.Arg(1)

	version, err := strconv.Atoi(versionStr)
	if err != nil {
		return fmt.Errorf("version must be a number")
	}

	if !*force {
		fmt.Printf("Revert skill %s to version %d? [y/N]: ", skillID, version)
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Cancelled")
			return nil
		}
	}

	var resp struct {
		SkillID    string `json:"skillId"`
		RevertedTo int    `json:"revertedTo"`
		NewVersion int    `json:"newVersion"`
		RestoredAt string `json:"restoredAt"`
	}
	if err := ctx.Post(fmt.Sprintf("/skills/%s/revert/%d", skillID, version), struct{}{}, &resp); err != nil {
		return fmt.Errorf("failed to revert: %w", err)
	}

	fmt.Printf("Reverted to version %d (new version: %d)\n", resp.RevertedTo, resp.NewVersion)
	return nil
}

// VariantResponse matches the API response for variants
type VariantResp struct {
	ID          string `json:"id"`
	SkillID     string `json:"skillId"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Content     string `json:"content,omitempty"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
	Revision    int    `json:"revision"`
}

func cmdVariants(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("variants", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: skill variants <id> [--json]")
	}
	skillID := fs.Arg(0)

	var variants []VariantResp
	if err := ctx.Get(fmt.Sprintf("/skills/%s/variants", skillID), &variants); err != nil {
		return fmt.Errorf("failed to list variants: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(variants)
	}

	if len(variants) == 0 {
		fmt.Println("No variants found")
		return nil
	}

	fmt.Printf("Variants for skill %s:\n", skillID)
	for _, v := range variants {
		desc := ""
		if v.Description != "" {
			desc = " - " + v.Description
		}
		fmt.Printf("  %s  %s%s  (updated %s) [%s]\n", v.ID, v.Name, desc, v.UpdatedAt, v.SkillID)
	}
	return nil
}

func cmdAddVariant(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("add-variant", flag.ContinueOnError)
	name := fs.String("name", "", "Variant name (required)")
	file := fs.String("file", "", "Path to file containing variant content")
	content := fs.String("content", "", "Variant content (alternative to --file)")
	description := fs.String("description", "", "Variant description")
	variantID := fs.String("id", "", "Variant ID (auto-generated if omitted)")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: skill add-variant <skill-id> --name NAME (--file FILE | --content CONTENT)")
	}
	skillID := fs.Arg(0)

	if *name == "" {
		return fmt.Errorf("--name is required")
	}

	var body string
	switch {
	case *file != "":
		data, err := os.ReadFile(*file)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", *file, err)
		}
		body = string(data)
	case *content != "":
		body = *content
	default:
		return fmt.Errorf("either --file or --content is required")
	}

	id := *variantID
	if id == "" {
		id = strings.ToLower(strings.ReplaceAll(*name, " ", "-"))
	}

	req := struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description,omitempty"`
		Content     string `json:"content"`
	}{
		ID:          id,
		Name:        *name,
		Description: *description,
		Content:     body,
	}

	var variant VariantResp
	if err := ctx.Post(fmt.Sprintf("/skills/%s/variants", skillID), req, &variant); err != nil {
		return fmt.Errorf("failed to create variant: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(variant)
	}

	fmt.Printf("Created variant: %s [%s] for skill %s\n", variant.Name, variant.ID, variant.SkillID)
	return nil
}

func cmdRmVariant(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("rm-variant", flag.ContinueOnError)
	force := fs.Bool("force", false, "Skip confirmation prompt")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 2 {
		return fmt.Errorf("usage: skill rm-variant <skill-id> <variant-id> [--force]")
	}
	skillID := fs.Arg(0)
	variantID := fs.Arg(1)

	if !*force {
		fmt.Printf("Delete variant %q from skill %s? [y/N]: ", variantID, skillID)
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Cancelled")
			return nil
		}
	}

	if err := ctx.Delete(fmt.Sprintf("/skills/%s/variants/%s", skillID, variantID)); err != nil {
		return fmt.Errorf("failed to delete variant: %w", err)
	}

	fmt.Printf("Deleted variant %s from skill %s\n", variantID, skillID)
	return nil
}

func outputIncludesSkills(output string) bool {
	switch output {
	case "skills", "both":
		return true
	default:
		return false
	}
}

func outputIncludesCombined(output string) bool {
	switch output {
	case "combined", "both":
		return true
	default:
		return false
	}
}

func reorderFlagArgs(args []string, flagsWithValues map[string]bool) []string {
	if len(args) == 0 {
		return args
	}
	var flagArgs []string
	var positional []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flagArgs = append(flagArgs, arg)
			name := strings.TrimLeft(arg, "-")
			if eq := strings.IndexRune(name, '='); eq != -1 {
				name = name[:eq]
			}
			if flagsWithValues[name] && !strings.Contains(arg, "=") && i+1 < len(args) {
				flagArgs = append(flagArgs, args[i+1])
				i++
			}
			continue
		}
		positional = append(positional, arg)
	}

	return append(flagArgs, positional...)
}
