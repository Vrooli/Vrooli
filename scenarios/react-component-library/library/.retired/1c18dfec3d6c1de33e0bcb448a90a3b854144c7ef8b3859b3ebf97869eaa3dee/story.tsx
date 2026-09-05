import { useState } from "react";

import { NumberField } from "./NumberField";

const frame = { padding: "var(--space-sm, 16px)" };

function Rig({
  initial = 14,
  min = 8,
  max = 32,
  step = 1,
  unit = "px",
  disabled = false,
}: {
  initial?: number;
  min?: number;
  max?: number;
  step?: number;
  unit?: string;
  disabled?: boolean;
}) {
  const [value, setValue] = useState(initial);
  return (
    <div style={frame}>
      <NumberField
        testId="story-number"
        label="Font size"
        value={value}
        onChange={setValue}
        min={min}
        max={max}
        step={step}
        unit={unit}
        disabled={disabled}
      />
      <p
        data-testid="story-echo"
        style={{ marginBlockStart: "var(--space-2xs, 8px)" }}
      >
        committed {value}
      </p>
    </div>
  );
}

export function Default() {
  return <Rig />;
}

/** Pressing the joined stepper moves the value it is attached to. */
export function Increment() {
  return <Rig />;
}

/** The ARIA spinbutton pattern the bare text box could not express. */
export function ArrowKeys() {
  return <Rig />;
}

/** At the ceiling the increase segment disables rather than clamping silently. */
export function AtMaximum() {
  return <Rig initial={32} />;
}

/** At the floor the decrease segment disables. */
export function AtMinimum() {
  return <Rig initial={8} />;
}

/**
 * Typing past the ceiling commits the ceiling — the bound that hand-rolled
 * numeric settings reliably declare and then never enforce.
 */
export function TypedOverMaximum() {
  return <Rig />;
}

/** An emptied field reverts to the committed value instead of resolving to 0. */
export function TypedEmpty() {
  return <Rig />;
}

/** A fractional step commits decimal-clean values, not binary float residue. */
export function FractionalStep() {
  return <Rig initial={1} min={0.1} max={4} step={0.1} unit="x" />;
}
