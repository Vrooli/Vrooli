# Trials outside the ordinary path

Read only the section the experiment needs.

## Copied production code

- Work in a task-owned directory outside the shared checkout, or an existing sandbox. Record its
  path. Copy only what a trial edits; prefer maintained generated code and pinned artifacts.
- Reuse the owner's interpreter and import environment, not just its path. For a managed Python
  runtime run `vrooli resource acquisition runtime-probe <resource> --imports <modules> --out
  <new-dir>` before loading models. Before a rebuild, read `vrooli resource acquisition explain
  <resource>`; a verified cache hit can be reused with `compose-candidate <resource> --out
  <new-dir> --offline`.
- Record each source path, copy, hash and purpose. No writable symlinks or shared hardlinks into
  production source. Reproduce the unmodified baseline before optimising, and keep patches against
  the frozen snapshot. Recheck current source before integrating.

## Source access and credentials

- Record the source, period, identifiers, field meanings, pagination, quota and retry limits.
  Keep source hashes separate from transformations.
- For credentials read the `secrets-manager` skill; record its opaque item reference and where
  the credential is placed, never its value or a credential-bearing URL. A stored key does not
  prove your executor can use it.
- At closure revoke task-owned sessions and leases through their owners.

## Unattended runs

- Use an existing managed execution owner. Record the run identity, absolute deadline, budget
  mode, checkpoint and cancellation route, and verify the deadline independently of the prompt.
  Prose deadlines are cooperative.
- If a hard token or time ceiling is required and the runner cannot enforce it, do not run
  unattended.
- At the deadline stop the worker and cancel its operations through their owners; report what
  remains.

## Evidence worth keeping

Keep only what the route needs: the frozen question and acceptance, budgets, acquisition ledger
(source, revision, hashes, terms, sizes, install paths), admission receipts, the trial ledger
(inputs, command, settings, versions, start and end, outcome, peaks, outputs), the review against
the baseline, cleanup readback and the decision per route. Retained Codex usage for chosen threads:
`web-console session native-usage <session-id> --threads <ids> --start <RFC3339> --end <RFC3339>
--json`; missing records stay unknown.
