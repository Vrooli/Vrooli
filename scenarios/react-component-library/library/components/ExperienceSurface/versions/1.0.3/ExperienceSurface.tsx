/**
 * @libraryId react-component-library:ExperienceSurface
 * @displayName Experience Surface
 * @version 1.0.3
 * @tags ["experience","lifecycle","surface","accessibility"]
 * @deps {"react":"^18"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import type { HTMLAttributes, ReactNode } from "react";

export type ExperienceSurfaceState = "loading" | "ready" | "empty" | "partial" | "error" | "static";

export interface ExperienceSurfaceProps extends HTMLAttributes<HTMLElement> {
  /** Stable authored region identity, never a CSS selector. */
  surfaceId: string;
  state: ExperienceSurfaceState;
  children: ReactNode;
  /** Human-readable state announcement for loading, partial, and error states. */
  statusMessage?: string;
  as?: "section" | "div" | "main" | "aside";
  /** Optional stable accessibility-test hook; defaults from surfaceId. */
  "data-testid"?: string;
}

// ExperienceSurface intentionally owns only machine-readable lifecycle and
// accessible announcement semantics. Layout and presentation stay with the
// scenario or the composed state-pattern components.
export const ExperienceSurface = withClassName(function ExperienceSurface({
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
      {...props}
      data-testid={props["data-testid"] ?? `experience-surface-${surfaceId}`}
      data-experience-surface={surfaceId}
      data-experience-state={state}
      aria-busy={state === "loading" || undefined}
    >
      {live && statusMessage ? (
        <p
          role="status"
          aria-live="polite"
          aria-label={statusMessage}
          style={{
            position: "absolute",
            inlineSize: 1,
            blockSize: 1,
            overflow: "hidden",
            clipPath: "inset(50%)",
            whiteSpace: "nowrap",
          }}
        >
          {statusMessage}
        </p>
      ) : null}
      {children}
    </Tag>
  );
});
