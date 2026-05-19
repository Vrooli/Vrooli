import { describe, expect, it } from "vitest";

import { TupleKind, validationRecordClient, Verdict } from "./validationRecord";

describe("api/validationRecord", () => {
  it("exposes the ValidationRecordService RPCs as client methods", () => {
    expect(typeof validationRecordClient.listRecords).toBe("function");
    expect(typeof validationRecordClient.getRecord).toBe("function");
  });

  it("re-exports the TupleKind and Verdict enums", () => {
    expect(TupleKind.SKILL).toBeDefined();
    expect(TupleKind.TOOL).toBeDefined();
    expect(Verdict.PASS).toBeDefined();
    expect(Verdict.UNEXPECTED_MUTATION).toBeDefined();
  });
});
