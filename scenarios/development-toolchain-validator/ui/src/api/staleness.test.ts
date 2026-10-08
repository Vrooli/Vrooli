import { describe, expect, it } from "vitest";

import { StaleKind, stalenessClient } from "./staleness";

describe("api/staleness", () => {
  it("exposes the StalenessService RPCs as client methods", () => {
    expect(typeof stalenessClient.listStale).toBe("function");
  });

  it("re-exports the StaleKind enum", () => {
    expect(StaleKind).toBeDefined();
  });
});
