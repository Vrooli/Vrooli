import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import { useState, type CSSProperties } from "react";
import { Draggable } from "./Draggable";

const shell: CSSProperties = {
  position: "relative",
  width: "min(100%, 36rem)",
  height: 300,
  overflow: "hidden",
  border: "1px solid var(--color-border)",
  borderRadius: "var(--radius-panel)",
  background: "var(--color-surface-muted)",
};
const card: CSSProperties = {
  display: "grid",
  gap: 6,
  width: 190,
  padding: 16,
  border: "1px solid var(--color-border)",
  borderRadius: 12,
  background: "var(--color-surface)",
  boxShadow: "var(--elev-raised)",
};

function Tile({ mode = "default" }: { mode?: "default" | "keyboard" | "bounds" | "disabled" }) {
  const libraryStrings = useStrings();
  const [position, setPosition] = useState({
    x: mode === "bounds" ? 80 : 24,
    y: mode === "bounds" ? 80 : 24,
  });
  return (
    <div style={shell}>
      <Draggable
        id="brief-card"
        label={libraryStrings("manipulation.draggable.label", "Project brief")}
        defaultPosition={position}
        position={mode === "bounds" ? undefined : position}
        onPositionChange={setPosition}
        bounds={mode === "bounds" ? { left: 8, right: 170, top: 8, bottom: 170 } : undefined}
        disabled={mode === "disabled"}
        onCancel={() => setPosition({ x: 24, y: 24 })}
      >
        <div style={card}>
          <strong>{libraryStrings("manipulation.draggable.project-brief", "Project brief")}</strong>
          <span
            style={{
              color: "var(--color-muted-foreground)",
              fontSize: 12,
            }}
          >
            {mode === "keyboard"
              ? "Focused: Space to pick up, arrows to move"
              : "Drag by pointer or use the keyboard"}
          </span>
          <span
            style={{
              color: "var(--color-primary)",
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
