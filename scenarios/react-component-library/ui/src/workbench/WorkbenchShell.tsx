import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { Link, NavLink, Outlet, useLocation, useNavigate } from "react-router-dom";
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
import "./brand.css";

interface NavItem {
  id: string;
  href: string;
  label: string;
  short: string;
  icon: ReactNode;
  count?: number;
  /** Supplemental hover/tooltip text; the visible label stays the destination. */
  title?: string;
}

function isCurrent(pathname: string, href: string): boolean {
  if (href === "/") return pathname === "/";
  if (href === "/library") {
    return (
      pathname === "/library" ||
      pathname === "/components" ||
      pathname === "/catalog" ||
      pathname.startsWith("/assets/")
    );
  }
  if (href === "/design") return pathname.startsWith("/design") || pathname.startsWith("/sketch/");
  if (href === "/stats") {
    return (
      pathname === "/stats" ||
      pathname === "/coverage" ||
      pathname === "/activity" ||
      pathname === "/capabilities"
    );
  }
  return pathname === href;
}

function crumbFor(pathname: string): { key: string; fallback: string } {
  if (pathname === "/") return { key: "workbench.nav.start", fallback: "Start" };
  if (pathname === "/library") return { key: "workbench.nav.library", fallback: "Library" };
  if (pathname === "/components" || pathname === "/catalog") {
    return { key: "workbench.nav.components", fallback: "Components" };
  }
  if (pathname.startsWith("/assets/")) {
    if (pathname.endsWith("/preview")) return { key: "workbench.nav.preview", fallback: "Preview" };
    return { key: "workbench.nav.component", fallback: "Component" };
  }
  if (pathname.startsWith("/design") || pathname.startsWith("/sketch/")) {
    return { key: "workbench.nav.designShort", fallback: "Design" };
  }
  if (pathname === "/review") return { key: "workbench.nav.review", fallback: "Review" };
  if (pathname === "/stats") return { key: "workbench.nav.stats", fallback: "Stats" };
  if (pathname === "/coverage") return { key: "workbench.nav.coverage", fallback: "Coverage" };
  if (pathname === "/capabilities")
    return { key: "workbench.nav.capabilities", fallback: "Capabilities" };
  if (pathname === "/activity") return { key: "workbench.nav.activity", fallback: "Activity" };
  if (pathname === "/settings") return { key: "workbench.nav.settings", fallback: "Settings" };
  if (pathname === "/new") return { key: "workbench.nav.create", fallback: "New asset" };
  return { key: "workbench.nav.notFound", fallback: "Not found" };
}

