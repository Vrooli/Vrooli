# Cookie & Storage Nodes

**Requirement Tag:** `[REQ:BAS-NODE-COOKIE-STORAGE]`

Vrooli Ascension now ships dedicated nodes for browser cookies and web storage so workflows can set up authenticated sessions, capture state between steps, and keep fixtures tidy. This document summarizes the nodes, configuration surfaces, and example usage.

## Cookie Nodes

### Set Cookie (`setCookie`)
- **Purpose:** Create or update a browser cookie before interacting with an app (e.g., bypass a login screen).
- **Key Parameters:**
  - `name`, `value`: Required cookie key/value.
  - `url` or `domain` + optional `path`: Target scope (must provide at least one of URL/domain).
  - `sameSite`: `lax`, `strict`, or `none`.
  - `secure`, `httpOnly`: Booleans mapped directly to `Network.setCookie`.
  - `ttlSeconds` or `expiresAt`: Either relative TTL or RFC3339/Unix timestamp for expiry.
  - `timeoutMs`, `waitForMs`: Execution timing controls.
- **Example:** Pre-seed `sessionId` on `https://app.example.com` with `SameSite=None` and `Secure` enabled before executing a workflow.

### Get Cookie (`getCookie`)
- **Purpose:** Read cookie values into workflow variables for assertions or re-use.
- **Key Parameters:**
  - `name`: Cookie key (required).
  - `url` / `domain` / `path`: Optional filters for multi-tenant environments.
  - `resultFormat`: `value` (default) or `object` to store the entire cookie payload.
  - `storeResult`: Variable alias (defaults to cookie name when omitted).
- **Behavior:** Stores the selected value/object in the execution context so downstream nodes can reference `{{myCookie}}` directly.

### Clear Cookie (`clearCookie`)
- **Purpose:** Remove targeted cookies or flush the entire jar between scenarios.
- **Key Parameters:**
  - `clearAll`: When true, issues `Network.clearBrowserCookies()`.
  - `name`, `url`, `domain`, `path`: Used together with `Network.deleteCookies()` for surgical deletions.
  - `timeoutMs`, `waitForMs`: Timing controls for stateful workflows.

## Storage Nodes

### Set Storage (`setStorage`)
- **Purpose:** Write deterministic state into `localStorage` or `sessionStorage` (e.g., feature flags, cached responses) before executing UI steps.
- **Key Parameters:**
  - `storageType`: `localStorage` (default) or `sessionStorage`.
  - `key`, `value`: Required key/value pair.
  - `valueType`: `text` or `json`. JSON payloads are validated and stringified in the browser context.
  - `timeoutMs`, `waitForMs`: Execution timing.

### Get Storage (`getStorage`)
- **Purpose:** Read storage values and expose them as workflow variables.
- **Key Parameters:**
  - `storageType`, `key`: Lookup target.
  - `resultFormat`: `text` (raw string) or `json` (parsed via `JSON.parse`).
  - `storeResult`: Alias for execution context (defaults to key name).

### Clear Storage (`clearStorage`)
- **Purpose:** Remove a specific key or flush storage entirely to keep environments deterministic.
- **Key Parameters:**
  - `storageType`: `localStorage`/`sessionStorage`.
  - `clearAll`: When checked, storage is fully cleared, otherwise the supplied `key` is removed.
  - `timeoutMs`, `waitForMs`: Execution timing.

## UI Availability
- **Node Palette:** Six new cards grouped under Interaction/Data so builders can drag Set/Get/Clear Cookie and Storage nodes onto the canvas quickly.
- **Workflow Builder:** React Flow components (`SetCookieNode`, `GetCookieNode`, `ClearCookieNode`, `SetStorageNode`, `GetStorageNode`, `ClearStorageNode`) expose validation, helpful defaults, and timing inputs.
- **Variable Suggestions:** Result-bearing nodes surface the `storeResult` field so values become first-class workflow variables immediately.

## Testing & Traceability
- **Go Unit Tests:** Coverage now lives in the automation stack and CDP adapters (`automation/executor/*_test.go`, `browserless/cdp/cookie_actions.go`, `browserless/cdp/storage_actions.go`), ensuring contract payloads map cleanly to Browserless.
- **CDP Exercises:** `cookie_actions.go` and `storage_actions.go` handle Chrome DevTools orchestration, including JSON parsing and expiry math.
- **UI Regression:** `ui/src/components/__tests__/NodePalette.test.tsx` asserts palette counts and discoverability.

Use these nodes to keep session-heavy workflows deterministic, seed authentication state without brittle login flows, and verify application behavior that depends on browser storage.
