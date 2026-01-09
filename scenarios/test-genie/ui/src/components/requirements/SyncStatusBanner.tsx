import { useEffect, useState } from "react";
import { RefreshCw, Check } from "lucide-react";
import { cn } from "../../lib/utils";
import { formatRelative } from "../../lib/formatters";
import { selectors } from "../../consts/selectors";
import type { SyncStatus } from "../../lib/api";

interface SyncStatusBannerProps {
  syncStatus: SyncStatus | null;
  onSync: () => void;
  isSyncing: boolean;
  lastSyncSuccess?: boolean;
}

export function SyncStatusBanner({ syncStatus, onSync, isSyncing, lastSyncSuccess }: SyncStatusBannerProps) {
  const [showSuccess, setShowSuccess] = useState(false);

  // Show success state briefly when sync completes successfully
  useEffect(() => {
    if (lastSyncSuccess && !isSyncing) {
      setShowSuccess(true);
      const timer = setTimeout(() => setShowSuccess(false), 2000);
      return () => clearTimeout(timer);
    }
  }, [lastSyncSuccess, isSyncing]);

  const lastSynced = syncStatus?.lastSyncedAt;
  const lastSyncedFormatted = formatRelative(lastSynced);
  const statusesChanged = syncStatus?.statusesChanged ?? 0;

  const getStatusText = () => {
    if (showSuccess) {
      return "Synced successfully";
    }
    if (lastSyncedFormatted !== "—") {
      return `Last synced ${lastSyncedFormatted}`;
    }
    return "Not yet synced";
  };

  return (
    <div className="flex items-center justify-between rounded-xl border border-white/10 bg-white/[0.02] px-4 py-3" data-testid={selectors.requirements.syncBanner}>
      <div className={cn("text-sm", showSuccess ? "text-emerald-400" : "text-slate-400")}>
        <span>{getStatusText()}</span>
        {!showSuccess && statusesChanged > 0 && lastSyncedFormatted !== "—" && (
          <>
            <span className="mx-2 text-slate-600">|</span>
            <span>{statusesChanged} updated</span>
          </>
        )}
      </div>

      <button
        type="button"
        onClick={onSync}
        disabled={isSyncing}
        className={cn(
          "flex items-center gap-2 rounded-lg border px-3 py-1.5 text-sm transition",
          showSuccess
            ? "border-emerald-500/30 bg-emerald-500/10 text-emerald-400"
            : "border-white/10 bg-white/5 hover:bg-white/10",
          "disabled:opacity-50"
        )}
      >
        {showSuccess ? (
          <Check className="h-4 w-4" />
        ) : (
          <RefreshCw className={cn("h-4 w-4", isSyncing && "animate-spin")} />
        )}
        <span>{isSyncing ? "Syncing..." : showSuccess ? "Synced" : "Sync"}</span>
      </button>
    </div>
  );
}
