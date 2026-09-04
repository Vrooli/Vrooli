import { useState } from "react";

import { useDrag } from "./useDrag";

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

const target = {
  padding: "var(--space-sm, 16px)",
  border: "var(--border-hairline, 1px) solid var(--color-border, #cbd5e1)",
  borderRadius: "var(--radius-control, .5rem)",
  background: "var(--color-surface-muted, #f1f5f9)",
};

function Rig({ disabled = false }: { disabled?: boolean }) {
  const [offset, setOffset] = useState({ x: 0, y: 0 });
  const [cancelled, setCancelled] = useState(false);
  const { isDragging, ...handlers } = useDrag({
    disabled,
    step: 8,
    coarseStep: 48,
    onStart: () => setCancelled(false),
    onKeyboardMove: (dx, dy) => setOffset((current) => ({ x: current.x + dx, y: current.y + dy })),
    onCancel: () => setCancelled(true),
  });

  return (
    <div style={frame}>
      <div
        {...handlers}
        data-testid="hooks.use-drag"
        data-dragging={isDragging ? "true" : "false"}
        data-cancelled={cancelled ? "true" : "false"}
        data-offset-x={String(offset.x)}
        data-offset-y={String(offset.y)}
        role="group"
        aria-label="Drag target"
        aria-disabled={disabled || undefined}
        tabIndex={disabled ? -1 : 0}
        style={{ ...target, transform: `translate(${offset.x}px, ${offset.y}px)` }}
      >
        <p style={{ margin: 0 }}>Space to pick up, arrows to move, Enter to drop.</p>
        <p style={{ margin: "var(--space-2xs, 8px) 0 0" }}>{`x ${offset.x} · y ${offset.y}`}</p>
      </div>
    </div>
  );
}

export function Default() {
  return <Rig />;
}

export function Cancelled() {
  return <Rig />;
}

export function Disabled() {
  return <Rig disabled />;
}
