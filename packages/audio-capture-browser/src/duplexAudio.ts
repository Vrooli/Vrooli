import { createCanonicalPcmCapture, type PcmCapture } from "./pcmCapture";
import { floatTo16BitPCM } from "./pcm";
import { acquireMicStream, type MicLease } from "./voice/micOwnership";
import { holdSharedAudioContext, ensureRunningSharedAudioContext } from "./voice/sharedAudioContext";

export interface DuplexAudioChunk {
  data: string;
  sampleRate: number;
  numChannels: number;
  samplesPerChannel: number | null;
  itemId: string | null;
}

export interface DuplexAudioCallbacks {
  onAudio: (chunk: DuplexAudioChunk) => void;
  onLevel: (level: number) => void;
  onConnected: () => void;
  onError: (error: Error) => void;
}

/** Continuous resampling at the native voice wire rate, across callback edges. */
export class VoicePcmEncoder {
  private pending = new Float32Array(0);
  private position = 0;
  private rate = 0;

  encode(frame: Float32Array, rate: number): DuplexAudioChunk {
    if (!Number.isFinite(rate) || rate < 24_000) throw new Error("Unsupported voice capture rate");
    if (this.rate !== rate) { this.pending = new Float32Array(0); this.position = 0; this.rate = rate; }
    const joined = new Float32Array(this.pending.length + frame.length);
    joined.set(this.pending); joined.set(frame, this.pending.length);
    const samples: number[] = [];
    const ratio = rate / 24_000;
    while (this.position + 1 < joined.length) {
      const index = Math.floor(this.position);
      const fraction = this.position - index;
      samples.push((joined[index] ?? 0) * (1 - fraction) + (joined[index + 1] ?? 0) * fraction);
      this.position += ratio;
    }
    const consumed = Math.min(Math.floor(this.position), Math.max(0, joined.length - 1));
    this.pending = joined.slice(consumed); this.position -= consumed;
    const pcm = floatTo16BitPCM(Float32Array.from(samples));
    const bytes = new Uint8Array(pcm.length * 2);
    const view = new DataView(bytes.buffer);
    for (let i = 0; i < pcm.length; i++) view.setInt16(i * 2, pcm[i] ?? 0, true);
    let binary = "";
    for (const byte of bytes) binary += String.fromCharCode(byte);
    return { data: btoa(binary), sampleRate: 24_000, numChannels: 1, samplesPerChannel: pcm.length, itemId: null };
  }
}

/** Bounded PCM playback. Interruption stops queued speech immediately. */
export class VoicePcmPlayback {
  private next = 0;
  private sources = new Set<AudioBufferSourceNode>();
  constructor(private readonly context: AudioContext) {}
  append(chunk: DuplexAudioChunk): void {
    if (chunk.sampleRate !== 24_000 || chunk.numChannels !== 1 || chunk.data.length > 65_536) throw new Error("Invalid voice playback format");
    const bytes = atob(chunk.data);
    if (!bytes.length || bytes.length % 2 !== 0) throw new Error("Invalid voice playback samples");
    if (this.next - this.context.currentTime > 2) throw new Error("Voice playback fell behind");
    const buffer = this.context.createBuffer(1, bytes.length / 2, chunk.sampleRate);
    const samples = buffer.getChannelData(0);
    for (let i = 0; i < samples.length; i++) {
      let value = bytes.charCodeAt(i * 2) | (bytes.charCodeAt(i * 2 + 1) << 8);
      if (value >= 32_768) value -= 65_536;
      samples[i] = value / 32_768;
    }
    const source = this.context.createBufferSource(); source.buffer = buffer; source.connect(this.context.destination);
    this.sources.add(source);
    source.onended = () => { this.sources.delete(source); source.disconnect(); };
    const start = Math.max(this.context.currentTime, this.next);
    source.start(start); this.next = start + buffer.duration;
  }
  stop(): void {
    for (const source of this.sources) { source.onended = null; try { source.stop(); } catch { /* already ended */ } source.disconnect(); }
    this.sources.clear(); this.next = 0;
  }
}

/** Provider-neutral media only. The host owns agent signaling and transcripts. */
export class DuplexAudio {
  private lease: MicLease | null = null;
  private capture: PcmCapture | null = null;
  private peer: RTCPeerConnection | null = null;
  private remote: MediaStreamAudioSourceNode | null = null;
  private remoteRenderer: HTMLAudioElement | null = null;
  private playback: VoicePcmPlayback | null = null;
  private closed = false;
  private connected = false;
  private muted = false;
  private mode: "webrtc" | "websocket" = "webrtc";
  private releaseContext: (() => void) | null = null;

  constructor(private readonly callbacks: DuplexAudioCallbacks) {}

