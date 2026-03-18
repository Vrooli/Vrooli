import { Tooltip } from "../ui/tooltip";

export interface ActionButtonProps {
  icon: React.ReactNode;
  tooltip: string;
  onClick: () => void;
  isActive?: boolean;
  className?: string;
}

export function ActionButton({ icon, tooltip, onClick, isActive, className }: ActionButtonProps) {
  return (
    <Tooltip content={tooltip} side="top">
      <button
        onClick={onClick}
        className={`p-1.5 rounded-md transition-colors ${
          isActive
            ? "bg-indigo-500/20 text-indigo-400"
            : "hover:bg-white/10 text-slate-400 hover:text-slate-200"
        } ${className || ""}`}
        aria-label={tooltip}
      >
        {icon}
      </button>
    </Tooltip>
  );
}
