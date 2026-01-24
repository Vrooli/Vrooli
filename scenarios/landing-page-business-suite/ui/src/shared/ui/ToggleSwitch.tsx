import * as React from "react";
import { RefreshCw } from "lucide-react";
import { cn } from "../lib/utils";

interface ToggleSwitchProps {
  id?: string;
  checked: boolean;
  onToggle: () => void;
  disabled?: boolean;
  loading?: boolean;
  className?: string;
  checkedClassName?: string;
  uncheckedClassName?: string;
  "aria-label"?: string;
}

export function ToggleSwitch({
  id,
  checked,
  onToggle,
  disabled,
  loading,
  className,
  checkedClassName,
  uncheckedClassName,
  "aria-label": ariaLabel,
}: ToggleSwitchProps) {
  const isDisabled = disabled || loading;

  return (
    <button
      id={id}
      type="button"
      role="switch"
      aria-checked={checked}
      aria-label={ariaLabel}
      disabled={isDisabled}
      onClick={onToggle}
      className={cn(
        "relative inline-flex h-6 w-11 items-center rounded-full transition-colors disabled:opacity-50",
        checked ? checkedClassName ?? "bg-amber-500" : uncheckedClassName ?? "bg-slate-700",
        className
      )}
    >
      {loading ? (
        <RefreshCw className="absolute left-1/2 h-4 w-4 -translate-x-1/2 animate-spin text-white" />
      ) : (
        <span
          className={cn(
            "inline-block h-4 w-4 transform rounded-full bg-white transition-transform",
            checked ? "translate-x-6" : "translate-x-1"
          )}
        />
      )}
    </button>
  );
}
