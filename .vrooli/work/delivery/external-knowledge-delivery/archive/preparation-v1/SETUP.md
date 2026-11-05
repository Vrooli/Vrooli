# Google Setup and Team Activation

Team: `external-knowledge-delivery`; leader: `external-knowledge-orchestrator`.
Keep both native enabled flags false during preparation. Do not trigger a heartbeat to test preparation.

## Stage A — Operator preparation before enablement

Walk the operator through these decisions when they request setup. Current API/client UI must be verified against Google's official documentation at that time.

1. Choose the Google account and a dedicated Google Cloud project. Enable only Drive API initially; Gmail/Calendar/Pub/Sub are added when their approved slices require them.
2. Configure the OAuth consent audience appropriate to actual personal/internal use. Select minimum workable read scopes after testing existing/new descendant access. Explain broad account access separately from selected-folder ingestion.
3. Choose the client type and a callback matching the native integration-hub route. The exact URI must come from the implemented runtime; do not invent localhost ports or publish a callback before that route exists.
4. Store client secrets only through the credential authority's owner route. Do not paste tokens or secrets into this goal home, chat prompts, logs, Git or committed fixtures. Record non-secret connection/client references.
5. Select exact Drive folder IDs, recursive/exclusion/shortcut behavior, retained-text policy, allowed agent grants and initial format scope. Record the choices in the governing source.
6. Decide how the Google driver will be bootstrapped. The observed Integration Hub pilot lacks it. If live consent is required before enabling the whole team, the operator must authorize a bounded bootstrap task in the direct session or another existing native owner. That task implements and qualifies the callback/refresh/storage seam; it does not enable this team. Alternatively, explicitly permit the team to implement the driver with fixtures before pausing for consent after activation.

Do not leave a dependable unattended setup relying on a Testing-mode seven-day refresh token. Personal-use verification exceptions and production publishing posture depend on actual scopes/deployment; explain the applicable path, not a generic mandatory review checklist.

## Stage B — Native connection and live qualification

After the callback/credential driver exists and the operator authorizes connection, use integration-hub's advertised operation to initiate consent and request offline refresh. Verify actual scopes and account identity, expiry/refresh, restart persistence, revoked access and reconnect handling. A ChatGPT plugin profile check or a fixture does not satisfy this gate.

## Stage C — Qualified commission, still disabled

- Reconcile scenario contracts through governed PRD/requirements/experience routes, preserving selected versus candidate UX.
- Resolve the email design handoff and any required governing originals at exact versions. The first Drive epoch may precede email execution, but no email epoch is admissible from an unselected prompt alone.
- Reconcile Personal Planner ownership and assignment boundaries. No new team replaces its scheduling authority or active delivery effort.
- Refresh source hashes and finite leader accepted revision when the governing target changes. Use the native disabled binding update, preserving disabled state.
- Run `prompt-manager team validate-contract external-knowledge-delivery --json` and `agent-manager effort lint docs/internal/external-knowledge-delivery`.
- Run a fresh reader through the normal executor-readable route. Verify requirement/source mapping, original asset access, and the required shared-contract boundary case. Retain the receipt and explicit gaps.
- Verify team and heartbeat readbacks are disabled, exact profile/effort/source references match, and no leader runtime is dispatched. Native profile existence is not proof of a future run's admission.

## Stage D — Explicit later activation

Only when the operator asks to enable the qualified commission, use Prompt Manager's existing native activation path and its required identity/permissions/budgets/safeguards. Do not mint substitute authority or drop headers after refusal. Use existing delivery profiles and the large-effort-orchestration standard. Creating this preparation is not permission to activate it.

For a native refusal, preserve the exact redacted action/response and continue only independent documentation work. Do not claim disabled staging was denied merely because no activation was attempted.

## How the UX agent updates this handoff

Read EMAIL-DESIGN-PROMPT.md and the governing architecture. Append selected operator decisions verbatim to FEEDBACK.md. Save the selected email specification and original visual sources with version/hash/readable-location records and requirement-to-screen/state mappings. Update the email candidate in QUEUE.md and source manifest; refresh the disabled binding's accepted revision when required. Never rewrite accepted historical epochs or enable/wake the team from a design update.

If the UX agent lacks repository access, store the complete handoff and original assets in a clearly named Drive location, return exact links/versions plus decisions, and report that local team updates remain pending import. It must not claim that changing a Drive document changed Prompt Manager's native configuration.
