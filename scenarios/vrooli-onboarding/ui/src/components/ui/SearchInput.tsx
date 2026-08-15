import { useRef, useImperativeHandle, forwardRef } from "react";
import { Search, Loader2, X } from "lucide-react";

interface SearchInputProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  ariaLabel: string;
  testId: string;
  clearTestId: string;
  /** Show a spinning indicator instead of the clear button */
  busy?: boolean;
  busyTestId?: string;
}

export interface SearchInputHandle {
  focus: () => void;
}

export const SearchInput = forwardRef<SearchInputHandle, SearchInputProps>(
  function SearchInput(
    { value, onChange, placeholder = "Search...", ariaLabel, testId, clearTestId, busy, busyTestId },
    ref,
  ) {
    const inputRef = useRef<HTMLInputElement>(null);

    useImperativeHandle(ref, () => ({
      focus: () => inputRef.current?.focus(),
    }));

    const clear = () => {
      onChange("");
      inputRef.current?.focus();
    };

    return (
      <div className="relative">
        <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted" aria-hidden="true" />
        <input
          ref={inputRef}
          data-testid={testId}
          type="search"
          placeholder={placeholder}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          aria-label={ariaLabel}
          className="w-full rounded-lg border border-muted bg-surface-muted py-2.5 pl-10 pr-10 text-sm text-foreground placeholder-slate-300 transition-colors focus:border-primary/50 focus:outline-none focus:ring-2 focus:ring-focus/20"
        />
        {busy && (
          <div className="absolute right-3 top-1/2 -translate-y-1/2" data-testid={busyTestId}>
            <Loader2 className="h-4 w-4 animate-spin text-muted" aria-hidden="true" />
            <span className="sr-only">Searching...</span>
          </div>
        )}
        {!busy && value && (
          <button
            type="button"
            onClick={clear}
            className="absolute right-3 top-1/2 -translate-y-1/2 rounded p-0.5 text-muted hover:text-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-focus"
            aria-label="Clear search"
            data-testid={clearTestId}
          >
            <X className="h-4 w-4" aria-hidden="true" />
          </button>
        )}
      </div>
    );
  },
);
