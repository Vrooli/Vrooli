package promptcatalog

// BacklogSyncProposalVariableKey is the template variable name reconcile and
// feedback prompts substitute to pull in the shared proposal-envelope
// contract. Keeping the key in code (not just docs) lets tests that pin
// prompt content reference the same constant the renderer uses.
const BacklogSyncProposalVariableKey = "BACKLOG_SYNC_PROPOSAL_SNIPPET"

// backlogSyncProposalSnippet is the canonical, single-source contract for
// the BacklogSyncPlan / proposal envelope an agent emits.
//
// Every surface that produces a backlog-sync proposal — reconcile phases on
// initiative-scoped operating modes, the initiative-feedback skill, and any
// future surface that proposes mutations against an initiative's item graph
// — substitutes this snippet into its prompt under the
// BACKLOG_SYNC_PROPOSAL_SNIPPET key. The snippet describes only the wire
// shape and rules that are universal across surfaces; surface-specific
// guidance (when to use the proposal, what context to read first, how to
// frame rationales for that surface's audience) stays on the surface.
//
// Editing rules (documented here so future maintainers don't drift this
// against `proposals/types.go`):
//
//   - Op names listed in the table MUST match `proposals.AllOps()`. Adding a
//     new op is a `proposals` change first; updating this snippet second.
//   - The rationale field rules MUST stay aligned with the apply path's
//     audit-event emission — `proposals.Source.Entrypoint` carries the
//     provenance, but the per-mutation rationale is what an operator reads
//     during decision review.
//   - The `id` field rule (`m1`, `m2`, … unique per envelope) is load-bearing
//     for the existing decision UI's checklist; the IDs are the keys
//     `decision.AcceptedMutationIDs` references.
const backlogSyncProposalSnippet = "## Required output: backlog-sync proposal envelope\n\n" +
	"End your response with a fenced `json` block containing a proposal envelope. The server " +
	"parses this block and presents each mutation to the operator as a checklist item.\n\n" +
	"```json\n" +
	"{\n" +
	"  \"form\": \"mutation_list\",\n" +
	"  \"rationale\": \"One-sentence summary of why this proposal addresses the work just completed.\",\n" +
	"  \"mutations\": [\n" +
	"    {\n" +
	"      \"id\": \"m1\",\n" +
	"      \"op\": \"update_item\",\n" +
	"      \"target\": \"execute/example-item\",\n" +
	"      \"patch\": {\n" +
	"        \"description\": \"Updated description that reflects the actual work done.\"\n" +
	"      },\n" +
	"      \"rationale\": \"Round artifacts show the implementation diverged from the original description.\"\n" +
	"    }\n" +
	"  ]\n" +
	"}\n" +
	"```\n\n" +
	"### Supported ops\n\n" +
	"| Op | When to use |\n" +
	"|----|-------------|\n" +
	"| `add_item` | New work the round surfaced that doesn't exist as an item yet. |\n" +
	"| `update_item` | Metadata correction on an existing item. **Must** use `patch: {...}` with supported keys (`title`, `description`, `priority`, `tags`, `depends_on`, `effort`, `acceptance_allow`, `acceptance_deny`, `note`). Do **not** put `title` / `description` at the top level of the mutation. |\n" +
	"| `change_status` | Move a non-lifecycle-controlled status. Allowed: `backlog`, `researching`, `ready`. **Never** propose `queued`, `in_progress`, `in_review`, `review_pending`, or any terminal status (`completed`, `failed`, `needs_followup`) — those are owned by the execution / review / user-decide systems. |\n" +
	"| `change_priority` | Priority change only (1–10). |\n" +
	"| `add_edge` | New `depends_on` relationship between two items in the initiative. |\n" +
	"| `remove_edge` | Remove an existing `depends_on`. |\n" +
	"| `move_initiative` | Transfer an item out of this initiative (destination must be an existing initiative name; empty string detaches). |\n" +
	"| `archive_item` | Item is no longer relevant. **Never** use \"remove_item\" — it is not a valid op. Archive is the way. |\n" +
	"| `interrupt_in_progress` | Stop a running execution before it finishes. **Must** be its own separate mutation so the operator sees and accepts the interruption explicitly. |\n" +
	"| `split_item` | Break one item into multiple (provide `into: [...]` with ≥2 new `ItemSpec`s). Dependents of the source are **not** retargeted automatically — emit explicit `add_edge` / `remove_edge` mutations alongside the split if you want any dependent repointed. |\n" +
	"| `merge_items` | Collapse 2+ coupled items into a single new item (provide `sources: [\"kind/a\",\"kind/b\",...]` and `item: {...}` for the merged item). External edges to/from the sources auto-retarget to the merged item; intra-source edges are dropped; sources are archived. **Validation rejects** if any source is `in_progress` — emit `interrupt_in_progress` as a separate prior mutation if interruption is the operator's intent. The merged item enters as `backlog`. |\n\n" +
	"### Rules you must follow\n\n" +
	"1. **Use stable IDs on every mutation** (`m1`, `m2`, …). The operator checks or unchecks them individually; the IDs are the keys the decision payload references.\n" +
	"2. **One rationale per mutation** — explain *why* that specific change addresses the work just completed. The decision UI shows this inline.\n" +
	"3. **One rationale at the envelope level** summarizing the overall intent.\n" +
	"4. **Don't propose terminal statuses.** If you believe an item is \"done\", propose `archive_item` or surface it in prose. Terminal transitions are a separate operator decision.\n" +
	"5. **Honor the mode's BacklogSyncPolicy capabilities.** If the mode does not allow `mark_complete` or `update_scope`, do not emit mutations that need those capabilities — your proposal will be rejected at apply time.\n" +
	"6. **Empty mutations are valid.** If the round didn't surface anything that warrants a backlog change, emit `mutations: []` with a one-sentence envelope rationale explaining what you considered and ruled out.\n" +
	"7. **Never write to initiative or item files directly.** Read-only CLI (listing files, reading item spec.json) is fine; the proposal is the only change mechanism.\n"

// BacklogSyncProposalSnippet returns the canonical proposal-envelope contract
// rendered as markdown. Both reconcile prompts and the initiative-feedback
// prompt substitute the result under BACKLOG_SYNC_PROPOSAL_SNIPPET so the
// contract never diverges across surfaces.
func BacklogSyncProposalSnippet() string {
	return backlogSyncProposalSnippet
}
