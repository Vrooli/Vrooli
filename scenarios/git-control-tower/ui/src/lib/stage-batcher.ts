export type StageBatchSuccess<Response> = (response: Response) => void;
export type StageBatchFailure = (error: unknown) => void;
export type StageBatchSettled = () => void;

export interface StageBatchCallbacks<Response> {
  onSuccess?: StageBatchSuccess<Response>;
  onError?: StageBatchFailure;
  onSettled?: StageBatchSettled;
}

export type StageBatchRunner<Response> = (paths: string[], callbacks: Required<StageBatchCallbacks<Response>>) => void;

type Schedule = (callback: () => void) => unknown;
type CancelSchedule = (handle: unknown) => void;

export interface StageBatcherOptions {
  scopeKey?: string;
  schedule?: Schedule;
  cancelSchedule?: CancelSchedule;
}

export interface StageBatcher<Response> {
  enqueue(paths: string[], callbacks?: StageBatchCallbacks<Response>): void;
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
  const queuedCallbacks: StageBatchCallbacks<Response>[] = [];
  let activeCallbacks: StageBatchCallbacks<Response>[] = [];
  let settleActive: (() => void) | undefined;
  let active = false;
  let scheduled: unknown = null;
  let disposed = false;

  const flush = () => {
    scheduled = null;
    if (disposed || active || queuedPaths.size === 0) return;

    const paths = [...queuedPaths];
    const callbacks = queuedCallbacks.splice(0);
    activeCallbacks = callbacks;
    queuedPaths.clear();
    onPendingPaths([]);
    active = true;

    let settled = false;
    const settle = () => {
      if (settled) return;
      settled = true;
      active = false;
      activeCallbacks = [];
      settleActive = undefined;
        callbacks.forEach((callback) => callback.onSettled?.());
      if (disposed || queuedPaths.size === 0) return;
      scheduled = schedule(flush);
    };
    settleActive = settle;
    run(paths, {
      onSuccess: (response) => {
        if (disposed) return;
        callbacks.forEach((callback) => callback.onSuccess?.(response));
      },
      onError: (error) => {
        if (disposed) return;
        callbacks.forEach((callback) => callback.onError?.(error));
      },
      onSettled: settle,
    });
  };

  return {
    scopeKey,
    enqueue(paths, callbacks = {}) {
      if (disposed) return;
      if (scheduled !== null) {
        cancelSchedule(scheduled);
        scheduled = null;
      }
      const validPaths = paths.filter((path) => path.length > 0);
      if (validPaths.length === 0) return;
      validPaths.forEach((path) => queuedPaths.add(path));
      if (callbacks.onSuccess || callbacks.onSettled || callbacks.onError) queuedCallbacks.push(callbacks);
      onPendingPaths([...queuedPaths]);
      flush();
    },
    dispose() {
      disposed = true;
      if (scheduled !== null) cancelSchedule(scheduled);
      scheduled = null;
      queuedPaths.clear();
      const queued = queuedCallbacks.splice(0);
      const interrupted = [...activeCallbacks, ...queued];
      const finishActive = settleActive;
      activeCallbacks = [];
      interrupted.forEach((callback) => callback.onError?.(new Error("Stage request interrupted when its repository view changed.")));
      if (finishActive) finishActive();
      queued.forEach((callback) => callback.onSettled?.());
      queuedCallbacks.length = 0;
      onPendingPaths([]);
    },
  };
}
