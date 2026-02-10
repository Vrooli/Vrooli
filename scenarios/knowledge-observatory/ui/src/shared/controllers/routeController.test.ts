import { describe, it, expect } from "vitest";
import { collectionToHash, getPageTitle, parseCollectionNameFromHash, parseRouteFromHash, routeToHash } from "./routeController";

describe("routeController", () => {
  it("parses known routes from hashes", () => {
    expect(parseRouteFromHash("#/search")).toBe("search");
    expect(parseRouteFromHash("#/explorer")).toBe("explorer");
    expect(parseRouteFromHash("#/viewer")).toBe("viewer");
    expect(parseRouteFromHash("#/metrics")).toBe("metrics");
    expect(parseRouteFromHash("#/graph")).toBe("graph");
    expect(parseRouteFromHash("#/collections/knowledge_chunks_v1")).toBe("collection");
    expect(parseRouteFromHash("#/")).toBe("dashboard");
  });

  it("falls back to dashboard for unknown hashes", () => {
    expect(parseRouteFromHash("#/unknown")).toBe("dashboard");
    expect(parseRouteFromHash("")).toBe("dashboard");
  });

  it("maps routes to hashes and titles", () => {
    expect(routeToHash("search")).toBe("#/search");
    expect(routeToHash("explorer")).toBe("#/explorer");
    expect(routeToHash("viewer")).toBe("#/viewer");
    expect(routeToHash("collection")).toBe("#/collections");
    expect(getPageTitle("graph")).toBe("Knowledge Graph");
    expect(getPageTitle("viewer")).toBe("Document Viewer");
    expect(getPageTitle("collection")).toBe("Collection Details");
  });

  it("parses and builds collection hashes", () => {
    expect(parseCollectionNameFromHash("#/collections/knowledge_chunks_v1")).toBe("knowledge_chunks_v1");
    expect(parseCollectionNameFromHash("#/metrics")).toBe("");
    expect(collectionToHash("knowledge chunks")).toBe("#/collections/knowledge%20chunks");
  });
});
