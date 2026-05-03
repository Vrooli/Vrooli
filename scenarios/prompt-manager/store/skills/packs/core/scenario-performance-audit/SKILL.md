## Practice focus: Scenario Performance Audit

Apply a reproducible, **headless** browser-perf audit to a Vrooli scenario UI: confirm the scenario has perf-build infrastructure (and add it from the canonical reference if missing), capture a Chrome performance trace while exercising a scripted interaction, and analyse it for component-level commit hotspots. Produces a trace JSON loadable in Chrome DevTools and a written findings report tied to file paths.

**Prerequisite:** the target scenario must be a React + Vite UI based on the `templates/scenarios/react-vite` template (the perf-build infra prescribed in Phase 1 assumes this stack). For non-react-vite scenarios the methodology is broadly valid but Phase 1 needs different infra and is out of scope here.

Required reading:
- `docs/agent-system/SKILL_AUTHORING.md`
- `prompt-manager skill read skill-authoring-practice`

Optional reading:
- `scenarios/swarm-manager/ui/vite.config.ts` — canonical perf-build implementation reference
- `templates/scenarios/react-vite/ui/` — what newly-generated scenarios get for free

---

### **1. When to Use This Methodology**

Use this skill when:
- A user reports laggy UI behaviour in a React-based Vrooli scenario
- Someone wants to validate a perf-related code change with before/after numbers
- A scenario hasn't been profiled before and you need a baseline

**Do NOT use** when:
- The slowness is on the API/Go side — that's a different audit (CPU profiling Go binaries, not browser tracing)
- The scenario doesn't have a React UI
- You already know the bottleneck and just need to fix it (skip straight to the fix; this skill is for *discovery*)
- A regular `console.log` would answer the question

---

