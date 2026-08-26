/**
 * @libraryId react-component-library:LoadingState
 * @displayName LoadingState
 * @description A compact loading state that keeps the promised content geometry stable while work is in progress.
 * @version 1.0.5
 * @tags ["feedback","async","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:LoadingState */
import { resolveStrings } from "@vrooli/react-component-library/useLocale/1.0.1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

import type { ReactNode } from "react";
import { AsyncBoundary } from "@vrooli/react-component-library/AsyncBoundary/1.0.0";

export const LoadingState = withClassName(function LoadingState({
  label = resolveStrings("feedback.loading-state.loading", "Loading…"),
  detail,
  children,
}: {
  label?: ReactNode;
  detail?: ReactNode;
  children?: ReactNode;
}) {
  return (
    <AsyncBoundary
      data-testid="feedback.loading-state"
      status="pending"
      pending={
        <span
          style={{
            display: "grid",
            gap: "var(--space-2xs)",
            textAlign: "center",
          }}
        >
          <strong style={{ font: "var(--text-subtitle)" }}>{label}</strong>
          {detail && (
            <span
              style={{
                color: "var(--color-muted-foreground)",
                font: "var(--text-body)",
              }}
            >
              {detail}
            </span>
          )}
        </span>
      }
    >
      {children}
    </AsyncBoundary>
  );
});
