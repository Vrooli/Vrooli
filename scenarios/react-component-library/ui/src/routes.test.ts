import { describe, expect, it } from "vitest";

import {
  appRoutes,
  assetInfoTab,
  assetPath,
  assetSearchForTab,
  assetStory,
  assetTestReportPath,
  designPath,
  libraryPath,
  previewPath,
  workspacePath,
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
      workspaceCapture: "/workspace-capture/:id",
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

  it("builds optional query and design route segments only when provided", () => {
    expect(libraryPath()).toBe("/library");
    expect(libraryPath("  buttons & forms  ")).toBe("/library?q=buttons%20%26%20forms");
    expect(previewPath("asset 42")).toBe("/assets/asset%2042/preview");
    expect(previewPath("asset 42", "loading story")).toBe(
      "/assets/asset%2042/preview?story=loading+story",
    );
    expect(designPath()).toBe("/design");
    expect(designPath("shop")).toBe("/design/shop");
    expect(designPath("shop/name", "home page")).toBe("/design/shop%2Fname/home%20page");
    expect(workspacePath("ws 1", { page: "home", display: "mobile", state: "empty" })).toBe(
      "/design?workspace=ws+1&page=home&display=mobile&state=empty",
    );
    expect(workspacePath("ws 1")).toBe("/design?workspace=ws+1");
    expect(assetPath("asset 42", { tab: "files", story: "details" })).toBe(
      "/assets/asset%2042?tab=files&story=details",
    );
    expect(assetStory(new URLSearchParams("story=%20%20"))).toBeUndefined();
    expect(assetSearchForTab("preview")).toBe("");
    expect(assetSearchForTab("files", undefined, "details")).toBe("?tab=files&story=details");
  });
});
