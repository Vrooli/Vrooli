import { useQuery } from "@tanstack/react-query";
import { Cloud, RefreshCw, CheckCircle2, XCircle } from "lucide-react";
import { fetchHealth } from "../../lib/api";
import { cn } from "../../lib/utils";

interface LayoutProps {
  children: React.ReactNode;
}

export function Layout({ children }: LayoutProps) {
  const { data: health, isLoading, error, refetch } = useQuery({
    queryKey: ["health"],
    queryFn: fetchHealth,
    refetchInterval: 30000, // Refresh every 30s
  });

  return (
    <div className="min-h-screen bg-slate-950 text-slate-50">
      {/* Header */}
      <header className="border-b border-white/10 bg-slate-900/50 backdrop-blur-lg sticky top-0 z-50">
        <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            {/* Logo & Title */}
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-lg bg-gradient-to-br from-blue-500 to-purple-600">
                <Cloud className="h-5 w-5 text-white" />
              </div>
              <div>
                <h1 className="text-lg font-semibold text-white">Scenario To Cloud</h1>
                <p className="text-xs text-slate-400 hidden sm:block">Deploy scenarios to VPS</p>
              </div>
            </div>

            {/* API Status */}
            <div className="flex items-center gap-3">
              <div className="flex items-center gap-2">
                {isLoading && (
                  <div className="flex items-center gap-2 text-sm text-slate-400">
                    <RefreshCw className="h-4 w-4 animate-spin" />
                    <span className="hidden sm:inline">Checking...</span>
                  </div>
                )}
                {!isLoading && error && (
                  <div className="flex items-center gap-2 text-sm text-red-400">
                    <XCircle className="h-4 w-4" />
                    <span className="hidden sm:inline">API Offline</span>
                  </div>
                )}
                {!isLoading && health && (
                  <div className="flex items-center gap-2 text-sm text-emerald-400">
                    <CheckCircle2 className="h-4 w-4" />
                    <span className="hidden sm:inline">API Online</span>
                  </div>
                )}
              </div>
              <button
                onClick={() => refetch()}
                disabled={isLoading}
                className={cn(
                  "p-2 rounded-lg border border-white/10 hover:bg-white/5 transition-colors",
                  isLoading && "opacity-50 cursor-not-allowed"
                )}
                aria-label="Refresh API status"
              >
                <RefreshCw className={cn("h-4 w-4", isLoading && "animate-spin")} />
              </button>
            </div>
          </div>
        </div>
      </header>

      {/* Main content */}
      <main className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 py-6 sm:py-8">
        {children}
      </main>
    </div>
  );
}
