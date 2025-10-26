import { Outlet, NavLink } from 'react-router-dom'
import { useState } from 'react'
import { LayoutDashboard, Layers, BarChart3, FileText, Menu, X } from 'lucide-react'

const Layout = () => {
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false)

  const navItems = [
    { to: '/dashboard', icon: LayoutDashboard, label: 'Dashboard' },
    { to: '/builder', icon: Layers, label: 'Builder' },
    { to: '/templates', icon: FileText, label: 'Templates' },
    { to: '/analytics', icon: BarChart3, label: 'Analytics' },
  ]

  return (
    <div className="min-h-screen bg-gray-50">
      <header className="flex items-center justify-between border-b border-gray-200 bg-white px-4 py-4 md:hidden">
        <h1 className="text-xl font-semibold text-primary-600">Funnel Builder</h1>
        <button
          type="button"
          onClick={() => setIsMobileMenuOpen((open) => !open)}
          className="inline-flex items-center justify-center rounded-md border border-gray-200 p-2 text-gray-700 shadow-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
          aria-label={isMobileMenuOpen ? 'Close navigation' : 'Open navigation'}
        >
          {isMobileMenuOpen ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
        </button>
      </header>

      {isMobileMenuOpen && (
        <div className="md:hidden">
          <div
            className="fixed inset-0 z-40 bg-black/40"
            onClick={() => setIsMobileMenuOpen(false)}
          />
          <aside className="fixed inset-y-0 left-0 z-50 w-72 max-w-full bg-white shadow-xl">
            <div className="border-b border-gray-200 p-6">
              <h1 className="text-2xl font-bold text-primary-600">Funnel Builder</h1>
            </div>
            <nav className="px-4 pb-6 pt-4">
              {navItems.map((item) => (
                <NavLink
                  key={item.to}
                  to={item.to}
                  onClick={() => setIsMobileMenuOpen(false)}
                  className={({ isActive }) =>
                    `flex items-center gap-3 rounded-lg px-4 py-3 text-base transition-colors ${
                      isActive
                        ? 'bg-primary-50 text-primary-600'
                        : 'text-gray-600 hover:bg-gray-50'
                    }`
                  }
                >
                  <item.icon className="h-5 w-5" />
                  <span className="font-medium">{item.label}</span>
                </NavLink>
              ))}
            </nav>
          </aside>
        </div>
      )}

      <div className="md:flex md:min-h-screen">
        <aside className="hidden w-64 shrink-0 border-r border-gray-200 bg-white md:block">
          <div className="p-6">
            <h1 className="text-2xl font-bold text-primary-600">Funnel Builder</h1>
          </div>
          <nav className="px-4 pb-4">
            {navItems.map((item) => (
              <NavLink
                key={item.to}
                to={item.to}
                className={({ isActive }) =>
                  `flex items-center gap-3 px-4 py-3 text-sm rounded-lg mb-1 transition-colors ${
                    isActive
                      ? 'bg-primary-50 text-primary-600'
                      : 'text-gray-600 hover:bg-gray-50'
                  }`
                }
              >
                <item.icon className="w-5 h-5" />
                <span className="font-medium">{item.label}</span>
              </NavLink>
            ))}
          </nav>
        </aside>
        <main className="flex-1 overflow-x-hidden md:overflow-auto">
          <Outlet />
        </main>
      </div>
    </div>
  )
}

export default Layout
