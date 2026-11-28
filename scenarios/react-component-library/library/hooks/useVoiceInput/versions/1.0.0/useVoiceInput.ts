/**
 * @libraryId react-component-library:useVoiceInput
 * @version 1.0.0
 * @status released
 * @deps {"react":"^18"}
 */
import { useCallback, useEffect, useRef, useState } from "react";

export type VoiceInputMode = "always-on" | "timeout";
export type VoiceInputState =
  | "idle"
  | "preparing"
  | "recording"
  | "recovering"
  | "unavailable"
  | "error";
export type VoiceInputTerminalReason =
  | "explicit-stop"
  | "timeout"
  | "permission-denied"
  | "device-ended"
  | "adapter-failed"
  | "cancelled";

export interface VoiceInputSegment {
  readonly id: string;
  readonly text: string;
  readonly final: boolean;
}

export type VoiceInputEvent =
  | { readonly type: "segment"; readonly segment: VoiceInputSegment }
  | { readonly type: "rejected"; readonly reason: string }
  | { readonly type: "recovering" }
  | { readonly type: "reconnected" }
  | { readonly type: "failure"; readonly error: Error };

export interface VoiceInputAdapter {
  connect(listener: (event: VoiceInputEvent) => void): Promise<void>;
  stop(reason: VoiceInputTerminalReason): Promise<void> | void;
  retry?(): Promise<void>;
}

export interface VoiceInputCapture {
  stop(): void;
  onEnded(listener: () => void): () => void;
}

export interface VoiceInputMedia {
  acquire(): Promise<VoiceInputCapture>;
}

export interface VoiceInputClock {
  setTimeout(
    callback: () => void,
    delayMs: number,
  ): ReturnType<typeof setTimeout>;
  clearTimeout(handle: ReturnType<typeof setTimeout>): void;
}

export interface VoiceInputCues {
  play(kind: "start" | "stop"): Promise<void> | void;
}

export interface VoiceInputOptions {
  readonly adapter: VoiceInputAdapter;
  readonly media: VoiceInputMedia;
  readonly mode?: VoiceInputMode;
  readonly timeoutMs?: number;
  readonly clock?: VoiceInputClock;
  readonly cues?: VoiceInputCues;
  readonly onSettledSegment?: (segment: VoiceInputSegment) => void;
}

export interface VoiceInputSnapshot {
  readonly state: VoiceInputState;
  readonly mode: VoiceInputMode;
  readonly terminalReason?: VoiceInputTerminalReason;
  readonly settledSegments: readonly VoiceInputSegment[];
  readonly rejectionReason?: string;
}

const systemClock: VoiceInputClock = {
  setTimeout: window.setTimeout.bind(window),
  clearTimeout: window.clearTimeout.bind(window),
};
const noCues: VoiceInputCues = { play: () => undefined };

/** A deterministic, single-owner lifecycle used by the React hook and test fakes. */
export class VoiceInputController {
  private capture: VoiceInputCapture | undefined;
  private removeEndedListener: (() => void) | undefined;
  private timeout: ReturnType<typeof setTimeout> | undefined;
  private stopped = false;
  private startedCue = false;
  private stoppedCue = false;
  private snapshotValue: VoiceInputSnapshot;

  constructor(
    private readonly options: VoiceInputOptions,
    private readonly notify: (snapshot: VoiceInputSnapshot) => void = () =>
      undefined,
  ) {
    this.snapshotValue = {
      state: "idle",
      mode: options.mode ?? "always-on",
      settledSegments: [],
    };
  }

  get snapshot(): VoiceInputSnapshot {
    return this.snapshotValue;
  }

