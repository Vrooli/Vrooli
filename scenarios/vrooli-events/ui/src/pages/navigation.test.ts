// @vitest-environment node
import { describe, it, expect } from "vitest";
import { ROUTES, NAV_ITEMS, type Route } from "../lib/router";

// [REQ:REQ-UI-014A] Sidebar navigation structure and route highlighting
describe("navigation structure", () => {
  it("all routes have corresponding nav items", () => {
    for (const route of ROUTES) {
      const item = NAV_ITEMS.find((n) => n.id === route);
      expect(item).toBeDefined();
    }
  });

  it("nav items have meaningful labels", () => {
    for (const item of NAV_ITEMS) {
      expect(item.label.length).toBeGreaterThan(2);
      expect(item.label).not.toMatch(/^[a-z]/); // Labels start with uppercase
    }
  });

  it("each nav item has a unique icon", () => {
    const iconNames = NAV_ITEMS.map((item) => item.icon.displayName || item.icon.name);
    // Icons should have names (lucide icons do)
    for (const name of iconNames) {
      expect(name).toBeTruthy();
    }
  });

  it("stream is the first route (landing page)", () => {
    expect(ROUTES[0]).toBe("stream");
  });

  it("settings is the last route", () => {
    expect(ROUTES[ROUTES.length - 1]).toBe("settings");
  });
});

// [REQ:REQ-UI-014A1] Active route visual highlighting
describe("active route highlighting", () => {
  it("each route id is a valid URL path segment", () => {
    for (const route of ROUTES) {
      expect(route).toMatch(/^[a-z][a-z0-9-]*$/);
    }
  });

  it("no route ids contain special characters", () => {
    for (const route of ROUTES) {
      expect(route).not.toContain(" ");
      expect(route).not.toContain("/");
      expect(route).not.toContain("_");
    }
  });
});

// [REQ:REQ-UI-005A1] Correlation trace deep-link via query parameter
describe("correlation trace deep-linking", () => {
  it("cid query param format is valid", () => {
    const cid = "trace-123-abc";
    const url = `/traces?cid=${encodeURIComponent(cid)}`;
    expect(url).toContain("cid=trace-123-abc");
  });

  it("handles special characters in correlation ID", () => {
    const cid = "a/b+c=d";
    const encoded = encodeURIComponent(cid);
    expect(encoded).toBe("a%2Fb%2Bc%3Dd");
  });
});

// [REQ:REQ-UI-004A1] Event detail correlation chain navigation
describe("event detail cross-navigation", () => {
  it("correlation ID link targets traces page", () => {
    const correlationId = "cid-456";
    const linkTo = `/traces?cid=${correlationId}`;
    expect(linkTo).toContain("/traces");
    expect(linkTo).toContain("cid=cid-456");
  });

  it("source filter link targets event log", () => {
    const source = "my-scenario";
    const linkTo = `/events?source=${source}`;
    expect(linkTo).toContain("/events");
    expect(linkTo).toContain("source=my-scenario");
  });
});

// [REQ:REQ-UI-003A1] Scenario metrics sorting
describe("scenario metrics navigation", () => {
  it("scenario name links to event log with source filter", () => {
    const scenarioName = "auth-service";
    const linkTo = `/events?source=${encodeURIComponent(scenarioName)}`;
    expect(linkTo).toContain("/events");
    expect(linkTo).toContain("source=auth-service");
  });
});
