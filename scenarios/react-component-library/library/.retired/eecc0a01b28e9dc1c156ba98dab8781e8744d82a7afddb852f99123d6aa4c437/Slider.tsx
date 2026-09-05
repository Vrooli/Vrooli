/**
 * @libraryId react-component-library:Slider
 * @displayName Slider
 * @description A token-bound range control with native semantics, separated live and committed change, and optional ticks, marks, and a default marker.
 * @version 1.1.1
 * @tags ["controls","input","accessibility","gesture","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource controls.slider */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import {
  useEffect,
  useId,
  useRef,
  useState,
  type ChangeEvent,
  type FocusEvent,
  type InputHTMLAttributes,
  type KeyboardEvent,
  type PointerEvent as ReactPointerEvent,
  type ReactNode,
} from "react";

/** A labelled position on the track. Distinct from a tick: a mark carries words. */
export interface SliderMark {
  value: number;
  label: ReactNode;
}

export type SliderValueDisplay = "none" | "inline" | "tooltip";

export interface SliderProps
  extends Omit<
    InputHTMLAttributes<HTMLInputElement>,
    "children" | "onChange" | "type" | "value" | "defaultValue" | "min" | "max" | "step"
  > {
  value?: number;
  defaultValue?: number;
  /**
   * Fires for every movement, including each frame of a drag. Drive on-screen
   * state from this.
   */
  onChange?: (value: number) => void;
  /**
   * Fires once the interaction ends — pointer release, key release, or blur.
   * Anything that persists, requests, or writes to a store belongs here; using
   * `onChange` for that turns one drag into dozens of writes.
   */
  onChangeCommit?: (value: number) => void;
  min?: number;
  max?: number;
  step?: number | "any";
  /**
   * Renders the value and, through `aria-valuetext`, speaks it. Without this a
   * screen reader announces "250" where the screen says "250 ms".
   */
  formatValue?: (value: number) => string;
  showValue?: SliderValueDisplay;
  /**
   * Off by default: a forty-step slider with forty dots is noise. A number is a
   * spacing; an array is explicit positions.
   */
  ticks?: number | number[];
  marks?: SliderMark[];
  /** Draws a notch at the value the control started from, so drift is visible. */
  defaultMarker?: number;
  /** Omit for a bare control and supply `aria-label`; a host row may own the label. */
  label?: ReactNode;
  description?: ReactNode;
  invalid?: boolean;
  testId?: string;
}

const DEFAULT_MIN = 0;
const DEFAULT_MAX = 100;
const DEFAULT_STEP = 1;

/** Multiplier applied to the step when a shift-modified arrow key is used. */
const COARSE_STEP_MULTIPLIER = 10;

