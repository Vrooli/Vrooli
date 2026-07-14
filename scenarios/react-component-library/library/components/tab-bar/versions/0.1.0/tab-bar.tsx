/**
 * @libraryId react-component-library:tab-bar
 * @displayName Tab Bar
 * @description Token-bound, keyboard-accessible tab strip with optional close and add actions.
 * @version 0.1.0
 * @tags ["navigation","tabs","layout"]
 * @originScenario web-console
 * @originPath ui/src/components/TabBar.tsx
 * @warning Ingested by React Component Library. Preserve this provenance header.
 */
import { useEffect, useRef, type ReactNode } from "react";

export interface TabBarItem {
  id: string;
  label: ReactNode;
  disabled?: boolean;
  badge?: ReactNode;
}

export interface TabBarProps {
  items?: TabBarItem[];
  activeId?: string | null;
  onActiveChange?: (id: string) => void;
  onClose?: (id: string) => void;
  onAdd?: () => void;
  addAriaLabel?: string;
  trailingActions?: ReactNode;
  className?: string;
}

/**
 * A generic replacement for the web-console workspace strip. Application
 * state, menus, and persistence stay with the consumer; this component owns
 * the reusable roving tab interaction, overflow behavior, and token styling.
 */
export function TabBar({
  items = [
    { id: "overview", label: "Overview" },
    { id: "activity", label: "Activity", badge: "3" },
  ],
  activeId = items[0]?.id ?? null,
  onActiveChange = () => {},
  onClose,
  onAdd,
  addAriaLabel = "Add tab",
  trailingActions,
  className = "",
}: TabBarProps) {
  const activeTabRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    activeTabRef.current?.scrollIntoView({ block: "nearest", inline: "nearest" });
  }, [activeId]);

  const moveFocus = (currentId: string, direction: -1 | 1 | "first" | "last") => {
    const enabled = items.filter((item) => !item.disabled);
    const currentIndex = enabled.findIndex((item) => item.id === currentId);
    if (enabled.length === 0) return;
    const next = direction === "first"
      ? enabled[0]
      : direction === "last"
        ? enabled[enabled.length - 1]
        : enabled[(currentIndex + direction + enabled.length) % enabled.length];
    if (next) onActiveChange(next.id);
  };

  return (
    <div className={`flex h-9 items-stretch border-b border-app-border bg-app-surface ${className}`}>
      <div className="flex flex-1 items-stretch overflow-x-auto" role="tablist" aria-label="Tabs">
        {items.map((item) => {
          const active = item.id === activeId;
          return (
            <div key={item.id} className="group flex shrink-0 items-stretch border-r border-app-border">
              <button
                ref={active ? activeTabRef : undefined}
                type="button"
                role="tab"
                aria-selected={active}
                disabled={item.disabled}
                onClick={() => onActiveChange(item.id)}
                onKeyDown={(event) => {
                  if (event.key === "ArrowRight") { event.preventDefault(); moveFocus(item.id, 1); }
                  if (event.key === "ArrowLeft") { event.preventDefault(); moveFocus(item.id, -1); }
                  if (event.key === "Home") { event.preventDefault(); moveFocus(item.id, "first"); }
                  if (event.key === "End") { event.preventDefault(); moveFocus(item.id, "last"); }
                }}
                className={
                  "flex items-center gap-2 px-3 text-xs transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-app-primary " +
                  (active
                    ? "bg-app-background font-medium text-app-foreground shadow-[inset_0_-2px_0_0_var(--rcl-color-primary)]"
                    : "text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-foreground")
                }
              >
                <span className="max-w-32 truncate">{item.label}</span>
                {item.badge ? <span className="rounded-pill bg-app-primary px-1.5 py-0.5 text-[10px] font-semibold text-app-primary-foreground">{item.badge}</span> : null}
              </button>
              {onClose ? (
                <button
                  type="button"
                  className="px-1.5 text-app-muted-foreground opacity-0 transition hover:bg-app-surface-muted hover:text-app-foreground focus:opacity-100 group-hover:opacity-100"
                  aria-label={`Close ${typeof item.label === "string" ? item.label : "tab"}`}
                  onClick={() => onClose(item.id)}
                >
                  <span aria-hidden>×</span>
                </button>
              ) : null}
            </div>
          );
        })}
      </div>
      {trailingActions}
      {onAdd ? (
        <button type="button" className="mx-1 self-center rounded-pill px-2 text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-app-primary" aria-label={addAriaLabel} onClick={onAdd}>
          <span aria-hidden>+</span>
        </button>
      ) : null}
    </div>
  );
}

export default TabBar;
