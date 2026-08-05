package recall

import "time"

type Node struct {
	ID, ParentID, EntryID, FacetID, Text string
	Vector                               []float64
	Depth, Span                          int
	CreatedAt                            time.Time
	Pinned, Frontier, Summary            bool
}

type (
	Hit struct {
		Node        Node
		Score       float64
		Descendants []Node
	}
	Wake struct {
		Hits     []Hit
		Overflow bool
		Budget   int
	}
	Config struct {
		FrontierTarget, WakeBudget int
		FacetBudgets               map[string]int
	}
)
