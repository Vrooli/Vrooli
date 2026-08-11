/**
 * @vrooliComponentSource react-component-library:BottomNav
 * @vrooliComponentVersion 1.3.0
 * @vrooliComponentAdoption f8ff9216-a8af-44bb-a4b1-c443315e2ad6
 * @vrooliComponentAppliedAt 2026-08-11T00:11:48Z
 * @vrooliComponentSourceSha256 4a7627933ff5c73c1aaa1587549d66a9ce0172243f08ae13171c1d7ef43c9f0a
 * @vrooliComponentDriftHash c8bae4939d08dab3d21db664349cfe5ff25eeb1822e03b0a8b4c961cab7c53be
 * @vrooliComponentTokenTranslation text-app-foreground->text-app-foreground,text-app-muted-foreground->text-app-muted-foreground,text-app-primary->text-app-primary
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
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

export function BottomNav({
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
            "data-testid": item.testId,
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
                {...commonProps}
              >
                {renderItemContent(item)}
              </a>
            );
          }

          return (
            <button key={item.id} type="button" disabled={item.disabled} {...commonProps}>
              {renderItemContent(item)}
            </button>
          );
        })}
      </nav>
    </>
  );
}
