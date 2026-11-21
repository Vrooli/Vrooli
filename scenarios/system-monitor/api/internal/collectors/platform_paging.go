package collectors

// DOC: docs/internal/SEAMS.md (Platform paging seam)

// pagingReading is the platform boundary for paging and memory-pressure
// evidence. Rates are intentionally computed by the shared counter tracker in
// the pressure collector, so every backend has identical first-sample
// semantics; a backend only reports cumulative counters and instantaneous
// gauges.
type pagingReading struct {
	// counters holds cumulative native counters keyed by native name.
	counters map[string]uint64
	// rateCounters lists the native counters the collector converts to
	// rates and publishes under "<native>_per_second".
	rateCounters []string
	// aliases maps a native counter to its stable consumer alias
	// (swap_in, swap_out, major_faults, page_faults, page_ins, page_reads,
	// compressions, decompressions). Rates are republished as
	// "<alias>_per_second" with "<alias>_per_second_status".
	aliases map[string]string
	// unsupportedAliases explains aliases this platform cannot measure.
	unsupportedAliases map[string]string
	// gauges are instantaneous memory-pressure signals (native level,
	// paging-file usage, commit charge) keyed by signal name.
	gauges map[string]PressureSignal
	// pageSize converts page counters to bytes; 0 when unknown.
	pageSize   uint64
	supported  bool
	reason     string
	provenance string
}
