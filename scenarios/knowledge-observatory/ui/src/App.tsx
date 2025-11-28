import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Activity, Database, GitGraph, Search, X } from "lucide-react";
import { Button } from "./components/ui/button";
import { fetchHealth } from "./lib/api";
import { SearchPanel } from "./components/SearchPanel";
import { MetricsPanel } from "./components/MetricsPanel";

type ActivePanel = "search" | "metrics" | "graph" | null;

export default function App() {
  const [activePanel, setActivePanel] = useState<ActivePanel>(null);
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["health"],
    queryFn: fetchHealth,
    refetchInterval: 5000
  });

  return (
    <div className="min-h-screen bg-black text-green-400 font-mono">
      {/* Matrix-style header */}
      <header className="border-b border-green-900/50 bg-black/90 backdrop-blur px-6 py-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Database className="h-8 w-8 text-green-500" />
            <div>
              <h1 className="text-2xl font-bold tracking-tight">Knowledge Observatory</h1>
              <p className="text-sm text-green-600">Consciousness Monitor • Semantic Intelligence System</p>
            </div>
          </div>
          <div className="flex items-center gap-4">
            <div className="flex items-center gap-2 px-3 py-1.5 rounded bg-green-950/30 border border-green-900/50">
              <Activity className="h-4 w-4 animate-pulse" />
              <span className="text-xs font-semibold uppercase tracking-wider">
                {isLoading ? "Syncing..." : error ? "Offline" : "Online"}
              </span>
            </div>
          </div>
        </div>
      </header>

      {/* Main dashboard */}
      <main className="p-6 space-y-6">
        {/* API Health Status */}
        <section className="border border-green-900/50 bg-green-950/10 p-6 rounded-lg">
          <div className="flex items-center gap-2 mb-4">
            <Activity className="h-5 w-5" />
            <h2 className="text-lg font-semibold">System Health</h2>
          </div>

          {isLoading && (
            <div className="space-y-2">
              <div className="h-2 bg-green-900/30 animate-pulse rounded"></div>
              <p className="text-sm text-green-600">Querying knowledge base status...</p>
            </div>
          )}

          {error && (
            <div className="p-4 border border-red-900/50 bg-red-950/20 rounded">
              <p className="text-red-400 font-semibold">⚠ Connection Error</p>
              <p className="text-sm text-red-600 mt-1">
                Unable to establish connection to API. Verify scenario is running.
              </p>
            </div>
          )}

          {data && (
            <div className="grid grid-cols-3 gap-4">
              <div className="border border-green-900/50 bg-black/40 p-4 rounded">
                <p className="text-xs text-green-600 uppercase tracking-wider">Status</p>
                <p className="text-xl font-bold mt-1">{data.status}</p>
              </div>
              <div className="border border-green-900/50 bg-black/40 p-4 rounded">
                <p className="text-xs text-green-600 uppercase tracking-wider">Service</p>
                <p className="text-xl font-bold mt-1">{data.service}</p>
              </div>
              <div className="border border-green-900/50 bg-black/40 p-4 rounded">
                <p className="text-xs text-green-600 uppercase tracking-wider">Last Update</p>
                <p className="text-sm font-semibold mt-1">{new Date(data.timestamp).toLocaleTimeString()}</p>
              </div>
            </div>
          )}

          <Button
            className="mt-4 bg-green-900 hover:bg-green-800 text-green-100 border border-green-700"
            onClick={() => refetch()}
          >
            Refresh Status
          </Button>
        </section>

        {/* Feature Overview */}
        <section className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <button
            onClick={() => setActivePanel("search")}
            className="border border-green-900/50 bg-green-950/10 p-6 rounded-lg hover:bg-green-950/20 hover:border-green-700 transition-all cursor-pointer text-left"
          >
            <Search className="h-8 w-8 text-green-500 mb-3" />
            <h3 className="text-lg font-semibold mb-2">Semantic Search</h3>
            <p className="text-sm text-green-600">
              Query knowledge base using natural language across all collections
            </p>
          </button>

          <button
            onClick={() => setActivePanel("graph")}
            className="border border-green-900/50 bg-green-950/10 p-6 rounded-lg hover:bg-green-950/20 hover:border-green-700 transition-all cursor-pointer text-left opacity-60"
            disabled
          >
            <GitGraph className="h-8 w-8 text-green-500 mb-3" />
            <h3 className="text-lg font-semibold mb-2">Knowledge Graph</h3>
            <p className="text-sm text-green-600">
              Visualize concept relationships and semantic connections
            </p>
            <p className="text-xs text-green-700 mt-2 italic">Coming soon</p>
          </button>

          <button
            onClick={() => setActivePanel("metrics")}
            className="border border-green-900/50 bg-green-950/10 p-6 rounded-lg hover:bg-green-950/20 hover:border-green-700 transition-all cursor-pointer text-left"
          >
            <Database className="h-8 w-8 text-green-500 mb-3" />
            <h3 className="text-lg font-semibold mb-2">Quality Metrics</h3>
            <p className="text-sm text-green-600">
              Monitor coherence, freshness, and redundancy scores
            </p>
          </button>
        </section>

        {/* Quick Start Info */}
        <section className="border border-green-900/50 bg-green-950/10 p-6 rounded-lg">
          <h2 className="text-lg font-semibold mb-4">CLI Commands</h2>
          <div className="space-y-2 text-sm">
            <div className="bg-black/40 p-3 rounded border border-green-900/30">
              <code className="text-green-300">knowledge-observatory search "your query"</code>
              <p className="text-green-600 text-xs mt-1">Semantic search across knowledge base</p>
            </div>
            <div className="bg-black/40 p-3 rounded border border-green-900/30">
              <code className="text-green-300">knowledge-observatory health --watch</code>
              <p className="text-green-600 text-xs mt-1">Real-time health monitoring</p>
            </div>
            <div className="bg-black/40 p-3 rounded border border-green-900/30">
              <code className="text-green-300">knowledge-observatory graph --center "concept"</code>
              <p className="text-green-600 text-xs mt-1">Generate knowledge relationship graph</p>
            </div>
          </div>
        </section>
      </main>

      {/* Feature Panel Modal */}
      {activePanel && (
        <div className="fixed inset-0 bg-black/80 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-black border border-green-900/50 rounded-lg max-w-4xl w-full max-h-[90vh] overflow-hidden flex flex-col">
            {/* Modal Header */}
            <div className="flex items-center justify-between p-6 border-b border-green-900/50">
              <div className="flex items-center gap-3">
                {activePanel === "search" && <Search className="h-6 w-6 text-green-500" />}
                {activePanel === "metrics" && <Database className="h-6 w-6 text-green-500" />}
                {activePanel === "graph" && <GitGraph className="h-6 w-6 text-green-500" />}
                <h2 className="text-xl font-semibold">
                  {activePanel === "search" && "Semantic Search"}
                  {activePanel === "metrics" && "Quality Metrics"}
                  {activePanel === "graph" && "Knowledge Graph"}
                </h2>
              </div>
              <button
                onClick={() => setActivePanel(null)}
                className="p-2 hover:bg-green-950/30 rounded transition-colors"
              >
                <X className="h-5 w-5" />
              </button>
            </div>

            {/* Modal Content */}
            <div className="p-6 overflow-y-auto flex-1">
              {activePanel === "search" && <SearchPanel />}
              {activePanel === "metrics" && <MetricsPanel />}
              {activePanel === "graph" && (
                <div className="text-center p-12">
                  <GitGraph className="h-16 w-16 text-green-700 mx-auto mb-4" />
                  <p className="text-green-600">Knowledge Graph visualization coming soon</p>
                  <p className="text-sm text-green-700 mt-2">
                    This feature will display semantic relationships between concepts
                  </p>
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
