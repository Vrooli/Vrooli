import { NavLink, Outlet } from "react-router-dom";
import { LayoutDashboard, ListTodo, FolderKanban, Activity } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { fetchHealth } from "../lib/api";

const navItems = [
  { to: "/", icon: LayoutDashboard, label: "Dashboard" },
  { to: "/tasks", icon: ListTodo, label: "Tasks" },
  { to: "/projects", icon: FolderKanban, label: "Projects" }
] as const;

function HealthIndicator() {
  const { data, isLoading, error } = useQuery({
    queryKey: ["health"],
    queryFn: fetchHealth,
    refetchInterval: 30000 // Refetch every 30 seconds
  });

  const status = isLoading ? "checking" : error ? "offline" : data?.status ?? "unknown";
  const statusColor = {
    checking: "text-yellow-400",
    offline: "text-red-400",
    healthy: "text-green-400",
    unknown: "text-slate-400"
  }[status] ?? "text-slate-400";

  return (
    <div
      data-testid="health-indicator"
      className="flex items-center gap-2 text-xs"
      title={error ? "API is not reachable" : `API Status: ${status}`}
    >
      <Activity className={`h-3.5 w-3.5 ${statusColor}`} />
      <span className="text-slate-400 capitalize">{status}</span>
    </div>
  );
}

export function Layout() {
  return (
    <div className="h-full flex flex-col overflow-hidden bg-slate-950 text-slate-50">
      {/* Header */}
      <header
        data-testid="app-header"
        className="flex-shrink-0 border-b border-white/10 bg-slate-900/50 px-4 py-3"
      >
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-6">
            <h1 className="text-lg font-semibold">Task Manager</h1>
            <nav className="flex items-center gap-1" data-testid="main-nav">
              {navItems.map(({ to, icon: Icon, label }) => (
                <NavLink
                  key={to}
                  to={to}
                  data-testid={`nav-${label.toLowerCase()}`}
                  className={({ isActive }) =>
                    `flex items-center gap-2 rounded-lg px-3 py-2 text-sm transition-colors ${
                      isActive
                        ? "bg-white/10 text-white"
                        : "text-slate-400 hover:bg-white/5 hover:text-slate-200"
                    }`
                  }
                >
                  <Icon className="h-4 w-4" />
                  {label}
                </NavLink>
              ))}
            </nav>
          </div>
          <HealthIndicator />
        </div>
      </header>

      {/* Main content area */}
      <main className="flex-1 overflow-auto p-6" data-testid="main-content">
        <Outlet />
      </main>
    </div>
  );
}
