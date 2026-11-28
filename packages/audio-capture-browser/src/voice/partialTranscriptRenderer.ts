import type { VoiceInputState } from "./types";

export type VoiceStateUpdate = (update: (state: VoiceInputState) => VoiceInputState) => void;

export interface AnimationFrameScheduler {
  request(callback: () => void): number;
  cancel?(handle: number): void;
}

function browserScheduler(): AnimationFrameScheduler | null {
  if (typeof requestAnimationFrame !== "function") return null;
  return {
    request: (callback) => requestAnimationFrame(callback),
    cancel: typeof cancelAnimationFrame === "function"
      ? (handle) => cancelAnimationFrame(handle)
      : undefined,
  };
}

/** Coalesces replaceable interim text into one UI update per animation frame. */
export class PartialTranscriptRenderer {
  private pendingText: string | undefined;
  private frameHandle: number | null = null;

  constructor(
    private readonly updateState: VoiceStateUpdate,
    private readonly scheduler: AnimationFrameScheduler | null = browserScheduler(),
  ) {}

  schedule(text: string): void {
    this.pendingText = text;
    if (!this.scheduler) {
      this.flush();
      return;
    }
    if (this.frameHandle !== null) return;
    this.frameHandle = this.scheduler.request(() => {
      this.frameHandle = null;
      this.flush();
    });
  }

  cancel(): void {
    if (this.frameHandle !== null) this.scheduler?.cancel?.(this.frameHandle);
    this.frameHandle = null;
    this.pendingText = undefined;
  }

  private flush(): void {
    const text = this.pendingText;
    this.pendingText = undefined;
    if (text === undefined) return;
    this.updateState((state) => state.partialTranscript === text && state.fallbackNotice === null
      ? state
      : { ...state, partialTranscript: text, fallbackNotice: null });
  }
}
