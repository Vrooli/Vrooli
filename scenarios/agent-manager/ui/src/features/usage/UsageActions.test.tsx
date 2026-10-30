import assert from "node:assert/strict";
import { screen } from "@testing-library/react";
import { create } from "@bufbuild/protobuf";
import { createElement } from "react";
import { test } from "vitest";
import { UsageRecommendationGroupSchema, type UsageLever } from "@vrooli/proto-types/agent-manager/v1/measures/measures_pb";
import { ActionGroups } from "./UsageActions.js";
import { renderWithProviders } from "../../test-utils/index.js";

function group(lever: Record<string, unknown>) {
  return create(UsageRecommendationGroupSchema, {
    rank: 1, targetKind: "session", targetId: "abcd1234", targetLabel: "Chat session", usedSoFarPoints: 40, atThisRatePoints: 80, recoverablePoints: 0, combinedSavedPoints: 0, stillRunning: false,
    levers: [{ savedPoints: 0, marginalPoints: 0, recoverablePoints: 0, action: "Look at it", rationale: "why", ...lever } as unknown as UsageLever],
  });
}

test("an unproven model move headlines its ceiling instead of saying nothing is left", () => {
  const g = group({ rule: "R-model-mix", lever: "model", confidence: "unproven", facts: { facts: { case: "model", value: { expensiveWeeklyPoints: 30, priceRatio: 0.5 } } } });
  renderWithProviders(createElement(ActionGroups, { groups: [g] }));
  assert.match(screen.getByTestId("usage-ceiling").textContent ?? "", /up to ≈ −15.*\(unproven\)/);
  assert.equal(screen.queryByText("Nothing left to save"), null);
  assert.equal(screen.queryByText("this target has stopped"), null);
});

test("a floor-only card headlines the floor's share of what the target used", () => {
  const g = group({ rule: "R-floor", lever: "context", confidence: "low", facts: { facts: { case: "context", value: { cachedShare: 0.25 } } } });
  renderWithProviders(createElement(ActionGroups, { groups: [g] }));
  assert.match(screen.getByTestId("usage-ceiling").textContent ?? "", /up to ≈ −10.*\(unproven\)/);
});

test("a proven lever on a stopped target still says nothing is left", () => {
  const g = group({ rule: "R-context", lever: "context", confidence: "high" });
  renderWithProviders(createElement(ActionGroups, { groups: [g] }));
  assert.ok(screen.getByText("Nothing left to save"));
  assert.equal(screen.queryByTestId("usage-ceiling"), null);
});
