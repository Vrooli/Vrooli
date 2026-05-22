import { describe, expect, it } from "vitest";

import { manifestClient } from "./manifest";

describe("api/manifest.manifestClient", () => {
  it("exposes every ManifestService RPC as a callable method", () => {
    const rpcs = ["validateManifest", "getManifest", "listDomains"] as const;
    for (const rpc of rpcs) {
      expect(typeof manifestClient[rpc]).toBe("function");
    }
  });
});