  async start(): Promise<void> {
    if (
      this.snapshotValue.state === "preparing" ||
      this.snapshotValue.state === "recording" ||
      this.snapshotValue.state === "recovering"
    )
      return;
    this.stopped = false;
    this.startedCue = false;
    this.stoppedCue = false;
    this.publish({
      state: "preparing",
      terminalReason: undefined,
      rejectionReason: undefined,
      settledSegments: [],
    });
    try {
      this.capture = await this.options.media.acquire();
      if (this.isStopped()) return;
      this.removeEndedListener = this.capture.onEnded(
        () => void this.end("device-ended"),
      );
      await this.options.adapter.connect((event) => this.handleEvent(event));
      if (this.isStopped()) return;
      this.publish({ state: "recording" });
      await this.playOnce("start");
      if (this.snapshotValue.mode === "timeout") {
        const timeoutMs = this.options.timeoutMs ?? 30_000;
        this.timeout = (this.options.clock ?? systemClock).setTimeout(
          () => void this.end("timeout"),
          timeoutMs,
        );
      }
    } catch (error) {
      await this.end(
        this.capture === undefined ? "permission-denied" : "adapter-failed",
        error instanceof Error ? error : new Error(String(error)),
      );
    }
  }

  async stop(): Promise<void> {
    await this.end("explicit-stop");
  }

  async retry(): Promise<void> {
    if (this.options.adapter.retry) await this.options.adapter.retry();
    await this.start();
  }

  private handleEvent(event: VoiceInputEvent): void {
    if (this.stopped) return;
    if (event.type === "segment") {
      if (
        this.snapshotValue.settledSegments.some(
          (segment) => segment.id === event.segment.id,
        )
      )
        return;
      const settledSegments = [
        ...this.snapshotValue.settledSegments,
        event.segment,
      ];
      this.publish({ settledSegments });
      this.options.onSettledSegment?.(event.segment);
    } else if (event.type === "rejected") {
      this.publish({ rejectionReason: event.reason });
    } else if (event.type === "recovering") {
      this.publish({ state: "recovering" });
    } else if (event.type === "reconnected") {
      this.publish({ state: "recording" });
    } else {
      void this.end("adapter-failed", event.error);
    }
  }

  private async end(
    reason: VoiceInputTerminalReason,
    _error?: Error,
  ): Promise<void> {
    if (this.stopped) return;
    this.stopped = true;
    if (this.timeout !== undefined)
      (this.options.clock ?? systemClock).clearTimeout(this.timeout);
    this.timeout = undefined;
    this.removeEndedListener?.();
    this.removeEndedListener = undefined;
    this.capture?.stop();
    this.capture = undefined;
    try {
      await this.options.adapter.stop(reason);
    } catch {
      /* cleanup must continue after transport failure */
    }
    if (this.startedCue) await this.playOnce("stop");
    this.publish({
      state:
        reason === "permission-denied"
          ? "unavailable"
          : reason === "adapter-failed"
            ? "error"
            : "idle",
      terminalReason: reason,
    });
  }

  private async playOnce(kind: "start" | "stop"): Promise<void> {
    if (
      (kind === "start" && this.startedCue) ||
      (kind === "stop" && this.stoppedCue)
    )
      return;
    if (kind === "start") this.startedCue = true;
    else this.stoppedCue = true;
    try {
      await (this.options.cues ?? noCues).play(kind);
    } catch {
      /* audible feedback is never a lifecycle dependency */
    }
  }

  private publish(next: Partial<VoiceInputSnapshot>): void {
    this.snapshotValue = { ...this.snapshotValue, ...next };
    this.notify(this.snapshotValue);
  }

  private isStopped(): boolean {
    return this.stopped;
  }
}

export function useVoiceInput(options: VoiceInputOptions) {
  const [snapshot, setSnapshot] = useState<VoiceInputSnapshot>({
    state: "idle",
    mode: options.mode ?? "always-on",
    settledSegments: [],
  });
  const controller = useRef<VoiceInputController>();
  if (!controller.current)
    controller.current = new VoiceInputController(options, setSnapshot);
  useEffect(
    () => () => {
      void controller.current?.stop();
    },
    [],
  );
  return {
    ...snapshot,
    start: useCallback(
      () => controller.current?.start() ?? Promise.resolve(),
      [],
    ),
    stop: useCallback(
      () => controller.current?.stop() ?? Promise.resolve(),
      [],
    ),
    retry: useCallback(
      () => controller.current?.retry() ?? Promise.resolve(),
      [],
    ),
  };
}
