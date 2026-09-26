import { Search, X } from "lucide-react";
import * as React from "react";

import { cn } from "../../lib/utils";

export interface SearchInputProps
  extends Omit<React.InputHTMLAttributes<HTMLInputElement>, "onChange" | "value" | "type"> {
  value: string;
  onChange: (next: string) => void;
  onClear?: () => void;
  ariaLabel: string;
  clearLabel: string;
}

export const SearchInput = React.forwardRef<HTMLInputElement, SearchInputProps>(function SearchInput(
  { value, onChange, onClear, ariaLabel, clearLabel, className, placeholder, ...props },
  ref,
) {
  return (
    <div className={cn("relative flex w-full items-center", className)}>
      <Search
        aria-hidden
        className="pointer-events-none absolute left-3 h-4 w-4 text-app-muted-foreground"
      />
      <input
        ref={ref}
        type="search"
        role="searchbox"
        aria-label={ariaLabel}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        className="h-11 min-h-touch w-full rounded-control border border-app-border bg-app-surface pl-9 pr-9 text-base md:text-sm text-app-foreground placeholder:text-app-muted-foreground focus:outline-none focus:ring-2 focus:ring-app-focus"
        {...props}
      />
      {value ? (
        <button
          type="button"
          aria-label={clearLabel}
          onClick={() => {
            onChange("");
            onClear?.();
          }}
          className="absolute right-2 inline-flex h-7 w-7 items-center justify-center rounded-pill text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-foreground"
        >
          <X aria-hidden className="h-4 w-4" />
        </button>
      ) : null}
    </div>
  );
});
