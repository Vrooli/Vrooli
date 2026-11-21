/* eslint-disable @typescript-eslint/require-await, @typescript-eslint/no-extraneous-class */
import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  client: {
    transcribe: vi.fn(async () => ({ text: " transcribed " })),
    transcribeStream: vi.fn(),
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
vi.mock("@vrooli/audio-capture-browser", async () => {
  const { openConnectVoiceStream } = await import("../../../../../../packages/audio-capture-browser/src/connectVoiceTransport");
  return {
    PcmVoiceStreamProvider: class {},
    PassiveListener: class {},
    bytesToFeatures: mocks.bytesToFeatures,
    createAudioFilterChain: mocks.createAudioFilterChain,
    createWakeWordEngine: mocks.createWakeWordEngine,
    openConnectVoiceStream,
    playRecordingFaultCue: vi.fn(),
    playRecordingStartCue: vi.fn(),
    playRecordingStopCue: vi.fn(),
    registerVoiceTransport: mocks.registerVoiceTransport,
  };
});

import { createPortalSpeechRuntime, createPortalVoiceServices } from "./portalVoice";
import type { VoiceTransport } from "@vrooli/audio-capture-browser";

describe("portal voice services", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    mocks.client.transcribe.mockReset().mockResolvedValue({ text: " transcribed " });
    mocks.client.transcribeStream.mockReset();
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

    const transportRegistration = mocks.registerVoiceTransport.mock.calls.at(-1)?.[0] as unknown as VoiceTransport;
    await expect(transportRegistration.transcribeRetained({ type: "audio/mp3", arrayBuffer: async () => new ArrayBuffer(5) } as unknown as Blob, "fr"))
      .resolves.toBe(" transcribed ");
    expect(mocks.client.transcribe).toHaveBeenCalled();
  });

  it("streams PCM through the generated Connect bidi client and maps final events", async () => {
    const received: unknown[] = [];
    const requests: Array<{ payload: { case?: string; value?: Record<string, unknown> } }> = [];
    mocks.client.transcribeStream.mockImplementation((input: AsyncIterable<unknown>) => {
      void (async () => { for await (const request of input) requests.push(request as typeof requests[number]); })();
      return (async function* () {
        yield { event: { case: "partial", value: { text: "hello" } } };
        yield { event: { case: "segment", value: { text: "hello", segmentId: "stable-segment", speakerPolicy: "applied", startSample: 0n, endSample: 1600n } } };
        yield { event: { case: "acknowledgement", value: { processedSequence: 0n } } };
        yield { event: { case: "done", value: { finalText: "hello", providerId: "local", modelId: "tiny" } } };
      })();
    });
    const voice = createPortalVoiceServices("https://audio.example");
    const registered = mocks.registerVoiceTransport.mock.calls.at(-1)?.[0] as unknown as VoiceTransport;
    const ready = vi.fn();
    const closed = vi.fn();
    const connection = registered.openStream({ language: "en", sessionId: "s1", resumeToken: "r1", protocolVersion: 2, generation: 3n, inputFormat: "pcm_s16le", sampleRateHz: 16000 }, {
      onReady: ready,
      onEvent: (event) => received.push(event),
      onClose: closed,
    });
    connection.sendAudio({ sequence: 0n, startSample: 0n, endSample: 1600n, audio: new Uint8Array([1, 2]), sha256: new Uint8Array([3]) });
    connection.end();

    await vi.waitFor(() => expect(requests).toHaveLength(3));
    await vi.waitFor(() => expect(received).toHaveLength(4));
    expect(ready).toHaveBeenCalledOnce();
    expect(requests.map((request) => request.payload.case)).toEqual(["start", "audioChunk", "end"]);
    expect(requests[0]?.payload.value).toMatchObject({
      language: "en", protocolVersion: 2, sessionId: "s1", resumeToken: "r1", generation: 3n, inputSampleRateHz: 16000,
    });
    expect(requests[1]?.payload.value).toMatchObject({ sequence: 0n, startSample: 0n, endSample: 1600n, audio: new Uint8Array([1, 2]) });
    expect(received).toEqual([
      { type: "partial", text: "hello" },
      { type: "segment-final", text: "hello", segmentId: "stable-segment", segmentIndex: 0, speakerPolicy: "applied" },
      { type: "status", code: "processed_acknowledgement", processedSequence: 0 },
      { type: "final", text: "hello", providerId: "local", modelId: "tiny" },
    ]);
    await vi.waitFor(() => expect(closed).toHaveBeenCalledOnce());
    expect(mocks.client.transcribeStream).toHaveBeenCalledOnce();
    expect(voice.services.PcmVoiceStreamProvider).toBeDefined();
  });

  it("keeps segment ordinals stable when a reconnect replays committed IDs", async () => {
    let call = 0;
    mocks.client.transcribeStream.mockImplementation(() => {
      call += 1;
      if (call === 1) {
        return (async function* () {
          yield { event: { case: "segment", value: { text: "first", segmentId: "seg-a", speakerPolicy: "", startSample: 0n, endSample: 1600n } } };
          throw new Error("stream interrupted");
        })();
      }
      return (async function* () {
        yield { event: { case: "segment", value: { text: "first", segmentId: "seg-a", speakerPolicy: "", startSample: 0n, endSample: 1600n } } };
        yield { event: { case: "segment", value: { text: "second", segmentId: "seg-b", speakerPolicy: "", startSample: 1600n, endSample: 3200n } } };
        yield { event: { case: "done", value: { finalText: "first second" } } };
      })();
    });
    createPortalVoiceServices("https://audio.example");
    const registered = mocks.registerVoiceTransport.mock.calls.at(-1)?.[0] as unknown as VoiceTransport;
    const onEvent = vi.fn();
    const makeStart = () => ({
      language: "en", sessionId: "replay-session", resumeToken: "resume", protocolVersion: 2 as const,
      generation: 0n, inputFormat: "pcm_s16le" as const, sampleRateHz: 16_000,
    });
    const firstClose = vi.fn();
    registered.openStream(makeStart(), { onReady: vi.fn(), onEvent, onClose: firstClose });
    await vi.waitFor(() => expect(firstClose).toHaveBeenCalledOnce());
    registered.openStream(makeStart(), { onReady: vi.fn(), onEvent, onClose: vi.fn() });
    await vi.waitFor(() => expect(onEvent).toHaveBeenCalledTimes(4));

    expect(onEvent.mock.calls.map(([event]) => event).filter((event) => event.type === "segment-final")).toEqual([
      { type: "segment-final", text: "first", segmentId: "seg-a", segmentIndex: 0, speakerPolicy: "" },
      { type: "segment-final", text: "first", segmentId: "seg-a", segmentIndex: 0, speakerPolicy: "" },
      { type: "segment-final", text: "second", segmentId: "seg-b", segmentIndex: 1, speakerPolicy: "" },
    ]);
  });
});
