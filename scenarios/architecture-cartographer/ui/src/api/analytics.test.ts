import { describe, expect, it } from "vitest";

import { analyticsClient } from "./analytics";

describe("api/analytics.analyticsClient", () => {
  it("exposes every AnalyticsService RPC as a callable method", () => {
    const rpcs = ["listEvents", "getStats", "listPlacements", "recordOverride"] as const;
    for (const rpc of rpcs) {
      expect(typeof analyticsClient[rpc]).toBe("function");
    }
  });
});
