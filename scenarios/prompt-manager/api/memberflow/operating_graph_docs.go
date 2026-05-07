package memberflow

import (
	"regexp"
	"sort"
	"strings"
)

var markdownTableSeparatorRE = regexp.MustCompile(`^\s*\|?\s*:?-{3,}:?\s*(\|\s*:?-{3,}:?\s*)+\|?\s*$`)

func ExtractOperatingGraphDocs(lines []string) OperatingGraphDocs {
	return OperatingGraphDocs{
		TopicCatalog: extractOperatingTopicCatalog(lines),
		Decisions:    extractOperatingDecisionTable(lines),
	}
}

func extractOperatingTopicCatalog(lines []string) OperatingTopicCatalogTable {
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
		row.Writers = parseOperatingActorReferences(cellByHeader(cells, index, "owner / primary writer"))
		row.Readers = parseOperatingActorReferences(cellByHeader(cells, index, "primary readers"))
		table.Rows = append(table.Rows, row)
	}
	return table
}

func extractOperatingDecisionTable(lines []string) OperatingDecisionTable {
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
		row.Owners = parseOperatingActorReferences(cellByHeader(cells, index, "owner"))
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
	if !ok || kind != "topic" {
		return "", ""
	}
	return value, parsedQualifier
}

func parseInlineCodeToken(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "`")
	raw = strings.TrimSuffix(raw, "`")
	raw = strings.TrimPrefix(raw, "path:")
	return strings.TrimSpace(raw)
}

func parseOperatingActorReferences(raw string) []OperatingActorReference {
	var refs []OperatingActorReference
	for _, part := range splitActorCell(raw) {
		ref := parseOperatingActorReference(part)
		if ref.Kind == "" && ref.Value == "" {
			continue
		}
		refs = append(refs, ref)
	}
	sort.SliceStable(refs, func(i, j int) bool {
		if refs[i].Kind != refs[j].Kind {
			return refs[i].Kind < refs[j].Kind
		}
		if refs[i].Value != refs[j].Value {
			return refs[i].Value < refs[j].Value
		}
		return refs[i].Raw < refs[j].Raw
	})
	return refs
}

func splitActorCell(raw string) []string {
	raw = strings.ReplaceAll(raw, " or ", ",")
	raw = strings.ReplaceAll(raw, " and ", ",")
	parts := strings.Split(raw, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func parseOperatingActorReference(raw string) OperatingActorReference {
	cleaned := parseInlineCodeToken(raw)
	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" {
		return OperatingActorReference{}
	}
	if kind, _, value, ok := parseOperatingGraphTypedToken(cleaned); ok {
		return OperatingActorReference{Kind: kind, Value: value, Raw: raw}
	}
	normalized := normalizeDocsCell(cleaned)
	normalized = strings.TrimSuffix(normalized, " when relevant")
	switch normalized {
	case "operator":
		return OperatingActorReference{Kind: "external", Value: "operator", Raw: raw}
	case "vision-walk", "vision walk":
		return OperatingActorReference{Kind: "external", Value: "vision-walk", Raw: raw}
	case "bookmark-intelligence-hub", "bookmark intelligence hub":
		return OperatingActorReference{Kind: "external", Value: "bookmark-intelligence-hub", Raw: raw}
	case "decision workflow", "live system", "system":
		return OperatingActorReference{Kind: "external", Value: strings.ReplaceAll(normalized, " ", "-"), Raw: raw}
	case "researcher":
		return OperatingActorReference{Kind: "member", Value: "researcher", Raw: raw}
	case "brand-manager", "brand manager":
		return OperatingActorReference{Kind: "member", Value: "brand-manager", Raw: raw}
	case "oss-advertiser", "oss advertiser":
		return OperatingActorReference{Kind: "member", Value: "oss-advertiser", Raw: raw}
	case "subscription-advertiser", "subscription advertiser":
		return OperatingActorReference{Kind: "member", Value: "subscription-advertiser", Raw: raw}
	case "publisher":
		return OperatingActorReference{Kind: "member", Value: "publisher", Raw: raw}
	case "marketing-contrarian", "marketing contrarian":
		return OperatingActorReference{Kind: "member", Value: "marketing-contrarian", Raw: raw}
	case "advertiser", "advertisers":
		return OperatingActorReference{Kind: "group", Value: "advertisers", Raw: raw}
	case "any marketing member":
		return OperatingActorReference{Kind: "group", Value: "marketing-members", Raw: raw}
	case "decision owner", "decision owners":
		return OperatingActorReference{Kind: "group", Value: "decision-owners", Raw: raw}
	case "monetization team", "monetization":
		return OperatingActorReference{Kind: "team", Value: "monetization", Raw: raw}
	case "meta-optimization", "director-swarm", "future growth analyst":
		return OperatingActorReference{Kind: "external", Value: strings.ReplaceAll(normalized, " ", "-"), Raw: raw}
	default:
		return OperatingActorReference{Kind: "unknown", Value: normalized, Raw: raw}
	}
}

func normalizeDocsCell(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.Trim(s, "`")
	s = strings.TrimPrefix(s, "topic:")
	return strings.Join(strings.Fields(s), " ")
}

func expandOperatingActorReference(ref OperatingActorReference) []OperatingActorReference {
	switch ref.Kind + ":" + ref.Value {
	case "group:advertisers":
		return []OperatingActorReference{
			{Kind: "member", Value: "oss-advertiser", Raw: ref.Raw},
			{Kind: "member", Value: "subscription-advertiser", Raw: ref.Raw},
		}
	case "group:marketing-members":
		return []OperatingActorReference{
			{Kind: "member", Value: "brand-manager", Raw: ref.Raw},
			{Kind: "member", Value: "marketing-contrarian", Raw: ref.Raw},
			{Kind: "member", Value: "oss-advertiser", Raw: ref.Raw},
			{Kind: "member", Value: "publisher", Raw: ref.Raw},
			{Kind: "member", Value: "researcher", Raw: ref.Raw},
			{Kind: "member", Value: "subscription-advertiser", Raw: ref.Raw},
		}
	case "group:decision-owners":
		return nil
	default:
		return []OperatingActorReference{ref}
	}
}
