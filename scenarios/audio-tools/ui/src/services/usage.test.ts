import { describe, it, expect, vi, beforeEach } from "vitest";

import { ProviderTier } from "@vrooli/proto-types/audio-tools/v1/common/common_pb";

const listRecentRpc = vi.fn();
const getSummaryRpc = vi.fn();

vi.mock("@connectrpc/connect", () => ({
  createClient: () => ({
    listRecent: (req: unknown) => listRecentRpc(req),
    getSummary: (req: unknown) => getSummaryRpc(req),
  }),
}));

import { listRecent, getSummary } from "./usage";

beforeEach(() => {
  vi.clearAllMocks();
});

describe("listRecent", () => {
  it("encodes the default window and decodes rows", async () => {
    listRecentRpc.mockResolvedValue({
      rows: [
        {
          operationId: "op1",
          emittedAt: { seconds: 1_700_000_000n, nanos: 0 },
          capability: "stt",
          operation: "transcribe",
          providerTier: ProviderTier.LOCAL,
          providerId: "whisper",
          modelId: "v3",
          latencyMs: 42,
          creditsCharged: 1,
          error: "",
          fallbackReason: "",
        },
      ],
    });
    const r = await listRecent();
    const req = listRecentRpc.mock.calls[0]![0];
    expect(req.sinceSeconds).toBe(BigInt(60 * 60 * 24));
    expect(req.limit).toBe(50);
    expect(r.ok).toBe(true);
    if (r.ok) {
      expect(r.data[0]!.operationId).toBe("op1");
      expect(r.data[0]!.providerTier).toBe("local");
      expect(r.data[0]!.emittedAt).toContain("T");
    }
  });

  it("passes through custom window and limit", async () => {
    listRecentRpc.mockResolvedValue({ rows: [] });
    await listRecent(120, 5);
    const req = listRecentRpc.mock.calls[0]![0];
    expect(req.sinceSeconds).toBe(120n);
    expect(req.limit).toBe(5);
  });

  it("decodes a row with an empty (unspecified) provider tier and no timestamp", async () => {
    listRecentRpc.mockResolvedValue({
      rows: [
        {
          operationId: "op2",
          emittedAt: undefined,
          capability: "tts",
          operation: "synthesize",
          providerTier: ProviderTier.UNSPECIFIED,
          providerId: "",
          modelId: "",
          latencyMs: 0,
          creditsCharged: 0,
          error: "boom",
          fallbackReason: "tier-down",
        },
      ],
    });
    const r = await listRecent();
    if (r.ok) {
      expect(r.data[0]!.providerTier).toBe("");
      expect(r.data[0]!.emittedAt).toBe("");
      expect(r.data[0]!.error).toBe("boom");
    }
  });

  it("maps connect errors", async () => {
    listRecentRpc.mockRejectedValue(new Error("down"));
    const r = await listRecent();
    expect(r.ok).toBe(false);
  });
});

describe("getSummary", () => {
  it("encodes the request and decodes a full summary", async () => {
    getSummaryRpc.mockResolvedValue({
      summary: {
        since: { seconds: 1_700_000_000n, nanos: 0 },
        until: { seconds: 1_700_003_600n, nanos: 0 },
        operationsTotal: 10n,
        creditsTotal: 25n,
        errorCount: 2n,
        distribution: [{ providerTier: ProviderTier.BYOK, providerId: "openai", count: 7n }],
        fallbackReasons: [{ reason: "rate-limit", count: 3n }],
      },
    });
    const r = await getSummary(3600, "stt");
    const req = getSummaryRpc.mock.calls[0]![0];
    expect(req.sinceSeconds).toBe(3600n);
    expect(req.capability).toBe("stt");
    expect(r.ok).toBe(true);
    if (r.ok) {
      expect(r.data.operationsTotal).toBe(10);
      expect(r.data.creditsTotal).toBe(25);
      expect(r.data.errorCount).toBe(2);
      expect(r.data.distribution[0]!.providerTier).toBe("byok");
      expect(r.data.distribution[0]!.count).toBe(7);
      expect(r.data.fallbackReasons[0]).toEqual({ reason: "rate-limit", count: 3 });
      expect(r.data.since).toContain("T");
    }
  });

  it("uses defaults and tolerates an absent summary", async () => {
    getSummaryRpc.mockResolvedValue({});
    const r = await getSummary();
    const req = getSummaryRpc.mock.calls[0]![0];
    expect(req.sinceSeconds).toBe(BigInt(60 * 60 * 24));
    expect(req.capability).toBe("");
    if (r.ok) {
      expect(r.data.operationsTotal).toBe(0);
      expect(r.data.distribution).toEqual([]);
      expect(r.data.fallbackReasons).toEqual([]);
      expect(r.data.since).toBe("");
    }
  });

  it("maps connect errors", async () => {
    getSummaryRpc.mockRejectedValue(new Error("x"));
    const r = await getSummary();
    expect(r.ok).toBe(false);
  });
});
