import { useEffect, useRef, useCallback } from "react";

export interface ContextMenuItem {
  label: string;
  icon?: React.ReactNode;
  onClick: () => void;
  variant?: "default" | "danger";
  testId?: string;
}

interface ContextMenuProps {
  isOpen: boolean;
  position: { x: number; y: number };
  items: ContextMenuItem[];
  onClose: () => void;
}

export function ContextMenu({ isOpen, position, items, onClose }: ContextMenuProps) {
  const menuRef = useRef<HTMLDivElement>(null);

  // Handle click outside to close
  const handleClickOutside = useCallback(
    (event: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        onClose();
      }
    },
    [onClose]
  );

  // Handle Escape key to close
  const handleKeyDown = useCallback(
    (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        onClose();
      }
    },
    [onClose]
  );

  // Handle scroll to close
  const handleScroll = useCallback(() => {
    onClose();
  }, [onClose]);

  useEffect(() => {
    if (!isOpen) return;

    document.addEventListener("mousedown", handleClickOutside);
    document.addEventListener("keydown", handleKeyDown);
    window.addEventListener("scroll", handleScroll, true);

    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
      document.removeEventListener("keydown", handleKeyDown);
      window.removeEventListener("scroll", handleScroll, true);
    };
  }, [isOpen, handleClickOutside, handleKeyDown, handleScroll]);

  // Adjust position to keep menu within viewport
  useEffect(() => {
    if (!isOpen || !menuRef.current) return;

    const menu = menuRef.current;
    const rect = menu.getBoundingClientRect();
    const viewportWidth = window.innerWidth;
    const viewportHeight = window.innerHeight;

    // Adjust horizontal position if menu would overflow right edge
    if (rect.right > viewportWidth) {
      menu.style.left = `${Math.max(0, viewportWidth - rect.width - 8)}px`;
    }

    // Adjust vertical position if menu would overflow bottom edge
    if (rect.bottom > viewportHeight) {
      menu.style.top = `${Math.max(0, viewportHeight - rect.height - 8)}px`;
    }
  }, [isOpen, position]);

  if (!isOpen || items.length === 0) return null;

  return (
    <div
      ref={menuRef}
      className="fixed z-50 min-w-[160px] rounded-lg border border-slate-700 bg-slate-900 shadow-xl py-1"
      style={{ left: position.x, top: position.y }}
      role="menu"
      aria-orientation="vertical"
    >
      {items.map((item, index) => (
        <button
          key={index}
          className={`w-full flex items-center gap-2 px-3 py-2 text-sm text-left transition-colors ${
            item.variant === "danger"
              ? "text-red-400 hover:bg-red-900/30"
              : "text-slate-200 hover:bg-slate-800"
          }`}
          onClick={() => {
            item.onClick();
            onClose();
          }}
          role="menuitem"
          data-testid={item.testId}
        >
          {item.icon && <span className="flex-shrink-0">{item.icon}</span>}
          <span>{item.label}</span>
        </button>
      ))}
    </div>
  );
}
