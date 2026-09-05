/**
 * SupersedeChainView — renders the linear supersede chain rooted at a record.
 *
 * Caps render at MAX_HOPS as a defensive measure against pathological chains.
 */

import { useEffect, useState } from "react";
import { recordsService } from "../../services/records-service";
import type { RecordItem } from "../../types";
import { RecordCard } from "./RecordCard";

interface SupersedeChainViewProps {
  rootId: string;
}

const MAX_HOPS = 25;

export function SupersedeChainView({ rootId }: SupersedeChainViewProps) {
  const [chain, setChain] = useState<RecordItem[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [capped, setCapped] = useState(false);

  useEffect(() => {
    let cancelled = false;
    setError(null);
    setCapped(false);
    (async () => {
      try {
        const collected: RecordItem[] = [];
        const seen = new Set<string>();
        let cursor: string | undefined = rootId;
        let hops = 0;
        while (cursor && !seen.has(cursor) && hops < MAX_HOPS) {
          seen.add(cursor);
          const rec = await recordsService.get(cursor);
          collected.push(rec);
          cursor = rec.supersededBy;
          hops += 1;
        }
        if (hops >= MAX_HOPS && cursor) {
          if (!cancelled) setCapped(true);
        }
        if (!cancelled) setChain(collected);
      } catch (err) {
        if (!cancelled) setError(err instanceof Error ? err.message : String(err));
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [rootId]);

  if (error) {
    return (
      <div className="rounded border border-red-700 bg-red-950/40 p-2 text-sm text-red-200">
        Failed to load chain: {error}
      </div>
    );
  }

  if (chain.length === 0) {
    return <div className="text-sm text-slate-400">Loading chain…</div>;
  }

  return (
    <div className="flex flex-col gap-2" data-testid="supersede-chain">
      {chain.map((r, idx) => (
        <div key={r.id} className="flex items-start gap-2">
          <span className="mt-3 w-5 text-center text-xs text-slate-500">{idx + 1}.</span>
          <div className="flex-1">
            <RecordCard record={r} highlight={r.id === rootId} />
          </div>
        </div>
      ))}
      {capped ? (
        <div className="rounded border border-amber-700 bg-amber-950/40 p-2 text-xs text-amber-200">
          Chain truncated at {MAX_HOPS} hops (possible cycle).
        </div>
      ) : null}
    </div>
  );
}
