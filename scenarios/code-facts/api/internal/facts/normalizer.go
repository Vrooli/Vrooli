package facts

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

type normalizer struct {
	includes map[factsv1.FactFamily]bool
}

func normalizeGraphFacts(unit *factsv1.ParseUnit, provider GraphProvider, result *GraphResult, includes []factsv1.FactFamily) ([]*factsv1.GenericFact, []*factsv1.Warning, []*factsv1.Evidence) {
	n := normalizer{includes: includeSet(includes)}
	analyzer := provider.AnalyzerName()
	var facts []*factsv1.GenericFact
	var warnings []*factsv1.Warning
	var evidence []*factsv1.Evidence

	graph := result.Graph
	if graph == nil {
		warnings = append(warnings, providerWarning(analyzer, "empty_graph", "Graph provider returned no graph payload.", factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN))
		return facts, warnings, evidence
	}
	for _, node := range graph.GetNodes() {
		family, ok := n.familyForNode(unit.GetLanguage(), node)
		if !ok || !n.includes[family] {
			continue
		}
		facts = append(facts, n.factFromNode(unit, analyzer, family, node))
	}
	for _, warning := range result.Warnings {
		warnings = append(warnings, graphWarning(analyzer, warning))
	}
	evidence = append(evidence, &factsv1.Evidence{
		Status:     factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN,
		Confidence: 1,
		Analyzer:   analyzer,
		Message:    fmt.Sprintf("Analyzed %s parse unit with %d graph node(s).", unit.GetLanguage(), len(graph.GetNodes())),
	})
	sort.SliceStable(facts, func(i, j int) bool { return facts[i].GetId() < facts[j].GetId() })
	return facts, warnings, evidence
}

func includeSet(families []factsv1.FactFamily) map[factsv1.FactFamily]bool {
	out := map[factsv1.FactFamily]bool{}
	for _, family := range families {
		out[family] = true
	}
	return out
}

func (n normalizer) familyForNode(language string, node *commonv1.CodeGraphNode) (factsv1.FactFamily, bool) {
	kind := nodeKind(node)
	switch language {
	case "go":
		switch kind {
		case "GO_NODE_KIND_IMPORT_SPEC", "go_import_spec":
			return factsv1.FactFamily_FACT_FAMILY_IMPORTS, true
		case "GO_NODE_KIND_REFERENCE", "GO_NODE_KIND_TYPE_USAGE", "go_reference", "go_type_usage":
			return factsv1.FactFamily_FACT_FAMILY_REFERENCES, true
		case "GO_NODE_KIND_CALL", "go_call":
			return factsv1.FactFamily_FACT_FAMILY_CALLS, true
		case "GO_NODE_KIND_TYPE", "GO_NODE_KIND_FUNC", "GO_NODE_KIND_VAR", "GO_NODE_KIND_CONST", "GO_NODE_KIND_INTERFACE", "GO_NODE_KIND_METHOD",
			"go_type", "go_func", "go_var", "go_const", "go_interface", "go_method":
			return factsv1.FactFamily_FACT_FAMILY_SYMBOLS, true
		}
	case "typescript":
		switch kind {
		case "TS_NODE_KIND_IMPORT_BINDING", "ts_import_binding":
			return factsv1.FactFamily_FACT_FAMILY_IMPORTS, true
		case "TS_NODE_KIND_REFERENCE", "ts_reference":
			return factsv1.FactFamily_FACT_FAMILY_REFERENCES, true
		case "TS_NODE_KIND_CALL", "TS_NODE_KIND_JSX_USAGE", "ts_call", "ts_jsx_usage":
			return factsv1.FactFamily_FACT_FAMILY_CALLS, true
		case "TS_NODE_KIND_COMPONENT", "TS_NODE_KIND_HOOK", "TS_NODE_KIND_CLASS", "TS_NODE_KIND_INTERFACE", "TS_NODE_KIND_TYPE", "TS_NODE_KIND_FUNCTION", "TS_NODE_KIND_VAR", "TS_NODE_KIND_CONST", "TS_NODE_KIND_EXPORT", "TS_NODE_KIND_RE_EXPORT",
			"ts_component", "ts_hook", "ts_class", "ts_interface", "ts_type", "ts_function", "ts_var", "ts_const", "ts_export", "ts_re_export":
			return factsv1.FactFamily_FACT_FAMILY_SYMBOLS, true
		}
	}

	switch node.GetKind() {
	case commonv1.NodeKind_NODE_KIND_FILE, commonv1.NodeKind_NODE_KIND_MODULE, commonv1.NodeKind_NODE_KIND_PACKAGE:
		return factsv1.FactFamily_FACT_FAMILY_PARSE_UNITS, true
	default:
		return factsv1.FactFamily_FACT_FAMILY_UNSPECIFIED, false
	}
}

