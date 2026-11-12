# Retained runtime commands

These commands use the lifecycle owner. Capture and verification never stop or
start a scenario:

- `vrooli scenario capture-runtime <name[@variant]> [--path <scenario-directory>] [--json]`
  captures the closure declared in `.vrooli/runtime-release.json`. Capture is an
  immutable filesystem snapshot, not proof that the historical process was healthy,
  that its loaded Node modules matched disk, or that its data remains compatible.
- `vrooli scenario verify-runtime <name[@variant]> <release-id> [--json]`
  verifies an exact retained manifest, files, interpreter identities, and current
  owner binding/Node closure. It uses the verified capture source path, including
  captures made with `--path`, and does not change selection.

JSON output preserves the lifecycle owner's release metadata. Human output names
the exact release ID. Errors do not emit successful release metadata. These two
commands currently support local instances only.

To select a retained runtime explicitly, use the existing restart command:

`vrooli scenario restart <name> --runtime-release <release-id> --expected-runtime-release <current-id|source>`

Both flags are required together. `source` means the working source rather than a
retained release. To return to source, provide `--runtime-release source` and the
currently selected retained ID as `--expected-runtime-release`. The lifecycle
owner checks the expected selection and verifies the retained release before stop,
under its lock. Neither `--force` nor lifecycle override flags bypass verification.
An ordinary start or restart with neither selection flag keeps durable selection.
Source CLI preparation is skipped when that lifecycle-owned selection is retained.

This is explicit operator-selected recovery. It does not add automatic rollback,
recovery after caller death, destructive retention, or a public setup-skip flag.

## Qualification and limits

The owner must author `.vrooli/runtime-release.json` before capture. Example:

```json
{"version":1,"state_contract":"external-storage-v1","paths":["api/example-api","ui/server.js","ui/package.json","ui/dist","ui/node_modules"],"interpreters":["node"]}
```

`state_contract` is a named owner compatibility contract, not a migration or data
backup. Capture excludes mutable state, credentials, environment files and
repository metadata. Service policy is retained as a normalized JSON digest only,
not copied configuration. Selection requires that digest and the current state
contract to match. A selection pointer is intent, not a running/healthy receipt.
A failed preflight may leave the selected intent pending; an ordinary subsequent
start re-verifies it. No automatic retry or fallback is started.

The supported launch subset is component executables or pinned Node script entry
points. Output, working directory and entry paths must stay in retained storage.
Node packages must have a closed declared production dependency graph, with no
external/global loader overrides. Every declared dependency rejects matching
external ancestor `node_modules` candidates at closure verification, even when an
internal directory exists: Node
can fall through when that directory has no loadable entry. Otherwise, missing
optional dependencies and explicitly optional peers are allowed. The
`--no-global-search-paths` option alone does not remove Node's ancestor lookup. Loose `.js`, `.json`, and `.node` module files
are rejected because they can shadow package directories in CommonJS resolution.
Declared `main` paths and their extension/index candidates must stay inside the
retained root, including through symlinks. These checks do not prove that every
entry is loadable or implement complete JavaScript resolution.

These are verification-time checks, not a runtime sandbox. Post-check ancestor
mutations, undeclared or dynamic imports, shared system libraries, exports,
application health, and historic modules already loaded by a running process
remain unqualified. Qualify actual behavior separately before using a release as
an operational recovery point.

Declared data/log-directory creation, arbitrary executable or conditional
lifecycle hooks, other interpreters and Baseline engagements are currently rejected.
These cases require an owner-supported packaging/state-routing extension, not an
ad hoc skip flag. Existing credentials, storage, dependencies, fences, containment,
health checks and process identity rules remain in force. Selection does not make
unknown physical executors safe to stop.

Runtime health identity uses `sha256:<release-id>` for retained bytes. The build
identity of the process observed at capture is provenance only; it is not asserted
as the origin of today's on-disk files. Existing runtime events bind the retained
ID to the new instance/generation, without a registry schema upgrade.

The expected-release guard compares current content selection under the scenario
lock. It is not an intent-revision or ABA-proof history comparison. Published
release bytes are preserved; no automatic garbage collection is added.
