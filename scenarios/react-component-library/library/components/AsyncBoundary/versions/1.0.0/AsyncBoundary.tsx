/** @vrooliComponentSource react-component-library:AsyncBoundary */
import type { ReactNode } from "react";
const panel = {
  border: "1px solid var(--color-border, #cbd5e1)",
  borderRadius: "var(--radius-panel, .75rem)",
  background: "var(--color-surface, #fff)",
  color: "var(--color-foreground, #0f172a)",
  padding: "var(--space-md, 24px)",
  boxShadow: "var(--elev-raised, 0 1px 3px rgb(15 23 42 / .08))",
};
export function AsyncBoundary({
  status = "idle",
  children,
  pending = "Loading…",
  error = "Something went wrong",
  retry,
}: {
  status?: "idle" | "pending" | "success" | "error";
  children?: ReactNode;
  pending?: ReactNode;
  error?: ReactNode;
  retry?: () => void;
}) {
  if (status === "pending")
    return (
      <div
        role="status"
        style={{
          ...panel,
          minHeight: 120,
          display: "grid",
          placeItems: "center",
        }}
      >
        {pending}
      </div>
    );
  if (status === "error")
    return (
      <div role="alert" style={{ ...panel, display: "grid", gap: 12 }}>
        {error}
        {retry && (
          <button type="button" onClick={retry}>
            Try again
          </button>
        )}
      </div>
    );
  return <>{children}</>;
}
