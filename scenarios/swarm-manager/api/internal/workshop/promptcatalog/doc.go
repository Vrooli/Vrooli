// Package promptcatalog hosts shared prompt-template content used by
// operating-mode reconcile prompts and the initiative-feedback prompt.
//
// The package exists for one reason: the proposal envelope shape an agent
// must emit (BacklogSyncPlan / mutation_list) is identical across every
// surface that produces one — reconcile phases on holistic-loop and
// phased-plan-drain, and the initiative-feedback skill. Without a single
// source of truth, those three skills drift, and an agent fluent on one
// surface emits malformed proposals on another. The snippet returned by
// BacklogSyncProposalSnippet is injected as the BACKLOG_SYNC_PROPOSAL_SNIPPET
// template variable wherever the catalog needs the shared contract; tests
// pin both reconcile prompts and the feedback prompt against the same
// snippet so the contract cannot drift silently.
//
// DOC: docs/internal/SEAMS.md#reconcile-phase-contract
package promptcatalog
