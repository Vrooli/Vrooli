import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DuplexAudio, VoicePcmEncoder, VoicePcmPlayback } from "./duplexAudio";
import { getActiveMicLeases, _resetMicOwnershipForTesting } from "./voice/micOwnership";
import { selectStaleLeases } from "./voice/micLifecyclePolicy";

const mocks = vi.hoisted(() => ({ capture: vi.fn(), context: vi.fn(), idle: vi.fn() }));
vi.mock("./pcmCapture", () => ({ createCanonicalPcmCapture: mocks.capture }));
vi.mock("./voice/sharedAudioContext", () => ({ ensureRunningSharedAudioContext: mocks.context, holdSharedAudioContext: () => mocks.idle }));

function callbacks() { return { onAudio: vi.fn(), onLevel: vi.fn(), onConnected: vi.fn(), onError: vi.fn() }; }
function microphone() {
  const track = { enabled: true, readyState: "live", stop: vi.fn(), addEventListener: vi.fn(), removeEventListener: vi.fn() };
  const stream = { getTracks: () => [track], getAudioTracks: () => [track] } as unknown as MediaStream;
  return { track, stream };
}

beforeEach(() => {
  _resetMicOwnershipForTesting(); vi.clearAllMocks();
  mocks.context.mockResolvedValue({ state: "running" }); mocks.capture.mockResolvedValue({ stop: vi.fn() });
});

afterEach(() => { vi.restoreAllMocks(); vi.unstubAllGlobals(); });

describe("native duplex media", () => {
  it("keeps an independent duplex lease when dictation is idle", () => {
    expect(selectStaleLeases({ leases: [{ id: "native", owner: "agent-voice" }, { id: "dictation", owner: "voice-stream" }], voiceState: "idle", passiveListenerActive: false })).toEqual([{ id: "dictation", owner: "voice-stream" }]);
  });
  it("resamples continuously across callback boundaries", () => {
    const frame = Float32Array.from({ length: 1000 }, (_, i) => Math.sin(i / 20));
    const whole = new VoicePcmEncoder().encode(frame, 48_000);
    const encoder = new VoicePcmEncoder();
    const a = encoder.encode(frame.slice(0, 333), 48_000);
    const b = encoder.encode(frame.slice(333), 48_000);
    expect(atob(a.data) + atob(b.data)).toBe(atob(whole.data));
    expect(whole.sampleRate).toBe(24_000); expect(whole.numChannels).toBe(1);
  });
  it("rejects unsupported capture formats", () => { expect(() => new VoicePcmEncoder().encode(new Float32Array(10), 16_000)).toThrow("Unsupported"); });
  it("releases a microphone that arrives after cancellation", async () => {
    const mic = microphone(); let resolve!: (stream: MediaStream) => void;
    vi.stubGlobal("navigator", { mediaDevices: { getUserMedia: () => new Promise<MediaStream>((r) => { resolve = r; }) } });
    const media = new DuplexAudio(callbacks()); const pending = media.start("websocket"); media.stop(); resolve(mic.stream); await pending;
    expect(mic.track.stop).toHaveBeenCalledOnce(); expect(getActiveMicLeases()).toHaveLength(0); expect(mocks.capture).not.toHaveBeenCalled();
  });
  it("gates audio on connection and mute, then releases capture once", async () => {
    const mic = microphone(); const cb = callbacks();
    vi.stubGlobal("navigator", { mediaDevices: { getUserMedia: vi.fn().mockResolvedValue(mic.stream) } });
    const stop = vi.fn(); mocks.capture.mockResolvedValue({ stop });
    const media = new DuplexAudio(cb); await media.start("websocket");
    const frame = mocks.capture.mock.calls[0]?.[2] as (audio: Float32Array, rate: number) => void;
    frame(new Float32Array(512).fill(0.2), 48_000); expect(cb.onAudio).not.toHaveBeenCalled();
    media.ready(); frame(new Float32Array(512).fill(0.2), 48_000); expect(cb.onAudio).toHaveBeenCalledOnce(); expect(cb.onConnected).toHaveBeenCalledOnce();
    media.mute(true); expect(mic.track.enabled).toBe(false); frame(new Float32Array(512), 48_000); expect(cb.onAudio).toHaveBeenCalledOnce();
    media.stop(); media.stop(); expect(stop).toHaveBeenCalledOnce(); expect(mic.track.stop).toHaveBeenCalledOnce(); expect(getActiveMicLeases()).toHaveLength(0);
  });
  it("propagates denied permission without holding a lease", async () => {
    vi.stubGlobal("navigator", { mediaDevices: { getUserMedia: vi.fn().mockRejectedValue(new Error("permission denied")) } });
    const media = new DuplexAudio(callbacks()); await expect(media.start("websocket")).rejects.toThrow("permission denied"); media.stop(); expect(getActiveMicLeases()).toHaveLength(0);
  });
  it("stops queued playback on interruption and refuses excessive latency", () => {
    const sources: { stop: ReturnType<typeof vi.fn>; disconnect: ReturnType<typeof vi.fn>; onended: (() => void) | null }[] = [];
    const context = { currentTime: 0, destination: {}, createBuffer: (_channels: number, size: number) => ({ duration: size / 24_000, getChannelData: () => new Float32Array(size) }), createBufferSource: () => { const source = { buffer: null, connect: vi.fn(), start: vi.fn(), stop: vi.fn(), disconnect: vi.fn(), onended: null }; sources.push(source); return source; } } as unknown as AudioContext;
    const playback = new VoicePcmPlayback(context);
    const chunk = { data: btoa("\0".repeat(24_000)), sampleRate: 24_000, numChannels: 1, samplesPerChannel: 12_000, itemId: null };
    for (let i = 0; i < 5; i++) playback.append(chunk);
    expect(() => playback.append(chunk)).toThrow("fell behind"); playback.stop(); expect(sources.every((s) => s.stop.mock.calls.length === 1)).toBe(true);
  });
});


