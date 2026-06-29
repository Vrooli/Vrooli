import { describe, it, expect, vi, beforeEach } from "vitest";

const createClipRpc = vi.fn();
const listClipsRpc = vi.fn();
const deleteClipRpc = vi.fn();
const getClipAudioRpc = vi.fn();
const runEvalRpc = vi.fn();

vi.mock("../api/client", () => ({ transport: {} }));
vi.mock("@connectrpc/connect", () => ({
  createClient: () => ({
    createClip: (req: unknown) => createClipRpc(req),
    listClips: (req: unknown) => listClipsRpc(req),
    deleteClip: (req: unknown) => deleteClipRpc(req),
    getClipAudio: (req: unknown) => getClipAudioRpc(req),
    runEval: (req: unknown) => runEvalRpc(req),
  }),
}));

import {
  ClipSource,
  createClip,
  listClips,
  deleteClip,
  getClipAudio,
  runEval,
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

describe("runEval", () => {
  it("defaults overlap stall rejects to -1 and decodes the report", async () => {
    runEvalRpc.mockResolvedValue({
      report: {
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
          },
        ],
        qualityMeasured: true,
        latencyMeasured: true,
      },
    });
    const out = await runEval({ strategies: [{ kind: "overlap_agree" }], realtimeRepeats: 3 });
    const req = runEvalRpc.mock.calls[0]![0];
    expect(req.strategies[0].overlapMaxStallRejects).toBe(-1);
    expect(req.realtimeRepeats).toBe(3);
    expect(out.perStrategy[0]!.label).toBe("overlap_agree");
    expect(out.qualityMeasured).toBe(true);
  });

  it("returns empty defaults when the report is absent", async () => {
    runEvalRpc.mockResolvedValue({});
    const out = await runEval();
    expect(out.perStrategy).toEqual([]);
    expect(out.qualityMeasured).toBe(false);
    expect(out.latencyMeasured).toBe(false);
  });
});