export function WorkbenchShell({ children }: { children?: ReactNode }) {
  const { t } = useTranslation();
  const location = useLocation();
  const navigate = useNavigate();
  const [commandOpen, setCommandOpen] = useState(false);
  // The Start rail advertises these shortcuts, so the shell owns them. A
  // sequence is only armed when the person is not typing into a field.
  const gArmedAt = useRef(0);

  const review = useQuery({
    queryKey: ["review", "summary"],
    queryFn: getReviewSummary,
    retry: 0,
  });
  const brand = useBrand().data ?? null;
  const openFindings = review.data?.open ?? 0;

  const items: NavItem[] = [
    {
      id: "start",
      href: appRoutes.start,
      label: t("workbench.nav.start", { defaultValue: "Start" }),
      short: t("workbench.nav.start", { defaultValue: "Start" }),
      icon: <House aria-hidden size={17} />,
    },
    {
      id: "library",
      href: appRoutes.library,
      label: t("workbench.nav.library", { defaultValue: "Library" }),
      short: t("workbench.nav.library", { defaultValue: "Library" }),
      icon: <LayoutGrid aria-hidden size={17} />,
    },
    {
      id: "design",
      href: appRoutes.design,
      label: t("workbench.nav.designShort", { defaultValue: "Design" }),
      short: t("workbench.nav.designShort", { defaultValue: "Design" }),
      icon: <PenTool aria-hidden size={17} />,
    },
    {
      id: "review",
      href: appRoutes.review,
      label: t("workbench.nav.review", { defaultValue: "Review" }),
      short: t("workbench.nav.review", { defaultValue: "Review" }),
      icon: <ClipboardCheck aria-hidden size={17} />,
      count: openFindings || undefined,
    },
    {
      id: "stats",
      href: appRoutes.stats,
      label: t("workbench.nav.stats", { defaultValue: "Stats" }),
      short: t("workbench.nav.statsShort", { defaultValue: "Stats" }),
      icon: <BarChart3 aria-hidden size={17} />,
    },
  ];

  const openCommand = useCallback(() => {
    setCommandOpen(true);
  }, []);

  useEffect(() => {
    applyWorkbenchPreferences(readWorkbenchPreferences());
  }, []);

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
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
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [navigate]);

  const go = (href: string) => {
    setCommandOpen(false);
    void navigate(href);
  };

  const crumb = crumbFor(location.pathname);

  return (
    <div className="wb-root" data-workbench="nocturne" data-testid="workbench-shell">
      <div className="wb-app">
        <aside
          className="wb-rail"
          aria-label={t("workbench.nav.rail", { defaultValue: "Library navigation" })}
        >
          <Link
            to={appRoutes.start}
            className="wb-brand"
            title={t("workbench.nav.home", { defaultValue: "Home" })}
            aria-label={brand?.name ?? t("workbench.brand", { defaultValue: "React Component Library" })}
          >
            <span className={`wb-brand-mark${brand?.assigned ? "" : " is-fallback"}`} aria-hidden>
              {brand?.assigned && brand.mark ? (
                <img src={brand.mark} alt="" />
              ) : (
                <span className="wb-brand-mono">{t("workbench.identity.initials", { defaultValue: "RCL" })}</span>
              )}
            </span>
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
          </Link>

          <nav
            className="wb-nav"
            aria-label={t("workbench.nav.label", { defaultValue: "Workbench" })}
          >
            {items.slice(0, 3).map((item) => (
              <NavItemLink
                key={item.id}
                item={item}
                current={isCurrent(location.pathname, item.href)}
              />
            ))}
            <div className="wb-nav-label">
              {t("workbench.nav.operations", { defaultValue: "Operations" })}
            </div>
            {items.slice(3).map((item) => (
              <NavItemLink
                key={item.id}
                item={item}
                current={isCurrent(location.pathname, item.href)}
              />
            ))}
          </nav>

          <div className="wb-rail-foot">
            <button
              type="button"
              className="wb-btn primary wide"
              onClick={() => go(appRoutes.create)}
            >
              <Plus aria-hidden size={15} />
              {t("workbench.nav.newAsset", { defaultValue: "New asset" })}
            </button>
            <NavItemLink
              item={{
                id: "settings",
                href: appRoutes.settings,
                label: t("workbench.nav.settings", { defaultValue: "Settings" }),
                short: t("workbench.nav.settings", { defaultValue: "Settings" }),
                icon: <SettingsIcon aria-hidden size={17} />,
              }}
              current={isCurrent(location.pathname, appRoutes.settings)}
            />
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
        </aside>

        <div className="wb-main">
          <header className="wb-topbar">
            <div className="wb-crumb">
              <Link to={appRoutes.start}>{t("workbench.brand", { defaultValue: "React Component Library" })}</Link>
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
          </header>

          <main className="wb-view" id="workbench-main">
            {children ?? <Outlet />}
          </main>
        </div>
      </div>

      <nav
        className="wb-mobile-nav"
        aria-label={t("workbench.nav.mobile", { defaultValue: "Workbench" })}
      >
        {items.map((item) => (
          <NavLink
            key={item.id}
            to={item.href}
            aria-current={isCurrent(location.pathname, item.href) ? "page" : undefined}
          >
            {item.icon}
            <span>{item.short}</span>
          </NavLink>
        ))}
      </nav>

      <SearchPalette open={commandOpen} onClose={() => setCommandOpen(false)} />
    </div>
  );
}

function NavItemLink({ item, current }: { item: NavItem; current: boolean }) {
  return (
    <Link
      to={item.href}
      title={item.title}
      className={cx("wb-nav-item", current && "is-current")}
      aria-current={current ? "page" : undefined}
    >
      {item.icon}
      <span>{item.label}</span>
      {typeof item.count === "number" ? <span className="wb-nav-count">{item.count}</span> : null}
    </Link>
  );
}
