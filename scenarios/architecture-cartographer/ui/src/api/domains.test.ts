import { describe, expect, it } from "vitest";

import { domainsClient } from "./domains";

describe("api/domains.domainsClient", () => {
  it("exposes every DomainsService RPC as a callable method", () => {
    const rpcs = ["extractDomains", "getDomainMap", "convergenceReport"] as const;
    for (const rpc of rpcs) {
      expect(typeof domainsClient[rpc]).toBe("function");
    }
  });
});
