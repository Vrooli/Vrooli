/**
 * @libraryId react-component-library:ExperienceSurface
 * @version 1.0.0
 * @status released
 * @deps {"react":"^18"}
 */
import type { HTMLAttributes, ReactNode } from "react";

export type ExperienceSurfaceState =
  | "loading"
  | "ready"
  | "empty"
  | "partial"
  | "error"
  | "static";

export interface ExperienceSurfaceProps extends HTMLAttributes<HTMLElement> {
  /** Stable authored region identity, never a CSS selector. */
  surfaceId: string;
  state: ExperienceSurfaceState;
  children: ReactNode;
  /** Human-readable state announcement for loading, partial, and error states. */
  statusMessage?: string;
  as?: "section" | "div" | "main" | "aside";
}

// ExperienceSurface intentionally owns only machine-readable lifecycle and
// accessible announcement semantics. Layout and presentation stay with the
// scenario or the composed state-pattern components.
export function ExperienceSurface({
  surfaceId,
  state,
  children,
  statusMessage,
  as: Tag = "section",
  ...props
}: ExperienceSurfaceProps) {
  const live = state === "loading" || state === "partial" || state === "error";
  return (
    <Tag
      data-experience-surface={surfaceId}
      data-experience-state={state}
      aria-busy={state === "loading" || undefined}
      {...props}
    >
      {live && statusMessage ? (
        <p
          role="status"
          aria-live="polite"
          aria-label={statusMessage}
          className="sr-only"
        >
          {statusMessage}
        </p>
      ) : null}
      {children}
    </Tag>
  );
}
