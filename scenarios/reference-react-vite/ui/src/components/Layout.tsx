import { NavLink, Outlet, useNavigate } from "react-router-dom";
import {
  LayoutDashboard,
  ListTodo,
  FolderKanban,
  Settings as SettingsIcon,
  ShieldCheck,
  Sparkles,
  Activity,
  Menu as MenuIcon,
  X as XIcon,
  ChevronDown,
} from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { useEffect, useRef, useState, type ReactNode } from "react";
import { fetchHealth } from "../lib/api";
import { useViewport } from "../hooks/useViewport";
import { useAppContext } from "../contexts/AppContext";
import { ROUTES } from "../routes.generated";

interface NavSpec {
  to: string;
  label: string;
  icon: typeof LayoutDashboard;
  topId: string;
  bottomId: string;
  menuId: string;
  show: (ctx: { role: "viewer" | "editor" | "admin"; featureBeta: boolean }) => boolean;
  hideInBottom?: boolean;
}

const NAV: NavSpec[] = [
  {
    to: ROUTES.dashboard,
    label: "Dashboard",
    icon: LayoutDashboard,
    topId: "top-nav-dashboard",
    bottomId: "bottom-nav-dashboard",
    menuId: "menu-dashboard",
    show: () => true,
  },
  {
    to: ROUTES.tasksIndex,
    label: "Tasks",
    icon: ListTodo,
    topId: "top-nav-tasks",
    bottomId: "bottom-nav-tasks",
    menuId: "menu-tasks",
    show: () => true,
  },
  {
    to: ROUTES.projectsIndex,
    label: "Projects",
    icon: FolderKanban,
    topId: "top-nav-projects",
    bottomId: "bottom-nav-projects",
    menuId: "menu-projects",
    show: () => true,
  },
  {
    to: ROUTES.settingsIndex,
    label: "Settings",
    icon: SettingsIcon,
    topId: "top-nav-settings",
    bottomId: "bottom-nav-settings",
    menuId: "menu-settings",
    show: () => true,
  },
  {
    to: ROUTES.adminUsers,
    label: "Admin",
    icon: ShieldCheck,
    topId: "top-nav-admin",
    bottomId: "bottom-nav-admin",
    menuId: "menu-admin",
    show: ({ role }) => role === "admin",
    hideInBottom: true,
  },
];

function HealthIndicator() {
  const { data, isLoading, error } = useQuery({
    queryKey: ["health"],
    queryFn: fetchHealth,
    refetchInterval: 30000,
  });
  const status = isLoading ? "checking" : error ? "offline" : data?.status ?? "unknown";
  const statusColor =
    {
      checking: "text-yellow-400",
      offline: "text-red-400",
      healthy: "text-green-400",
      unknown: "text-slate-400",
    }[status] ?? "text-slate-400";
  return (
    <div
      data-testid="health-indicator"
      className="flex items-center gap-2 text-xs"
      title={error ? "API is not reachable" : `API Status: ${status}`}
    >
      <Activity className={`h-3.5 w-3.5 ${statusColor}`} />
      <span className="capitalize text-slate-400">{status}</span>
    </div>
  );
}

function TopNavBar({ role, featureBeta }: { role: "viewer" | "editor" | "admin"; featureBeta: boolean }) {
  return (
    <nav data-testid="top-nav-bar" className="flex items-center gap-1">
      {NAV.filter((n) => n.show({ role, featureBeta })).map(({ to, icon: Icon, label, topId }) => (
        <NavLink
          key={to}
          to={to}
          data-testid={topId}
          className={({ isActive }) =>
            `flex items-center gap-2 rounded-lg px-3 py-2 text-sm transition-colors ${
              isActive ? "bg-white/10 text-white" : "text-slate-400 hover:bg-white/5 hover:text-slate-200"
            }`
          }
        >
          <Icon className="h-4 w-4" />
          {label}
        </NavLink>
      ))}
    </nav>
  );
}

function BottomNav({ role, featureBeta }: { role: "viewer" | "editor" | "admin"; featureBeta: boolean }) {
  return (
    <nav
      data-testid="bottom-nav"
      className="fixed inset-x-0 bottom-0 flex items-stretch border-t border-white/10 bg-slate-900/95 backdrop-blur"
    >
      {NAV.filter((n) => !n.hideInBottom && n.show({ role, featureBeta })).map(({ to, icon: Icon, label, bottomId }) => (
        <NavLink
          key={to}
          to={to}
          data-testid={bottomId}
          className={({ isActive }) =>
            `flex flex-1 flex-col items-center gap-1 px-2 py-2 text-[10px] ${
              isActive ? "text-white" : "text-slate-400"
            }`
          }
        >
          <Icon className="h-5 w-5" />
          {label}
        </NavLink>
      ))}
    </nav>
  );
}

