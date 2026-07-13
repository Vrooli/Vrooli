import { buildVoiceStreamWsUrl, transcribeAudioWithRetry } from "../../api/voice";
import {
  concatInt16,
  createCanonicalPcmCapture,
  dispatchStreamMessage,
  digestAudio,
  encodeAudioFrame,
  encodeWavFromPcm16,
  forgetUnfinishedSession,
  frameToCanonicalPcm16,
  IndexedDBTurnJournalStore,
  loadUnfinishedSession,
  MemoryTurnJournalStore,
  newSessionIdentity,
  rememberUnfinishedSession,
  TARGET_SAMPLE_RATE,
  TurnJournal,
  type PcmCapture,
} from "@vrooli/audio-capture-browser";
import { getSharedAudioContext } from "./sharedAudioContext";
import { computeFinalTimeout, WHISPER_FAILED_SENTINEL, type LastTurnAudio, type TranscriptionProvider } from "./types";

/**
 * Swarm Manager's thin host adapter for the governed replay-safe PCM-v2
 * transport. It owns only host mic lifecycle and UI callbacks; frame identity,
 * journaling, digesting, and recovery payloads live in audio-capture-browser.
 */
export class PcmVoiceStreamProvider implements TranscriptionProvider {
  private static readonly MAX_RETAINED_PCM_BYTES = 16 * 1024 * 1024;

  private ws: WebSocket | null = null;
  private capture: PcmCapture | null = null;
  private stream: MediaStream | null = null;
  private sessionId = "";
  private resumeToken = "";
  private wsURL = "";
  private journal: TurnJournal | null = null;
  private writes: Promise<void> = Promise.resolve();
  private pendingFrames: ArrayBuffer[] = [];
  private allPCM: Int16Array[] = [];
  private retainedPcmBytes = 0;
  private nextSequence = 0n;
  private nextSample = 0n;
  private reconnects = 0;
  private stopped = false;
  private doneSent = false;
  private tailDropArmed = false;
  private finalReceived = false;
  private finalTimer: ReturnType<typeof setTimeout> | null = null;
	// A final after an error is not evidence that the server covered every
	// captured frame. Preserve the replay journal for the host recovery UI.
  private terminalFailure = false;
  private startedAt = 0;
  private lastTurn: LastTurnAudio | null = null;
  private readonly deliveredSegmentIDs = new Set<string>();

  language = "en";
  retainStream = false;
  onResult: ((text: string) => void) | null = null;
  onError: ((error: string) => void) | null = null;
  onStatus: ((status: { code: string; message: string }) => void) | null = null;
  onPartial: ((text: string) => void) | null = null;
  onSegmentFinal: ((text: string, segmentIndex: number) => void) | null = null;
  onSegmentAccepted: ((segmentIndex: number, score: number, threshold: number) => void) | null = null;
  onSegmentRejected: ((segmentIndex: number, score: number, threshold: number) => void) | null = null;
  onSpeakerStatus: ((enabled: boolean, profileConfigured: boolean) => void) | null = null;
  onVadState: ((state: { voiced: boolean; silenceElapsedMs: number; silenceTimeoutMs: number; tickSeq: number; silenceTimedOut: boolean }) => void) | null = null;

  getStream(): MediaStream | null {
    return this.stream;
  }

  getLastTurnAudio(): LastTurnAudio | null {
    return this.lastTurn;
  }

  disposeLastTurn(): void {
    this.lastTurn = null;
  }

  private status(code: string, message: string): void {
    this.onStatus?.({ code, message });
  }

  private newIdentity(): boolean {
    try {
      this.sessionId = newSessionIdentity();
      this.resumeToken = newSessionIdentity();
      this.deliveredSegmentIDs.clear();
      rememberUnfinishedSession({ sessionId: this.sessionId, resumeToken: this.resumeToken });
      return true;
    } catch {
      this.onError?.("Secure browser identity generation is unavailable; dictation recovery cannot start.");
      return false;
    }
  }

  private async initializeJournal(): Promise<void> {
    const persistent = new TurnJournal(new IndexedDBTurnJournalStore(), this.sessionId, 0n, PcmVoiceStreamProvider.MAX_RETAINED_PCM_BYTES, "persistent");
    try {
      const snapshot = await persistent.restore();
      this.journal = persistent;
      this.nextSequence = snapshot.nextSequence;
      this.nextSample = snapshot.nextSample;
    } catch {
      this.journal = new TurnJournal(new MemoryTurnJournalStore(), this.sessionId, 0n, PcmVoiceStreamProvider.MAX_RETAINED_PCM_BYTES, "reduced");
      this.status("durability_reduced", "Persistent audio recovery is unavailable in this browser.");
    }
  }

