/**
 * @vrooliComponentSource react-component-library:AppNavigation
 * @vrooliComponentVersion 1.1.0
 * @vrooliComponentAdoption c503e6de-f602-43df-bc0b-c2e5b9cce4cd
 * @vrooliComponentAppliedAt 2026-08-18T01:12:43Z
 * @vrooliComponentSourceSha256 4b7339bac5ea3161c7220540110fe378e63f79f908f7b60ae87428fa549aa6ab
 * @vrooliComponentDriftHash 1ec676240fb7cffeebe7881bc8f8cde0c455612e208958ae7486c9ef8a2f6585
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import type { ReactNode } from "react";
import { Home, LayoutGrid, Settings } from "lucide-react";

/**
 * 1.1.0 — icons sized in CSS instead of through the icon library's `size` prop.
 *
 * 1.0.0 passed `size="var(--space-sm)"` to the lucide icons in `defaultItems`.
 * That prop is forwarded straight to the SVG `width`/`height` attributes, whose
 * grammar is `<length>`; `var()` is not a length, so the browser rejected them:
 *
 *   Error: <svg> attribute width: Expected length, "var(--space-sm)"
 *
 * With no accepted geometry the icons fell back to the replaced-element default
 * and rendered far larger than the nav rows containing them. Sizing them from
 * the stylesheet keeps the token indirection the prop was reaching for while
 * using a property that actually accepts it, and matches how SidebarShell
 * already sizes its icons.
 */
const appNavigationStyles = `
[data-rcl-app-navigation] { display: grid; min-inline-size: 0; gap: var(--space-sm); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); background: var(--color-surface); color: var(--color-foreground); padding: var(--space-sm); box-shadow: var(--elev-raised); }
[data-rcl-app-navigation] [data-rcl-app-navigation-brand] { display: flex; min-block-size: var(--tap-target-min); align-items: center; gap: var(--space-xs); border-block-end: var(--border-hairline) solid var(--color-border); color: var(--color-foreground); padding: 0 var(--space-2xs) var(--space-sm); font: var(--text-heading-sm); }
[data-rcl-app-navigation] [data-rcl-app-navigation-mark] { display: grid; inline-size: var(--space-lg); block-size: var(--space-lg); flex: 0 0 auto; place-items: center; border-radius: var(--radius-control); background: var(--color-primary); color: var(--color-primary-foreground); }
[data-rcl-app-navigation] [data-rcl-app-navigation-list] { display: grid; gap: var(--space-3xs); margin: 0; padding: 0; list-style: none; }
[data-rcl-app-navigation] [data-rcl-app-navigation-list] a { display: flex; min-block-size: var(--tap-target-min); align-items: center; gap: var(--space-xs); border-radius: var(--radius-control); color: var(--color-muted-foreground); padding: var(--space-2xs) var(--space-xs); font: var(--text-body-sm); text-decoration: none; transition: background-color var(--dur-quick) var(--ease-standard), color var(--dur-quick) var(--ease-standard); }
[data-rcl-app-navigation] [data-rcl-app-navigation-list] a:hover { background: var(--color-surface-muted); color: var(--color-foreground); }
[data-rcl-app-navigation] [data-rcl-app-navigation-list] a[aria-current="page"] { background: color-mix(in srgb, var(--color-primary) 10%, var(--color-surface)); color: var(--color-foreground); font-weight: 650; box-shadow: inset var(--space-3xs) 0 var(--color-primary); }
[data-rcl-app-navigation] [data-rcl-app-navigation-list] svg { inline-size: var(--icon-size-sm); block-size: var(--icon-size-sm); flex: 0 0 auto; color: var(--color-primary); }
[data-rcl-app-navigation] [data-rcl-app-navigation-list] span { min-inline-size: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
[data-rcl-app-navigation] a:focus-visible { outline: var(--border-focus) solid var(--color-focus); outline-offset: var(--space-4xs); }
[data-rcl-app-navigation][data-viewport-mode="mobile"] { border: 0; border-block-start: var(--border-hairline) solid var(--color-border); border-radius: 0; box-shadow: var(--elev-floating); padding: var(--space-2xs) var(--space-xs) calc(var(--space-2xs) + env(safe-area-inset-bottom, 0px)); }
[data-rcl-app-navigation][data-viewport-mode="mobile"] [data-rcl-app-navigation-brand] { display: none; }
[data-rcl-app-navigation][data-viewport-mode="mobile"] [data-rcl-app-navigation-list] { display: flex; justify-content: space-around; gap: var(--space-3xs); }
[data-rcl-app-navigation][data-viewport-mode="mobile"] [data-rcl-app-navigation-list] a { min-inline-size: 0; flex: 1; justify-content: center; padding-inline: var(--space-2xs); }
[data-rcl-app-navigation][data-viewport-mode="mobile"] [data-rcl-app-navigation-list] span { display: none; }
[data-rcl-app-navigation][data-viewport-mode="tablet"] { border-radius: 0 var(--radius-panel) var(--radius-panel) 0; }
@media (prefers-reduced-motion: reduce) { [data-rcl-app-navigation] * { transition: none; } }
@media (forced-colors: active) { [data-rcl-app-navigation] [data-rcl-app-navigation-list] a[aria-current="page"] { background: Highlight; color: HighlightText; } }
`;

