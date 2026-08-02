/**
 * @vrooliComponentSource react-component-library:Select
 * @vrooliComponentVersion 1.1.0
 * @vrooliComponentAdoption template:react-vite:select
 * @vrooliComponentAppliedAt 2026-07-14T20:14:31Z
 * @vrooliComponentSourceSha256 b4032163d23c306846d6bdd1afe6aa9dc32acf823ddceaf6cb42b8b5542daf51
 * @vrooliComponentDriftHash b4032163d23c306846d6bdd1afe6aa9dc32acf823ddceaf6cb42b8b5542daf51
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
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
          <option key={option.value} value={option.value} disabled={option.disabled}>
            {option.label}
          </option>
        ))}
      </select>
    );
  },
);

