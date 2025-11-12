# Network Mock Node

**Requirement Tag:** `[REQ:BAS-NODE-NETWORK-MOCK]`

The Network Mock node lets BAS workflows intercept outbound HTTP(S) requests mid-run so we can stub API responses, simulate network failures, or inject deterministic latency. Patterns can match via simple globs (e.g., `https://api.example.com/*`) or regular expressions by prefixing with `regex:`. Once the rule is registered it stays active for the remainder of the execution context (tabs, pop-ups, frames) and is stored in the execution artifacts for debugging.

## Configuration

| Field | Description |
| --- | --- |
| `urlPattern` | Glob (`*`, `?`) or regex (`regex:` prefix) that evaluates against the request URL. |
| `method` | Optional HTTP verb filter. Leave empty for "any". |
| `mockType` | `response`, `abort`, or `delay`. Responses fulfill the request with custom payloads, aborts fail before hitting the network, and delays pause before letting Chrome continue. |
| `statusCode` | HTTP status to return when `mockType=response` (defaults to 200, clamped to 100-599). |
| `headers` | Key/value pairs applied to synthetic responses (one per line in the UI). |
| `body` | Raw text or JSON serialized when responding. Stored preview appears in execution debug context. |
| `delayMs` | Optional delay before responding/passthrough. Required when `mockType=delay`. |
| `abortReason` | Chrome Fetch abort reason when `mockType=abort` (Failed, BlockedByClient, TimedOut, etc.). |

## Examples

### 1. Stub login API
1. Drag **Network Mock** onto the canvas and set `urlPattern` to `https://auth.example.com/api/login`.
2. Leave method `POST`, mock type `response`, and set status to `200`.
3. Paste `{ "token": "test-token" }` into the body and optionally set `Content-Type: application/json` header.
4. Downstream nodes can now skip real authentication while still exercising UI flows.

### 2. Simulate flaky dependency
1. Add a delay mock with `urlPattern` `https://api.example.com/search*`, `mockType=delay`, and `delayMs=5000`.
2. Follow up with assertions that UI shows "Loading" state for at least 5 seconds.

### 3. Force gateway failures
1. Duplicate the node, switch to `mockType=abort`, and choose `BlockedByClient` or `TimedOut` as the abort reason.
2. Use conditionals to verify the UI surfaces retry banners or fallback content when the API fails instantly.

## Limitations & Notes
- Chrome's Fetch domain intercepts requests before they reach the network; CORS/mixed-content rules still apply if the page's JS tries to read blocked URLs.
- Interception is only enabled when at least one Network Mock node exists to avoid overhead; rules apply to all subsequent tabs/popups opened after registration.
- Regular expressions should be prefixed with `regex:` to distinguish them from glob patterns.
- Large response bodies are base64 encoded under the hood; keep fixtures concise for maintainability.
