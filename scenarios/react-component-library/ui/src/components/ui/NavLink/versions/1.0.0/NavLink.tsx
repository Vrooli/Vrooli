/**
 * @vrooliComponentSource navigation.nav-link
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption 2ff0a768-e91f-4729-b46a-cb025d3c42a4
 * @vrooliComponentAppliedAt 2026-08-12T11:41:02Z
 * @vrooliComponentSourceSha256 b89af3aaf3e12776d06fe84044562ecdd64b3ed035e11578c812efb6a7e0d010
 * @vrooliComponentDriftHash 638851cb0ec5d14b1a7d60b9c8ea8864b73b59ab036ed5a40434c690bbf8510e
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
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

export function NavLink({
  label = "Home",
  current = false,
  href = "/",
  icon,
  description,
}: NavLinkProps) {
  return (
    <>
      <style data-rcl-nav-link-styles dangerouslySetInnerHTML={{ __html: navLinkStyles }} />
      <a
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
}
