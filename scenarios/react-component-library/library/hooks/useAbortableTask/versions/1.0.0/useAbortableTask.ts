/** @vrooliComponentSource hooks.use-abortable-task */
import { useCallback, useEffect, useRef } from "react";

export function useAbortableTask<T>(task: (signal: AbortSignal) => Promise<T>) {
  const controller = useRef<AbortController | null>(null);
  const run = useCallback(() => {
    controller.current?.abort();
    controller.current = new AbortController();
    return task(controller.current.signal);
  }, [task]);
  useEffect(() => () => controller.current?.abort(), []);
  return { run, abort: useCallback(() => controller.current?.abort(), []) };
}