type NavigationItem = {
  label: string;
  href?: string;
  icon?: ReactNode;
  current?: boolean;
};

const defaultItems: NavigationItem[] = [
  {
    label: "Home",
    href: "/",
    current: true,
    icon: <Home aria-hidden />,
  },
  {
    label: "Library",
    href: "/library",
    icon: <LayoutGrid aria-hidden />,
  },
  {
    label: "Settings",
    href: "/settings",
    icon: <Settings aria-hidden />,
  },
];

/**
 * `brand` is optional as of 1.1.0: when omitted, the brand block is not
 * rendered at all.
 *
 * Previously it defaulted to a literal string and the block always rendered, so
 * any page that already had its own header — the workspace sidebar, which also
 * carries the collapse and settings controls — ended up with the product name
 * twice. A component that unconditionally renders chrome cannot be composed
 * into a surface that already provides it, and the page has no way to opt out.
 * Making the block conditional lets the composing surface decide who owns the
 * brand, which is the only place that knows.
 */
export function AppNavigation({
  mode = "desktop",
  items = defaultItems,
  children,
  brand,
}: {
  mode?: "mobile" | "tablet" | "desktop" | "wide";
  items?: NavigationItem[];
  children?: ReactNode;
  brand?: string;
}) {
  return (
    <>
      <style
        data-rcl-app-navigation-styles
        dangerouslySetInnerHTML={{ __html: appNavigationStyles }}
      />
      <div
        data-responsive-transformation="sidebar-to-drawer modal-to-bottom-sheet header-to-bottom-navigation"
        data-viewport-mode={mode}
        data-presentation={
          mode === "mobile" ? "bottom-navigation" : mode === "tablet" ? "drawer" : "sidebar"
        }
        data-rcl-app-navigation
      >
        {brand ? (
          <div data-rcl-app-navigation-brand>
            <span data-rcl-app-navigation-mark aria-hidden="true">
              <LayoutGrid />
            </span>
            <span>{brand}</span>
          </div>
        ) : null}
        <nav aria-label="Application navigation">
          {children ?? (
            <ul data-rcl-app-navigation-list>
              {items.map((item) => (
                <li key={item.label}>
                  <a href={item.href ?? "#"} aria-current={item.current ? "page" : undefined}>
                    {item.icon}
                    <span>{item.label}</span>
                  </a>
                </li>
              ))}
            </ul>
          )}
        </nav>
      </div>
    </>
  );
}
