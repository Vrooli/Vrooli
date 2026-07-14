import { useRef, useState, type ReactNode } from "react";
import { Loader2, MoreVertical } from "lucide-react";
import { Button, type ButtonProps } from "./button";
import { Popover } from "./popover";
import { cn } from "../../lib/utils";

export interface ActionMenuItem {
  label: string;
  onSelect: () => void;
  icon?: ReactNode;
  disabled?: boolean;
  destructive?: boolean;
  loading?: boolean;
  title?: string;
  testId?: string;
}

export interface ActionMenuProps {
  items: ActionMenuItem[];
  label?: string;
  triggerTestId?: string;
  menuTestId?: string;
  triggerIcon?: ReactNode;
  triggerVariant?: ButtonProps["variant"];
  triggerSize?: ButtonProps["size"];
  onItemSelected?: () => void;
  className?: string;
  /** Render the menu as a full-width bottom sheet on mobile (uses `label` as the sheet title). */
  mobileSheet?: boolean;
}

export function ActionMenu({
  items,
  label = "Actions",
  triggerTestId,
  menuTestId,
  triggerIcon = <MoreVertical className="h-4 w-4" />,
  triggerVariant = "ghost",
  triggerSize = "icon",
  onItemSelected,
  className,
  mobileSheet,
}: ActionMenuProps) {
  const [open, setOpen] = useState(false);
  const triggerRef = useRef<HTMLButtonElement>(null);

  if (items.length === 0) return null;

  return (
    <>
      <Button
        ref={triggerRef}
        variant={triggerVariant}
        size={triggerSize}
        // Stop the mousedown from reaching the popover's click-outside listener
        // so clicking the trigger toggles cleanly instead of closing-then-reopening.
        onMouseDown={(event) => event.stopPropagation()}
        onClick={() => setOpen((current) => !current)}
        aria-label={label}
        aria-haspopup="menu"
        aria-expanded={open}
        title={label}
        data-testid={triggerTestId}
        className={className}
      >
        {triggerIcon}
      </Button>
      <Popover
        isOpen={open}
        onClose={() => setOpen(false)}
        triggerRef={triggerRef}
        placement="bottom-end"
        mobileSheet={mobileSheet}
        mobileTitle={label}
        className="min-w-[200px] overflow-hidden py-1"
        testId={menuTestId}
      >
        <ActionMenuItems
          items={items}
          onItemSelected={() => {
            setOpen(false);
            onItemSelected?.();
          }}
          role="menu"
          itemRole="menuitem"
        />
      </Popover>
    </>
  );
}

export function ActionMenuSheetContent({
  items,
  onItemSelected,
  className,
}: {
  items: ActionMenuItem[];
  onItemSelected?: () => void;
  className?: string;
}) {
  if (items.length === 0) return null;

  return (
    <ActionMenuItems
      items={items}
      onItemSelected={onItemSelected}
      className={cn("py-1", className)}
    />
  );
}

export function ActionMenuPanel({
  children,
  className,
  testId,
}: {
  children: ReactNode;
  className?: string;
  testId?: string;
}) {
  return (
    <div
      role="menu"
      className={cn(
        "min-w-[200px] overflow-hidden rounded-md border border-white/10 bg-slate-900 py-1 shadow-lg",
        className,
      )}
      data-testid={testId}
    >
      {children}
    </div>
  );
}

export function ActionMenuItems({
  items,
  onItemSelected,
  className,
  role,
  itemRole,
}: {
  items: ActionMenuItem[];
  onItemSelected?: () => void;
  className?: string;
  role?: "menu";
  itemRole?: "menuitem";
}) {
  return (
    <div className={cn("flex flex-col", className)} role={role}>
      {items.map((item) => (
        <ActionMenuItemButton
          key={item.label}
          item={item}
          onItemSelected={onItemSelected}
          role={itemRole}
        />
      ))}
    </div>
  );
}

export function ActionMenuItemButton({
  item,
  onItemSelected,
  role,
}: {
  item: ActionMenuItem;
  onItemSelected?: () => void;
  role?: "menuitem";
}) {
  return (
    <button
      type="button"
      role={role}
      disabled={item.disabled}
      title={item.title}
      onClick={(event) => {
        event.preventDefault();
        if (item.disabled) return;
        onItemSelected?.();
        item.onSelect();
      }}
      className={cn(
        "flex h-9 w-full items-center gap-2 px-3 text-left text-sm transition-colors",
        "disabled:cursor-not-allowed disabled:opacity-50",
        "[&>svg]:h-4 [&>svg]:w-4 [&>svg]:shrink-0",
        item.destructive
          ? "text-red-300 hover:bg-red-500/10"
          : "text-slate-200 hover:bg-slate-800",
      )}
      data-testid={item.testId}
    >
      {item.loading ? <Loader2 className="animate-spin" /> : item.icon}
      <span className="min-w-0 truncate">{item.label}</span>
    </button>
  );
}

export function ActionMenuSeparator() {
  return <div className="my-1 h-px bg-slate-800" role="separator" />;
}
