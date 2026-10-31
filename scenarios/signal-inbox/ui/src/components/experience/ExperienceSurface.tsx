import type { HTMLAttributes, ReactNode } from "react";

export type ExperienceSurfaceState = "loading" | "ready" | "empty" | "partial" | "error" | "static";

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
  const live = state === "loading" || state === "partial" || state === "error";
  return (
    <section
      data-experience-surface={surfaceId}
      data-experience-state={state}
      aria-busy={state === "loading" || undefined}
      {...props}
    >
      {live && statusMessage ? <p role="status" aria-live="polite" className="sr-only">{statusMessage}</p> : null}
      {children}
    </section>
  );
}
