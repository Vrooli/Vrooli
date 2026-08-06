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
      catalog: "/",
      assetCatalog: "/catalog",
      asset: "/assets/:id",
      coverage: "/coverage",
      capabilities: "/capabilities",
      settings: "/settings",
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

  it("normalizes tab state and never leaks a report outside the tests tab", () => {
    expect(assetInfoTab(new URLSearchParams("tab=tests"))).toBe("tests");
    expect(assetInfoTab(new URLSearchParams("tab=unknown"))).toBe("preview");
    expect(assetSearchForTab("overview", "ctr_1")).toBe("?tab=overview");
    expect(assetSearchForTab("versions", "ctr_1")).toBe("?tab=versions");
    expect(assetStory(new URLSearchParams("story=loading"))).toBe("loading");
  });
});
