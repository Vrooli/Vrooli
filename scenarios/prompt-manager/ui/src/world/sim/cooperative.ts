export interface WorkProgress { completed: number; total: number }

/** Stable sorting with bounded copy/compare loops; never mutates the input. */
export function* sortSteps<T>(values: readonly T[], compare: (a: T, b: T) => number): Generator<WorkProgress, T[]> {
  const count = values.length
  let source: T[] = []
  let target: T[] = []
  let completed = 0
  const total = count * (1 + Math.ceil(Math.log2(Math.max(1, count))))
  for (const value of values) {
    if (completed % 128 === 0) yield { completed, total }
    source.push(value)
    completed++
  }
  for (let width = 1; width < count; width *= 2) {
    for (let start = 0; start < count; start += width * 2) {
      const middle = Math.min(start + width, count)
      const end = Math.min(start + width * 2, count)
      let left = start
      let right = middle
      for (let index = start; index < end; index++) {
        if (completed % 128 === 0) yield { completed, total }
        // Bounds above guarantee these slots exist, including generic undefined values.
        target[index] = left < middle && (right >= end || compare(source[left] as T, source[right] as T) <= 0)
          ? source[left++] as T : source[right++] as T
        completed++
      }
    }
    ;[source, target] = [target, source]
  }
  return source
}

/** Shared scheduler for pure generation iterators. Closing a cancelled iterator
 * releases its unpublished arrays; callers receive only completed results.
 */
export async function runCooperatively<P, T>(steps: Generator<P, T>, options: {
  signal?: AbortSignal
  onProgress?: (progress: P) => void
  yieldTask?: () => Promise<void>
  sliceMs?: number
} = {}): Promise<T> {
  const yieldTask = options.yieldTask ?? (() => new Promise<void>(resolve => setTimeout(resolve, 0)))
  const sliceMs = options.sliceMs ?? 8
  if (!Number.isFinite(sliceMs) || sliceMs < 0) throw new Error('Generation slice must be finite and nonnegative')
  let deadline = 0
  try {
    for (;;) {
      options.signal?.throwIfAborted()
      const step = steps.next()
      if (step.done) {
        options.signal?.throwIfAborted()
        return step.value
      }
      options.onProgress?.(step.value)
      if (performance.now() >= deadline) {
        await yieldTask()
        deadline = performance.now() + sliceMs
      }
    }
  } finally {
    // Generator.return's input is ignored by these iterators. It closes the
    // suspended stack without publishing any partially built result.
    steps.return(undefined as T)
  }
}
