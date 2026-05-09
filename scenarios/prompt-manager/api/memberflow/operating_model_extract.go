package memberflow

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func LoadOperatingModelDocuments(repoRoot string) ([]OperatingModelDocument, error) {
	docsDir := filepath.Join(repoRoot, "docs")
	var models []OperatingModelDocument
	err := filepath.WalkDir(docsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".md") {
			return nil
		}
		rel, _ := filepath.Rel(repoRoot, path)
		found, err := ExtractOperatingModelDocuments(path, rel)
		if err != nil {
			return err
		}
		models = append(models, found...)
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	sort.Slice(models, func(i, j int) bool {
		if models[i].ID != models[j].ID {
			return models[i].ID < models[j].ID
		}
		return models[i].Source.Path < models[j].Source.Path
	})
	return models, nil
}

func ExtractOperatingModelDocuments(path, sourcePath string) ([]OperatingModelDocument, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	sections := parseOperatingModelMarkdownSections(lines)
	blocks, err := extractOperatingGraphBlocksFromLines(lines, sourcePath, sections)
	if err != nil {
		return nil, err
	}
	if operatingModelPath(sourcePath) {
		var contractCount int
		for _, block := range blocks {
			if block.Metadata.Mode == OperatingGraphModeContract {
				contractCount++
			}
		}
		if contractCount > 1 {
			return nil, fmt.Errorf("%s: canonical operating model documents support one contract graph, found %d", sourcePath, contractCount)
		}
	}
	models := make([]OperatingModelDocument, 0, len(blocks))
	for _, block := range blocks {
		models = append(models, operatingModelDocumentFromGraphBlock(block, sections))
	}
	return models, nil
}

func operatingModelPath(sourcePath string) bool {
	return strings.EqualFold(filepath.Base(sourcePath), "OPERATING_MODEL.md")
}

func operatingModelDocumentFromGraphBlock(block OperatingGraphBlock, sections map[string][]OperatingMarkdownSection) OperatingModelDocument {
	modelSections := OperatingModelSections{
		Mission:        operatingModelSection(sections, "Mission"),
		Scope:          operatingModelSection(sections, "Scope"),
		OperatingLoops: operatingModelSection(sections, "Operating Loops"),
		Graph:          OperatingGraphSection{OperatingGraphBlock: block, Heading: operatingModelGraphHeading(sections), Present: true},
		TopicCatalog:   block.Docs.TopicCatalog,
		Decisions:      block.Docs.Decisions,
		ExternalInputs: extractOperatingExternalInputsTable(sections),
		Outputs:        extractOperatingOutputsTable(sections),
		FeedbackLoop:   extractOperatingFeedbackSection(sections),
		Gaps:           extractOperatingGapsSection(sections),
		Adoption:       extractOperatingAdoptionSection(sections),
	}
	return OperatingModelDocument{
		ID:       block.Metadata.ID,
		Team:     block.Metadata.Team,
		Status:   block.Metadata.Status,
		Source:   OperatingModelSource{Path: block.Source.Path, Line: block.Source.Line},
		Sections: modelSections,
		Graphs:   []OperatingGraphBlock{block},
	}
}

func extractOperatingExternalInputsTable(sections map[string][]OperatingMarkdownSection) OperatingExternalInputsTable {
	section := operatingModelSection(sections, "External Inputs / Triggers")
	table := OperatingExternalInputsTable{OperatingMarkdownSection: section}
	if !section.Present {
		return table
	}
	headerLine, rows := extractMarkdownTableAfterHeading(operatingMarkdownSectionLines(section), "## External Inputs / Triggers")
	if headerLine == 0 {
		return table
	}
	table.HeaderLine = operatingModelAbsoluteLine(section, headerLine)
	table.Present = true
	table.Table = true
	table.Headers = normalizeMarkdownTableHeader(rows[0])
	index := markdownHeaderIndex(table.Headers)
	for _, rowLine := range rows[2:] {
		cells := splitMarkdownTableRow(rowLine.text)
		table.Rows = append(table.Rows, OperatingExternalInputRow{
			ProducerTrigger: cellByHeader(cells, index, "producer / trigger"),
			EntrySurface:    cellByHeader(cells, index, "entry surface"),
			Drainer:         cellByHeader(cells, index, "drainer"),
			RoutingRule:     cellByHeader(cells, index, "routing rule"),
			SourceLine:      operatingModelAbsoluteLine(section, rowLine.line),
		})
	}
	return table
}

