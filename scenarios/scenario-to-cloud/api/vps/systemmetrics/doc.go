// Package systemmetrics provides OS-aware remote VPS metric snapshot parsing.
//
// Collectors declare commands for an SSH runner to execute on the target VPS and
// then parse those remote command results into domain metrics. This package is
// not a local host inventory authority; local host probing belongs in
// internal/hostinventory.
package systemmetrics
