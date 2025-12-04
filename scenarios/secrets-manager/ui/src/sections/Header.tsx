import { RefreshCcw } from "lucide-react";
import { Button } from "../components/ui/button";

interface HeaderProps {
  isInitialLoading: boolean;
  isRefreshing: boolean;
  onRefresh: () => void;
}

export const Header = ({ isInitialLoading, isRefreshing, onRefresh }: HeaderProps) => (
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
      <div className="flex items-center gap-2">
        {isInitialLoading ? (
          <div className="flex items-center gap-2 rounded-full border border-cyan-400/40 bg-cyan-400/10 px-3 py-2 text-[11px] uppercase tracking-[0.2em] text-cyan-100">
            <RefreshCcw className="h-3 w-3 animate-spin" />
            Loading
          </div>
        ) : null}
        <Button
          variant="outline"
          size="sm"
          onClick={onRefresh}
          disabled={isRefreshing}
          aria-label="Refresh data"
          className="h-10 w-10 p-0"
        >
          <RefreshCcw className={`h-4 w-4 ${isRefreshing ? "animate-spin" : ""}`} />
        </Button>
      </div>
    </div>
  </header>
);
