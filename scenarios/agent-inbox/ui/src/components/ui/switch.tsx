import { cn } from "../../lib/utils";

interface SwitchProps {
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
  disabled?: boolean;
  className?: string;
  tone?: "indigo" | "yellow";
  "aria-label"?: string;
  "data-testid"?: string;
}

export function Switch({
  checked,
  onCheckedChange,
  disabled = false,
  className,
  tone = "indigo",
  ...props
}: SwitchProps) {
  const toneClass = tone === "yellow" ? "bg-yellow-500" : "bg-indigo-500";

  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      disabled={disabled}
      onClick={() => onCheckedChange(!checked)}
      className={cn(
        "relative inline-flex h-6 w-11 items-center rounded-full transition-colors",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500/50",
        checked ? toneClass : "bg-slate-600",
        disabled && "cursor-not-allowed opacity-50",
        className
      )}
      {...props}
    >
      <span
        className={cn(
          "absolute left-1 top-1 h-4 w-4 rounded-full bg-white transition-transform",
          checked && "translate-x-5"
        )}
      />
    </button>
  );
}
