# Asset update flow

An asset change follows one loop: declare intent, open a draft, edit the
version source, build derived artifacts, run the changed-asset gates, and
publish a new immutable version. Never edit a released version directory.

## 1. Declare intent

Edit the asset declaration under `catalog/assets/<domain>/`. If the file is
missing, the catalog gate reports the gap.

Failure: the declaration schema fails. Recovery: fix the reported JSON path
and rerun `react-component-library catalog build --check`.

## 2. Open and edit a draft

```bash
react-component-library components draft-begin react-component-library:Button
```

Edit the returned draft directory. Keep implementation source, `story.tsx`,
and any co-located `.css` together. Do not modify a released version.

Failure: the draft is not writable or the version is released. Recovery: use
`components draft-begin` again and discard the abandoned draft through the
governed lifecycle command.

## 3. Generate derived artifacts

```bash
react-component-library catalog build
react-component-library catalog build --check
```

The generator owns locks, story contracts, package exports, and release
indexes. A stale-output error means an authored source changed without a
build; run the first command and inspect the resulting diff.

## 4. Validate only the changed asset

```bash
react-component-library catalog gates types --asset-id controls.button
```

The response includes the inspected result, finding location, rule source,
and recovery documentation. A zero-file result is a runner fault, not a pass.

Failure: a blocking finding remains. Recovery: fix the named authored source,
rerun the generator, and repeat the asset gate. Do not widen the gate or add an
allowlist to hide the finding.

## 5. Publish

```bash
react-component-library components draft-publish react-component-library:Button
```

Publishing creates a new immutable release and invalidates evidence for the
asset and its dependents. If publishing is rejected, resolve the reported
shape, story, dependency, or gate failure in the draft and repeat steps 3–5.

The scenario-owned suite remains the final workflow check:
`vrooli scenario test react-component-library`.

```bash
react-component-library catalog gates types --asset-id controls.button
```

The response includes the inspected result and the rule source and declaring
file for each finding. Run the scenario-owned suite for the complete workflow:
`vrooli scenario test react-component-library`.

## Canonical version shape

Each released version contains one authored entrypoint named after the asset,
plus `story.tsx`, `story.json`, and `dependencies.json`. A co-located
`<Asset>.css` or `<Asset>.strings.ts` file is optional. Generated JavaScript,
declarations, export maps, and release-hash projections belong to package
tooling and must not be copied into a version directory.

If a shape gate reports an extra file, remove the generated or abandoned file
from the draft, rerun the catalog build, and repeat the changed-asset gates.
If a release hash reports drift, inspect the authored file and use the
explicit migration workflow only when the historical release change is
intentional and documented.
