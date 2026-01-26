import { describe, it, expect } from "vitest";
import { getPageTitle, parseRouteFromHash, routeToHash } from "./routeController";

describe("routeController", () => {
  it("parses known routes from hashes", () => {
    expect(parseRouteFromHash("#/search")).toBe("search");
    expect(parseRouteFromHash("#/metrics")).toBe("metrics");
    expect(parseRouteFromHash("#/graph")).toBe("graph");
    expect(parseRouteFromHash("#/")).toBe("dashboard");
  });

  it("falls back to dashboard for unknown hashes", () => {
    expect(parseRouteFromHash("#/unknown")).toBe("dashboard");
    expect(parseRouteFromHash("")).toBe("dashboard");
  });

  it("maps routes to hashes and titles", () => {
    expect(routeToHash("search")).toBe("#/search");
    expect(getPageTitle("graph")).toBe("Knowledge Graph");
  });
});
