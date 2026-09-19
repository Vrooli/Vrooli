# Extracting Conversation Content from AI Chat Share Links

**Status:** Prior-art / spike notes. **NOT YET IMPLEMENTED.** This documents a
working manual extraction performed on **2026-06-14** against a ChatGPT share
link, so that when the conversation extractor is built (`SIG-P1-004`) the
technique is not reinvented. Treat it as a recipe to productionize, not as
shipped code.

> **Provenance.** This is the single artifact carried forward from the
> predecessor scenario `bookmark-intelligence-hub`, whose implementation was
> otherwise discarded (D-004). It was kept because it is the one thing in that
> scenario an agent could not regenerate: it took a live reverse-engineering
> session against a moving target. The technical content below is unchanged
> from the original; only the framing was updated for this scenario.

## Why this belongs here

AI chat share links are high-value saved material — an operator bookmarks a
conversation because of what was worked out in it — but their content is **not**
in the page you fetch. Naive extraction fails, and the failure mode is silent:
you get the page chrome and conclude the conversation is empty.

That silent-failure property is why this document is load-bearing rather than
merely useful. It is the observed case behind **D-019** and `SIG-P0-003`: an
empty extraction is a failure, not an empty document. The general rule was
derived from this specific gotcha.

## TL;DR

ChatGPT share pages are client-rendered, but the **full conversation is shipped
inside the initial HTML** — encoded in React Router's "turbo-stream" format (a
flattened, deduplicated object graph referenced by integer indices). You can't
read it with a markdown-extracting fetcher or by hitting the backend API
(Cloudflare 403). You **can** read it by pulling the raw HTML and decoding the
turbo-stream payload yourself. The data is there; it's just index-encoded so the
browser can hydrate without a second request.

## What does NOT work

| Approach | Result | Why |
|---|---|---|
| Markdown-extracting fetcher (readability-style) | Empty | Page is a JS shell; messages hydrate client-side. The extractor sees nav/login chrome only. |
| `GET chatgpt.com/backend-api/share/<id>` | **403** | Cloudflare bot protection. |
| `GET chatgpt.com/backend-api/public/conversation/<id>` | **403** | Same. |

Key lesson: an empty extraction is not "empty conversation" — it's "content is
in a format the extractor didn't decode." Build the source so this case is
detectable: assert at least one message was found, else flag the signal
`needs-attention` instead of importing a blank.

## What DOES work

1. **Fetch raw HTML with a browser User-Agent.** A plain `curl -A "<browser UA>"`
   of the share URL returns ~1 MB of HTML that *contains* the conversation.
2. **Locate the streamed loader data.** No `__NEXT_DATA__`. Instead the page has
   `window.__reactRouterContext.streamController.enqueue("…")` calls. The big
   one (hundreds of KB) holds the conversation as a JS string literal whose
   contents are a JSON turbo-stream payload.
3. **Decode the turbo-stream graph** (see format + decoder below).
4. **Navigate to the conversation** and read the turns.

### The turbo-stream format

The decoded payload is a single JSON **array** that acts as a flat pool of
values referenced by index. Rules:

- Index `0` is the root value.
- Strings/numbers are stored **once** in the pool and referenced by index
  (this is the dedup — `"role"` appears once even if used 50×).
- An object is encoded with `_<n>` keys: the key's real name is the string at
  index `n`, and the key's *value* is itself an index to resolve.
- Negative numbers are sentinels (treat as `undefined`/`null`).

Worked example — this payload:

```json
[{"_1":2}, "role", "user"]
```

decodes to `{"role": "user"}`:
- root = pool[0] = `{"_1":2}`
- key `_1` → name = pool[1] = `"role"`
- value `2` → resolve(2) = pool[2] = `"user"`

The one trick: ChatGPT's message tree has back-references (parent pointers), so
a naive recursive decoder loops forever. **Memoize on entry** (cache the
container before filling it) to break cycles.

### Reference decoder (Python)

The production implementation will be Go. This is kept in the original Python
because it is the artifact that was actually run and verified; translate it,
don't re-derive it.

