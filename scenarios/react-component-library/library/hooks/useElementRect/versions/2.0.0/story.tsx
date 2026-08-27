import { useRef } from "react";

import { useElementRect } from "./useElementRect";

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

function Rig({ disabled = false }: { disabled?: boolean }) {
  const boxRef = useRef<HTMLDivElement>(null);
  const rect = useElementRect(boxRef, { disabled });

  return (
    <div style={frame}>
      <div
        ref={boxRef}
        data-testid="hooks.use-element-rect"
        data-measured={rect ? "true" : "false"}
        style={{
          padding: "var(--space-sm, 16px)",
          border: "var(--border-hairline, 1px) solid var(--color-border, #cbd5e1)",
          borderRadius: "var(--radius-control, .5rem)",
          background: "var(--color-surface-muted, #f1f5f9)",
        }}
      >
        <p style={{ margin: 0 }}>
          {rect ? `${Math.round(rect.width)} × ${Math.round(rect.height)}` : "Not observed"}
        </p>
      </div>
    </div>
  );
}

export function Default() {
  return <Rig />;
}

export function Disabled() {
  return <Rig disabled />;
}
