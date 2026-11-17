/** Single-consumer push queue for Connect bidi request iterables. */
export class PushableAsyncIterable<T> implements AsyncIterable<T> {
  private readonly values: T[] = [];
  private waiter: ((result: IteratorResult<T>) => void) | null = null;
  private ended = false;

  constructor(private readonly onYield?: (value: T) => void) {}

  push(value: T): void {
    if (this.ended) return;
    if (this.waiter) {
      const waiter = this.waiter;
      this.waiter = null;
      waiter({ done: false, value });
    } else {
      this.values.push(value);
    }
  }

  finish(): void {
    this.ended = true;
    if (this.values.length === 0) {
      this.waiter?.({ done: true, value: undefined });
      this.waiter = null;
    }
  }

  [Symbol.asyncIterator](): AsyncIterator<T> {
    const deliver = (value: T): IteratorResult<T> => {
      this.onYield?.(value);
      return { done: false, value };
    };
    return {
      next: () => {
        const value = this.values.shift();
        if (value !== undefined) return Promise.resolve(deliver(value));
        if (this.ended) return Promise.resolve({ done: true, value: undefined });
        return new Promise((resolve) => {
          this.waiter = (result) => resolve(result.done ? { done: true, value: undefined } : deliver(result.value));
        });
      },
      return: () => {
        this.finish();
        return Promise.resolve({ done: true, value: undefined });
      },
    };
  }
}
