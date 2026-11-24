import { useEffect, useRef } from "react";

type KeydownTarget = "document" | "window";

interface UseGlobalKeydownOptions {
  target?: KeydownTarget;
  enabled?: boolean;
}

/** Registers one lifecycle-safe global keydown listener for a focused hook. */
export function useGlobalKeydown(
  handler: (event: KeyboardEvent) => void,
  { target = "window", enabled = true }: UseGlobalKeydownOptions = {},
) {
  const handlerRef = useRef(handler);
  handlerRef.current = handler;

  useEffect(() => {
    if (!enabled) return;
    const eventTarget = target === "document" ? document : window;
    const onKeyDown = (event: Event) => {
      if (event instanceof KeyboardEvent) handlerRef.current(event);
    };
    eventTarget.addEventListener("keydown", onKeyDown);
    return () => eventTarget.removeEventListener("keydown", onKeyDown);
  }, [enabled, target]);
}
