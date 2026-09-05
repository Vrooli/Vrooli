import { useState } from "react";
import { Switch } from "@vrooli/react-component-library/Switch";
import { Slider } from "@vrooli/react-component-library/Slider";
import { useSettingsRowLabelId } from "@vrooli/react-component-library/SettingsList/1";
import { cn } from "../../lib/classnames";

/**
 * The settings-row switch. A thin binding over the library control: the row
 * owns the label, so the switch renders bare and takes its accessible name from
 * the row through context.
 */
export function SettingsToggle({
  checked,
  onCheckedChange,
  disabled,
  testId,
  ariaLabel,
}: {
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
  disabled?: boolean;
  testId?: string;
  ariaLabel?: string;
}) {
  const rowLabelId = useSettingsRowLabelId();
  return (
    <Switch
      data-testid={testId}
      checked={checked}
      onCheckedChange={onCheckedChange}
      disabled={disabled}
      aria-label={ariaLabel}
      aria-labelledby={ariaLabel ? undefined : rowLabelId}
    />
  );
}

/**
 * The settings-row slider.
 *
 * The live value is held here and only handed to `onCommit` when the
 * interaction ends, so a drag across the track produces one store write instead
 * of one per step. `onChange` exists for previews that need to follow the
 * finger; it must never be used to persist.
 */
export function SettingsSlider({
  value,
  onChange,
  onCommit,
  min,
  max,
  step,
  formatValue,
  defaultMarker,
  ticks,
  disabled,
  testId,
  ariaLabel,
  className,
}: {
  value: number;
  onChange?: (value: number) => void;
  onCommit: (value: number) => void;
  min: number;
  max: number;
  step?: number;
  formatValue?: (value: number) => string;
  defaultMarker?: number;
  ticks?: number | number[];
  disabled?: boolean;
  testId?: string;
  ariaLabel?: string;
  className?: string;
}) {
  const rowLabelId = useSettingsRowLabelId();
  // Non-null only while the control is being moved; the committed value in
  // `value` remains the source of truth the moment the interaction ends.
  const [draft, setDraft] = useState<number | null>(null);
  return (
    <div className={cn("w-[min(16rem,60vw)]", className)}>
      <Slider
        data-testid={testId}
        value={draft ?? value}
        onChange={(next) => {
          setDraft(next);
          onChange?.(next);
        }}
        onChangeCommit={(next) => {
          setDraft(null);
          onCommit(next);
        }}
        min={min}
        max={max}
        step={step}
        ticks={ticks}
        defaultMarker={defaultMarker}
        formatValue={formatValue}
        disabled={disabled}
        aria-label={ariaLabel}
        aria-labelledby={ariaLabel ? undefined : rowLabelId}
      />
    </div>
  );
}
