import { describe, it, expect, vi, beforeEach } from "vitest";

const createClipRpc = vi.fn();
const listClipsRpc = vi.fn();
const deleteClipRpc = vi.fn();
const getClipAudioRpc = vi.fn();

vi.mock("../api/client", () => ({ transport: {} }));
vi.mock("@connectrpc/connect", () => ({
  createClient: () => ({
    createClip: (req: unknown) => createClipRpc(req),
    listClips: (req: unknown) => listClipsRpc(req),
    deleteClip: (req: unknown) => deleteClipRpc(req),
    getClipAudio: (req: unknown) => getClipAudioRpc(req),
  }),
}));

import {
  ClipSource,
  createClip,
  listClips,
  deleteClip,
  getClipAudio,
  decodeEvalReport,
} from "./corpus";

const protoClip = {
  id: "c1",
  referenceText: "hello world",
  tags: ["domain:test"],
  durationMs: 1_500n,
  sampleRateHz: 16_000,
  format: "pcm_s16le",
  source: ClipSource.FREE_FORM,
  createdAt: { seconds: 1_700_000_000n, nanos: 0 },
};

beforeEach(() => {
  vi.clearAllMocks();
});

describe("createClip", () => {
  it("encodes the request and decodes the saved clip", async () => {
    createClipRpc.mockResolvedValue({ clip: protoClip });
    const meta = await createClip({
      audio: new Uint8Array([1, 2, 3]),
      referenceText: "hello world",
      tags: ["domain:test"],
      durationMs: 1_500,
      sampleRateHz: 16_000,
      format: "pcm_s16le",
      source: ClipSource.SCRIPTED,
    });
    const req = createClipRpc.mock.calls[0]![0];
    expect(req.audio).toEqual(new Uint8Array([1, 2, 3]));
    expect(req.durationMs).toBe(1_500n);
    expect(req.source).toBe(ClipSource.SCRIPTED);
    expect(meta.id).toBe("c1");
    expect(meta.durationMs).toBe(1_500);
    expect(meta.createdAt).toContain("T");
  });

  it("throws when the response omits the clip", async () => {
    createClipRpc.mockResolvedValue({});
    await expect(
      createClip({
        audio: new Uint8Array(),
        referenceText: "x",
        tags: [],
        durationMs: 0,
        sampleRateHz: 16_000,
        format: "pcm_s16le",
        source: ClipSource.FREE_FORM,
      }),
    ).rejects.toThrow();
  });
});

describe("listClips / deleteClip / getClipAudio", () => {
  it("lists and decodes clips with default args", async () => {
    listClipsRpc.mockResolvedValue({ clips: [protoClip] });
    const out = await listClips();
    expect(listClipsRpc.mock.calls[0]![0]).toEqual({ tagContains: "", limit: 0, offset: 0 });
    expect(out).toHaveLength(1);
    expect(out[0]!.referenceText).toBe("hello world");
  });

  it("passes through list filters", async () => {
    listClipsRpc.mockResolvedValue({ clips: [] });
    await listClips({ tagContains: "x", limit: 5, offset: 10 });
    expect(listClipsRpc.mock.calls[0]![0]).toEqual({ tagContains: "x", limit: 5, offset: 10 });
  });

  it("deletes by id", async () => {
    deleteClipRpc.mockResolvedValue({});
    await deleteClip("c1");
    expect(deleteClipRpc).toHaveBeenCalledWith({ id: "c1" });
  });

  it("returns audio bytes plus decoded metadata", async () => {
    getClipAudioRpc.mockResolvedValue({ audio: new Uint8Array([9]), clip: protoClip });
    const out = await getClipAudio("c1");
    expect(Array.from(out.audio)).toEqual([9]);
    expect(out.clip?.id).toBe("c1");
  });

  it("tolerates a missing clip on audio fetch", async () => {
    getClipAudioRpc.mockResolvedValue({ audio: new Uint8Array() });
    const out = await getClipAudio("c1");
    expect(out.clip).toBeNull();
  });
});

describe("decodeEvalReport", () => {
  it("decodes the report", () => {
    const report = {
      perStrategy: [
        {
          strategy: "overlap_agree",
          label: "",
          wer: 0.12,
          substitutions: 1,
          insertions: 0,
          deletions: 0,
          refWords: 10,
          whisperCalls: 3,
          whisperAudioSeconds: 4.5,
          rtf: 0.8,
          finalizationLatencyP50Ms: 120,
          finalizationLatencyP95Ms: 240,
          partialRevisions: 2,
          werDeltaVsWinner: 0.02,
          p95DeltaMsVsWinner: 100,
          callMultiplierVsWinner: 1.5,
          verdict: "tradeoff",
          reasons: ["Uses more calls."],
          warnings: [{ code: "higher_compute", message: "Compute is higher.", severity: "warning" }],
          perClip: [
            {
              clipId: "clip-1",
              reference: "Hello, world!",
              hypothesis: "hello word",
              wer: 0.5,
              whisperCalls: 1,
              whisperAudioSeconds: 1.5,
              rtf: 0.5,
              segmentCount: 1,
              partialRevisions: 1,
              finalizationLatencyP50Ms: 100,
              finalizationLatencyP95Ms: 200,
              error: "",
              substitutions: 1,
              insertions: 0,
              deletions: 0,
              refWords: 2,
              hypWords: 2,
              normalizedReference: "hello world",
              normalizedHypothesis: "hello word",
              editOperations: [
                { kind: "match", referenceToken: "hello", hypothesisToken: "hello", referenceIndex: 0, hypothesisIndex: 0 },
                { kind: "substitution", referenceToken: "world", hypothesisToken: "word", referenceIndex: 1, hypothesisIndex: 1 },
              ],
            },
          ],
        },
      ],
      qualityMeasured: true,
      latencyMeasured: true,
      summary: {
        winnerStrategy: "batch",
        winnerLabel: "batch",
        recommendation: "Prefer batch.",
        confidence: "low",
        reasons: ["Best WER."],
        confidenceNotes: ["Tiny corpus."],
      },
      warnings: [{ code: "tiny_corpus", message: "Only 1 clip.", severity: "warning" }],
      normalizationPolicy: {
        werPolicy: "WER policy.",
        overlapAgreementPolicy: "Agreement policy.",
      },
    } as Parameters<typeof decodeEvalReport>[0];
    const out = decodeEvalReport(report);
    expect(out.perStrategy[0]!.label).toBe("overlap_agree");
    expect(out.perStrategy[0]!.perClip[0]!.normalizedReference).toBe("hello world");
    expect(out.perStrategy[0]!.perClip[0]!.editOperations[1]!.kind).toBe("substitution");
    expect(out.perStrategy[0]!.verdict).toBe("tradeoff");
    expect(out.summary?.recommendation).toBe("Prefer batch.");
    expect(out.warnings[0]!.code).toBe("tiny_corpus");
    expect(out.normalizationPolicy?.werPolicy).toBe("WER policy.");
    expect(out.qualityMeasured).toBe(true);
  });

  it("returns empty defaults when the report is absent", () => {
    const out = decodeEvalReport();
    expect(out.perStrategy).toEqual([]);
    expect(out.qualityMeasured).toBe(false);
    expect(out.latencyMeasured).toBe(false);
    expect(out.summary).toBeNull();
    expect(out.warnings).toEqual([]);
    expect(out.normalizationPolicy).toBeNull();
  });
});
