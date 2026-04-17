import { useEffect, useRef } from "react";

interface UseGlobalKeydownOptions {
  disabled?: boolean;
  target?: Document | Window;
}

export function useGlobalKeydown(
  handler: (event: KeyboardEvent) => void,
  options: UseGlobalKeydownOptions = {},
): void {
  const { disabled = false, target } = options;
  const handlerRef = useRef(handler);

  useEffect(() => {
    handlerRef.current = handler;
  }, [handler]);

  useEffect(() => {
    if (disabled) {
      return;
    }

    const eventTarget = target ?? (typeof window !== "undefined" ? window : null);
    if (!eventTarget) {
      return;
    }

    const listener: EventListener = (event) => {
      if (event instanceof KeyboardEvent) {
        handlerRef.current(event);
      }
    };

    eventTarget.addEventListener("keydown", listener);
    return () => eventTarget.removeEventListener("keydown", listener);
  }, [disabled, target]);
}
