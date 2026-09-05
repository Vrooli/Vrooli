import { useState, type ReactNode } from "react";
import { Slider } from "./Slider";

function Showcase({
  title,
  hint,
  children,
}: {
  title: string;
  hint: string;
  children: ReactNode;
}) {
  return (
    <section
      style={{
        boxSizing: "border-box",
        display: "grid",
        gap: "var(--space-md)",
        width: "min(100%, 560px)",
        padding: "var(--space-xl)",
        border: "1px solid var(--color-border)",
        borderRadius: "var(--radius-panel)",
        background: "var(--color-surface-raised)",
        boxShadow: "var(--elev-raised)",
      }}
    >
      <div style={{ display: "grid", gap: "var(--space-2xs)" }}>
        <span
          style={{
            color: "var(--color-primary)",
            font: "var(--text-overline)",
            letterSpacing: ".08em",
            textTransform: "uppercase",
          }}
        >
          Continuous setting
        </span>
        <strong style={{ font: "var(--text-title)" }}>{title}</strong>
        <span
          style={{
            color: "var(--color-muted-foreground)",
            font: "var(--text-body)",
          }}
        >
          {hint}
        </span>
      </div>
      {children}
    </section>
  );
}

export function Default() {
  const [value, setValue] = useState(1);
  return (
    <Showcase
      title="Touch scroll sensitivity"
      hint="Track, fill, thumb, and a formatted readout. Everything else is opt-in."
    >
      <Slider
        label="Touch scroll sensitivity"
        min={0.1}
        max={4}
        step={0.1}
        value={value}
        onChange={setValue}
        formatValue={(v) => v.toFixed(1)}
      />
    </Showcase>
  );
}

export function Ticked() {
  const [value, setValue] = useState(250);
  return (
    <Showcase
      title="Prediction latency threshold"
      hint="Ticks mark each step and the taller notch marks the value the setting started from."
    >
      <Slider
        label="Prediction latency threshold"
        description="Underline speculative characters only above this round-trip time."
        min={0}
        max={1000}
        step={50}
        ticks={50}
        defaultMarker={250}
        value={value}
        onChange={setValue}
        formatValue={(v) => `${v} ms`}
      />
    </Showcase>
  );
}

export function Marked() {
  const [value, setValue] = useState(1);
  return (
    <Showcase
      title="Speech rate"
      hint="Marks carry words where a bare number would not help."
    >
      <Slider
        label="Speech rate"
        min={0.5}
        max={2}
        step={0.25}
        ticks={0.25}
        defaultMarker={1}
        showValue="tooltip"
        marks={[
          { value: 0.5, label: "Slower" },
          { value: 1, label: "Normal" },
          { value: 2, label: "Faster" },
        ]}
        value={value}
        onChange={setValue}
        formatValue={(v) => `${v.toFixed(2)}x`}
      />
    </Showcase>
  );
}

export function Bare() {
  const [value, setValue] = useState(60);
  return (
    <Showcase
      title="Host-owned labelling"
      hint="A settings row already renders the label, so the control renders none of its own."
    >
      <div
        style={{
          display: "grid",
          gridTemplateColumns: "minmax(0, 1fr) 12rem",
          alignItems: "center",
          gap: "var(--space-sm)",
        }}
      >
        <span
          id="host-owned-slider-label"
          style={{ font: "var(--text-body)", fontWeight: 650 }}
        >
          Output volume
        </span>
        {/* The row owns the label, so the control adopts it by id rather than
            repeating the string — and must not discard it. */}
        <Slider
          aria-labelledby="host-owned-slider-label"
          value={value}
          onChange={setValue}
          formatValue={(v) => `${v}%`}
        />
      </div>
    </Showcase>
  );
}

export function Disabled() {
  return (
    <Showcase
      title="Unavailable"
      hint="A disabled slider keeps its value legible."
    >
      <Slider
        label="Output volume"
        defaultValue={40}
        disabled
        formatValue={(v) => `${v}%`}
      />
    </Showcase>
  );
}
