/**
 * @libraryId react-component-library:SelectionControl
 * @displayName SelectionControl
 * @description
 * @version 1.0.4
 * @tags ["primitive","selection","accessibility","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:SelectionControl */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.2";

import {
  useEffect,
  useId,
  useRef,
  useState,
  type ChangeEvent,
  type InputHTMLAttributes,
  type MouseEvent,
  type PointerEvent,
  type ReactNode,
} from "react";

export type SelectionControlKind = "checkbox" | "radio" | "switch";

/** Which side of the copy the indicator sits on. */
export type SelectionLabelPlacement = "start" | "end";

export interface SelectionControlProps
  extends Omit<InputHTMLAttributes<HTMLInputElement>, "children" | "onChange" | "type"> {
  kind: SelectionControlKind;
  /**
   * Omit to render a bare control. Surfaces that already own their labelling —
   * a settings row, a table cell, a toolbar — supply `aria-label` instead, and
   * rendering a second label here would duplicate it on the wrong side.
   */
  label?: ReactNode;
  /**
   * Which edge the indicator sits on. Settings rows conventionally put the
   * control on the trailing edge; forms put it on the leading edge.
   */
  labelPlacement?: SelectionLabelPlacement;
  description?: ReactNode;
  error?: ReactNode;
  onCheckedChange?: (checked: boolean) => void;
  onChange?: (event: ChangeEvent<HTMLInputElement>) => void;
  indeterminate?: boolean;
}

/**
 * Pointer travel, in px, before a press on a switch is treated as a drag rather
 * than a tap. Below this the thumb stays put so a slightly-shaky tap still
 * reads as a tap.
 */
const AXIS_SLOP = 4;

/**
 * The thumb's travel as a fraction of the track's width. Mirrors the geometry
 * declared on `--rcl-switch-*` below: (2.5 - 1.1 - 2 x 0.2) / 2.5. Expressed as
 * a ratio rather than a rem constant so the gesture stays correct at any root
 * font size.
 */
const SWITCH_TRAVEL_RATIO = 0.4;

/**
 * How long the post-drag click swallow stays armed. Long enough to cover the
 * click the label is about to synthesize, short enough that it can never reach
 * a subsequent tap.
 */
const CLICK_SUPPRESSION_MS = 350;

