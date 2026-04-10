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
      {options.map((opt) => {
        const isChecked = value === opt.value;
        return (
          <label
            key={opt.value}
            className={cn("flex items-center gap-2 text-sm text-slate-200", disabled ? "opacity-50 cursor-not-allowed" : "cursor-pointer")}
            data-testid={testIdPrefix ? `${testIdPrefix}-${opt.value}` : undefined}
          >
            <input
              type="radio"
              name={name}
              value={opt.value}
              checked={isChecked}
              onChange={() => onChange(opt.value)}
              disabled={disabled}
              className="sr-only"
            />
            <span
              aria-hidden="true"
              className={cn(
                "flex h-4 w-4 shrink-0 items-center justify-center rounded-full border-2 transition-colors",
                isChecked
                  ? "border-cyan-500 bg-cyan-500"
                  : "border-slate-500 bg-transparent",
              )}
            >
              {isChecked && (
                <span className="h-1.5 w-1.5 rounded-full bg-white" />
              )}
            </span>
            {opt.label}
          </label>
        );
      })}
    </div>
  );
}
