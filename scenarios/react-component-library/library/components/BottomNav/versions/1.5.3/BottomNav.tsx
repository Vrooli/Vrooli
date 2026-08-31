/**
 * @libraryId react-component-library:BottomNav
 * @displayName Bottom Nav
 * @version 1.5.3
 * @tags ["layout","navigation","mobile"]
 * @deps {"react":"^18"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";
import {
  NotificationBadge,
  type NotificationBadgeTone,
} from "@vrooli/react-component-library/NotificationBadge/1";
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { useLayoutEffect, useRef, useState, type MouseEvent, type ReactNode } from "react";
export const bottomNavStyles = `
[data-rcl-bottom-nav] { position: fixed; inset-inline: 0; inset-block-end: 0; z-index: var(--layer-sticky); box-sizing: border-box; border-block-start: var(--border-hairline) solid var(--color-border); background: color-mix(in srgb, var(--color-surface-raised) 96%, transparent); box-shadow: var(--elev-raised); padding-block-start: var(--space-2xs); padding-inline: env(safe-area-inset-left, 0px) env(safe-area-inset-right, 0px); backdrop-filter: blur(var(--space-xs)); }
[data-rcl-bottom-nav][data-rcl-bottom-nav-presentation="flow"] { position: relative; inset: auto; z-index: auto; }
[data-rcl-bottom-nav][data-rcl-bottom-nav-safe-area="inset"] { padding-block-end: env(safe-area-inset-bottom, 0px); }
[data-rcl-bottom-nav][data-rcl-bottom-nav-safe-area="floor"] { padding-block-end: max(var(--space-lg), env(safe-area-inset-bottom, 0px)); }
[data-rcl-bottom-nav][data-rcl-bottom-nav-safe-area="none"] { padding-block-end: 0; }
[data-rcl-bottom-nav-track] { position: relative; display: grid; grid-template-columns: repeat(auto-fit, minmax(0, 1fr)); min-inline-size: 0; }
[data-rcl-bottom-nav-item] { position: relative; display: flex; min-inline-size: 0; min-block-size: var(--tap-target-min); flex-direction: column; align-items: center; justify-content: center; gap: var(--space-3xs); border: 0; border-radius: var(--radius-control); background: transparent; color: var(--color-muted-foreground); padding: var(--space-2xs) var(--space-3xs); font: var(--text-caption); text-decoration: none; cursor: pointer; transition: background-color var(--dur-quick) var(--ease-standard), color var(--dur-quick) var(--ease-standard), transform var(--dur-quick) var(--ease-standard); }
[data-rcl-bottom-nav-item][data-active="true"] { color: var(--color-primary); font-weight: 700; }
[data-rcl-bottom-nav-item]:hover:not(:disabled) { background: var(--color-surface-muted); color: var(--color-foreground); }
[data-rcl-bottom-nav-item]:active:not(:disabled) { transform: translateY(var(--space-3xs)); }
[data-rcl-bottom-nav-item][data-disabled="true"], [data-rcl-bottom-nav-item]:disabled { cursor: not-allowed; opacity: .58; }
[data-rcl-bottom-nav-active-indicator] { position: absolute; inset-block-end: 0; inset-inline-start: 0; z-index: 1; block-size: var(--space-3xs); border-radius: var(--radius-pill); background: var(--color-primary); pointer-events: none; transition: transform var(--dur-moderate) var(--ease-standard), inline-size var(--dur-moderate) var(--ease-standard); }
[data-rcl-bottom-nav][data-rcl-bottom-nav-indicator="static"] [data-rcl-bottom-nav-active-indicator] { transition: none; }
[data-rcl-bottom-nav-icon] { display: grid; inline-size: var(--space-md); block-size: var(--space-md); flex: 0 0 auto; place-items: center; }
[data-rcl-bottom-nav-icon] svg { inline-size: 100%; block-size: 100%; }
[data-rcl-bottom-nav-label] { max-inline-size: 100%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
@media (prefers-reduced-motion: reduce) { [data-rcl-bottom-nav-active-indicator] { transition: none; } }
`;
export interface BottomNavBadge {
  value?: number | string;
  max?: number;
  dot?: boolean;
  tone?: NotificationBadgeTone;
  label?: string;
}
export interface BottomNavItem {
  id: string;
  label: string;
  icon: ReactNode;
  href?: string;
  active?: boolean;
  disabled?: boolean;
  ariaLabel?: string;
  testId?: string;
  badge?: BottomNavBadge;
}
export interface BottomNavProps {
  items: BottomNavItem[];
  label: string;
  testId?: string;
  onItemSelect?: (
    item: BottomNavItem,
    event: MouseEvent<HTMLAnchorElement | HTMLButtonElement>,
  ) => void;
  className?: string;
  itemClassName?: string;
  activeItemClassName?: string;
  inactiveItemClassName?: string;
  presentation?: "fixed" | "flow";
  safeArea?: "inset" | "floor" | "none";
  activeIndicator?: "slide" | "static" | "none";
}

const joinClasses = (...classes: Array<string | undefined | false>) =>
  classes.filter(Boolean).join(" ");

export const BottomNav = withClassName(function BottomNav({
  items,
  label,
  testId = "bottom-nav",
  onItemSelect,
  className,
  itemClassName: itemClassNameOverride,
  activeItemClassName,
  inactiveItemClassName,
  presentation = "fixed",
  safeArea = "floor",
  activeIndicator = "slide",
}: BottomNavProps) {
  const trackRef = useRef<HTMLDivElement>(null);
  const itemRefs = useRef<Record<string, HTMLElement | null>>({});
  const [indicator, setIndicator] = useState<{ left: number; width: number } | null>(null);

  useLayoutEffect(() => {
    const sync = () => {
      const track = trackRef.current;
      const active = items.find((item) => item.active);
      const element = active ? itemRefs.current[active.id] : null;
      if (!track || !element || activeIndicator === "none") {
        setIndicator(null);
        return;
      }
      const trackRect = track.getBoundingClientRect();
      const itemRect = element.getBoundingClientRect();
      setIndicator({ left: itemRect.left - trackRect.left, width: itemRect.width });
    };
    sync();
    window.addEventListener("resize", sync);
    const observer = typeof ResizeObserver === "undefined" ? null : new ResizeObserver(sync);
    if (observer && trackRef.current) observer.observe(trackRef.current);
    return () => {
      window.removeEventListener("resize", sync);
      observer?.disconnect();
    };
  }, [activeIndicator, items]);

  return (
    <>
      <StyleSheet name="bottom-nav-1-5-0" css={bottomNavStyles} />
      <nav
        data-testid={testId}
        data-rcl-bottom-nav
        data-rcl-bottom-nav-presentation={presentation}
        data-rcl-bottom-nav-safe-area={safeArea}
        data-rcl-bottom-nav-indicator={activeIndicator}
        aria-label={label}
        className={className}
      >
        <div ref={trackRef} data-rcl-bottom-nav-track>
          {items.map((item) => {
            const itemClassName = joinClasses(
              itemClassNameOverride,
              item.active ? activeItemClassName : inactiveItemClassName,
            );
            const handleClick = (event: MouseEvent<HTMLAnchorElement | HTMLButtonElement>) => {
              if (item.disabled) {
                event.preventDefault();
                return;
              }
              if (onItemSelect) event.preventDefault();
              onItemSelect?.(item, event);
            };
            const commonProps = {
              "data-rcl-bottom-nav-item": true,
              "data-active": item.active ? "true" : "false",
              "data-disabled": item.disabled ? "true" : "false",
              "aria-label": item.ariaLabel,
              "aria-current": item.active ? ("page" as const) : undefined,
              className: itemClassName,
              onClick: handleClick,
              ref: (element: HTMLElement | null) => {
                itemRefs.current[item.id] = element;
              },
            };
            const content = (
              <>
                <NotificationBadge
                  {...item.badge}
                  badgeLabel={item.badge?.label}
                  data-rcl-bottom-nav-badge
                >
                  <span data-rcl-bottom-nav-icon aria-hidden="true">
                    {item.icon}
                  </span>
                </NotificationBadge>
                <span data-rcl-bottom-nav-label>{item.label}</span>
              </>
            );
            return item.href ? (
              <a
                key={item.id}
                href={item.disabled ? undefined : item.href}
                aria-disabled={item.disabled ? "true" : undefined}
                data-testid={item.testId ?? "navigation.bottom-navigation"}
                {...commonProps}
              >
                {content}
              </a>
            ) : (
              <button
                key={item.id}
                type="button"
                disabled={item.disabled}
                data-testid={item.testId ?? "navigation.bottom-navigation"}
                {...commonProps}
              >
                {content}
              </button>
            );
          })}
          {activeIndicator !== "none" && indicator ? (
            <span
              aria-hidden="true"
              data-rcl-bottom-nav-active-indicator
              style={{
                inlineSize: `${indicator.width}px`,
                transform: `translateX(${indicator.left}px)`,
              }}
            />
          ) : null}
        </div>
      </nav>
    </>
  );
});
