/**
 * One save lane per design.
 *
 * Autosave used to skip a timer tick whenever a previous save was in flight
 * (`if (savingRef.current) return`), so an edit made during a slow save was
 * dropped until the next edit happened to schedule another tick. The queue
 * makes that impossible: a request while a save is running marks the id dirty
 * again, and the queue starts exactly one follow-up run when the first settles.
 *
 * A failed save is not retried immediately — the draft stays dirty and the
 * next edit (or an explicit retry) schedules it. Retrying a broken server in a
 * loop is worse than showing the failure.
 */
export interface SaveQueue {
  /** Ask for a save now; coalesces with an in-flight run. */
  run: (id: string) => void;
  /** True while a request for this id is in flight. */
  isSaving: (id: string) => boolean;
  /** Drop queued follow-ups (in-flight requests finish on their own). */
  dispose: () => void;
}

export function createSaveQueue(
  persist: (id: string) => Promise<void>,
  onActiveChange: (active: boolean) => void,
): SaveQueue {
  const inFlight = new Set<string>();
  const pending = new Set<string>();
  let disposed = false;

  const notify = () => onActiveChange(inFlight.size > 0);

  const pump = async (id: string): Promise<void> => {
    if (disposed) return;
    if (inFlight.has(id)) {
      pending.add(id);
      return;
    }
    inFlight.add(id);
    notify();
    let ok = true;
    try {
      await persist(id);
    } catch {
      ok = false;
    } finally {
      inFlight.delete(id);
      notify();
      const again = pending.delete(id);
      // pump() re-checks `disposed`, so no second guard here.
      if (again && ok) void pump(id);
    }
  };

  return {
    run: (id) => {
      void pump(id);
    },
    isSaving: (id) => inFlight.has(id),
    dispose: () => {
      disposed = true;
      pending.clear();
    },
  };
}
