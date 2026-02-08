// Package systemmetrics provides OS-aware system metric collection/parsing logic.
//
// The package is intentionally structured around collectors so support for new
// operating systems can be added by implementing Collector and wiring it in
// CollectorForOS.
package systemmetrics
