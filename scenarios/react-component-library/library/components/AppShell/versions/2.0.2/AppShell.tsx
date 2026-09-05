/**
 * @libraryId react-component-library:AppShell
 * @displayName AppShell
 * @description The one application frame a generated scenario mounts: brand, a SidebarShell-owned navigation column (resizable sidebar or icon rail), a BottomNav-owned phone tab bar or drawer, an optional header strip, a skip link, and the routed main pane in scroll or fill mode. Navigation is data; the router plugs in through renderLink and onNavigate.
 * @version 2.0.2
 * @tags []
 * @deps {"react":"^18","lucide-react":"^0.424.0"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:AppShell */
import { Menu } from "lucide-react";
import { useRef, useState, type ReactNode } from "react";

import { BottomNav, type BottomNavItem } from "@vrooli/react-component-library/BottomNav/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";
import { IconButton } from "@vrooli/react-component-library/IconButton/3";
import { type NotificationBadgeTone } from "@vrooli/react-component-library/NotificationBadge/1";
import { SidebarShell } from "@vrooli/react-component-library/SidebarShell/2";
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { useBreakpoint } from "@vrooli/react-component-library/useMediaQuery/1";
import { useStrings } from "@vrooli/react-component-library/useLocale/1";

/**
 * 2.0.0 — the shell every generated scenario mounts, and does not own.
 *
 * 1.x drew its own navigation column with a hard-coded width, collapsed to a
 * wrapped row of links on a phone, and shipped a default navigation with fake
 * destinations. 2.0.0 composes the two navigation assets the library already
 * had right: `SidebarShell` owns the desktop column (persistent, resizable, or
 * a drawer), `BottomNav` owns the phone. Navigation is data, so the same item
 * list drives both; a router is plugged in through `renderLink` and
 * `onNavigate` rather than assumed. Every geometry decision below is a
 * stylesheet rule on a token, and the breakpoint is the library's `md`.
 */

export interface AppShellNavBadge {
  value: number | string;
  tone?: NotificationBadgeTone;
  /** Accessible description of the badge, e.g. "3 waiting". */
  label?: string;
}

export interface AppShellNavItem {
  /** Stable identity; also seeds the derived test ids. */
  id: string;
  label: string;
  /** Used under the icon in the rail and on the phone tab bar; falls back to `label`. */
  shortLabel?: string;
  href: string;
  icon: ReactNode;
  current?: boolean;
  disabled?: boolean;
  badge?: AppShellNavBadge;
  /** Test id for the desktop link; the phone tab derives `${testId}-tab`. */
  testId?: string;
}

export interface AppShellLinkProps {
  href: string;
  className?: string;
  "aria-current"?: "page";
  "aria-disabled"?: boolean;
  "data-testid"?: string;
  "data-rcl-app-shell-link"?: "";
  onClick?: (event: { preventDefault: () => void }) => void;
  children: ReactNode;
}

export type AppShellDensity = "sidebar" | "rail";
export type AppShellMobileNav = "tabs" | "drawer";
export type AppShellMainMode = "scroll" | "fill";

export interface AppShellProps {
  /** Product name, shown at the head of the sidebar and the phone drawer. */
  brand: ReactNode;
  /** Compact mark shown beside the brand and alone in the rail. */
  brandMark?: ReactNode;
  brandHref?: string;
  items: readonly AppShellNavItem[];
  /**
   * Router adapter for desktop links. Receives the item and the props the
   * shell would put on a plain anchor; return your router's link element with
   * those props spread. Defaults to `<a>`.
   */
  renderLink?: (item: AppShellNavItem, props: AppShellLinkProps) => ReactNode;
  /**
   * Router adapter for the phone tab bar and drawer, which select rather than
   * link. When supplied the shell prevents the default anchor navigation and
   * calls this instead.
   */
  onNavigate?: (item: AppShellNavItem) => void;
  /** `sidebar` shows icon and label in a resizable column; `rail` shows icon over a short label in a fixed narrow column. */
  density?: AppShellDensity;
  /** Below `md`: `tabs` replaces the column with a tab bar; `drawer` keeps the column as a drawer behind a menu button. */
  mobileNav?: AppShellMobileNav;
  /** Optional strip above the routed content. Absent by default; the shell spends no chrome it was not asked for. */
  header?: ReactNode;
  /** Foot of the navigation column: session, settings, theme. */
  utility?: ReactNode;
  /** `scroll` pads and scrolls the routed content; `fill` hands the page the full pane so it can pin its own chrome. */
  mainMode?: AppShellMainMode;
  children: ReactNode;
  navigationLabel?: string;
  mobileNavigationLabel?: string;
  skipLabel?: string;
  menuLabel?: string;
  closeLabel?: string;
  /** Persist the resized sidebar width under this key. */
  sidebarStorageKey?: string;
  /** Root test id; every part derives from it. */
  testId?: string;
  className?: string;
  mainClassName?: string;
}

