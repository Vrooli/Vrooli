/**
 * @vrooliComponentSource react-component-library:AsyncPanel
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption 14e7a96d-0082-4b4e-9f57-214af00edefe
 * @vrooliComponentAppliedAt 2026-08-06T03:51:03Z
 * @vrooliComponentSourceSha256 9957f1ab828b3d779216ac0f68c78eb3fbe936d00f81d52e00aa3e040db1e9b7
 * @vrooliComponentDriftHash 714d32de1a78f392a66cd9ca92c2b4b6aaa8d6d7e7c9c9712750ac873806c32b
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import type { ReactNode } from "react";
import {
  ExperienceSurface,
  type ExperienceSurfaceState,
} from "../../../ExperienceSurface/versions/1.0.0/ExperienceSurface";

export interface AsyncPanelProps {
  surfaceId: string;
  state: ExperienceSurfaceState;
  children?: ReactNode;
  loading?: ReactNode;
  empty?: ReactNode;
  partial?: ReactNode;
  error?: ReactNode;
  onRetry?: () => void;
  className?: string;
}

const fallback: Record<ExperienceSurfaceState, string> = {
  loading: "Loading…",
  ready: "",
  empty: "Nothing to show yet.",
  partial: "Some information is unavailable.",
  error: "This section could not be loaded.",
  static: "",
};

// AsyncPanel presents common lifecycle states while ExperienceSurface remains
// the source of semantic runtime evidence. It intentionally owns no card,
// grid, or page-shell styling so scenarios retain their visual composition.
export function AsyncPanel({
  surfaceId,
  state,
  children,
  loading,
  empty,
  partial,
  error,
  onRetry,
  className,
}: AsyncPanelProps) {
  const content =
    state === "ready" || state === "static"
      ? children
      : state === "loading"
        ? (loading ?? <p>{fallback.loading}</p>)
        : state === "empty"
          ? (empty ?? <p>{fallback.empty}</p>)
          : state === "partial"
            ? (partial ?? <p>{fallback.partial}</p>)
            : (error ?? (
                <>
                  <p>{fallback.error}</p>
                  {onRetry ? (
                    <button type="button" onClick={onRetry}>
                      Retry
                    </button>
                  ) : null}
                </>
              ));
  const statusMessage =
    state === "loading" || state === "partial" || state === "error" ? fallback[state] : undefined;
  return (
    <ExperienceSurface
      surfaceId={surfaceId}
      state={state}
      statusMessage={statusMessage}
      className={className}
    >
      {content}
    </ExperienceSurface>
  );
}