func extractOperatingOutputsTable(sections map[string][]OperatingMarkdownSection) OperatingOutputsTable {
	section := operatingModelSection(sections, "Outputs / Downstream Consumers")
	table := OperatingOutputsTable{OperatingMarkdownSection: section}
	if !section.Present {
		return table
	}
	headerLine, rows := extractMarkdownTableAfterHeading(operatingMarkdownSectionLines(section), "## Outputs / Downstream Consumers")
	if headerLine == 0 {
		return table
	}
	table.HeaderLine = operatingModelAbsoluteLine(section, headerLine)
	table.Present = true
	table.Table = true
	table.Headers = normalizeMarkdownTableHeader(rows[0])
	index := markdownHeaderIndex(table.Headers)
	for _, rowLine := range rows[2:] {
		cells := splitMarkdownTableRow(rowLine.text)
		table.Rows = append(table.Rows, OperatingOutputRow{
			Output:     cellByHeader(cells, index, "output"),
			Surface:    cellByHeader(cells, index, "surface"),
			Consumer:   cellByHeader(cells, index, "consumer"),
			Purpose:    cellByHeader(cells, index, "purpose"),
			SourceLine: operatingModelAbsoluteLine(section, rowLine.line),
		})
	}
	return table
}

func extractOperatingFeedbackSection(sections map[string][]OperatingMarkdownSection) OperatingFeedbackSection {
	section := operatingModelSection(sections, "Feedback / Capability Improvement Loop")
	feedback := OperatingFeedbackSection{OperatingMarkdownSection: section}
	if !section.Present {
		return feedback
	}
	for i, line := range section.Body {
		text, ok := operatingFeedbackStepText(line)
		if !ok {
			continue
		}
		feedback.Steps = append(feedback.Steps, OperatingFeedbackStep{
			Text:       text,
			References: extractInlineCodeTokens(text),
			SourceLine: section.Line + i + 1,
		})
	}
	return feedback
}

func extractOperatingGapsSection(sections map[string][]OperatingMarkdownSection) OperatingGapsSection {
	section := operatingModelSection(sections, "Current Implementation Gaps")
	gaps := OperatingGapsSection{OperatingMarkdownSection: section}
	if !section.Present {
		return gaps
	}
	for i, line := range section.Body {
		text, ok := operatingListItemText(line)
		if !ok {
			continue
		}
		gaps.Items = append(gaps.Items, OperatingGapItem{
			Text:        text,
			References:  extractInlineCodeTokens(text),
			TargetState: operatingGapItemHasTargetState(text),
			SourceLine:  section.Line + i + 1,
		})
	}
	return gaps
}

func extractOperatingAdoptionSection(sections map[string][]OperatingMarkdownSection) OperatingAdoptionSection {
	section := operatingModelSection(sections, "Adoption / Validation")
	adoption := OperatingAdoptionSection{OperatingMarkdownSection: section}
	if !section.Present {
		return adoption
	}
	for i, line := range section.Body {
		for _, command := range extractInlineCodeCommands(line) {
			adoption.Commands = append(adoption.Commands, OperatingAdoptionCommand{
				Command:    command,
				SourceLine: section.Line + i + 1,
			})
		}
	}
	return adoption
}

func operatingMarkdownSectionLines(section OperatingMarkdownSection) []string {
	lines := make([]string, 0, len(section.Body)+1)
	lines = append(lines, "## "+section.Heading)
	lines = append(lines, section.Body...)
	return lines
}

func operatingModelAbsoluteLine(section OperatingMarkdownSection, relativeLine int) int {
	if relativeLine <= 0 {
		return 0
	}
	return section.Line + relativeLine - 1
}

