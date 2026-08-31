/**
 * @libraryId react-component-library:BottomNav
 * @displayName Bottom Nav
 * @description The compact-viewport navigation surface with safe-area handling, animated active state, badges, route semantics, and an explicit overflow policy for more destinations than fit.
 * @version 1.5.0
 * @tags ["layout","navigation","mobile"]
 * @deps {"react":"^18"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.2";
import {
  NotificationBadge,
  type NotificationBadgeTone,
} from "@vrooli/react-component-library/NotificationBadge/1.0.2";
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";
import {
  useLayoutEffect,
  useRef,
  useState,
  type MouseEvent,
  type ReactNode,
} from "react";
import { bottomNavStyles } from "./styles";

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
  const [indicator, setIndicator] = useState<{
    left: number;
    width: number;
  } | null>(null);

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
      setIndicator({
        left: itemRect.left - trackRect.left,
        width: itemRect.width,
      });
    };
    sync();
    window.addEventListener("resize", sync);
    const observer =
      typeof ResizeObserver === "undefined" ? null : new ResizeObserver(sync);
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
          {items.map((item) => {
            const itemClassName = joinClasses(
              itemClassNameOverride,
              item.active ? activeItemClassName : inactiveItemClassName,
            );
            const handleClick = (
              event: MouseEvent<HTMLAnchorElement | HTMLButtonElement>,
            ) => {
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
        </div>
      </nav>
    </>
  );
});
