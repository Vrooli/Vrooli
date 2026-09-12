/** @vrooliComponentSource overlays.popover */
import type { ReactNode } from "react";
import { ChevronDown } from "lucide-react";
import { Popover, PopoverParts, usePopover } from "@vrooli/react-component-library/Popover/1";
import { cn } from "../../lib/utils";

interface AnchoredMenuProps {
  label: string;
  summary?: string;
  children: ReactNode;
  triggerTestId: string;
  panelTestId: string;
  icon?: ReactNode;
  className?: string;
  compactOnMobile?: boolean;
}

/** Form disclosure built on the released Popover contract. */
export function AnchoredMenu({
  label,
  summary,
  children,
  triggerTestId,
  panelTestId,
  icon,
  className,
  compactOnMobile = false,
}: AnchoredMenuProps) {
  return (
    <Popover placement="bottom-start">
      <div className={cn("relative min-w-0", className)}>
        <PopoverParts.Trigger
          data-testid={triggerTestId}
          aria-haspopup="dialog"
          className={cn(
            "touch-target inline-flex h-touch min-h-touch max-w-full items-center justify-center gap-space-2xs rounded-control border border-app-border bg-app-surface px-space-xs text-xs font-medium text-app-foreground transition hover:bg-app-surface-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50",
            compactOnMobile && "max-sm:w-touch max-sm:px-0",
          )}
        >
          {icon}
          <span className={cn("truncate", compactOnMobile && "max-sm:hidden")}>{label}</span>
          {summary && (
            <span
              className={cn(
                "max-w-control-wide break-words text-app-muted-foreground",
                compactOnMobile && "max-sm:hidden",
              )}
            >
              {summary}
            </span>
          )}
          <ChevronDown
            aria-hidden
            className={cn(
              "h-icon-compact w-icon-compact shrink-0 transition-transform",
              compactOnMobile && "max-sm:hidden",
            )}
          />
        </PopoverParts.Trigger>
        <AnchoredMenuContent panelTestId={panelTestId} label={label}>
          {children}
        </AnchoredMenuContent>
      </div>
    </Popover>
  );
}

function AnchoredMenuContent({
  panelTestId,
  label,
  children,
}: {
  panelTestId: string;
  label: string;
  children: ReactNode;
}) {
  const popover = usePopover();
  return (
    <PopoverParts.Content
      data-testid={panelTestId}
      aria-label={label}
      className="fixed overflow-y-auto p-space-xs text-app-foreground"
      onKeyDown={(event) => {
        if (event.key === "Escape") {
          event.preventDefault();
          popover.setOpen(false);
        }
      }}
    >
      {children}
    </PopoverParts.Content>
  );
}
