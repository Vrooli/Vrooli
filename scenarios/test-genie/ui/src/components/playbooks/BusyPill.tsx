import { useMemo, useState } from "react";
import { Loader2, AlertCircle } from "lucide-react";
import type { PlaybooksClaim } from "../../lib/api";
import { releasePlaybooksClaim } from "../../lib/api";
import { Button } from "../ui/button";
import { cn } from "../../lib/utils";

interface BusyPillProps {
  claim: PlaybooksClaim;
  onReleased?: () => void;
}

function ageSeconds(iso: string): number {
  const t = new Date(iso).getTime();
  if (Number.isNaN(t)) return 0;
  return Math.max(0, Math.round((Date.now() - t) / 1000));
}

function formatAge(secs: number): string {
  if (secs < 60) return `${secs}s ago`;
  if (secs < 3600) return `${Math.floor(secs / 60)}m ago`;
  return `${Math.floor(secs / 3600)}h ago`;
}

export function BusyPill({ claim, onReleased }: BusyPillProps) {
  const [releasing, setReleasing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const heartbeatAgeS = useMemo(() => ageSeconds(claim.heartbeat_at), [claim.heartbeat_at]);
  const stale = !claim.alive;

  const handleRelease = async () => {
    if (!window.confirm(`Force-release run ${claim.run_id} on ${claim.scenario_name}?`)) return;
    setReleasing(true);
    setError(null);
    try {
      await releasePlaybooksClaim(claim.scenario_name);
      onReleased?.();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setReleasing(false);
    }
  };

  return (
    <div
      className={cn(
        "flex items-center justify-between gap-3 rounded-xl border px-4 py-3 text-sm",
        stale
          ? "border-amber-400/40 bg-amber-400/10 text-amber-100"
          : "border-sky-400/40 bg-sky-400/10 text-sky-100"
      )}
      data-testid="playbooks-busy-pill"
    >
      <div className="flex items-center gap-3">
        {stale ? (
          <AlertCircle className="h-4 w-4" />
        ) : (
          <Loader2 className="h-4 w-4 animate-spin" />
        )}
        <div>
          <p className="font-medium">
            {stale ? "Stale playbooks claim" : "Playbooks running"}
          </p>
          <p className="text-xs opacity-80">
            run {claim.run_id} · mode {claim.mode} · started by {claim.started_by} · heartbeat {formatAge(heartbeatAgeS)}
          </p>
          {error && <p className="mt-1 text-xs text-red-300">{error}</p>}
        </div>
      </div>
      <Button
        type="button"
        variant="outline"
        size="sm"
        disabled={releasing}
        onClick={handleRelease}
        data-testid="playbooks-busy-pill-release"
      >
        {releasing ? "Releasing…" : "Force release"}
      </Button>
    </div>
  );
}
