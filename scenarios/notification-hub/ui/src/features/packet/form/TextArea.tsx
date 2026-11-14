import { useId } from "react";

interface TextAreaProps {
  label: string;
  value: string;
  onChange: (value: string) => void;
  hideLabel?: boolean;
  required?: boolean;
  disabled?: boolean;
  maxLength?: number;
  rows?: number;
  testId?: string;
  placeholder?: string;
  className?: string;
}

/** A labelled multi-line field at the phone-safe 16px size. */
export function TextArea({ label, value, onChange, hideLabel, required, disabled, maxLength, rows = 3, testId, placeholder, className = "" }: TextAreaProps) {
  const id = useId();
  return (
    <div className={`flex flex-col gap-1 ${className}`}>
      <label htmlFor={id} className={hideLabel ? "sr-only" : "text-sm font-medium"}>
        {label}
      </label>
      <textarea
        id={id}
        data-testid={testId}
        aria-required={required || undefined}
        rows={rows}
        maxLength={maxLength}
        placeholder={placeholder}
        value={value}
        disabled={disabled}
        onChange={(event) => onChange(event.target.value)}
        className="min-h-24 w-full resize-y rounded-control border border-app-border bg-app-surface p-3 text-base leading-snug text-app-foreground placeholder:text-app-muted-foreground disabled:opacity-70"
      />
    </div>
  );
}
