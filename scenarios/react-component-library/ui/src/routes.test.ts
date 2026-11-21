import { describe, expect, it } from "vitest";

import {
  appRoutes,
  assetInfoTab,
  assetPath,
  assetSearchForTab,
  assetStory,
  assetTestReportPath,
} from "./routes";

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
    });
  });

  it("encodes asset identities and produces a stable test-evidence deep link", () => {
    expect(assetPath("react-component-library:useVoiceInput")).toBe(
      "/assets/react-component-library%3AuseVoiceInput",
    );
    expect(assetPath("asset 42", { tab: "preview", story: "loading" })).toBe(
      "/assets/asset%2042?story=loading",
    );
    expect(assetTestReportPath("asset 42", "ctr/a")).toBe(
      "/assets/asset%2042?tab=tests&testReport=ctr%2Fa",
    );
  });

  it("normalizes tab state and keeps removed legacy tabs out of the workspace", () => {
    expect(assetInfoTab(new URLSearchParams("tab=tests"))).toBe("tests");
    expect(assetInfoTab(new URLSearchParams("tab=unknown"))).toBe("preview");
    expect(assetSearchForTab("overview", "ctr_1")).toBe("?tab=tests&testReport=ctr_1");
    expect(assetSearchForTab("versions", "ctr_1")).toBe("?tab=versions");
    expect(assetInfoTab(new URLSearchParams("tab=experience"))).toBe("experience");
    expect(assetStory(new URLSearchParams("story=loading"))).toBe("loading");
  });
});
