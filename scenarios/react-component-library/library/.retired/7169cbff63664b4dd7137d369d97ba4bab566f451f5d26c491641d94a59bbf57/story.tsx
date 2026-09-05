import { useState, type CSSProperties } from "react";
import { Draggable } from "./Draggable";

const shell: CSSProperties = {
  position: "relative",
  width: "min(100%, 36rem)",
  height: 300,
  overflow: "hidden",
  border: "1px solid var(--color-border, #cbd5e1)",
  borderRadius: "var(--radius-panel, 16px)",
  background: "var(--color-surface-muted, #f8fafc)",
};
const card: CSSProperties = {
  display: "grid",
  gap: 6,
  width: 190,
  padding: 16,
  border: "1px solid var(--color-border, #cbd5e1)",
  borderRadius: 12,
  background: "var(--color-surface, #fff)",
  boxShadow: "var(--elev-raised, 0 8px 20px rgb(15 23 42 / .1))",
};

function Tile({
  mode = "default",
}: {
  mode?: "default" | "keyboard" | "bounds" | "disabled";
}) {
  const [position, setPosition] = useState({
    x: mode === "bounds" ? 80 : 24,
    y: mode === "bounds" ? 80 : 24,
  });
  return (
    <div style={shell}>
      <Draggable
        id="brief-card"
        label="Project brief"
        defaultPosition={position}
        position={mode === "bounds" ? undefined : position}
        onPositionChange={setPosition}
        bounds={
          mode === "bounds"
            ? { left: 8, right: 170, top: 8, bottom: 170 }
            : undefined
        }
        disabled={mode === "disabled"}
        onCancel={() => setPosition({ x: 24, y: 24 })}
      >
        <div style={card}>
          <strong>Project brief</strong>
          <span
            style={{
              color: "var(--color-muted-foreground, #64748b)",
              fontSize: 12,
            }}
          >
            {mode === "keyboard"
              ? "Focused: Space to pick up, arrows to move"
              : "Drag by pointer or use the keyboard"}
          </span>
          <span
            style={{
              color: "var(--color-primary, #2563eb)",
              fontSize: 12,
              fontWeight: 700,
            }}
          >
            {Math.round(position.x)}, {Math.round(position.y)}
          </span>
        </div>
      </Draggable>
    </div>
  );
}
export function Default() {
  return <Tile />;
}
export function Keyboard() {
  return <Tile mode="keyboard" />;
}
export function Bounds() {
  return <Tile mode="bounds" />;
}
export function Disabled() {
  return <Tile mode="disabled" />;
}
