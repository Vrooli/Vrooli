import { createContext, useContext, useId, useState, type ReactNode } from "react";
import { Switch } from "@vrooli/react-component-library/Switch";
import { Slider } from "@vrooli/react-component-library/Slider";
import { cn } from "../../lib/classnames";

/**
 * The id of the label a `SettingsRow` renders. Controls placed in the row's
 * `control` slot read it so the row's visible label becomes their accessible
 * name — without every call site passing the same string twice.
 */
const SettingsRowLabelContext = createContext<string | undefined>(undefined);

export function SettingsSectionIntro({
  eyebrow,
  title,
  description,
}: {
  eyebrow: string;
  title: string;
  description: string;
}) {
  return (
    <div className="space-y-1">
      <p className="text-[11px] font-semibold uppercase tracking-[0.22em] text-wc-text-muted">
        {eyebrow}
      </p>
      <div>
        <h3 className="text-lg font-semibold text-wc-text-primary">{title}</h3>
        <p className="text-sm text-wc-text-faint">{description}</p>
      </div>
    </div>
  );
}

export function SettingsCard({
  className,
  children,
}: {
  className?: string;
  children: ReactNode;
}) {
  return (
    <div className={cn("rounded-2xl border border-wc-default bg-wc-surface-input/80 p-4", className)}>
      {children}
    </div>
  );
}

export function SettingsRow({
  label,
  hint,
  hintClassName,
  control,
  className,
}: {
  label: string;
  hint?: string;
  hintClassName?: string;
  control: ReactNode;
  className?: string;
}) {
  const generatedId = useId();
  const labelId = `settings-row-${generatedId.replace(/:/g, "")}`;
  return (
    // The control keeps its intrinsic width (`shrink-0`), so on a narrow
    // surface a row layout has nothing left to give the label but collapse:
    // the text wraps to one word per line and runs under the control. Below
    // `sm` the row stacks instead, which is the only arrangement where both
    // halves get the full inline size.
    <div
      className={cn(
        "flex flex-col items-start gap-2 sm:flex-row sm:items-center sm:justify-between sm:gap-4",
        className,
      )}
    >
      <div className="min-w-0">
        <div id={labelId} className="text-sm font-medium text-wc-text-secondary">
          {label}
        </div>
        {hint && <div className={cn("text-[11px] text-wc-text-muted", hintClassName)}>{hint}</div>}
      </div>
      <div className="max-w-full shrink-0">
        <SettingsRowLabelContext.Provider value={labelId}>{control}</SettingsRowLabelContext.Provider>
      </div>
    </div>
  );
}

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
  const rowLabelId = useContext(SettingsRowLabelContext);
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
  const rowLabelId = useContext(SettingsRowLabelContext);
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
