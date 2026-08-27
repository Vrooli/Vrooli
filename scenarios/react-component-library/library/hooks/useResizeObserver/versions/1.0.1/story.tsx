import { useCallback, useState } from "react";

import { useResizeObserver } from "./useResizeObserver";

const frame = {
  display: "grid",
  gap: "var(--space-sm, 16px)",
  padding: "var(--space-lg, 24px)",
  border: "var(--border-hairline, 1px) solid var(--color-border, #cbd5e1)",
  borderRadius: "var(--radius-panel, .75rem)",
  background: "var(--color-surface, #fff)",
  color: "var(--color-foreground, #0f172a)",
  inlineSize: "min(100%, 420px)",
};

export function Default() {
  const { ref, rect } = useResizeObserver<HTMLDivElement>();
  const [attached, setAttached] = useState(false);
  const attach = useCallback(
    (node: HTMLDivElement | null) => {
      ref(node);
      setAttached(Boolean(node));
    },
    [ref],
  );

  return (
    <div style={frame}>
      <div
        ref={attach}
        data-testid="hooks.use-resize-observer"
        data-attached={attached ? "true" : "false"}
        data-measured={rect ? "true" : "false"}
        style={{
          padding: "var(--space-sm, 16px)",
          border: "var(--border-hairline, 1px) solid var(--color-border, #cbd5e1)",
          borderRadius: "var(--radius-control, .5rem)",
          background: "var(--color-surface-muted, #f1f5f9)",
        }}
      >
        <p style={{ margin: 0 }}>
          {rect ? `${Math.round(rect.width)} × ${Math.round(rect.height)}` : "Awaiting observation"}
        </p>
      </div>
    </div>
  );
}
