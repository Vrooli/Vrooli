// Package archtest holds the architecture conformance tests of the
// scenario-to-cloud API module. Every test here proves an ownership boundary
// the deletion ledger and docs/internal/SEAMS.md declare: who may reach a
// target, who may write an operation record, which routes are classified,
// what a compiled plan may carry into argv, and that documented commands
// exist. The tests read the module source with go/parser and go/ast, so a
// boundary breach is caught at `go test ./archtest/` before any runtime.
//
// Allowlists are deliberate: each entry names the package that is allowed to
// cross a boundary and the reason it is allowed to. Adding an entry is an
// architecture decision, not a test fix.
package archtest
