# @vrooli/react-component-library

This governed in-repository package exposes the React Component Library as a
single file dependency. Use the major-scoped subpath as the house style:

```tsx
import { Button } from "@vrooli/react-component-library/Button/2";
```

It resolves to the newest non-deprecated release in that major. Use an exact
subpath such as `Button/2.2.1` only when pinning a reproduction. Exact aliases
remain available for deprecated releases. A historical major with no active
release can retain a compatibility alias until it is explicitly retired. Use the bare `Button` form only inside the library gallery; it follows
the manifest's `latest` pointer and may cross a major boundary.

Run `pnpm sync-exports` after adding, deprecating, or retiring a library
version; CI treats a stale export map as a build failure. Consumers install
this package through a `file:` dependency, which package managers materialize
as a copy. Rebuild/reinstall the governed dependency before diagnosing a
consumer that still resolves an older alias map.

Withdraw an obsolete major alias through the owner command:

```sh
react-component-library components manifest-update react-component-library:SidebarShell --retired-major-aliases 1
```

The manifest's `retiredMajorAliases` removes the named major alias while retaining
exact historical releases. The current major cannot be retired. Omitted metadata
preserves the decision; `--clear-retired-major-aliases` explicitly reverses it.
Regenerate exports and rebuild the package after changing this metadata.
SidebarShell/1 is retired; use SidebarShell/2 for current compositions.

## Runtime artifact contract

The authored catalog is mutable workbench input. It is never the runtime
dependency by itself. A package build creates a disposable candidate, validates
its package metadata, export targets, declarations, runtime files, and input
identity, then publishes one immutable verified snapshot. The runtime pointer
selects a complete snapshot only after those checks pass.

The local store is shaped as follows (the physical root is resolved from the
control-plane runtime home or an explicit test override):

```text
<runtime-home>/artifacts/react-component-library/
├── artifacts/<sha256>/
│   ├── package.json
│   ├── dist/**
│   └── .rcl-artifact.json
├── current.json
├── previous.json
└── runtime-status.json
```

`current.json` is the selection boundary. Candidate failure, interrupted
publication, or artifact corruption must not replace it. `previous.json` is a
rollback floor and is protected by retention. A consumer may use the existing
`file:` package path as a compatibility facade, but that facade may expose only
the selected verified snapshot; it must never point at an unpublished staging
directory.

Artifact identity includes the selected catalog versions and dependency
closure, source digest, package-lock digest, toolchain digest, package output
digest, and package metadata identity. Exact consumer exports are fail-closed:
the runtime may use a prior verified artifact after a newer candidate fails,
but it may not silently substitute a different version when the requested
export is absent. A successful startup in that state is reported as degraded
with both candidate and selected identities.

`runtime-status.json` is an atomic, machine-readable operator record of the
latest `verified`, `degraded`, or `failed` attempt. It includes the candidate
attempt identity, selected/candidate artifact identities, compatibility result,
and failure reason. This evidence does not turn an unverified or incompatible
artifact into an acceptable runtime dependency.

The governed build command declares `artifact_selection` for this package.
Lifecycle freshness hashes that selected-artifact pointer alongside package
inputs. Draft catalog edits therefore remain isolated from running consumers;
when a new artifact is published and selected, the pointer changes and the
compatibility facade is eligible for refresh.

Operator checks:

```bash
cat "$HOME/.vrooli/artifacts/react-component-library/current.json"
cat "$HOME/.vrooli/artifacts/react-component-library/runtime-status.json"
cd packages/react-component-library
pnpm build --runtime --strict       # candidate/publication gate
pnpm test:artifact                  # isolated artifact contract tests
```

`--strict` is the bootstrap and release-validation mode: a candidate failure
is a hard failure. Lifecycle setup uses runtime mode, which may report
`degraded` only when the selected artifact passes integrity and exact-export
compatibility checks. The `RCL_BUILD_FAIL_STAGE` and
`RCL_ARTIFACT_FAIL_STAGE` variables are test-only failure injection points; do
not use them as production recovery controls.

The first implementation intentionally retains one package-wide compiler and
does not create one npm package per component. Component-closure compilation
can be evaluated later after the snapshot, integrity, fallback, and refresh
contracts have evidence.