function HamburgerMenu({
  role,
  featureBeta,
}: {
  role: "viewer" | "editor" | "admin";
  featureBeta: boolean;
}) {
  const [open, setOpen] = useState(false);
  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpen(false);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open]);
  return (
    <>
      <button
        type="button"
        data-testid="hamburger-button"
        aria-label="Menu"
        aria-expanded={open}
        onClick={() => setOpen((v) => !v)}
        className="rounded-lg p-2 text-slate-300 hover:bg-white/5"
      >
        {open ? <XIcon className="h-5 w-5" /> : <MenuIcon className="h-5 w-5" />}
      </button>
      {open ? (
        <>
          <div
            data-testid="hamburger-backdrop"
            className="fixed inset-0 z-40 bg-black/50"
            onClick={() => setOpen(false)}
          />
          <div
            data-testid="hamburger-menu"
            role="dialog"
            aria-modal="true"
            className="fixed inset-y-0 left-0 z-50 flex w-64 flex-col gap-1 border-r border-white/10 bg-slate-900 p-3"
          >
            {NAV.filter((n) => n.show({ role, featureBeta })).map(({ to, icon: Icon, label, menuId }) => (
              <NavLink
                key={to}
                to={to}
                data-testid={menuId}
                onClick={() => setOpen(false)}
                className={({ isActive }) =>
                  `flex items-center gap-2 rounded-lg px-3 py-2 text-sm ${
                    isActive ? "bg-white/10 text-white" : "text-slate-300 hover:bg-white/5"
                  }`
                }
              >
                <Icon className="h-4 w-4" />
                {label}
              </NavLink>
            ))}
            {featureBeta ? (
              <NavLink
                to={ROUTES.betaDashboard}
                data-testid="menu-beta"
                onClick={() => setOpen(false)}
                className="flex items-center gap-2 rounded-lg px-3 py-2 text-sm text-slate-300 hover:bg-white/5"
              >
                <Sparkles className="h-4 w-4" />
                Beta features
              </NavLink>
            ) : null}
          </div>
        </>
      ) : null}
    </>
  );
}

function UserMenu({ featureBeta }: { featureBeta: boolean }) {
  const { setAuth } = useAppContext();
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!open) return;
    const onClickOutside = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpen(false);
    };
    document.addEventListener("mousedown", onClickOutside);
    window.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onClickOutside);
      window.removeEventListener("keydown", onKey);
    };
  }, [open]);
  return (
    <div className="relative" ref={ref}>
      <button
        type="button"
        data-testid="user-menu-trigger"
        aria-haspopup="menu"
        aria-expanded={open}
        onClick={() => setOpen((v) => !v)}
        className="flex items-center gap-1 rounded-lg px-2 py-1 text-xs text-slate-300 hover:bg-white/5"
      >
        Account
        <ChevronDown className="h-3 w-3" />
      </button>
      {open ? (
        <div
          data-testid="user-menu"
          role="menu"
          className="absolute right-0 mt-1 flex w-48 flex-col rounded-lg border border-white/10 bg-slate-900 p-1 shadow-lg"
        >
          {featureBeta ? (
            <NavLink
              to={ROUTES.betaDashboard}
              data-testid="user-menu-beta"
              onClick={() => setOpen(false)}
              className="rounded px-2 py-1 text-sm text-slate-200 hover:bg-white/5"
            >
              Beta features
            </NavLink>
          ) : null}
          <button
            type="button"
            data-testid="logout-item"
            onClick={() => {
              setOpen(false);
              setAuth("logged_out");
              navigate(ROUTES.login, { replace: true });
            }}
            className="rounded px-2 py-1 text-left text-sm text-slate-200 hover:bg-white/5"
          >
            Log out
          </button>
        </div>
      ) : null}
    </div>
  );
}

function AuthFooter() {
  return (
    <footer
      data-testid="auth-footer"
      className="border-t border-white/10 bg-slate-900/30 px-4 py-2 text-center text-xs text-slate-500"
    >
      Reference React Vite template · illustrative auth only
    </footer>
  );
}

interface LayoutProps {
  children?: ReactNode;
}

export function Layout({ children }: LayoutProps = {}) {
  const viewport = useViewport();
  const { auth, role, featureBeta } = useAppContext();
  const showTopBar = viewport === "desktop" && auth === "logged_in";
  const showBottomNav = viewport === "mobile" && auth === "logged_in";
  const showHamburger = (viewport === "mobile" || viewport === "tablet") && auth === "logged_in";

  return (
    <div
      data-testid="app-shell"
      data-viewport={viewport}
      data-auth={auth}
      className="flex h-full flex-col overflow-hidden bg-slate-950 text-slate-50"
    >
      <header
        data-testid="app-header"
        className="flex-shrink-0 border-b border-white/10 bg-slate-900/50 px-4 py-3"
      >
        <div className="flex items-center justify-between gap-3">
          <div className="flex items-center gap-3">
            {showHamburger ? <HamburgerMenu role={role} featureBeta={featureBeta} /> : null}
            <h1 className="text-lg font-semibold">Task Manager</h1>
            {showTopBar ? <TopNavBar role={role} featureBeta={featureBeta} /> : null}
          </div>
          <div className="flex items-center gap-3">
            <HealthIndicator />
            {auth === "logged_in" ? <UserMenu featureBeta={featureBeta} /> : null}
          </div>
        </div>
      </header>

      <main className={`flex-1 overflow-auto p-6 ${showBottomNav ? "pb-20" : ""}`} data-testid="main-content">
        {children ?? <Outlet />}
      </main>

      {showBottomNav ? <BottomNav role={role} featureBeta={featureBeta} /> : null}
      {auth === "logged_out" ? <AuthFooter /> : null}
    </div>
  );
}
