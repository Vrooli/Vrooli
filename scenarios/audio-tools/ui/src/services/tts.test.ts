import { describe, it, expect, vi, beforeEach } from "vitest";

import { ProviderTier, ResponseFormat } from "@vrooli/proto-types/audio-tools/v1/common/common_pb";
import { ProviderState } from "@vrooli/proto-types/audio-tools/v1/shared/shared_pb";

const synthesizeRpc = vi.fn();
const listVoicesRpc = vi.fn();
const getStatusRpc = vi.fn();

vi.mock("@connectrpc/connect", () => ({
  createClient: () => ({
    synthesize: (req: unknown) => synthesizeRpc(req),
    listVoices: (req: unknown) => listVoicesRpc(req),
    getStatus: (req: unknown) => getStatusRpc(req),
  }),
}));

import { synthesize, listVoices, getStatus } from "./tts";

beforeEach(() => {
  vi.clearAllMocks();
});

describe("synthesize", () => {
  it("encodes defaults (speed 1.0, wav) and decodes the audio result", async () => {
    synthesizeRpc.mockResolvedValue({
      audio: new Uint8Array([7, 8]),
      contentType: "audio/wav",
      providerTier: ProviderTier.LOCAL,
      providerId: "kokoro",
      modelId: "k1",
      latencyMs: 30,
    });
    const r = await synthesize("hi", "alloy");
    const req = synthesizeRpc.mock.calls[0]![0];
    expect(req.speed).toBe(1.0);
    expect(req.responseFormat).toBe(ResponseFormat.WAV);
    expect(req.voice).toBe("alloy");
    expect(r.ok).toBe(true);
    if (r.ok) {
      expect(Array.from(r.data.audio)).toEqual([7, 8]);
      expect(r.data.providerTier).toBe("local");
      expect(r.data.contentType).toBe("audio/wav");
    }
  });

  it.each([
    ["mp3", ResponseFormat.MP3],
    ["wav", ResponseFormat.WAV],
    ["opus", ResponseFormat.OPUS],
    ["flac", ResponseFormat.FLAC],
    ["weird", ResponseFormat.UNSPECIFIED],
  ])("maps the %s response format", async (fmt, expected) => {
    synthesizeRpc.mockResolvedValue({
      audio: new Uint8Array(),
      contentType: "",
      providerTier: ProviderTier.UNSPECIFIED,
      providerId: "",
      modelId: "",
      latencyMs: 0,
    });
    const r = await synthesize("t", "v", 1.5, fmt);
    expect(synthesizeRpc.mock.calls[0]![0].responseFormat).toBe(expected);
    if (r.ok) expect(r.data.providerTier).toBe("");
  });

  it("maps connect errors", async () => {
    synthesizeRpc.mockRejectedValue(new Error("synth down"));
    const r = await synthesize("t", "v");
    expect(r.ok).toBe(false);
  });
});

describe("listVoices", () => {
  it("decodes the voice list", async () => {
    listVoicesRpc.mockResolvedValue({ voices: [{ id: "alloy", name: "Alloy" }] });
    const r = await listVoices();
    if (r.ok) expect(r.data).toEqual([{ id: "alloy", name: "Alloy" }]);
  });

  it("maps connect errors", async () => {
    listVoicesRpc.mockRejectedValue(new Error("x"));
    const r = await listVoices();
    expect(r.ok).toBe(false);
  });
});

describe("getStatus", () => {
  it("decodes availability rows with tier labels and AVAILABLE state", async () => {
    getStatusRpc.mockResolvedValue({
      status: {
        capability: "tts",
        capabilityLabel: "Text to Speech",
        availability: [
          { tier: ProviderTier.LOCAL, providerId: "kokoro", state: ProviderState.AVAILABLE },
          { tier: ProviderTier.BYOK, providerId: "openai", state: ProviderState.UNAVAILABLE },
        ],
      },
    });
    const r = await getStatus();
    expect(r.ok).toBe(true);
    if (r.ok) {
      expect(r.data.capability).toBe("tts");
      expect(r.data.availability).toEqual([
        { tier: "local", providerId: "kokoro", available: true },
        { tier: "byok", providerId: "openai", available: false },
      ]);
    }
  });

  it("throws (failure envelope) when status is missing", async () => {
    getStatusRpc.mockResolvedValue({});
    const r = await getStatus();
    expect(r.ok).toBe(false);
    if (!r.ok) expect(r.error.code).toBe("internal");
  });

  it("maps connect errors", async () => {
    getStatusRpc.mockRejectedValue(new Error("x"));
    const r = await getStatus();
    expect(r.ok).toBe(false);
  });
});
