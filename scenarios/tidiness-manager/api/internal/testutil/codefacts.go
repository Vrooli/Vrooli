// Package testutil provides shared Code Facts protobuf builders for API tests.
package testutil

import (
	facts "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	common "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

type GraphUnit struct {
	ID       string
	Language string
	RootPath string
	Nodes    []*common.CodeGraphNode
}

func GraphNode(id, name, path string, attributes map[string]string) *common.CodeGraphNode {
	return &common.CodeGraphNode{Id: id, Name: name, Path: path, Attributes: attributes}
}

func GraphReport(targetRoot string, units ...GraphUnit) *facts.CodeFactsReport {
	report := &facts.CodeFactsReport{Target: &facts.TargetContext{RootPath: targetRoot}}
	for _, unit := range units {
		report.Graphs = append(report.Graphs, &facts.LanguageGraph{
			ParseUnit: &facts.ParseUnit{Id: unit.ID, Language: unit.Language, RootPath: unit.RootPath},
			Graph:     &common.CodeGraph{Nodes: unit.Nodes},
		})
	}
	return report
}
