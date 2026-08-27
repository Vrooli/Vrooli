import type { ReactNode } from "react";
import { cn } from "../../lib/classnames";

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
        <div className="text-sm font-medium text-wc-text-secondary">{label}</div>
        {hint && <div className={cn("text-[11px] text-wc-text-muted", hintClassName)}>{hint}</div>}
      </div>
      <div className="max-w-full shrink-0">{control}</div>
    </div>
  );
}

export function SettingsToggle({
  checked,
  onClick,
  testId,
}: {
  checked: boolean;
  onClick: () => void;
  testId?: string;
}) {
  return (
    <button
      data-testid={testId}
      role="switch"
      aria-checked={checked}
      className={cn(
        "relative inline-flex h-6 w-11 items-center rounded-full transition-colors",
        checked ? "bg-wc-accent" : "bg-wc-surface-base",
      )}
      onClick={onClick}
    >
      <span
        className={cn(
          "inline-block h-4.5 w-4.5 rounded-full bg-white transition-transform",
          checked ? "translate-x-[22px]" : "translate-x-[3px]",
        )}
      />
    </button>
  );
}
