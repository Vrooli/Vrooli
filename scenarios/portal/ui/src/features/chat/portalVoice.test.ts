/* eslint-disable @typescript-eslint/require-await, @typescript-eslint/no-extraneous-class */
import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  client: {
    transcribe: vi.fn(async () => ({ text: " transcribed " })),
    synthesize: vi.fn(async () => ({ audio: new Uint8Array([1, 2]), contentType: "audio/custom" })),
    getStreamConfig: vi.fn(async () => ({ enabled: true })),
  },
  createClient: vi.fn(),
  createTransport: vi.fn(() => ({ kind: "transport" })),
  registerVoiceTransport: vi.fn(),
  bytesToFeatures: vi.fn(async () => ({ rms: 0 })),
  createAudioFilterChain: vi.fn(() => ({ process: vi.fn() })),
  createWakeWordEngine: vi.fn(() => ({ detect: vi.fn() })),
}));

mocks.createClient.mockReturnValue(mocks.client);

vi.mock("@connectrpc/connect", () => ({ createClient: mocks.createClient }));
vi.mock("@connectrpc/connect-web", () => ({ createConnectTransport: mocks.createTransport }));
vi.mock("@vrooli/audio-capture-browser", () => ({
  PcmVoiceStreamProvider: class {},
  PassiveListener: class {},
  bytesToFeatures: mocks.bytesToFeatures,
  createAudioFilterChain: mocks.createAudioFilterChain,
  createWakeWordEngine: mocks.createWakeWordEngine,
  playRecordingFaultCue: vi.fn(),
  playRecordingStartCue: vi.fn(),
  playRecordingStopCue: vi.fn(),
  registerVoiceTransport: mocks.registerVoiceTransport,
}));

import { createPortalSpeechRuntime, createPortalVoiceServices } from "./portalVoice";

describe("portal voice services", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    mocks.client.transcribe.mockReset().mockResolvedValue({ text: " transcribed " });
    mocks.client.synthesize.mockReset().mockResolvedValue({ audio: new Uint8Array([1, 2]), contentType: "audio/custom" });
    mocks.client.getStreamConfig.mockReset().mockResolvedValue({ enabled: true });
    mocks.createClient.mockReset().mockReturnValue(mocks.client);
    mocks.createTransport.mockClear();
    mocks.registerVoiceTransport.mockClear();
  });

  it("reports an unconfigured speech provider without making a request", async () => {
    const runtime = createPortalSpeechRuntime("");

    expect(runtime.enabled).toBe(false);
    await expect(runtime.capabilityCheck()).resolves.toBe(false);
    await expect(runtime.synthesize("hello")).rejects.toThrow("not configured");
    expect(mocks.client.synthesize).not.toHaveBeenCalled();
  });

  it("checks and synthesizes through a configured speech provider", async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, status: 200 });
    vi.stubGlobal("fetch", fetchMock);
    const runtime = createPortalSpeechRuntime("https://audio.example/");
    const signal = new AbortController().signal;

    expect(runtime.enabled).toBe(true);
    await expect(runtime.capabilityCheck()).resolves.toBe(true);
    const blob = await runtime.synthesize("hello", signal);

    expect(fetchMock).toHaveBeenCalledWith("https://audio.example/health", { cache: "no-store" });
    expect(mocks.client.synthesize).toHaveBeenCalledWith(expect.objectContaining({ text: "hello" }), { signal });
    expect(blob.type).toBe("audio/custom");
  });

  it("returns a diagnostic when speech health is rejected or unavailable", async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce({ ok: false, status: 503 })
      .mockRejectedValueOnce(new Error("offline"));
    vi.stubGlobal("fetch", fetchMock);
    const runtime = createPortalSpeechRuntime("http://audio.example");

    await expect(runtime.capabilityCheck()).resolves.toBe(false);
    await expect(runtime.capabilityCheck()).resolves.toBe(false);
  });

  it("keeps voice optional and exposes retained-transcription services", async () => {
    const disabled = createPortalVoiceServices("");
    await expect(disabled.capabilityCheck()).resolves.toEqual({
      whisperHealthy: false,
      streamingAvailable: false,
      capabilityReason: "Audio Tools is not configured.",
    });

    const fetchMock = vi.fn().mockResolvedValue({ ok: true, status: 200 });
    vi.stubGlobal("fetch", fetchMock);
    const voice = createPortalVoiceServices("https://audio.example/");
    const blobTypes = ["audio/wav", "audio/ogg", "audio/mpeg", "audio/flac", "audio/webm"];

    await expect(voice.capabilityCheck()).resolves.toEqual({ whisperHealthy: true, streamingAvailable: true });
    for (const type of blobTypes) {
      await voice.services.transcribeAudio({ type, arrayBuffer: async () => new ArrayBuffer(5) } as unknown as Blob);
    }
    await voice.services.transcribeAudioBypassFilter({ type: "audio/wav", arrayBuffer: async () => new ArrayBuffer(5) } as unknown as Blob);
    await voice.services.getVoiceStreamConfig();
    await voice.services.getWakeWordConfig();
    await voice.services.bytesToFeatures(new Uint8Array([1]), {} as never);
    voice.services.createAudioFilterChain({} as never, {} as never);
    voice.services.createWakeWordEngine();

    const transportRegistration = mocks.registerVoiceTransport.mock.calls.at(-1)?.[0] as {
      buildStreamUrl: (language: string, sessionID: string, resumeToken: string) => string;
      transcribeRetained: (audio: Blob, language: string) => Promise<string>;
    };
    expect(transportRegistration.buildStreamUrl("en", "session", "resume"))
      .toBe("wss://audio.example/api/v1/voice/stream?format=pcm_s16le&protocol_version=2&language=en&session_id=session&resume_token=resume");
    await expect(transportRegistration.transcribeRetained({ type: "audio/mp3", arrayBuffer: async () => new ArrayBuffer(5) } as unknown as Blob, "fr"))
      .resolves.toBe(" transcribed ");
    expect(mocks.client.transcribe).toHaveBeenCalled();
  });
});
