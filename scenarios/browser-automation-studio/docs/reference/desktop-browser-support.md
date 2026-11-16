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
That acquisition-to-bundle transfer is not implemented yet; a clean host install
must therefore be treated as unqualified.

## Support matrix

| Target | Browser closure | Online install | Offline install/update | Status |
|---|---|---|---|---|
| Linux x64 | Chromium under `resources/playwright` | Not qualified | Not qualified | Unqualified; no clean-host receipt |
| Linux arm64 | No inspected closure | Not qualified | Not qualified | Unqualified; no node |
| macOS x64 | No inspected closure | Not qualified | Not qualified | Unqualified; no node |
| macOS arm64 | No inspected closure | Not qualified | Not qualified | Unqualified; no node |
| Windows x64 | No inspected closure | Not qualified | Not qualified | Unqualified; no node |
| Windows arm64 | No inspected closure | Not qualified | Not qualified | Unqualified; no node |
| Private hosted worker | No worker artifact/identity binding | Not qualified | Not applicable | Unsupported in this increment |

The executable override is `BROWSER_EXECUTABLE_PATH`, matching the BAS driver.
The Playwright cache contract is `PLAYWRIGHT_BROWSERS_PATH`; it names the
versioned closure directory, not a host cache. Browser health, update,
rollback, offline install, uninstall and hosted-worker isolation remain
unverified until a supported clean target can receive the governed artifact.
