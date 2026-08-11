/**
 * @vrooliComponentSource react-component-library:AsyncPanel
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption 6323c8e4-8bee-4081-9279-97931a8c27a3
 * @vrooliComponentAppliedAt 2026-08-11T00:47:54Z
 * @vrooliComponentSourceSha256 d64611b7fa7f5514f7cf60b86785800dc08e7906c966f37b2faab72477916dd5
 * @vrooliComponentDriftHash 70524f0f3412c500bbcc862ee8e59f02cd264a681009a64acc377e4e91e8d37a
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
