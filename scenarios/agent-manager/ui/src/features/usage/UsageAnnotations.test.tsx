import assert from "node:assert/strict";
import { screen, within } from "@testing-library/react";
import { renderWithProviders } from "../../test-utils/index.js";
import { Annotations } from "./UsageAnnotations.js";
import { parseGate, type GateCells } from "./annotationGate.js";

const PROVEN_E4B = {
  gate_state: "proven", gate_model: "winnow:e4b", gate_role: "decide.local.accurate", split: "holdout", precision: "0.9", recall: "0.8",
  ece: "0.042", gold: "44", gold_positives: "9", temperature: "1.4", calibration: "calibrated", target_precision: "0.85", target_recall: "0.7", min_gold: "40",
};

function gateRows(): GateCells[] {
  return [
    { kind: "coverage", pack: "default", pack_version: "7", eligible_runs: "10", observed_runs: "4", coverage: "0.4", human_text_runs: "3", human_messages: "30" },
    { kind: "label", label: "work.blocked_by_tool", group: "work", scope: "run", state: "proven", friction: "true", observed: "4", eligible: "4", proven_observed: "4", rate: "0.5", ...PROVEN_E4B },
    { kind: "model", label: "work.blocked_by_tool", model: "winnow:e4b", resource_role: "decide.local.accurate", observed: "4", proven: "true", gate_state: "proven" },
    { kind: "label", label: "operator.dissatisfied_run", group: "operator", scope: "run", state: "partial", friction: "true", observed: "6", eligible: "9", proven_observed: "2", rate: "0.5", ...PROVEN_E4B },
    { kind: "model", label: "operator.dissatisfied_run", model: "winnow:e4b", resource_role: "decide.local.accurate", observed: "2", proven: "true", gate_state: "proven" },
    { kind: "model", label: "operator.dissatisfied_run", model: "laya:en", resource_role: "decide.local", observed: "4", proven: "false", gate_state: "unmeasured" },
    { kind: "label", label: "agent.discovery_slow", group: "agent", scope: "run", state: "unproven", friction: "true", observed: "3", eligible: "4", proven_observed: "0", gate_state: "tuning_only", gate_model: "rule", split: "tuning", precision: "1", recall: "1", gold: "47", calibration: "rule", target_precision: "0.85", target_recall: "0.7", min_gold: "40" },
    { kind: "label", label: "operator.dissatisfied", group: "operator", scope: "message", state: "unproven", friction: "false", observed: "3", eligible: "30", proven_observed: "0", gate_state: "unmeasured", target_precision: "0.85", target_recall: "0.7", min_gold: "40" },
    { kind: "label", label: "run.type", group: "run", scope: "run", state: "proven", friction: "false", observed: "4", eligible: "4", proven_observed: "4", ...PROVEN_E4B },
    { kind: "label", label: "agent.never_seen", group: "agent", scope: "run", state: "not_observed", friction: "true", observed: "0", eligible: "0", proven_observed: "0", gate_state: "unmeasured", target_precision: "0.85", target_recall: "0.7", min_gold: "40" },
  ];
}

test("every rate names the model that proves it and how its probabilities are read", () => {
  renderWithProviders(<Annotations data={parseGate(gateRows())} windowLabel="30 days" />);
  const card = screen.getByTestId("usage-annotations");
  assert.match(card.textContent ?? "", /Coverage 40% \(4 of 10 runs observed\) · pack default 7 · last 30 days/);
  const rows = screen.getAllByTestId("usage-annotation-label");
  const [proven, partial] = rows;
  assert.match(proven?.textContent ?? "", /work\.blocked_by_tool.*proven.*Proven for winnow:e4b · decide\.local\.accurate · calibrated \(T 1\.4\).*hold-out precision 0\.90 · recall 0\.80 · ECE 0\.042 · gold 44 \(9 positive\).*4\/4.*50\.0%/s);
  assert.match(partial?.textContent ?? "", /partial.*2 of 6 answers are from a proven model/s);
  assert.doesNotMatch(partial?.textContent ?? "", /none of the observed answers/, "a partial label has proven answers, so no 'none came from it' reason");
  assert.match(within(partial as HTMLElement).getByTestId("usage-annotation-models").textContent ?? "", /laya:en · decide\.local: 4 observed, unmeasured/);
});

test("an unproven label says so plainly, gives its reason and never a rate", () => {
  renderWithProviders(<Annotations data={parseGate(gateRows())} windowLabel="30 days" />);
  const slow = screen.getAllByTestId("usage-annotation-label").find((row) => /agent\.discovery_slow/.test(row.textContent ?? ""));
  assert.ok(slow);
  const text = slow.textContent ?? "";
  assert.match(text, /unproven/);
  assert.match(text, /rule was measured on its tuning rows only: no hold-out yet/);
  assert.doesNotMatch(text, /\d+(\.\d+)?%/, "an unproven label carries no percentage");
});

test("the friction card lists friction labels only and counts silent ones as unknown", () => {
  renderWithProviders(<Annotations data={parseGate(gateRows())} windowLabel="30 days" />);
  const card = screen.getByTestId("usage-annotations");
  assert.equal(screen.getAllByTestId("usage-annotation-label").length, 3);
  assert.equal(/operator\.dissatisfied\b(?!_run)|run\.type/.test(card.textContent ?? ""), false, "mood and facet labels are not friction labels");
  assert.match(card.textContent ?? "", /1 label\(s\) have no observation in this window \(unknown, not zero\)/);
  assert.match(card.textContent ?? "", /An unproven label is evidence, not a rate/);
});

test("an empty annotation window reads as unknown, not zero friction", () => {
  renderWithProviders(<Annotations data={parseGate([{ kind: "coverage", pack: "default", pack_version: "1", eligible_runs: "0", observed_runs: "0", coverage: "0" }])} windowLabel="30 days" />);
  const card = screen.getByTestId("usage-annotations");
  assert.match(within(card).getByRole("status").textContent ?? "", /Unknown, not zero friction/);
  assert.match(card.textContent ?? "", /Coverage 0%/);
});