export const appShellStyles = `
[data-rcl-app-shell] { position: relative; display: flex; flex-direction: column; block-size: var(--rcl-app-shell-block-size, 100dvh); min-block-size: 0; min-inline-size: 0; background: var(--color-background); color: var(--color-foreground); }
[data-rcl-app-shell-skip] { position: absolute; inset-block-start: var(--space-2xs); inset-inline-start: var(--space-2xs); z-index: var(--layer-modal); padding: var(--space-2xs) var(--space-sm); border-radius: var(--radius-control); background: var(--color-primary); color: var(--color-primary-foreground); font: var(--text-label); text-decoration: none; transform: translateY(-200%); transition: transform var(--dur-quick) var(--ease-standard); }
[data-rcl-app-shell-skip]:focus-visible { transform: none; outline: var(--border-focus) solid var(--color-focus); outline-offset: 2px; }
[data-rcl-app-shell-body] { position: relative; display: flex; flex: 1; min-block-size: 0; min-inline-size: 0; }
[data-rcl-app-shell-sidebar] { display: contents; }
[data-rcl-app-shell-column] { display: flex; flex-direction: column; block-size: 100%; min-block-size: 0; min-inline-size: 0; }
[data-rcl-app-shell][data-density="rail"] [data-rcl-sidebar-shell][data-mode="persistent"] { inline-size: calc(var(--space-xl) + var(--space-lg)); }
[data-rcl-app-shell] .rcl-app-shell__brand { display: flex; align-items: center; gap: var(--space-xs); min-block-size: var(--control-height); padding: var(--space-xs) var(--space-sm); color: var(--color-foreground); text-decoration: none; font: var(--text-heading-sm); }
[data-rcl-app-shell] .rcl-app-shell__brand:hover { color: var(--color-foreground); }
[data-rcl-app-shell-brand-mark] { display: grid; flex: none; place-items: center; inline-size: var(--control-size-xs); block-size: var(--control-size-xs); border-radius: var(--radius-control); background: var(--color-foreground); color: var(--color-background); }
[data-rcl-app-shell-brand-mark] svg { inline-size: var(--icon-size-md); block-size: var(--icon-size-md); }
[data-rcl-app-shell-brand-label] { min-inline-size: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
[data-rcl-app-shell][data-density="rail"] .rcl-app-shell__brand { justify-content: center; padding-inline: var(--space-2xs); }
[data-rcl-app-shell][data-density="rail"] [data-rcl-app-shell-brand-label] { position: absolute; inline-size: 1px; block-size: 1px; overflow: hidden; clip: rect(0 0 0 0); clip-path: inset(50%); white-space: nowrap; }
[data-rcl-app-shell-list] { display: grid; gap: var(--space-3xs); margin: 0; padding: var(--space-2xs); list-style: none; min-inline-size: 0; }
[data-rcl-app-shell-list] > li { min-inline-size: 0; }
[data-rcl-app-shell-link] { position: relative; display: flex; min-inline-size: 0; max-inline-size: 100%; align-items: center; gap: var(--space-xs); min-block-size: var(--tap-target-min); padding: var(--space-2xs) var(--space-xs); border-radius: var(--radius-control); color: var(--color-muted-foreground); font: var(--text-label); text-decoration: none; transition: background-color var(--dur-quick) var(--ease-standard), color var(--dur-quick) var(--ease-standard); }
[data-rcl-app-shell-link]:hover { background: var(--color-surface-muted); color: var(--color-foreground); }
[data-rcl-app-shell-link][aria-current="page"] { background: color-mix(in srgb, var(--color-primary) 10%, var(--color-surface)); color: var(--color-foreground); font-weight: 600; box-shadow: inset var(--space-3xs) 0 var(--color-primary); }
[data-rcl-app-shell-link][aria-disabled="true"] { opacity: var(--opacity-disabled); pointer-events: none; }
[data-rcl-app-shell-link]:focus-visible { outline: var(--border-focus) solid var(--color-focus); outline-offset: 2px; }
[data-rcl-app-shell-link-icon] { display: grid; flex: none; place-items: center; inline-size: var(--icon-size-md); block-size: var(--icon-size-md); }
[data-rcl-app-shell-link-icon] svg { inline-size: 100%; block-size: 100%; }
[data-rcl-app-shell-link-label] { min-inline-size: 0; flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
[data-rcl-app-shell][data-density="rail"] [data-rcl-app-shell-link] { flex-direction: column; justify-content: center; gap: var(--space-3xs); padding: var(--space-2xs) var(--space-3xs); font: var(--text-caption); text-align: center; }
[data-rcl-app-shell][data-density="rail"] [data-rcl-app-shell-link][aria-current="page"] { box-shadow: none; }
[data-rcl-app-shell][data-density="rail"] [data-rcl-app-shell-link-label] { flex: none; max-inline-size: 100%; }
[data-rcl-app-shell][data-density="rail"] [data-rcl-app-shell-link-badge] { position: absolute; inset-block-start: var(--space-3xs); inset-inline-start: calc(50% + var(--space-3xs)); min-inline-size: var(--space-sm); block-size: var(--space-sm); padding-inline: var(--space-4xs); box-shadow: 0 0 0 var(--border-medium) var(--color-surface); }
[data-rcl-app-shell-link-badge] { display: inline-grid; flex: none; margin-inline-start: auto; place-items: center; min-inline-size: var(--space-md); block-size: var(--space-md); padding-inline: var(--space-3xs); border-radius: var(--radius-pill); font: var(--text-caption); font-variant-numeric: tabular-nums; color: var(--color-primary-foreground); background: var(--color-warning); }
[data-rcl-app-shell-link-badge][data-tone="neutral"] { background: var(--color-surface-muted); color: var(--color-foreground); }
[data-rcl-app-shell-link-badge][data-tone="info"] { background: var(--color-info); }
[data-rcl-app-shell-link-badge][data-tone="success"] { background: var(--color-success); }
[data-rcl-app-shell-link-badge][data-tone="danger"] { background: var(--color-danger); }
[data-rcl-app-shell-utility] { margin-block-start: auto; border-block-start: var(--border-hairline) solid var(--color-border); padding: var(--space-2xs); }
[data-rcl-app-shell][data-density="rail"] [data-rcl-app-shell-utility] { text-align: center; overflow-wrap: anywhere; }
[data-rcl-app-shell-content] { display: flex; flex: 1; flex-direction: column; min-block-size: 0; min-inline-size: 0; }
[data-rcl-app-shell-header] { display: flex; flex: none; align-items: center; gap: var(--space-xs); min-block-size: var(--control-height-lg); padding-inline: var(--space-sm); border-block-end: var(--border-hairline) solid var(--color-border); background: var(--color-surface); }
[data-rcl-app-shell-header][data-mobile-only="true"] { display: none; }
[data-rcl-app-shell-header-brand] { display: none; min-inline-size: 0; flex: 1; }
[data-rcl-app-shell-menu] { display: none; }
[data-rcl-app-shell-main] { flex: 1; min-block-size: 0; min-inline-size: 0; overflow: auto; padding: clamp(var(--space-sm), 3vw, var(--space-lg)); padding-block-end: calc(clamp(var(--space-sm), 3vw, var(--space-lg)) + var(--rcl-safe-bottom, env(safe-area-inset-bottom, 0px))); }
[data-rcl-app-shell][data-main-mode="fill"] [data-rcl-app-shell-main] { display: flex; flex-direction: column; overflow: hidden; padding: 0; }
[data-rcl-app-shell-tabs] { display: none; flex: none; }
@media (max-width: 47.999rem) {
  [data-rcl-app-shell][data-mobile-nav="tabs"] [data-rcl-app-shell-sidebar] { display: none; }
  [data-rcl-app-shell][data-mobile-nav="tabs"] [data-rcl-app-shell-tabs] { display: block; }
  [data-rcl-app-shell][data-mobile-nav="tabs"] [data-rcl-app-shell-main] { padding-block-end: clamp(var(--space-sm), 3vw, var(--space-lg)); }
  [data-rcl-app-shell-header][data-mobile-only="true"] { display: flex; }
  [data-rcl-app-shell][data-mobile-nav="drawer"] [data-rcl-app-shell-menu] { display: inline-grid; }
  [data-rcl-app-shell][data-mobile-nav="drawer"] [data-rcl-app-shell-header-brand] { display: flex; }
}
@media (prefers-reduced-motion: reduce) { [data-rcl-app-shell-skip], [data-rcl-app-shell-link] { transition: none; } }
@media (forced-colors: active) { [data-rcl-app-shell-link][aria-current="page"] { outline: var(--border-medium) solid CanvasText; box-shadow: none; } [data-rcl-app-shell-brand-mark] { border: var(--border-hairline) solid CanvasText; } }
`;

