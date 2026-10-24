package rewrite

// Schema returns "" — the rewrite domain is stateless at the database
// layer. Plans live in an in-memory PlanStore (REQ-P1-002 calls for
// SQLite persistence as a follow-up; until then there is nothing to
// migrate).
//
// The registry collects this re-export anyway so a future stateful
// turn is a one-line schema swap rather than a structural change.
func Schema() string { return "" }
