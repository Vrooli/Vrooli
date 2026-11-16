// Tests for the Notification Hub chart kit and its four templates.
//   node --test scenarios/notification-hub/skills/notification-hub/kit/tools/kit.test.mjs
// The browser tests run each template with its sample data in headless Chrome
// (no Hub bridge, so read-only) and skip when Chrome is not installed.
import { test } from "node:test";
import assert from "node:assert/strict";
import { execFileSync, spawnSync } from "node:child_process";
import { mkdtempSync, readFileSync, writeFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import {
  KIT_DIR, MAX_PAGE_BYTES, embedJSON, kitSources, loadSample, loadSchema, loadTemplate, render, syncTemplate, templateNames, validate,
} from "./kit.mjs";

const TEMPLATES = ["daily-briefing", "opportunity-roadmap", "research-report", "structural-change", "usage-mode"];
const clone = (v) => JSON.parse(JSON.stringify(v));

test("all five templates exist, each with a schema and a sample", () => {
  assert.deepEqual(templateNames(), TEMPLATES);
  for (const name of TEMPLATES) { loadSchema(name); loadSample(name); }
});

test("every template carries the current kit (run tools/sync.mjs after editing hub-charts.*)", () => {
  const kit = kitSources();
  for (const name of TEMPLATES) {
    const html = loadTemplate(name);
    assert.equal(syncTemplate(html, kit), html, `templates/${name}.html is stale`);
  }
});

test("every sample validates against its schema", () => {
  for (const name of TEMPLATES) assert.deepEqual(validate(loadSchema(name), loadSample(name)), [], name);
});

test("schemas reject data the templates cannot render honestly", () => {
  const usage = clone(loadSample("usage-mode"));
  delete usage.meta;
  assert.ok(validate(loadSchema("usage-mode"), usage).some((e) => e.includes("meta")));

  const daily = clone(loadSample("daily-briefing"));
  daily.pace.tiers[0].slots = 0;
  daily.invented = true;
  const dailyErrors = validate(loadSchema("daily-briefing"), daily);
  assert.ok(dailyErrors.some((e) => e.includes("slots")), dailyErrors.join("\n"));
  assert.ok(dailyErrors.some((e) => e.includes("invented")), dailyErrors.join("\n"));

  const structural = clone(loadSample("structural-change"));
  structural.checks.items[0].guardId = "has spaces";
  structural.checks.items[1].hardStop = "yes";
  const structuralErrors = validate(loadSchema("structural-change"), structural);
  assert.ok(structuralErrors.some((e) => e.includes("guardId")), structuralErrors.join("\n"));
  assert.ok(structuralErrors.some((e) => e.includes("hardStop")), structuralErrors.join("\n"));

  const roadmap = clone(loadSample("opportunity-roadmap"));
  roadmap.lines.find((l) => l.gate).gate.from = "next week";
  assert.ok(validate(loadSchema("opportunity-roadmap"), roadmap).some((e) => e.includes("gate")));

  // A scorecard level is the supervisor's judgment: it must come with its reason.
  const research = clone(loadSample("research-report"));
  delete research.scorecard.dimensions[0].reason;
  research.measurements.items[0].verdict = "great";
  const researchErrors = validate(loadSchema("research-report"), research);
  assert.ok(researchErrors.some((e) => e.includes("reason")), researchErrors.join("\n"));
  assert.ok(researchErrors.some((e) => e.includes("verdict")), researchErrors.join("\n"));

  const incompletePair = clone(loadSample("research-report"));
  delete incompletePair.scorecard.dimensions[0].now;
  assert.ok(validate(loadSchema("research-report"), incompletePair).some((e) => e.includes("now")));
  const qualitativeTarget = clone(loadSample("research-report"));
  const qualitative = qualitativeTarget.scorecard.dimensions.find((d) => d.key === "runtime");
  delete qualitative.was; delete qualitative.now;
  qualitative.target = 100;
  assert.ok(validate(loadSchema("research-report"), qualitativeTarget).some((e) => e.includes("was")));
});

test("research report sample preserves unknown numeric progress and current weight", () => {
  const d = loadSample("research-report");
  const qualitative = d.scorecard.dimensions.find((x) => x.key === "runtime");
  assert.ok(qualitative && qualitative.level && qualitative.reason);
  assert.equal(qualitative.was, undefined);
  assert.equal(qualitative.now, undefined);
  const unknown = d.workstreams.items.find((x) => x.id === "LM16");
  assert.ok(unknown);
  assert.equal(unknown.weight, undefined);
});

test("the daily pace can be sent without a timeline and with an unknown slot count", () => {
  const daily = clone(loadSample("daily-briefing"));
  delete daily.pace.window;
  const last = daily.pace.tiers[daily.pace.tiers.length - 1];
  last.slots = null;
  delete last.points;
  delete daily.schedules[last.key];
  assert.deepEqual(validate(loadSchema("daily-briefing"), daily), []);
});

test("every field in every schema is documented, and shared definitions agree", () => {
  const missing = [];
  function walk(node, path) {
    if (!node || typeof node !== "object") return;
    if (node.properties) for (const [k, v] of Object.entries(node.properties)) {
      if (!v.description && !v.$ref) missing.push(`${path}.${k}`);
      walk(v, `${path}.${k}`);
    }
    if (node.items) walk(node.items, `${path}[]`);
    if (node.additionalProperties && typeof node.additionalProperties === "object") walk(node.additionalProperties, `${path}{}`);
    if (node.$defs) for (const [k, v] of Object.entries(node.$defs)) { if (!v.description) missing.push(`${path}#${k}`); walk(v, `${path}#${k}`); }
    for (const alt of node.anyOf || []) walk(alt, path);
    for (const alt of node.oneOf || []) walk(alt, path);
  }
  const shared = {};
  for (const name of TEMPLATES) {
    const schema = loadSchema(name);
    assert.ok(schema.description, `${name}: schema description`);
    walk(schema, name);
    for (const def of ["source", "technical", "tone", "pill"]) {
      if (!schema.$defs || !schema.$defs[def]) continue;
      const text = JSON.stringify(schema.$defs[def]);
      if (shared[def]) assert.equal(text, shared[def].text, `$defs.${def} differs between ${shared[def].name} and ${name}`);
      else shared[def] = { name, text };
    }
  }
  assert.deepEqual(missing, []);
});

test("render escapes data so it can never close the script element", () => {
  const data = { meta: { title: "</script><script>alert(1)</script> &  " } };
  const json = embedJSON(data);
  assert.ok(!json.includes("<") && !json.includes(">") && !json.includes("&"));
  assert.ok(!json.includes(" "));
  assert.deepEqual(JSON.parse(json), data);
  const page = render(loadTemplate("usage-mode"), data);
  assert.equal(page.match(/<\/script>/g).length, loadTemplate("usage-mode").match(/<\/script>/g).length);
});

test("render refuses a template without exactly one hub-data element, and pages over 2 MiB", () => {
  assert.throws(() => render("<html><body></body></html>", {}), /exactly one hub-data/);
  const twice = '<script type="application/json" id="hub-data">{}</script><script type="application/json" id="hub-data">{}</script>';
  assert.throws(() => render(twice, {}), /exactly one hub-data/);
  assert.throws(() => render(loadTemplate("usage-mode"), { pad: "x".repeat(MAX_PAGE_BYTES) }), /2 MiB/);
});

test("templates follow the Hub page contract", () => {
  for (const name of TEMPLATES) {
    const html = loadTemplate(name);
    const page = render(html, loadSample(name));
    assert.ok(Buffer.byteLength(page) < MAX_PAGE_BYTES, `${name}: under 2 MiB`);
    assert.match(html, /<meta name="viewport"/, `${name}: viewport meta`);
    assert.equal((html.match(/<script type="application\/json" id="hub-data">/g) || []).length, 1, `${name}: one hub-data`);
    assert.doesNotMatch(html, /<(form|input|select|textarea)\b/i, `${name}: no form controls`);
    assert.doesNotMatch(html, /\b(src|href)\s*=\s*["']?(https?:)?\/\//i, `${name}: nothing loaded from the network`);
    assert.doesNotMatch(html, /@import|url\(\s*["']?(https?:)?\/\//i, `${name}: no remote CSS`);
    assert.doesNotMatch(html, /\bfetch\(|XMLHttpRequest|WebSocket|EventSource/, `${name}: no network calls`);
    assert.match(html, /role="img"/, `${name}: static chart skeleton for the packet check`);
  }
});

test("each daily tier with known slots has a contingency schedule with the same slot count", () => {
  const d = loadSample("daily-briefing");
  assert.ok(d.pace.tiers.length >= 3);
  const counts = new Set();
  for (const t of d.pace.tiers) {
    if (t.slots === null) continue;
    assert.ok(d.schedules[t.key], `schedule for ${t.key}`);
    assert.equal(d.schedules[t.key].slots, t.slots, t.key);
    counts.add(t.slots);
    for (const b of d.schedules[t.key].blocks) assert.ok(b.slot >= 1 && b.slot <= t.slots, `${t.key}: block slot ${b.slot}`);
  }
  assert.ok(counts.size >= 3, "faster tiers add slots");
});

/* ---------- browser behaviour ---------- */

const CHROME = ["google-chrome", "chromium", "chromium-browser"].find((c) => spawnSync(c, ["--version"]).status === 0);

// Runs the page in headless Chrome. `act` runs after load; it writes its findings with report(obj).
function runPage(name, { data, width = 400, act = "", hub = null } = {}) {
  const probe = `<script>
    window.__errors = [];
    window.addEventListener("error", function (e) { window.__errors.push(String(e.message)); });
    window.__log = [];
    ${hub ? `window.hub = { context: ${JSON.stringify(hub.context || {})}, prefill: function (q, v) { window.__log.push(["prefill", q, v]); },
      answer: { set: function (k, l, v) { window.__log.push(["set", k, v]); }, remove: function (k) { window.__log.push(["remove", k]); } } };` : ""}
  </script>`;
  const finish = `<script>
    window.addEventListener("load", function () { setTimeout(function () {
      var out = {};
      function report(o) { for (var k in o) out[k] = o[k]; }
      try { (function (report) { ${act} })(report); } catch (e) { window.__errors.push("act: " + e.message); }
      setTimeout(function () {
        out.log = window.__log;
        out.errors = window.__errors.filter(function (m) { return m.indexOf("ResizeObserver") < 0; });
        var pre = document.createElement("pre"); pre.id = "__result"; pre.textContent = JSON.stringify(out); document.body.appendChild(pre);
      }, 200);
    }, 400); });
  </script>`;
  const page = render(loadTemplate(name), data || loadSample(name)).replace("<head>", "<head>" + probe).replace("</body>", finish + "</body>");
  const dir = mkdtempSync(join(tmpdir(), "hub-kit-"));
  try {
    const file = join(dir, "page.html");
    writeFileSync(file, page);
    const dom = execFileSync(CHROME, ["--headless=new", "--disable-gpu", "--no-sandbox", "--no-first-run", `--window-size=${width},2000`, "--virtual-time-budget=4000", "--dump-dom", "file://" + file], { encoding: "utf8", timeout: 90000, stdio: ["ignore", "pipe", "ignore"] });
    const m = dom.match(/<pre id="__result">([\s\S]*?)<\/pre>/);
    assert.ok(m, `${name}: page did not finish`);
    return JSON.parse(m[1].replace(/&quot;/g, '"').replace(/&lt;/g, "<").replace(/&gt;/g, ">").replace(/&amp;/g, "&"));
  } finally { rmSync(dir, { recursive: true, force: true }); }
}

const browser = { skip: CHROME ? false : "no headless Chrome on this host" };

test("usage-mode draws the forecast and the answer cards, read-only without the Hub", browser, () => {
  const d = loadSample("usage-mode");
  const r = runPage("usage-mode", { act: `
    report({ lines: document.querySelectorAll("#burn svg polyline").length, bands: document.querySelectorAll("#burn svg polygon").length, cards: document.querySelectorAll("#answers .kc-answer").length });
    document.querySelector("#answers .kc-btn").click();
    report({ status: document.getElementById("answer-status").textContent, note: document.getElementById("answer-note").textContent });` });
  assert.deepEqual(r.errors, []);
  assert.ok(r.lines >= 2, "readings and projection lines");
  assert.ok(r.bands >= 1, "projection range band");
  assert.equal(r.cards, d.options.length);
  assert.match(r.status, /nothing is sent/i);
  assert.match(r.note, /nothing is sent/i);
});

test("usage-mode shows the Hub's own recurrence history, with answers named by their options", browser, () => {
  const d = loadSample("usage-mode");
  const ctx = { kind: "decision", recurrence: { count: 6, dates: ["2026-11-02T18:00:00Z", "2026-11-05T18:00:00Z", "2026-11-08T18:00:00Z", "2026-11-12T18:00:00Z", "2026-11-15T18:00:00Z", "2026-11-16T18:00:00Z"],
    previousAnswers: [{ answer: d.options[1].key, note: "", answeredAt: "2026-11-02T20:00:00Z" }, { answer: d.options[0].key, note: "", answeredAt: "2026-11-15T19:00:00Z" }] } };
  const r = runPage("usage-mode", { hub: { context: ctx }, act: `
    report({ recur: document.getElementById("recur").textContent, src: document.getElementById("sec-why").textContent });
    var card = [].filter.call(document.querySelectorAll("#answers .kc-answer"), function (c) { return c.textContent.indexOf(${JSON.stringify(d.options[3].title)}) >= 0; })[0];
    card.querySelector(".kc-btn").click();` });
  assert.deepEqual(r.errors, []);
  assert.match(r.recur, /6×/);
  assert.match(r.recur, /sixth time/);
  const name = (o) => o.short || o.title;
  assert.ok(r.recur.includes(name(d.options[1])) && r.recur.includes(name(d.options[0])));
  assert.match(r.src, /Notification Hub history/);
  assert.deepEqual(r.log.filter((e) => e[1] === "answer").map((e) => e[0]), ["prefill", "set"]);
  assert.equal(r.log.find((e) => e[0] === "set")[2], d.options[3].key);
});

test("daily briefing: a faster pace switches to that tier's contingency schedule", browser, () => {
  const d = loadSample("daily-briefing");
  const fast = d.pace.tiers.reduce((a, t) => (t.slots > a.slots ? t : a));
  const r = runPage("daily-briefing", { act: `
    report({ before: document.getElementById("h-sched").textContent });
    document.querySelector('#pace-seg button[data-v="${fast.key}"]').click();
    report({ after: document.getElementById("h-sched").textContent, note: document.getElementById("sched-note").textContent,
      moved: document.getElementById("moved").textContent });` });
  assert.deepEqual(r.errors, []);
  const base = d.schedules[d.pace.current].slots;
  assert.match(r.before, new RegExp(`What the ${base} agent slots`));
  assert.match(r.after, new RegExp(`What the ${fast.slots} agent slots`));
  assert.match(r.note, /Contingency schedule/);
  const moved = d.schedules[fast.key].movedLater || [];
  if (moved.length) assert.ok(r.moved.includes(moved[0].label));
});

test("daily briefing marks a missing pace timeline and unknown slots instead of drawing them", browser, () => {
  const d = clone(loadSample("daily-briefing"));
  delete d.pace.window;
  const last = d.pace.tiers[d.pace.tiers.length - 1];
  last.slots = null; delete last.points; delete d.schedules[last.key];
  const r = runPage("daily-briefing", { data: d, act: `
    document.querySelector('#pace-seg button[data-v="${last.key}"]').click();
    report({ chart: document.getElementById("pace-chart").textContent, slots: document.getElementById("pace-slots").textContent, note: document.getElementById("sched-note").textContent });` });
  assert.deepEqual(r.errors, []);
  assert.match(r.chart, /unavailable/);
  assert.match(r.slots, /Unknown/);
  assert.match(r.note, /No contingency schedule/);
});

test("daily briefing labels friction duration as unknown when no measure is supplied", browser, () => {
  const data = clone(loadSample("daily-briefing"));
  delete data.friction.rows.find((row) => row.runs > 0).hours;
  const r = runPage("daily-briefing", { data, width: 390, act: `
    report({ friction: document.querySelector("#fric .kc-fr__meta").textContent });` });
  assert.deepEqual(r.errors, []);
  assert.match(r.friction, /duration unknown/i);
});

test("daily and roadmap source badges wrap complete long labels at phone widths", browser, () => {
  const label = "portfolio steward evidence collected from the operator steering workstream and verified against the current notification hub payload";
  for (const [name, section] of [["daily-briefing", "target"], ["opportunity-roadmap", "hours"]]) {
    const data = clone(loadSample(name));
    data[section].source.label = label;
    for (const width of [320, 390]) {
      const r = runPage(name, { data, width: 500, act: `
        document.documentElement.style.width = "${width}px";
        document.body.style.width = "${width}px";
        document.body.style.margin = "0";
        var badge = document.querySelector(".kc-src");
        var rect = badge.getBoundingClientRect();
        report({ text: badge.textContent, width: badge.clientWidth, scrollWidth: badge.scrollWidth,
          right: rect.right, height: rect.height, viewport: document.documentElement.getBoundingClientRect().width,
          pageWidth: document.body.scrollWidth, whiteSpace: getComputedStyle(badge).whiteSpace });` });
      assert.deepEqual(r.errors, [], `${name} ${width}px page has no script errors`);
      assert.ok(r.text.includes(label), `${name} ${width}px preserves the full source label`);
      assert.ok(r.height > 20, `${name} ${width}px wraps the long label onto multiple lines`);
      assert.ok(r.scrollWidth <= r.width, `${name} ${width}px badge has no clipped text`);
      assert.ok(r.right <= r.viewport, `${name} ${width}px badge stays in the viewport: ${JSON.stringify(r)}`);
      assert.ok(r.pageWidth <= r.viewport, `${name} ${width}px page does not overflow horizontally`);
      assert.equal(r.whiteSpace, "normal");
    }
  }
});

test("structural change: hard stops and unmapped checks can never be switched", browser, () => {
  const d = loadSample("structural-change");
  const items = d.checks.items, selectable = items.filter((c) => c.guardId && !c.hardStop);
  const r = runPage("structural-change", { act: `
    report({ cards: document.querySelectorAll(".kc-ck").length, toggles: document.querySelectorAll(".kc-toggle").length, locks: document.querySelectorAll(".kc-lock").length,
      mapButtons: document.querySelectorAll('#map [role="button"]').length });` });
  assert.deepEqual(r.errors, []);
  assert.equal(r.cards, items.length);
  assert.equal(r.toggles, selectable.length);
  assert.equal(r.locks, items.length - selectable.length);
  assert.equal(r.mapButtons, selectable.length);
});

test("structural change does not draw an empty daily chart when history is unavailable", browser, () => {
  const data = clone(loadSample("structural-change"));
  data.blocked.days = [];
  data.blocked.source = { status: "partial", label: "Agent Manager runtime-friction report",
    note: "The feed provides aggregate run and response counts, not a per-day series." };
  data.blocked.responses = [{ label: "Stopped", count: 2, tone: "crit" },
    { label: "Tried a workaround", count: 0, tone: "warn" }];
  const r = runPage("structural-change", { data, width: 390, act: `
    report({ dailyText: document.querySelector("#daily").textContent,
      dailySvgs: document.querySelectorAll("#daily svg").length,
      dailyRole: document.querySelector("#daily").getAttribute("role"),
      responseText: document.querySelector("#did").textContent,
      responseLegend: document.querySelector("#did-legend").textContent });` });
  assert.deepEqual(r.errors, []);
  assert.equal(r.dailySvgs, 0);
  assert.equal(r.dailyRole, null);
  assert.match(r.dailyText, /Daily blocked-run history: unavailable/i);
  assert.equal(r.responseText, "2");
  assert.match(r.responseLegend, /Stopped/, "real aggregate response counts remain visible");
});

test("opportunity roadmap draws a row per line on a wide screen and a card per line on a phone", browser, () => {
  const d = loadSample("opportunity-roadmap");
  const wide = runPage("opportunity-roadmap", { width: 1100, act: `report({ rows: document.querySelectorAll("#view .kc-row").length, detail: document.getElementById("detail").hidden });` });
  assert.deepEqual(wide.errors, []);
  assert.equal(wide.rows, d.lines.length);
  assert.equal(wide.detail, false);
  const phone = runPage("opportunity-roadmap", { width: 400, act: `report({ cards: document.querySelectorAll("#view .kc-card").length });` });
  assert.deepEqual(phone.errors, []);
  assert.equal(phone.cards, d.lines.length);
});

test("opportunity roadmap renders measured funnel counts and linked-step notes", browser, () => {
  const data = clone(loadSample("opportunity-roadmap"));
  data.funnel = {
    source: { status: "partial", label: "measured steward evidence", note: "Unmatched work omitted" },
    steps: [
      { key: "planned", label: "Planned", count: 58 },
      { key: "linked", label: "Linked to work", count: 35 },
      { key: "runtime", label: "Matched to live work", count: 9 },
      { key: "accepted", label: "Accepted evidence", count: 5 },
    ],
  };
  assert.deepEqual(validate(loadSchema("opportunity-roadmap"), data), []);
  const r = runPage("opportunity-roadmap", { data, width: 390, act: `
    report({ text: document.getElementById("funnel-chart").textContent,
      counts: [].slice.call(document.querySelectorAll("#funnel-chart .kc-fun-n")).map(function (el) { return el.textContent; }),
      rows: document.querySelectorAll("#funnel-chart .kc-fun-row").length,
      source: document.querySelector("#sec-funnel .kc-src").textContent,
      note: document.getElementById("funnel-note").textContent,
      overflow: document.documentElement.scrollWidth > document.documentElement.clientWidth });` });
  assert.deepEqual(r.errors, []);
  assert.deepEqual(r.counts, ["58", "35", "9", "5"]);
  assert.equal(r.rows, 4);
  assert.match(r.source, /partly real/i);
  assert.match(r.note, /distinct recorded roadmap steps/i);
  assert.equal(r.overflow, false);
});

test("opportunity roadmap states the funnel is unavailable without drawing invented counts", browser, () => {
  const data = clone(loadSample("opportunity-roadmap"));
  data.funnel = { source: { status: "unavailable", note: "No linked source facts" }, steps: [] };
  assert.deepEqual(validate(loadSchema("opportunity-roadmap"), data), []);
  const r = runPage("opportunity-roadmap", { data, width: 390, act: `
    report({ notice: document.querySelector("#sec-funnel .kc-unavailable").textContent,
      rows: document.querySelectorAll("#sec-funnel .kc-fun-row").length,
      chartExists: !!document.querySelector("#sec-funnel .kc-fun") });` });
  assert.deepEqual(r.errors, []);
  assert.match(r.notice, /unavailable/i);
  assert.match(r.notice, /No linked source facts/);
  assert.match(r.notice, /Nothing is estimated in its place/);
  assert.equal(r.rows, 0);
  assert.equal(r.chartExists, false);
});

test("opportunity roadmap weight controls stay readable at 320px and 390px", browser, () => {
  for (const width of [320, 390]) {
    const r = runPage("opportunity-roadmap", { width, act: `
      var seg = document.querySelector("#view .kc-weight .kc-seg");
      var buttons = [].slice.call(seg.querySelectorAll("button"));
      report({ columns: new Set(buttons.map(function (b) { return Math.round(b.getBoundingClientRect().left); })).size,
        buttons: buttons.map(function (b) { return { height: b.getBoundingClientRect().height, whiteSpace: getComputedStyle(b).whiteSpace, clipped: b.scrollWidth > b.clientWidth }; }) });` });
    assert.deepEqual(r.errors, [], `${width}px page has no script errors`);
    assert.equal(r.columns, 2, `${width}px uses a readable two-column weight selector`);
    assert.equal(r.buttons.length, 4);
    for (const button of r.buttons) {
      assert.ok(button.height >= 44, `${width}px controls remain touch-sized`);
      assert.equal(button.whiteSpace, "nowrap", `${width}px weight labels do not wrap`);
      assert.equal(button.clipped, false, `${width}px weight labels are fully visible`);
    }
  }
});

test("opportunity roadmap weights do not invent allocation or move saved forecast dates", browser, () => {
  const r = runPage("opportunity-roadmap", { width: 390, hub: { context: { kind: "status" } }, act: `
    var before = document.getElementById("sum").textContent;
    var gateBefore = document.querySelector("#view .kc-card__sub").textContent;
    var button = document.querySelector('#view .kc-weight button[data-v="most"]');
    button.click();
    report({ before: before, summary: document.getElementById("sum").textContent,
      gateBefore: gateBefore,
      gate: document.querySelector("#view .kc-card__sub").textContent,
      effect: document.querySelector("#view .kc-effect").textContent,
      unavailable: document.querySelector("#alloc").textContent });` });
  assert.deepEqual(r.errors, []);
  assert.equal(r.before, r.summary);
  assert.equal(r.gateBefore, r.gate);
  assert.match(r.summary, /Unknown/);
  assert.match(r.unavailable, /unavailable/i);
  assert.match(r.effect, /next steward planning order/i);
  assert.doesNotMatch(r.effect, /Gets .* a week|Estimate:/);
  assert.match(JSON.stringify(r.log), /weight\.O5/);
});

test("research report: qualitative progress and unknown weights stay explicit and choices go into the answer", browser, () => {
  const d = loadSample("research-report");
  const steerable = d.workstreams.items.filter((w) => w.steerable !== false);
  const known = steerable.find((w) => w.weight);
  const unknown = steerable.find((w) => !w.weight);
  const dim = d.scorecard.dimensions[0];
  const r = runPage("research-report", { hub: { context: { kind: "status" } }, act: `
    var rows = document.querySelectorAll("#ws .kc-ws");
    report({ rows: rows.length, segs: document.querySelectorAll("#ws .kc-ws .kc-seg").length, wins: document.querySelectorAll("#wins .kc-win").length,
      dims: document.querySelectorAll("#score .kc-prog:not(.kc-prog--axis)").length, qualitative: document.querySelectorAll("#score .kc-prog--qual").length,
      qualitativeText: document.querySelector("#score .kc-prog--qual .kc-prog__v").textContent, numericSummary: document.querySelector("#overall").textContent,
      unknownText: [].filter.call(rows, function (row) { return row.querySelector(".kc-ws__id").textContent === ${JSON.stringify(unknown.id)}; })[0].querySelector(".kc-ws__current").textContent,
      unknownPressed: [].filter.call(rows, function (row) { return row.querySelector(".kc-ws__id").textContent === ${JSON.stringify(unknown.id)}; })[0].querySelectorAll('.kc-seg button[aria-pressed="true"]').length,
      ranges: document.querySelectorAll("#next .kc-range__bar").length,
      closed: document.querySelector("#score .kc-prog__more").hidden });
    document.querySelector("#score .kc-prog:not(.kc-prog--axis)").click();
    report({ opened: document.querySelector("#score .kc-prog__more").textContent, open: !document.querySelector("#score .kc-prog__more").hidden });
    function weight(id, v) { [].filter.call(document.querySelectorAll("#ws .kc-ws"), function (r) { return r.querySelector(".kc-ws__id").textContent === id; })[0].querySelector('button[data-v="' + v + '"]').click(); }
    weight(${JSON.stringify(known.id)}, "more"); weight(${JSON.stringify(known.id)}, ${JSON.stringify(known.weight)}); weight(${JSON.stringify(unknown.id)}, "hold");
    [].filter.call(document.querySelectorAll("#ws .kc-ws"), function (r) { return r.querySelector(".kc-ws__id").textContent === ${JSON.stringify(unknown.id)}; })[0].querySelector(".kc-ws__clear").click();
    report({ changes: document.getElementById("ws-changes").textContent });
    var groups = {}; [].forEach.call(document.querySelectorAll("#ws details.kc-wsg"), function (g) { groups[g.getAttribute("data-state")] = { open: g.open, rows: g.querySelectorAll(".kc-ws").length, ids: g.querySelector(".kc-wsg__ids").textContent }; });
    report({ groups: groups, first: document.querySelector("#ws details.kc-wsg").getAttribute("data-state") });` });
  assert.deepEqual(r.errors, []);
  assert.equal(r.rows, d.workstreams.items.length);
  assert.equal(r.segs, steerable.length, "finished workstreams have no weight control");
  assert.equal(r.wins, d.measurements.items.length);
  assert.equal(r.dims, d.scorecard.dimensions.length);
  assert.equal(r.qualitative, 1);
  assert.match(r.qualitativeText, /Numeric progress unavailable/);
  assert.match(r.numericSummary, /Unavailable/);
  assert.match(r.numericSummary, /1 of 7 dimensions/);
  assert.match(r.unknownText, /weight unknown/i);
  assert.equal(r.unknownPressed, 0, "an unknown weight does not select a default");
  assert.equal(r.ranges, d.next.items.length);
  assert.equal(r.closed, true);
  assert.equal(r.open, true);
  assert.ok(r.opened.includes(dim.reason) && r.opened.includes(dim.level));
  assert.deepEqual(r.log, [["set", `weight.${known.id}`, "more"], ["remove", `weight.${known.id}`], ["set", `weight.${unknown.id}`, "hold"], ["remove", `weight.${unknown.id}`]]);
  assert.doesNotMatch(r.changes, new RegExp(known.id));
  assert.doesNotMatch(r.changes, new RegExp(`\\b${unknown.id}\\b`));
  // Rows are grouped by state: what needs the operator first and open, parked work collapsed, changes marked.
  assert.equal(r.first, "waiting_on_you");
  assert.equal(r.groups.waiting_on_you.open, true);
  assert.equal(r.groups.parked.open, true, "the group stays open so the operator can see the cleared row");
  assert.equal(r.groups.landed.open, false);
  for (const [k, g] of Object.entries(r.groups)) assert.equal(g.rows, d.workstreams.items.filter((w) => w.state === k).length, k);
  assert.doesNotMatch(r.groups[known.state].ids, /changed/);
});

test("research report says which sections have no source instead of drawing them", browser, () => {
  const d = clone(loadSample("research-report"));
  d.scorecard = { source: { status: "unavailable", note: "The supervisor has not reviewed this commission yet" } };
  delete d.measurements;
  const r = runPage("research-report", { data: d, act: `
    report({ score: document.getElementById("sec-score").textContent, wins: document.getElementById("sec-measure").textContent, ws: document.querySelectorAll("#ws .kc-ws").length });` });
  assert.deepEqual(r.errors, []);
  assert.match(r.score, /unavailable/i);
  assert.match(r.score, /not reviewed/);
  assert.match(r.wins, /unavailable/i);
  assert.equal(r.ws, d.workstreams.items.length, "other sections still render");
});

test("every template opens with empty data and says what is missing", browser, () => {
  for (const name of TEMPLATES) {
    const r = runPage(name, { data: { meta: { title: "Empty" } }, act: `report({ unavailable: document.querySelectorAll(".kc-unavailable").length });` });
    assert.deepEqual(r.errors, [], name);
    assert.ok(r.unavailable >= 1, `${name}: shows unavailable sections`);
  }
});

test("kit folder is where the docs say", () => {
  assert.ok(readFileSync(join(KIT_DIR, "hub-charts.js"), "utf8").includes("window.HubCharts"));
});
