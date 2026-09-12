import { describe, expect, it } from "vitest";

import { reportClient } from "./report";

describe("api/report", () => {
  it("exposes the ReportService RPCs as client methods", () => {
    expect(typeof reportClient.getGoldenSummary).toBe("function");
    expect(typeof reportClient.getTupleHistory).toBe("function");
    expect(typeof reportClient.getCoverage).toBe("function");
  });
});
