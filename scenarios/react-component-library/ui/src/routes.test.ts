import { describe, expect, it } from "vitest";

import { appRoutes, designPath, statsPath, statsView, workspacePath } from "./routes";

describe("workspace route contract", () => {
  it("owns the route patterns used by the application shell", () => {
    expect(appRoutes).toEqual({
      start: "/",
      library: "/library",
      components: "/components",
      assetCatalog: "/catalog",
      asset: "/assets/:id",
      preview: "/assets/:id/preview",
      stats: "/stats",
      coverage: "/coverage",
      capabilities: "/capabilities",
      activity: "/activity",
      review: "/review",
      settings: "/settings",
      create: "/new",
      design: "/design",
      designScenario: "/design/:scenario",
      designPage: "/design/:scenario/:page",
      legacySketch: "/sketch/:scenario/:page",
      workspaceCapture: "/workspace-capture/:id",
    });
  });

  it("builds optional design route segments only when provided", () => {
    expect(designPath()).toBe("/design");
    expect(designPath("shop")).toBe("/design/shop");
    expect(designPath("shop/name", "home page")).toBe("/design/shop%2Fname/home%20page");
    expect(workspacePath("ws 1", { page: "home", display: "mobile", state: "empty" })).toBe(
      "/design?workspace=ws+1&page=home&display=mobile&state=empty",
    );
    expect(workspacePath("ws 1")).toBe("/design?workspace=ws+1");
  });

  it("normalizes the Stats view", () => {
    expect(statsView(new URLSearchParams("view=quality"))).toBe("quality");
    expect(statsView(new URLSearchParams("view=unknown"))).toBe("overview");
    expect(statsPath()).toBe("/stats");
    expect(statsPath("activity")).toBe("/stats?view=activity");
  });
});
