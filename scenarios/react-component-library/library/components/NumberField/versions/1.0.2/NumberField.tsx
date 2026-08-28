/**
 * @libraryId react-component-library:NumberField
 * @displayName Number Field
 * @description Bounded numeric input with joined steppers, a bound unit suffix, and draft-then-commit editing.
 * @version 1.0.2
 * @tags ["form","control","token-bound","numeric"]
 * @deps {"react":"^18"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:NumberField */
import { forwardRef, useEffect, useState, type CSSProperties, type ReactNode } from "react";
import {
  InputGroup,
  type InputGroupShape,
  type InputGroupSize,
} from "@vrooli/react-component-library/InputGroup/1.2.0";

const MinusGlyph = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" aria-hidden>
    <path d="M5 12h14" />
  </svg>
);

const PlusGlyph = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" aria-hidden>
    <path d="M5 12h14" />
    <path d="M12 5v14" />
  </svg>
);

/** Bounds first, so a caller cannot express a range the field cannot honour. */
export function clampToRange(value: number, min: number, max: number): number {
  if (!Number.isFinite(value)) return min;
  return Math.min(max, Math.max(min, value));
}

/**
 * Decimal places implied by the step, so `0.1` commits `1.3` rather than
 * `1.3000000000000003` after a few increments of binary floating point.
 */
function roundToStep(value: number, step: number): number {
  const decimals = (String(step).split(".")[1] ?? "").length;
  return decimals === 0 ? Math.round(value) : Number(value.toFixed(decimals));
}

export interface NumberFieldProps {
  value: number;
  onChange: (next: number) => void;
  /** Accessible name for the value itself, not for the group around it. */
  label: string;
  min?: number;
  max?: number;
  step?: number;
  /** Rendered as a suffix bound to the value — `px`, `s`, `%`. */
  unit?: ReactNode;
  decreaseLabel?: string;
  increaseLabel?: string;
  decreaseIcon?: ReactNode;
  increaseIcon?: ReactNode;
  shape?: InputGroupShape;
  size?: InputGroupSize;
  disabled?: boolean;
  className?: string;
  style?: CSSProperties;
  testId?: string;
}

/**
 * A bounded number with its steppers and its unit inside one field.
 *
 * Three behaviours are the reason this exists as a component rather than as
 * three slots a caller wires up each time:
 *
 *  - **Both bounds are enforced.** Hand-rolled numeric settings reliably clamp
 *    the floor and forget the ceiling, because the floor is the one you hit
 *    while testing.
 *  - **Typing edits a draft; the value commits on blur or Enter.** Committing
 *    per keystroke means a half-typed `1` in a field bounded at 8 immediately
 *    becomes `8`, and the digit you were about to type lands on the clamp.
 *  - **A stepper at its bound disables itself** rather than clamping silently,
 *    so the limit is visible before it is hit.
 */
export const NumberField = forwardRef<HTMLInputElement, NumberFieldProps>(function NumberField(
  {
    className,
    decreaseIcon,
    decreaseLabel = "Decrease",
    disabled = false,
    increaseIcon,
    increaseLabel = "Increase",
    label,
    max = Number.MAX_SAFE_INTEGER,
    min = 0,
    onChange,
    shape = "rounded",
    size = "md",
    step = 1,
    style,
    testId,
    unit,
    value,
  },
  ref,
) {
  const committed = clampToRange(value, min, max);
  const [draft, setDraft] = useState(() => String(committed));

  // The draft mirrors the committed value whenever the value changes from
  // outside — a reset button, a preset, another surface editing the same
  // setting. It deliberately does not run on every draft keystroke.
  useEffect(() => {
    setDraft(String(committed));
  }, [committed]);

  const commit = (raw: string) => {
    // An empty or non-numeric draft reverts rather than resolving to a number.
    // `Number("")` is 0, so an empty field would otherwise commit the floor
    // and silently discard what the caller had.
    const trimmed = raw.trim();
    const parsed = trimmed === "" ? Number.NaN : Number(trimmed);
    if (!Number.isFinite(parsed)) {
      setDraft(String(committed));
      return;
    }
    const next = clampToRange(roundToStep(parsed, step), min, max);
    setDraft(String(next));
    if (next !== committed) onChange(next);
  };

  const nudge = (direction: 1 | -1) => {
    const next = clampToRange(roundToStep(committed + direction * step, step), min, max);
    if (next !== committed) onChange(next);
  };

  const atMin = committed <= min;
  const atMax = committed >= max;
  // Wide enough for the longest bound the field can display, so the box does
  // not resize as the value grows through an order of magnitude.
  const digits = Math.max(String(min).length, String(max === Number.MAX_SAFE_INTEGER ? 9999 : max).length);

  return (
    <InputGroup
      className={className}
      disabled={disabled}
      shape={shape}
      size={size}
      style={{ inlineSize: "max-content", ...style }}
      testId={testId ?? "forms.number-field"}
    >
      <InputGroup.Segment
        side="leading"
        aria-label={decreaseLabel}
        disabled={disabled || atMin}
        onClick={() => { nudge(-1); }}
        testId={testId ? `${testId}-decrease` : undefined}
      >
        {decreaseIcon ?? <MinusGlyph />}
      </InputGroup.Segment>

      <InputGroup.Field>
        <input
          ref={ref}
          data-rcl-input="true"
          data-testid={testId ? `${testId}-value` : "forms.number-field-value"}
          type="text"
          inputMode="numeric"
          // The spinbutton pattern, which a bare textbox cannot express: it
          // publishes the range to assistive technology and is why the arrow
          // keys below are an expectation rather than a nicety.
          role="spinbutton"
          aria-label={label}
          aria-valuenow={committed}
          aria-valuemin={min}
          aria-valuemax={max}
          aria-valuetext={typeof unit === "string" ? `${committed} ${unit}` : undefined}
          disabled={disabled}
          value={draft}
          onChange={(event) => { setDraft(event.target.value); }}
          onBlur={(event) => { commit(event.target.value); }}
          onKeyDown={(event) => {
            if (event.key === "Enter") {
              event.preventDefault();
              commit(event.currentTarget.value);
              return;
            }
            if (event.key === "ArrowUp") {
              event.preventDefault();
              nudge(1);
              return;
            }
            if (event.key === "ArrowDown") {
              event.preventDefault();
              nudge(-1);
            }
          }}
          style={{
            inlineSize: `${digits + 1}ch`,
            paddingInline: 0,
            textAlign: "center",
            fontVariantNumeric: "tabular-nums",
          }}
        />
        {unit === undefined ? null : (
          <InputGroup.Adornment
            side="trailing"
            style={{ paddingInlineStart: "var(--space-3xs, 4px)", fontSize: "var(--text-label-size, .75rem)" }}
            testId={testId ? `${testId}-unit` : undefined}
          >
            {unit}
          </InputGroup.Adornment>
        )}
      </InputGroup.Field>

      <InputGroup.Segment
        side="trailing"
        aria-label={increaseLabel}
        disabled={disabled || atMax}
        onClick={() => { nudge(1); }}
        testId={testId ? `${testId}-increase` : undefined}
      >
        {increaseIcon ?? <PlusGlyph />}
      </InputGroup.Segment>
    </InputGroup>
  );
});
