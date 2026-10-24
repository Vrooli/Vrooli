export type StageBatchSuccess<Response> = (response: Response) => void;

export type StageBatchRunner<Response> = (
  paths: string[],
  onSuccess: StageBatchSuccess<Response>,
  onSettled: () => void,
) => void;

type Schedule = (callback: () => void) => unknown;
type CancelSchedule = (handle: unknown) => void;

export interface StageBatcherOptions {
  scopeKey?: string;
  schedule?: Schedule;
  cancelSchedule?: CancelSchedule;
}

export interface StageBatcher<Response> {
  enqueue(paths: string[], onSuccess?: StageBatchSuccess<Response>): void;
  dispose(): void;
  readonly scopeKey: string;
}

/**
 * Sends the first stage request immediately and combines later clicks that
 * arrive while it is active into one follow-up request. The Git endpoint
 * already accepts multiple paths, so this avoids creating one authorization
 * and index-write operation per row click without delaying the first action.
 */
export function createStageBatcher<Response>(
  run: StageBatchRunner<Response>,
  onPendingPaths: (paths: string[]) => void,
  options: StageBatcherOptions = {},
): StageBatcher<Response> {
  const schedule = options.schedule ?? ((callback: () => void) => setTimeout(callback, 0));
  const cancelSchedule = options.cancelSchedule ?? ((handle: unknown) => clearTimeout(handle as ReturnType<typeof setTimeout>));
  const scopeKey = options.scopeKey ?? "default";
  const queuedPaths = new Set<string>();
  const queuedCallbacks: Array<StageBatchSuccess<Response>> = [];
  let active = false;
  let scheduled: unknown = null;
  let disposed = false;

  const flush = () => {
    scheduled = null;
    if (disposed || active || queuedPaths.size === 0) return;

    const paths = [...queuedPaths];
    const callbacks = queuedCallbacks.splice(0);
    queuedPaths.clear();
    onPendingPaths([]);
    active = true;

    run(
      paths,
      (response) => {
        if (disposed) return;
        callbacks.forEach((callback) => callback(response));
      },
      () => {
        active = false;
        if (disposed || queuedPaths.size === 0) return;
        scheduled = schedule(flush);
      },
    );
  };

  return {
    scopeKey,
    enqueue(paths, onSuccess) {
      if (disposed) return;
      if (scheduled !== null) {
        cancelSchedule(scheduled);
        scheduled = null;
      }
      const validPaths = paths.filter((path) => path.length > 0);
      if (validPaths.length === 0) return;
      validPaths.forEach((path) => queuedPaths.add(path));
      if (onSuccess) queuedCallbacks.push(onSuccess);
      onPendingPaths([...queuedPaths]);
      flush();
    },
    dispose() {
      disposed = true;
      if (scheduled !== null) cancelSchedule(scheduled);
      scheduled = null;
      queuedPaths.clear();
      queuedCallbacks.length = 0;
      onPendingPaths([]);
    },
  };
}
