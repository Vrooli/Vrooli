import { describe, expect, it } from "vitest";

import { conflictsClient } from "./conflicts";

describe("api/conflicts.conflictsClient", () => {
  it("exposes every ConflictsService RPC as a callable method", () => {
    const rpcs = [
      "detectConflicts",
      "listConflicts",
      "getConflict",
      "assignConflict",
      "resolveConflict",
      "reopenConflict",
      "validateConflicts",
      "listDetectors",
      "listResolvers",
    ] as const;
    for (const rpc of rpcs) {
      expect(typeof conflictsClient[rpc]).toBe("function");
    }
  });
});
