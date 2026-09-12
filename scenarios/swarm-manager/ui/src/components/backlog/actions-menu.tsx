import { MoreHorizontal } from "lucide-react";
import { ActionMenu, type ActionMenuItem } from "../ui/action-menu";

export interface ActionsMenuItem {
  label: string;
  icon: React.ReactNode;
  onClick: () => void;
  destructive?: boolean;
}

export function ActionsMenu({ items }: { items: ActionsMenuItem[] }) {
  if (items.length === 0) return null;

  const menuItems: ActionMenuItem[] = items.map((item) => ({
    label: item.label,
    icon: item.icon,
    onSelect: item.onClick,
    destructive: item.destructive,
  }));

  return (
    <div className="relative">
      {/* Desktop: inline icon buttons */}
      <div className="hidden items-center gap-0.5 sm:flex">
        {items.map((item) => (
          <button
            key={item.label}
            type="button"
            onClick={(e) => { e.preventDefault(); item.onClick(); }}
            className={`rounded p-1 text-slate-500 ${item.destructive ? "hover:text-red-400" : "hover:text-slate-300"}`}
            title={item.label}
          >
            {item.icon}
          </button>
        ))}
      </div>
      {/* Mobile: ellipsis dropdown */}
      <div className="sm:hidden">
        <ActionMenu
          items={menuItems}
          label="Actions"
          triggerIcon={<MoreHorizontal className="h-4 w-4" />}
        />
      </div>
    </div>
  );
}
