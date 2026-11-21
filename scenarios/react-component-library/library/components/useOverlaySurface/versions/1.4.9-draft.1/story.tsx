import { useState, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { layerManager } from "@vrooli/react-component-library/LayerManager/2";
import { useOverlaySurface } from "./useOverlaySurface";

interface HarnessProps {
  swipe?: boolean;
  children?: ReactNode;
}

function Harness({ swipe = false, children }: HarnessProps) {
  const [open, setOpen] = useState(true);
  const overlay = useOverlaySurface({
    open,
    onOpenChange: setOpen,
    kind: "drawer",
    dismiss: { escape: true, backdrop: true, swipe: swipe ? "down" : false },
  });
  const portal = layerManager.portalContainer();
  if (!portal) return null;
  if (!overlay.present) {
    return createPortal(<output data-testid="runtime.use-overlay-surface.dismissed">dismissed</output>, portal);
  }
  return createPortal(
    <div data-testid="runtime.use-overlay-surface">
      <button
        type="button"
        data-testid="runtime.use-overlay-surface.backdrop"
        aria-label="Dismiss"
        {...overlay.backdropProps}
      />
      <section {...overlay.surfaceProps} role="dialog" aria-modal="true" aria-label="Surface">
        {swipe ? (
          <button
            {...overlay.grabberProps}
            data-testid="runtime.use-overlay-surface.grabber"
            aria-label="Dismiss"
          >
            <span aria-hidden />
          </button>
        ) : null}
        {children ?? <p data-testid="runtime.use-overlay-surface.state">{overlay.state}</p>}
      </section>
    </div>,
    portal,
  );
}

/** The substrate mounted without a gesture: escape and backdrop only. */
export function Default() {
  return <Harness />;
}

/** The same substrate with a dismissing drag, which adds the grabber props. */
export function WithSwipeDismissal() {
  return <Harness swipe />;
}

/** Consumer-owned loading content inside the open overlay shell. */
export function Loading() {
  return (
    <Harness>
      <p data-testid="runtime.use-overlay-surface.loading" role="status" aria-live="polite">
        Loading content…
      </p>
    </Harness>
  );
}

/** Consumer-owned error content inside the open overlay shell. */
export function ErrorState() {
  return (
    <Harness>
      <p data-testid="runtime.use-overlay-surface.error" role="alert">
        Unable to load content.
      </p>
    </Harness>
  );
}
