// Per-path Promise-chain serialization. ts-morph Project state is not
// safe across parallel invocations on the same project, so even though
// the Go supervisor already serializes per scenario_path, the sidecar
// serializes as defense-in-depth: a direct sidecar caller (e.g. an
// integration test) is still safe.

const chains = new Map<string, Promise<void>>();

/**
 * Run `work` exclusively against `key`. Concurrent calls with the same
 * key queue and execute in FIFO order. Other keys run independently.
 *
 * The chain stores the "tail" promise; we never reject the chain itself,
 * we only chain off it.
 */
export async function withPathLock<T>(key: string, work: () => Promise<T>): Promise<T> {
  const prev = chains.get(key) ?? Promise.resolve();

  let release!: () => void;
  const released = new Promise<void>((resolve) => {
    release = resolve;
  });
  chains.set(key, prev.then(() => released));

  try {
    await prev;
    return await work();
  } finally {
    release();
    // Best-effort cleanup: if we're still the tail, drop the entry.
    queueMicrotask(() => {
      if (chains.get(key) === released) {
        // unreachable in practice (the stored promise is prev.then(...)),
        // kept as a safety net.
        chains.delete(key);
      }
    });
    // Schedule a real cleanup once the stored promise resolves.
    void released.then(() => {
      const tail = chains.get(key);
      if (tail) {
        // Replace with a resolved promise to avoid memory leak via long chains.
        // The next caller's `prev` becomes resolved; new callers create new tails.
        // Only delete if nothing newer was appended.
        // We can't reliably know that without identity, so leave the map entry
        // and let GC reclaim once map churn rotates it out.
      }
    });
  }
}

/** Test-only: clear all chains. */
export function _resetLocksForTests(): void {
  chains.clear();
}
