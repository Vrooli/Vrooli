import { afterEach, describe, expect, it, vi } from "vitest";
import type { DuplexAudioCallbacks } from "./duplexAudio";
import { GPTLiveSessionClient, type GPTLiveMedia, type GPTLiveSocket, type GPTLiveSessionEvent } from "./gptLiveSession";

class FakeSocket implements GPTLiveSocket {
  readyState = 0;
  bufferedAmount = 0;
  onopen: ((event: Event) => void) | null = null;
  onmessage: ((event: MessageEvent) => void) | null = null;
  onerror: ((event: Event) => void) | null = null;
  onclose: ((event: CloseEvent) => void) | null = null;
  sent: string[] = [];
  close = vi.fn((_code?: number, _reason?: string) => { this.readyState = 3; });
  send = vi.fn((data: string) => { this.sent.push(data); });
  open(): void { this.readyState = 1; this.onopen?.(new Event("open")); }
  message(value: unknown): void { this.onmessage?.({ data: typeof value === "string" ? value : JSON.stringify(value) } as MessageEvent); }
  fail(): void { this.onerror?.(new Event("error")); }
}

class FakeMedia implements GPTLiveMedia {
  callbacks!: DuplexAudioCallbacks;
  start = vi.fn(async () => undefined);
  ready = vi.fn();
  acceptAudio = vi.fn();
  interrupt = vi.fn();
  mute = vi.fn();
  stop = vi.fn();
}

const request = {
  voice: "alloy", language: "en", model: "gpt-live-1.1", provider_id: "openai-gpt-live",
  remote_allowed: true, consent: true, consent_receipt: "consent-receipt", budget_micros: 100_000,
};

function harness(overrides: Partial<ConstructorParameters<typeof GPTLiveSessionClient>[0]> = {}) {
  const socket = new FakeSocket();
  const socketFactory = vi.fn(() => socket);
  const media = new FakeMedia();
  const onEvent = vi.fn();
  const onError = vi.fn();
  const client = new GPTLiveSessionClient({
    url: "ws://audio-tools.test/api/v1/duplex/sessions",
    socketFactory,
    mediaFactory: (callbacks) => { media.callbacks = callbacks; return media; },
    onEvent, onError, ...overrides,
  });
  return { client, socket, socketFactory, media, onEvent, onError };
}

async function admit(h: ReturnType<typeof harness>): Promise<void> {
  const started = h.client.start(request);
  h.socket.open();
  await waitUntil(() => h.socket.sent.length > 0);
  h.socket.message({ type: "ready" });
  await started;
}

async function waitUntil(predicate: () => boolean): Promise<void> {
  for (let i = 0; i < 10 && !predicate(); i++) await Promise.resolve();
}

afterEach(() => { vi.useRealTimers(); });

