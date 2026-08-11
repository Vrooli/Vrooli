import { useState } from "react";
import { useHover } from "./useHover";

const frame = {
  display: "grid",
  gap: "var(--space-md)",
  width: "min(100%, 620px)",
  minWidth: 0,
  boxSizing: "border-box",
  padding: "var(--space-xl)",
  border: "var(--border-hairline) solid var(--color-border)",
  borderRadius: "var(--radius-panel)",
  background: "var(--color-surface-raised)",
  boxShadow: "var(--elev-raised)",
} as const;

function Demo() {
  const relatedRef: { current: HTMLDivElement | null } = { current: null };
  const [events, setEvents] = useState<string[]>([]);
  const hover = useHover({
    enterDelay: 0,
    exitDelay: 0,
    relatedRefs: [relatedRef],
    onChange: (next) =>
      setEvents((previous) => [
        ...previous.slice(-2),
        next ? "entered" : "left",
      ]),
  });
  return (
    <div style={{ display: "grid", gap: "var(--space-sm)" }}>
      <button
        type="button"
        {...hover.triggerProps}
        style={{
          minBlockSize: "var(--tap-target-min)",
          padding: "var(--space-sm) var(--space-md)",
          border: "var(--border-hairline) solid var(--color-primary)",
          borderRadius: "var(--radius-control)",
          background: hover.hovered ? "var(--color-primary)" : "transparent",
          color: hover.hovered
            ? "var(--color-primary-foreground)"
            : "var(--color-primary)",
          font: "var(--text-label)",
          cursor: "pointer",
        }}
      >
        Hover or focus preview
      </button>
      <div
        ref={relatedRef}
        {...hover.floatingProps}
        role="status"
        style={{
          padding: "var(--space-md)",
          border: "var(--border-hairline) solid var(--color-border)",
          borderRadius: "var(--radius-panel)",
          background: "var(--color-surface-muted)",
          color: "var(--color-foreground)",
          opacity: hover.hovered ? 1 : 0.72,
        }}
      >
        <strong>
          {hover.hovered ? "Intent acquired" : "Waiting for hover intent"}
        </strong>
        <span
          style={{
            display: "block",
            marginBlockStart: "var(--space-2xs)",
            color: "var(--color-muted-foreground)",
            font: "var(--text-body-sm)",
          }}
        >
          Touch and pen pointers never open this surface from hover.
        </span>
      </div>
      <span
        style={{
          color: "var(--color-muted-foreground)",
          font: "var(--text-caption)",
        }}
      >
        Pointer: {hover.pointerType ?? "none"} · Events:{" "}
        {events.join(" → ") || "none yet"}
      </span>
    </div>
  );
}

export function Default() {
  return (
    <section style={frame}>
      <div style={{ display: "grid", gap: "var(--space-2xs)" }}>
        <span
          style={{
            color: "var(--color-primary)",
            font: "var(--text-overline)",
            letterSpacing: ".1em",
            textTransform: "uppercase",
          }}
        >
          Hover intent
        </span>
        <strong style={{ font: "var(--text-title)" }}>
          A pointer contract with an accessible focus path
        </strong>
        <span
          style={{
            color: "var(--color-muted-foreground)",
            font: "var(--text-body)",
          }}
        >
          Hover opens after intent; keyboard focus remains an explicit
          alternative, and moving into the related surface does not close it.
        </span>
      </div>
      <Demo />
    </section>
  );
}
