/**
 * @libraryId react-component-library:AsyncPanel
 * @version 1.0.0
 * @status released
 * @deps {"react":"^18"}
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

import type { ReactNode } from "react";
import {
  ExperienceSurface,
  type ExperienceSurfaceState,
} from "@vrooli/react-component-library/ExperienceSurface/1.0.0";

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
export const AsyncPanel = withClassName(function AsyncPanel({
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
    state === "loading" || state === "partial" || state === "error"
      ? fallback[state]
      : undefined;
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
});
