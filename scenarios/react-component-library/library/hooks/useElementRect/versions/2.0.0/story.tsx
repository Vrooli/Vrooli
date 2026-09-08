import { useRef } from "react";

import { useElementRect } from "./useElementRect";

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
          padding: "var(--space-sm)",
          border: "var(--border-hairline) solid var(--color-border)",
          borderRadius: "var(--radius-control)",
          background: "var(--color-surface-muted)",
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
