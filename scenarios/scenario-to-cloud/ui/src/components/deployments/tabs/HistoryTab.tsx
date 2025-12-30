import { useState } from "react";
import {
  RefreshCw,
  Loader2,
  AlertCircle,
  Clock,
  Search,
  Filter,
  ChevronDown,
} from "lucide-react";
import { useHistory, useLogs, getTimeSince } from "../../../hooks/useLiveState";
import { Timeline } from "./Timeline";
import { LogViewer } from "./LogViewer";
import { cn } from "../../../lib/utils";

interface HistoryTabProps {
  deploymentId: string;
}

export function HistoryTab({ deploymentId }: HistoryTabProps) {
  const [activeSection, setActiveSection] = useState<"timeline" | "logs">("timeline");
  const [logSource, setLogSource] = useState("all");
  const [logLevel, setLogLevel] = useState("all");
  const [logSearch, setLogSearch] = useState("");
  const [logTail, setLogTail] = useState(200);

  const {
    data: history,
    isLoading: historyLoading,
    error: historyError,
    refetch: refetchHistory,
    isFetching: historyFetching,
  } = useHistory(deploymentId);

  const {
    data: logsData,
    isLoading: logsLoading,
    error: logsError,
    refetch: refetchLogs,
    isFetching: logsFetching,
  } = useLogs(deploymentId, {
    source: logSource !== "all" ? logSource : undefined,
    level: logLevel !== "all" ? logLevel : undefined,
    tail: logTail,
    search: logSearch || undefined,
  });

  const handleRefresh = () => {
    if (activeSection === "timeline") {
      refetchHistory();
    } else {
      refetchLogs();
    }
  };

  const isLoading = activeSection === "timeline" ? historyLoading : logsLoading;
  const error = activeSection === "timeline" ? historyError : logsError;
  const isFetching = activeSection === "timeline" ? historyFetching : logsFetching;

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-blue-400" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-4 rounded-lg border border-red-500/30 bg-red-500/10 text-red-400">
        <div className="flex items-center gap-2">
          <AlertCircle className="h-5 w-5" />
          <span>Failed to load {activeSection}: {error.message}</span>
        </div>
      </div>
    );
  }

  const sections = [
    { id: "timeline" as const, label: "Timeline", icon: Clock },
    { id: "logs" as const, label: "Logs", icon: Filter },
  ];

  return (
    <div className="space-y-4">
      {/* Header with refresh */}
      <div className="flex items-center justify-between">
        <div className="flex gap-1 border-b border-white/10 pb-px">
          {sections.map(({ id, label, icon: Icon }) => (
            <button
              key={id}
              onClick={() => setActiveSection(id)}
              className={cn(
                "flex items-center gap-2 px-4 py-2 text-sm font-medium transition-colors rounded-t-lg",
                activeSection === id
                  ? "bg-slate-800 text-white border-b-2 border-blue-500"
                  : "text-slate-400 hover:text-white hover:bg-slate-800/50"
              )}
            >
              <Icon className="h-4 w-4" />
              {label}
            </button>
          ))}
        </div>
        <button
          onClick={handleRefresh}
          disabled={isFetching}
          className={cn(
            "flex items-center gap-2 px-3 py-1.5 rounded-lg border border-white/10",
            "hover:bg-white/5 transition-colors text-sm font-medium",
            isFetching && "opacity-50 cursor-not-allowed"
          )}
        >
          {isFetching ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <RefreshCw className="h-4 w-4" />
          )}
          Refresh
        </button>
      </div>

      {/* Log filters (only shown for logs section) */}
      {activeSection === "logs" && (
        <div className="flex flex-wrap items-center gap-3 p-3 rounded-lg border border-white/10 bg-slate-900/50">
          {/* Source filter */}
          <div className="flex items-center gap-2">
            <label className="text-xs text-slate-400">Source:</label>
            <select
              value={logSource}
              onChange={(e) => setLogSource(e.target.value)}
              className="px-2 py-1 text-sm rounded bg-slate-800 border border-white/10 text-white"
            >
              <option value="all">All Sources</option>
              <option value="scenario">Scenario</option>
              <option value="caddy">Caddy</option>
              {logsData?.sources?.filter(s => s !== "caddy" && s !== deploymentId.split("-")[0]).map(src => (
                <option key={src} value={src}>{src}</option>
              ))}
            </select>
          </div>

          {/* Level filter */}
          <div className="flex items-center gap-2">
            <label className="text-xs text-slate-400">Level:</label>
            <select
              value={logLevel}
              onChange={(e) => setLogLevel(e.target.value)}
              className="px-2 py-1 text-sm rounded bg-slate-800 border border-white/10 text-white"
            >
              <option value="all">All Levels</option>
              <option value="ERROR">Error</option>
              <option value="WARN">Warning</option>
              <option value="INFO">Info</option>
              <option value="DEBUG">Debug</option>
            </select>
          </div>

          {/* Tail count */}
          <div className="flex items-center gap-2">
            <label className="text-xs text-slate-400">Lines:</label>
            <select
              value={logTail}
              onChange={(e) => setLogTail(Number(e.target.value))}
              className="px-2 py-1 text-sm rounded bg-slate-800 border border-white/10 text-white"
            >
              <option value={50}>50</option>
              <option value={100}>100</option>
              <option value={200}>200</option>
              <option value={500}>500</option>
              <option value={1000}>1000</option>
            </select>
          </div>

          {/* Search */}
          <div className="flex-1 flex items-center gap-2 min-w-[200px]">
            <Search className="h-4 w-4 text-slate-400" />
            <input
              type="text"
              value={logSearch}
              onChange={(e) => setLogSearch(e.target.value)}
              placeholder="Search logs..."
              className="flex-1 px-2 py-1 text-sm rounded bg-slate-800 border border-white/10 text-white placeholder-slate-500"
            />
          </div>
        </div>
      )}

      {/* Content */}
      <div className="mt-4">
        {activeSection === "timeline" && history && (
          <Timeline events={history} />
        )}
        {activeSection === "logs" && logsData && (
          <LogViewer
            logs={logsData.logs}
            total={logsData.total}
            sources={logsData.sources}
          />
        )}
      </div>
    </div>
  );
}
