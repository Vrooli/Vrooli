import { Activity, Home, Settings } from "lucide-react";
import { NavLink, useLocation } from "react-router-dom";
import { AppShell as LibraryAppShell } from "@vrooli/react-component-library/AppShell/2";

interface AppShellProps {
  children: React.ReactNode;
}

const navigation = [
  { id: "dashboard", label: "Dashboard", href: "/", icon: Home },
  { id: "campaigns", label: "Campaigns", href: "/campaigns", icon: Activity },
  { id: "settings", label: "Settings", href: "/settings", icon: Settings },
] as const;

export function AppShell({ children }: AppShellProps) {
  const { pathname } = useLocation();

  return (
    <LibraryAppShell
      density="sidebar"
      mobileNav="drawer"
      mainMode="scroll"
      brand={<span>Tidiness Manager</span>}
      brandHref="/"
      items={navigation.map(({ id, label, href, icon: Icon }) => ({
        id,
        label,
        href,
        icon: <Icon size={16} aria-hidden />,
        current: id === "dashboard"
          ? pathname === "/" || pathname === "/dashboard"
          : pathname.startsWith(href),
      }))}
      renderLink={(item, props) => (
        <NavLink
          to={props.href}
          end={item.id === "dashboard"}
          aria-current={props["aria-current"]}
          aria-disabled={props["aria-disabled"]}
          data-testid={props["data-testid"]}
        >
          {props.children}
        </NavLink>
      )}
      menuLabel="Toggle navigation menu"
      closeLabel="Close navigation menu"
      navigationLabel="Application navigation"
      mobileNavigationLabel="Application navigation"
      skipLabel="Skip to content"
      sidebarStorageKey="tidiness-manager.sidebar-width"
      testId="tidiness-manager-app-shell"
      className="bg-slate-950 text-slate-50"
    >
      <div>{children}</div>
      <footer className="mt-auto border-t border-white/10 bg-black/20 px-4 py-4 text-center text-xs text-slate-400 sm:flex sm:items-center sm:justify-between sm:text-left">
        <p>Continuous tidiness prevents emergencies</p>
        <p className="hidden sm:block">Comprehensive coverage, zero redundancy</p>
      </footer>
    </LibraryAppShell>
  );
}
