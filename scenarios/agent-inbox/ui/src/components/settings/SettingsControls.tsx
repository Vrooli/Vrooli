import type { ReactNode } from "react";
import { Input } from "../ui/input";
import { Switch } from "../ui/switch";

interface SettingsSectionProps {
  title: string;
  description?: string;
  children: ReactNode;
  className?: string;
}

export function SettingsSection({ title, description, children, className = "" }: SettingsSectionProps) {
  return (
    <section className={className}>
      <h3 className="text-sm font-medium text-slate-300 mb-2">{title}</h3>
      {description && <p className="text-xs text-slate-500 mb-3">{description}</p>}
      {children}
    </section>
  );
}

interface SettingsSwitchRowProps {
  title: string;
  description?: string;
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
  disabled?: boolean;
  tone?: "indigo" | "yellow";
  testId?: string;
}

export function SettingsSwitchRow({
  title,
  description,
  checked,
  onCheckedChange,
  disabled = false,
  tone = "indigo",
  testId,
}: SettingsSwitchRowProps) {
  return (
    <div className="flex items-center justify-between gap-4 p-3 bg-white/5 border border-white/10 rounded-lg">
      <div className="min-w-0">
        <p className="text-sm text-white">{title}</p>
        {description && <p className="text-xs text-slate-500">{description}</p>}
      </div>
      <Switch
        checked={checked}
        onCheckedChange={onCheckedChange}
        disabled={disabled}
        tone={tone}
        data-testid={testId}
        aria-label={title}
      />
    </div>
  );
}

interface SettingsNumberFieldProps {
  label: string;
  value: number;
  onChange: (next: number) => void;
  min: number;
  max: number;
  step?: number;
  disabled?: boolean;
}

export function SettingsNumberField({
  label,
  value,
  onChange,
  min,
  max,
  step = 1,
  disabled = false,
}: SettingsNumberFieldProps) {
  return (
    <label className="text-xs text-slate-400">
      {label}
      <Input
        type="number"
        min={min}
        max={max}
        step={step}
        value={value}
        disabled={disabled}
        onChange={(e) => {
          const next = Number(e.target.value);
          onChange(Number.isFinite(next) ? next : value);
        }}
        className="mt-1"
      />
    </label>
  );
}
