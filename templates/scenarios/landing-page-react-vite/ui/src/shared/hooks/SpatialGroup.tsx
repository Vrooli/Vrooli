import { useEffect, useRef, type ReactNode } from 'react';
import type { SpatialNavController, FocusGroupMode, FocusGroupOptions } from '@vrooli/iframe-bridge/spatial';

interface SpatialGroupProps {
  /** Navigation mode for this focus group. */
  mode: FocusGroupMode;
  /** Ref to the SpatialNavController returned by useSpatialNav(). */
  controllerRef: React.RefObject<SpatialNavController | null>;
  /** Optional focus group options (e.g., { wrap: true }). */
  options?: FocusGroupOptions;
  children: ReactNode;
  className?: string;
}

/**
 * Wraps children in a `<div>` that registers a spatial navigation focus group.
 *
 * - `mode="spatial"` — D-pad navigates between focusable children (default UI).
 * - `mode="passthrough"` — raw input flows to the component (graphs, canvases).
 * - `mode="grid"` — children treated as a grid for row/col navigation.
 * - `mode="modal"` — pushes a scope that traps spatial navigation inside
 *   (for dialogs/modals).  Auto-focuses the first focusable child.
 *
 * ```tsx
 * const spatialNav = useSpatialNav();
 *
 * <SpatialGroup controllerRef={spatialNav} mode="spatial">
 *   <button>Item 1</button>
 *   <button>Item 2</button>
 * </SpatialGroup>
 *
 * <SpatialGroup controllerRef={spatialNav} mode="modal">
 *   <Dialog>...</Dialog>
 * </SpatialGroup>
 * ```
 */
export function SpatialGroup({
  mode,
  controllerRef,
  options,
  children,
  className,
}: SpatialGroupProps) {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const el = ref.current;
    const ctrl = controllerRef.current;
    if (!el || !ctrl) return;

    if (mode === 'modal') {
      ctrl.pushScope(el);
      return () => { ctrl.popScope(); };
    }

    return ctrl.registerGroup(el, mode, options);
  }, [mode, controllerRef, options]);

  return (
    <div ref={ref} style={{ display: 'contents' }} className={className}>
      {children}
    </div>
  );
}
