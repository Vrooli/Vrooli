/**
 * @libraryId react-component-library:BottomNav
 * @displayName Bottom Nav
 * @description Mobile-first bottom navigation bar with safe-area spacing, active state styling, and accessible link semantics.
 * @version 1.3.7
 * @tags ["layout","navigation","mobile"]
 * @deps {"react":"^18"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

import type { MouseEvent, ReactNode } from "react";
import { bottomNavStyles } from "./styles";

export interface BottomNavItem {
  id: string;
  label: string;
  icon: ReactNode;
  href?: string;
  active?: boolean;
  disabled?: boolean;
  ariaLabel?: string;
  testId?: string;
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
}

const joinClasses = (...classes: Array<string | undefined | false>) =>
  classes.filter(Boolean).join(" ");

const activeClass = "text-app-primary";
const inactiveClass = "text-app-muted-foreground hover:text-app-foreground";

export const BottomNav = withClassName(function BottomNav({
  items,
  label,
  testId = "bottom-nav",
  onItemSelect,
  className,
  itemClassName: itemClassNameOverride,
  activeItemClassName,
  inactiveItemClassName,
}: BottomNavProps) {
  const renderItemContent = (item: BottomNavItem) => (
    <>
      <span data-rcl-bottom-nav-icon aria-hidden="true">
        {item.icon}
      </span>
      <span data-rcl-bottom-nav-label>{item.label}</span>
    </>
  );

  return (
    <>
      <style data-rcl-bottom-nav-styles dangerouslySetInnerHTML={{ __html: bottomNavStyles }} />
      <nav
        data-testid={testId}
        data-rcl-bottom-nav
        aria-label={label}
        className={joinClasses("pb-safe pl-safe pr-safe", className)}
      >
        {items.map((item) => {
          const itemClassName = joinClasses(
            item.active ? activeClass : inactiveClass,
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
          };

          if (item.href) {
            return (
              <a
                key={item.id}
                href={item.disabled ? undefined : item.href}
                aria-disabled={item.disabled ? "true" : undefined}
                data-testid={item.testId ?? "navigation.bottom-navigation"}
                {...commonProps}
              >
                {renderItemContent(item)}
              </a>
            );
          }

          return (
            <button
              key={item.id}
              type="button"
              disabled={item.disabled}
              data-testid={item.testId ?? "navigation.bottom-navigation"}
              {...commonProps}
            >
              {renderItemContent(item)}
            </button>
          );
        })}
      </nav>
    </>
  );
});
