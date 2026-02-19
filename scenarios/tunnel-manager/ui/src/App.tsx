import { useState, useEffect } from "react";
import { BrowserRouter, Routes, Route, NavLink, useLocation } from "react-router-dom";
import { LayoutDashboard, GitBranch, Shield, Radio, BarChart3, Heart, Menu, X } from "lucide-react";
import DashboardPage from "./pages/DashboardPage";
import RoutesPage from "./pages/RoutesPage";
import RecoveryPage from "./pages/RecoveryPage";
import ProbesPage from "./pages/ProbesPage";
import MetricsPage from "./pages/MetricsPage";
import HealthPage from "./pages/HealthPage";
import RouteDetailPage from "./pages/RouteDetailPage";

const navItems = [
  { to: "/", label: "Dashboard", icon: LayoutDashboard },
  { to: "/routes", label: "Routes", icon: GitBranch },
  { to: "/probes", label: "Probes", icon: Radio },
  { to: "/metrics", label: "Metrics", icon: BarChart3 },
  { to: "/health", label: "Health", icon: Heart },
  { to: "/recovery", label: "Recovery", icon: Shield },
] as const;

const routeNames: Record<string, string> = {
  "/": "Dashboard",
  "/routes": "Routes",
  "/probes": "Probes",
  "/metrics": "Metrics",
  "/health": "Health",
  "/recovery": "Recovery",
};

/** Announces page changes to screen readers and updates document title */
function RouteAnnouncer() {
  const location = useLocation();
  const [announcement, setAnnouncement] = useState("");

  useEffect(() => {
    const name = routeNames[location.pathname] ?? "Page";
    setAnnouncement(`Navigated to ${name}`);
    document.title = name === "Dashboard" ? "Tunnel Manager" : `${name} - Tunnel Manager`;
  }, [location.pathname]);

  return (
    <div aria-live="assertive" aria-atomic="true" role="status" className="sr-only">
      {announcement}
    </div>
  );
}

function NavItem({ to, label, icon: Icon, onClick }: { to: string; label: string; icon: React.ElementType; onClick?: () => void }) {
  return (
    <NavLink
      to={to}
      onClick={onClick}
      className={({ isActive }) =>
        `flex items-center gap-2 rounded-lg px-3 py-2 text-sm transition-colors ${
          isActive ? "bg-blue-500/20 text-blue-400" : "text-slate-300 hover:text-slate-100 hover:bg-white/5"
        }`
      }
    >
      {({ isActive }) => (
        <>
          <Icon className="h-4 w-4" aria-hidden="true" />
          <span aria-current={isActive ? "page" : undefined}>{label}</span>
        </>
      )}
    </NavLink>
  );
}

export default function App() {
  const [mobileNavOpen, setMobileNavOpen] = useState(false);

  return (
    <BrowserRouter basename="/">
      <div className="min-h-screen bg-slate-950 text-slate-50">
        <RouteAnnouncer />
        <a
          href="#main-content"
          className="sr-only focus:not-sr-only focus:fixed focus:top-2 focus:left-2 focus:z-50 focus:rounded-lg focus:bg-blue-600 focus:px-4 focus:py-2 focus:text-sm focus:text-white focus:shadow-lg"
          data-testid="skip-link"
        >
          Skip to content
        </a>
        <header className="sticky top-0 z-40 border-b border-white/10 bg-slate-950/95 backdrop-blur-sm px-4 py-3 sm:px-6 sm:py-4" data-testid="app-header">
          <div className="flex items-center justify-between">
            <div className="min-w-0">
              <h1 className="text-lg font-semibold sm:text-xl" data-testid="app-title">Tunnel Manager</h1>
              <p className="text-xs text-slate-300 sm:text-sm">Cloudflare tunnel monitoring and management</p>
            </div>

            {/* Desktop nav */}
            <nav aria-label="Main navigation" className="hidden md:flex items-center gap-1" data-testid="desktop-nav">
              {navItems.map((item) => (
                <NavItem key={item.to} {...item} />
              ))}
            </nav>

            {/* Mobile menu button */}
            <button
              className="md:hidden flex items-center justify-center h-10 w-10 rounded-lg text-slate-300 hover:bg-white/10 transition-colors"
              onClick={() => setMobileNavOpen(!mobileNavOpen)}
              aria-label={mobileNavOpen ? "Close menu" : "Open menu"}
              aria-expanded={mobileNavOpen}
              data-testid="mobile-menu-toggle"
            >
              {mobileNavOpen ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
            </button>
          </div>

          {/* Mobile nav drawer */}
          {mobileNavOpen && (
            <nav
              aria-label="Main navigation"
              className="mt-3 flex flex-col gap-1 border-t border-white/10 pt-3 md:hidden"
              data-testid="mobile-nav"
            >
              {navItems.map((item) => (
                <NavItem key={item.to} {...item} onClick={() => setMobileNavOpen(false)} />
              ))}
            </nav>
          )}
        </header>

        <main id="main-content" className="mx-auto max-w-6xl p-4 sm:p-6" data-testid="main-content">
          <Routes>
            <Route path="/" element={<DashboardPage />} />
            <Route path="/routes" element={<RoutesPage />} />
            <Route path="/routes/:id" element={<RouteDetailPage />} />
            <Route path="/probes" element={<ProbesPage />} />
            <Route path="/metrics" element={<MetricsPage />} />
            <Route path="/health" element={<HealthPage />} />
            <Route path="/recovery" element={<RecoveryPage />} />
          </Routes>
        </main>
      </div>
    </BrowserRouter>
  );
}
