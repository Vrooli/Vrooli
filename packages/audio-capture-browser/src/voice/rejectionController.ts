import type { Dispatch, SetStateAction } from "react";
import type { TranscriptionProvider, VoiceInputState, VoiceRejection } from "./types";

const REJECTION_RETENTION_TTL_MS = 5 * 60 * 1000;

function generateRejectionId(): string {
  return `rej-${Date.now()}-${Math.random().toString(36).slice(2, 10)}`;
}

export interface VoiceRejectionControllerOptions {
  getProvider: () => TranscriptionProvider | null;
  setState: Dispatch<SetStateAction<VoiceInputState>>;
  getLanguage: () => string;
  transcribeAudio: (audio: Blob, language: string) => Promise<string>;
  transcribeAudioBypassFilter: (audio: Blob, language: string) => Promise<string>;
  onTranscript: (text: string) => void;
}

/** Owns retained turn-audio notices and their retry/expiry lifecycle. */
export class VoiceRejectionController {
  private readonly options: VoiceRejectionControllerOptions;
  private pending: { score: number; threshold: number } | null = null;
  private current: VoiceRejection | null = null;
  private ttl: ReturnType<typeof setTimeout> | null = null;

  constructor(options: VoiceRejectionControllerOptions) {
    this.options = options;
  }

  get hasPending(): boolean {
    return this.pending !== null;
  }

  markPending(score: number, threshold: number): void {
    this.pending = { score, threshold };
  }

  clearPending(): void {
    this.pending = null;
  }

  clearForUnmount(): void {
    this.clearTimer();
    this.current = null;
    this.pending = null;
  }

  get currentRejection(): VoiceRejection | null {
    return this.current;
  }

  surfacePending(): void {
    const pending = this.pending;
    if (!pending) return;
    this.pending = null;

    const audio = this.options.getProvider()?.getLastTurnAudio() ?? null;
    const id = generateRejectionId();
    const createdAt = Date.now();
    const rejection: VoiceRejection = audio
      ? {
          kind: "retryable",
          cause: "speaker-rejected",
          id,
          blob: audio.blob,
          mimeType: audio.mimeType,
          durationMs: audio.durationMs,
          score: pending.score,
          threshold: pending.threshold,
          createdAt,
          status: "idle",
        }
      : {
          kind: "explanatory",
          id,
          reason: "This provider does not retain audio; please record again to retry.",
          score: pending.score,
          threshold: pending.threshold,
          createdAt,
        };
    this.show(rejection);
  }

  surfaceEmptyTranscript(): void {
    const audio = this.options.getProvider()?.getLastTurnAudio() ?? null;
    if (!audio) {
      this.options.setState((s) => ({ ...s, error: "No speech detected — tap to try again" }));
      return;
    }
    this.show({
      kind: "retryable",
      cause: "empty-transcript",
      id: generateRejectionId(),
      blob: audio.blob,
      mimeType: audio.mimeType,
      durationMs: audio.durationMs,
      score: 0,
      threshold: 0,
      createdAt: Date.now(),
      status: "idle",
    });
  }

  dismiss(): void {
    this.clearTimer();
    this.options.getProvider()?.disposeLastTurn();
    this.current = null;
    this.options.setState((s) => (s.rejectedAudio ? { ...s, rejectedAudio: null } : s));
  }

  async retry(): Promise<void> {
    const current = this.current;
    if (!current || current.kind !== "retryable" || current.status === "retrying") return;

    this.updateCurrent(current.id, { ...current, status: "retrying", errorMessage: undefined });
    const langSetting = this.options.getLanguage();
    const lang = langSetting === "auto" ? "" : (langSetting.split("-")[0] ?? "en");

    try {
      const text = current.cause === "empty-transcript"
        ? await this.options.transcribeAudio(current.blob, lang)
        : await this.options.transcribeAudioBypassFilter(current.blob, lang);
      const trimmed = text.trim();
      if (this.current?.id !== current.id) return;
      if (trimmed) {
        this.options.onTranscript(trimmed);
        this.clearTimer();
        this.options.getProvider()?.disposeLastTurn();
        this.current = null;
        this.options.setState((s) => (s.rejectedAudio?.id === current.id
          ? { ...s, rejectedAudio: null }
          : s));
      } else {
        this.updateCurrent(current.id, {
          ...current,
          status: "failed",
          errorMessage: "No speech detected in audio",
        });
      }
    } catch (err) {
      if (this.current?.id !== current.id) return;
      const raw = err instanceof Error ? err.message : "Network error";
      const msg = raw.length > 200 ? raw.slice(0, 197) + "…" : raw;
      this.updateCurrent(current.id, { ...current, status: "failed", errorMessage: msg });
    }
  }

  private show(rejection: VoiceRejection): void {
    this.current = rejection;
    this.options.setState((s) => ({ ...s, rejectedAudio: rejection }));
    this.clearTimer();
    this.ttl = setTimeout(() => {
      if (this.current?.id !== rejection.id) return;
      this.options.getProvider()?.disposeLastTurn();
      this.current = null;
      this.ttl = null;
      this.options.setState((s) => (s.rejectedAudio?.id === rejection.id
        ? { ...s, rejectedAudio: null }
        : s));
    }, REJECTION_RETENTION_TTL_MS);
  }

  private updateCurrent(id: string, rejection: VoiceRejection): void {
    this.current = rejection;
    this.options.setState((s) => (s.rejectedAudio?.id === id
      ? { ...s, rejectedAudio: rejection }
      : s));
  }

  private clearTimer(): void {
    if (this.ttl) clearTimeout(this.ttl);
    this.ttl = null;
  }
}