func (n normalizer) factFromNode(unit *factsv1.ParseUnit, analyzer string, family factsv1.FactFamily, node *commonv1.CodeGraphNode) *factsv1.GenericFact {
	attrs := copyAttributes(node.GetAttributes())
	attrs["analyzer"] = analyzer
	attrs["language"] = unit.GetLanguage()
	attrs["parse_unit_id"] = unit.GetId()
	if node.GetPath() != "" {
		attrs["path"] = node.GetPath()
	}
	if node.GetName() != "" {
		attrs["name"] = node.GetName()
	}
	if attrs["kind"] == "" {
		attrs["kind"] = node.GetKind().String()
	}
	return &factsv1.GenericFact{
		Id:         analyzer + ":" + node.GetId(),
		Family:     family,
		Kind:       genericKind(unit.GetLanguage(), node),
		Subject:    factSubject(node),
		Evidence:   []*factsv1.Evidence{nodeEvidence(unit.GetLanguage(), analyzer, node)},
		Attributes: attrs,
	}
}

func nodeKind(node *commonv1.CodeGraphNode) string {
	if node == nil {
		return ""
	}
	if kind := node.GetAttributes()["kind"]; kind != "" {
		return kind
	}
	return node.GetKind().String()
}

func genericKind(language string, node *commonv1.CodeGraphNode) string {
	kind := strings.TrimPrefix(nodeKind(node), "GO_NODE_KIND_")
	kind = strings.TrimPrefix(kind, "TS_NODE_KIND_")
	kind = strings.TrimPrefix(kind, "NODE_KIND_")
	kind = strings.ToLower(kind)
	if kind == "" || kind == "unspecified" {
		kind = "node"
	}
	return language + "_" + kind
}

func factSubject(node *commonv1.CodeGraphNode) string {
	attrs := node.GetAttributes()
	for _, key := range []string{"import_path", "source_module", "callee", "referenced_name", "qualified_name", "type", "name"} {
		if value := strings.TrimSpace(attrs[key]); value != "" {
			return value
		}
	}
	if node.GetName() != "" {
		return node.GetName()
	}
	if node.GetPath() != "" {
		return node.GetPath()
	}
	return node.GetId()
}

func nodeEvidence(language, analyzer string, node *commonv1.CodeGraphNode) *factsv1.Evidence {
	ev := &factsv1.Evidence{
		Status:     factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN,
		Confidence: 1,
		Analyzer:   analyzer,
		Symbol:     node.GetName(),
		Message:    "Graph provider emitted " + genericKind(language, node) + " fact.",
	}
	rng := sourceRangeFromAttrs(node.GetPath(), node.GetAttributes())
	if rng.GetFile() != "" {
		ev.Range = rng
	}
	return ev
}

func sourceRangeFromAttrs(fallbackFile string, attrs map[string]string) *factsv1.SourceRange {
	rng := &factsv1.SourceRange{File: firstNonEmpty(attrs["file"], attrs["path"], fallbackFile)}
	rng.StartLine = atoi(attrs["start_line"])
	rng.StartColumn = atoi(attrs["start_column"])
	rng.EndLine = atoi(attrs["end_line"])
	rng.EndColumn = atoi(attrs["end_column"])
	return rng
}

func graphWarning(analyzer string, warning *commonv1.CodeGraphWarning) *factsv1.Warning {
	code := strings.ToLower(strings.TrimPrefix(warning.GetKind().String(), "CODE_GRAPH_WARNING_KIND_"))
	if code == "" || code == "unspecified" {
		code = "provider_warning"
	}
	msg := warning.GetMessage()
	if warning.GetFile() != "" {
		msg = warning.GetFile() + ": " + msg
	}
	return providerWarning(analyzer, code, msg, factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN)
}

func providerWarning(analyzer, code, message string, status factsv1.EvidenceStatus) *factsv1.Warning {
	return &factsv1.Warning{
		Code:    analyzer + "." + code,
		Message: message,
		Status:  status,
	}
}

func copyAttributes(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func atoi(raw string) int32 {
	if raw == "" {
		return 0
	}
	n, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || n < 0 {
		return 0
	}
	return int32(n)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
