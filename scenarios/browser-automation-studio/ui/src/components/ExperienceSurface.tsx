/**
 * @vrooliComponentSource react-component-library:ExperienceSurface
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption 1c392e22-3b73-4c50-bc31-334608119f04
 * @vrooliComponentAppliedAt 2026-08-20T17:17:31Z
 * @vrooliComponentSourceSha256 a7fe644950883cd6fddc56dceaf0007f2f3bae55ad2f08b34afee853f57d77c3
 * @vrooliComponentDriftHash 40fcf8e5928016e9aea2dd2160267db0af7bc1a6a47d668e97bbf3285faa911f
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
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
  /** Optional stable accessibility-test hook; defaults from surfaceId. */
  "data-testid"?: string;
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
          className="sr-only"
        >
          {statusMessage}
        </p>
      ) : null}
      {children}
    </Tag>
  );
}
