import { type LucideIcon, Check } from "lucide-react";
import { cn } from "../../lib/utils";

export interface SelectableCardConfig {
  id: string;
  name: string;
  description: string;
  icon: LucideIcon;
}

interface SelectableCardProps {
  config: SelectableCardConfig;
  selected: boolean;
  onSelect: () => void;
  selectionMode: "radio" | "checkbox";
  disabled?: boolean;
  className?: string;
}

export function SelectableCard({
  config,
  selected,
  onSelect,
  selectionMode,
  disabled = false,
  className,
}: SelectableCardProps) {
  const Icon = config.icon;

  return (
    <button
      type="button"
      onClick={onSelect}
      disabled={disabled}
      className={cn(
        "relative flex flex-col items-center gap-2 p-4 rounded-lg border-2 transition-all",
        "focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 focus:ring-offset-slate-900",
        selected
          ? "border-blue-500 bg-blue-500/10"
          : "border-slate-700 bg-slate-800/50 hover:border-slate-600 hover:bg-slate-800",
        disabled && "opacity-50 cursor-not-allowed",
        className
      )}
    >
      {/* Selection indicator */}
      <div
        className={cn(
          "absolute top-2 right-2 w-5 h-5 rounded-full border-2 flex items-center justify-center transition-colors",
          selectionMode === "radio" ? "rounded-full" : "rounded",
          selected
            ? "border-blue-500 bg-blue-500"
            : "border-slate-600 bg-transparent"
        )}
      >
        {selected && <Check className="h-3 w-3 text-white" />}
      </div>

      {/* Icon */}
      <div
        className={cn(
          "w-10 h-10 rounded-lg flex items-center justify-center transition-colors",
          selected
            ? "bg-blue-500/20 text-blue-400"
            : "bg-slate-700/50 text-slate-400"
        )}
      >
        <Icon className="h-5 w-5" />
      </div>

      {/* Name */}
      <span
        className={cn(
          "text-sm font-medium transition-colors",
          selected ? "text-blue-400" : "text-slate-200"
        )}
      >
        {config.name}
      </span>

      {/* Description */}
      <span className="text-xs text-slate-500 text-center leading-tight">
        {config.description}
      </span>
    </button>
  );
}

// Compact variant for permission/effort cards
interface CompactSelectableCardProps extends Omit<SelectableCardProps, "className"> {
  compact?: boolean;
}

export function CompactSelectableCard({
  config,
  selected,
  onSelect,
  selectionMode,
  disabled = false,
  compact = true,
}: CompactSelectableCardProps) {
  const Icon = config.icon;

  return (
    <button
      type="button"
      onClick={onSelect}
      disabled={disabled}
      className={cn(
        "relative flex items-center gap-3 px-3 py-2 rounded-lg border-2 transition-all text-left",
        "focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 focus:ring-offset-slate-900",
        selected
          ? "border-blue-500 bg-blue-500/10"
          : "border-slate-700 bg-slate-800/50 hover:border-slate-600 hover:bg-slate-800",
        disabled && "opacity-50 cursor-not-allowed"
      )}
    >
      {/* Icon */}
      <div
        className={cn(
          "w-8 h-8 rounded-md flex items-center justify-center shrink-0 transition-colors",
          selected
            ? "bg-blue-500/20 text-blue-400"
            : "bg-slate-700/50 text-slate-400"
        )}
      >
        <Icon className="h-4 w-4" />
      </div>

      {/* Content */}
      <div className="flex-1 min-w-0">
        <span
          className={cn(
            "text-sm font-medium block transition-colors",
            selected ? "text-blue-400" : "text-slate-200"
          )}
        >
          {config.name}
        </span>
        <span className="text-xs text-slate-500 truncate block">
          {config.description}
        </span>
      </div>

      {/* Selection indicator */}
      <div
        className={cn(
          "w-5 h-5 border-2 flex items-center justify-center shrink-0 transition-colors",
          selectionMode === "radio" ? "rounded-full" : "rounded",
          selected
            ? "border-blue-500 bg-blue-500"
            : "border-slate-600 bg-transparent"
        )}
      >
        {selected && <Check className="h-3 w-3 text-white" />}
      </div>
    </button>
  );
}
