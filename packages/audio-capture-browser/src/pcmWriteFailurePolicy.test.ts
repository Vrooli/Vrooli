import { describe, expect, it } from "vitest";
import { classifyPcmWriteFailure } from "./pcmWriteFailurePolicy";

describe("classifyPcmWriteFailure", () => {
  it("reports sequence discontinuities with cursor metadata", () => {
    expect(classifyPcmWriteFailure("Error", "expected sequence 4 at sample 6400, got sequence 6 at sample 9600; contiguous sequence identities required")).toEqual({
      reason: "sequence_mismatch",
      sequenceDetail: ["4", "6400", "6", "9600"],
    });
  });

  it("classifies exhausted storage quota", () => {
    expect(classifyPcmWriteFailure("QuotaExceededError", "storage quota exceeded")).toEqual({
      reason: "quota_exhausted",
      sequenceDetail: null,
    });
  });

  it("recognizes transaction failures from either exception name or message", () => {
    expect(classifyPcmWriteFailure("AbortError", "write failed").reason).toBe("storage_transaction");
    expect(classifyPcmWriteFailure("Error", "IndexedDB transaction aborted").reason).toBe("storage_transaction");
  });

  it("uses a generic storage failure for unclassified errors", () => {
    expect(classifyPcmWriteFailure("Error", "disk write failed")).toEqual({ reason: "storage_write", sequenceDetail: null });
  });
});