function remotePlaybackHarness() {
  const mic = microphone();
  const peer = {
    ontrack: null as ((event: RTCTrackEvent) => void) | null,
    onconnectionstatechange: null,
    iceGatheringState: "complete", localDescription: { type: "offer", sdp: "offer" },
    addTrack: vi.fn(), createDataChannel: vi.fn(() => ({ onmessage: null })),
    createOffer: vi.fn().mockResolvedValue({ type: "offer", sdp: "offer" }),
    setLocalDescription: vi.fn().mockResolvedValue(undefined), close: vi.fn(),
  };
  const source = { connect: vi.fn(), disconnect: vi.fn() };
  const context = { state: "running", destination: {}, createMediaStreamSource: vi.fn(() => source) };
  mocks.context.mockResolvedValue(context);
  vi.stubGlobal("navigator", { mediaDevices: { getUserMedia: vi.fn().mockResolvedValue(mic.stream) } });
  vi.stubGlobal("RTCPeerConnection", vi.fn(function () { return peer; }));
  const play = vi.spyOn(HTMLMediaElement.prototype, "play").mockResolvedValue(undefined);
  const pause = vi.spyOn(HTMLMediaElement.prototype, "pause").mockImplementation(() => {});
  const cb = callbacks();
  const media = new DuplexAudio(cb);
  const receive = () => { peer.ontrack?.({ streams: [mic.stream], track: mic.track } as unknown as RTCTrackEvent); };
  return { media, receive, source, context, mic, play, pause, cb };
}

describe("remote WebRTC playback", () => {
  it("renders remote audio once through the unlocked context and releases it on stop", async () => {
    const h = remotePlaybackHarness(); await h.media.start("webrtc"); h.receive();
    const renderer = document.querySelector("audio")!;
    expect(renderer.hidden).toBe(true); expect(renderer.muted).toBe(true);
    expect(renderer.srcObject).toBe(h.mic.stream);
    expect(h.play).toHaveBeenCalledOnce();
    expect(h.source.connect).toHaveBeenCalledWith(h.context.destination);
    h.media.stop(); h.media.stop();
    expect(h.pause).toHaveBeenCalledOnce(); expect(renderer.srcObject).toBeNull();
    expect(document.querySelector("audio")).toBeNull(); expect(h.source.disconnect).toHaveBeenCalledOnce();
  });
  it("releases the previous remote renderer when the provider replaces a track", async () => {
    const h = remotePlaybackHarness(); await h.media.start("webrtc"); h.receive();
    const first = document.querySelector("audio")!; h.receive();
    expect(first.srcObject).toBeNull(); expect(first.isConnected).toBe(false);
    expect(document.querySelectorAll("audio")).toHaveLength(1);
    h.media.stop(); expect(document.querySelector("audio")).toBeNull();
  });
  it("ignores a late playback rejection from a replaced renderer", async () => {
    const h = remotePlaybackHarness(); let reject!: (error: Error) => void;
    h.play.mockImplementationOnce(() => new Promise<void>((_resolve, fail) => { reject = fail; }));
    await h.media.start("webrtc"); h.receive(); h.receive(); reject(new Error("old renderer stopped"));
    await Promise.resolve();
    expect(h.cb.onError).not.toHaveBeenCalled(); expect(document.querySelectorAll("audio")).toHaveLength(1);
    h.media.stop();
  });
  it("ends capture and reports a playback rejection instead of silently listening", async () => {
    const h = remotePlaybackHarness(); h.play.mockRejectedValue(new Error("playback denied"));
    await h.media.start("webrtc"); h.receive(); await Promise.resolve();
    expect(h.cb.onError).toHaveBeenCalledWith(expect.objectContaining({ message: expect.stringContaining("playback could not start") }));
    expect(h.mic.track.stop).toHaveBeenCalledOnce(); expect(getActiveMicLeases()).toHaveLength(0);
    expect(document.querySelector("audio")).toBeNull();
  });
});
