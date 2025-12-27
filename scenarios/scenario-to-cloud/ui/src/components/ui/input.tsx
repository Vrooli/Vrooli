import { forwardRef } from "react";
import { cn } from "../../lib/utils";

export interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  hint?: string;
  error?: string;
  warning?: string;
  /** Show a loading spinner in the input */
  isLoading?: boolean;
}

export const Input = forwardRef<HTMLInputElement, InputProps>(
  ({ className, label, hint, error, warning, isLoading, id, ...props }, ref) => {
    const inputId = id || label?.toLowerCase().replace(/\s+/g, "-");

    return (
      <div className="space-y-1.5">
        {label && (
          <label
            htmlFor={inputId}
            className="block text-sm font-medium text-slate-300"
          >
            {label}
          </label>
        )}
        <div className="relative">
          <input
            id={inputId}
            ref={ref}
            className={cn(
              "w-full rounded-lg border bg-black/30 px-3 py-2 text-sm text-slate-100",
              "placeholder:text-slate-500",
              "focus:outline-none focus:ring-2 focus:ring-slate-500",
              "disabled:cursor-not-allowed disabled:opacity-50",
              isLoading && "pr-8",
              error
                ? "border-red-500/50 focus:ring-red-500/50"
                : warning
                  ? "border-amber-500/50 focus:ring-amber-500/50"
                  : "border-white/10",
              className
            )}
            {...props}
          />
          {isLoading && (
            <div className="absolute right-2 top-1/2 -translate-y-1/2">
              <div className="h-4 w-4 animate-spin rounded-full border-2 border-slate-500 border-t-transparent" />
            </div>
          )}
        </div>
        {hint && !error && !warning && (
          <p className="text-xs text-slate-500">{hint}</p>
        )}
        {warning && !error && (
          <p className="text-xs text-amber-400">{warning}</p>
        )}
        {error && (
          <p className="text-xs text-red-400">{error}</p>
        )}
      </div>
    );
  }
);

Input.displayName = "Input";

export interface TextareaProps extends React.TextareaHTMLAttributes<HTMLTextAreaElement> {
  label?: string;
  hint?: string;
  error?: string;
}

export const Textarea = forwardRef<HTMLTextAreaElement, TextareaProps>(
  ({ className, label, hint, error, id, ...props }, ref) => {
    const inputId = id || label?.toLowerCase().replace(/\s+/g, "-");

    return (
      <div className="space-y-1.5">
        {label && (
          <label
            htmlFor={inputId}
            className="block text-sm font-medium text-slate-300"
          >
            {label}
          </label>
        )}
        <textarea
          id={inputId}
          ref={ref}
          className={cn(
            "w-full rounded-lg border bg-black/30 px-3 py-2 text-sm text-slate-100 font-mono",
            "placeholder:text-slate-500",
            "focus:outline-none focus:ring-2 focus:ring-slate-500",
            "disabled:cursor-not-allowed disabled:opacity-50",
            "resize-y min-h-[120px]",
            error
              ? "border-red-500/50 focus:ring-red-500/50"
              : "border-white/10",
            className
          )}
          {...props}
        />
        {hint && !error && (
          <p className="text-xs text-slate-500">{hint}</p>
        )}
        {error && (
          <p className="text-xs text-red-400">{error}</p>
        )}
      </div>
    );
  }
);

Textarea.displayName = "Textarea";