### **2. The Process**

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                       SCENARIO PERFORMANCE AUDIT                             │
├──────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│   ┌────────┐  ┌────────┐  ┌────────┐  ┌────────┐  ┌────────┐  ┌────────┐    │
│   │CLARIFY │─▶│READINESS│─▶│ BUILD │─▶│ PLAN  │─▶│CAPTURE │─▶│ANALYSE │    │
│   │ (P0)   │  │ (P1)   │  │ (P2)   │  │ (P3)   │  │ (P4)   │  │+REPORT │    │
│   └────────┘  └────────┘  └────────┘  └────────┘  └────────┘  │ (P5)   │    │
│        │           │                                          └────────┘    │
│        ▼           ▼ if infra missing → fix from canonical          │       │
│                       references, then continue                     ▼       │
│                                                              ┌──────────────┐│
│                                                              │ RESTORE      ││
│                                                              │ DEFAULT (P6) ││
│                                                              └──────┬───────┘│
│                                                                     ▼        │
│                                                              ┌──────────────┐│
│                                                              │ PERSIST      ││
│                                                              │ AUDIT (P7)   ││
│                                                              └──────────────┘│
│                                                                              │
│  After a fix lands, **re-run from Phase 2** with the same Phase 3 script    │
│  for a side-by-side comparison. See *Comparison runs* below.                │
└──────────────────────────────────────────────────────────────────────────────┘
```

---

### **Phase 0: Clarify the complaint**

**Entry criteria:** Someone has asked for a perf audit on a scenario.

Most audits go sideways because the agent skipped this phase. Profiling is a discovery tool, not a guess generator. Before opening any code, get specific:

**Required clarifications (ask the user if any are unclear):**

1. **Which interaction is slow?** Not "the app feels slow" — *which click, which drag, which page transition*.
2. **Where is the user observing it?** Desktop browser? In-app via the embedded WebView? Through the proxy/tunnel? Production deployment vs local dev? Each can have different bottlenecks.
3. **Is this a regression?** If yes, since when / which change. If yes, the audit's job may be confirming a hypothesis, not full discovery.
4. **What's the expected behaviour?** "Snappy" is not measurable; "should respond within 150 ms of click" is.
5. **Is there a specific trigger condition?** Only with N items in the list? Only after some other action? Only on first load vs. warm?

If (1) and (2) are not answered, **stop and ask the requester** before continuing. This skill produces noise without a specific target — a targeted Phase 3 interaction script is the difference between an actionable trace and 40 MB of garbage. Don't guess; ask.

**Exit criteria:** A specific user-observable interaction with a specific environment is identified.

**Artifacts:** A one-paragraph framing recorded at the top of the eventual `findings.md`. Include verbatim user quotes if available.

---

### **Phase 1: Audit perf-build readiness**

**Entry criteria:** A scenario slug to audit (e.g. `swarm-manager`) and a written user-facing complaint or specific interaction to investigate.

The scenario must have four pieces of infra in place. Check each. Any missing → fix from the canonical references *before* proceeding to Phase 2.

| Piece | Where | What to look for | Canonical reference |
|---|---|---|---|
| Perf-mode in vite config | `<scenario>/ui/vite.config.ts` | `mode === "profile"` branch with `react-dom/client` → `react-dom/profiling` alias and `esbuild.keepNames: true` | `templates/scenarios/react-vite/ui/vite.config.ts` |
| Conditional build script | `<scenario>/ui/package.json` | `build` appends `--mode profile` when `VROOLI_BUILD_MODE=profile`; `build:profile` alias exists | `templates/scenarios/react-vite/ui/package.json` |
| `onProfilerRender` util | `<scenario>/ui/src/lib/profiler.ts` | Exports `onProfilerRender` callback that calls `performance.measure` with `⚛` prefix | `templates/scenarios/react-vite/ui/src/lib/profiler.ts` |
| Top-level Profiler boundary | `<scenario>/ui/src/main.tsx` (or App entry) | `<React.Profiler id="App" onRender={onProfilerRender}>` wrapping the app | `templates/scenarios/react-vite/ui/src/main.tsx` |

**Why it matters that the boundary is *permanent*:** `<React.Profiler>`'s `onRender` is only invoked when the perf-build's `react-dom/profiling` bundle is loaded. The fiber exists in regular prod but the callback never fires — so wrappers cost effectively nothing in default builds. They're load-bearing only when auditing. Remove a wrapper at your peril; you'll lose data and may not notice for months.

**If the scenario is structurally divergent from the template** (different bundler, no `vite.config.ts`, custom React entrypoint, different package manager), Phase 1 is *not* a small fixup. Stop and surface this to the requester. Either (a) the scenario should be brought up to template-conformance as a separate engineering task and then audited, or (b) this skill doesn't apply and a different methodology is needed. Don't paper over the divergence — the readings will be unreliable.

**Adding inner boundaries (recommended when auditing a specific subtree):** if the user complaint is localized (e.g. "the sidebar feels slow"), add `<Profiler id="Sidebar">` around that subtree before capturing. Pattern: extract the body to `XImpl`, export a wrapper that does `<Profiler id="X" onRender={onProfilerRender}><XImpl ... /></Profiler>`. See `swarm-manager/ui/src/surfaces/graph/components/sidebar/BacklogTab.tsx` for the pattern with `memo()`.

**Exit criteria:** All four infra pieces present; inner boundaries added around any subtree the audit will scrutinise.

**Artifacts:** Code edits committed (or staged) to bring the scenario up to perf-audit readiness. Note these in the eventual findings report — they're a deliverable of the audit too.

---

### **Phase 2: Build and serve the perf bundle**

**Entry criteria:** Phase 1 complete; the scenario is currently running.

```bash
# Pick a deterministic working dir keyed on the scenario slug.
SCENARIO=<slug>
WORKDIR="/tmp/${SCENARIO}/perf"
mkdir -p "${WORKDIR}"

# Restart with the perf-build env var. The build script's mode-aware vite
# invocation produces the perf bundle; the lifecycle's force_setup=true
# rebuilds, which is what we want here.
#
# Build time: first profile-mode build is 5–10 min (cold UI build is the
# worst case). Warm rebuilds with the Vite cache are 30–60s. If a "restart"
# returns in <60s, the cached profile bundle is being reused — that's
# expected, not a sign the rebuild was skipped.
VROOLI_BUILD_MODE=profile vrooli scenario restart "${SCENARIO}"

