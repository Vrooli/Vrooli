# BAS desktop browser closure

The desktop bundle's `asset_dirs: ["resources/playwright"]` is a runtime transfer declaration, not an acquisition step. SDA currently has no Chromium/browser artifact route. Once an artifact is provisioned by an admitted owner, stage it before calling scenario-to-desktop's packager by running from `playwright-driver`:

```sh
pnpm bundle:stage-browser --artifact-root /path/to/admitted/playwright-artifacts --output-dir ../resources/playwright
```

The source root must contain the exact `chromium`, `chromium-headless-shell`, and `ffmpeg` revision directories pinned by the installed `rebrowser-playwright-core/browsers.json`. Staging implements Playwright's Linux x64/arm64, macOS x64/arm64 and Windows x64 executable layouts, including host-platform revision overrides (for example ffmpeg on macOS 12). It consumes [`browser-support-matrix.json`](./browser-support-matrix.json), rejects missing/mismatched revisions, non-executable POSIX artifacts, links, unsupported platform/host-platform pairs, and the implicit developer browser cache. It copies to BAS `resources/playwright` and emits `browser-closure.json` with one `runtime_identity`: engine, exact browser revision, `rebrowser-playwright-core` driver version, deterministic bundle version and digest of every staged payload path/mode/content. The executable path and its own checksum remain separate fields for the single launch contract. Runtime revalidates the identity and payload digest alongside explicit source-layout/artifact/clean-host statuses. Fixtures prove the layouts and cross-language digest framing; actual browser artifacts on these platforms are not qualified.

## Offline transfer between build machines

After staging on the artifact-provisioning machine, create a deterministic offline archive and detached complete manifest:

```sh
cd scenarios/browser-automation-studio/playwright-driver
pnpm bundle:offline-create -- --closure ../resources/playwright --archive /transfer/bas-playwright-linux-x64.tar.gz --manifest /transfer/bas-playwright-linux-x64.json
```

On the disconnected packaging machine, install it into the source tree before calling scenario-to-desktop:

```sh
cd scenarios/browser-automation-studio/playwright-driver
pnpm bundle:offline-install -- --archive /transfer/bas-playwright-linux-x64.tar.gz --manifest /transfer/bas-playwright-linux-x64.json --destination ../resources/playwright
```

After a verified update, the install owner retains one previously verified generation. Roll back with `pnpm bundle:offline-rollback -- --destination ../resources/playwright`. The operation validates both installed receipts, pinned identity and payload digests before swapping; repeated rollback toggles between the verified generations. Remove only a verified install with `pnpm bundle:offline-remove -- --destination ../resources/playwright`. Removal refuses symlinks, malformed or foreign receipts, mismatched identity, and locally changed payloads. These are explicit local file operations; they do not acquire or launch a browser.

The detached manifest binds archive SHA-256 and size, every regular member's relative path/mode/size/SHA-256, the same `runtime_identity`, OS/architecture, Playwright host platform and executable binding. The installer checks expected driver and browser pins from the local package metadata, verifies the complete archive and every member before extraction, recomputes the runtime identity digest, rejects links/traversal/duplicates/partial archives, extracts into a private sibling and commits with a rollback directory. The installed receipt carries the exact same identity and archive digest. An archive digest already recorded at the destination is rejected as a replay. Failed verification leaves the previous closure untouched. The shared implementation and fixture tests are in `packages/browser-closure`; the CLI only derives the expected package lock and delegates to that owner.

This is an offline *build-input transfer* path. Current fixture evidence is Linux x64 only. It does not replace desktop end-user installation, authenticate an external acquisition source, or prove the offline desktop user journey. The archive hash detects transfer corruption; signed provenance/acquisition remains with the existing SDA authority and is still unverified here.

Then package the BAS scenario using scenario-to-desktop. Its existing `asset_dirs` copier carries the staged closure into the desktop bundle. On startup, the scenario-to-desktop runtime validates the same runtime identity and full payload digest, platform, path containment, executable bit and executable checksum before setting the sole `BROWSER_EXECUTABLE_PATH`. The Playwright driver and runtime do not download browsers, inspect a user cache, search the host, or fall back to Electron Chromium.

The source fixture status applies only to Linux x64. Linux arm64, macOS x64/arm64 and Windows x64 are `unqualified-no-node`; Windows arm64 has no layout and remains unsupported. The same support matrix records the private hosted worker as unsupported until an existing owner provides the browser-task lifecycle and receipt contract. No platform has a real artifact or clean-host receipt; source layout fixtures do not qualify executable compatibility, OS dependencies, clean-host installation, launch or update. No browser artifact was acquired or launched by these checks.

Support matrix schema v3 records delivery channels and per-target runtime launch, browser task, Web Console pane, update/rollback, and uninstall status end-to-end. Linux x64 lifecycle cells remain `unqualified-no-receipt`; targets without a node remain `unqualified-no-node`; Windows arm64 remains `unsupported-no-layout` and the private hosted worker remains unsupported. Online acquisition is `unsupported-no-governed-browser-source` until SDA provides an approved browser resource/composite closure route. Offline archive/install is `source-fixture-tested-linux-x64`, scoped to deterministic fixtures run on Linux x64; its installed receipt does not qualify a real browser artifact or clean host. The scenario-to-desktop prelaunch verifier now requires the exact install receipt and rejects missing, tampered or stale receipts before setting `BROWSER_EXECUTABLE_PATH`. Runtime rejects channel status promotion, and the offline manifest/receipt retain the same explicit limits. `packages/browser-closure` also retains one prior verified generation for source-level rollback and performs receipt/digest-gated removal; neither behavior has a real host receipt.
