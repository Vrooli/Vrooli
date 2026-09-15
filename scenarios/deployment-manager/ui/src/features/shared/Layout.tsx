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
    { id: "dashboard", href: "/", label: "Dashboard", icon: <LayoutGrid className="h-4 w-4" />, current: location.pathname === "/" },
    { id: "profiles", href: "/profiles", label: "Profiles", icon: <Package className="h-4 w-4" />, current: location.pathname.startsWith("/profiles") },
    { id: "analyze", href: "/analyze", label: "Analyze", icon: <FolderTree className="h-4 w-4" />, current: location.pathname.startsWith("/analyze") },
    { id: "deployments", href: "/deployments", label: "Deployments", icon: <Rocket className="h-4 w-4" />, current: location.pathname.startsWith("/deployments") },
    { id: "approvals", href: "/approvals", label: "Approvals", icon: <Shield className="h-4 w-4" />, current: location.pathname.startsWith("/approvals") },
    { id: "evidence", href: "/evidence", label: "Evidence", icon: <FileCheck className="h-4 w-4" />, current: location.pathname.startsWith("/evidence") },
    { id: "releases", href: "/releases", label: "Releases", icon: <Package className="h-4 w-4" />, current: location.pathname.startsWith("/releases") },
    { id: "telemetry", href: "/telemetry", label: "Telemetry", icon: <Monitor className="h-4 w-4" />, current: location.pathname.startsWith("/telemetry") },
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
      mobileNav="drawer"
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
