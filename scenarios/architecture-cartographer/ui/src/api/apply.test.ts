import { describe, expect, it } from "vitest";

import { applyClient } from "./apply";

describe("api/apply.applyClient", () => {
  it("exposes every ApplyService RPC as a callable method", () => {
    const rpcs = ["planApply", "runApply", "listApplyHistory", "getBuildBaseline"] as const;
    for (const rpc of rpcs) {
      expect(typeof applyClient[rpc]).toBe("function");
    }
  });
});
