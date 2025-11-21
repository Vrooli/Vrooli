import { RefreshCcw } from "lucide-react";
import { Button } from "../components/ui/button";

interface HeaderProps {
  isInitialLoading: boolean;
  isRefreshing: boolean;
  totalFindings?: number;
  onRefresh: () => void;
}

export const Header = ({ isInitialLoading, isRefreshing, totalFindings, onRefresh }: HeaderProps) => (
  <header className="rounded-3xl border border-white/10 bg-black/30 p-6 shadow-2xl shadow-emerald-500/10">
    <p className="text-xs uppercase tracking-[0.3em] text-emerald-300">Security Vault Access Terminal</p>
    <div className="mt-3 flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
      <div>
        <h1 className="text-3xl font-semibold tracking-tight text-white">Secrets Manager</h1>
        <p className="mt-2 max-w-2xl text-sm text-white/70">
          Discover, validate, and provision secrets across the Vrooli resource graph. Monitor vault readiness,
          identify critical vulnerabilities, and trigger remediation from one unified control room.
        </p>
      </div>
      <div className="flex items-center gap-3">
        {isInitialLoading ? (
          <div className="flex items-center gap-2 rounded-full border border-cyan-400/40 bg-cyan-400/10 px-4 py-2 text-xs uppercase tracking-[0.2em] text-cyan-100">
            <RefreshCcw className="h-3 w-3 animate-spin" />
            Loading data...
          </div>
        ) : (
          <div className="rounded-full border border-emerald-400/40 bg-emerald-400/10 px-4 py-2 text-xs uppercase tracking-[0.2em] text-emerald-100">
            {totalFindings !== undefined ? `${totalFindings} Findings` : "Ready"}
          </div>
        )}
        <Button variant="outline" size="sm" onClick={onRefresh} disabled={isRefreshing} className="gap-2">
          <RefreshCcw className={`h-4 w-4 ${isRefreshing ? "animate-spin" : ""}`} />
          Refresh data
        </Button>
      </div>
    </div>
  </header>
);
