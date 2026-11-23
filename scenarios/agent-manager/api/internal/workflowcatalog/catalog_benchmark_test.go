package workflowcatalog

import (
	"fmt"
	"testing"

	"agent-manager/internal/domain"
)

// BenchmarkValidateContinuationFanout tracks cost when many continuation nodes
// share one prior run, a common workflow shape. Run with -benchmem to expose
// repeated graph traversal allocations as well as elapsed time.
func BenchmarkValidateContinuationFanout(b *testing.B) {
	for _, count := range []int{500, 1500, 3000} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			d := validDefinition()
			d.Nodes = []domain.WorkflowNode{{ID: "start", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "start"}}}
			d.Edges = nil
			for i := 0; i < count; i++ {
				id := fmt.Sprintf("c%d", i)
				d.Nodes = append(d.Nodes, domain.WorkflowNode{ID: id, Kind: domain.WorkflowNodeContinue, Continue: &domain.WorkflowContinueNode{ConversationFromNode: "start", PromptTemplate: "continue"}})
				from := "start"
				if i > 0 {
					from = fmt.Sprintf("c%d", i-1)
				}
				d.Edges = append(d.Edges, domain.WorkflowEdge{From: from, To: id})
			}
			d.Nodes = append(d.Nodes, domain.WorkflowNode{ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}})
			d.Edges = append(d.Edges, domain.WorkflowEdge{From: fmt.Sprintf("c%d", count-1), To: "done"})
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := Validate(d, nil); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