func operatingFeedbackStepText(line string) (string, bool) {
	return operatingNumberedListItemText(line)
}

func operatingListItemText(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "- ") {
		text := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
		return text, text != ""
	}
	return operatingNumberedListItemText(trimmed)
}

func operatingNumberedListItemText(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	dot := strings.Index(trimmed, ".")
	if dot <= 0 {
		return "", false
	}
	for _, r := range trimmed[:dot] {
		if r < '0' || r > '9' {
			return "", false
		}
	}
	text := strings.TrimSpace(trimmed[dot+1:])
	if text == "" {
		return "", false
	}
	return text, true
}

func operatingGapItemHasTargetState(text string) bool {
	normalized := strings.ToLower(text)
	for _, token := range []string{
		"target-state",
		"target state",
		"future",
		"deferred",
		"stub",
		"not modeled",
		"not yet",
		"until",
		"should",
		"remain",
		"graduate",
		"migration",
		"transition",
		"transitional",
		"accepted",
		"add ",
	} {
		if strings.Contains(normalized, token) {
			return true
		}
	}
	return false
}

func extractInlineCodeCommands(line string) []string {
	var commands []string
	for _, token := range extractInlineCodeTokens(line) {
		if strings.HasPrefix(token, "prompt-manager ") {
			commands = append(commands, token)
		}
	}
	return commands
}

func extractInlineCodeTokens(line string) []string {
	var commands []string
	rest := line
	for {
		start := strings.Index(rest, "`")
		if start < 0 {
			break
		}
		rest = rest[start+1:]
		end := strings.Index(rest, "`")
		if end < 0 {
			break
		}
		token := strings.TrimSpace(rest[:end])
		if token != "" {
			commands = append(commands, token)
		}
		rest = rest[end+1:]
	}
	return commands
}

func operatingModelGraphHeading(sections map[string][]OperatingMarkdownSection) string {
	if operatingModelSection(sections, "Operating Graph").Present {
		return "Operating Graph"
	}
	return ""
}

func operatingModelSection(sections map[string][]OperatingMarkdownSection, heading string) OperatingMarkdownSection {
	found := sections[heading]
	if len(found) == 0 {
		return OperatingMarkdownSection{}
	}
	section := found[0]
	for _, duplicate := range found[1:] {
		section.Duplicates = append(section.Duplicates, duplicate.Line)
	}
	return section
}

func parseOperatingModelMarkdownSections(lines []string) map[string][]OperatingMarkdownSection {
	type heading struct {
		text string
		line int
	}
	var headings []heading
	inFence := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inFence = !inFence
			continue
		}
		if inFence || !strings.HasPrefix(trimmed, "## ") || strings.HasPrefix(trimmed, "### ") {
			continue
		}
		headings = append(headings, heading{text: strings.TrimSpace(strings.TrimPrefix(trimmed, "## ")), line: i + 1})
	}
	sections := map[string][]OperatingMarkdownSection{}
	for i, h := range headings {
		end := len(lines)
		if i+1 < len(headings) {
			end = headings[i+1].line - 1
		}
		bodyStart := h.line
		bodyEnd := end
		body := []string{}
		if bodyStart < bodyEnd {
			body = append(body, lines[bodyStart:bodyEnd]...)
		}
		sections[h.text] = append(sections[h.text], OperatingMarkdownSection{
			Heading: h.text,
			Line:    h.line,
			EndLine: end,
			Body:    body,
			Present: true,
		})
	}
	return sections
}

func operatingGraphBlocksFromModels(models []OperatingModelDocument) []OperatingGraphBlock {
	var blocks []OperatingGraphBlock
	for _, model := range models {
		blocks = append(blocks, model.Graphs...)
	}
	return blocks
}

func filterOperatingModelDocuments(models []OperatingModelDocument, teamFilter, idFilter string) []OperatingModelDocument {
	var out []OperatingModelDocument
	for _, model := range models {
		if teamFilter != "" && model.Team != teamFilter {
			continue
		}
		if idFilter != "" && model.ID != idFilter {
			continue
		}
		out = append(out, model)
	}
	return out
}
