import { useState } from "react";
import { Link, useLocation } from "react-router-dom";
import {
  LayoutGrid,
  FolderTree,
  Monitor,
  Rocket,
  Settings,
  Package,
  Menu,
  X
} from "lucide-react";
import { cn } from "../lib/utils";

interface LayoutProps {
  children: React.ReactNode;
}

export function Layout({ children }: LayoutProps) {
  const location = useLocation();
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  const navItems = [
    { path: "/", label: "Dashboard", icon: LayoutGrid },
    { path: "/profiles", label: "Profiles", icon: Package },
    { path: "/analyze", label: "Analyze", icon: FolderTree },
    { path: "/deployments", label: "Deployments", icon: Rocket },
    { path: "/telemetry", label: "Telemetry", icon: Monitor },
  ];

  return (
    <div className="min-h-screen bg-slate-950 text-slate-50">
      {/* Header */}
      <header className="sticky top-0 z-50 border-b border-white/10 bg-slate-950/80 backdrop-blur">
        <div className="flex h-16 items-center px-4 md:px-6">
          <button
            className="mr-4 md:hidden"
            onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
            aria-label="Toggle menu"
          >
            {mobileMenuOpen ? (
              <X className="h-6 w-6 text-slate-400" />
            ) : (
              <Menu className="h-6 w-6 text-slate-400" />
            )}
          </button>
          <div className="flex items-center gap-3">
            <Settings className="h-6 w-6 text-cyan-400" />
            <h1 className="text-lg md:text-xl font-semibold">Deployment Manager</h1>
          </div>
        </div>
      </header>

      <div className="flex">
        {/* Mobile Overlay */}
        {mobileMenuOpen && (
          <div
            className="fixed inset-0 z-40 bg-black/50 md:hidden"
            onClick={() => setMobileMenuOpen(false)}
          />
        )}

        {/* Sidebar */}
        <aside
          className={cn(
            "fixed md:sticky top-16 z-40 h-[calc(100vh-4rem)] w-64 border-r border-white/10 bg-slate-950 md:block",
            mobileMenuOpen
              ? "block left-0"
              : "hidden"
          )}
        >
          <nav className="flex flex-col gap-1 p-4">
            {navItems.map((item) => {
              const Icon = item.icon;
              const isActive = location.pathname === item.path;

              return (
                <Link
                  key={item.path}
                  to={item.path}
                  onClick={() => setMobileMenuOpen(false)}
                  className={cn(
                    "flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors",
                    isActive
                      ? "bg-cyan-500/10 text-cyan-400"
                      : "text-slate-400 hover:bg-white/5 hover:text-slate-200"
                  )}
                >
                  <Icon className="h-4 w-4" />
                  {item.label}
                </Link>
              );
            })}
          </nav>
        </aside>

        {/* Main Content */}
        <main className="flex-1 p-4 md:p-6">
          {children}
        </main>
      </div>
    </div>
  );
}
