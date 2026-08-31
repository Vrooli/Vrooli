/**
 * @libraryId react-component-library:useOverlaySurface
 * @displayName useOverlaySurface
 * @description Composes the shared lifecycle, focus, dismissal, portal, and motion contract for overlays.
 * @version 1.1.1
 * @tags ["runtime","overlay","accessibility"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import {
  useCallback,
  useEffect,
  useId,
  useRef,
  useState,
  type RefObject,
} from "react";
import { layerManager } from "@vrooli/react-component-library/LayerManager/2.0.0";
import { useControllableState } from "@vrooli/react-component-library/useControllableState/1.0.0";
import { useEscapeKey } from "@vrooli/react-component-library/useEscapeKey/1.0.0";
import { useFocusReturn } from "@vrooli/react-component-library/useFocusReturn/1.1.0";
import { useFocusTrap } from "@vrooli/react-component-library/useFocusTrap/1.0.0";
import { useReducedMotion } from "@vrooli/react-component-library/useReducedMotion/1.0.0";
import { useScrollLock } from "@vrooli/react-component-library/useScrollLock/2.0.0";
import { baseStyles } from "@vrooli/react-component-library/BaseStyles/1.0.0";
import { useLibraryStyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";

export type OverlayKind =
  | "dialog"
  | "alertdialog"
  | "menu"
  | "popover"
  | "sheet"
  | "drawer";
export interface OverlayDismissPolicy {
  escape?: boolean;
  backdrop?: boolean;
}
export interface UseOverlaySurfaceOptions {
  open?: boolean;
  defaultOpen?: boolean;
  onOpenChange?: (open: boolean) => void;
  modal?: boolean;
  kind: OverlayKind;
  dismiss?: OverlayDismissPolicy;
  initialFocusRef?: RefObject<HTMLElement | null>;
  returnFocusRef?: RefObject<HTMLElement | null>;
  scrollLock?: boolean;
  exitDurationMs?: number;
}

export function useOverlaySurface({
  open: controlledOpen,
  defaultOpen = false,
  onOpenChange,
  modal = true,
  kind,
  dismiss = { escape: true, backdrop: true },
  initialFocusRef,
  returnFocusRef,
  scrollLock = modal,
  exitDurationMs = 180,
}: UseOverlaySurfaceOptions) {
  useLibraryStyleSheet("base-styles", baseStyles);
  const [open, setOpen] = useControllableState({
    value: controlledOpen,
    defaultValue: defaultOpen,
    onChange: onOpenChange,
  });
  const reducedMotion = useReducedMotion();
  const [present, setPresent] = useState(open);
  const surfaceRef = useRef<HTMLElement | null>(null);
  const id = useId();

  useEffect(() => {
    if (open) {
      setPresent(true);
      return;
    }
    if (reducedMotion) {
      setPresent(false);
      return;
    }
    const timer = window.setTimeout(() => setPresent(false), exitDurationMs);
    return () => window.clearTimeout(timer);
  }, [exitDurationMs, open, reducedMotion]);

  const close = useCallback(() => setOpen(false), [setOpen]);
  useEffect(() => {
    if (!open) return;
    return layerManager.push({ id, kind, modal, dismiss: close });
  }, [close, id, kind, modal, open]);
  useScrollLock(open && scrollLock);
  useFocusTrap(open && modal, surfaceRef);
  useFocusReturn(open, returnFocusRef);
  useEscapeKey(open && dismiss.escape !== false, () => {
    if (layerManager.isTop(id)) close();
  });
  useEffect(() => {
    if (!open) return;
    (
      initialFocusRef?.current ??
      surfaceRef.current?.querySelector<HTMLElement>(
        "[autofocus], button, input, select, textarea, [tabindex]:not([tabindex='-1'])",
      )
    )?.focus();
  }, [initialFocusRef, open]);

  return {
    id,
    open,
    present,
    state: open ? ("open" as const) : ("closed" as const),
    setOpen,
    close,
    surfaceRef,
    surfaceProps: { ref: surfaceRef, "data-state": open ? "open" : "closed" },
    backdropProps: {
      onPointerDown: (event: {
        target: EventTarget | null;
        currentTarget: EventTarget | null;
      }) => {
        if (
          dismiss.backdrop !== false &&
          event.target === event.currentTarget &&
          layerManager.isTop(id)
        )
          close();
      },
    },
  };
}
