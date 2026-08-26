/**
 * @libraryId react-component-library:RadioGroup
 * @displayName RadioGroup
 * @description A native radio group with controlled selection, keyboard-operable options, supporting context, disabled choices, and recoverable errors.
 * @version 1.0.3
 * @tags ["controls","selection","forms","accessibility","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource controls.radio-group */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

import { useId, useState, type ReactNode } from "react";
import { SelectionControl } from "@vrooli/react-component-library/SelectionControl/1.0.0";

export interface RadioOption {
  value: string;
  label: ReactNode;
  description?: ReactNode;
  disabled?: boolean;
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
@media (prefers-reduced-motion: reduce) { [data-rcl-radio-group] * { transition-duration: 0s !important; } }
`;

function RadioGroupStyles() {
  return <style data-rcl-radio-group-styles dangerouslySetInnerHTML={{ __html: styleSheet }} />;
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
        <div data-rcl-radio-options data-orientation={orientation}>
          {options.map((option) => (
            <SelectionControl
              key={option.value}
              kind="radio"
              name={groupName}
              value={option.value}
              label={option.label}
              description={option.description}
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