# Verify the served bundle is the perf one. Names like `BacklogRow`,
# `BacklogTabImpl`, `onProfilerRender` should appear; if they're mangled,
# the perf build did not take.
PORT=$(vrooli scenario port "${SCENARIO}" 2>/dev/null | grep -oE 'UI_PORT=[0-9]+' | cut -d= -f2)
MAIN=$(curl -s "http://localhost:${PORT}/" | grep -oE 'assets/index-[A-Za-z0-9_-]+\.js' | head -1)
curl -s "http://localhost:${PORT}/${MAIN}" | grep -oE "onProfilerRender|[A-Z][a-zA-Z]+Impl" | sort -u | head
```

> **Why curl/grep here?** No Vrooli CLI returns "the served bundle's contents" or "is the perf bundle loaded?" yet. Using `curl` for read-only black-box verification of a local UI port is the documented exception to the "wrap external tools" principle (see skill-validation §4.4). When the eventual `browser-automation-studio perf trace` CLI lands, this snippet collapses to a single subcommand call.

If `onProfilerRender` isn't in the served bundle, Phase 1 didn't complete — `lib/profiler.ts` is missing or unimported. Stop and fix.

**Exit criteria:** `onProfilerRender` is present in the served main bundle.

---

### **Phase 3: Plan the interaction**

The interaction script must be:
- **Reproducible** — same actions, same timing, every run. Before/after comparisons hinge on this.
- **Reflective** — covers what the user actually reports, not what you guess might be slow.
- **Scoped** — fits within a 5–15 s capture window. Longer captures bloat the trace and dilute the signal.

Use this decision table to pick interactions:

| User report | Recommended interactions |
|---|---|
| "Page X feels slow" | navigate → wait for shell → run primary user actions on page X (clicks, typing) |
| "Resizing/dragging is laggy" | navigate → mount-settle → drag the handle 3–5 cycles |
| "Switching tabs is slow" | navigate → click each tab in sequence with brief settle delay |
| "List/feed is slow" | navigate → trigger filter/sort/scroll on the list |
| Unspecified / "the app feels heavy" | navigate → 30 s of varied interaction across major surfaces |

Capture a brief mount-settle (~800 ms) after navigation so initial network/render churn doesn't dominate the trace.

---

### **Phase 4: Capture the trace**

Write a self-contained Playwright + CDP script at `${WORKDIR}/capture.js` and run it.

The script borrows Playwright from BAS's existing install (`scenarios/browser-automation-studio/playwright-driver/node_modules/rebrowser-playwright`) so no new deps are introduced. Long-term, this graduates into a `browser-automation-studio perf trace` CLI command — see *Future direction* at the bottom of this skill.

**Capture script template** (write to `${WORKDIR}/capture.js`, customise the `exerciseTarget` function for the chosen interaction):

```javascript
#!/usr/bin/env node
const path = require("node:path");
const fs = require("node:fs");
const BAS = "/home/matthalloran8/Vrooli/scenarios/browser-automation-studio/playwright-driver/node_modules";
const { chromium } = require(path.join(BAS, "rebrowser-playwright"));

const TRACING_CATEGORIES = [
  "devtools.timeline",
  "disabled-by-default-devtools.timeline",
  "disabled-by-default-devtools.timeline.frame",
  "disabled-by-default-devtools.timeline.stack",
  "disabled-by-default-v8.cpu_profiler",
  "disabled-by-default-devtools.screenshot",
  "blink.user_timing",
  "loading",
  "v8.execute",
];

const WEB_VITALS_INIT = `
(() => {
  if (window.__perfInstalled) return;
  window.__perfInstalled = true;
  const data = { longTasks: [], paint: [], lcp: null, marks: [] };
  window.__perfData = data;
  try { new PerformanceObserver((l) => { for (const e of l.getEntries()) data.longTasks.push({start:e.startTime,duration:e.duration,name:e.name}); }).observe({type:"longtask",buffered:true}); } catch {}
  try { new PerformanceObserver((l) => { for (const e of l.getEntries()) data.paint.push({name:e.name,start:e.startTime}); }).observe({type:"paint",buffered:true}); } catch {}
  try { new PerformanceObserver((l) => { const e = l.getEntries(); const last = e[e.length-1]; if (last) data.lcp = {start:last.startTime,size:last.size}; }).observe({type:"largest-contentful-paint",buffered:true}); } catch {}
})();
`;

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

// --- Reusable interaction helpers. Add to your audit's capture.js as needed. ---

// Drag a target horizontally by `dx` over N steps (settles transition queues
// better than a one-shot move, so resize-lag traces are reproducible).
async function dragHorizontalOnce(page, selector, dx, steps = 20) {
  const handle = await page.locator(selector).boundingBox();
  if (!handle) throw new Error(`dragHorizontalOnce: ${selector} not found`);
  const startX = handle.x + handle.width / 2;
  const startY = handle.y + handle.height / 2;
  await page.mouse.move(startX, startY);
  await page.mouse.down();
  for (let i = 1; i <= steps; i++) {
    await page.mouse.move(startX + (dx * i) / steps, startY, { steps: 1 });
  }
  await page.mouse.up();
}

