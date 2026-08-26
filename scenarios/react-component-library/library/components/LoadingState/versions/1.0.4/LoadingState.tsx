/**
 * @libraryId react-component-library:LoadingState
 * @displayName LoadingState
 * @description A compact loading state that keeps the promised content geometry stable while work is in progress.
 * @version 1.0.4
 * @tags ["feedback","async","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:LoadingState */
import { translate } from "../../../../hooks/useLocale/versions/1.0.1/useLocale";
import { withClassName } from "../../../../foundations/ClassMerge/versions/1.0.1/ClassMerge";

import type { ReactNode } from "react";
import { AsyncBoundary } from "../../../AsyncBoundary/versions/1.0.0/AsyncBoundary";

export const LoadingState = withClassName(function LoadingState({
  label = translate("feedback.loading-state.label.1", "Loading…"),
  detail,
  children,
}: {
  label?: ReactNode;
  detail?: ReactNode;
  children?: ReactNode;
}) {
  return (
    <AsyncBoundary
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
