/**
 * @libraryId react-component-library:SelectionControl
 * @displayName SelectionControl
 * @description Native checkbox, radio, and switch semantics with a shared token-bound selection surface.
 * @version 1.0.1
 * @tags ["primitive","selection","accessibility","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:SelectionControl */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

import {
  useEffect,
  useId,
  useRef,
  useState,
  type ChangeEvent,
  type InputHTMLAttributes,
  type ReactNode,
} from "react";

export type SelectionControlKind = "checkbox" | "radio" | "switch";

export interface SelectionControlProps
  extends Omit<
    InputHTMLAttributes<HTMLInputElement>,
    "children" | "onChange" | "type"
  > {
  kind: SelectionControlKind;
  label: ReactNode;
  description?: ReactNode;
  error?: ReactNode;
  onCheckedChange?: (checked: boolean) => void;
  onChange?: (event: ChangeEvent<HTMLInputElement>) => void;
  indeterminate?: boolean;
}

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
[data-rcl-selection-row][data-kind="switch"] { grid-template-columns: 2.75rem minmax(0, 1fr); }
[data-rcl-selection-row][data-kind="switch"] [data-rcl-selection-indicator] {
  inline-size: 2.5rem;
  block-size: 1.5rem;
  margin-block-start: calc((var(--tap-target-min) - 1.5rem) / 2);
  border-radius: var(--radius-pill);
  background: var(--color-surface-muted);
}
[data-rcl-selection-row][data-kind="switch"] [data-rcl-selection-indicator]::before {
  content: "";
  position: absolute;
  inset-block-start: .2rem;
  inset-inline-start: .2rem;
  inline-size: 1rem;
  block-size: 1rem;
  border-radius: var(--radius-pill);
  background: var(--color-muted-foreground);
  box-shadow: var(--elev-raised);
  transition: background-color var(--dur-quick) var(--ease-standard), transform var(--dur-quick) var(--ease-standard);
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
  transform: translateX(1rem);
}
[data-rcl-selection-copy] { display: grid; gap: var(--space-3xs); min-inline-size: 0; align-content: center; min-block-size: var(--tap-target-min); }
[data-rcl-selection-label] { font: var(--text-body); font-weight: 650; }
[data-rcl-selection-description], [data-rcl-selection-error] { font: var(--text-caption); color: var(--color-muted-foreground); }
[data-rcl-selection-error] { color: var(--color-danger); }
[data-rcl-selection-row][data-invalid="true"] [data-rcl-selection-indicator] { border-color: var(--color-danger); }

`;

function SelectionStyles() {
  return <StyleSheet name="selectioncontrol-1-0-1-1" css={styleSheet} />;
}

export const SelectionControl = withClassName(function SelectionControl({
  kind,
  label,
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
  ...inputProps
}: SelectionControlProps) {
  const generatedId = useId();
  const id = providedId ?? `rcl-selection-${generatedId.replace(/:/g, "")}`;
  const [internalChecked, setInternalChecked] = useState(defaultChecked);
  const isControlled = checked !== undefined;
  const resolvedChecked = isControlled ? checked : internalChecked;
  const inputRef = useRef<HTMLInputElement>(null);
  const labelId = `${id}-label`;
  const descriptionId = description ? `${id}-description` : undefined;
  const errorId = error ? `${id}-error` : undefined;

  useEffect(() => {
    if (inputRef.current) inputRef.current.indeterminate = indeterminate;
  }, [indeterminate]);

  const handleChange = (event: ChangeEvent<HTMLInputElement>) => {
    const next = event.target.checked;
    if (!isControlled) setInternalChecked(next);
    onCheckedChange?.(next);
    onChange?.(event);
  };

  return (
    <>
      <SelectionStyles data-testid="primitives.selection-control" />
      <label
        data-rcl-selection-row
        data-kind={kind}
        data-state={
          indeterminate ? "mixed" : resolvedChecked ? "checked" : "unchecked"
        }
        data-disabled={disabled ? "true" : "false"}
        data-invalid={error ? "true" : "false"}
        htmlFor={id}
      >
        <input
          {...inputProps}
          ref={inputRef}
          id={id}
          type={kind === "radio" ? "radio" : "checkbox"}
          role={kind === "switch" ? "switch" : kind}
          aria-labelledby={labelId}
          aria-checked={kind === "switch" ? resolvedChecked : undefined}
          aria-describedby={
            [descriptionId, errorId].filter(Boolean).join(" ") || undefined
          }
          aria-invalid={error ? true : undefined}
          checked={resolvedChecked}
          disabled={disabled}
          required={required}
          onChange={handleChange}
          data-rcl-selection-input
        />
        <span aria-hidden="true" data-rcl-selection-indicator />
        <span data-rcl-selection-copy>
          <span id={labelId} data-rcl-selection-label>
            {label}
          </span>
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
      </label>
    </>
  );
});
