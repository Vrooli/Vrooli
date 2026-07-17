import { describe, expect, it } from "vitest";

import { appRoutes, assetInfoTab, assetPath, assetSearchForTab, assetTestReportPath } from "./routes";

describe("workspace route contract", () => {
  it("owns the route patterns used by the application shell", () => {
    expect(appRoutes).toEqual({ catalog: "/", asset: "/assets/:id", settings: "/settings" });
  });

  it("encodes asset identities and produces a stable test-evidence deep link", () => {
    expect(assetPath("react-component-library:useVoiceInput")).toBe("/assets/react-component-library%3AuseVoiceInput");
    expect(assetTestReportPath("asset 42", "ctr/a")).toBe("/assets/asset%2042?tab=tests&testReport=ctr%2Fa");
  });

  it("normalizes tab state and never leaks a report outside the tests tab", () => {
    expect(assetInfoTab(new URLSearchParams("tab=tests"))).toBe("tests");
    expect(assetInfoTab(new URLSearchParams("tab=unknown"))).toBe("overview");
    expect(assetSearchForTab("overview", "ctr_1")).toBe("");
    expect(assetSearchForTab("versions", "ctr_1")).toBe("?tab=versions");
  });
});
