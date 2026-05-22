import { describe, expect, it } from "vitest";

import { signalsClient } from "./signals";

describe("api/signals.signalsClient", () => {
  it("exposes every SignalsService RPC as a callable method", () => {
    const rpcs = ["scoreChunk", "explainVerdict", "listSignals"] as const;
    for (const rpc of rpcs) {
      expect(typeof signalsClient[rpc]).toBe("function");
    }
  });
});
