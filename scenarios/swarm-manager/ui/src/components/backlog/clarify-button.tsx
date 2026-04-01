import { HelpCircle } from "lucide-react";
import { cn } from "../../lib";

interface ClarifyButtonProps {
  disabled?: boolean;
  isActive?: boolean;
  onClick: () => void;
}

export function ClarifyButton({ disabled, isActive, onClick }: ClarifyButtonProps) {
  return (
    <button
      type="button"
      disabled={disabled}
      onClick={onClick}
      className={cn(
        "shrink-0 rounded p-1 transition-colors",
        isActive
          ? "text-cyan-400 bg-cyan-500/10"
          : "text-slate-500 hover:text-cyan-400 hover:bg-cyan-500/10",
        disabled && "opacity-50 cursor-not-allowed",
      )}
      title="Ask for clarification"
    >
      <HelpCircle className="h-3.5 w-3.5" />
    </button>
  );
}
