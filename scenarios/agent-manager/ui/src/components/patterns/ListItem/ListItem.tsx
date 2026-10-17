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

interface BoundedListProps<T> {
  items: T[];
  getKey: (item: T, index: number) => React.Key;
  renderItem: (item: T, index: number) => React.ReactNode;
  initialCount?: number;
  increment?: number;
}

const DEFAULT_INITIAL_LIST_COUNT = 80;
const DEFAULT_LIST_INCREMENT = 80;

export function BoundedList<T>({
  items,
  getKey,
  renderItem,
  initialCount = DEFAULT_INITIAL_LIST_COUNT,
  increment = DEFAULT_LIST_INCREMENT,
}: BoundedListProps<T>) {
  const [visibleCount, setVisibleCount] = React.useState(() =>
    Math.min(items.length, initialCount)
  );

  React.useEffect(() => {
    setVisibleCount((current) => {
      if (items.length <= initialCount) return items.length;
      return Math.min(Math.max(current, initialCount), items.length);
    });
  }, [initialCount, items.length]);

  const visibleItems = React.useMemo(
    () => items.slice(0, visibleCount),
    [items, visibleCount]
  );
  const hiddenCount = items.length - visibleItems.length;

  return (
    <>
      {visibleItems.map((item, index) => (
        <React.Fragment key={getKey(item, index)}>
          {renderItem(item, index)}
        </React.Fragment>
      ))}
      {hiddenCount > 0 && (
        <div className="border-b border-border px-4 py-3">
          <button
            type="button"
            className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
            onClick={() =>
              setVisibleCount((current) => Math.min(items.length, current + increment))
            }
          >
            Show more ({visibleItems.length} of {items.length})
          </button>
        </div>
      )}
    </>
  );
}
