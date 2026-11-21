import assert from "node:assert/strict";
import { gateWord, isSilent, measurementText, parseGate, proofModel, whyUnproven } from "./annotationGate.js";

const rows = [
  { kind: "coverage", pack: "default", pack_version: "1", eligible_runs: "20", observed_runs: "5", coverage: "0.25" },
  { kind: "label", label: "operator.pleased", group: "operator", scope: "message", state: "unproven", friction: "false", observed: "7", eligible: "30", proven_observed: "0", gate_state: "insufficient_gold", gold: "12", min_gold: "40", target_precision: "0.85", target_recall: "0.7" },
  { kind: "model", label: "operator.pleased", model: "laya:en", resource_role: "decide.local", observed: "7", proven: "false", gate_state: "insufficient_gold" },
];

test("a cell the API left out stays undefined instead of reading as zero", () => {
  const gate = parseGate(rows);
  const [label] = gate.labels;
  assert.equal(gate.humanMessages, undefined);
  assert.equal(label?.rate, undefined);
  assert.equal(label?.precision, undefined);
  assert.equal(label?.calibration, undefined);
  assert.equal(measurementText(label!), "", "no measurement, no measured numbers");
  assert.equal(proofModel(label!), "");
  assert.equal(label?.models[0]?.model, "laya:en");
});

test("the gate word is proven, partial or unproven and nothing else", () => {
  assert.equal(gateWord({ state: "proven" }), "proven");
  assert.equal(gateWord({ state: "partial" }), "partial");
  for (const state of ["unproven", "not_observed", "unavailable", ""]) assert.equal(gateWord({ state }), "unproven");
});

test("why a label is unproven names what is missing or how far it falls short", () => {
  const base = { gateModel: "laya:en", gateRole: "decide.local", gold: 12, minGold: 40, precision: 0.5, recall: 0.4, targetPrecision: 0.85, targetRecall: 0.7 };
  assert.match(whyUnproven({ ...base, gateState: "insufficient_gold" }), /12 of 40 gold rows/);
  assert.match(whyUnproven({ ...base, gateState: "unproven" }), /precision 0\.50 \/ recall 0\.40 is below target 0\.85 \/ 0\.70/);
  assert.match(whyUnproven({ ...base, gateState: "tuning_only" }), /laya:en was measured on its tuning rows only/);
  assert.match(whyUnproven({ ...base, gateState: "unmeasured" }), /no hold-out measurement/);
  assert.match(whyUnproven({ ...base, gateState: "derived" }), /derived from other labels/);
  assert.match(whyUnproven({ ...base, gateState: "proven" }), /proven for laya:en · decide\.local, but none of the observed answers came from it/);
});

test("a label is silent when nothing was observed and it is not proven", () => {
  const [label] = parseGate(rows).labels;
  assert.equal(isSilent(label!), false);
  assert.equal(isSilent({ ...label!, observed: 0 }), true);
  assert.equal(isSilent({ ...label!, observed: 0, state: "proven" }), false);
});
