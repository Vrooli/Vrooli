/**
 * @libraryId react-component-library:BottomNav
 * @version 1.0.0
 * @status released
 * @deps {"react":"^18"}
 */
import type { MouseEvent, ReactNode } from "react";

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
  onItemSelect?: (item: BottomNavItem, event: MouseEvent<HTMLAnchorElement | HTMLButtonElement>) => void;
  className?: string;
  itemClassName?: string;
  activeItemClassName?: string;
  inactiveItemClassName?: string;
}

const joinClasses = (...classes: Array<string | undefined | false>) =>
  classes.filter(Boolean).join(" ");

const baseItemClass =
  "touch-target flex flex-1 flex-col items-center justify-center gap-0.5 py-2 text-xs transition";
const activeClass = "text-app-primary";
const inactiveClass = "text-app-muted-foreground hover:text-app-foreground";
const disabledClass = "cursor-not-allowed opacity-50 hover:text-app-muted-foreground";

export function BottomNav({
  items,
  label,
  testId = "bottom-nav",
  onItemSelect,
  className,
  itemClassName,
  activeItemClassName,
  inactiveItemClassName,
}: BottomNavProps) {
  const renderItemContent = (item: BottomNavItem) => (
    <>
      {item.icon}
      <span>{item.label}</span>
    </>
  );

  return (
    <nav
      data-testid={testId}
      aria-label={label}
      className={joinClasses(
        "pb-safe pl-safe pr-safe fixed inset-x-0 bottom-0 z-30 flex border-t border-app-border bg-app-surface md:hidden",
        className,
      )}
    >
      {items.map((item) => {
        const classNames = joinClasses(
          baseItemClass,
          item.active ? activeItemClassName ?? activeClass : inactiveItemClassName ?? inactiveClass,
          item.disabled && disabledClass,
          itemClassName,
        );
        const handleClick = (event: MouseEvent<HTMLAnchorElement | HTMLButtonElement>) => {
          if (item.disabled) {
            event.preventDefault();
            return;
          }
          if (onItemSelect) {
            event.preventDefault();
          }
          onItemSelect?.(item, event);
        };

        if (item.href) {
          return (
            <a
              key={item.id}
              href={item.href}
              data-testid={item.testId}
              aria-label={item.ariaLabel}
              aria-current={item.active ? "page" : undefined}
              aria-disabled={item.disabled ? "true" : undefined}
              className={classNames}
              onClick={handleClick}
            >
              {renderItemContent(item)}
            </a>
          );
        }

        return (
          <button
            key={item.id}
            type="button"
            data-testid={item.testId}
            aria-label={item.ariaLabel}
            aria-current={item.active ? "page" : undefined}
            disabled={item.disabled}
            className={classNames}
            onClick={handleClick}
          >
            {renderItemContent(item)}
          </button>
        );
      })}
    </nav>
  );
}
