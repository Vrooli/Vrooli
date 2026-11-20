import assert from "node:assert/strict";
import { screen, within } from "@testing-library/react";
import { renderWithProviders } from "../../test-utils/index.js";
import { Annotations } from "./UsageFriction.js";
import type { AnnotationCoverage } from "./api.js";

function coverage(overrides: Partial<AnnotationCoverage> = {}): AnnotationCoverage {
  return {
    window: {},
    pack: { name: "default", version: "1" },
    eligible_runs: 10,
    observed_runs: 4,
    coverage: 0.4,
    labels: [
      {
        id: "agent.spinning", group: "agent", state: "observed", observed: 2, eligible: 4, rate: 0.5,
        gate: { state: "proven", precision: 0.9, recall: 0.8, gold: 40, min_gold: 40, target_precision: 0.8, target_recall: 0.7 },
        friction: { enabled: true },
      },
      {
        id: "operator.valence", group: "operator", state: "observed", observed: 3, eligible: 4, rate: null,
        gate: { state: "unmeasured", precision: 0, recall: 0, gold: 0, min_gold: 40, target_precision: 0.85, target_recall: 0.7 },
        friction: { enabled: true },
      },
      {
        id: "run.type", group: "run", state: "observed", observed: 4, eligible: 4,
        gate: { state: "unmeasured", precision: 0, recall: 0, gold: 0, min_gold: 100, target_precision: 0.95, target_recall: 0.95 },
        friction: { enabled: false },
      },
    ],
    ...overrides,
  };
}

test("the annotation card quotes a rate only for a proven gate and ignores non-friction facets", () => {
  renderWithProviders(<Annotations data={coverage()} />);
  const card = screen.getByTestId("usage-annotations");
  assert.match(card.textContent ?? "", /Coverage 40% \(4 of 10 runs observed\)/);
  const rows = screen.getAllByTestId("usage-annotation-label");
  assert.equal(rows.length, 2);
  assert.match(rows[0]?.textContent ?? "", /agent\.spinning.*proven.*2\/4.*50\.0%/i);
  assert.match(rows[1]?.textContent ?? "", /operator\.valence.*unmeasured.*3\/4.*—/i);
  assert.equal(card.textContent?.toLowerCase().includes("run.type"), false, "a non-friction facet is not a friction label");
  assert.match(card.textContent ?? "", /An unproven label is evidence, not a rate/);
});

test("an empty annotation window reads as unknown, not zero friction", () => {
  renderWithProviders(<Annotations data={coverage({ coverage: 0, observed_runs: 0, eligible_runs: 0, labels: [] })} />);
  const card = screen.getByTestId("usage-annotations");
  assert.match(within(card).getByRole("status").textContent ?? "", /Unknown, not zero friction/);
  assert.match(card.textContent ?? "", /Coverage 0%/);
});
