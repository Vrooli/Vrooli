import { ReactNode, useEffect, useMemo, useState } from 'react'
import { Link, useLocation } from 'react-router-dom'
import {
  LayoutDashboard,
  FolderTree,
  Puzzle,
  Settings,
  Menu,
  Plus,
  PanelsTopLeft,
} from 'lucide-react'
import { Button, buttonVariants } from './ui/button'
import { cn } from '../lib/utils'

interface LayoutProps {
  children: ReactNode
}

function Layout({ children }: LayoutProps) {
  const location = useLocation()
  const [mobileOpen, setMobileOpen] = useState(false)

  const navItems = useMemo(
    () => [
      { path: '/dashboard', label: 'Dashboard', icon: LayoutDashboard },
      { path: '/graphs', label: 'Graphs', icon: FolderTree },
      { path: '/plugins', label: 'Plugins', icon: Puzzle },
      { path: '/settings', label: 'Settings', icon: Settings },
    ],
    [],
  )

  useEffect(() => {
    setMobileOpen(false)
  }, [location.pathname])

  const isActive = (path: string) =>
    location.pathname === path || location.pathname.startsWith(`${path}/`)

  return (
    <div className="flex min-h-screen bg-background/80">
      <aside className="hidden w-64 flex-col border-r border-border/70 bg-card/70 backdrop-blur-lg lg:flex">
        <div className="flex items-center gap-3 px-6 pb-6 pt-8">
          <div className="flex h-12 w-12 items-center justify-center rounded-full gradient-accent text-primary-foreground shadow-lg shadow-primary/40">
            <PanelsTopLeft className="h-6 w-6" />
          </div>
          <div>
            <p className="text-lg font-semibold leading-tight tracking-tight text-foreground">
              Graph Studio
            </p>
            <p className="text-xs font-medium uppercase text-primary/80">
              Visual intelligence lab
            </p>
          </div>
        </div>

        <nav className="flex-1 space-y-1 px-4">
          {navItems.map(({ path, label, icon: Icon }) => (
            <Link
              key={path}
              to={path}
              className={cn(
                buttonVariants({ variant: isActive(path) ? 'default' : 'ghost', size: 'sm' }),
                'justify-start gap-3 px-3 font-medium',
              )}
            >
              <Icon className="h-4 w-4" />
              {label}
            </Link>
          ))}
        </nav>

        <div className="px-4 pb-8 pt-4">
          <Button className="w-full gap-2 shadow-md shadow-primary/30">
            <Plus className="h-4 w-4" />
            New Graph
          </Button>
        </div>
      </aside>

      <div className="relative flex flex-1 flex-col overflow-hidden">
        <header className="sticky top-0 z-30 border-b border-border/60 bg-background/80 backdrop-blur supports-[backdrop-filter]:bg-background/65">
          <div className="flex h-16 items-center gap-3 px-4 sm:px-6">
            <Button
              type="button"
              variant="outline"
              size="icon"
              className="lg:hidden"
              onClick={() => setMobileOpen((prev) => !prev)}
            >
              <Menu className="h-5 w-5" />
              <span className="sr-only">Toggle navigation</span>
            </Button>
            <div className="flex items-center gap-3 lg:hidden">
              <div className="flex h-10 w-10 items-center justify-center rounded-full gradient-accent text-primary-foreground shadow-md shadow-primary/30">
                <PanelsTopLeft className="h-5 w-5" />
              </div>
              <div>
                <p className="text-base font-semibold leading-tight">Graph Studio</p>
                <p className="text-xs text-muted-foreground">Visual intelligence</p>
              </div>
            </div>
            <div className="ml-auto flex items-center gap-2">
              <Button variant="outline" className="hidden gap-2 lg:inline-flex">
                <Plus className="h-4 w-4" />
                New Graph
              </Button>
            </div>
          </div>
        </header>

        {mobileOpen && (
          <div className="lg:hidden">
            <div
              className="fixed inset-0 z-40 bg-black/40 backdrop-blur-sm"
              onClick={() => setMobileOpen(false)}
            />
            <div className="fixed inset-x-3 top-20 z-50 space-y-2 rounded-xl border border-border/70 bg-background/95 p-4 shadow-xl">
              {navItems.map(({ path, label, icon: Icon }) => (
                <Link
                  key={path}
                  to={path}
                  className={cn(
                    buttonVariants({
                      variant: isActive(path) ? 'default' : 'ghost',
                      size: 'default',
                    }),
                    'w-full justify-start gap-3',
                  )}
                >
                  <Icon className="h-4 w-4" />
                  {label}
                </Link>
              ))}
              <Button className="w-full gap-2">
                <Plus className="h-4 w-4" />
                New Graph
              </Button>
            </div>
          </div>
        )}

        <main className="flex-1 overflow-y-auto px-4 py-8 sm:px-8">
          <div className="mx-auto w-full max-w-6xl space-y-8 pb-16">
            {children}
          </div>
        </main>

        <footer className="border-t border-border/70 bg-background/80">
          <div className="mx-auto flex w-full max-w-6xl flex-col gap-2 px-4 py-6 text-center text-sm text-muted-foreground sm:flex-row sm:items-center sm:justify-between sm:px-8">
            <span>Graph Studio v1.0.0</span>
            <span className="flex items-center justify-center gap-2">
              Crafted on Vrooli
              <span className="h-1 w-1 rounded-full bg-primary/60" />
              Adaptive graph intelligence
            </span>
          </div>
        </footer>
      </div>
    </div>
  )
}

export default Layout