// Walk up from a target element to find its nearest scrollable ancestor.
// Useful when scrolling a virtualized list whose scroll-parent isn't the
// element you started from (e.g. a panel-level <main>).
async function findScrollableAncestor(page, selector) {
  return await page.evaluate((sel) => {
    let el = document.querySelector(sel);
    while (el && el !== document.body) {
      const cs = window.getComputedStyle(el);
      if (cs.overflowY === "auto" || cs.overflowY === "scroll") {
        // Return a CSS-locatable handle (id or generated path) the test can grab.
        if (el.id) return `#${el.id}`;
        const parent = el.parentElement;
        if (parent) {
          const idx = Array.from(parent.children).indexOf(el);
          return `${parent.tagName.toLowerCase()} > :nth-child(${idx + 1})`;
        }
      }
      el = el.parentElement;
    }
    return null;
  }, selector);
}
// ----------------------------------------------------------------------

// --- Customise this for the audit. Each phase logs a label so the trace
//     is annotated and analysis can correlate timestamps to phases. The DOM
//     counts logged after structural changes are a hard-to-fake confirmation
//     that the change you're auditing actually shipped (e.g. virtualization
//     reducing 278 backlog rows to 9 visible). ---
async function exerciseTarget(page, log) {
  log("waiting for shell to mount");
  await page.waitForSelector("[data-testid='sidebar']", { timeout: 15000 }).catch(() => {});
  await sleep(800);
  log(`pre-phase DOM count [data-testid='backlog-item']: ${await page.locator("[data-testid='backlog-item']").count()}`);

  log("phase 1: <describe what the user complained about>");
  // ... interactions: page.click(), page.mouse.down/move/up(), page.keyboard.type(), etc.
  // Example: await dragHorizontalOnce(page, "[data-testid='sidebar-resize-handle']", 80);
  await sleep(500);
  log(`post-phase DOM count [data-testid='backlog-item']: ${await page.locator("[data-testid='backlog-item']").count()}`);
}
// ----------------------------------------------------------------------

async function streamCdpIo(client, handle) {
  const chunks = [];
  while (true) {
    const { data, eof, base64Encoded } = await client.send("IO.read", { handle, size: 1 << 20 });
    if (data) chunks.push(base64Encoded ? Buffer.from(data, "base64") : Buffer.from(data, "utf8"));
    if (eof) break;
  }
  await client.send("IO.close", { handle });
  return Buffer.concat(chunks);
}

async function main() {
  const url = process.argv[2] || (() => { throw new Error("usage: capture.js <url> [out]"); })();
  const out = process.argv[3] || path.join(__dirname, "trace.json");
  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  await context.addInitScript({ content: WEB_VITALS_INIT });
  const page = await context.newPage();
  page.on("pageerror", (e) => console.error("[page]", e.message));

  await page.goto(url, { waitUntil: "domcontentloaded", timeout: 30000 });
  const client = await context.newCDPSession(page);

  await client.send("Tracing.start", {
    transferMode: "ReturnAsStream",
    streamFormat: "json",
    streamCompression: "none",
    traceConfig: { includedCategories: TRACING_CATEGORIES, recordMode: "recordContinuously" },
  });
  const traceComplete = new Promise((res, rej) => {
    client.once("Tracing.tracingComplete", res);
    // 120s covers multi-phase scripts (resize + scroll + click sequences).
    // Bump higher only if your `exerciseTarget` deliberately runs >90s.
    setTimeout(() => rej(new Error("tracingComplete timeout")), 120000);
  });

  await exerciseTarget(page, (m) => console.log(`[capture] ${m}`));
  await client.send("Tracing.end");
  const ev = await traceComplete;
  fs.writeFileSync(out, await streamCdpIo(client, ev.stream));
  fs.writeFileSync(out.replace(/\.json$/, ".web-vitals.json"), JSON.stringify(await page.evaluate(() => window.__perfData), null, 2));
  console.log(`[capture] wrote ${out}`);
  await context.close();
  await browser.close();
}
main().catch((e) => { console.error(e); process.exit(1); });
```

**Run it:**

```bash
node "${WORKDIR}/capture.js" "http://localhost:${PORT}" "${WORKDIR}/trace.json"
```

**Sanity-check the trace** (must pass before analysis is meaningful):

```bash
node -e "
const t = require('${WORKDIR}/trace.json');
const evs = t.traceEvents;
const r = evs.filter(e => e.cat?.includes('blink.user_timing') && e.name?.startsWith('⚛'));
console.log('⚛ entries:', r.length);
console.log('CPU profile chunks:', evs.filter(e => e.name === 'ProfileChunk').length);
"
```

Expect dozens to hundreds of `⚛` entries (one pair per Profiler boundary per commit) and hundreds to thousands of `ProfileChunk` events. Zero `⚛` entries means the perf bundle isn't really in service — return to Phase 2.

**Exit criteria:** Trace JSON produced, sanity-check passes.

**Artifacts:** `${WORKDIR}/trace.json`, `${WORKDIR}/trace.web-vitals.json`, `${WORKDIR}/capture.js`.

---

### **Phase 5: Analyse and report**

Aggregate Profiler entries into a per-component summary. The user_timing entries are paired begin/end events keyed by `id2.local`:

```bash
node -e "
const t = require('${WORKDIR}/trace.json');
const r = t.traceEvents.filter(e => e.cat?.includes('blink.user_timing') && e.name?.startsWith('⚛'));
const begins = new Map(); const dur = [];
for (const e of r) {
  const k = e.id2?.local;
  if (e.ph === 'b') begins.set(k, e);
  else if (e.ph === 'e') { const b = begins.get(k); if (b) { dur.push({name: b.name, dur: e.ts - b.ts}); begins.delete(k); } }
}
const agg = {};
for (const d of dur) { const id = d.name.replace(/ \\([a-z-]+\\)\$/, ''); if (!agg[id]) agg[id]={count:0,total:0,max:0}; agg[id].count++; agg[id].total+=d.dur; if (d.dur > agg[id].max) agg[id].max = d.dur; }
console.log('component'.padEnd(22), 'count'.padStart(6), 'total(ms)'.padStart(11), 'avg(μs)'.padStart(9), 'max(μs)'.padStart(9));
for (const [id, s] of Object.entries(agg).sort((a,b) => b[1].total - a[1].total))
  console.log(id.slice(0,22).padEnd(22), String(s.count).padStart(6), (s.total/1000).toFixed(1).padStart(11), Math.round(s.total/s.count).toString().padStart(9), Math.round(s.max).toString().padStart(9));
