import type { HTMLAttributes, ReactNode } from "react";
import type { SurfaceState } from "../../hooks/useSurfaceState";

export type ExperienceSurfaceState = SurfaceState;

interface ExperienceSurfaceProps extends HTMLAttributes<HTMLElement> {
  surfaceId: string;
  state: ExperienceSurfaceState;
  children: ReactNode;
  statusMessage?: string;
}

// A generated scenario owns layout and presentation, while this small
// semantic boundary keeps its authored readiness contract observable.
export function ExperienceSurface({
  surfaceId,
  state,
  children,
  statusMessage,
  ...props
}: ExperienceSurfaceProps) {
  const busy = ["loading", "saving", "syncing", "refreshing", "retrying"].includes(state);
  const announces = ["success", "validation-error", "request-error"].includes(state);
  return (
    <section
      data-experience-surface={surfaceId}
      data-experience-state={state}
      aria-busy={busy || undefined}
      {...props}
    >
      {announces && statusMessage ? (
        <p role="status" aria-live="polite" className="sr-only" data-experience-announcement>
          {statusMessage}
        </p>
      ) : null}
      {children}
    </section>
  );
}
