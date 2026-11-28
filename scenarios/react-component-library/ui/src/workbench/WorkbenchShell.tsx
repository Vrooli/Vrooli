import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { Link, NavLink, Outlet, useLocation, useNavigate } from "react-router-dom";
import {
  AppShell as LibraryAppShell,
  type AppShellLinkProps,
  type AppShellNavItem,
} from "@vrooli/react-component-library/AppShell/2";
import {
  BarChart3,
  Bell,
  ClipboardCheck,
  House,
  LayoutGrid,
  PenTool,
  Plus,
  Search,
  Settings as SettingsIcon,
} from "lucide-react";
import { useQuery } from "@tanstack/react-query";

import { useTranslation } from "../i18n";
import { appRoutes } from "../routes";
import { getReviewSummary } from "../api/review";
import { cx } from "./cx";
import { SearchPalette } from "./search/SearchPalette";
import { applyWorkbenchPreferences, readWorkbenchPreferences } from "./preferences";
import { useBrand } from "./useBrand";
import { selectors } from "../consts/selectors";
import { useGlobalKeydown } from "../hooks/useGlobalKeydown";
import "./brand.css";

function isCurrent(pathname: string, href: string): boolean {
  if (href === appRoutes.start) return pathname === appRoutes.start;
  if (href === appRoutes.library) {
    return (
      pathname === appRoutes.library ||
      pathname === appRoutes.components ||
      pathname === appRoutes.assetCatalog ||
      pathname.startsWith("/assets/")
    );
  }
  if (href === appRoutes.design) {
    return pathname.startsWith(appRoutes.design) || pathname.startsWith("/sketch/");
  }
  if (href === appRoutes.stats) {
    return (
      pathname === appRoutes.stats ||
      pathname === appRoutes.coverage ||
      pathname === appRoutes.activity ||
      pathname === appRoutes.capabilities
    );
  }
  return pathname === href;
}

function crumbFor(pathname: string): { key: string; fallback: string } {
  if (pathname === appRoutes.start) return { key: "workbench.nav.start", fallback: "Start" };
  if (pathname === appRoutes.library) return { key: "workbench.nav.library", fallback: "Library" };
  if (pathname === appRoutes.components || pathname === appRoutes.assetCatalog) {
    return { key: "workbench.nav.components", fallback: "Components" };
  }
  if (pathname.startsWith("/assets/")) {
    if (pathname.endsWith("/preview")) return { key: "workbench.nav.preview", fallback: "Preview" };
    return { key: "workbench.nav.component", fallback: "Component" };
  }
  if (pathname.startsWith(appRoutes.design) || pathname.startsWith("/sketch/")) {
    return { key: "workbench.nav.designShort", fallback: "Design" };
  }
  if (pathname === appRoutes.review) return { key: "workbench.nav.review", fallback: "Review" };
  if (pathname === appRoutes.stats) return { key: "workbench.nav.stats", fallback: "Stats" };
  if (pathname === appRoutes.coverage)
    return { key: "workbench.nav.coverage", fallback: "Coverage" };
  if (pathname === appRoutes.capabilities) {
    return { key: "workbench.nav.capabilities", fallback: "Capabilities" };
  }
  if (pathname === appRoutes.activity)
    return { key: "workbench.nav.activity", fallback: "Activity" };
  if (pathname === appRoutes.settings)
    return { key: "workbench.nav.settings", fallback: "Settings" };
  if (pathname === appRoutes.create) return { key: "workbench.nav.create", fallback: "New" };
  return { key: "workbench.nav.notFound", fallback: "Not found" };
}

function RouterLink(item: AppShellNavItem, props: AppShellLinkProps) {
  const { href, children, ...linkProps } = props;
  const className = cx(item.id === "brand" ? "wb-brand" : "wb-nav-item", linkProps.className);
  if (item.id === "brand") {
    return (
      <Link to={href} {...linkProps} className={className}>
        {children}
      </Link>
    );
  }
  return (
    <NavLink to={href} {...linkProps} className={className}>
      {children}
    </NavLink>
  );
}

