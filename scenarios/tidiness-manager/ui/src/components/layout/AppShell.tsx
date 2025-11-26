import { Link, useLocation, useNavigate } from "react-router-dom";
import { Home, Settings, Activity, Menu, X } from "lucide-react";
import { cn } from "../../lib/utils";
import { useState } from "react";
import { KeyboardShortcuts } from "../ui/keyboard-shortcuts";

interface AppShellProps {
  children: React.ReactNode;
}

const navigation = [
  { name: "Dashboard", href: "/", icon: Home },
  { name: "Campaigns", href: "/campaigns", icon: Activity },
  { name: "Settings", href: "/settings", icon: Settings },
];

export function AppShell({ children }: AppShellProps) {
  const location = useLocation();
  const navigate = useNavigate();
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  return (
    <div className="min-h-screen bg-slate-950 text-slate-50">
      {/* Skip to content link for keyboard navigation */}
      <a href="#main-content" className="skip-to-content">
        Skip to content
      </a>

      {/* Header */}
      <header className="border-b border-white/10 bg-black/20 backdrop-blur-sm sticky top-0 z-50">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="flex h-16 items-center justify-between">
            <div className="flex items-center gap-4 md:gap-8">
              <Link to="/" className="flex items-center gap-2" onClick={() => setMobileMenuOpen(false)}>
                <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-gradient-to-br from-blue-500 to-purple-600">
                  <span className="text-sm font-bold">TM</span>
                </div>
                <span className="text-base md:text-lg font-semibold">Tidiness Manager</span>
              </Link>

              {/* Desktop Navigation */}
              <nav className="hidden md:flex items-center gap-1">
                {navigation.map((item) => {
                  const Icon = item.icon;
                  const isActive = location.pathname === item.href ||
                    (item.href !== "/" && location.pathname.startsWith(item.href));

                  return (
                    <Link
                      key={item.name}
                      to={item.href}
                      className={cn(
                        "flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium transition-colors",
                        isActive
                          ? "bg-white/10 text-white"
                          : "text-slate-400 hover:text-slate-200 hover:bg-white/5"
                      )}
                    >
                      <Icon className="h-4 w-4" />
                      {item.name}
                    </Link>
                  );
                })}
              </nav>
            </div>

            <div className="flex items-center gap-3">
              <div className="hidden sm:block text-xs text-slate-400">v1.0.0</div>

              {/* Mobile menu button */}
              <button
                type="button"
                className="md:hidden -m-2.5 inline-flex items-center justify-center rounded-md p-2.5 text-slate-400 hover:text-slate-200 hover:bg-white/5"
                onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
                aria-label="Toggle navigation menu"
              >
                {mobileMenuOpen ? (
                  <X className="h-6 w-6" aria-hidden="true" />
                ) : (
                  <Menu className="h-6 w-6" aria-hidden="true" />
                )}
              </button>
            </div>
          </div>
        </div>
      </header>

      {/* Mobile Navigation Menu */}
      {mobileMenuOpen && (
        <div className="md:hidden border-b border-white/10 bg-black/30 backdrop-blur-sm">
          <div className="mx-auto max-w-7xl px-4 py-4 space-y-1">
            {navigation.map((item) => {
              const Icon = item.icon;
              const isActive = location.pathname === item.href ||
                (item.href !== "/" && location.pathname.startsWith(item.href));

              return (
                <Link
                  key={item.name}
                  to={item.href}
                  onClick={() => setMobileMenuOpen(false)}
                  className={cn(
                    "flex items-center gap-3 px-4 py-3 rounded-lg text-base font-medium transition-colors",
                    isActive
                      ? "bg-white/10 text-white"
                      : "text-slate-400 hover:text-slate-200 hover:bg-white/5"
                  )}
                >
                  <Icon className="h-5 w-5" />
                  {item.name}
                </Link>
              );
            })}
          </div>
        </div>
      )}

      {/* Main Content */}
      <main id="main-content" className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 py-6 md:py-8">
        {children}
      </main>

      {/* Footer */}
      <footer className="border-t border-white/10 bg-black/20 mt-auto">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 py-4">
          <div className="flex flex-col sm:flex-row items-center justify-between gap-2 text-xs text-slate-400 text-center sm:text-left">
            <p>Continuous tidiness prevents emergencies</p>
            <p className="hidden sm:block">Comprehensive coverage, zero redundancy</p>
          </div>
        </div>
      </footer>

      {/* Keyboard Shortcuts */}
      <KeyboardShortcuts onNavigate={navigate} />
    </div>
  );
}
