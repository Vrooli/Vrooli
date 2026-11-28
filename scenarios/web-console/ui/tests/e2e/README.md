# web-console browser E2E probes

Reproducers that drive the real browser + real `web-console` + real `claude`
binary so we can iterate on terminal-rendering bugs without human-in-the-loop
screenshots.

These are **not** run by the normal `vitest` unit-test suite because they need a
live web-console, a live Claude login, and Playwright browsers downloaded.

## `claude-standard-backend.mjs`

Regression probe for the "Claude Code hangs in the standard backend" bug
(xterm.js v6 `requestMode` crash on `\x1b[?2026$p`).

Walks the full UI path:

1. Opens the web-console UI in headless Chromium or Firefox.
2. Clicks `toolbar-new` to open the launcher.
3. Expands "Session Options" and forces `backend=standard` via
   `launcher-backend-select` (crucial — the launcher silently falls back to
   the user's default backend, which is usually `persistent`; a test that
   doesn't force this ends up testing a backend that was never broken).
4. Clicks `launcher-shortcut-claude-code`.
5. Verifies via the REST API that the newly created session is
   `backend=standard` — fails loudly with exit code 2 if not, instead of
   silently testing the wrong path.
6. Polls the xterm DOM for up to 15 s for `Claude Code v` + `❯` + the
   "bypass permissions on" footer, which only appear once the interactive
   UI has actually rendered.
7. Deletes the session it created.
8. Artifacts (screenshot, xterm buffer, WS frame preview, console logs)
   land in `/tmp/wc-probe-<browser>/` so regressions can be diagnosed
   post-mortem.

### Running

```bash
# One-time setup in a scratch dir (outside the repo because web-console
# doesn't ship its own playwright dep):
mkdir -p /tmp/wc-e2e && cd /tmp/wc-e2e
npm init -y >/dev/null && npm install playwright --no-save --silent
npx playwright install chromium firefox

# Then, any time you want to repro / verify a fix:
cp <repo>/scenarios/web-console/ui/tests/e2e/claude-standard-backend.mjs .
node claude-standard-backend.mjs --browser chromium
node claude-standard-backend.mjs --browser firefox
```

### Environment

| Variable           | Default                 | Purpose                           |
|--------------------|-------------------------|-----------------------------------|
| `WC_UI_URL`        | `http://localhost:21233` | web-console UI origin             |
| `WC_API_URL`       | `http://localhost:16382` | web-console API origin            |
| `CHROME_EXECUTABLE`| `/usr/bin/google-chrome` | override Chromium binary path     |

### Exit codes

| Code | Meaning                                                          |
|------|------------------------------------------------------------------|
| 0    | Pass — Claude banner + prompt + footer all rendered              |
| 1    | Fail — hang reproduced, artifacts in `/tmp/wc-probe-<browser>/`  |
| 2    | Test infra failure (Claude not logged in, UI structure changed,  |
|      | session was created with wrong backend, etc.)                    |

### Known limitation

No WebKit probe — Playwright WebKit needs `sudo apt install` deps that
aren't present on this dev machine. Chromium + Firefox pass is a strong
proxy because the root cause was an xterm.js-level parser crash (engine-
agnostic), not a Safari-specific WS or DOM quirk.