const styleSheet = `
[data-rcl-slider] {
  --rcl-slider-thumb: 1.25rem;
  --rcl-slider-track: .3125rem;
  display: grid;
  gap: var(--space-3xs);
  inline-size: 100%;
  min-inline-size: 0;
  color: var(--color-foreground);
}
[data-rcl-slider][data-disabled="true"] { opacity: max(var(--opacity-disabled), .72); }

[data-rcl-slider-label] { font: var(--text-body); font-weight: 650; }
[data-rcl-slider-description] { font: var(--text-caption); color: var(--color-muted-foreground); }

/* The control row: the track claims the free space, the readout keeps its own. */
[data-rcl-slider-row] {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  align-items: center;
  gap: var(--space-sm);
}
[data-rcl-slider][data-show-value="inline"] [data-rcl-slider-row] { grid-template-columns: minmax(0, 1fr) auto; }

[data-rcl-slider-area] {
  position: relative;
  display: grid;
  align-items: center;
  min-block-size: var(--tap-target-min);
  min-inline-size: 0;
  cursor: pointer;
  /* The area owns every horizontal gesture that starts on it. See the note on
     the input below for why the native control no longer handles pointers. */
  touch-action: none;
  -webkit-tap-highlight-color: transparent;
}
[data-rcl-slider][data-disabled="true"] [data-rcl-slider-area] { cursor: not-allowed; touch-action: auto; }

[data-rcl-slider-track] {
  position: relative;
  block-size: var(--rcl-slider-track);
  border-radius: var(--radius-pill);
  background: var(--color-surface-muted);
  border: var(--border-hairline) solid var(--color-border);
}
[data-rcl-slider-fill] {
  position: absolute;
  inset-block: 0;
  inset-inline-start: 0;
  /* The thumb's centre travels between half a thumb from each end, so the fill
     tracks that centre rather than a naive percentage of the full width. */
  inline-size: calc(var(--rcl-slider-thumb) / 2 + var(--rcl-slider-pct, 0) * (100% - var(--rcl-slider-thumb)));
  border-radius: var(--radius-pill);
  background: var(--color-primary);
}
[data-rcl-slider][data-invalid="true"] [data-rcl-slider-fill] { background: var(--color-danger); }

[data-rcl-slider-thumb] {
  position: absolute;
  inset-block-start: 50%;
  inset-inline-start: calc(var(--rcl-slider-thumb) / 2 + var(--rcl-slider-pct, 0) * (100% - var(--rcl-slider-thumb)));
  inline-size: var(--rcl-slider-thumb);
  block-size: var(--rcl-slider-thumb);
  box-sizing: border-box;
  border-radius: var(--radius-pill);
  background: var(--color-surface);
  border: var(--border-strong) solid var(--color-primary);
  box-shadow: var(--elev-raised);
  transform: translate(-50%, -50%);
  transition: box-shadow var(--dur-quick) var(--ease-standard);
}
[data-rcl-slider][data-invalid="true"] [data-rcl-slider-thumb] { border-color: var(--color-danger); }
[data-rcl-slider][data-active="true"] [data-rcl-slider-thumb] {
  box-shadow: 0 0 0 var(--space-2xs) color-mix(in srgb, var(--color-primary) 18%, transparent), var(--elev-raised);
}

[data-rcl-slider-tick] {
  position: absolute;
  inset-block-start: 50%;
  inline-size: var(--border-strong);
  block-size: var(--border-strong);
  border-radius: var(--radius-pill);
  background: var(--color-muted-foreground);
  opacity: .55;
  transform: translate(-50%, -50%);
}
/* Taller and opaque: the origin is a reference point, not a step. */
[data-rcl-slider-origin] {
  position: absolute;
  inset-block-start: 50%;
  inline-size: var(--border-strong);
  block-size: .7rem;
  border-radius: var(--radius-pill);
  background: var(--color-muted-foreground);
  transform: translate(-50%, -50%);
}

[data-rcl-slider-marks] {
  position: relative;
  block-size: 1rem;
  font: var(--text-caption);
  color: var(--color-muted-foreground);
}
[data-rcl-slider-mark] { position: absolute; transform: translateX(-50%); white-space: nowrap; }

[data-rcl-slider-readout] {
  font: var(--text-caption);
  font-variant-numeric: tabular-nums;
  color: var(--color-foreground);
  text-align: end;
}

[data-rcl-slider-tooltip] {
  position: absolute;
  inset-block-end: calc(50% + var(--rcl-slider-thumb));
  inset-inline-start: calc(var(--rcl-slider-thumb) / 2 + var(--rcl-slider-pct, 0) * (100% - var(--rcl-slider-thumb)));
  transform: translateX(-50%);
  padding: var(--space-3xs) var(--space-2xs);
  border-radius: var(--radius-control);
  background: var(--color-shell);
  color: var(--color-surface);
  font: var(--text-caption);
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
  pointer-events: none;
  opacity: 0;
  transition: opacity var(--dur-quick) var(--ease-standard);
}
[data-rcl-slider][data-active="true"] [data-rcl-slider-tooltip] { opacity: 1; }

/* The native control keeps what is expensive and risky to re-implement —
   keyboard stepping, form participation, and assistive-technology semantics —
   but no longer handles pointers. Its own drag only engages when the press
   lands on the native thumb, a moving ~20px target, so grabbing the thumb
   worked only some of the time. The area above handles the gesture instead:
   a press anywhere moves the thumb to the pointer and it tracks from there. */
[data-rcl-slider-input] {
  pointer-events: none;
  position: absolute;
  inset-block: 0;
  inset-inline: 0;
  inline-size: 100%;
  block-size: 100%;
  margin: 0;
  padding: 0;
  opacity: 0;
  background: transparent;
  -webkit-appearance: none;
  appearance: none;
}
[data-rcl-slider-input]::-webkit-slider-runnable-track { block-size: 100%; background: transparent; }
[data-rcl-slider-input]::-webkit-slider-thumb {
  -webkit-appearance: none;
  inline-size: var(--rcl-slider-thumb);
  block-size: var(--rcl-slider-thumb);
  border-radius: var(--radius-pill);
  background: transparent;
}
[data-rcl-slider-input]::-moz-range-track { block-size: 100%; background: transparent; }
[data-rcl-slider-input]::-moz-range-thumb {
  inline-size: var(--rcl-slider-thumb);
  block-size: var(--rcl-slider-thumb);
  border: 0;
  border-radius: var(--radius-pill);
  background: transparent;
}

@media (prefers-reduced-motion: reduce) {
  [data-rcl-slider-thumb], [data-rcl-slider-tooltip] { transition: none; }
}
`;

