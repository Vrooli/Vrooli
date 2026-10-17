package collectors

// pagingReading is the platform boundary for cumulative paging counters.
// Rates are intentionally computed by the shared counter tracker in the
// pressure collector, so every backend has identical first-sample semantics.
type pagingReading struct {
	counters   map[string]uint64
	supported  bool
	reason     string
	provenance string
}