"
```

**Reading the table:**

| Pattern | Likely meaning |
|---|---|
| Top-of-tree component (App/Shell) commits dozens of times in the window | Upstream state churn — store updates with new array refs, polling tick re-renders, unstable selectors. Investigate stores subscribed by the top of tree. |
| Specific subtree has high *avg* commit time but low count | The component is genuinely expensive when it renders. Look for un-memoized lists, heavy `useMemo` deps, expensive sync filtering/sorting. |
| Single commit with a max far above the avg | Outlier — usually a state-cascade or a specific user action. Find its timestamp in the trace, look at what fired then. |
| `nested-update` commits | A `setState` fired *during* a render. Look for hooks that update state in render (selector instability, derived state in render bodies). |
| Memoized leaf has commit count proportional to (parent commits × N rows) | **Memo is being defeated.** Look for unstable props chained from above: fresh closures (inline arrow functions per render), fresh array literals, `Map.get(rebuiltMap)` patterns where the map is created in the parent body, and TanStack `useMutation`'s `mutate` (only stable until reset). Lift derived data to context, hoist Maps into `useMemo`, and bake mutation pending state out of callbacks. |

**Watch `avg(μs)` more than `total(ms)`.** When a fix changes commit cardinality (e.g. virtualization makes a list mount many cheap rows instead of one giant render), `total(ms)` can rise even though each commit is now far cheaper. `avg(μs)` is the per-commit cost; that's the metric that tracks "did the fix actually make rendering this thing cheaper". Validate the win by confirming `avg(μs)` drops; the long-task delta in `*.web-vitals.json` corroborates from a different angle.

**Open the trace in Chrome DevTools** (Performance → "Load profile…") for visual analysis: the user_timing track shows the `⚛` markers aligned with frames; the CPU samples flame graph shows readable hook/utility names. Use the bottom-up view filtered to scripting-time to identify dominant cost sites.

**Findings report:** write to `${WORKDIR}/findings.md`. Each finding should have:
- File path + line numbers
- Quantitative evidence (commit count, avg, max — copy-pasted from the analysis table)
- Hypothesis for the cause (selector instability, missing memo, etc.)
- Suggested next step (which to actually do is *out of scope for this skill*; that's a separate engineering task)

**List new dependencies the suggested fixes would add.** If a recommendation needs a package the scenario doesn't currently have (e.g. `@tanstack/react-virtual` for virtualization), surface it under a "New dependencies" bullet in `findings.md`. The user authorizes deps before implementation begins, not in the middle of it — surfacing late wastes the implementer's loop.

**Exit criteria:** Findings written; trace JSON archived; user has a quantitative answer to "which subtree is slow."

---

### **Phase 6: Restore the default build**

The perf build is ~5–15 % slower than regular prod and ~10–20 KB heavier. Don't leave a scenario in perf mode after the audit.

```bash
unset VROOLI_BUILD_MODE
vrooli scenario restart "${SCENARIO}"
# Verify: served bundle should NOT contain `onProfilerRender` or *Impl names.
```

Optionally clean up `${WORKDIR}` if disk is tight. The trace is large (~40 MB for a 12 s capture). Keep it if a follow-up engineering task is going to act on the findings; archive it to a durable path if the audit results need to be referenced later.

**Exit criteria:** Scenario back on the regular bundle. Audit complete.

---

### **Phase 7: Persist the audit**

If the audit reached actionable findings, persist them as a durable artifact in the scenario's docs tree so future sessions can find them. Free-form `findings.md` files in `/tmp/` disappear; `docs/perf/` files in the scenario tree survive across sessions, can be cross-linked, and are validated by `knowledge-observatory`.

```bash
SLUG="<short-kebab-slug>"   # e.g. sidebar-resize-and-backlog-scroll
DEST="scenarios/${SCENARIO}/docs/perf/$(date -I)-${SLUG}.md"
mkdir -p "$(dirname "${DEST}")"
knowledge-observatory docs template perf-audit > "${DEST}"
# Edit the file: fill in frontmatter (date, scenario, interactions, status,
# trace paths) and replace the placeholder sections with the audit content
# (Framing / Methodology / Per-component table / Long-task summary /
# Findings / Recommendations + outcome).
```

Validate the shape:

```bash
knowledge-observatory docs audit "${SCENARIO}"
# Look for any "perf-audit:" issues in the output. Zero issues = the
# frontmatter and per-component table conform.
```

Register the doc in the scenario's `docs/manifest.json` under a `perf` section so it surfaces in the in-app docs viewer.

**Exit criteria:** Audit findings live at `scenarios/${SCENARIO}/docs/perf/<date>-<slug>.md`, registered in the manifest, and pass `knowledge-observatory docs audit`.

---

### **Comparison runs (validating a fix)**

This skill is for *discovery*. To validate that a fix actually helped, re-run it after the fix lands. The discipline is the same — **don't let the comparison interaction drift from the baseline**. If the script changed between runs, the diff is meaningless.

```bash
# After the discovery audit, archive the trace under a name you'll recognise.
mv "${WORKDIR}/trace.json" "${WORKDIR}/trace.before.json"
mv "${WORKDIR}/trace.web-vitals.json" "${WORKDIR}/trace.before.web-vitals.json"