  private queueFrame(samples: Int16Array): void {
    if (this.tailDropArmed) return;
    if (this.retainedPcmBytes + samples.byteLength > PcmVoiceStreamProvider.MAX_RETAINED_PCM_BYTES) {
      this.tailDropArmed = true;
      this.status("recovery_quota_exhausted", "Audio recovery storage reached its limit; this turn was stopped before further audio could be lost.");
      this.onError?.("Audio recovery storage reached its limit. Start a new turn to continue.");
      return;
    }

    const sequence = this.nextSequence++;
    const startSample = this.nextSample;
    const endSample = startSample + BigInt(samples.length);
    this.nextSample = endSample;
    this.retainedPcmBytes += samples.byteLength;
    this.allPCM.push(samples);

    const audio = new Uint8Array(samples.byteLength);
    audio.set(new Uint8Array(samples.buffer, samples.byteOffset, samples.byteLength));
    this.writes = this.writes
      .then(async () => {
        const sha256 = await digestAudio(audio.buffer);
        const frame = encodeAudioFrame({ sequence, startSample, endSample, audio: samples, sha256: new Uint8Array(sha256) });
        await this.journal?.append({ sequence, startSample, endSample, audio: audio.buffer, sha256 });
        if (this.ws?.readyState === WebSocket.OPEN && this.pendingFrames.length === 0) {
          this.ws.send(frame);
        } else {
          this.pendingFrames.push(frame);
        }
      })
      .catch(() => this.onError?.("Audio recovery storage failed; stop and retry this turn."));
  }

