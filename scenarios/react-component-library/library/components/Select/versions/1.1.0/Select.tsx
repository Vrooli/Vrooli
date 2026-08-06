/**
 * @libraryId react-component-library:Select
 * @version 1.1.0
 * @status released
 * @deps {"react":"^18","clsx":"^2.1.1","tailwind-merge":"^2.3.0"}
 */
import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";
import { forwardRef, type SelectHTMLAttributes } from "react";

export interface SelectOption {
  value: string;
  label: string;
  disabled?: boolean;
}

export interface SelectProps extends SelectHTMLAttributes<HTMLSelectElement> {
  options: SelectOption[];
  placeholder?: string;
}

const cn = (...inputs: ClassValue[]) => twMerge(clsx(inputs));

export const Select = forwardRef<HTMLSelectElement, SelectProps>(
  function Select({ className, options, placeholder, ...props }, ref) {
    return (
      <select
        ref={ref}
        className={cn(
          "min-h-11 w-full rounded-control border border-app-border bg-app-surface px-3 py-2 text-base text-app-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50 disabled:cursor-not-allowed disabled:opacity-60 md:text-sm",
          className,
        )}
        {...props}
      >
        {placeholder && <option value="">{placeholder}</option>}
        {options.map((option) => (
          <option
            key={option.value}
            value={option.value}
            disabled={option.disabled}
          >
            {option.label}
          </option>
        ))}
      </select>
    );
  },
);
