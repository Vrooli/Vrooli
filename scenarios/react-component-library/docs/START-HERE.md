# React Component Library: start here

The product loop is: choose a catalog asset, open a draft, edit its
implementation and story, run the single catalog build, validate the changed
asset, and publish a new immutable version.

## First session

Confirm the scenario is healthy:

```bash
vrooli scenario start react-component-library
react-component-library status
```

Read [Asset derivation](concepts/ASSET-DERIVATION.md) for the ownership rule
between catalog intent, implementation source, and generated outputs. Then
read the [asset update flow](guides/asset-update-flow.md).

## Update an asset

```bash
react-component-library components draft-begin react-component-library:Button
react-component-library catalog build
react-component-library catalog gates types --asset-id controls.button
react-component-library components draft-publish react-component-library:Button
```

The draft is the only writable version. A released version is immutable and
must never be edited in place. The catalog declaration is the home for desired
metadata; version source owns behavior, story, and co-located styling.

## Validate and troubleshoot

Use the smallest relevant command first:

```bash
react-component-library catalog build --check
react-component-library catalog shape-census
react-component-library catalog duplication-census
react-component-library catalog gates types --asset-id controls.button
```

Fix the authored source named by a finding, rebuild derived artifacts, and
rerun the affected gate. A gate that inspects zero files is a runner failure,
not a passing result. Do not add an exemption to hide a finding.

For the complete scenario workflow, run:

```bash
vrooli scenario test react-component-library
```

The package build is separate from the scenario UI build. It emits declarations
and JavaScript while inlining authored asset stylesheets; no stylesheet file
is required from a consumer bundler.
