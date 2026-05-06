import { useEffect, useRef, useState } from "react";
import { MoreVertical, type LucideIcon } from "lucide-react";
import { Button } from "../ui/button";
import { cn } from "../../lib/utils";

export interface SessionActionItem {
  label: string;
  icon: LucideIcon;
  onSelect: () => void;
  disabled?: boolean;
  destructive?: boolean;
  testId?: string;
  loading?: boolean;
}

interface SessionActionsMenuProps {
  items: SessionActionItem[];
  variant: "desktop" | "mobile";
  onItemSelected?: () => void;
}

export function SessionActionsMenu({ items, variant, onItemSelected }: SessionActionsMenuProps) {
  if (variant === "mobile") {
    return (
      <div className="flex flex-col gap-2 p-2">
        {items.map((item) => (
          <SessionActionButton key={item.label} item={item} onItemSelected={onItemSelected} />
        ))}
      </div>
    );
  }
  return <DesktopSessionActionsMenu items={items} onItemSelected={onItemSelected} />;
}

function DesktopSessionActionsMenu({ items, onItemSelected }: Pick<SessionActionsMenuProps, "items" | "onItemSelected">) {
  const [open, setOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const handlePointerDown = (event: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener("mousedown", handlePointerDown);
    return () => document.removeEventListener("mousedown", handlePointerDown);
  }, [open]);

  return (
    <div ref={menuRef} className="relative">
      <Button
        variant="ghost"
        size="icon"
        onClick={() => setOpen((current) => !current)}
        aria-label="Session actions"
        aria-expanded={open}
        data-testid="session-desktop-header-actions"
      >
        <MoreVertical className="h-4 w-4" />
      </Button>
      {open && (
        <div
          role="menu"
          className="absolute right-0 z-30 mt-1 min-w-[180px] rounded-md border border-slate-700 bg-slate-950 py-1 shadow-lg"
          data-testid="session-desktop-actions-menu"
        >
          {items.map((item) => (
            <SessionActionMenuItem
              key={item.label}
              item={item}
              onItemSelected={() => {
                setOpen(false);
                onItemSelected?.();
              }}
            />
          ))}
        </div>
      )}
    </div>
  );
}

function SessionActionMenuItem({ item, onItemSelected }: { item: SessionActionItem; onItemSelected?: () => void }) {
  const Icon = item.icon;
  return (
    <button
      type="button"
      role="menuitem"
      disabled={item.disabled}
      onClick={() => {
        if (item.disabled) return;
        onItemSelected?.();
        item.onSelect();
      }}
      className={cn(
        "flex w-full items-center gap-2 px-3 py-2 text-left text-sm disabled:cursor-not-allowed disabled:opacity-50",
        item.destructive ? "text-red-300 hover:bg-red-500/10" : "text-slate-200 hover:bg-slate-800",
      )}
      data-testid={item.testId}
    >
      <Icon className={cn("h-4 w-4", item.loading && "animate-spin")} />
      <span>{item.label}</span>
    </button>
  );
}

function SessionActionButton({ item, onItemSelected }: { item: SessionActionItem; onItemSelected?: () => void }) {
  const Icon = item.icon;
  return (
    <Button
      variant="ghost"
      onClick={() => {
        if (item.disabled) return;
        onItemSelected?.();
        item.onSelect();
      }}
      disabled={item.disabled}
      data-testid={item.testId}
      className={cn(item.destructive && "text-red-300 hover:bg-red-500/10 hover:text-red-200")}
    >
      <Icon className={cn("mr-2 h-4 w-4", item.loading && "animate-spin")} />
      {item.label}
    </Button>
  );
}
