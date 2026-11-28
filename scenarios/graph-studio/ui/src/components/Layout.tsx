import { ReactNode } from 'react'
import {
  AppShell as LibraryAppShell,
  type AppShellLinkProps,
  type AppShellNavItem,
} from '@vrooli/react-component-library/AppShell/2'
import { LayoutDashboard, FolderTree, Puzzle, Settings, PanelsTopLeft, Plus } from 'lucide-react'
import { NavLink, useLocation } from 'react-router-dom'
import { Button } from './ui/button'

interface LayoutProps { children: ReactNode }

function Layout({ children }: LayoutProps) {
  const { pathname } = useLocation()
  const items: AppShellNavItem[] = [
    { id: 'dashboard', href: '/dashboard', label: 'Dashboard', icon: <LayoutDashboard className="h-4 w-4" />, current: pathname === '/dashboard' },
    { id: 'graphs', href: '/graphs', label: 'Graphs', icon: <FolderTree className="h-4 w-4" />, current: pathname.startsWith('/graphs') },
    { id: 'plugins', href: '/plugins', label: 'Plugins', icon: <Puzzle className="h-4 w-4" />, current: pathname.startsWith('/plugins') },
    { id: 'settings', href: '/settings', label: 'Settings', icon: <Settings className="h-4 w-4" />, current: pathname.startsWith('/settings') },
  ]
  const renderLink = (item: AppShellNavItem, { href: _href, children: linkChildren, ...props }: AppShellLinkProps) => (
    <NavLink to={item.href} end={item.id === 'dashboard'} {...props}>{linkChildren}</NavLink>
  )
  return (
    <LibraryAppShell
      brand="Graph Studio"
      brandMark={<PanelsTopLeft aria-hidden className="h-4 w-4" />}
      brandHref="/dashboard"
      items={items}
      renderLink={renderLink}
      density="sidebar"
      mobileNav="drawer"
      mainMode="scroll"
      header={<Button className="gap-2"><Plus className="h-4 w-4" />New Graph</Button>}
      sidebarStorageKey="graph-studio.sidebar-width"
      navigationLabel="Primary navigation"
      mobileNavigationLabel="Primary navigation"
      skipLabel="Skip to main content"
      menuLabel="Open navigation"
      closeLabel="Close navigation"
      testId="graph-studio-app-shell"
    >
      {children}
    </LibraryAppShell>
  )
}

export default Layout
