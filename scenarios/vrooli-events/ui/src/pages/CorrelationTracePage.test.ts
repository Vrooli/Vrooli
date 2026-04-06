// @vitest-environment node
// [REQ:REQ-UI-005] Correlation trace view — timeline, search, node rendering
import { describe, it, expect } from "vitest";
import { selectors } from "../consts/selectors";

describe("CorrelationTracePage selectors", () => {
  const trace = selectors.correlationTrace;

  it("has a page container selector", () => {
    expect(trace.page).toBe("correlation-trace-page");
  });

  it("has a correlation ID input selector", () => {
    expect(trace.correlationInput).toBe("trace-correlation-input");
  });

  it("has a search button selector", () => {
    expect(trace.searchButton).toBe("trace-search-button");
  });

  it("has a timeline container selector", () => {
    expect(trace.timeline).toBe("trace-timeline");
  });
});
