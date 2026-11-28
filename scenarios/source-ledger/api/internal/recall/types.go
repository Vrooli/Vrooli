package recall

import "time"

type Node struct {
	ID, ParentID, EntryID, FacetID, Text string
	CategoryIDs                          []string
	EvidenceIDs, ReconciledIDs           []string
	Evidence, RoutingHint                string
	ActorID, RunID, SourceRuntime        string
	CaseID                               string
	Open                                 bool
	// Vectors holds the node's embeddings: one for a leaf (its topic text) and
	// one for a summary. Entries written before the single-view change carried
	// two more prefix-only views until they are pruned (see
	// journal.PruneRedundantFacetViews); scoring takes the best vector, so a
	// node scores the same with or without them to within near-ties.
	Vectors                   [][]float64
	Depth, Span               int
	CreatedAt                 time.Time
	Pinned, Frontier, Summary bool
}

type (
	CompactionLiveness struct {
		UnsummarizedLeafCount    int
		OldestUnsummarizedLeafAt string
		LastSummaryAt            string
	}

	Hit struct {
		Node        Node
		Score       float64
		Descendants []Node
	}
	Wake struct {
		Hits        []Hit
		Overflow    bool
		Refused     int
		LinesUsed   int
		CharsUsed   int
		BudgetLines int
		BudgetChars int
	}
	// Config carries resolved policy values. FacetBudgets counts ENTRIES per
	// facet — it is a residency ceiling, not a size ceiling. WakeBudget and
	// WakeBudgetChars bound the whole ambient view; MaxEntryLines and
	// MaxEntryChars bound one memory's contribution so one verbose entry cannot
	// crowd out an entire facet.
	Config struct {
		FrontierTarget               int
		WakeBudget, WakeBudgetChars  int
		MaxEntryLines, MaxEntryChars int
		FacetBudgets                 map[string]int
	}
)
