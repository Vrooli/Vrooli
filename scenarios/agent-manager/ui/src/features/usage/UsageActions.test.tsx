import assert from "node:assert/strict";
import { screen } from "@testing-library/react";
import { create } from "@bufbuild/protobuf";
import { createElement } from "react";
import { test } from "vitest";
import { UsageActionSchema } from "@vrooli/proto-types/agent-manager/v1/measures/measures_pb";
import { ActionList, effectText, scoreText } from "./UsageActions.js";
import { FrictionDetail } from "./UsageFriction.js";
import { renderWithProviders } from "../../test-utils/index.js";

test("an unproven range headlines its top, never a point saving", () => {
  const action = create(UsageActionSchema, { title: "Try sonnet", ownerKind: "harness", owner: "claude-code sub-agent model", source: "lever", kind: "R-model-mix", confidence: "unproven", effect: { upper: 15, unit: "points", basis: "range" } });
  assert.equal(scoreText(action), "unproven");
  assert.equal(effectText(action.effect), "0 to 15 pts · unproven");
  renderWithProviders(createElement(ActionList, { actions: [action] }));
  assert.ok(screen.getByText("harness: claude-code sub-agent model"));
});

test("the score is the share of its own kind of harm", () => {
  assert.equal(scoreText(create(UsageActionSchema, { score: 0.214, effect: { value: 3373, unit: "calls", basis: "observed" } })), "21.4%");
});

test("effects read in their own unit: calls, tokens, minutes and finished savings", () => {
  assert.equal(effectText(create(UsageActionSchema, { effect: { value: 161, unit: "calls", basis: "observed" } }).effect), "161 calls · observed");
  assert.match(effectText(create(UsageActionSchema, { effect: { value: 12_000_000, unit: "tokens", basis: "observed" } }).effect), /tokens re-read · observed$/);
  assert.equal(effectText(create(UsageActionSchema, { effect: { value: 3.2, unit: "points", basis: "would have saved" } }).effect), "≈ 3.2 pts · would have saved");
});

test("a root-cause roll-up, guesses and a concentrated finding read legibly on the action", () => {
  const action = create(UsageActionSchema, {
    title: "Fix the identity root cause", ownerKind: "scenario", owner: "vrooli", source: "tool", kind: "failure", confidence: "high", rootCause: "identity",
    scoreBasis: "20.0%; 62% of it in one run, so the effect counts each run at most 10x the median run", topRunShare: 0.62,
    subs: [{ kind: "failure", signature: "identity broker did not answer", calls: 30n }, { kind: "degraded", signature: "identity_refresh_unavailable", calls: 10n }],
    guesses: [{ token: "runs", suggestion: "run", calls: 8n }, { token: "--tail", calls: 4n }],
    effect: { value: 40, unit: "calls", basis: "observed" },
  });
  renderWithProviders(createElement(ActionList, { actions: [action] }));
  assert.ok(screen.getByText(/rolls up 2 lines/));
  assert.ok(screen.getByText("identity_refresh_unavailable"));
  assert.equal(screen.getAllByText(/add alias/).length, 1);
  assert.ok(screen.getByText(/fix the help/));
  assert.ok(screen.getByTestId("usage-action-share").textContent?.includes("62% of it in one run"));
});

test("output weight states printed against read tokens and a slow command reads in minutes", () => {
  renderWithProviders(createElement(FrictionDetail, { rootCause: "", subs: [], guesses: [], printedTokens: 262_144n, readTokens: 10_000n }));
  assert.match(screen.getByTestId("usage-friction-detail").textContent ?? "", /printed ~262 K tokens; the model read 10 K/);
  assert.equal(effectText(create(UsageActionSchema, { effect: { value: 720_000, unit: "ms", basis: "observed" } }).effect), "12 min · observed");
});

test("quota levers and friction actions are headed as separate groups, never interleaved as one scale", () => {
  const lever = (title: string) => create(UsageActionSchema, { title, ownerKind: "harness", owner: "codex session context", source: "lever", kind: "R-context", score: 0.26, effect: { value: 13, unit: "points", basis: "projected" } });
  const tool = (title: string) => create(UsageActionSchema, { title, ownerKind: "command", owner: "vrooli scenario status", source: "tool", kind: "polling", score: 0.2, effect: { value: 200, unit: "calls", basis: "observed" } });
  renderWithProviders(createElement(ActionList, { actions: [lever("Compact codex"), lever("Restart sub-agents"), tool("Steer polling"), tool("Fix identity")] }));
  assert.deepEqual(screen.getAllByTestId("usage-action-group").map((heading) => heading.textContent), ["Quota levers · share of the window's points", "Friction actions · √(harm share × runs touched)"]);
});
