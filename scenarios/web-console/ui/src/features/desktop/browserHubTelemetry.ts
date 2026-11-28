export interface BrowserHubTelemetrySnapshot {
  frames: number;
  bytes: number;
  inputApplied: number;
  attachToFirstFrameMs?: number;
  inputToPaintedFrameMs?: number;
  inputToPaintedFrameSamples: number;
  frameAgeMs?: number;
  reconnectCount: number;
  reconnectAttempts: number;
  reconnectMs?: number;
}

type PendingInput = { sentAt: number; appliedAt?: number };

/** In-memory, per-viewer measurements for the same-origin BAS stream. */
export class BrowserHubTelemetry {
  private frames = 0;
  private bytes = 0;
  private inputApplied = 0;
  private attachStartedAt?: number;
  private attachToFirstFrameMs?: number;
  private reconnectStartedAt?: number;
  private reconnectCount = 0;
  private reconnectAttempts = 0;
  private reconnectMs?: number;
  private frameAgeMs?: number;
  private inputToPaintedFrameMs?: number;
  private inputToPaintedFrameSamples = 0;
  private readonly pendingInputs = new Map<string, PendingInput>();
  private readonly acknowledgedInputs = new Set<string>();

  addBytes(bytes: number): void {
    if (Number.isFinite(bytes) && bytes > 0) this.bytes += Math.floor(bytes);
  }

  beginAttach(at: number): void {
    if (this.attachStartedAt === undefined) this.attachStartedAt = at;
  }

  startReconnect(at: number): void {
    this.reconnectAttempts += 1;
    if (this.reconnectStartedAt === undefined) this.reconnectStartedAt = at;
  }

  markDisconnected(at: number): void {
    if (this.reconnectStartedAt === undefined) this.reconnectStartedAt = at;
  }

  private setFrameAge(capturedAt: string | undefined, paintedAtEpochMs: number): void {
    const capturedAtMs = capturedAt ? Date.parse(capturedAt) : NaN;
    this.frameAgeMs = Number.isFinite(capturedAtMs) ? Math.max(0, paintedAtEpochMs - capturedAtMs) : undefined;
  }

  recordInputSent(inputID: string, at: number): void {
    if (!inputID) return;
    this.pendingInputs.set(inputID, { sentAt: at });
    // Bound the in-memory correlation queue if an owner never acknowledges input.
    while (this.pendingInputs.size > 128) {
      const oldest = this.pendingInputs.keys().next().value as string | undefined;
      if (!oldest) break;
      this.pendingInputs.delete(oldest);
      this.acknowledgedInputs.delete(oldest);
    }
  }

  recordInputApplied(inputID: string | undefined, at: number): void {
    this.inputApplied += 1;
    if (!inputID) return;
    const pending = this.pendingInputs.get(inputID);
    if (!pending) return;
    pending.appliedAt = at;
    this.acknowledgedInputs.add(inputID);
  }

  recordFramePainted(at: number, capturedAt?: string, paintedAtEpochMs = Date.now()): void {
    this.frames += 1;
    this.setFrameAge(capturedAt, paintedAtEpochMs);
    if (this.attachToFirstFrameMs === undefined && this.attachStartedAt !== undefined) {
      this.attachToFirstFrameMs = Math.max(0, Math.round(at - this.attachStartedAt));
    }
    if (this.reconnectStartedAt !== undefined) {
      this.reconnectMs = Math.max(0, Math.round(at - this.reconnectStartedAt));
      this.reconnectCount += 1;
      this.reconnectStartedAt = undefined;
    }
    for (const inputID of this.acknowledgedInputs) {
      const pending = this.pendingInputs.get(inputID);
      if (pending?.appliedAt !== undefined) {
        // This is send-to-next-painted-frame latency, not proof that a pixel changed.
        this.inputToPaintedFrameMs = Math.max(0, Math.round(at - pending.sentAt));
        this.inputToPaintedFrameSamples += 1;
        this.pendingInputs.delete(inputID);
      }
    }
    this.acknowledgedInputs.clear();
  }

  snapshot(): BrowserHubTelemetrySnapshot {
    return {
      frames: this.frames,
      bytes: this.bytes,
      inputApplied: this.inputApplied,
      attachToFirstFrameMs: this.attachToFirstFrameMs,
      inputToPaintedFrameMs: this.inputToPaintedFrameMs,
      inputToPaintedFrameSamples: this.inputToPaintedFrameSamples,
      frameAgeMs: this.frameAgeMs,
      reconnectCount: this.reconnectCount,
      reconnectAttempts: this.reconnectAttempts,
      reconnectMs: this.reconnectMs,
    };
  }
}
