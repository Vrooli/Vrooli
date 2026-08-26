/**
 * @libraryId react-component-library:useOptimisticAction
 * @displayName useOptimisticAction
 * @description An optimistic state controller that commits successful predictions and deterministically restores the previous value on failure.
 * @version 1.0.1
 * @tags ["runtime","async","optimistic","recovery"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource hooks.use-optimistic-action */
import { useCallback, useEffect, useRef, useState } from "react";
import {
  useAsyncAction,
  type AsyncActionStatus,
} from "@vrooli/react-component-library/useAsyncAction/1.0.0";

export interface UseOptimisticActionOptions<T, R> {
  value: T;
  action: (nextValue: T, signal: AbortSignal) => Promise<R>;
  onCommit?: (nextValue: T, result: R) => void;
  onRollback?: (previousValue: T, error: unknown) => void;
}

export function useOptimisticAction<T, R>({
  value: externalValue,
  action,
  onCommit,
  onRollback,
}: UseOptimisticActionOptions<T, R>) {
  const [value, setValue] = useState(externalValue);
  const previousRef = useRef(externalValue);
  const nextRef = useRef(externalValue);
  const externalRef = useRef(externalValue);
  const actionRef = useRef(action);
  const commitRef = useRef(onCommit);
  const rollbackRef = useRef(onRollback);
  externalRef.current = externalValue;
  actionRef.current = action;
  commitRef.current = onCommit;
  rollbackRef.current = onRollback;
  const asyncAction = useAsyncAction<R>((signal) => actionRef.current(nextRef.current, signal), {
    onSuccess: (result) => {
      setValue(nextRef.current);
      commitRef.current?.(nextRef.current, result);
    },
    onError: (error) => {
      setValue(previousRef.current);
      rollbackRef.current?.(previousRef.current, error);
    },
  });
  const {
    run: runAction,
    retry,
    reset: resetAction,
    cancel: cancelAction,
    status: actionStatus,
    error,
    value: result,
  } = asyncAction;
  useEffect(() => {
    if (actionStatus === "idle") setValue(externalValue);
  }, [actionStatus, externalValue]);
  const run = useCallback(
    (nextValue: T) => {
      previousRef.current = value;
      nextRef.current = nextValue;
      setValue(nextValue);
      return runAction();
    },
    [runAction, value],
  );
  const reset = useCallback(() => {
    resetAction();
    setValue(externalRef.current);
    previousRef.current = externalRef.current;
    nextRef.current = externalRef.current;
  }, [resetAction]);
  const rollback = useCallback(() => {
    cancelAction();
    setValue(previousRef.current);
  }, [cancelAction]);
  const status: AsyncActionStatus = actionStatus;
  return {
    value,
    status,
    error,
    result,
    run,
    retry,
    rollback,
    reset,
    cancel: cancelAction,
  };
}
