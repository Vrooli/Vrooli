package recall

import "time"

type Node struct {
	ID, ParentID, EntryID, FacetID, Text string
	// Vectors holds one embedding per derived facet text — topic, rule and
	// entities for a leaf, and a single summary embedding for a summary. They
	// share a model and differ only in how the source text was framed, so a
	// query embedding is comparable to each. Scoring takes the best space, which
	// is what lets a query match on entities when it does not match on topic.
	Vectors                   [][]float64
	Depth, Span               int
	CreatedAt                 time.Time
	Pinned, Frontier, Summary bool
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
	// Config carries resolved policy values. FacetBudgets counts ENTRIES per
	// facet — it is a residency ceiling, not a size ceiling. WakeBudget and
	// MaxEntryLines are the two size ceilings: the first bounds the whole
	// ambient view, the second bounds any single memory's contribution to it so
	// one verbose entry cannot crowd out an entire facet.
	Config struct {
		FrontierTarget, WakeBudget int
		MaxEntryLines              int
		FacetBudgets               map[string]int
	}
)