  private sendDone(ws: WebSocket): void {
    if (this.stopped && !this.doneSent && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type: "done" }));
      this.doneSent = true;
    }
  }

  private flush(ws: WebSocket, replay: boolean): void {
    void this.writes.then(() => {
      if (ws.readyState !== WebSocket.OPEN) return;
      if (replay) {
        this.pendingFrames = [];
        for (const chunk of this.journal?.replayAfter(-1n) ?? []) {
          ws.send(encodeAudioFrame({
            sequence: chunk.sequence,
            startSample: chunk.startSample,
            endSample: chunk.endSample,
            audio: new Uint8Array(chunk.audio),
            sha256: new Uint8Array(chunk.sha256),
          }));
        }
      } else {
        for (const frame of this.pendingFrames) ws.send(frame);
        this.pendingFrames = [];
      }
      this.sendDone(ws);
    });
  }

  private handleMessage(event: MessageEvent<string>): void {
    dispatchStreamMessage(event.data, {
      onStatus: (code, text, processedSequence) => {
        if (code === "processed_acknowledgement" && processedSequence !== undefined) {
          this.writes = this.writes.then(async () => this.journal?.acknowledgeProcessed(processedSequence) ?? undefined);
        }
        this.status(code, text);
      },
      onPartial: (text) => this.onPartial?.(text),
      onSegmentFinal: (text, index) => this.onSegmentFinal?.(text, index),
      onSegmentAccepted: (index, score, threshold) => this.onSegmentAccepted?.(index, score, threshold),
      onSegmentRejected: (index, score, threshold) => this.onSegmentRejected?.(index, score, threshold),
      onSpeakerStatus: (enabled, profileConfigured) => this.onSpeakerStatus?.(enabled, profileConfigured),
      onVadState: (state) => this.onVadState?.(state),
      onFinal: (text) => {
		if (this.terminalFailure) {
		  this.finalReceived = true;
		  if (this.nextSequence > 0n) {
			this.status("recovery_retained", "The backend did not confirm all captured audio. Retained audio is available for recovery.");
		  } else {
			this.writes = this.writes.then(async () => this.journal?.discard() ?? undefined);
			forgetUnfinishedSession();
		  }
		  return;
		}
        this.finalReceived = true;
        if (this.finalTimer) clearTimeout(this.finalTimer);
        this.onResult?.(text);
        this.writes = this.writes.then(async () => this.journal?.discard() ?? undefined);
        forgetUnfinishedSession();
      },
	  onError: (_code, text) => {
		this.terminalFailure = true;
		this.onError?.(text);
	  },
    }, this.deliveredSegmentIDs);
  }

  private connect(replay: boolean): void {
    const ws = new WebSocket(this.wsURL);
    ws.onopen = () => {
      this.ws = ws;
      this.flush(ws, replay);
      this.status(replay ? "replaying" : "stream_connected", replay ? "Replaying retained audio after reconnect." : "Streaming transcription connected.");
    };
    ws.onmessage = (event) => this.handleMessage(event as MessageEvent<string>);
    ws.onclose = () => {
      if (this.finalReceived || this.stopped || this.capture === null) return;
      if (this.reconnects >= 2) {
        this.onError?.(WHISPER_FAILED_SENTINEL);
        return;
      }
      const delay = this.reconnects++ === 0 ? 1_000 : 3_000;
      this.status("reconnecting", "Connection interrupted; replaying retained audio.");
      setTimeout(() => this.connect(true), delay);
    };
  }

  preConnect(language: string): void {
    if (this.ws?.readyState === WebSocket.OPEN || this.ws?.readyState === WebSocket.CONNECTING) return;
    this.language = language;
    if (!this.newIdentity()) return;
    this.wsURL = buildVoiceStreamWsUrl(this.language, this.sessionId, this.resumeToken);
    this.connect(false);
  }

  async start(preWarmedStream?: MediaStream): Promise<void> {
    const recovered = loadUnfinishedSession();
    if (recovered) {
      this.sessionId = recovered.sessionId;
      this.resumeToken = recovered.resumeToken;
    } else if (!this.sessionId && !this.newIdentity()) {
      return;
    }

    this.wsURL = buildVoiceStreamWsUrl(this.language, this.sessionId, this.resumeToken);
    const tracksAreLive = preWarmedStream?.getTracks().every((track) => track.readyState === "live") ?? false;
    this.stream = tracksAreLive ? preWarmedStream ?? null : await navigator.mediaDevices.getUserMedia({ audio: true }).catch(() => null);
    if (!this.stream) {
      this.onError?.("Microphone access denied");
      return;
    }

    this.stopped = false;
    this.doneSent = false;
    this.tailDropArmed = false;
    this.finalReceived = false;
	this.terminalFailure = false;
    this.reconnects = 0;
    this.pendingFrames = [];
    this.allPCM = [];
    this.retainedPcmBytes = 0;
    this.nextSequence = 0n;
    this.nextSample = 0n;
    this.writes = Promise.resolve();
    this.startedAt = Date.now();
    await this.initializeJournal();
    this.capture = await createCanonicalPcmCapture(getSharedAudioContext(), this.stream, (samples, rate) => this.queueFrame(frameToCanonicalPcm16(samples, rate)));
    if (!this.ws) this.connect(this.nextSequence > 0n);
    else if (this.ws.readyState === WebSocket.OPEN) this.flush(this.ws, false);
  }

  sendSegmentBoundary(): void {
    if (this.ws?.readyState === WebSocket.OPEN) this.ws.send(JSON.stringify({ type: "segment-boundary" }));
  }

  sendVadState(speaking: boolean): void {
    if (this.ws?.readyState === WebSocket.OPEN) this.ws.send(JSON.stringify({ type: speaking ? "vad-speech-start" : "vad-speech-end" }));
  }

  dropTail(): void {
    this.tailDropArmed = true;
  }

  stop(): void {
    this.stopped = true;
    this.capture?.stop();
    this.capture = null;
    const now = Date.now();
    this.lastTurn = this.tailDropArmed || this.allPCM.length === 0
      ? null
      : { blob: encodeWavFromPcm16(concatInt16(this.allPCM), TARGET_SAMPLE_RATE), mimeType: "audio/wav", durationMs: now - this.startedAt, capturedAt: now };
    if (!this.retainStream) {
      this.stream?.getTracks().forEach((track) => track.stop());
      this.stream = null;
    }
    this.writes = this.writes.then(() => {
      if (this.ws) this.sendDone(this.ws);
    });
    this.finalTimer = setTimeout(() => {
      if (this.finalReceived) return;
      void transcribeAudioWithRetry(encodeWavFromPcm16(concatInt16(this.allPCM), TARGET_SAMPLE_RATE), 2, this.language)
        .then((text) => this.onResult?.(text.trim()))
        .catch(() => this.onError?.(WHISPER_FAILED_SENTINEL));
    }, computeFinalTimeout(now - this.startedAt));
  }

  dispose(): void {
    this.stopped = true;
    if (this.finalTimer) clearTimeout(this.finalTimer);
    this.capture?.stop();
    this.capture = null;
    this.ws?.close();
    this.stream?.getTracks().forEach((track) => track.stop());
    this.stream = null;
    this.lastTurn = null;
  }
}
