/**
 * @vrooliComponentSource react-component-library:BottomNav
 * @vrooliComponentVersion 1.2.0
 * @vrooliComponentAdoption 3d6eafdd-bc70-4404-861d-e64d16386467
 * @vrooliComponentAppliedAt 2026-07-09T04:31:15Z
 * @vrooliComponentSourceSha256 852e5580fbfcc836b25ac8f54fea55df9c86c4225416b11569c9b63abbce3f14
 * @vrooliComponentDriftHash 852e5580fbfcc836b25ac8f54fea55df9c86c4225416b11569c9b63abbce3f14
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 *
 * LOCAL EDIT (2026-08-20): the adopted source pinned the bar with `bottom-safe`,
 * a utility no consumer defined, so the bar was not pinned at all and rendered
 * at its static position — over the top chrome on a short page. It also set
 * `pb-safe pb-4`, two utilities writing the same property, so the safe-area
 * inset was silently overridden by a fixed 16px and tap targets reached the
 * viewport edge. Both are fixed here: pinned at `bottom-0`, with a single
 * `bottom-nav-safe` gutter of `max(inset, 1.5rem)` that keeps targets clear of
 * the device bottom interaction zone. Worth pushing back upstream.
 */
import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";
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

const cn = (...inputs: ClassValue[]) => twMerge(clsx(inputs));

const baseItemClass =
  "touch-target flex min-w-0 flex-1 flex-col items-center justify-center gap-0.5 px-1 py-2 text-xs transition";
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
      <span aria-hidden className="flex h-5 w-5 items-center justify-center">{item.icon}</span>
      <span className="max-w-full truncate whitespace-nowrap">{item.label}</span>
    </>
  );

  return (
    <nav
      data-testid={testId}
      aria-label={label}
      className={cn(
        "bottom-nav-safe fixed inset-x-0 bottom-0 z-30 flex border-t border-app-border bg-app-surface pl-safe pr-safe md:hidden",
        className,
      )}
    >
      {items.map((item) => {
        const classNames = cn(
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
