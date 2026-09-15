import * as React from "react";

import { cn } from "../../lib/utils";

export interface SelectOption<V extends string = string> {
  value: V;
  label: string;
  disabled?: boolean;
}

export interface SelectProps<V extends string = string>
  extends Omit<React.SelectHTMLAttributes<HTMLSelectElement>, "onChange" | "value"> {
  value: V;
  options: ReadonlyArray<SelectOption<V>>;
  onChange: (next: V) => void;
  ariaLabel: string;
}

function SelectInner<V extends string>(
  { value, options, onChange, ariaLabel, className, ...props }: SelectProps<V>,
  ref: React.ForwardedRef<HTMLSelectElement>,
) {
  return (
    <select
      ref={ref}
      value={value}
      aria-label={ariaLabel}
      onChange={(e) => onChange(e.target.value as V)}
      className={cn(
        "h-11 min-h-touch w-full rounded-control border border-app-border bg-app-surface px-3 text-base md:text-sm text-app-foreground focus:outline-none focus:ring-2 focus:ring-app-focus",
        className,
      )}
      {...props}
    >
      {options.map((option) => (
        <option key={option.value} value={option.value} disabled={option.disabled}>
          {option.label}
        </option>
      ))}
    </select>
  );
}

export const Select = React.forwardRef(SelectInner) as <V extends string>(
  props: SelectProps<V> & { ref?: React.ForwardedRef<HTMLSelectElement> },
) => React.ReactElement;
