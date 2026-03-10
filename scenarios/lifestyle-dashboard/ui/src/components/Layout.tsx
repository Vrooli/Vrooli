/**
 * Layout Component
 *
 * Main application layout with header and navigation.
 * Wraps all pages with consistent styling and health status.
 *
 * [REQ:LD-DASHBOARD-TIMELINE] - Unified dashboard UI structure
 */
import { useQuery } from "@tanstack/react-query";
import { Outlet, NavLink } from "react-router-dom";
import { Home, Heart, Activity, Settings, RefreshCw, Sun } from "lucide-react";

import { fetchHealth } from "../lib/api";

export function Layout() {
  const healthQuery = useQuery({
    queryKey: ["health"],
    queryFn: fetchHealth,
    refetchInterval: 30000,
  });

  const handleRefresh = () => {
    healthQuery.refetch();
  };

  return (
    // INTEROP-CRITICAL: Use h-full with overflow-auto for iframe compatibility
    // min-h-screen uses vh which can be incorrect inside iframes
    <div className="h-full flex flex-col overflow-hidden bg-slate-950 text-slate-50">
      {/* Header */}
      <header className="flex-shrink-0 sticky top-0 z-10 bg-slate-950/80 backdrop-blur border-b border-white/10">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            {/* Logo and nav */}
            <div className="flex items-center gap-8">
              <NavLink to="/" className="flex items-center gap-3">
                <Heart className="w-8 h-8 text-rose-500" />
                <span className="text-xl font-bold hidden sm:block">Lifestyle Dashboard</span>
              </NavLink>

              {/* Navigation */}
              <nav className="flex items-center gap-1">
                <NavLink
                  to="/"
                  end
                  className={({ isActive }) =>
                    `flex items-center gap-2 px-3 py-2 rounded-lg text-sm transition-colors ${
                      isActive
                        ? "bg-white/10 text-white"
                        : "text-slate-400 hover:text-white hover:bg-white/5"
                    }`
                  }
                >
                  <Home className="w-4 h-4" />
                  <span className="hidden sm:inline">Dashboard</span>
                </NavLink>
                <NavLink
                  to="/domains"
                  className={({ isActive }) =>
                    `flex items-center gap-2 px-3 py-2 rounded-lg text-sm transition-colors ${
                      isActive
                        ? "bg-white/10 text-white"
                        : "text-slate-400 hover:text-white hover:bg-white/5"
                    }`
                  }
                >
                  <Heart className="w-4 h-4" />
                  <span className="hidden sm:inline">Domains</span>
                </NavLink>
                <NavLink
                  to="/events"
                  className={({ isActive }) =>
                    `flex items-center gap-2 px-3 py-2 rounded-lg text-sm transition-colors ${
                      isActive
                        ? "bg-white/10 text-white"
                        : "text-slate-400 hover:text-white hover:bg-white/5"
                    }`
                  }
                >
                  <Activity className="w-4 h-4" />
                  <span className="hidden sm:inline">Events</span>
                </NavLink>
                <NavLink
                  to="/briefs"
                  className={({ isActive }) =>
                    `flex items-center gap-2 px-3 py-2 rounded-lg text-sm transition-colors ${
                      isActive
                        ? "bg-white/10 text-white"
                        : "text-slate-400 hover:text-white hover:bg-white/5"
                    }`
                  }
                >
                  <Sun className="w-4 h-4" />
                  <span className="hidden sm:inline">Briefs</span>
                </NavLink>
                <NavLink
                  to="/settings"
                  className={({ isActive }) =>
                    `flex items-center gap-2 px-3 py-2 rounded-lg text-sm transition-colors ${
                      isActive
                        ? "bg-white/10 text-white"
                        : "text-slate-400 hover:text-white hover:bg-white/5"
                    }`
                  }
                >
                  <Settings className="w-4 h-4" />
                  <span className="hidden sm:inline">Settings</span>
                </NavLink>
              </nav>
            </div>

            {/* Health status and refresh */}
            <div className="flex items-center gap-4">
              <div className="flex items-center gap-2 text-sm">
                <div
                  className={`w-2 h-2 rounded-full ${
                    healthQuery.data?.status === "ok"
                      ? "bg-green-500"
                      : healthQuery.error
                      ? "bg-red-500"
                      : "bg-yellow-500"
                  }`}
                />
                <span className="text-slate-400 hidden sm:inline">
                  {healthQuery.data?.status === "ok" ? "Healthy" : healthQuery.error ? "Error" : "Checking..."}
                </span>
              </div>
              <button
                onClick={handleRefresh}
                className="p-2 rounded-lg hover:bg-white/5 transition-colors"
                title="Refresh"
              >
                <RefreshCw className={`w-5 h-5 ${healthQuery.isFetching ? "animate-spin" : ""}`} />
              </button>
            </div>
          </div>
        </div>
      </header>

      {/* Main content - scrollable area */}
      <main className="flex-1 overflow-auto">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
          <Outlet />
        </div>
      </main>

      {/* Footer with system info */}
      {healthQuery.data && (
        <footer className="flex-shrink-0 max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 pb-6">
          <div className="rounded-xl border border-white/10 bg-white/5 p-4">
            <div className="flex flex-wrap items-center gap-4 text-sm text-slate-500">
              <span>Service: {healthQuery.data.service}</span>
              <span>•</span>
              <span>Version: {healthQuery.data.version || "1.0.0"}</span>
              <span>•</span>
              <span>Uptime: {Math.floor(healthQuery.data.uptime_seconds || 0)}s</span>
              {healthQuery.data.dependencies?.database && (
                <>
                  <span>•</span>
                  <span>
                    Database: {healthQuery.data.dependencies.database.connected ? "connected" : "disconnected"}
                    {healthQuery.data.dependencies.database.latency_ms && (
                      <> ({healthQuery.data.dependencies.database.latency_ms.toFixed(1)}ms)</>
                    )}
                  </span>
                </>
              )}
            </div>
          </div>
        </footer>
      )}
    </div>
  );
}
