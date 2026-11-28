/** Pairs recording cues across the many capture shutdown paths. */
export class AudioCueSession {
  private active = false;

  constructor(private readonly cues: {
    start: () => void;
    stop: () => void;
    fault: () => void;
  }) {}

  start(): void {
    this.active = true;
    this.cues.start();
  }

  stop(): boolean {
    if (!this.active) return false;
    this.active = false;
    this.cues.stop();
    return true;
  }

  fault(): boolean {
    if (!this.active) return false;
    this.active = false;
    this.cues.fault();
    return true;
  }

  /** Clear on teardown paths where playing a cue would be misleading. */
  clear(): void {
    this.active = false;
  }
}
