# Asset update flow

Behavioral claims are current only within the specific checks in the register. Other guidance and unverified descriptions below are **design intent**, not claims of current implementation. See the [behavior claim register](../internal/TESTING.md#behavior-claim-register).

Use one governed draft for an asset change, then rebuild the consumer and inspect the rendered result. The checked boundaries are listed in [Testing](../internal/TESTING.md#behavior-claim-register); broader quality recommendations here are design intent.

## Run the scenario

From `scenarios/react-component-library`, run `make start`. Lifecycle owns compilation prerequisites, ports, health and installed CLI freshness. Use `make logs` for startup errors. Dependency installation goes through Scenario Dependency Analyzer.

## 1. Declare intent

Update the relevant `catalog/assets/` declaration when the intended capability changes. Catalog declarations and implementation source answer different questions; catalog gates compare them.

## 2. Open and edit a draft

```bash
react-component-library components draft-begin react-component-library:Button --json
cp "<returned-draft-source-path>" /tmp/Button.tsx
sha256sum "<returned-draft-source-path>"
react-component-library components content-set "<returned-component-id>" /tmp/Button.tsx --expected-sha256 "<returned-sha256>"
```

Use the returned component identity and draft source path. Record the digest before editing. Edit a temporary copy, then send it through `content-set`; use `--path story.json` or another draft companion name for that file. Never edit a release directory. A conflicting digest requires reading and reconciling the current draft.

## 3. Generate derived artifacts

`react-component-library catalog build --list-stages` lists catalog generation ownership. Run the applicable generator and check its output. Catalog generation is separate from package compilation and consumer bundling. Existing immutable dependency choices must not be silently repinned.

## 4. Validate only the changed asset

Run `react-component-library components test <library-id> --version <draft-version> --json` for the edited story subject, and the named catalog gate relevant to the change. A zero-input or unmeasured result is not a behavioral pass. `asset check <catalog-id>` provides the broader aggregate when needed; historical corpus failures must be dispositioned rather than hidden by exemptions.

## 5. Publish

```bash
react-component-library components draft-publish react-component-library:Button --json
```

Resolve reported draft failures and retry the owning operation. A published successor is immutable. A major import resolves only a compatible released version, so opening a draft alone does not change the running app's package import.

## Edit, rebuild and look

Run these steps from the repository root after publishing a library change:

```bash
(cd packages/react-component-library && node tooling/build.mjs)
(cd scenarios/react-component-library/ui && node scripts/design-token-generate.mjs)
(cd scenarios/react-component-library/ui && node scripts/library-pins-check.mjs)
(cd scenarios/react-component-library/ui && node scripts/experience-routes-check.mjs)
(cd scenarios/react-component-library/ui && ./node_modules/.bin/vite build)
react-component-library page inspect react-component-library / --wait-selector '[data-testid="catalog-asset"]' --json
```

The package build compiles source and declarations into consumer exports. The Vite build then replaces the bundle served by the running production UI server. A browser refresh alone performs neither build. For a stopped or stale API/CLI, use `make start` from the scenario directory. Inspect the returned `screenshot_path` and DOM/source report; do not infer appearance from a successful build.

The package build measured 11.809 seconds on 2026-09-08 (a dated observation, not a budget). It reported no broken version imports. Measure it again on another machine rather than treating that cost as a guarantee.

For a shared-token change, publish BaseStyles first; generate the Tokens successor to a temporary path with `--tokens-out <file> --tokens-version <draft-version>`, write it through `content-set`, and publish it. Then run the steps above and `node ui/scripts/design-token-generate.mjs --check` from the scenario root. The generator will not alter an immutable Tokens release for you.

For a UI-only change, the Vite build and capture suffice. For an app-local token probe, save `ui/src/app-tokens.css`, change `--layout-field-wide` from `11rem` to `18rem`, build the UI, and capture `/`: the header search field widens. Restore the saved value, build again and capture the restored UI. This reversible example exercises the same served-bundle boundary without creating disposable library releases. It was executed and captured on 2026-09-08.

## Adoption obligations

Use `adoptions obligations <scenario> --json` and `adoptions preflight` before changing a consumer. The target UI manifest chooses owned locations. Linking, ejection, token synchronization and publication are distinct operations; use their CLI help and the [token contract](../reference/token-contract.md).
