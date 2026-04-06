// @vitest-environment node
// [REQ:REQ-UI-003] Per-scenario metrics — table, sorting, aggregation
import { describe, it, expect } from "vitest";
import { selectors } from "../consts/selectors";

describe("ScenarioMetricsPage selectors", () => {
  const metrics = selectors.scenarioMetrics;

  it("has a page container selector", () => {
    expect(metrics.page).toBe("scenario-metrics-page");
  });

  it("has a metrics table selector", () => {
    expect(metrics.table).toBe("scenario-metrics-table");
  });

  it("has sort button selectors for columns", () => {
    expect(metrics.sortOutbound).toBe("sort-outbound");
    expect(metrics.sortInbound).toBe("sort-inbound");
    expect(metrics.sortErrorRate).toBe("sort-errorRate");
  });
});
