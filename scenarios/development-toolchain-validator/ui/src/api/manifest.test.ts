import { describe, expect, it } from "vitest";

import { ConvergenceTarget, manifestClient } from "./manifest";

describe("api/manifest", () => {
  it("exposes the ManifestService RPCs as client methods", () => {
    expect(typeof manifestClient.listManifests).toBe("function");
    expect(typeof manifestClient.getManifest).toBe("function");
    expect(typeof manifestClient.upsertManifest).toBe("function");
    expect(typeof manifestClient.clearStale).toBe("function");
  });

  it("re-exports the ConvergenceTarget enum", () => {
    expect(ConvergenceTarget.EMPTY_DIFF).toBeDefined();
    expect(ConvergenceTarget.NONE).toBeDefined();
  });
});
