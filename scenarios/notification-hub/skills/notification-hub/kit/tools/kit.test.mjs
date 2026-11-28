// Tests for the Notification Hub chart kit and its four templates.
//   node --test scenarios/notification-hub/skills/notification-hub/kit/tools/kit.test.mjs
// The browser tests run each template with its sample data in headless Chrome
// (no Hub bridge, so read-only) and skip when Chrome is not installed.
import { test } from "node:test";
import assert from "node:assert/strict";
import { execFileSync, spawnSync } from "node:child_process";
import { mkdtempSync, mkdirSync, readFileSync, writeFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import {
  KIT_DIR, MAX_PAGE_BYTES, embedJSON, kitSources, loadSample, loadSchema, loadTemplate, render, syncTemplate, templateNames, validate,
} from "./kit.mjs";

const TEMPLATES = ["daily-briefing", "evidence-decision", "opportunity-roadmap", "research-report", "structural-change", "usage-mode"];
const clone = (v) => JSON.parse(JSON.stringify(v));

test("all six templates exist, each with a schema and a sample", () => {
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

  const evidence = clone(loadSample("evidence-decision"));
  evidence.record.decidedBy = "system";
  evidence.record.title = "x".repeat(81);
  const evidenceErrors = validate(loadSchema("evidence-decision"), evidence);
  assert.ok(evidenceErrors.some((e) => e.includes("decidedBy")), evidenceErrors.join("\n"));
  assert.ok(evidenceErrors.some((e) => e.includes("title")), evidenceErrors.join("\n"));

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

test("research report enforces brief lines and carries asks, evidence, and feedback receipts", () => {
  const schema = loadSchema("research-report");
  const sample = clone(loadSample("research-report"));
  assert.deepEqual(validate(schema, sample), []);
  assert.equal(schema.properties.evidence.items.$ref, "#/$defs/evidence");
  assert.deepEqual(schema.$defs.evidence, loadSchema("evidence-decision").$defs.evidence, "report embeds the same canonical evidence record definition");
  const badLine = clone(sample); badLine.workstreams.items[0].note = "x".repeat(91);
  assert.ok(validate(schema, badLine).some((e) => e.includes("note")));
  const longLede = clone(sample); longLede.meta.lede = "x".repeat(141);
  assert.ok(validate(schema, longLede).some((e) => e.includes("lede")));
  const badTitle = clone(sample); badTitle.meta.title = "x".repeat(81);
  assert.ok(validate(schema, badTitle).some((e) => e.includes("title")));
  const badVerdict = clone(sample); badVerdict.evidence[0].verdict.text = "x".repeat(141);
  assert.ok(validate(schema, badVerdict).some((e) => e.includes("verdict") || e.includes("text")));
  const badMethod = clone(sample); badMethod.evidence[0].method[0] = "x".repeat(91);
  assert.ok(validate(schema, badMethod).some((e) => e.includes("method")));
  const badAsk = clone(sample); badAsk.asks[0].state = "complete";
  assert.ok(validate(schema, badAsk).some((e) => e.includes("state")));
  assert.ok(sample.asks.every((ask) => ask.id && ask.title && ask.workstream));
  assert.ok(sample.evidence.every((record) => record.format && record.sides.a && record.sides.b));
  assert.ok(sample.feedbackReceipts.every((receipt) => receipt.said && receipt.status && receipt.progress));
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

test("evidence-decision embeds the canonical evidence record schema without drift", () => {
  const standalone = clone(loadSchema("evidence"));
  const decision = loadSchema("evidence-decision");
  const side = standalone.$defs.side;
  delete standalone.$schema; delete standalone.$id; delete standalone.title; delete standalone.description; delete standalone.$defs;
  standalone.properties.sides.properties = { a: side, b: clone(side) };
  standalone.description = decision.$defs.evidence.description;
  assert.deepEqual(decision.$defs.evidence, standalone);
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
function runPage(name, { data, width = 400, act = "", hub = null, assets = {} } = {}) {
  const probe = `<script>
    window.__errors = [];
    window.addEventListener("error", function (e) { window.__errors.push(String(e.message)); });
    window.__log = [];
    ${hub ? `var __testAssets = ${JSON.stringify(hub.assets || {})}; window.hub = { context: ${JSON.stringify(hub.context || {})}, prefill: function (q, v) { window.__log.push(["prefill", q, v]); },
      answer: { set: function (k, l, v) { window.__log.push(["set", k, v]); }, remove: function (k) { window.__log.push(["remove", k]); } }, note: function(k,l){window.__log.push(["note",k,l]);}, openAsk: function(id){window.__log.push(["openAsk",id]);}, asset: function (ref) { window.__log.push(["asset", ref]); return Promise.resolve(__testAssets[ref] || ref); } };` : ""}
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
      }, 700);
    }, 400); });
  </script>`;
  let page = render(loadTemplate(name), data || loadSample(name)).replace("<head>", "<head>" + probe).replace("</body>", finish + "</body>");
  const dir = mkdtempSync(join(tmpdir(), "hub-kit-"));
  try {
    const file = join(dir, "page.html");
    for (const [name, contents] of Object.entries(assets)) { const dest = join(dir, name); mkdirSync(dest.slice(0, dest.lastIndexOf("/")), { recursive: true }); writeFileSync(dest, contents); }
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
        var badge = document.querySelector("${name === "opportunity-roadmap" ? "#hours-src .kc-src" : "#sec-target .kc-src"}");
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

test("opportunity roadmap shows current milestone range and its saved slip history without a first-money gate", browser, () => {
  const data = clone(loadSample("opportunity-roadmap"));
  const line = data.lines[0];
  delete line.gate;
  delete line.draft;
  line.milestoneGate = { label: "M1 · Current actionable step", from: "2026-11-17T00:00:00Z", to: "2026-11-25T00:00:00Z" };
  line.slip = {
    source: { status: "real", label: "saved milestone reviews" },
    reviews: [
      { label: "Prior review", from: "2026-11-15T00:00:00Z", to: "2026-11-23T00:00:00Z" },
      { label: "Latest review", from: "2026-11-17T00:00:00Z", to: "2026-11-25T00:00:00Z" },
    ],
  };
  assert.deepEqual(validate(loadSchema("opportunity-roadmap"), data), []);
  const r = runPage("opportunity-roadmap", { data, width: 390, act: `
    var detail = document.querySelector("#view .kc-card__detail");
    report({ detail: detail.textContent,
      charts: detail.querySelectorAll(".kc-box:last-child svg").length,
      sub: document.querySelector("#view .kc-card__sub").textContent,
      overflow: document.documentElement.scrollWidth > document.documentElement.clientWidth });` });
  assert.deepEqual(r.errors, []);
  assert.match(r.detail, /Current milestone range · M1 · Current actionable step: 17 Nov–25 Nov\. This is not a first-money date\./);
  assert.match(r.detail, /Saved forecast history · M1 · Current actionable step/);
  assert.match(r.detail, /Named milestone · slipped 2 days over 2 reviews/);
  assert.equal(r.charts, 1, "the milestone history chart renders without a terminal first-money gate");
  assert.match(r.sub, /Current milestone · M1/);
  assert.doesNotMatch(r.sub, /First money/);
  assert.equal(r.overflow, false);
});

test("opportunity roadmap marks recorded if-launched-now ranges as conditional in the card and timeline", browser, () => {
  const data = clone(loadSample("opportunity-roadmap"));
  const line = data.lines[0];
  line.plan = true;
  line.gate = { label: "First payment", from: "2026-11-17T00:00:00Z", to: "2026-11-25T00:00:00Z", ifLaunchedNow: true };
  line.slip = {
    source: { status: "real", label: "saved final-step gate history" },
    reviews: [{ label: "Latest review", from: line.gate.from, to: line.gate.to, ifLaunchedNow: true }],
  };
  assert.deepEqual(validate(loadSchema("opportunity-roadmap"), data), []);
  const r = runPage("opportunity-roadmap", { data, width: 390, act: `
    var card = document.querySelector("#view .kc-card");
    var range = card.querySelector(".kc-card__sub");
    var gate = card.querySelector("svg rect[stroke-dasharray='4 3']");
    var detail = card.querySelector(".kc-card__detail");
    report({ range: range.textContent, conditionalNote: detail.textContent,
      dashed: !!gate, overflow: document.documentElement.scrollWidth > document.documentElement.clientWidth });` });
  assert.deepEqual(r.errors, []);
  assert.match(r.range, /Conditional first-money range/);
  assert.match(r.conditionalNote, /Conditional range: this saved estimate assumes launch at the saved review time shown in its history/);
  assert.equal(r.dashed, true, "the timeline marks the recorded assumption visually");
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

test("roadmap changed row and data notes respond to selection and undo", browser, () => {
  const data = clone(loadSample("opportunity-roadmap"));
  data.meta.notice = "Phase intervals and prices are unavailable.";
  for (const width of [390, 1100]) {
    const r = runPage("opportunity-roadmap", { data, width, hub: { context: { kind: "status" } }, act: `
      var notes = document.getElementById("data-notes");
      var button = document.querySelector(".kc-weight button[data-v=most]"); button.click();
      var changed = document.querySelectorAll("#view .is-changed").length;
      var key = window.__log.filter(function (x) { return x[0] === "set"; }).pop()[1];
      window.dispatchEvent(new CustomEvent("hub:answer-remove", { detail: { key: key } }));
      var undone = document.querySelectorAll("#view .is-changed").length;
      var changes = {}; changes[key] = { label: "Restored", value: "most" };
      window.dispatchEvent(new CustomEvent("hub:answer-restore", { detail: { changes: changes } }));
      report({ closed: !notes.open, summary: notes.querySelector("summary").textContent, details: notes.textContent, changed: changed, undone: undone, restored: document.querySelectorAll("#view .is-changed").length });` });
    assert.deepEqual(r.errors, []);
    assert.equal(r.closed, true); assert.equal(r.summary, "Data notes");
    assert.match(r.details, /Phase intervals and prices/);
    assert.match(r.details, /Missing dates stay blank/);
    assert.equal(r.changed, 1); assert.equal(r.undone, 0); assert.equal(r.restored, 1); assert.equal(r.log.filter((entry) => entry[0] === "set").length, 1);
  }
});

test("roadmap sample keeps its illustrative label visible and its data notes collapsed", browser, () => {
  const r = runPage("opportunity-roadmap", { act: `
    var notes = document.getElementById("data-notes");
    report({ open: notes.open, label: notes.querySelector("summary").textContent,
      missingOutside: Array.from(document.querySelectorAll("#page > header > p")).some(function (p) { return /Missing dates stay blank/.test(p.textContent); }) });` });
  assert.deepEqual(r.errors, []); assert.equal(r.open, false);
  assert.match(r.label, /Illustrative sample/); assert.equal(r.missingOutside, false);
});

test("research report keeps qualitative progress, priority controls, and weight answer keys", browser, () => {
  const d = loadSample("research-report");
  const steerable = d.workstreams.items.filter((w) => w.steerable !== false);
  const known = steerable.find((w) => w.weight);
  const unknown = steerable.find((w) => !w.weight);
  const dim = d.scorecard.dimensions.find((x) => x.key === "runtime");
  const r = runPage("research-report", { hub: { context: { kind: "status" } }, act: `
    function row(id) { return [].find.call(document.querySelectorAll("#ws .rr-ws"), function (x) { return x.querySelector(".rr-tag").textContent === id; }); }
    function weight(id, label) { var r=row(id); r.open=true; [].find.call(r.querySelectorAll(".rr-seg button"), function (b) { return b.textContent === label; }).click(); }
    report({ rows: document.querySelectorAll("#ws .rr-ws").length, segs: Array.from(document.querySelectorAll("#ws .rr-ws")).filter(function (x) { return x.querySelectorAll(".rr-seg button").length > 0; }).length,
      wins: document.querySelectorAll("#changed .rr-mrow").length, dims: document.querySelectorAll("#score .rr-score-row").length,
      qualitative: [].find.call(document.querySelectorAll("#score .rr-score-row"), function (x) { return x.querySelector(".rr-score-title").textContent === ${JSON.stringify(dim.title)}; }).textContent,
      ranges: document.querySelectorAll("#ranges .rr-range").length,
      groups: Array.from(document.querySelectorAll("#ws .rr-wgroup")).map(function (x) { return x.getAttribute("data-state"); }) });
    weight(${JSON.stringify(known.id)}, "More"); weight(${JSON.stringify(known.id)}, "Normal"); weight(${JSON.stringify(unknown.id)}, "Hold");
    report({ changes: document.getElementById("ws-changes").textContent });` });
  assert.deepEqual(r.errors, []);
  assert.equal(r.rows, d.workstreams.items.length);
  assert.equal(r.segs, steerable.length, "finished workstreams have no priority control");
  assert.equal(r.wins, d.measurements.items.length);
  assert.equal(r.dims, d.scorecard.dimensions.length);
  assert.match(r.qualitative, new RegExp(dim.level));
  assert.match(r.qualitative, new RegExp(dim.reason));
  assert.equal(r.ranges, d.next.items.length);
  assert.deepEqual(r.log, [["set", `weight.${known.id}`, "more"], ["remove", `weight.${known.id}`], ["set", `weight.${unknown.id}`, "hold"]]);
  assert.doesNotMatch(r.changes, new RegExp(`\\b${known.id}\\b`));
  assert.ok(r.groups.includes("waiting_on_you") && r.groups.includes("parked") && r.groups.includes("landed"));
});

test("research report links live asks and renders evidence and feedback receipts", browser, () => {
  const d = loadSample("research-report");
  const open = d.asks[0], closed = d.asks[1];
  const ctx = { kind: "status", asks: {
    [open.id]: { state: "open", askedAt: "2026-11-16T18:00:00Z" },
    [closed.id]: { state: "approved", answeredAt: "2026-11-16T20:00:00Z" },
  } };
  const r = runPage("research-report", { data: d, width: 320, hub: { context: ctx }, act: `
    var asks = document.querySelectorAll("#asks .rr-need");
    asks[0].querySelector("button").click();
    var receipt = document.querySelector("#receipts .rr-receipt");
    receipt.querySelector(".ev-note").click();
    document.documentElement.style.width = "320px"; document.body.style.width = "320px"; document.body.style.margin = "0";
    location.hash = "#evidence";
    report({ askCount: asks.length, askState: asks[0].textContent, approvedState: asks[1].textContent, evidenceCount: document.querySelectorAll("#evidence-list .ev-card").length,
      evidenceTitle: document.querySelector("#evidence-list .ev-card h2").textContent, teaserCount: document.querySelectorAll("#teaser .rr-teaser-card").length,
      receiptCount: document.querySelectorAll("#receipts .rr-receipt").length, gauge: receipt.querySelector(".rr-gauge").getAttribute("aria-label") || receipt.querySelector(".rr-gauge").textContent, pageWidth: document.documentElement.scrollWidth, bodyWidth: document.body.scrollWidth, rightmost: Array.from(document.querySelectorAll("#view-evidence *")).filter(function(el){return el.getClientRects().length;}).reduce(function(m,el){return Math.max(m,el.getBoundingClientRect().right);},0) });` });
  assert.deepEqual(r.errors, []);
  assert.equal(r.askCount, 2, "approved asks remain visible until their decision is run");
  assert.match(r.approvedState, /Approved.*not run yet/i);
  assert.match(r.askState, /open/i);
  assert.match(r.askState, /asked/i);
  assert.deepEqual(r.log.filter((entry) => entry[0] === "openAsk").map((entry) => entry[1]), [open.id]);
  assert.deepEqual(r.log.filter((entry) => entry[0] === "note").map((entry) => entry[1]), [`fb.${d.feedbackReceipts[0].id}`]);
  assert.equal(r.evidenceCount, d.evidence.length);
  assert.equal(r.evidenceTitle, d.evidence[0].title);
  assert.equal(r.teaserCount, d.evidence.length);
  assert.equal(r.receiptCount, d.feedbackReceipts.length);
  assert.match(r.gauge, /was/i);
  assert.ok(r.bodyWidth <= 320, `report body overflows at 320px: ${r.bodyWidth}px`);
  assert.ok(r.rightmost <= 320, `evidence content crosses the 320px viewport: ${r.rightmost}px`);
});

test("research report marks unavailable scorecard and measurements without hiding workstreams", browser, () => {
  const d = clone(loadSample("research-report"));
  d.scorecard = { source: { status: "unavailable", note: "The supervisor has not reviewed this commission yet" } };
  d.measurements = { source: { status: "unavailable", note: "No measurements were retained" } };
  const r = runPage("research-report", { data: d, act: `
    report({ score: document.getElementById("score").textContent, wins: document.getElementById("changed").textContent,
      ws: document.querySelectorAll("#ws .rr-ws").length });` });
  assert.deepEqual(r.errors, []);
  assert.match(r.score, /unavailable/i);
  assert.match(r.score, /not reviewed/i);
  assert.match(r.wins, /unavailable/i);
  assert.equal(r.ws, d.workstreams.items.length, "other sections still render");
});

test("every template opens with empty data and says what is missing", browser, () => {
  for (const name of TEMPLATES) {
    const r = runPage(name, { data: { meta: { title: "Empty" } }, act: `var empty=document.getElementById("evidence-empty"); report({ unavailable: document.querySelectorAll(".kc-unavailable").length, evidenceEmpty: !!empty && !empty.hidden });` });
    assert.deepEqual(r.errors, [], name);
    assert.ok(r.unavailable >= 1 || (name === "research-report" && r.evidenceEmpty), `${name}: shows a clear empty-state notice`);
  }
});

test("kit folder is where the docs say", () => {
  assert.ok(readFileSync(join(KIT_DIR, "hub-charts.js"), "utf8").includes("window.HubCharts"));
});


test("evidence kit renders each viewer with real image, audio and video attachments at 320px", browser, () => {
  const dirAssets = {
    "media/a.png": Buffer.from("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR4nGP4z8DwHwAFAAH/iZk9HQAAAABJRU5ErkJggg==", "base64"),
    "media/b.png": Buffer.from("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR4nGNgYPj/HwADAgH/5ncLrgAAAABJRU5ErkJggg==", "base64"),
  };
  const wav = Buffer.alloc(44 + 8000);
  wav.write("RIFF",0); wav.writeUInt32LE(wav.length-8,4); wav.write("WAVEfmt ",8); wav.writeUInt32LE(16,16); wav.writeUInt16LE(1,20); wav.writeUInt16LE(1,22); wav.writeUInt32LE(8000,24); wav.writeUInt32LE(8000,28); wav.writeUInt16LE(1,32); wav.writeUInt16LE(8,34); wav.write("data",36); wav.writeUInt32LE(8000,40); wav.fill(128,44);
  dirAssets["media/a.wav"] = wav; dirAssets["media/b.wav"] = wav;
  const dir = mkdtempSync(join(tmpdir(), "hub-evidence-media-"));
  try {
    const video = join(dir, "media.webm");
    const ffmpeg = spawnSync("ffmpeg", ["-loglevel","error","-f","lavfi","-i","color=c=red:s=32x32:d=1:r=10","-c:v","libvpx","-deadline","realtime","-y",video]);
    assert.equal(ffmpeg.status, 0, "ffmpeg creates a real WebM test clip");
    dirAssets["media/a.webm"] = readFileSync(video); dirAssets["media/b.webm"] = readFileSync(video);
  } finally { rmSync(dir, { recursive:true, force:true }); }
  const side = (label, src, fidelity) => ({ label, src, ...(fidelity ? { fidelity } : {}) });
  const rec = (id, format, sides, items = []) => ({ id, title: id, format, decidedBy:"agent", sides, items, verdict:{text:"Compared with retained inputs."}, method:["Same source data."], limits:["Small fixture."] });
  const records = [
    rec("image", "image", {a:side("A","media/a.png"),b:side("B","media/b.png")}, [{label:"image",a:"media/a.png",b:"media/b.png"}]),
    rec("audio", "audio", {a:{...side("A","media/a.wav"),transcript:[{word:"hello",start:0,end:.1},{word:"world",start:.1,end:.4}]},b:{...side("B","media/b.wav"),transcript:[{word:"hello",start:0,end:.1},{word:"world",start:.1,end:.4}]}}),
    rec("video", "video", {a:side("A","media/a.webm"),b:side("B","media/b.webm")}),
    rec("text", "text", {a:{label:"A",text:"blue sky"},b:{label:"B",text:"blue sea"}}),
    rec("fields", "fields", {a:{label:"A",fields:{score:2}},b:{label:"B",fields:{score:3}}}),
    rec("ui", "ui", {a:side("Before","media/a.png","captured"),b:side("After","media/b.png","composited")}, [{label:"screen",a:"media/a.png",b:"media/b.png"}]),
    rec("measure", "measure", {a:side("A"),b:side("B")}, [{label:"Latency",a:[21,22,19],b:[15,15.5,14],unit:"ms"}]),
  ];
  const r = runPage("evidence-decision", {data:{records,options:[{key:"candidate",label:"Use candidate"},{key:"baseline",label:"Keep baseline"}]}, assets:dirAssets, hub:{assets:{"media/a.png":"data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR4nGP4z8DwHwAFAAH/iZk9HQAAAABJRU5ErkJggg==","media/b.png":"data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR4nGNgYPj/HwADAgH/5ncLrgAAAABJRU5ErkJggg=="}}, width:320, act:`
    document.documentElement.style.width="320px"; document.body.style.width="320px"; document.body.style.margin="0";
    var image=document.querySelector('[data-format="image"]');
    var modes=image.querySelectorAll('.ev-seg button'); modes[1].click();
    var slider=image.querySelector('input[type=range]'); slider.value='73'; slider.dispatchEvent(new Event('input',{bubbles:true}));
    image.querySelectorAll(".ev-seg button")[2].click();
    setTimeout(function(){var hasDiff=!!image.querySelector("canvas.ev-diff"), imagesLoaded=Array.from(image.querySelectorAll(".ev-stage-image img")).every(function(x){return x.complete&&x.naturalWidth>0;});image.querySelectorAll(".ev-seg button")[1].click();slider=image.querySelector('input[type=range]');report({diffCanvas:hasDiff,assetCalls:window.__log.filter(function(x){return x[0]==="asset";}).length,imageLoaded:imagesLoaded,audioLoaded:document.querySelectorAll("[data-format=audio] audio").length===2,videoLoaded:document.querySelectorAll("[data-format=video] video").length===2,pressed:Array.from(image.querySelectorAll(".ev-seg button")).map(x=>x.getAttribute("aria-pressed")),sliderValue:slider.value,sliderPos:image.querySelector(".ev-clip").style.width,audioPosition:document.querySelector("[data-format=audio] audio").currentTime,transcriptNow:document.querySelector("[data-format=audio] .ev-lane.b .ev-word.now")&&document.querySelector("[data-format=audio] .ev-lane.b .ev-word.now").textContent,videoControls:document.querySelectorAll("[data-format=video] .ev-video-controls button").length,videoPosition:document.querySelectorAll("[data-format=video] video")[1].currentTime});},500);
    var audio=document.querySelector('[data-format="audio"]'); audio.querySelector('audio').currentTime=.2; audio.querySelector('audio').dispatchEvent(new Event("timeupdate"));
    audio.querySelectorAll('.ev-seg button')[1].click();
    var va=document.querySelector('[data-format="video"] video'); va.currentTime=.35; va.dispatchEvent(new Event('timeupdate'));
    document.querySelector('[data-format="image"] .ev-note').click();
    document.querySelector("[data-format=image] .ev-media-side");
    document.querySelector(".kc-panel .kc-btn").click();
    report({formats:Array.from(document.querySelectorAll('.ev-card')).map(function(card){return {format:card.querySelector('.ev-pill').textContent,who:card.querySelector('.ev-who').textContent.trim()};}), measureDots:document.querySelectorAll("[data-format=measure] .track .dot").length, imageLoaded:!!document.querySelector('[data-format="image"] img').naturalWidth,
      audioLoaded:document.querySelectorAll('[data-format="audio"] audio').length===2, videoLoaded:document.querySelectorAll('[data-format="video"] video').length===2,
      audioPosition:audio.querySelectorAll('audio')[0].currentTime,
      videoPosition:document.querySelectorAll('[data-format="video"] video')[1].currentTime, uiFidelity:Array.from(document.querySelectorAll('[data-format="ui"] .ev-label')).map(x=>x.textContent), width:document.body.scrollWidth, viewport:document.body.clientWidth});`});
  assert.deepEqual(r.errors, []);
  assert.deepEqual(r.formats, ["Image","Audio","Video","Text","Fields","UI","Measure"].map((format) => ({format, who:"Your call"}))); assert.equal(r.measureDots,6);
  assert.ok(r.imageLoaded); assert.ok(r.diffCanvas); assert.ok(r.assetCalls>=8); assert.ok(r.log.some(function(x){return x[0]==="note"&&x[1]==="evidence.image";})); assert.ok(r.log.some(function(x){return x[0]==="prefill"&&x[2]==="candidate";})); assert.ok(r.audioLoaded); assert.ok(r.videoLoaded);
  assert.equal(r.sliderValue,"73"); assert.equal(r.sliderPos,"73%");
  assert.ok(Math.abs(r.audioPosition-.2)<.01, "A/B switch retains playback position"); assert.equal(r.transcriptNow,"world"); assert.ok(r.videoControls>=3);
  assert.ok(Math.abs(r.videoPosition-.35)<.05, "video follower seeks to leader clock");
  assert.deepEqual(r.uiFidelity, ["Before · captured", "After · composited"]);
  assert.equal(r.viewport, 320, JSON.stringify(r));
  assert.ok(r.width <= r.viewport, `no horizontal overflow at 320px (${r.width}/${r.viewport})`);
});
