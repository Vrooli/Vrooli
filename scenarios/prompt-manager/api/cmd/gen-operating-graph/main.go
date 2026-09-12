// Command gen-operating-graph writes each team's graph-presentation.json by
// transcribing the readability layer out of its current hand-drawn block, then
// reports what generation from declarations would still drop.
//
// It never rewrites an OPERATING_MODEL.md. Replacing a checked-in block is a
// separate, deliberate step that must happen only once the drop list holds
// nothing but presentation.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"prompt-manager/internal/memberflow"
)

func main() {
	repoRoot := "../../.."
	apply := false
	for _, arg := range os.Args[1:] {
		if arg == "--apply" {
			apply = true
			continue
		}
		repoRoot = arg
	}
	storeDir := filepath.Join(repoRoot, "scenarios", "prompt-manager", "store")

	members, err := memberflow.LoadAll(storeDir)
	fatal(err, "load members")
	contracts, err := memberflow.LoadAllTeamContracts(storeDir)
	fatal(err, "load team contracts")
	// Without the contract registry the topic catalog has no authored status or
	// purpose to carry through, and every generated row comes out blank.
	runtime := memberflow.OperatingGraphRuntime{RepoRoot: repoRoot, StoreDir: storeDir, Members: members, Contracts: contracts}

	blocks, err := memberflow.LoadOperatingGraphBlocks(repoRoot)
	fatal(err, "load blocks")

	for _, block := range blocks {
		if block.Metadata.Mode != memberflow.OperatingGraphModeContract || block.Metadata.Team == "" {
			continue
		}
		team := block.Metadata.Team
		presentation := memberflow.ExtractGraphPresentation(block)
		if err := presentation.Validate(); err != nil {
			fmt.Fprintf(os.Stderr, "%s: extracted presentation is invalid: %v\n", team, err)
			continue
		}
		path := filepath.Join(storeDir, "teams", team, "graph-presentation.json")
		encoded, err := json.MarshalIndent(presentation, "", "  ")
		fatal(err, "encode "+team)
		fatal(os.WriteFile(path, append(encoded, '\n'), 0o644), "write "+team)

		generated, err := memberflow.GenerateOperatingGraphBlock(memberflow.GenerateOperatingGraphInput{
			TeamID: team, Runtime: runtime, Presentation: presentation,
		})
		fatal(err, "generate "+team)
		lines := strings.Split(strings.TrimSpace(generated), "\n")
		parsed, err := memberflow.ParseOperatingMermaid(team, lines[1:len(lines)-1], 1)
		fatal(err, "reparse "+team)

		have := map[string]bool{}
		for _, node := range parsed.Nodes {
			have[string(node.Kind)+":"+node.Value] = true
		}
		var dropped []string
		for _, node := range block.Graph.Nodes {
			key := string(node.Kind) + ":" + node.Value
			if node.Kind != "" && node.Value != "" && !have[key] {
				dropped = append(dropped, key)
			}
		}
		sort.Strings(dropped)
		if len(dropped) > 0 && !apply {
			fmt.Printf("%-18s presentation written; would still drop %d: %v\n", team, len(dropped), dropped)
			continue
		}
		if !apply {
			fmt.Printf("%-18s presentation written; generation is lossless\n", team)
			continue
		}
		catalog := memberflow.GenerateTopicCatalogTable(memberflow.GenerateOperatingGraphInput{
			TeamID: team, Runtime: runtime, Presentation: presentation,
		})
		if err := replaceMarkdownTable(filepath.Join(repoRoot, block.Source.Path), "## Topic Catalog", catalog); err != nil {
			fmt.Fprintf(os.Stderr, "%s catalog: %v\n", team, err)
			os.Exit(1)
		}
		if err := replaceOperatingGraphBlock(filepath.Join(repoRoot, block.Source.Path), generated); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", team, err)
			os.Exit(1)
		}
		fmt.Printf("%-18s block regenerated (%d dropped: %v)\n", team, len(dropped), dropped)
	}
}

func fatal(err error, what string) {
	if err != nil {
		fmt.Fprintln(os.Stderr, what+":", err)
		os.Exit(1)
	}
}

// replaceOperatingGraphBlock swaps the fenced Mermaid diagram in place, leaving
// the surrounding prose and the prompt-manager-graph metadata comment untouched.
// The metadata carries actor groups and aliases that are authored, not derived.
func replaceOperatingGraphBlock(path, generated string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(raw), "\n")
	// Anchor on the contract metadata comment, then take the NEXT fenced
	// diagram. A document may carry an explanatory diagram before the contract
	// one — marketing-crew does — and taking the first fence overwrites the
	// explanation while leaving the contract graph untouched.
	anchor := 0
	for i, line := range lines {
		if strings.Contains(line, "prompt-manager-graph:") {
			anchor = i
			break
		}
	}
	start, end := -1, -1
	for i := anchor; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "```mermaid" && start == -1 {
			start = i
			continue
		}
		if start != -1 && strings.TrimSpace(lines[i]) == "```" {
			end = i
			break
		}
	}
	if start == -1 || end == -1 {
		return fmt.Errorf("no fenced mermaid block in %s", path)
	}
	replacement := strings.Split(strings.TrimRight(generated, "\n"), "\n")
	out := append([]string{}, lines[:start]...)
	out = append(out, replacement...)
	out = append(out, lines[end+1:]...)
	return os.WriteFile(path, []byte(strings.Join(out, "\n")), 0o644)
}

// replaceMarkdownTable swaps the table immediately following a heading, leaving
// the heading and surrounding prose untouched.
func replaceMarkdownTable(path, heading, table string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(raw), "\n")
	start := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == heading {
			start = i
			break
		}
	}
	if start == -1 {
		return nil // team has no such table
	}
	tableStart := -1
	for i := start + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "|") {
			tableStart = i
			break
		}
		if strings.HasPrefix(trimmed, "#") {
			return nil
		}
	}
	if tableStart == -1 {
		return nil
	}
	tableEnd := tableStart
	for tableEnd < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[tableEnd]), "|") {
		tableEnd++
	}
	replacement := strings.Split(strings.TrimRight(table, "\n"), "\n")
	out := append([]string{}, lines[:tableStart]...)
	out = append(out, replacement...)
	out = append(out, lines[tableEnd:]...)
	return os.WriteFile(path, []byte(strings.Join(out, "\n")), 0o644)
}
