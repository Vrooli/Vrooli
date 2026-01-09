import { cn } from "../../lib/utils";

interface SelectableCardProps {
  selected: boolean;
  onClick: () => void;
  children: React.ReactNode;
  className?: string;
  disabled?: boolean;
}

/**
 * A clickable card that shows selection state with consistent styling.
 * Used for profile, scenario, and tier selection throughout the UI.
 */
export function SelectableCard({
  selected,
  onClick,
  children,
  className,
  disabled = false,
}: SelectableCardProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      className={cn(
        "rounded-lg border px-3 py-2 text-left transition",
        selected
          ? "border-cyan-400 bg-cyan-500/10"
          : "border-white/10 bg-white/5 hover:border-white/20",
        disabled && "opacity-50 cursor-not-allowed",
        className
      )}
    >
      {children}
    </button>
  );
}