```python
import re, json

def extract_turbo_stream_payloads(html: str) -> list[str]:
    # Each enqueue("...") arg is a JS string literal; json.loads decodes the
    # outer literal to the inner JSON text. Return largest-first.
    calls = re.findall(r'streamController\.enqueue\(("(?:[^"\\]|\\.)*")\)', html, re.DOTALL)
    payloads = [json.loads(c) for c in calls]
    return sorted(payloads, key=len, reverse=True)

def decode_turbo_stream(payload: str):
    rows = json.loads(payload)
    cache = {}

    def resolve(v):
        if isinstance(v, bool):      # bool is a subclass of int — check first
            return v
        if isinstance(v, int):       # values are index references
            return deref(v)
        return v                     # literal string/float/None

    def deref(idx):
        if idx < 0:                  # sentinel: undefined/null/etc.
            return None
        if idx in cache:             # memoize -> breaks parent-pointer cycles
            return cache[idx]
        node = rows[idx]
        if isinstance(node, dict):
            out = {}; cache[idx] = out
            for k, v in node.items():
                key = deref(int(k[1:])) if k.startswith("_") else k
                out[key] = resolve(v)
            return out
        if isinstance(node, list):
            out = []; cache[idx] = out
            for x in node:
                out.append(resolve(x))
            return out
        cache[idx] = node
        return node

    return deref(0)
```

### Navigating to the conversation

```python
html = open("share.html", encoding="utf-8", errors="replace").read()
root = decode_turbo_stream(extract_turbo_stream_payloads(html)[0])

data = (root["loaderData"]
            ["routes/share.$shareId.($action)"]
            ["serverResponse"]["data"])

# data also has: title, create_time, update_time, conversation_id, mapping,
# current_node, linear_conversation, default_model_slug, safe_urls, ...

turns = []
for node in data["linear_conversation"]:
    msg = node.get("message")
    if not msg:
        continue
    role = (msg.get("author") or {}).get("role")          # user | assistant | system | tool
    content = msg.get("content") or {}
    if content.get("content_type") == "text":             # skip thoughts/code/tool ctypes for clean text
        text = "\n".join(p for p in (content.get("parts") or []) if isinstance(p, str))
        if text.strip():
            turns.append({"role": role, "text": text})
```

Notes on the message stream:
- `linear_conversation` is the already-linearized path (use it instead of
  walking `mapping` + `current_node` yourself).
- `content.content_type` is often `text`, but also `thoughts`,
  `reasoning_recap`, `code`, `model_editable_context`, `tool` results, etc.
  For an importable artifact, keep `text` (and optionally `reasoning_recap`);
  drop the rest or store them separately.
- `safe_urls` is a list of *index references* to URL strings — useful for
  capturing cited sources alongside the text.
- ChatGPT inline-citation markers appear in assistant text as opaque tokens
  (e.g. `citeturn…`). Strip or resolve these during import; they are not
  human-readable.

## Caveats / fragility

- **Implementation-coupled.** This depends on OpenAI shipping turbo-stream in
  the share page and on the `routes/share.$shareId.($action)` loader path. A
  framework change or a move to an authenticated fetch will break the decoder.
  Detect breakage explicitly (zero messages decoded) and fall back.
- **Cloudflare.** Raw HTML fetch worked with a browser UA; the JSON API did
  not. Expect to need realistic headers, and possibly a headless browser
  (`browser-automation-studio`'s CaptureService with `inline_dom=true`) as the
  robust fallback that simply renders the page and reads the returned DOM,
  sidestepping the encoding entirely.
- **Per-provider.** Each provider encodes differently. ChatGPT = turbo-stream.
  Claude/Gemini/Grok share pages will need their own probes. Structure the
  feature as one interface with per-provider adapters.
- **ToS / rate limits / robots.** A share link is content the operator chose to
  save, and extraction is user-initiated — but respect rate limits and cache by
  URL. If a provider ever requires an authenticated session to read a share
  page, that adapter becomes **tier 2** under the risk-tier contract
  (`SIG-P0-014`) and inherits its safety envelope.

## When this is built for real

Suggested shape, satisfying `SIG-P1-004`:

- A conversation extractor inside the `enrichment` domain, with a
  provider-adapter interface: `detect(url) -> provider`,
  `fetch(url) -> rawHtml|dom`,
  `extract(raw) -> { title, model, createdAt, turns[], citedUrls[] }`.
- **ChatGPT adapter:** turbo-stream decode (this doc), with a
  `browser-automation-studio` capture (`inline_dom`) render fallback.
- **Generic fallback adapter:** BAS capture render plus DOM scrape for providers
  not yet reverse-engineered.
- Emit the extracted transcript as the signal's extracted content, so a saved
  conversation is classified, embedded, and retrievable like any other signal —
  not stored as opaque text.
- **Add a regression fixture**: save one real share-page HTML snapshot and
  assert the decoder still yields N turns, so a silent upstream format change
  fails a test instead of silently importing blanks. This fixture is the
  mechanical enforcement of D-019 for this extractor specifically.

## Cross-References

- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — D-019 (empty extraction is a failure) generalizes the lesson recorded here
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — enrichment delegation boundaries and risk tiers