const styleSheet = `
[data-rcl-selection-row] {
  position: relative;
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  align-items: start;
  gap: var(--space-sm);
  min-block-size: var(--tap-target-min);
  padding: var(--space-xs) var(--space-sm);
  border: var(--border-hairline) solid transparent;
  border-radius: var(--radius-control);
  color: var(--color-foreground);
  cursor: pointer;
  transition: background-color var(--dur-quick) var(--ease-standard), border-color var(--dur-quick) var(--ease-standard), box-shadow var(--dur-quick) var(--ease-standard);
}
[data-rcl-selection-row]:hover:not([data-disabled="true"]) {
  border-color: var(--color-border);
  background: color-mix(in srgb, var(--color-primary) 5%, transparent);
}
[data-rcl-selection-row]:focus-within {
  border-color: var(--color-focus);
  box-shadow: 0 0 0 var(--space-3xs) color-mix(in srgb, var(--color-focus) 18%, transparent);
}
[data-rcl-selection-row][data-disabled="true"] { cursor: not-allowed; opacity: max(var(--opacity-disabled), .72); }

/* The native control carries every semantic; the indicator is its paint. */
[data-rcl-selection-input] {
  position: absolute;
  inline-size: 1px;
  block-size: 1px;
  margin: -1px;
  overflow: hidden;
  clip: rect(0 0 0 0);
  clip-path: inset(50%);
  white-space: nowrap;
}

/* Trailing indicator: the copy claims the free column and the indicator moves
   after it in the grid without changing DOM order. */
[data-rcl-selection-row][data-label-placement="end"] { grid-template-columns: minmax(0, 1fr) auto; }
[data-rcl-selection-row][data-label-placement="end"] [data-rcl-selection-indicator] { order: 2; }
[data-rcl-selection-row][data-kind="switch"][data-label-placement="end"] { grid-template-columns: minmax(0, 1fr) var(--rcl-switch-w); }

/* Bare: no copy to align against, so the row is only a tap target. */
[data-rcl-selection-row][data-bare="true"] {
  display: inline-grid;
  grid-template-columns: auto;
  place-items: center;
  min-inline-size: var(--tap-target-min);
  padding: 0;
}
[data-rcl-selection-row][data-bare="true"] [data-rcl-selection-indicator] { margin-block-start: 0; }
[data-rcl-selection-row][data-bare="true"]:hover:not([data-disabled="true"]) { border-color: transparent; background: transparent; }

[data-rcl-selection-indicator] {
  position: relative;
  display: grid;
  flex: none;
  place-items: center;
  inline-size: 1.25rem;
  block-size: 1.25rem;
  margin-block-start: calc((var(--tap-target-min) - 1.25rem) / 2);
  border: var(--border-strong) solid var(--color-border);
  border-radius: var(--radius-control);
  background: var(--color-surface);
  color: var(--color-primary-foreground);
  transition: background-color var(--dur-quick) var(--ease-standard), border-color var(--dur-quick) var(--ease-standard), transform var(--dur-quick) var(--ease-standard), box-shadow var(--dur-quick) var(--ease-standard);
}
[data-rcl-selection-indicator]::after {
  content: "";
  display: block;
  inline-size: .32rem;
  block-size: .62rem;
  border: solid currentColor;
  border-width: 0 var(--border-strong) var(--border-strong) 0;
  opacity: 0;
  transform: translateY(-.06rem) rotate(45deg) scale(.7);
  transition: opacity var(--dur-quick) var(--ease-standard), transform var(--dur-quick) var(--ease-standard);
}
[data-rcl-selection-row][data-kind="radio"] [data-rcl-selection-indicator] { border-radius: var(--radius-pill); }
[data-rcl-selection-row][data-kind="radio"] [data-rcl-selection-indicator]::after {
  inline-size: .42rem;
  block-size: .42rem;
  border: 0;
  border-radius: var(--radius-pill);
  background: currentColor;
  transform: scale(.4);
}

/* ---- switch geometry -------------------------------------------------------
   One declaration of the track, thumb, and inset; travel is derived from them
   so the thumb always lands flush against either end. SWITCH_TRAVEL_RATIO in
   the module above mirrors this arithmetic for the drag gesture. */
/* The tick belongs to checkbox and radio. A switch says on or off by the
   position of its thumb, and a mark floating on the track reads as a defect. */
[data-rcl-selection-row][data-kind="switch"] [data-rcl-selection-indicator]::after { content: none; }

[data-rcl-selection-row][data-kind="switch"] {
  --rcl-switch-w: 2.5rem;
  --rcl-switch-h: 1.5rem;
  --rcl-switch-thumb: 1.1rem;
  --rcl-switch-inset: .2rem;
  --rcl-switch-travel: calc(var(--rcl-switch-w) - var(--rcl-switch-thumb) - 2 * var(--rcl-switch-inset));
  --rcl-switch-stretch: 1.45rem;
  grid-template-columns: calc(var(--rcl-switch-w) + var(--space-3xs)) minmax(0, 1fr);
}
[data-rcl-selection-row][data-kind="switch"][data-bare="true"] { grid-template-columns: auto; }
[data-rcl-selection-row][data-kind="switch"] [data-rcl-selection-indicator] {
  inline-size: var(--rcl-switch-w);
  block-size: var(--rcl-switch-h);
  margin-block-start: calc((var(--tap-target-min) - var(--rcl-switch-h)) / 2);
  border-radius: var(--radius-pill);
  background: var(--color-surface-muted);
  /* Deliberately "none" rather than "pan-y". "pan-y" only asks the browser to prefer vertical
     panning, and on a target this small it takes every drag that drifts even
     slightly off-axis — cancelling the gesture mid-flight. Owning the target
     outright costs the ability to scroll the page from the switch itself;
     the rest of the row still scrolls. */
  touch-action: none;
}
[data-rcl-selection-row][data-kind="switch"] [data-rcl-selection-indicator]::before {
  content: "";
  position: absolute;
  inset-block-start: var(--rcl-switch-inset);
  inset-inline-start: var(--rcl-switch-inset);
  inline-size: var(--rcl-switch-thumb);
  block-size: var(--rcl-switch-thumb);
  border-radius: var(--radius-pill);
  background: var(--color-muted-foreground);
  box-shadow: var(--elev-raised);
  transform: translateX(var(--rcl-switch-x, 0px));
  transition:
    background-color var(--dur-quick) var(--ease-standard),
    inline-size var(--dur-quick) var(--ease-standard),
    transform var(--dur-moderate) var(--ease-spring, cubic-bezier(.34, 1.4, .64, 1));
}
[data-rcl-selection-row][data-state="checked"] [data-rcl-selection-indicator],
[data-rcl-selection-row][data-state="mixed"] [data-rcl-selection-indicator] {
  border-color: var(--color-primary);
  background: var(--color-primary);
  box-shadow: 0 0 0 var(--space-3xs) color-mix(in srgb, var(--color-primary) 14%, transparent);
}
[data-rcl-selection-row][data-state="checked"] [data-rcl-selection-indicator]::after,
[data-rcl-selection-row][data-state="mixed"] [data-rcl-selection-indicator]::after { opacity: 1; transform: translateY(-.06rem) rotate(45deg) scale(1); }
[data-rcl-selection-row][data-state="mixed"] [data-rcl-selection-indicator]::after {
  inline-size: .62rem;
  block-size: var(--border-strong);
  border: 0;
  border-radius: var(--radius-pill);
  background: currentColor;
  transform: scale(1);
}
[data-rcl-selection-row][data-kind="radio"][data-state="checked"] [data-rcl-selection-indicator]::after { transform: scale(1); }
[data-rcl-selection-row][data-kind="switch"][data-state="checked"] [data-rcl-selection-indicator]::before {
  background: var(--color-primary-foreground);
  transform: translateX(var(--rcl-switch-x, var(--rcl-switch-travel)));
}

/* Press: the thumb stretches toward its travel, the way a physical control
   gives before it moves. It grows from its fixed leading edge, so the checked
   thumb pulls back by the growth to stay inside the track. */
[data-rcl-selection-row][data-kind="switch"][data-pressed="true"]:not([data-dragging="true"]) [data-rcl-selection-indicator]::before {
  inline-size: var(--rcl-switch-stretch);
}
[data-rcl-selection-row][data-kind="switch"][data-pressed="true"][data-state="checked"]:not([data-dragging="true"]) [data-rcl-selection-indicator]::before {
  transform: translateX(calc(var(--rcl-switch-travel) - (var(--rcl-switch-stretch) - var(--rcl-switch-thumb))));
}
/* A drag is direct manipulation: the thumb is already under the finger, so it
   tracks 1:1 with no easing to lag behind it. */
[data-rcl-selection-row][data-kind="switch"][data-dragging="true"] [data-rcl-selection-indicator]::before {
  transition: background-color var(--dur-quick) var(--ease-standard);
}
[data-rcl-selection-row][data-kind="switch"][data-dragging="true"] { user-select: none; -webkit-user-select: none; }

[data-rcl-selection-copy] { display: grid; gap: var(--space-3xs); min-inline-size: 0; align-content: center; min-block-size: var(--tap-target-min); }
[data-rcl-selection-label] { font: var(--text-body); font-weight: 650; }
[data-rcl-selection-description], [data-rcl-selection-error] { font: var(--text-caption); color: var(--color-muted-foreground); }
[data-rcl-selection-error] { color: var(--color-danger); }
[data-rcl-selection-row][data-invalid="true"] [data-rcl-selection-indicator] { border-color: var(--color-danger); }

@media (prefers-reduced-motion: reduce) {
  [data-rcl-selection-row],
  [data-rcl-selection-indicator],
  [data-rcl-selection-indicator]::after,
  [data-rcl-selection-indicator]::before { transition: none; }
}
`;

