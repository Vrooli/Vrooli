# Desktop browser closure support

The BAS desktop bundle has one runtime owner for the driver: the API sidecar
supervisor. It receives `PLAYWRIGHT_BROWSERS_PATH=resources/playwright`, and
scenario-to-desktop resolves that path against the installed bundle root. The
bundle manifest requires the directory as an asset; it cannot silently use a
developer's `~/.cache/ms-playwright`. The driver selects the Chromium revision
declared by its locked `rebrowser-playwright-core/browsers.json` inventory.

Driver compilation no longer invokes Playwright's browser downloader. The
`pnpm browsers:ensure` script is verification-only and reports the expected
revision and path. Browser bytes must be supplied by the governed dependency or
artifact acquisition owner before a complete desktop bundle can be staged.
Artifact staging and offline build-input transfer are now implemented for explicit,
fixture-fed trees. SDA still has no browser artifact route, so a clean host install
must remain unqualified.

## Support matrix

| Target | Source layout | Real artifact | Clean-host install | Status |
|---|---|---|---|---|
| Linux x64 | Fixture-tested | Unqualified; no SDA-provisioned artifact | Unqualified; no receipt | Unqualified |
| Linux arm64 | Fixture-tested | Unqualified; no SDA-provisioned artifact | Unqualified; no receipt | Unqualified; no node |
| macOS x64 | Fixture-tested | Unqualified; no SDA-provisioned artifact | Unqualified; no receipt | Unqualified; no node |
| macOS arm64 | Fixture-tested | Unqualified; no SDA-provisioned artifact | Unqualified; no receipt | Unqualified; no node |
| Windows x64 | Fixture-tested | Unqualified; no SDA-provisioned artifact | Unqualified; no receipt | Unqualified; no node |
| Windows arm64 | Unsupported; no source layout | Unsupported | Unsupported | Unsupported |
| Private hosted worker | Unsupported; no browser worker owner | Not applicable | Not applicable | Unsupported |

The checked machine-readable source is [`../bundle/browser-support-matrix.json`](../../bundle/browser-support-matrix.json).
Staging consumes that matrix and writes its exact non-qualification statuses into
`browser-closure.json`; scenario-to-desktop rejects a closure that claims a
qualified artifact or clean-host result. This records source capability separately
from real receipts and prevents fixture layouts from being rendered as supported.

For hosted workers, BAS `BackendRemote` currently rejects before invoking its
local `Session.Manager`, including when an external capability report is supplied.
Agent Manager's `SandboxLauncher` is an isolated coding-agent process path, not a
browser task owner: it has no browser closure mount, BAS run/grant/session binding,
browser task/artifact receipt, or remote browser teardown lifecycle. No second
worker scheduler was added.

The executable override is `BROWSER_EXECUTABLE_PATH`, matching the BAS driver.
The Playwright cache contract is `PLAYWRIGHT_BROWSERS_PATH`; it names the
versioned closure directory, not a host cache. Browser health, update,
rollback, offline install, uninstall and hosted-worker isolation remain
unverified until a supported clean target can receive the governed artifact.
