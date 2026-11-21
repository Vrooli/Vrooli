import assert from "node:assert/strict";
import { test } from "vitest";
import { acceptanceClause, breakEven, breakEvenSentence, comparisonSentence, evidenceView, formatRatio, headlineRow, invertRange, rowsByKey, sessionCoverage } from "./evidence.js";
import { makeUsageOutcomes, makeUsageParity } from "../../test-utils/fixtures/usage.js";

test("ratios read one decimal under ten, whole above, and never invent a value", () => {
  assert.equal(formatRatio(3.64), "3.6x");
  assert.equal(formatRatio(12.4), "12x");
  assert.equal(formatRatio(undefined), "—");
  assert.equal(formatRatio(Number.NaN), "—");
});

test("an observational result states the path, the cost ratio with its range and the unknown share, and claims no cause", () => {
  const view = evidenceView(makeUsageOutcomes("observational"), "last 30 days");
  assert.equal(view.state, "observational");
  assert.equal(view.conclusion, "On operator sessions, luna cost about 3.6x less per accepted outcome (range 2.6x to 4.5x) than terra, using 1.4x the reference tokens; 28% of outcomes are unknown.");
  assert.match(view.strength, /an association to look into, not what the model caused/);
  assert.doesNotMatch(`${view.conclusion} ${view.strength}`, /\bso\b.*cheaper|because|fair/);
});

test("a randomized reading is gated on the recorded design and never calls the comparison fair", () => {
  const view = evidenceView(makeUsageOutcomes("randomized"), "last 30 days");
  assert.equal(view.state, "randomized");
  assert.match(view.strength, /drew its model by chance, recorded per run/);
  assert.doesNotMatch(view.strength, /fair/);
  const days = evidenceView({ ...makeUsageOutcomes("randomized"), design: "day_block_randomized" }, "last 30 days");
  assert.equal(days.state, "day_block_randomized");
  assert.match(days.strength, /Whole days drew their model by chance, so a day counts once/);
  const unrecorded = evidenceView({ ...makeUsageOutcomes("randomized"), design: "" }, "last 30 days");
  assert.equal(unrecorded.state, "observational");
});

test("a range too wide to call reads inconclusive, never about the same or cheaper", () => {
  const response = makeUsageOutcomes("inconclusive");
  const row = headlineRow(response.rows);
  assert.ok(row);
  const sentence = comparisonSentence(row);
  assert.match(sentence, /^Inconclusive on operator sessions: luna cost <0\.1x to 7\.7x of terra per accepted outcome/);
  assert.doesNotMatch(sentence, /about the same|cheaper by|cost about/);
  assert.match(comparisonSentence({ ...row, verdict: "about_same", cheaperBy: { ...row.cheaperBy!, value: 1, low: 0.9, high: 1.1 } }), /about the same per accepted outcome: luna cost 0\.9x to 1\.1x of terra, all inside the 30% margin/);
});

test("the strength says what acceptance rests on and what only finished", () => {
  const row = headlineRow(makeUsageOutcomes("observational").rows);
  assert.ok(row);
  assert.equal(acceptanceClause({ ...row, completed: 0n, evidence: [] }), "");
  const clause = acceptanceClause({ ...row, completed: 12n, evidence: [{ source: "validation", units: 2n }, { source: "agent_verdict", units: 1n }, { source: "headless_status", units: 12n }] } as typeof row);
  assert.equal(clause, "Accepted here means a passing validation or the agent's own verdict, not your review. 12 more units only finished; acceptance is unknown and they do not count as accepted.");
  assert.match(evidenceView(makeUsageOutcomes("early"), "last 7 days").strength, /12 more units only finished, with acceptance unknown/);
});

test("rows of one role across paths list once", () => {
  const rows = makeUsageOutcomes("observational").rows;
  const merged = rowsByKey([{ ...rows[0]!, stratum: "headless", units: 3n }, { ...rows[0]!, stratum: "operator", units: 4n }]);
  assert.equal(merged.length, 1);
  assert.equal(merged[0]?.units, 7n);
});

test("with no mode in the period and no comparable models the state is no mode, with no ratio", () => {
  const view = evidenceView(makeUsageOutcomes("none"), "last 30 days");
  assert.equal(view.state, "no_mode");
  assert.match(view.conclusion, /No usage mode ran in the last 30 days/);
  assert.doesNotMatch(view.conclusion, /\dx/);
});

test("with a mode but too few accepted outcomes the state is not enough data and says how many exist", () => {
  const view = evidenceView(makeUsageOutcomes("early"), "last 7 days");
  assert.equal(view.state, "not_enough_data");
  assert.match(view.conclusion, /Too early to say\. 4 accepted outcomes so far in the last 7 days; a model needs 5 on one path/);
});

test("a row costing more is stated as such, inverted from the cheaper-by range", () => {
  const row = headlineRow(makeUsageOutcomes("observational").rows);
  assert.ok(row);
  const dearer = { ...row, verdict: "more_expensive", cheaperBy: { ...row.cheaperBy!, value: 0.5, low: 0.4, high: 0.8 } };
  assert.match(comparisonSentence(dearer as typeof row), /luna cost about 2\.0x more per accepted outcome \(range 1\.3x to 2\.5x\)/);
  assert.equal(invertRange({ ...row.cheaperBy!, available: false }), undefined);
});

test("break-even separates the realized price from the model price and names a prior", () => {
  const luna = makeUsageOutcomes("observational").rows[1];
  const prior = breakEven(luna);
  assert.ok(prior?.prior);
  const sentence = breakEvenSentence(prior);
  assert.match(sentence, /luna cost 5\.0x less per reference token than terra on these units/);
  assert.match(sentence, /The model price alone is 4\.0x less at one fixed token mix \(a prior from its tier, not a list price\)/);
  assert.match(sentence, /It used 1\.4x the reference tokens per accepted outcome \(range 1\.1x to 1\.9x\), so it stays cheaper while that stays under 5\.0x/);
  const fitted = breakEven(makeUsageOutcomes("randomized").rows[1]);
  assert.equal(fitted?.prior, false);
  assert.match(breakEvenSentence(fitted!), /from the price book/);
  assert.doesNotMatch(breakEvenSentence(fitted!), /prior/);
  assert.equal(breakEven(makeUsageOutcomes("observational").rows[0]), undefined);
  // The two figures multiply back to the cost ratio shown beside them.
  assert.ok(Math.abs(prior.advantage / prior.used.value - prior.cheaperBy.value) < 0.05);
  // No model price on record is said, not filled in.
  assert.match(breakEvenSentence({ ...prior, modelPrice: undefined }), /No model price is on record/);
});

test("coverage by session type sums paths and reports unknown shares as undefined", () => {
  const coverage = sessionCoverage(makeUsageParity().rows);
  assert.deepEqual(coverage.map((c) => [c.id, c.usageShare, c.outcomeShare]), [["headless", 0.92, 0.8], ["interactive", 0.5, 0.4], ["imported", 0.35, 0.1]]);
  assert.equal(sessionCoverage([])[0]?.usageShare, undefined);
});
