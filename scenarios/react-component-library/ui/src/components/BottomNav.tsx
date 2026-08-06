/**
 * @vrooliComponentSource react-component-library:BottomNav
 * @vrooliComponentVersion 1.2.0
 * @vrooliComponentAdoption 7ccf61ef-1bff-42ab-9025-8edbf0599e30
 * @vrooliComponentAppliedAt 2026-08-06T03:46:47Z
 * @vrooliComponentSourceSha256 852e5580fbfcc836b25ac8f54fea55df9c86c4225416b11569c9b63abbce3f14
 * @vrooliComponentDriftHash 55e0519629d893d0153c8569678369e4ba723091006c306c76beacb2f7be70a2
 * @vrooliComponentTokenTranslation bg-app-surface->bg-app-surface,border-app-border->border-app-border,text-app-foreground->text-app-foreground,text-app-muted-foreground->text-app-muted-foreground,text-app-primary->text-app-primary
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
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
        "fixed inset-x-0 bottom-0 z-30 flex border-t border-app-border bg-app-surface pl-safe pr-safe pb-safe md:hidden",
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
