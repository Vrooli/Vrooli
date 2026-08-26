/**
 * @libraryId react-component-library:NavLink
 * @displayName NavLink
 * @description A semantic route link with current-page state and a full navigation touch target.
 * @version 1.0.4
 * @tags ["navigation","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:NavLink */
import { translate } from "../../../../hooks/useLocale/versions/1.0.1/useLocale";
import { withClassName } from "../../../../foundations/ClassMerge/versions/1.0.1/ClassMerge";

import type { ReactNode } from "react";

const navLinkStyles = `
[data-rcl-nav-link] { display: flex; min-block-size: var(--tap-target-min); min-inline-size: 0; align-items: center; gap: var(--space-xs); border: var(--border-hairline) solid transparent; border-radius: var(--radius-control); color: var(--color-muted-foreground); padding: var(--space-2xs) var(--space-xs); font: var(--text-body-sm); text-decoration: none; transition: background-color var(--dur-quick) var(--ease-standard), border-color var(--dur-quick) var(--ease-standard), color var(--dur-quick) var(--ease-standard), transform var(--dur-quick) var(--ease-standard); }
[data-rcl-nav-link]:hover { background: var(--color-surface-muted); color: var(--color-foreground); }
[data-rcl-nav-link]:active { transform: translateY(var(--space-4xs)); }
[data-rcl-nav-link][aria-current="page"] { border-color: color-mix(in srgb, var(--color-primary) 22%, var(--color-border)); background: color-mix(in srgb, var(--color-primary) 10%, var(--color-surface)); color: var(--color-foreground); font-weight: 650; box-shadow: inset var(--space-3xs) 0 var(--color-primary), var(--elev-flat); }
[data-rcl-nav-link] [data-rcl-nav-link-icon] { display: grid; flex: 0 0 auto; place-items: center; color: var(--color-muted-foreground); }
[data-rcl-nav-link][aria-current="page"] [data-rcl-nav-link-icon] { color: var(--color-primary); }
[data-rcl-nav-link] [data-rcl-nav-link-label] { min-inline-size: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
[data-rcl-nav-link]:focus-visible { outline: var(--border-focus) solid var(--color-focus); outline-offset: var(--space-4xs); }
@media (prefers-reduced-motion: reduce) { [data-rcl-nav-link] { transition: none; } }
@media (forced-colors: active) { [data-rcl-nav-link][aria-current="page"] { border-color: Highlight; background: Highlight; color: HighlightText; } [data-rcl-nav-link][aria-current="page"] [data-rcl-nav-link-icon] { color: HighlightText; } }
`;

export interface NavLinkProps {
  label?: string;
  current?: boolean;
  href?: string;
  icon?: ReactNode;
  description?: string;
}

export const NavLink = withClassName(function NavLink({
  label = translate("navigation.nav-link.label.1", "Home"),
  current = false,
  href = "/",
  icon,
  description,
}: NavLinkProps) {
  return (
    <>
      <style data-rcl-nav-link-styles dangerouslySetInnerHTML={{ __html: navLinkStyles }} />
      <a
        data-testid="navigation.nav-link"
        href={href}
        aria-current={current ? "page" : undefined}
        data-rcl-nav-link
        title={description}
      >
        {icon ? (
          <span data-rcl-nav-link-icon aria-hidden="true">
            {icon}
          </span>
        ) : null}
        <span data-rcl-nav-link-label>{label}</span>
      </a>
    </>
  );
});
