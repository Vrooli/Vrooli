/**
 * Header component for the Lifestyle Dashboard.
 * Displays the app title, health status, and refresh controls.
 *
 * [REQ:LD-DASHBOARD-TIMELINE] - Dashboard header with status
 */
import { Activity, RefreshCw, CheckCircle2, AlertCircle } from "lucide-react";
import { Button } from "../ui/button";
import { StatusBadge } from "./StatusBadge";
import type { HealthResponse } from "../../lib/api";

interface HeaderProps {
  health?: HealthResponse;
  isLoading: boolean;
  onRefresh: () => void;
}

export function Header({ health, isLoading, onRefresh }: HeaderProps) {
  return (
    <header className="border-b border-white/10 bg-slate-950/80 backdrop-blur sticky top-0 z-10">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-violet-500 to-fuchsia-500 flex items-center justify-center">
              <Activity className="w-5 h-5 text-white" />
            </div>
            <div>
              <h1 className="text-xl font-semibold">Lifestyle Dashboard</h1>
              <p className="text-xs text-slate-500">Personal health intelligence</p>
            </div>
          </div>
          <div className="flex items-center gap-3">
            {health && (
              <div className="flex items-center gap-2">
                {health.status === "healthy" ? (
                  <CheckCircle2 className="w-4 h-4 text-emerald-400" />
                ) : (
                  <AlertCircle className="w-4 h-4 text-amber-400" />
                )}
                <StatusBadge status={health.status} />
              </div>
            )}
            <Button
              variant="outline"
              size="sm"
              onClick={onRefresh}
              disabled={isLoading}
              className="border-white/10 hover:bg-white/5"
            >
              <RefreshCw className={`w-4 h-4 mr-2 ${isLoading ? "animate-spin" : ""}`} />
              Refresh
            </Button>
          </div>
        </div>
      </div>
    </header>
  );
}
