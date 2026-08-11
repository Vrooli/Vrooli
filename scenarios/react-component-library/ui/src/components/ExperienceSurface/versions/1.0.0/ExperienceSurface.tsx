/**
 * @vrooliComponentSource react-component-library:ExperienceSurface
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption 37e0f421-999f-47e7-8af9-1c30944ebafc
 * @vrooliComponentAppliedAt 2026-08-10T20:01:11Z
 * @vrooliComponentSourceSha256 7b3d5a41f540687bd3a5b73ff5f9b6b5a10276f0245fe922b2284ff5aafee742
 * @vrooliComponentDriftHash 99f7a8d2103b27ce527e54ad7f30a6ed848215608b46f5c7cae15991bb867c66
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
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
        <p role="status" aria-live="polite" aria-label={statusMessage} className="sr-only">
          {statusMessage}
        </p>
      ) : null}
      {children}
    </Tag>
  );
}