describe("GPTLiveSessionClient", () => {
  it("sends open and gates capture until explicit server admission ready", async () => {
    const h = harness();
    const started = h.client.start(request);
    expect(h.media.start).toHaveBeenCalledWith("websocket");
    h.socket.open();
    await waitUntil(() => h.socket.sent.length > 0);
    expect(JSON.parse(h.socket.sent[0] ?? "{}")).toEqual({ type: "open", ...request });
    h.media.callbacks.onAudio({ data: "AQI=", sampleRate: 24_000, numChannels: 1, samplesPerChannel: 2, itemId: null });
    expect(h.socket.sent).toHaveLength(1);
    h.socket.message({ type: "ready" });
    await started;
    expect(h.media.ready).toHaveBeenCalledOnce();
    h.media.callbacks.onAudio({ data: "AQI=", sampleRate: 24_000, numChannels: 1, samplesPerChannel: 2, itemId: null });
    expect(JSON.parse(h.socket.sent[1] ?? "{}")).toMatchObject({ type: "audio", audio_base64: "AQI=", format: "pcm_s16le", sample_rate: 24_000, sequence: 0 });
  });

  it("rejects provider events received before admission readiness", async () => {
    const h = harness();
    const started = h.client.start(request);
    const result = expect(started).rejects.toThrow("before admission");
    h.socket.open(); await waitUntil(() => h.socket.sent.length > 0);
    h.socket.message({ Type: "assistant_delta", AssistantDelta: { Text: "early" } });
    await result;
    expect(h.media.ready).not.toHaveBeenCalled();
    expect(h.media.stop).toHaveBeenCalledOnce();
  });

  it("maps server audio output and delivers transcript events", async () => {
    const h = harness(); await admit(h);
    const audioEvent = { EventID: "e1", Type: "audio_output", AudioOutput: { Bytes: "AQI=", Format: "pcm_s16le", SampleRate: 24_000, Sequence: 1 } } satisfies GPTLiveSessionEvent;
    h.socket.message(audioEvent);
    expect(h.media.acceptAudio).toHaveBeenCalledWith({ data: "AQI=", sampleRate: 24_000, numChannels: 1, samplesPerChannel: null, itemId: null });
    const transcript = { EventID: "e2", Type: "transcript_delta", TranscriptDelta: { Text: "hello" } } satisfies GPTLiveSessionEvent;
    h.socket.message(transcript);
    expect(h.onEvent).toHaveBeenCalledWith(transcript);
  });

  it("interrupts queued playback for VAD, server barge-in, and explicit barge-in", async () => {
    const h = harness(); await admit(h);
    h.socket.message({ EventID: "e1", Type: "vad", VAD: { State: "speech_start" } });
    h.socket.message({ EventID: "e2", Type: "barge_in_cancel", BargeInCancel: { Reason: "explicit" } });
    h.client.bargeIn();
    expect(h.media.interrupt).toHaveBeenCalledTimes(3);
    expect(JSON.parse(h.socket.sent.at(-1) ?? "{}")).toMatchObject({ type: "barge_in", reason: "explicit" });
  });

  it("fails closed on outbound audio backpressure and releases media/socket", async () => {
    const h = harness({ maxBufferedAudioBytes: 1 }); await admit(h);
    h.media.callbacks.onAudio({ data: "AQI=", sampleRate: 24_000, numChannels: 1, samplesPerChannel: 2, itemId: null });
    expect(h.onError).toHaveBeenCalledWith(expect.objectContaining({ message: expect.stringContaining("backpressure") }));
    expect(h.media.stop).toHaveBeenCalledOnce();
    expect(h.socket.close).toHaveBeenCalledOnce();
  });

  it("closes cleanly and releases resources on close event or provider error", async () => {
    const closed = harness(); await admit(closed);
    closed.socket.message({ EventID: "e1", Type: "closed", Closed: { Reason: "done" } });
    expect(closed.media.stop).toHaveBeenCalledOnce(); expect(closed.socket.close).toHaveBeenCalledOnce();

    const failed = harness(); await admit(failed);
    failed.socket.message({ EventID: "e2", Type: "provider_error", ProviderError: { Message: "provider failed" } });
    expect(failed.onError).toHaveBeenCalledWith(expect.objectContaining({ message: "provider failed" }));
    expect(failed.media.stop).toHaveBeenCalledOnce(); expect(failed.socket.close).toHaveBeenCalledOnce();
  });

  it("rejects server errors, oversized frames, and socket errors with cleanup", async () => {
    const denied = harness();
    const startDenied = denied.client.start(request);
    const deniedResult = expect(startDenied).rejects.toThrow("session denied");
    denied.socket.open(); await waitUntil(() => denied.socket.sent.length > 0);
    denied.socket.message({ type: "error", message: "session denied" });
    await deniedResult;
    expect(denied.media.stop).toHaveBeenCalledOnce();

    const oversized = harness({ maxInboundBytes: 40 }); await admit(oversized);
    oversized.socket.message({ Type: "assistant_delta", AssistantDelta: { Text: "too large" } });
    expect(oversized.onError).toHaveBeenCalledWith(expect.objectContaining({ message: expect.stringContaining("size limit") }));
    expect(oversized.media.stop).toHaveBeenCalledOnce();

    const socketError = harness(); const started = socketError.client.start(request); socketError.socket.fail();
    await expect(started).rejects.toThrow("WebSocket failed");
    expect(socketError.media.stop).toHaveBeenCalledOnce();
  });

  it("times out a socket that never opens and validates the session budget before setup", async () => {
    vi.useFakeTimers();
    const waiting = harness({ socketOpenTimeoutMs: 25 });
    const opening = waiting.client.start(request);
    const openResult = expect(opening).rejects.toThrow("WebSocket open timed out");
    await vi.advanceTimersByTimeAsync(26);
    await openResult;
    expect(waiting.media.stop).toHaveBeenCalledOnce();
    expect(waiting.socket.close).toHaveBeenCalledOnce();

    vi.useRealTimers();
    const invalid = harness();
    await expect(invalid.client.start({ ...request, budget_micros: Number.MAX_SAFE_INTEGER + 1 }))
      .rejects.toThrow("positive safe integer");
    await expect(invalid.client.start({ ...request, budget_micros: 0 }))
      .rejects.toThrow("positive safe integer");
    expect(invalid.media.start).not.toHaveBeenCalled();
    expect(invalid.socketFactory).not.toHaveBeenCalled();
    expect(invalid.socket.sent).toHaveLength(0);
  });

  it("bounds ready wait and supports manual mute/close", async () => {
    vi.useFakeTimers();
    const timed = harness({ readyTimeoutMs: 20 });
    const starting = timed.client.start(request);
    const timedResult = expect(starting).rejects.toThrow("admission timed out");
    timed.socket.open(); await waitUntil(() => timed.socket.sent.length > 0);
    await vi.advanceTimersByTimeAsync(21);
    await timedResult;
    expect(timed.media.stop).toHaveBeenCalledOnce();

    vi.useRealTimers();
    const h = harness(); await admit(h);
    h.client.mute(true); h.client.close();
    expect(h.media.mute).toHaveBeenCalledWith(true); expect(h.media.stop).toHaveBeenCalledOnce();
    expect(JSON.parse(h.socket.sent.at(-1) ?? "{}")).toEqual({ type: "close" });
    expect(h.socket.close).toHaveBeenCalledOnce();
  });
});
