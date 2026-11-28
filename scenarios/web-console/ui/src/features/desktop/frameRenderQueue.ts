/** Single-flight preparation queue that keeps only the latest pending frame. */
export class LatestFrameRenderQueue<Input, Output> {
  private pending?: { value: Input };
  private running = false;

  constructor(
    private readonly prepare: (input: Input) => Promise<Output>,
    private readonly commit: (input: Input, output: Output) => void,
    private readonly discard: (output: Output) => void,
    private readonly onError: (error: unknown, input: Input) => void,
  ) {}

  enqueue(input: Input): void {
    this.pending = { value: input };
    this.pump();
  }

  clear(): void {
    this.pending = undefined;
  }

  private pump(): void {
    if (this.running) return;
    this.running = true;
    void (async () => {
      try {
        while (this.pending) {
          const current = this.pending.value;
          this.pending = undefined;
          let output: Output;
          try {
            output = await this.prepare(current);
          } catch (error) {
            this.onError(error, current);
            continue;
          }
          if (this.pending) {
            this.discard(output);
            continue;
          }
          try {
            this.commit(current, output);
          } catch (error) {
            this.onError(error, current);
          }
        }
      } finally {
        this.running = false;
        if (this.pending) this.pump();
      }
    })();
  }
}