  async start(mode: "webrtc" | "websocket"): Promise<string | undefined> {
    if (this.closed) return;
    this.mode = mode;
    this.releaseContext = holdSharedAudioContext();
    // Called directly from the mic gesture, before awaiting device permission.
    const contextPromise = ensureRunningSharedAudioContext();
    void contextPromise.catch(() => {}); // permission may reject before this promise is awaited
    const lease = await acquireMicStream("agent-voice", { audio: { echoCancellation: true, noiseSuppression: true, autoGainControl: true } }, {
      onRelease: () => { if (!this.closed) this.fail(new Error("Microphone disconnected")); },
    });
    if (this.closed) { lease.release("owner-replaced"); return; }
    this.lease = lease;
    const context = await contextPromise;
    if (this.closed) return;
    if (context.state !== "running") throw new Error("Tap the microphone to enable browser audio");
    this.playback = new VoicePcmPlayback(context);
    const encoder = new VoicePcmEncoder();
    const capture = await createCanonicalPcmCapture(context, lease.stream, (frame, rate) => {
      if (this.closed) return;
      let energy = 0; for (const value of frame) energy += value * value;
      this.callbacks.onLevel(this.muted ? 0 : Math.min(1, Math.sqrt(energy / Math.max(1, frame.length)) * 4));
      if (mode === "websocket" && this.connected && !this.muted) {
        try {
          const audio = encoder.encode(frame, rate);
          if (audio.samplesPerChannel) this.callbacks.onAudio(audio);
        } catch (error) { this.fail(error instanceof Error ? error : new Error("Voice capture failed")); }
      }
    });
    if (this.closed) { capture.stop(); return; } this.capture = capture;
    if (mode === "websocket") return;
    const peer = new RTCPeerConnection(); this.peer = peer;
    for (const track of lease.stream.getAudioTracks()) peer.addTrack(track, lease.stream);
    // Codex's WebRTC transport expects the Realtime events data channel.
    const channel = peer.createDataChannel("oai-events");
    channel.onmessage = (event: MessageEvent<string>) => {
      try { const message: unknown = JSON.parse(event.data); if (typeof message === "object" && message !== null && "type" in message && message.type === "input_audio_buffer.speech_started") this.interrupt(); } catch { /* non-JSON provider telemetry */ }
    };
    peer.ontrack = (event) => {
      if (this.closed) return;
      this.stopRemotePlayback();
      try {
        const stream = event.streams[0] ?? new MediaStream([event.track]);
        // Chromium can deliver silence to Web Audio for a remote RTC stream
        // until a media element renders it (crbug.com/40094084). Keep this
        // renderer muted: the gesture-unlocked context is the only audible path.
        const renderer = document.createElement("audio");
        renderer.hidden = true;
        renderer.autoplay = true;
        renderer.muted = true;
        renderer.setAttribute("playsinline", "");
        renderer.srcObject = stream;
        this.remoteRenderer = renderer;
        document.body.append(renderer);
        this.remote = context.createMediaStreamSource(stream);
        this.remote.connect(context.destination);
        void renderer.play().catch(() => {
          if (this.remoteRenderer === renderer) this.fail(new Error("Agent audio playback could not start. Tap the microphone to retry."));
        });
      } catch {
        this.fail(new Error("Agent audio playback could not start. Tap the microphone to retry."));
      }
    };
    peer.onconnectionstatechange = () => {
      if (this.closed) return;
      if (peer.connectionState === "connected") this.ready();
      else if (peer.connectionState === "failed" || peer.connectionState === "disconnected") this.fail(new Error("Voice connection interrupted"));
    };
    await peer.setLocalDescription(await peer.createOffer());
    if (peer.iceGatheringState !== "complete") {
      await new Promise<void>((resolve, reject) => {
        const timer = setTimeout(() => { peer.removeEventListener("icegatheringstatechange", check); reject(new Error("Voice network negotiation timed out")); }, 8_000);
        const check = () => { if (peer.iceGatheringState === "complete" || this.closed) { clearTimeout(timer); peer.removeEventListener("icegatheringstatechange", check); resolve(); } };
        peer.addEventListener("icegatheringstatechange", check); check();
      });
    }
    if (this.closed) return;
    const sdp = peer.localDescription?.sdp;
    if (!sdp) throw new Error("Voice offer is missing");
    return sdp;
  }

  async acceptSDP(sdp: string): Promise<void> { if (!this.closed && this.peer) await this.peer.setRemoteDescription({ type: "answer", sdp }); }
  ready(): void { if (!this.closed && !this.connected) { this.connected = true; this.callbacks.onConnected(); } }
  acceptAudio(chunk: DuplexAudioChunk): void { if (!this.closed && this.mode === "websocket") this.playback?.append(chunk); }
  interrupt(): void { this.playback?.stop(); }
  mute(muted: boolean): void { this.muted = muted; for (const track of this.lease?.stream.getAudioTracks() ?? []) track.enabled = !muted; if (muted) this.callbacks.onLevel(0); }
  stop(): void {
    if (this.closed) return; this.closed = true; this.connected = false;
    this.capture?.stop(); this.capture = null;
    if (this.peer) { this.peer.onconnectionstatechange = null; this.peer.ontrack = null; this.peer.close(); this.peer = null; }
    this.stopRemotePlayback(); this.playback?.stop(); this.playback = null;
    this.lease?.release("manual-stop"); this.lease = null;
    this.callbacks.onLevel(0);
    this.releaseContext?.(); this.releaseContext = null;
  }
  private stopRemotePlayback(): void {
    this.remote?.disconnect(); this.remote = null;
    if (this.remoteRenderer) {
      this.remoteRenderer.pause();
      this.remoteRenderer.srcObject = null;
      this.remoteRenderer.remove();
      this.remoteRenderer = null;
    }
  }
  private fail(error: Error): void { if (!this.closed) { this.stop(); this.callbacks.onError(error); } }
}
