import { useCallback, useState } from "react";

import { useResizeObserver } from "./useResizeObserver";

const frame = {
  display: "grid",
  gap: "var(--space-sm)",
  padding: "var(--space-lg)",
  border: "var(--border-hairline) solid var(--color-border)",
  borderRadius: "var(--radius-panel)",
  background: "var(--color-surface)",
  color: "var(--color-foreground)",
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
          padding: "var(--space-sm)",
          border: "var(--border-hairline) solid var(--color-border)",
          borderRadius: "var(--radius-control)",
          background: "var(--color-surface-muted)",
        }}
      >
        <p style={{ margin: 0 }}>
          {rect ? `${Math.round(rect.width)} × ${Math.round(rect.height)}` : "Awaiting observation"}
        </p>
      </div>
    </div>
  );
}
