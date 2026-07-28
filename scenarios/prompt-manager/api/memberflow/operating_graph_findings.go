package memberflow

import "sort"

type OperatingFindingBuilder struct {
	GraphID  string
	Team     string
	RuleID   string
	Severity Severity
}

func NewOperatingFindingBuilder(ctx RuleContext, rule Rule) OperatingFindingBuilder {
	return OperatingFindingBuilder{
		GraphID:  ctx.Block.Metadata.ID,
		Team:     ctx.Block.Metadata.Team,
		RuleID:   rule.ID(),
		Severity: rule.DefaultSeverity(),
	}
}

func (b OperatingFindingBuilder) WithNode(sourcePath string, node OperatingGraphNode, detail string) OperatingGraphFinding {
	f := b.base(sourcePath, node.SourceLine, detail)
	f.NodeID = node.ID
	switch node.Kind {
	case "member":
		f.Member = node.Value
	case "topic":
		f.Topic = node.Value
	case "decision":
		f.Decision = node.Value
	case "por":
		f.Path = node.Value
	}
	return f
}

func (b OperatingFindingBuilder) WithEdge(sourcePath string, edge OperatingGraphEdge, detail string) OperatingGraphFinding {
	f := b.base(sourcePath, edge.SourceLine, detail)
	f.Edge = edge.From + "->" + edge.To
	return f
}

func (b OperatingFindingBuilder) WithRelationship(rel OperatingRelationship, detail string) OperatingGraphFinding {
	f := b.base(rel.Source.Path, rel.Source.Line, detail)
	f.Member = rel.Member
	f.Topic = rel.Topic
	f.Decision = rel.Decision
	f.Path = rel.Path
	return f
}

func (b OperatingFindingBuilder) base(sourcePath string, line int, detail string) OperatingGraphFinding {
	return OperatingGraphFinding{
		Rule:       b.RuleID,
		Severity:   string(b.Severity),
		GraphID:    b.GraphID,
		Team:       b.Team,
		SourcePath: sourcePath,
		Line:       line,
		Detail:     detail,
	}
}

func addOperatingFindings(result *OperatingGraphValidationResult, findings []OperatingGraphFinding) {
	for _, f := range findings {
		result.Findings = append(result.Findings, f)
		switch f.Severity {
		case string(SeverityError):
			result.Errors++
		case string(SeverityWarning):
			result.Warnings++
		}
	}
}

func sortOperatingFindings(findings []OperatingGraphFinding) {
	sort.Slice(findings, func(i, j int) bool {
		a, b := findings[i], findings[j]
		if a.Severity != b.Severity {
			return a.Severity < b.Severity
		}
		if a.Rule != b.Rule {
			return a.Rule < b.Rule
		}
		return a.Detail < b.Detail
	})
}
