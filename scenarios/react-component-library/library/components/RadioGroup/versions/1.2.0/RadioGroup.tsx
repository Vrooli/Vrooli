/**
 * @libraryId react-component-library:RadioGroup
 * @displayName RadioGroup
 * @description The single-selection collection with roving focus, optional card-style options carrying descriptions, disabled choices, responsive layout, and animated selection.
 * @version 1.2.0
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource controls.radio-group */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import { useId, useState, type ReactNode } from "react";
import { SelectionControl } from "@vrooli/react-component-library/SelectionControl/1";

export const RADIO_GROUP_VARIANTS = ["control", "card"] as const;
export type RadioGroupVariant = (typeof RADIO_GROUP_VARIANTS)[number];

export interface RadioOption {
  value: string;
  label: ReactNode;
  description?: ReactNode;
  disabled?: boolean;
  /**
   * Trailing content on the option's title row — a chosen mark, a price, a
   * count. Rendered inside the control's own label, so it is part of the
   * clickable target rather than a decoration sitting beside one.
   *
   * `card` only: at control size there is no room for it, and a badge beside a
   * bare radio reads as a second control.
   */
  badge?: ReactNode;
  /**
   * Automation handle for this option's control.
   *
   * A group is addressable by its label; one option of it is not, and every
   * flow that picks a posture, a plan or a preset has to name the one it picks.
   * Without this the only handle is the value attribute, which ties a test to
   * the server's vocabulary rather than to the choice being made.
   */
  testId?: string;
}

export interface RadioGroupProps {
  options: RadioOption[];
  value?: string;
  defaultValue?: string;
  name?: string;
  label: ReactNode;
  description?: ReactNode;
  error?: ReactNode;
  orientation?: "vertical" | "horizontal";
  /**
   * `control` is a list of radios. `card` gives each option a bordered surface
   * whose whole area is the target, which is what a choice with a sentence of
   * consequence attached needs — a permission posture, an install preset.
   */
  variant?: RadioGroupVariant;
  onValueChange?: (value: string) => void;
  disabled?: boolean;
}

const styleSheet = `
[data-rcl-radio-group] { display: grid; gap: var(--space-xs); min-inline-size: 0; }
[data-rcl-radio-group] > [data-rcl-radio-legend] { display: grid; gap: var(--space-3xs); margin: 0; padding: 0; border: 0; color: var(--color-foreground); font: var(--text-body); font-weight: 700; }
[data-rcl-radio-description] { color: var(--color-muted-foreground); font: var(--text-caption); font-weight: 400; }
[data-rcl-radio-options] { display: grid; gap: var(--space-2xs); min-inline-size: 0; }
[data-rcl-radio-options][data-orientation="horizontal"] { grid-template-columns: repeat(auto-fit, minmax(min(100%, 11rem), 1fr)); }
[data-rcl-radio-error] { color: var(--color-danger); font: var(--text-caption); }

/* ── card ──
   The card is not a new control; it is the same SelectionControl given a
   surface. Every rule below is scoped under the variant so the control
   presentation is untouched, and each one beats SelectionControl's own at
   (0,2,0) against its (0,1,0) without !important. */
[data-rcl-radio-options][data-variant="card"] { gap: var(--space-2xs, 8px); }
[data-rcl-radio-options][data-variant="card"] [data-rcl-selection-row] {
  align-items: center;
  padding: var(--space-xs, 12px) var(--space-sm, 16px);
  border: var(--border-hairline, 1px) solid var(--color-border);
  border-radius: var(--radius-panel, .75rem);
  background: var(--color-surface);
}
[data-rcl-radio-options][data-variant="card"] [data-rcl-selection-row]:hover:not([data-disabled="true"]) {
  border-color: color-mix(in srgb, var(--color-primary) 50%, var(--color-border));
}
[data-rcl-radio-options][data-variant="card"] [data-rcl-selection-row][data-state="checked"] {
  border-color: color-mix(in srgb, var(--color-primary) 55%, var(--color-border));
  background: color-mix(in srgb, var(--color-primary) 10%, var(--color-surface));
}
/* The indicator is centred with the title row rather than pinned to the first
   text line, because on a card the copy block is the taller element and a
   top-aligned radio reads as belonging to the title alone. */
[data-rcl-radio-options][data-variant="card"] [data-rcl-selection-indicator] { margin-block-start: 0; }
[data-rcl-radio-options][data-variant="card"] [data-rcl-selection-copy] { min-inline-size: 0; }
[data-rcl-radio-card-head] {
  display: flex; align-items: center; justify-content: space-between;
  gap: var(--space-sm, 16px); min-inline-size: 0;
}
[data-rcl-radio-card-title] { min-inline-size: 0; overflow-wrap: anywhere; }
[data-rcl-radio-card-badge] { flex: 0 0 auto; }
`;

function RadioGroupStyles() {
  return <StyleSheet name="radiogroup-1-1-0" css={styleSheet} />;
}

export const RadioGroup = withClassName(function RadioGroup({
  options,
  value,
  defaultValue,
  name,
  label,
  description,
  error,
  orientation = "vertical",
  variant = "control",
  onValueChange,
  disabled = false,
}: RadioGroupProps) {
  const generatedId = useId();
  const groupId = `rcl-radio-group-${generatedId.replace(/:/g, "")}`;
  const groupName = name ?? groupId;
  const [internalValue, setInternalValue] = useState(defaultValue ?? options[0]?.value ?? "");
  const isControlled = value !== undefined;
  const selectedValue = isControlled ? value : internalValue;
  const descriptionId = description ? `${groupId}-description` : undefined;
  const errorId = error ? `${groupId}-error` : undefined;
  const labelId = `${groupId}-label`;

  const handleValueChange = (next: string) => {
    if (!isControlled) setInternalValue(next);
    onValueChange?.(next);
  };

  return (
    <>
      <RadioGroupStyles data-testid="controls.radio-group" />
      <div
        role="radiogroup"
        aria-labelledby={labelId}
        aria-describedby={[descriptionId, errorId].filter(Boolean).join(" ") || undefined}
        data-rcl-radio-group
        data-disabled={disabled ? "true" : "false"}
      >
        <div data-rcl-radio-legend>
          <span id={labelId}>{label}</span>
          {description && (
            <span id={descriptionId} data-rcl-radio-description>
              {description}
            </span>
          )}
        </div>
        <div data-rcl-radio-options data-orientation={orientation} data-variant={variant}>
          {options.map((option) => (
            <SelectionControl
              key={option.value}
              kind="radio"
              name={groupName}
              value={option.value}
              label={
                variant === "card" && option.badge !== undefined ? (
                  <span data-rcl-radio-card-head>
                    <span data-rcl-radio-card-title>{option.label}</span>
                    <span data-rcl-radio-card-badge>{option.badge}</span>
                  </span>
                ) : (
                  option.label
                )
              }
              description={option.description}
              data-testid={option.testId}
              checked={selectedValue === option.value}
              disabled={disabled || option.disabled}
              onCheckedChange={(checked) => {
                if (checked) handleValueChange(option.value);
              }}
              error={error && selectedValue === option.value ? error : undefined}
            />
          ))}
        </div>
        {error && (
          <span id={errorId} data-rcl-radio-error role="alert">
            {error}
          </span>
        )}
      </div>
    </>
  );
});
