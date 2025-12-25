import { cn } from "../../lib/utils";

interface BadgeProps {
  children: React.ReactNode;
  color?: string;
  className?: string;
  onClick?: () => void;
  onRemove?: () => void;
}

export function Badge({ children, color = "#6366f1", className, onClick, onRemove }: BadgeProps) {
  const isInteractive = onClick || onRemove;

  return (
    <span
      onClick={onClick}
      className={cn(
        "inline-flex items-center gap-1 px-2 py-0.5 text-xs font-medium rounded-full transition-colors",
        isInteractive && "cursor-pointer hover:opacity-80",
        className
      )}
      style={{
        backgroundColor: `${color}20`,
        color: color,
        borderColor: `${color}40`,
        borderWidth: "1px"
      }}
    >
      {children}
      {onRemove && (
        <button
          onClick={(e) => {
            e.stopPropagation();
            onRemove();
          }}
          className="ml-0.5 hover:opacity-70 transition-opacity"
          aria-label="Remove"
        >
          <svg className="h-3 w-3" viewBox="0 0 12 12" fill="currentColor">
            <path d="M4.293 4.293a1 1 0 011.414 0L6 4.586l.293-.293a1 1 0 111.414 1.414L7.414 6l.293.293a1 1 0 01-1.414 1.414L6 7.414l-.293.293a1 1 0 01-1.414-1.414L4.586 6l-.293-.293a1 1 0 010-1.414z" />
          </svg>
        </button>
      )}
    </span>
  );
}
