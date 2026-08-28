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
      // The preview popout is a real route: `routes.ts` declares it and
      // `App.tsx` renders `PreviewPopoutPage` at `appRoutes.preview`. This
      // contract test simply had not been updated when it was added.
      preview: "/assets/:id/preview",
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

  it("normalizes tab state and keeps removed legacy tabs out of the workspace", () => {
    expect(assetInfoTab(new URLSearchParams("tab=tests"))).toBe("preview");
    expect(assetInfoTab(new URLSearchParams("tab=unknown"))).toBe("preview");
    expect(assetSearchForTab("overview", "ctr_1")).toBe("?tab=tests&testReport=ctr_1");
    expect(assetSearchForTab("versions", "ctr_1")).toBe("?tab=versions");
    expect(assetInfoTab(new URLSearchParams("tab=experience"))).toBe("experience");
    expect(assetStory(new URLSearchParams("story=loading"))).toBe("loading");
  });
});
