import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";
import { Link, NavLink, Outlet, useLocation, useNavigate } from "react-router-dom";
import {
  Activity,
  ArrowRight,
  BarChart3,
  Bell,
  ClipboardCheck,
  Command,
  House,
  LayoutGrid,
  PenTool,
  Plus,
  Search,
  Settings as SettingsIcon,
  Sparkles,
} from "lucide-react";

import { useTranslation } from "../i18n";
import { appRoutes } from "../routes";
import { useCatalog, useDesignScenarios, useCoverage, findingsFromCoverage } from "./data";
import { cx } from "./cx";
import { applyWorkbenchPreferences, readWorkbenchPreferences } from "./preferences";

interface NavItem {
  id: string;
  href: string;
  label: string;
  short: string;
  icon: ReactNode;
  count?: number;
}

function isCurrent(pathname: string, href: string): boolean {
  if (href === "/") return pathname === "/";
  if (href === "/components") {
    return pathname === "/components" || pathname.startsWith("/assets/") || pathname === "/catalog";
  }
  if (href === "/design") return pathname.startsWith("/design") || pathname.startsWith("/sketch/");
  return pathname === href;
}

function crumbFor(pathname: string): { key: string; fallback: string } {
  if (pathname === "/") return { key: "workbench.nav.home", fallback: "Home" };
  if (pathname === "/components" || pathname === "/catalog") {
    return { key: "workbench.nav.components", fallback: "Components" };
  }
  if (pathname.startsWith("/assets/")) {
    if (pathname.endsWith("/preview")) return { key: "workbench.nav.preview", fallback: "Preview" };
    return { key: "workbench.nav.component", fallback: "Component" };
  }
  if (pathname.startsWith("/design") || pathname.startsWith("/sketch/")) {
    return { key: "workbench.nav.design", fallback: "Design studio" };
  }
  if (pathname === "/review") return { key: "workbench.nav.review", fallback: "Review" };
  if (pathname === "/coverage") return { key: "workbench.nav.coverage", fallback: "Coverage" };
  if (pathname === "/capabilities") return { key: "workbench.nav.capabilities", fallback: "Capabilities" };
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
  const [query, setQuery] = useState("");
  const commandInput = useRef<HTMLInputElement>(null);

  const catalog = useCatalog();
  const coverage = useCoverage();
  const scenarios = useDesignScenarios();

  const findings = useMemo(() => findingsFromCoverage(coverage.data), [coverage.data]);
  const openFindings = findings.length;

  const items: NavItem[] = [
    {
      id: "home",
      href: appRoutes.home,
      label: t("workbench.nav.home", { defaultValue: "Home" }),
      short: t("workbench.nav.home", { defaultValue: "Home" }),
      icon: <House aria-hidden size={17} />,
    },
    {
      id: "components",
      href: appRoutes.components,
      label: t("workbench.nav.components", { defaultValue: "Components" }),
      short: t("workbench.nav.componentsShort", { defaultValue: "Browse" }),
      icon: <LayoutGrid aria-hidden size={17} />,
      count: catalog.data?.length,
    },
    {
      id: "design",
      href: appRoutes.design,
      label: t("workbench.nav.design", { defaultValue: "Design studio" }),
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
      id: "coverage",
      href: appRoutes.coverage,
      label: t("workbench.nav.coverage", { defaultValue: "Coverage" }),
      short: t("workbench.nav.coverage", { defaultValue: "Coverage" }),
      icon: <BarChart3 aria-hidden size={17} />,
    },
    {
      id: "capabilities",
      href: appRoutes.capabilities,
      label: t("workbench.nav.capabilities", { defaultValue: "Capabilities" }),
      short: t("workbench.nav.capabilitiesShort", { defaultValue: "Capabilities" }),
      icon: <Sparkles aria-hidden size={17} />,
    },
    {
      id: "activity",
      href: appRoutes.activity,
      label: t("workbench.nav.activity", { defaultValue: "Activity" }),
      short: t("workbench.nav.activity", { defaultValue: "Activity" }),
      icon: <Activity aria-hidden size={17} />,
    },
  ];

  const openCommand = useCallback(() => {
    setQuery("");
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
      }
      if (event.key === "Escape") setCommandOpen(false);
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, []);

  useEffect(() => {
    if (commandOpen) commandInput.current?.focus();
  }, [commandOpen]);

  const results = useMemo(() => {
    const term = query.trim().toLowerCase();
    const assets = (catalog.data ?? [])
      .filter((asset) => {
        if (!term) return true;
        return (
          asset.displayName.toLowerCase().includes(term) ||
          asset.libraryId.toLowerCase().includes(term) ||
          asset.catalogId.toLowerCase().includes(term) ||
          asset.description.toLowerCase().includes(term)
        );
      })
      .slice(0, 8);
    const pages = (scenarios.data ?? [])
      .filter((entry) => (entry.pageCount ?? 0) > 0 && (!term || entry.scenario.includes(term)))
      .slice(0, 5);
    return { assets, pages };
  }, [catalog.data, scenarios.data, query]);

  const go = (href: string) => {
    setCommandOpen(false);
    void navigate(href);
  };

  const crumb = crumbFor(location.pathname);

  return (
    <div className="wb-root" data-workbench="nocturne" data-testid="workbench-shell">
      <div className="wb-app">
        <aside className="wb-rail">
          <Link to={appRoutes.home} className="wb-brand">
            <span className="wb-brand-mark" aria-hidden>
              <LayoutGrid size={16} />
            </span>
            <span>
              {t("workbench.brand", { defaultValue: "Library" })}
              <small>{t("workbench.brandMeta", { defaultValue: "RCL · Nocturne" })}</small>
            </span>
          </Link>

          <nav className="wb-nav" aria-label={t("workbench.nav.label", { defaultValue: "Workbench" })}>
            {items.slice(0, 3).map((item) => (
              <NavItemLink key={item.id} item={item} current={isCurrent(location.pathname, item.href)} />
            ))}
            <div className="wb-nav-label">
              {t("workbench.nav.operations", { defaultValue: "Operations" })}
            </div>
            {items.slice(3).map((item) => (
              <NavItemLink key={item.id} item={item} current={isCurrent(location.pathname, item.href)} />
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
              <Link to={appRoutes.home}>{t("workbench.brand", { defaultValue: "Library" })}</Link>
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
                <span>{t("workbench.command.placeholder", { defaultValue: "Search assets, props, evidence…" })}</span>
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

      <nav className="wb-mobile-nav" aria-label={t("workbench.nav.mobile", { defaultValue: "Workbench" })}>
        {items.slice(0, 5).map((item) => (
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

      {commandOpen ? (
        <div
          className="wb-command-layer"
          role="dialog"
          aria-modal="true"
          aria-label={t("workbench.command.label", { defaultValue: "Workbench search" })}
        >
          <div className="wb-command-panel">
            <div className="wb-search" style={{ margin: 12 }}>
              <Search aria-hidden size={15} style={{ color: "var(--wb-muted-2)" }} />
              <input
                ref={commandInput}
                value={query}
                onChange={(event) => setQuery(event.target.value)}
                aria-label={t("workbench.command.input", { defaultValue: "Search assets and pages" })}
                placeholder={t("workbench.command.placeholder", {
                  defaultValue: "Search assets, props, evidence…",
                })}
              />
              <kbd className="wb-kbd">{t("workbench.command.escape", { defaultValue: "esc" })}</kbd>
            </div>
            <div className="wb-command-results">
              <div className="wb-command-group">
                <span className="wb-caption">{t("workbench.command.assets", { defaultValue: "Assets" })}</span>
                {results.assets.length === 0 ? (
                  <p className="wb-muted wb-small" style={{ padding: "6px 10px" }}>
                    {t("workbench.command.noAssets", { defaultValue: "No matching assets." })}
                  </p>
                ) : (
                  results.assets.map((asset) => (
                    <button
                      key={asset.id}
                      type="button"
                      className="wb-command-row"
                      onClick={() => go(`/assets/${encodeURIComponent(asset.id)}`)}
                    >
                      <span className="wb-command-icon" aria-hidden>
                        <LayoutGrid size={14} />
                      </span>
                      <span style={{ minWidth: 0, flex: 1 }}>
                        <b>{asset.displayName}</b>
                        <small>{asset.catalogId || asset.libraryId}</small>
                      </span>
                      <span className="wb-tag">{asset.latestVersion || asset.version || "draft"}</span>
                    </button>
                  ))
                )}
              </div>
              <div className="wb-command-group">
                <span className="wb-caption">
                  {t("workbench.command.pages", { defaultValue: "Design pages" })}
                </span>
                {results.pages.length === 0 ? (
                  <p className="wb-muted wb-small" style={{ padding: "6px 10px" }}>
                    {t("workbench.command.noPages", { defaultValue: "No matching scenarios." })}
                  </p>
                ) : (
                  results.pages.map((entry) => (
                    <button
                      key={entry.scenario}
                      type="button"
                      className="wb-command-row"
                      onClick={() => go(`/design/${encodeURIComponent(entry.scenario)}`)}
                    >
                      <span className="wb-command-icon" aria-hidden>
                        <PenTool size={14} />
                      </span>
                      <span style={{ minWidth: 0, flex: 1 }}>
                        <b>{entry.scenario}</b>
                        <small>
                          {t("workbench.command.pageCount", {
                            defaultValue: "{{count}} declared pages",
                            count: entry.pageCount ?? 0,
                          })}
                        </small>
                      </span>
                      <ArrowRight aria-hidden size={13} />
                    </button>
                  ))
                )}
              </div>
            </div>
            <div className="wb-command-foot">
              <span>
                <Command aria-hidden size={12} />
                {t("workbench.command.foot", { defaultValue: "Enter to open · Esc to close" })}
              </span>
              <button type="button" className="wb-btn quiet sm" onClick={() => setCommandOpen(false)}>
                {t("workbench.command.close", { defaultValue: "Close" })}
              </button>
            </div>
          </div>
        </div>
      ) : null}
    </div>
  );
}

function NavItemLink({ item, current }: { item: NavItem; current: boolean }) {
  return (
    <Link
      to={item.href}
      className={cx("wb-nav-item", current && "is-current")}
      aria-current={current ? "page" : undefined}
    >
      {item.icon}
      <span>{item.label}</span>
      {typeof item.count === "number" ? <span className="wb-nav-count">{item.count}</span> : null}
    </Link>
  );
}