function SelectionStyles() {
  return <StyleSheet name="selectioncontrol-1-0-4-1" css={styleSheet} />;
}

interface DragState {
  pointerId: number;
  startX: number;
  startChecked: boolean;
  travel: number;
  moved: boolean;
}

export const SelectionControl = withClassName(function SelectionControl({
  kind,
  label,
  labelPlacement = "start",
  description,
  error,
  checked,
  defaultChecked = false,
  disabled,
  id: providedId,
  indeterminate = false,
  onCheckedChange,
  onChange,
  required,
  // A host that already renders the label passes its id; the internal label
  // wins when there is one, but it must never erase the host's.
  "aria-labelledby": ariaLabelledBy,
  ...inputProps
}: SelectionControlProps) {
  const generatedId = useId();
  const id = providedId ?? `rcl-selection-${generatedId.replace(/:/g, "")}`;
  const [internalChecked, setInternalChecked] = useState(defaultChecked);
  const isControlled = checked !== undefined;
  const resolvedChecked = isControlled ? checked : internalChecked;
  const inputRef = useRef<HTMLInputElement>(null);
  const indicatorRef = useRef<HTMLSpanElement>(null);
  const dragRef = useRef<DragState | null>(null);
  /** A drag ends with a synthetic click from the label; it must not toggle again. */
  const suppressClickRef = useRef(false);
  const suppressTimerRef = useRef<number | null>(null);
  const [pressed, setPressed] = useState(false);
  /** Non-null only mid-drag: the state the thumb's position currently implies. */
  const [dragChecked, setDragChecked] = useState<boolean | null>(null);

  const hasCopy = label != null || description != null || error != null;
  const labelId = `${id}-label`;
  const descriptionId = description ? `${id}-description` : undefined;
  const errorId = error ? `${id}-error` : undefined;
  const gestural = kind === "switch" && !disabled;
  const shownChecked = dragChecked ?? resolvedChecked;
  /** `dragChecked` is set on the first move past the slop and cleared on release. */
  const dragging = dragChecked !== null;

  useEffect(() => {
    if (inputRef.current) inputRef.current.indeterminate = indeterminate;
  }, [indeterminate]);

  const handleChange = (event: ChangeEvent<HTMLInputElement>) => {
    const next = event.target.checked;
    if (!isControlled) setInternalChecked(next);
    onCheckedChange?.(next);
    onChange?.(event);
  };

  /**
   * Arm the one-shot click swallow, with an expiry. A drag that ends outside
   * the control may produce no click at all, and a flag left standing would
   * silently eat the next tap.
   */
  const armClickSuppression = () => {
    suppressClickRef.current = true;
    if (suppressTimerRef.current !== null) window.clearTimeout(suppressTimerRef.current);
    suppressTimerRef.current = window.setTimeout(() => {
      suppressClickRef.current = false;
      suppressTimerRef.current = null;
    }, CLICK_SUPPRESSION_MS);
  };

  useEffect(
    () => () => {
      if (suppressTimerRef.current !== null) window.clearTimeout(suppressTimerRef.current);
    },
    [],
  );

  const clearDrag = () => {
    dragRef.current = null;
    setPressed(false);
    setDragChecked(null);
    indicatorRef.current?.style.removeProperty("--rcl-switch-x");
  };

  /** Where the thumb sits for a pointer at `clientX`, clamped to the track. */
  const offsetFor = (drag: DragState, clientX: number) => {
    const base = drag.startChecked ? drag.travel : 0;
    return Math.max(0, Math.min(drag.travel, base + (clientX - drag.startX)));
  };

  const onPointerDown = (event: PointerEvent<HTMLSpanElement>) => {
    if (!gestural || event.button !== 0) return;
    const el = indicatorRef.current;
    if (!el) return;
    suppressClickRef.current = false;
    dragRef.current = {
      pointerId: event.pointerId,
      startX: event.clientX,
      startChecked: resolvedChecked,
      // clientWidth, not the border-box rect: the ratio is derived from the
      // track's content width, and the indicator carries a border.
      travel: el.clientWidth * SWITCH_TRAVEL_RATIO,
      moved: false,
    };
    setPressed(true);
    try {
      el.setPointerCapture(event.pointerId);
    } catch {
      // Capture is an optimisation; element-level events still deliver the drag.
    }
  };

  const onPointerMove = (event: PointerEvent<HTMLSpanElement>) => {
    const drag = dragRef.current;
    if (!drag || drag.pointerId !== event.pointerId) return;
    if (!drag.moved) {
      if (Math.abs(event.clientX - drag.startX) < AXIS_SLOP) return;
      drag.moved = true;
    }
    const offset = offsetFor(drag, event.clientX);
    indicatorRef.current?.style.setProperty("--rcl-switch-x", `${offset}px`);
    const implied = offset > drag.travel / 2;
    setDragChecked((current) => (current === implied ? current : implied));
  };

  const onPointerUp = (event: PointerEvent<HTMLSpanElement>) => {
    const drag = dragRef.current;
    if (!drag || drag.pointerId !== event.pointerId) return;
    const committed = drag.moved ? offsetFor(drag, event.clientX) > drag.travel / 2 : null;
    clearDrag();
    if (committed === null) return; // A tap: the label's own click toggles it.
    // The gesture already decided; swallow the click the label is about to
    // synthesize, then drive the native control so `onChange` still fires with
    // a real event.
    armClickSuppression();
    if (committed !== resolvedChecked) inputRef.current?.click();
  };

  const onPointerCancel = (event: PointerEvent<HTMLSpanElement>) => {
    const drag = dragRef.current;
    if (!drag || drag.pointerId !== event.pointerId) return;
    // The browser took the gesture back — abandon it rather than commit a value
    // the person never released on.
    if (drag.moved) armClickSuppression();
    clearDrag();
  };

  const onLabelClick = (event: MouseEvent<HTMLLabelElement>) => {
    if (!suppressClickRef.current) return;
    if (event.target === inputRef.current) return; // our own programmatic commit
    suppressClickRef.current = false;
    if (suppressTimerRef.current !== null) {
      window.clearTimeout(suppressTimerRef.current);
      suppressTimerRef.current = null;
    }
    event.preventDefault();
  };

  return (
    <>
      <SelectionStyles data-testid="primitives.selection-control" />
      <label
        data-rcl-selection-row
        data-kind={kind}
        data-state={indeterminate ? "mixed" : shownChecked ? "checked" : "unchecked"}
        data-disabled={disabled ? "true" : "false"}
        data-invalid={error ? "true" : "false"}
        data-label-placement={labelPlacement}
        data-bare={hasCopy ? "false" : "true"}
        data-pressed={pressed ? "true" : "false"}
        data-dragging={dragging ? "true" : "false"}
        htmlFor={id}
        onClick={onLabelClick}
      >
        <input
          {...inputProps}
          ref={inputRef}
          id={id}
          type={kind === "radio" ? "radio" : "checkbox"}
          role={kind === "switch" ? "switch" : kind}
          aria-labelledby={hasCopy ? labelId : ariaLabelledBy}
          aria-checked={kind === "switch" ? resolvedChecked : undefined}
          aria-describedby={[descriptionId, errorId].filter(Boolean).join(" ") || undefined}
          aria-invalid={error ? true : undefined}
          checked={resolvedChecked}
          disabled={disabled}
          required={required}
          onChange={handleChange}
          data-rcl-selection-input
        />
        <span
          ref={indicatorRef}
          aria-hidden="true"
          data-rcl-selection-indicator
          onPointerDown={gestural ? onPointerDown : undefined}
          onPointerMove={gestural ? onPointerMove : undefined}
          onPointerUp={gestural ? onPointerUp : undefined}
          onPointerCancel={gestural ? onPointerCancel : undefined}
        />
        {hasCopy && (
          <span data-rcl-selection-copy>
            {label != null && (
              <span id={labelId} data-rcl-selection-label>
                {label}
              </span>
            )}
            {description && (
              <span id={descriptionId} data-rcl-selection-description>
                {description}
              </span>
            )}
            {error && (
              <span id={errorId} data-rcl-selection-error role="alert">
                {error}
              </span>
            )}
          </span>
        )}
      </label>
    </>
  );
});