# … developer applies the fix in source, commits …

# Re-run from Phase 2 (perf-build restart) onward with the *same* capture.js.
# Do NOT modify exerciseTarget between runs.
VROOLI_BUILD_MODE=profile vrooli scenario restart "${SCENARIO}"
node "${WORKDIR}/capture.js" "http://localhost:${PORT}" "${WORKDIR}/trace.after.json"
```

**Diff the per-component table.** Run the Phase 5 aggregation against both files and produce a side-by-side diff:

```bash
node -e "
function agg(path) {
  const t = require(path);
  const r = t.traceEvents.filter(e => e.cat?.includes('blink.user_timing') && e.name?.startsWith('⚛'));
  const begins = new Map(); const dur = [];
  for (const e of r) {
    const k = e.id2?.local;
    if (e.ph === 'b') begins.set(k, e);
    else if (e.ph === 'e') { const b = begins.get(k); if (b) { dur.push({name: b.name, dur: e.ts - b.ts}); begins.delete(k); } }
  }
  const out = {};
  for (const d of dur) { const id = d.name.replace(/ \\([a-z-]+\\)\$/, ''); if (!out[id]) out[id]={count:0,total:0,max:0}; out[id].count++; out[id].total+=d.dur; if (d.dur > out[id].max) out[id].max = d.dur; }
  return out;
}
const before = agg('${WORKDIR}/trace.before.json');
const after = agg('${WORKDIR}/trace.after.json');
const all = new Set([...Object.keys(before), ...Object.keys(after)]);
const colHeader = (s, w) => s.padStart(w);
console.log(
  'component'.padEnd(22),
  colHeader('b-cnt', 7), colHeader('a-cnt', 7),
  colHeader('before(ms)', 11), colHeader('after(ms)', 11),
  colHeader('b-avg(μs)', 10), colHeader('a-avg(μs)', 10),
  colHeader('delta(ms)', 11), colHeader('delta(%)', 9),
);
for (const id of [...all].sort()) {
  const b = before[id] || {count:0,total:0,max:0};
  const a = after[id] || {count:0,total:0,max:0};
  const dms = (a.total - b.total) / 1000;
  const dpct = b.total > 0 ? ((a.total - b.total) / b.total) * 100 : (a.total > 0 ? Infinity : 0);
  const bAvg = b.count > 0 ? Math.round(b.total / b.count) : 0;
  const aAvg = a.count > 0 ? Math.round(a.total / a.count) : 0;
  console.log(
    id.slice(0,22).padEnd(22),
    String(b.count).padStart(7), String(a.count).padStart(7),
    (b.total/1000).toFixed(1).padStart(11), (a.total/1000).toFixed(1).padStart(11),
    String(bAvg).padStart(10), String(aAvg).padStart(10),
    dms.toFixed(1).padStart(11),
    (isFinite(dpct) ? dpct.toFixed(0)+'%' : 'new').padStart(9),
  );
}
"
```

The added `b-cnt / a-cnt / b-avg / a-avg` columns surface the case the original `before/after/delta-%` layout could hide: virtualization can take a list from 14 expensive commits to 60 cheap ones (total rises slightly, avg-per-commit drops 30×). Always check `a-avg < b-avg` for the targeted component before declaring the fix worked.

**Long-task delta (first-class, not a footnote):** the long-task signal is the cleanest correlate of *felt* performance. Run this directly on the `*.web-vitals.json` files captured by `WEB_VITALS_INIT`:

```bash
node -e "
const b = require('${WORKDIR}/trace.before.web-vitals.json').longTasks || [];
const a = require('${WORKDIR}/trace.after.web-vitals.json').longTasks || [];
const sum = (xs) => xs.reduce((s,e) => s + e.duration, 0);
const max = (xs) => xs.reduce((m,e) => Math.max(m, e.duration), 0);
console.log('Long tasks  | count |  total(ms) |   max(ms)');
console.log('before      | %s | %s | %s', String(b.length).padStart(5), sum(b).toFixed(0).padStart(10), max(b).toFixed(0).padStart(9));
console.log('after       | %s | %s | %s', String(a.length).padStart(5), sum(a).toFixed(0).padStart(10), max(a).toFixed(0).padStart(9));
console.log('delta       | %s | %s | %s', String(a.length - b.length).padStart(5), (sum(a) - sum(b)).toFixed(0).padStart(10), (max(a) - max(b)).toFixed(0).padStart(9));
"
```

A 1300ms → 175ms long-task total drop with a flat per-component table means the fix landed at a layer the Profiler doesn't see (event handlers, layout thrash, browser-side work). Trust the long-task delta.

**Run the scenario test suite before declaring the comparison successful:**

```bash
vrooli scenario test "${SCENARIO}"
```

A perf "win" that breaks tests isn't a win. Tests must be green on the post-fix tree before you record the comparison as conclusive.

Read the table. Validate:

| Pattern | Diagnostic |
|---|---|
| Targeted component's total drops as expected | Fix worked. Magnitude should match expectation; if it's much smaller, the fix may have only addressed part of the problem |
| Targeted component drops, but *another* component rises by a comparable amount | The fix moved cost rather than eliminating it (e.g. memo'd work pushed to parent). Check the rising component |
| All components drop uniformly by 5–15 % | This is run-to-run noise, not a signal. Re-run both sides to confirm direction; if you can't reproduce the difference, the fix is in the noise floor |
| Targeted component unchanged | The fix didn't take, or it addresses code outside the captured interaction. Check that the perf bundle was rebuilt and that the interaction actually exercises the changed code |
| Long-task count from `*.web-vitals.json` drops without component-level changes | The fix landed at a layer Profiler doesn't see (raw event handlers, layout-thrash, browser-side work). Trust the long-task signal |

**Capture both traces with the same OS and load conditions.** A lunchtime laptop on battery vs an idle morning machine differ by more than most fixes do.

Comparison reports add a top-line summary alongside the per-component delta table:

```
Discovery run:    /tmp/<slug>/perf/trace.before.json    (<long-task ms>, <commit count>)
Validation run:   /tmp/<slug>/perf/trace.after.json     (<long-task ms>, <commit count>)
Net change:       <Δ long-task ms>, <Δ commits>
```

---

### **3. Knowledge Capture**

Every audit produces:

1. **`scenarios/<slug>/docs/perf/<date>-<slug>.md`** (Phase 7 output) — the durable, in-repo record of the audit. Validated by `knowledge-observatory docs audit`. This is the artifact that survives across sessions and is the canonical reference; the `${WORKDIR}/findings.md` is a draft on the way to this.
2. **`${WORKDIR}/trace.json` and `trace.web-vitals.json`** — raw evidence. Re-loadable in DevTools, re-analysable, comparable across audits if archived. Referenced by absolute path from the persisted doc; may be GC'd from `/tmp` and that's OK — the persisted doc remains.
3. **Permanent code edits** from Phase 1 if the scenario lacked perf-build infra. Those edits are scenario assets — they make the next audit faster.

The persisted `docs/perf/<date>-<slug>.md` doc references traces by absolute path so a reader can re-open the evidence while it lasts.

---

### **4. Anti-Patterns**

- **Don't audit a dev build.** Vite dev mode disables batching, runs StrictMode double-renders, and performs un-tree-shaken module-graph work. The perf characteristics do not match deployed behaviour and the data will mislead.
- **Don't skip Phase 1.** A capture against a scenario without `onProfilerRender` produces a trace with zero `⚛` entries. You'll waste time analysing CPU samples whose component names are mangled.
- **Don't mix the capture window with infra fixes.** Phase 1 changes ship before Phase 4 runs. Capturing during a build switch produces unreproducible traces.
- **Don't conclude from a single capture for marginal differences.** Captures vary by 5–15 % run-to-run from background-task noise. If two component totals are within that range, the audit doesn't distinguish them — say so.
- **Don't fix during the audit.** This skill produces *findings* about hotspots. Implementing the fixes for those hotspots is a separate engineering loop — mixing the two obscures whether the fix actually changed the numbers. (Phase 1 infra additions don't count: they're a precondition for the audit, not a fix to anything the audit identified.)
- **Don't keep the perf build running.** Phase 6 is mandatory.

---

### **5. Out of Scope**

- API/server-side performance — that's a Go-pprof or equivalent methodology, not browser tracing.
- Memory leaks. The Tracing categories used here don't capture allocation lifetimes well; use a heap snapshot session (`HeapProfiler.startSampling`) for that, ideally as a separate skill.
- Graph/render-engine internals (e.g. WebGL, canvas paint cost). Those need GPU profiling categories not enabled here.
- Network performance audits. Use HAR + waterfall analysis for that, not user-timing aggregation.

---

### **6. Output Expectations**

A successful application of this skill produces:

- A trace JSON loadable in Chrome DevTools Performance panel
- A per-component aggregation table with count / total / avg / max
- A `scenarios/<slug>/docs/perf/<date>-<slug>.md` doc (Phase 7) with file:line-anchored hypotheses and quantitative evidence, registered in the scenario's `docs/manifest.json` and passing `knowledge-observatory docs audit`
- A long-task delta line (count, total ms, max ms) corroborating or replacing the per-component signal
- (If Phase 1 added infra) committed code changes bringing the scenario up to perf-audit readiness
- (If a comparison run was performed) a delta table with `b-cnt / a-cnt / b-avg / a-avg / delta` columns and a net-change summary tying the change to the fix that produced it

What this skill does *not* produce:
- Code fixes for the identified hotspots — those are a separate task
- Promises about how much faster things will be after fixes
- Bundle-size optimisation recommendations

---

### **Future direction**

The browser-launch + CDP-tracing flow encoded in Phase 4's script is a candidate for graduation into the Browser Automation Studio scenario as `browser-automation-studio perf trace`. When that lands, this skill's Phase 4 will collapse to a single CLI invocation. The capture-script template here exists because that CLI does not yet exist; treat it as a placeholder for the eventual tool surface, and look for the meta-optimisation team to consolidate it.

---

### **Troubleshooting & Edge Cases**

| Symptom | Likely cause | Fix |
|---|---|---|
| Zero `⚛` entries in trace | Perf bundle not actually served (check `onProfilerRender` is in the served main bundle); Profiler boundaries not imported in source | Re-run Phase 2 verification; if `onProfilerRender` absent, return to Phase 1 |
| `ProfileChunk` events present, names still mangled | `keepNames: true` not in vite.config's profile mode block | Phase 1 audit missed it; add it |
| Restart leaves the perf bundle in place when env var is unset | The `build` script lacks the conditional substitution | Phase 1: check `package.json`; the `build` script must use `$([ "$VROOLI_BUILD_MODE" = profile ] && echo --mode profile)` |
| Capture script can't find `rebrowser-playwright` | BAS not installed or path wrong | Run `cd scenarios/browser-automation-studio/playwright-driver && pnpm install` once |
| `Tracing.tracingComplete` timeout | Interaction took too long, or the page crashed mid-capture | Check `page.on("pageerror")` log; trim the interaction or raise the 120 s timeout |
| Trace JSON is hundreds of MB | The interaction window is too long | Cap interactions at ~15 s; long captures aren't more useful, just heavier |