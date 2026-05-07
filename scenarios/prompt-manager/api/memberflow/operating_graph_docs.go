package memberflow

import (
	"regexp"
	"strings"
)

var markdownTableSeparatorRE = regexp.MustCompile(`^\s*\|?\s*:?-{3,}:?\s*(\|\s*:?-{3,}:?\s*)+\|?\s*$`)

func ExtractOperatingGraphDocs(lines []string) OperatingGraphDocs {
	return ExtractOperatingGraphDocsForGraph(lines, OperatingGraphMetadata{})
}

func ExtractOperatingGraphDocsForGraph(lines []string, meta OperatingGraphMetadata, graphs ...OperatingGraph) OperatingGraphDocs {
	resolver := NewOperatingActorResolver(meta, graphs...)
	return OperatingGraphDocs{
		TopicCatalog: extractOperatingTopicCatalog(lines, resolver),
		Decisions:    extractOperatingDecisionTable(lines, resolver),
	}
}

func extractOperatingTopicCatalog(lines []string, resolver OperatingActorResolver) OperatingTopicCatalogTable {
	table := OperatingTopicCatalogTable{}
	headerLine, rows := extractMarkdownTableAfterHeading(lines, "## Topic Catalog")
	if headerLine == 0 {
		return table
	}
	table.HeaderLine = headerLine
	table.Present = true
	header := normalizeMarkdownTableHeader(rows[0])
	index := markdownHeaderIndex(header)
	for _, rowLine := range rows[2:] {
		cells := splitMarkdownTableRow(rowLine.text)
		row := OperatingTopicCatalogRow{
			RawTopic:   cellByHeader(cells, index, "topic family"),
			Status:     cellByHeader(cells, index, "status"),
			Purpose:    cellByHeader(cells, index, "purpose"),
			SourceLine: rowLine.line,
		}
		row.Topic, row.Qualifier = parseDocsTopicToken(row.RawTopic)
		row.Writers = parseOperatingActorReferences(resolver, cellByHeader(cells, index, "owner / primary writer"))
		row.Readers = parseOperatingActorReferences(resolver, cellByHeader(cells, index, "primary readers"))
		table.Rows = append(table.Rows, row)
	}
	return table
}

func extractOperatingDecisionTable(lines []string, resolver OperatingActorResolver) OperatingDecisionTable {
	table := OperatingDecisionTable{}
	headerLine, rows := extractMarkdownTableAfterHeading(lines, "## Decisions")
	if headerLine == 0 {
		return table
	}
	table.HeaderLine = headerLine
	table.Present = true
	header := normalizeMarkdownTableHeader(rows[0])
	index := markdownHeaderIndex(header)
	for _, rowLine := range rows[2:] {
		cells := splitMarkdownTableRow(rowLine.text)
		row := OperatingDecisionRow{
			RawDecision: cellByHeader(cells, index, "decision context"),
			Purpose:     cellByHeader(cells, index, "purpose"),
			SourceLine:  rowLine.line,
		}
		row.Decision = parseInlineCodeToken(row.RawDecision)
		row.Owners = parseOperatingActorReferences(resolver, cellByHeader(cells, index, "owner"))
		table.Rows = append(table.Rows, row)
	}
	return table
}

type markdownLine struct {
	text string
	line int
}

func extractMarkdownTableAfterHeading(lines []string, heading string) (int, []markdownLine) {
	for i, line := range lines {
		if strings.TrimSpace(line) != heading {
			continue
		}
		for j := i + 1; j < len(lines); j++ {
			trimmed := strings.TrimSpace(lines[j])
			if strings.HasPrefix(trimmed, "## ") {
				break
			}
			if trimmed == "" {
				continue
			}
			if !strings.HasPrefix(trimmed, "|") {
				continue
			}
			var rows []markdownLine
			for j < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[j]), "|") {
				rows = append(rows, markdownLine{text: lines[j], line: j + 1})
				j++
			}
			if len(rows) >= 2 && markdownTableSeparatorRE.MatchString(rows[1].text) {
				return rows[0].line, rows
			}
			return 0, nil
		}
	}
	return 0, nil
}

func normalizeMarkdownTableHeader(row markdownLine) []string {
	cells := splitMarkdownTableRow(row.text)
	for i := range cells {
		cells[i] = normalizeDocsCell(cells[i])
	}
	return cells
}

func markdownHeaderIndex(header []string) map[string]int {
	out := map[string]int{}
	for i, cell := range header {
		out[cell] = i
	}
	return out
}

func splitMarkdownTableRow(row string) []string {
	row = strings.TrimSpace(row)
	row = strings.TrimPrefix(row, "|")
	row = strings.TrimSuffix(row, "|")
	parts := strings.Split(row, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func cellAt(cells []string, index int) string {
	if index < 0 || index >= len(cells) {
		return ""
	}
	return strings.TrimSpace(cells[index])
}

func cellByHeader(cells []string, index map[string]int, header string) string {
	i, ok := index[header]
	if !ok {
		return ""
	}
	return cellAt(cells, i)
}

func parseDocsTopicToken(raw string) (topic, qualifier string) {
	token := parseInlineCodeToken(raw)
	kind, parsedQualifier, value, ok := parseOperatingGraphTypedToken(token)
	if !ok || kind != OperatingGraphNodeKindTopic {
		return "", ""
	}
	return value, string(parsedQualifier)
}

func parseInlineCodeToken(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "`")
	raw = strings.TrimSuffix(raw, "`")
	raw = strings.TrimPrefix(raw, "path:")
	return strings.TrimSpace(raw)
}

func parseOperatingActorReferences(resolver OperatingActorResolver, raw string) []OperatingActorReference {
	if resolver == nil {
		resolver = DefaultOperatingActorResolver{}
	}
	return resolver.Resolve("", OperatingGraphRuntime{}, raw)
}

func normalizeDocsCell(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.Trim(s, "`")
	s = strings.TrimPrefix(s, "topic:")
	return strings.Join(strings.Fields(s), " ")
}
