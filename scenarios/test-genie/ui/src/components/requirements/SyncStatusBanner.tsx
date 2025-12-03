import { useState } from "react";
import { RefreshCw, ChevronDown, Check, X } from "lucide-react";
import { cn } from "../../lib/utils";
import { formatRelative } from "../../lib/formatters";
import { selectors } from "../../consts/selectors";
import type { SyncStatus } from "../../lib/api";

interface SyncStatusBannerProps {
  syncStatus: SyncStatus | null;
  onSync: (options?: { dryRun?: boolean; pruneOrphans?: boolean }) => void;
  isSyncing: boolean;
}

export function SyncStatusBanner({ syncStatus, onSync, isSyncing }: SyncStatusBannerProps) {
  const [dropdownOpen, setDropdownOpen] = useState(false);

  const isEnabled = syncStatus?.enabled ?? true;
  const lastSynced = syncStatus?.lastSyncedAt;
  const statusesChanged = syncStatus?.statusesChanged ?? 0;

  return (
    <div className="flex items-center justify-between rounded-xl border border-white/10 bg-white/[0.02] px-4 py-3" data-testid={selectors.requirements.syncBanner}>
      <div className="flex items-center gap-3">
        <div
          className={cn(
            "h-2 w-2 rounded-full",
            isEnabled ? "bg-emerald-400" : "bg-slate-500"
          )}
        />
        <div className="text-sm">
          <span className="text-slate-300">
            Auto-sync {isEnabled ? "enabled" : "disabled"}
          </span>
          {lastSynced && (
            <>
              <span className="mx-2 text-slate-600">|</span>
              <span className="text-slate-400">
                Last synced {formatRelative(lastSynced)}
              </span>
            </>
          )}
          {statusesChanged > 0 && (
            <>
              <span className="mx-2 text-slate-600">|</span>
              <span className="text-slate-400">
                {statusesChanged} updated
              </span>
            </>
          )}
        </div>
      </div>

      <div className="relative">
        <button
          type="button"
          onClick={() => setDropdownOpen(!dropdownOpen)}
          disabled={isSyncing}
          className={cn(
            "flex items-center gap-2 rounded-lg border border-white/10 bg-white/5 px-3 py-1.5 text-sm transition",
            "hover:bg-white/10 disabled:opacity-50"
          )}
        >
          <RefreshCw className={cn("h-4 w-4", isSyncing && "animate-spin")} />
          <span>Sync Now</span>
          <ChevronDown className="h-3 w-3" />
        </button>

        {dropdownOpen && (
          <>
            <div
              className="fixed inset-0 z-10"
              onClick={() => setDropdownOpen(false)}
            />
            <div className="absolute right-0 top-full z-20 mt-1 w-56 rounded-lg border border-white/10 bg-slate-900 py-1 shadow-xl">
              <button
                type="button"
                onClick={() => {
                  onSync();
                  setDropdownOpen(false);
                }}
                className="flex w-full items-center gap-2 px-3 py-2 text-left text-sm hover:bg-white/5"
              >
                <RefreshCw className="h-4 w-4" />
                Sync Now
              </button>
              <button
                type="button"
                onClick={() => {
                  onSync({ dryRun: true });
                  setDropdownOpen(false);
                }}
                className="flex w-full items-center gap-2 px-3 py-2 text-left text-sm hover:bg-white/5"
              >
                <Check className="h-4 w-4" />
                Preview Changes (Dry Run)
              </button>
              <button
                type="button"
                onClick={() => {
                  onSync({ pruneOrphans: true });
                  setDropdownOpen(false);
                }}
                className="flex w-full items-center gap-2 px-3 py-2 text-left text-sm hover:bg-white/5"
              >
                <X className="h-4 w-4" />
                Sync with Prune Orphans
              </button>
              <div className="my-1 border-t border-white/10" />
              <div className="flex items-center justify-between px-3 py-2 text-sm">
                <span className="text-slate-400">Auto-sync</span>
                <span className={cn(
                  "rounded-full px-2 py-0.5 text-xs",
                  isEnabled ? "bg-emerald-500/20 text-emerald-300" : "bg-slate-500/20 text-slate-400"
                )}>
                  {isEnabled ? "Enabled" : "Disabled"}
                </span>
              </div>
            </div>
          </>
        )}
      </div>
    </div>
  );
}
