/**
 * ActionsMenu — responsive dropdown for CRUD action buttons.
 * Desktop: inline icon buttons. Mobile: ellipsis dropdown.
 */

import { useState, useRef, useEffect } from "react";
import { MoreHorizontal } from "lucide-react";

export interface ActionsMenuItem {
  label: string;
  icon: React.ReactNode;
  onClick: () => void;
  destructive?: boolean;
}

export function ActionsMenu({ items }: { items: ActionsMenuItem[] }) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const handleClick = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, [open]);

  if (items.length === 0) return null;

  return (
    <div ref={ref} className="relative">
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
        <button
          type="button"
          onClick={(e) => { e.preventDefault(); setOpen(!open); }}
          className="rounded p-1 text-slate-500 hover:text-slate-300"
          title="Actions"
        >
          <MoreHorizontal className="h-4 w-4" />
        </button>
        {open && (
          <div className="absolute right-0 z-10 mt-1 min-w-[160px] rounded-md border border-slate-700 bg-slate-900 py-1 shadow-md">
            {items.map((item) => (
              <button
                key={item.label}
                type="button"
                onClick={(e) => { e.preventDefault(); item.onClick(); setOpen(false); }}
                className={`flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm ${item.destructive ? "text-red-400 hover:bg-red-500/10" : "text-slate-300 hover:bg-slate-800"}`}
              >
                {item.icon}
                <span>{item.label}</span>
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
