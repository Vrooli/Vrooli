import { describe, expect, it } from "vitest";
import { contextLimitRecord } from "./usageMode";

describe("context limit record", () => {
  it("filters, deduplicates, and orders options against a verified window", () => {
    const record = contextLimitRecord({
      contextWindowTokens: 200_000n,
      contextWindowSource: "codex-models-cache/v1",
      contextLimitOptions: [300_000n, 150_000n, 150_000n, 0n, -1n, Number.POSITIVE_INFINITY],
    });
    expect(record.modelWindowTokens).toBe(200_000);
    expect(record.modelWindowSource).toBe("codex-models-cache/v1");
    expect(record.offeredCompactionPoints).toEqual([150_000]);
  });

  it("keeps server-admitted options when the window itself is unknown", () => {
    const record = contextLimitRecord({
      contextWindowTokens: 0,
      contextWindowSource: "stale-reader-source",
      contextLimitOptions: [150_000n, 1_000_000n, 300_000n, 300_000n],
    });
    expect(record.modelWindowTokens).toBeNull();
    expect(record.modelWindowSource).toBe("unknown");
    expect(record.offeredCompactionPoints).toEqual([1_000_000, 300_000, 150_000]);
  });

  it("does not carry stale provenance without a positive value", () => {
    const record = contextLimitRecord({
      contextWindowTokens: null,
      contextWindowSource: "codex-models-cache/v1",
      contextLimitTokens: null,
      contextLimitSource: "session launch descriptor",
    });
    expect(record.modelWindowSource).toBe("unknown");
    expect(record.compactionPointSource).toBe("unknown");
  });

  it("treats a positive window without verified provenance as unknown", () => {
    const oldConsumer = contextLimitRecord({
      contextWindowTokens: 900_000,
      contextLimitOptions: [300_000, 1_000_000],
    });
    const unverified = contextLimitRecord({
      contextWindowTokens: 900_000,
      contextWindowSource: "unknown",
      contextLimitOptions: [300_000, 1_000_000],
    });
    const retiredReader = contextLimitRecord({
      contextWindowTokens: 900_000,
      contextWindowSource: "codex-models-cache",
      contextLimitOptions: [300_000, 1_000_000],
    });
    for (const record of [oldConsumer, unverified, retiredReader]) {
      expect(record.modelWindowTokens).toBeNull();
      expect(record.modelWindowSource).toBe("unknown");
      expect(record.offeredCompactionPoints).toEqual([1_000_000, 300_000]);
    }
  });
});
