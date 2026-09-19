import { useEffect } from "react";

interface UseGlobalKeyDownOptions {
  enabled?: boolean;
  target?: "window" | "document";
}

export function useGlobalKeyDown(
  handler: (event: KeyboardEvent) => void,
  { enabled = true, target = "window" }: UseGlobalKeyDownOptions = {},
): void {
  useEffect(() => {
    if (!enabled) return undefined;

    const eventTarget = target === "document" ? document : window;
    const listener: EventListener = (event) => handler(event as KeyboardEvent);
    eventTarget.addEventListener("keydown", listener);
    return () => eventTarget.removeEventListener("keydown", listener);
  }, [enabled, handler, target]);
}
