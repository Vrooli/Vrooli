import { useState } from "react";
import { useOverlaySurface } from "./useOverlaySurface";

interface HarnessProps {
  swipe?: boolean;
}

function Harness({ swipe = false }: HarnessProps) {
  const [open, setOpen] = useState(true);
  const overlay = useOverlaySurface({
    open,
    onOpenChange: setOpen,
    kind: "drawer",
    dismiss: { escape: true, backdrop: true, swipe: swipe ? "down" : false },
  });
  if (!overlay.present) {
    return (
      <output data-testid="runtime.use-overlay-surface.dismissed">
        dismissed
      </output>
    );
  }
  return (
    <div data-testid="runtime.use-overlay-surface">
      <button
        type="button"
        data-testid="runtime.use-overlay-surface.backdrop"
        aria-label="Dismiss"
        {...overlay.backdropProps}
      />
      <section
        {...overlay.surfaceProps}
        role="dialog"
        aria-modal="true"
        aria-label="Surface"
      >
        {swipe ? (
          <button
            {...overlay.grabberProps}
            data-testid="runtime.use-overlay-surface.grabber"
            aria-label="Dismiss"
          >
            <span aria-hidden />
          </button>
        ) : null}
        <p data-testid="runtime.use-overlay-surface.state">{overlay.state}</p>
      </section>
    </div>
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
