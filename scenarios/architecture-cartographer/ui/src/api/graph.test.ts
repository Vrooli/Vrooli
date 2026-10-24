/**
 * Boundary test for api/graph: confirms the GraphService Connect client is
 * constructed and every RPC declared by the service is exposed as a callable
 * method. No business assertions — that lives in controller tests.
 */
import { describe, expect, it } from "vitest";

import { graphClient } from "./graph";

describe("api/graph.graphClient", () => {
  it("exposes every GraphService RPC as a callable method", () => {
    const rpcs = [
      "extractGraph",
      "getGraphSnapshot",
      "listGraphSnapshots",
      "clearGraphSnapshots",
      "exportGraph",
    ] as const;
    for (const rpc of rpcs) {
      expect(typeof graphClient[rpc]).toBe("function");
    }
  });
});
