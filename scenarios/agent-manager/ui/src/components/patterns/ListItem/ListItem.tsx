import * as React from "react";
import { ChevronRight } from "lucide-react";
import { cn } from "../../../lib/utils";

interface ListItemProps {
  selected?: boolean;
  highlighted?: boolean;
  onClick?: () => void;
  onKeyDown?: (e: React.KeyboardEvent) => void;
  showChevron?: boolean;
  checkbox?: React.ReactNode;
  icon?: React.ReactNode;
  children: React.ReactNode;
  actions?: React.ReactNode;
  className?: string;
}

export function ListItem({
  selected,
  highlighted,
  onClick,
  onKeyDown,
  showChevron = true,
  checkbox,
  icon,
  children,
  actions,
  className,
}: ListItemProps) {
  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (onKeyDown) {
      onKeyDown(e);
    } else if (onClick && (e.key === "Enter" || e.key === " ")) {
      e.preventDefault();
      onClick();
    }
  };

  return (
    <div
      className={cn(
        "flex items-center justify-between px-4 py-3 transition-colors border-b border-border last:border-b-0",
        onClick && "cursor-pointer",
        selected
          ? "bg-primary/10 border-l-2 border-l-primary"
          : highlighted
          ? "bg-primary/5"
          : "hover:bg-muted/50",
        className
      )}
      onClick={onClick}
      role={onClick ? "button" : undefined}
      tabIndex={onClick ? 0 : undefined}
      onKeyDown={onClick ? handleKeyDown : undefined}
    >
      <div className="flex items-center gap-3 min-w-0 flex-1">
        {checkbox}
        {icon}
        <div className="min-w-0 flex-1">{children}</div>
      </div>
      <div className="flex items-center gap-2 flex-shrink-0">
        {actions}
        {showChevron && (
          <ChevronRight className="h-4 w-4 text-muted-foreground flex-shrink-0" />
        )}
      </div>
    </div>
  );
}

interface ListItemTitleProps {
  children: React.ReactNode;
  className?: string;
}

export function ListItemTitle({ children, className }: ListItemTitleProps) {
  return (
    <p className={cn("font-medium text-sm truncate", className)}>{children}</p>
  );
}

interface ListItemSubtitleProps {
  children: React.ReactNode;
  className?: string;
}

export function ListItemSubtitle({
  children,
  className,
}: ListItemSubtitleProps) {
  return (
    <p className={cn("text-xs text-muted-foreground truncate", className)}>
      {children}
    </p>
  );
}
