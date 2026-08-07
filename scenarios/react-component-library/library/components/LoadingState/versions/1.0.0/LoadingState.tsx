/** @vrooliComponentSource react-component-library:LoadingState */
import type { ReactNode } from "react";
import { AsyncBoundary } from "../../../AsyncBoundary/versions/1.0.0/AsyncBoundary";

export function LoadingState({
  label = "Loading…",
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
}
