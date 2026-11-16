import {
  AppShell as LibraryAppShell,
  type AppShellLinkProps,
  type AppShellNavItem,
} from "@vrooli/react-component-library/AppShell/2";
import { useLocation, NavLink } from "react-router-dom";
import {
  LayoutGrid,
  FolderTree,
  Monitor,
  Rocket,
  Settings,
  Package,
  Shield,
  FileCheck,
} from "lucide-react";

interface LayoutProps {
  children: React.ReactNode;
}

export function Layout({ children }: LayoutProps) {
  const location = useLocation();

  const navItems: AppShellNavItem[] = [
    { id: "dashboard", href: "/", label: "Dashboard", shortLabel: "Home", icon: <LayoutGrid className="h-4 w-4" />, current: location.pathname === "/", mobile: true },
    { id: "profiles", href: "/profiles", label: "Profiles", icon: <Package className="h-4 w-4" />, current: location.pathname.startsWith("/profiles"), mobile: false },
    { id: "analyze", href: "/analyze", label: "Analyze", icon: <FolderTree className="h-4 w-4" />, current: location.pathname.startsWith("/analyze"), mobile: true },
    { id: "deployments", href: "/deployments", label: "Deployments", shortLabel: "Deploy", icon: <Rocket className="h-4 w-4" />, current: location.pathname.startsWith("/deployments"), mobile: true },
    { id: "approvals", href: "/approvals", label: "Approvals", icon: <Shield className="h-4 w-4" />, current: location.pathname.startsWith("/approvals"), mobile: false },
    { id: "evidence", href: "/evidence", label: "Evidence", icon: <FileCheck className="h-4 w-4" />, current: location.pathname.startsWith("/evidence"), mobile: false },
    { id: "releases", href: "/releases", label: "Releases", icon: <Package className="h-4 w-4" />, current: location.pathname.startsWith("/releases"), mobile: true },
    { id: "telemetry", href: "/telemetry", label: "Telemetry", icon: <Monitor className="h-4 w-4" />, current: location.pathname.startsWith("/telemetry"), mobile: false },
  ];

  const renderLink = (item: AppShellNavItem, { href: _href, children, ...props }: AppShellLinkProps) => (
    <NavLink to={item.href} end={item.href === "/"} {...props}>{children}</NavLink>
  );

  return (
    <LibraryAppShell
      brand="Deployment Manager"
      brandMark={<Settings aria-hidden className="h-4 w-4" />}
      brandHref="/"
      items={navItems}
      renderLink={renderLink}
      density="sidebar"
      mobileNav="menu-tabs"
      mainMode="scroll"
      sidebarStorageKey="deployment-manager.sidebar-width"
      navigationLabel="Primary navigation"
      mobileNavigationLabel="Primary navigation"
      skipLabel="Skip to main content"
      menuLabel="Open navigation"
      closeLabel="Close navigation"
      testId="deployment-manager-app-shell"
    >
      {children}
    </LibraryAppShell>
  );
}