export function WorkbenchShell({ children }: { children?: ReactNode }) {
  const { t } = useTranslation();
  const location = useLocation();
  const navigate = useNavigate();
  const [commandOpen, setCommandOpen] = useState(false);
  const gArmedAt = useRef(0);
  // The Start rail advertises these shortcuts, so the shell owns them. A
  // sequence is only armed when the person is not typing into a field.

  const review = useQuery({
    queryKey: ["review", "summary"],
    queryFn: getReviewSummary,
    retry: 0,
  });
  const brand = useBrand().data ?? null;
  const openFindings = review.data?.open ?? 0;

  const items: AppShellNavItem[] = [
    {
      id: "start",
      href: appRoutes.start,
      label: t("workbench.nav.start", { defaultValue: "Start" }),
      shortLabel: t("workbench.nav.start", { defaultValue: "Start" }),
      icon: <House aria-hidden size={17} />,
      current: isCurrent(location.pathname, appRoutes.start),
      testId: selectors.workbench.nav.start,
    },
    {
      id: "library",
      href: appRoutes.library,
      label: t("workbench.nav.library", { defaultValue: "Library" }),
      shortLabel: t("workbench.nav.library", { defaultValue: "Library" }),
      icon: <LayoutGrid aria-hidden size={17} />,
      current: isCurrent(location.pathname, appRoutes.library),
      testId: selectors.workbench.nav.library,
    },
    {
      id: "design",
      href: appRoutes.design,
      label: t("workbench.nav.designShort", { defaultValue: "Design" }),
      shortLabel: t("workbench.nav.designShort", { defaultValue: "Design" }),
      icon: <PenTool aria-hidden size={17} />,
      current: isCurrent(location.pathname, appRoutes.design),
      testId: selectors.workbench.nav.design,
    },
    {
      id: "review",
      href: appRoutes.review,
      label: t("workbench.nav.review", { defaultValue: "Review" }),
      shortLabel: t("workbench.nav.review", { defaultValue: "Review" }),
      icon: <ClipboardCheck aria-hidden size={17} />,
      current: isCurrent(location.pathname, appRoutes.review),
      badge: openFindings
        ? {
            value: openFindings,
            label: t("workbench.nav.openFindings", {
              defaultValue: "{{count}} open findings",
              count: openFindings,
            }),
          }
        : undefined,
      testId: selectors.workbench.nav.review,
    },
    {
      id: "stats",
      href: appRoutes.stats,
      label: t("workbench.nav.stats", { defaultValue: "Stats" }),
      shortLabel: t("workbench.nav.statsShort", { defaultValue: "Stats" }),
      icon: <BarChart3 aria-hidden size={17} />,
      current: isCurrent(location.pathname, appRoutes.stats),
      testId: selectors.workbench.nav.stats,
    },
  ];

  const openCommand = useCallback(() => {
    setCommandOpen(true);
  }, []);

  useEffect(() => {
    applyWorkbenchPreferences(readWorkbenchPreferences());
  }, []);

  useGlobalKeydown((event) => {
    if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
      event.preventDefault();
      setCommandOpen((open) => !open);
      return;
    }
    if (event.key === "Escape") {
      setCommandOpen(false);
      return;
    }
    if (event.metaKey || event.ctrlKey || event.altKey) return;
    const target = event.target;
    if (
      target instanceof HTMLElement &&
      target.closest("input, textarea, select, [contenteditable='true']")
    ) {
      return;
    }
    const key = event.key.toLowerCase();
    if (key === "g") {
      gArmedAt.current = Date.now();
      return;
    }
    if (Date.now() - gArmedAt.current < 1200) {
      gArmedAt.current = 0;
      if (key === "d") {
        event.preventDefault();
        void navigate(appRoutes.design);
        return;
      }
      if (key === "s") {
        event.preventDefault();
        void navigate(appRoutes.stats);
        return;
      }
    }
    if (key === "n") {
      event.preventDefault();
      void navigate(`${appRoutes.design}?new=1`);
    }
  });

  const go = (href: string) => {
    setCommandOpen(false);
    void navigate(href);
  };
  const crumb = crumbFor(location.pathname);
  const workspaceStudio =
    location.pathname === appRoutes.design && new URLSearchParams(location.search).has("workspace");

  return (
    <div className="wb-root" data-workbench="nocturne">
      <LibraryAppShell
        brand={
          <span className="wb-brand-text">
            <span className="wb-brand-name">
              {brand?.name ?? t("workbench.brand", { defaultValue: "React Component Library" })}
            </span>
            <small className="wb-brand-meta">
              <i className={`wb-brand-dot${brand?.assigned ? " is-assigned" : ""}`} aria-hidden />
              {brand?.assigned
                ? t("workbench.brandSource", { defaultValue: "Brand Manager" })
                : t("workbench.brandUnassigned", { defaultValue: "Brand not assigned" })}
            </small>
          </span>
        }
        brandMark={
          <span className={`wb-brand-mark${brand?.assigned ? "" : " is-fallback"}`} aria-hidden>
            {brand?.assigned && brand.mark ? (
              <img src={brand.mark} alt="" />
            ) : (
              <span className="wb-brand-mono">
                {t("workbench.identity.initials", { defaultValue: "RCL" })}
              </span>
            )}
          </span>
        }
        brandHref={appRoutes.start}
        items={items}
        density="sidebar"
        mobileNav="tabs"
        mainMode="fill"
        header={
          <div className="wb-topbar">
            <div className="wb-crumb">
              <Link to={appRoutes.start}>
                {t("workbench.brand", { defaultValue: "React Component Library" })}
              </Link>
              <span aria-hidden>/</span>
              <b>{t(crumb.key, { defaultValue: crumb.fallback })}</b>
            </div>
            <div className="wb-top-actions">
              <button
                type="button"
                className="wb-command"
                onClick={openCommand}
                aria-label={t("workbench.command.open", { defaultValue: "Search the workbench" })}
              >
                <Search aria-hidden size={14} />
                <span>
                  {t("workbench.command.placeholder", {
                    defaultValue: "Search pages, states, assets, commands…",
                  })}
                </span>
                <kbd>{t("workbench.command.shortcut", { defaultValue: "⌘ K" })}</kbd>
              </button>
              <button
                type="button"
                className="wb-iconbtn"
                aria-label={t("workbench.command.notifications", { defaultValue: "Notifications" })}
                onClick={() => go(appRoutes.review)}
              >
                <Bell aria-hidden size={15} />
              </button>
            </div>
          </div>
        }
        utility={
          <div className="wb-shell-utility">
            <Link
              to={appRoutes.create}
              className="wb-btn primary wide wb-shell-utility-new"
              onClick={() => setCommandOpen(false)}
            >
              <Plus aria-hidden size={15} />
              {t("workbench.nav.newAsset", { defaultValue: "New" })}
            </Link>
            <NavLink
              to={appRoutes.settings}
              className="wb-nav-item"
              aria-current={isCurrent(location.pathname, appRoutes.settings) ? "page" : undefined}
              onClick={() => setCommandOpen(false)}
            >
              <SettingsIcon aria-hidden size={17} />
              <span>{t("workbench.nav.settings", { defaultValue: "Settings" })}</span>
            </NavLink>
            <div className="wb-identity">
              <span className="wb-identity-mark" aria-hidden>
                {t("workbench.identity.initials", { defaultValue: "RCL" })}
              </span>
              <span>
                <span className="wb-identity-name">
                  {t("workbench.identity.name", { defaultValue: "Local workbench" })}
                </span>
                <span className="wb-identity-meta">
                  {t("workbench.identity.meta", { defaultValue: "single user · local" })}
                </span>
              </span>
            </div>
          </div>
        }
        renderLink={RouterLink}
        onNavigate={(item) => go(item.href)}
        navigationLabel={t("workbench.nav.label", { defaultValue: "Workbench" })}
        mobileNavigationLabel={t("workbench.nav.mobile", { defaultValue: "Workbench" })}
        sidebarStorageKey="react-component-library.workbench-sidebar-width"
        sidebarResize={{ defaultSize: 236, min: 220, max: 320, adjacentMin: 480 }}
        testId={selectors.app.shell}
        className="wb-app-shell"
        mainClassName={cx("wb-main", "wb-view", workspaceStudio && "is-workspace-studio")}
      >
        <>{children ?? <Outlet />}</>
      </LibraryAppShell>
      <SearchPalette open={commandOpen} onClose={() => setCommandOpen(false)} />
    </div>
  );
}
