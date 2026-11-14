// @vitest-environment node
import { describe, expect, it } from "vitest";
import { selectors, selectorsManifest } from "./selectors";

// [REQ:REQ-UI-014] Navigation selectors exist for all routes
describe("selector manifest - nav selectors", () => {
  it("includes nav selectors for all pages", () => {
    expect(selectorsManifest.selectors["nav.stream"]).toEqual({
      testId: "nav-stream",
      selector: '[data-testid="nav-stream"]',
    });
    expect(selectorsManifest.selectors["nav.analytics"]).toEqual({
      testId: "nav-analytics",
      selector: '[data-testid="nav-analytics"]',
    });
    expect(selectorsManifest.selectors["nav.events"]).toEqual({
      testId: "nav-events",
      selector: '[data-testid="nav-events"]',
    });
    expect(selectorsManifest.selectors["nav.settings"]).toEqual({
      testId: "nav-settings",
      selector: '[data-testid="nav-settings"]',
    });
    expect(selectorsManifest.selectors["nav.healthIndicator"]).toEqual({
      testId: "nav-health-indicator",
      selector: '[data-testid="nav-health-indicator"]',
    });
  });
});

describe("dynamic selector parameter validation", () => {
  it("formats valid event row selectors and rejects invalid parameters", () => {
    expect(selectors.eventTable.rowByIndex({ index: 2 })).toBe('[data-testid="event-row-2"]');
    expect(selectors.eventTable.rowByEventId({ eventId: "evt-1" })).toContain("evt-1");
    expect(() => selectors.eventTable.rowByIndex(undefined as never)).toThrow(/missing parameter/);
    expect(() => selectors.eventTable.rowByIndex({ index: "2" as never })).toThrow(/must be numeric/);
    expect(() => selectors.eventTable.rowByIndex({ index: 2, extra: "x" } as never)).toThrow(/unknown parameter/);
  });
});

// [REQ:REQ-UI-001] Stream page selectors exist
describe("selector manifest - stream selectors", () => {
  it("includes stream page controls", () => {
    expect(selectorsManifest.selectors["stream.pauseButton"]).toBeDefined();
    expect(selectorsManifest.selectors["stream.clearButton"]).toBeDefined();
    expect(selectorsManifest.selectors["stream.typeFilter"]).toBeDefined();
    expect(selectorsManifest.selectors["stream.sourceFilter"]).toBeDefined();
    expect(selectorsManifest.selectors["stream.connectionStatus"]).toBeDefined();
  });
});

// [REQ:REQ-UI-002] Analytics selectors exist
describe("selector manifest - analytics selectors", () => {
  it("includes analytics stat selectors", () => {
    expect(selectorsManifest.selectors["analytics.totalEvents"]).toBeDefined();
    expect(selectorsManifest.selectors["analytics.storeSize"]).toBeDefined();
    expect(selectorsManifest.selectors["analytics.subscribers"]).toBeDefined();
  });
});

// [REQ:REQ-UI-014] Event log and detail selectors exist
describe("selector manifest - event log and detail", () => {
  it("includes event log selectors", () => {
    expect(selectorsManifest.selectors["eventLog.table"]).toBeDefined();
    expect(selectorsManifest.selectors["eventLog.refreshButton"]).toBeDefined();
  });

  it("includes event detail selectors", () => {
    expect(selectorsManifest.selectors["eventDetail.panel"]).toBeDefined();
    expect(selectorsManifest.selectors["eventDetail.eventId"]).toBeDefined();
    expect(selectorsManifest.selectors["eventDetail.payload"]).toBeDefined();
  });
});

// Scenario metrics and correlation trace selectors
describe("selector manifest - new page selectors", () => {
  it("includes scenario metrics page and table selectors", () => {
    expect(selectorsManifest.selectors["scenarioMetrics.page"]).toBeDefined();
    expect(selectorsManifest.selectors["scenarioMetrics.table"]).toBeDefined();
    expect(selectorsManifest.selectors["scenarioMetrics.sortOutbound"]).toBeDefined();
  });

  it("includes correlation trace page selectors", () => {
    expect(selectorsManifest.selectors["correlationTrace.page"]).toBeDefined();
    expect(selectorsManifest.selectors["correlationTrace.correlationInput"]).toBeDefined();
    expect(selectorsManifest.selectors["correlationTrace.timeline"]).toBeDefined();
  });
});

// [REQ:REQ-UI-014] Manifest completeness and structure
describe("selector manifest structure", () => {
  it("contains sufficient literal selectors", () => {
    const keys = Object.keys(selectorsManifest.selectors);
    expect(keys.length).toBeGreaterThan(15);
  });

  it("contains dynamic selectors with param definitions", () => {
    const rowByIndex = selectorsManifest.dynamicSelectors["eventTable.rowByIndex"];
    expect(rowByIndex).toBeDefined();
    if (rowByIndex) {
      expect(rowByIndex.params).toEqual([
        { name: "index", type: "number", values: undefined },
      ]);
    }
  });

  it("literal selectors have testId and selector fields", () => {
    const entry = selectorsManifest.selectors["nav.stream"];
    expect(entry).toHaveProperty("testId");
    expect(entry).toHaveProperty("selector");
  });
});