function defaultRenderLink(_item: AppShellNavItem, props: AppShellLinkProps) {
  const { children, ...rest } = props;
  return <a {...rest}>{children}</a>;
}

export const AppShell = withClassName(function AppShell({
  brand,
  brandMark,
  brandHref = "/",
  items,
  renderLink = defaultRenderLink,
  onNavigate,
  density = "sidebar",
  mobileNav = "tabs",
  header,
  utility,
  mainMode = "scroll",
  children,
  navigationLabel,
  mobileNavigationLabel,
  skipLabel,
  menuLabel,
  closeLabel,
  sidebarStorageKey,
  testId = "app-shell",
  className,
  mainClassName,
}: AppShellProps) {
  const libraryStrings = useStrings();
  const resolvedNavigationLabel =
    navigationLabel ??
    libraryStrings("navigation.app-shell.primary-navigation", "Primary navigation");
  const resolvedMobileLabel =
    mobileNavigationLabel ??
    libraryStrings("navigation.app-shell.mobile-navigation", "Mobile navigation");
  const resolvedSkipLabel =
    skipLabel ?? libraryStrings("navigation.app-shell.skip-to-content", "Skip to content");
  const resolvedMenuLabel =
    menuLabel ?? libraryStrings("navigation.app-shell.open-navigation", "Open navigation");
  const resolvedCloseLabel =
    closeLabel ?? libraryStrings("navigation.app-shell.close-navigation", "Close navigation");

  const bodyRef = useRef<HTMLDivElement>(null);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const desktop = useBreakpoint("md");
  const drawer = mobileNav === "drawer";
  const mainId = `${testId}-main`;

  const select = (item: AppShellNavItem) => {
    setDrawerOpen(false);
    onNavigate?.(item);
  };

  const list = (
    <ul data-rcl-app-shell-list="">
      {items.map((item) => {
        const label = density === "rail" ? (item.shortLabel ?? item.label) : item.label;
        const badge = item.badge ? (
          <span
            data-rcl-app-shell-link-badge=""
            data-tone={item.badge.tone ?? "warning"}
            aria-label={item.badge.label}
            role={item.badge.label ? "img" : undefined}
          >
            {item.badge.value}
          </span>
        ) : null;
        const content = (
          <>
            <span data-rcl-app-shell-link-icon="" aria-hidden="true">
              {item.icon}
            </span>
            <span data-rcl-app-shell-link-label="">{label}</span>
            {badge}
          </>
        );
        const props: AppShellLinkProps = {
          href: item.href,
          "data-rcl-app-shell-link": "",
          "data-testid": item.testId,
          "aria-current": item.current ? "page" : undefined,
          "aria-disabled": item.disabled ? true : undefined,
          onClick: onNavigate
            ? (event) => {
                event.preventDefault();
                if (!item.disabled) select(item);
              }
            : drawer
              ? () => setDrawerOpen(false)
              : undefined,
          children: content,
        };
        return (
          <li key={item.id} title={density === "rail" ? item.label : undefined}>
            {renderLink(item, props)}
          </li>
        );
      })}
    </ul>
  );

  const brandItem: AppShellNavItem = { id: "brand", label: "", href: brandHref, icon: null };
  const renderBrand = (suffix: string) =>
    renderLink(brandItem, {
      href: brandHref,
      "data-testid": `${testId}-${suffix}`,
      className: "rcl-app-shell__brand",
      onClick: onNavigate
        ? (event) => {
            event.preventDefault();
            select(brandItem);
          }
        : undefined,
      children: (
        <>
          {brandMark ? <span data-rcl-app-shell-brand-mark="">{brandMark}</span> : null}
          <span data-rcl-app-shell-brand-label="">{brand}</span>
        </>
      ),
    });
  const brandNode = renderBrand("brand");

  const column = (
    <div data-rcl-app-shell-column="">
      {brandNode}
      <nav aria-label={resolvedNavigationLabel} data-testid={`${testId}-navigation`}>
        {list}
      </nav>
      {utility ? <div data-rcl-app-shell-utility="">{utility}</div> : null}
    </div>
  );

  const tabItems: BottomNavItem[] = items.map((item) => ({
    id: item.id,
    label: item.shortLabel ?? item.label,
    icon: item.icon,
    href: item.href,
    active: item.current,
    disabled: item.disabled,
    testId: item.testId ? `${item.testId}-tab` : undefined,
    badge: item.badge
      ? { value: item.badge.value, tone: item.badge.tone, label: item.badge.label }
      : undefined,
  }));

  const showHeader = header !== undefined || drawer;

  return (
    <div
      data-rcl-app-shell=""
      data-density={density}
      data-mobile-nav={mobileNav}
      data-main-mode={mainMode}
      data-testid={testId}
      className={className}
    >
      <StyleSheet name="app-shell-2-0-2" css={appShellStyles} />
      <a data-rcl-app-shell-skip="" href={`#${mainId}`} data-testid={`${testId}-skip`}>
        {resolvedSkipLabel}
      </a>
      <div data-rcl-app-shell-body="" ref={bodyRef}>
        <div data-rcl-app-shell-sidebar="">
          <SidebarShell
            mode={drawer ? "responsive" : "persistent"}
            mobileOpen={drawer && !desktop ? drawerOpen : false}
            onMobileClose={() => setDrawerOpen(false)}
            onMobileOpen={drawer ? () => setDrawerOpen(true) : undefined}
            mobileLabel={resolvedMobileLabel}
            desktopLabel={resolvedNavigationLabel}
            closeLabel={resolvedCloseLabel}
            mobileHeader={<span data-rcl-app-shell-brand-label="">{brand}</span>}
            mobileWidth="min(20rem, calc(100% - 3.5rem))"
            testId={`${testId}-sidebar`}
            {...(density === "sidebar"
              ? {
                  resizable: {
                    containerRef: bodyRef,
                    min: 200,
                    max: 360,
                    defaultSize: 256,
                    adjacentMin: 480,
                    panelName: resolvedNavigationLabel,
                    storageKey: sidebarStorageKey,
                  },
                }
              : {})}
          >
            {column}
          </SidebarShell>
        </div>
        <div data-rcl-app-shell-content="">
          {showHeader ? (
            <header
              data-rcl-app-shell-header=""
              data-mobile-only={header === undefined ? "true" : "false"}
              data-testid={`${testId}-header`}
            >
              {drawer ? (
                <IconButton
                  aria-label={resolvedMenuLabel}
                  data-rcl-app-shell-menu=""
                  data-testid={`${testId}-menu`}
                  onClick={() => setDrawerOpen(true)}
                >
                  <Menu aria-hidden="true" />
                </IconButton>
              ) : null}
              {drawer ? (
                <div data-rcl-app-shell-header-brand="">{renderBrand("header-brand")}</div>
              ) : null}
              {header}
            </header>
          ) : null}
          <main
            id={mainId}
            data-rcl-app-shell-main=""
            data-testid={mainId}
            tabIndex={-1}
            className={mainClassName}
          >
            {children}
          </main>
        </div>
      </div>
      <div data-rcl-app-shell-tabs="">
        <BottomNav
          items={tabItems}
          label={resolvedMobileLabel}
          testId={`${testId}-tabs`}
          presentation="flow"
          safeArea="inset"
          onItemSelect={
            onNavigate
              ? (tab) => {
                  const item = items.find((entry) => entry.id === tab.id);
                  if (item) select(item);
                }
              : undefined
          }
        />
      </div>
    </div>
  );
});
