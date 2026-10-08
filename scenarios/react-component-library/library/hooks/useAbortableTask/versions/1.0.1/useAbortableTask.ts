/**
 * @libraryId react-component-library:useAbortableTask
 * @displayName useAbortableTask
 * @description A task primitive that automatically cancels obsolete work when inputs change, the component unmounts, or a newer invocation supersedes the current one.
 * @version 1.0.1
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:useAbortableTask
 * @vrooliComponentSourceSlot hooks.use-abortable-task */
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
