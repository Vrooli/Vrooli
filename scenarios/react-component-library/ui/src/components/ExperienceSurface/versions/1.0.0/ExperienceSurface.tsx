/**
 * @vrooliComponentSource react-component-library:ExperienceSurface
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption 1764068d-c846-4f31-bee2-08a89e048b12
 * @vrooliComponentAppliedAt 2026-08-06T03:51:02Z
 * @vrooliComponentSourceSha256 32faa950b40d6c0351e9c7f409a4965e874492731a5f21eaff28d00a854f0017
 * @vrooliComponentDriftHash 95b27fb9f20eec7721ee1ed402b4d628a7c00403b9951b252608501853299897
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
