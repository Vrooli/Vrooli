import { cn } from "../../lib/utils";

export interface RadioOption {
  value: string;
  label: string;
}

export interface RadioGroupProps {
  name: string;
  value: string;
  onChange: (value: string) => void;
  options: RadioOption[];
  className?: string;
  testIdPrefix?: string;
  disabled?: boolean;
}

export function RadioGroup({ name, value, onChange, options, className, testIdPrefix, disabled }: RadioGroupProps) {
  return (
    <div className={cn("space-y-2", className)} role="radiogroup">
      {options.map((opt) => (
        <label
          key={opt.value}
          className={cn("flex items-center gap-2 text-sm text-slate-200", disabled ? "opacity-50 cursor-not-allowed" : "cursor-pointer")}
          data-testid={testIdPrefix ? `${testIdPrefix}-${opt.value}` : undefined}
        >
          <input
            type="radio"
            name={name}
            value={opt.value}
            checked={value === opt.value}
            onChange={() => onChange(opt.value)}
            disabled={disabled}
            className="accent-cyan-500"
          />
          {opt.label}
        </label>
      ))}
    </div>
  );
}
