/** @vrooliComponentSource hooks.use-async-action */
import { useCallback, useEffect, useRef, useState } from "react";

export type AsyncActionStatus =
  | "idle"
  | "pending"
  | "success"
  | "error"
  | "cancelled";

export type AsyncAction<T> = (signal: AbortSignal) => Promise<T>;

export interface UseAsyncActionOptions<T> {
  onSuccess?: (value: T) => void;
  onError?: (error: unknown) => void;
}

export function useAsyncAction<T>(
  action: AsyncAction<T> | (() => Promise<T>),
  options: UseAsyncActionOptions<T> = {},
) {
  const actionRef = useRef(action);
  const optionsRef = useRef(options);
  const mountedRef = useRef(true);
  const generationRef = useRef(0);
  const controllerRef = useRef<AbortController | null>(null);
  const [state, setState] = useState<{
    status: AsyncActionStatus;
    value?: T;
    error?: unknown;
  }>({ status: "idle" });

  actionRef.current = action;
  optionsRef.current = options;

  useEffect(() => {
    return () => {
      mountedRef.current = false;
      generationRef.current += 1;
      controllerRef.current?.abort();
    };
  }, []);

  const run = useCallback(async () => {
    controllerRef.current?.abort();
    const controller = new AbortController();
    controllerRef.current = controller;
    const generation = ++generationRef.current;
    if (mountedRef.current) setState({ status: "pending" });

    try {
      const value = await actionRef.current(controller.signal);
      if (generation !== generationRef.current) return value;
      if (mountedRef.current) {
        setState({ status: "success", value });
        optionsRef.current.onSuccess?.(value);
      }
      return value;
    } catch (error) {
      if (generation !== generationRef.current) throw error;
      if (controller.signal.aborted) {
        if (mountedRef.current) setState({ status: "cancelled" });
        throw error;
      }
      if (mountedRef.current) {
        setState({ status: "error", error });
        optionsRef.current.onError?.(error);
      }
      throw error;
    }
  }, []);

  const cancel = useCallback(() => {
    generationRef.current += 1;
    controllerRef.current?.abort();
    if (mountedRef.current) setState({ status: "cancelled" });
  }, []);

  const reset = useCallback(() => {
    generationRef.current += 1;
    controllerRef.current?.abort();
    if (mountedRef.current) setState({ status: "idle" });
  }, []);

  return { ...state, run, retry: run, cancel, reset };
}