function SliderStyles() {
  return <StyleSheet name="slider-1-1-0-1" css={styleSheet} />;
}

/** Decimal places implied by a step, so a snapped value doesn't accrue float dust. */
function decimalsOf(step: number): number {
  const text = String(step);
  const dot = text.indexOf(".");
  return dot < 0 ? 0 : text.length - dot - 1;
}

/** Positions for a `ticks` prop expressed as a spacing rather than a list. */
function ticksFromSpacing(min: number, max: number, spacing: number): number[] {
  if (!Number.isFinite(spacing) || spacing <= 0) return [];
  const out: number[] = [];
  // Guard against a spacing so fine it would emit a dot per pixel.
  const count = Math.floor((max - min) / spacing);
  if (count > 200) return [];
  for (let i = 0; i <= count; i += 1) out.push(min + i * spacing);
  return out;
}

export const Slider = withClassName(function Slider({
  value,
  defaultValue,
  onChange,
  onChangeCommit,
  min = DEFAULT_MIN,
  max = DEFAULT_MAX,
  step = DEFAULT_STEP,
  formatValue,
  showValue = "inline",
  ticks,
  marks,
  defaultMarker,
  label,
  description,
  disabled,
  invalid,
  id: providedId,
  testId,
  onKeyDown,
  onBlur,
  // A host row that renders its own label passes its id; the component's own
  // label wins when there is one, but it must never erase the host's.
  "aria-labelledby": ariaLabelledBy,
  ...inputProps
}: SliderProps) {
  const generatedId = useId();
  const id = providedId ?? `rcl-slider-${generatedId.replace(/:/g, "")}`;
  const isControlled = value !== undefined;
  const [internalValue, setInternalValue] = useState(defaultValue ?? min);
  const resolved = isControlled ? value : internalValue;
  const [active, setActive] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const trackRef = useRef<HTMLDivElement>(null);
  const thumbRef = useRef<HTMLSpanElement>(null);
  const detachRef = useRef<(() => void) | null>(null);
  /** Set by any movement, cleared by the commit it causes. */
  const dirtyRef = useRef(false);
  const latestRef = useRef(resolved);
  latestRef.current = resolved;

  const labelId = label != null ? `${id}-label` : undefined;
  const descriptionId = description != null ? `${id}-description` : undefined;

  const span = max - min;
  const fraction = span > 0 ? Math.min(1, Math.max(0, (resolved - min) / span)) : 0;
  const display = formatValue ? formatValue(resolved) : String(resolved);
  const numericStep = step === "any" ? undefined : step;

  const tickValues = Array.isArray(ticks)
    ? ticks
    : typeof ticks === "number"
      ? ticksFromSpacing(min, max, ticks)
      : [];

  const positionOf = (at: number) => (span > 0 ? Math.min(1, Math.max(0, (at - min) / span)) : 0);
  /** Mirrors the fill/thumb geometry so a tick lines up with the thumb over it. */
  const offsetStyle = (at: number) => ({
    insetInlineStart: `calc(var(--rcl-slider-thumb) / 2 + ${positionOf(at)} * (100% - var(--rcl-slider-thumb)))`,
  });

  const emit = (next: number) => {
    dirtyRef.current = true;
    latestRef.current = next;
    if (!isControlled) setInternalValue(next);
    onChange?.(next);
  };

  const commit = () => {
    if (!dirtyRef.current) return;
    dirtyRef.current = false;
    onChangeCommit?.(latestRef.current);
  };

  const handleChange = (event: ChangeEvent<HTMLInputElement>) => {
    emit(event.target.valueAsNumber);
  };

  /**
   * The value implied by a pointer at `clientX`.
   *
   * Mirrors the paint: the thumb's centre travels between half a thumb from
   * each end, so the usable span is the track minus one thumb.
   */
  const valueAt = (clientX: number): number => {
    const track = trackRef.current;
    const thumb = thumbRef.current;
    if (!track) return latestRef.current;
    const rect = track.getBoundingClientRect();
    const thumbSize = thumb ? thumb.getBoundingClientRect().width : 0;
    const usable = rect.width - thumbSize;
    if (usable <= 0) return latestRef.current;
    let ratio = (clientX - rect.left - thumbSize / 2) / usable;
    if (getComputedStyle(track).direction === "rtl") ratio = 1 - ratio;
    ratio = Math.min(1, Math.max(0, ratio));
    const raw = min + ratio * span;
    if (step === "any") return raw;
    const places = decimalsOf(step);
    const snapped = Math.round((raw - min) / step) * step + min;
    return Math.min(max, Math.max(min, Number.parseFloat(snapped.toFixed(places))));
  };

  const detachWindow = () => {
    detachRef.current?.();
    detachRef.current = null;
  };

  useEffect(() => () => detachWindow(), []);

  /**
   * A press anywhere on the area — track, thumb, or the padding around them —
   * moves the thumb to the pointer and begins tracking. The pointer stream is
   * taken from the window so a drag that leaves the control still finishes.
   */
  const onAreaPointerDown = (event: ReactPointerEvent<HTMLDivElement>) => {
    if (disabled || event.button !== 0) return;
    event.preventDefault();
    inputRef.current?.focus();
    setActive(true);
    emit(valueAt(event.clientX));

    const move = (e: globalThis.PointerEvent) => {
      if (e.pointerId !== event.pointerId) return;
      emit(valueAt(e.clientX));
    };
    const end = (e: globalThis.PointerEvent) => {
      if (e.pointerId !== event.pointerId) return;
      detachWindow();
      setActive(false);
      // A cancelled drag keeps the value it already applied; the interaction is
      // simply over, and there is nothing to revert to that the person expects.
      commit();
    };
    detachWindow();
    window.addEventListener("pointermove", move);
    window.addEventListener("pointerup", end);
    window.addEventListener("pointercancel", end);
    detachRef.current = () => {
      window.removeEventListener("pointermove", move);
      window.removeEventListener("pointerup", end);
      window.removeEventListener("pointercancel", end);
    };
  };

  const handleKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    onKeyDown?.(event);
    if (event.defaultPrevented || disabled) return;
    // Shift-arrow for a coarse step is conventional but not native.
    if (!event.shiftKey || numericStep === undefined) return;
    const direction =
      event.key === "ArrowRight" || event.key === "ArrowUp"
        ? 1
        : event.key === "ArrowLeft" || event.key === "ArrowDown"
          ? -1
          : 0;
    if (direction === 0) return;
    event.preventDefault();
    const raw = latestRef.current + direction * numericStep * COARSE_STEP_MULTIPLIER;
    emit(Math.min(max, Math.max(min, raw)));
  };

  const handleBlur = (event: FocusEvent<HTMLInputElement>) => {
    onBlur?.(event);
    setActive(false);
    commit();
  };

  return (
    <>
      <SliderStyles data-testid="controls.slider" />
      <div
        data-rcl-slider
        data-testid={testId}
        data-disabled={disabled ? "true" : "false"}
        data-invalid={invalid ? "true" : "false"}
        data-active={active ? "true" : "false"}
        data-show-value={showValue}
        style={{ ["--rcl-slider-pct" as string]: String(fraction) }}
      >
        {label != null && (
          <label id={labelId} data-rcl-slider-label htmlFor={id}>
            {label}
          </label>
        )}
        {description != null && (
          <span id={descriptionId} data-rcl-slider-description>
            {description}
          </span>
        )}

        <div data-rcl-slider-row>
          <div data-rcl-slider-area onPointerDown={onAreaPointerDown}>
            <div data-rcl-slider-track ref={trackRef}>
              <div data-rcl-slider-fill />
              {defaultMarker !== undefined && (
                <span
                  data-rcl-slider-origin
                  aria-hidden="true"
                  style={offsetStyle(defaultMarker)}
                />
              )}
              {tickValues.map((at) => (
                <span key={at} data-rcl-slider-tick aria-hidden="true" style={offsetStyle(at)} />
              ))}
              <span data-rcl-slider-thumb aria-hidden="true" ref={thumbRef} />
            </div>
            {showValue === "tooltip" && (
              <span data-rcl-slider-tooltip aria-hidden="true">
                {display}
              </span>
            )}
            <input
              {...inputProps}
              ref={inputRef}
              id={id}
              type="range"
              data-rcl-slider-input
              min={min}
              max={max}
              step={step}
              value={resolved}
              disabled={disabled}
              aria-labelledby={labelId ?? ariaLabelledBy}
              aria-describedby={descriptionId}
              aria-valuetext={display}
              aria-invalid={invalid ? true : undefined}
              onChange={handleChange}
              onKeyDown={handleKeyDown}
              onKeyUp={commit}
              onFocus={() => setActive(true)}
              onBlur={handleBlur}
            />
          </div>
          {showValue === "inline" && <span data-rcl-slider-readout>{display}</span>}
        </div>

        {marks && marks.length > 0 && (
          <div data-rcl-slider-marks aria-hidden="true">
            {marks.map((mark) => (
              <span key={mark.value} data-rcl-slider-mark style={offsetStyle(mark.value)}>
                {mark.label}
              </span>
            ))}
          </div>
        )}
      </div>
    </>
  );
});
