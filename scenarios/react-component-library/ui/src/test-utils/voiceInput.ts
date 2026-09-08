import type {
  VoiceInputAdapter,
  VoiceInputCapture,
  VoiceInputClock,
  VoiceInputCues,
  VoiceInputEvent,
  VoiceInputMedia,
  VoiceInputTerminalReason,
} from "@vrooli/react-component-library/useVoiceInput/1.0.0";

export class FakeVoiceCapture implements VoiceInputCapture {
  stopped = 0;
  private listeners = new Set<() => void>();
  stop() {
    this.stopped += 1;
  }
  onEnded(listener: () => void) {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  }
  end() {
    this.listeners.forEach((listener) => listener());
  }
}

export class FakeVoiceMedia implements VoiceInputMedia {
  readonly capture = new FakeVoiceCapture();
  acquireCalls = 0;
  error: Error | undefined;
  async acquire() {
    this.acquireCalls += 1;
    if (this.error) throw this.error;
    return this.capture;
  }
}

export class FakeVoiceAdapter implements VoiceInputAdapter {
  connectCalls = 0;
  stopReasons: VoiceInputTerminalReason[] = [];
  retryCalls = 0;
  error: Error | undefined;
  private listener: ((event: VoiceInputEvent) => void) | undefined;
  async connect(listener: (event: VoiceInputEvent) => void) {
    this.connectCalls += 1;
    this.listener = listener;
    if (this.error) throw this.error;
  }
  stop(reason: VoiceInputTerminalReason) {
    this.stopReasons.push(reason);
  }
  async retry() {
    this.retryCalls += 1;
  }
  emit(event: VoiceInputEvent) {
    this.listener?.(event);
  }
}

export class FakeVoiceClock implements VoiceInputClock {
  private next = 0;
  private callbacks = new Map<number, () => void>();
  setTimeout(callback: () => void) {
    const id = this.next++;
    this.callbacks.set(id, callback);
    return id as unknown as ReturnType<typeof setTimeout>;
  }
  clearTimeout(handle: ReturnType<typeof setTimeout>) {
    this.callbacks.delete(handle as unknown as number);
  }
  fireAll() {
    [...this.callbacks.values()].forEach((callback) => callback());
  }
  get pending() {
    return this.callbacks.size;
  }
}

export class FakeVoiceCues implements VoiceInputCues {
  readonly played: Array<"start" | "stop"> = [];
  play(kind: "start" | "stop") {
    this.played.push(kind);
  }
}
