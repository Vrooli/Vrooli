/**
 * @vrooliComponentSource react-component-library:AppNavigation
 * @deps {"react":"^18","lucide-react":"^0.424.0"}
 */
import type { ReactNode } from "react";
import { Home, LayoutGrid, Settings } from "lucide-react";

const appNavigationStyles = `
[data-rcl-app-navigation] { display: grid; min-inline-size: 0; gap: var(--space-sm); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); background: var(--color-surface); color: var(--color-foreground); padding: var(--space-sm); box-shadow: var(--elev-raised); }
[data-rcl-app-navigation] [data-rcl-app-navigation-brand] { display: flex; min-block-size: var(--tap-target-min); align-items: center; gap: var(--space-xs); border-block-end: var(--border-hairline) solid var(--color-border); color: var(--color-foreground); padding: 0 var(--space-2xs) var(--space-sm); font: var(--text-heading-sm); }
[data-rcl-app-navigation] [data-rcl-app-navigation-mark] { display: grid; inline-size: var(--space-lg); block-size: var(--space-lg); flex: 0 0 auto; place-items: center; border-radius: var(--radius-control); background: var(--color-primary); color: var(--color-primary-foreground); }
[data-rcl-app-navigation] [data-rcl-app-navigation-list] { display: grid; gap: var(--space-3xs); margin: 0; padding: 0; list-style: none; }
[data-rcl-app-navigation] [data-rcl-app-navigation-list] a { display: flex; min-block-size: var(--tap-target-min); align-items: center; gap: var(--space-xs); border-radius: var(--radius-control); color: var(--color-muted-foreground); padding: var(--space-2xs) var(--space-xs); font: var(--text-body-sm); text-decoration: none; transition: background-color var(--dur-quick) var(--ease-standard), color var(--dur-quick) var(--ease-standard); }
[data-rcl-app-navigation] [data-rcl-app-navigation-list] a:hover { background: var(--color-surface-muted); color: var(--color-foreground); }
[data-rcl-app-navigation] [data-rcl-app-navigation-list] a[aria-current="page"] { background: color-mix(in srgb, var(--color-primary) 10%, var(--color-surface)); color: var(--color-foreground); font-weight: 650; box-shadow: inset var(--space-3xs) 0 var(--color-primary); }
[data-rcl-app-navigation] [data-rcl-app-navigation-list] svg { color: var(--color-primary); }
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
    icon: <Home aria-hidden size="var(--space-sm)" />,
  },
  {
    label: "Library",
    href: "/library",
    icon: <LayoutGrid aria-hidden size="var(--space-sm)" />,
  },
  {
    label: "Settings",
    href: "/settings",
    icon: <Settings aria-hidden size="var(--space-sm)" />,
  },
];

export function AppNavigation({
  mode = "desktop",
  items = defaultItems,
  children,
  brand = "Component Library",
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
          mode === "mobile"
            ? "bottom-navigation"
            : mode === "tablet"
              ? "drawer"
              : "sidebar"
        }
        data-rcl-app-navigation
      >
        <div data-rcl-app-navigation-brand>
          <span data-rcl-app-navigation-mark aria-hidden="true">
            <LayoutGrid size="var(--space-sm)" />
          </span>
          <span>{brand}</span>
        </div>
        <nav aria-label="Application navigation">
          {children ?? (
            <ul data-rcl-app-navigation-list>
              {items.map((item) => (
                <li key={item.label}>
                  <a
                    href={item.href ?? "#"}
                    aria-current={item.current ? "page" : undefined}
                  >
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
