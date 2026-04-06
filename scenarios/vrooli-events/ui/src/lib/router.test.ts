// @vitest-environment node
// [REQ:REQ-UI-001] Route configuration coherence — single source of truth for navigation
import { describe, it, expect } from "vitest";
import { ROUTES, NAV_ITEMS, type Route } from "./router";

describe("ROUTES", () => {
  it("derives from NAV_ITEMS", () => {
    const navIds = NAV_ITEMS.map((item) => item.id);
    expect(ROUTES).toEqual(navIds);
  });

  it("has no duplicate route ids", () => {
    const unique = new Set(ROUTES);
    expect(unique.size).toBe(ROUTES.length);
  });

  it("includes all expected routes", () => {
    const expected: Route[] = ["stream", "analytics", "events", "settings", "scenarios", "traces", "policies", "circuit-breakers", "subscriptions", "compliance"];
    for (const route of expected) {
      expect(ROUTES).toContain(route);
    }
  });

  it("has exactly 10 routes", () => {
    expect(ROUTES.length).toBe(10);
  });
});

describe("NAV_ITEMS", () => {
  it("each item has an id, label, and icon", () => {
    for (const item of NAV_ITEMS) {
      expect(item.id).toBeTruthy();
      expect(item.label).toBeTruthy();
      expect(item.icon).toBeDefined();
    }
  });

  it("labels are non-empty and unique", () => {
    const labels = NAV_ITEMS.map((item) => item.label);
    for (const label of labels) {
      expect(label.length).toBeGreaterThan(0);
    }
    const unique = new Set(labels);
    expect(unique.size).toBe(labels.length);
  });

  it("id order matches ROUTES order", () => {
    const ids = NAV_ITEMS.map((item) => item.id);
    expect(ids).toEqual(ROUTES);
  });
});
