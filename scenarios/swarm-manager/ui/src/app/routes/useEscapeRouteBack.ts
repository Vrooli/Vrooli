import { useCallback } from "react";
import { useGlobalKeyDown } from "../../hooks/useGlobalKeyDown";

function isInputElement(el: HTMLElement): boolean {
  return (
    el.tagName === "INPUT" ||
    el.tagName === "TEXTAREA" ||
    el.isContentEditable ||
    el.closest(".monaco-editor") !== null
  );
}

export function useEscapeRouteBack(onBack: () => void, enabled = true): void {
  const handleKeyDown = useCallback((event: KeyboardEvent) => {
      const target = event.target as HTMLElement | null;
      if (target && isInputElement(target)) return;
      if (event.key === "Escape") {
        event.preventDefault();
        onBack();
      }
    }, [onBack]);

  useGlobalKeyDown(handleKeyDown, { enabled });
}
