/**
 * AISearchResults - Sidebar panel shown when the search-mode toggle is in
 * "AI" mode. Calls POST /search/ai with the current debounced query and
 * renders scored results across backlog items and goals.
 *
 * When AI search is unavailable (Ollama/Qdrant down), this component is not
 * rendered; the parent hides the AI mode toggle via useAISearchStatus.
 */

import { useEffect, useState } from "react";
import { Sparkles } from "lucide-react";
import {
  searchAI,
  type AISearchResponse,
  type AISearchResult,
} from "../../../../lib/ai-search";
import { isApiError } from "../../../../lib/api-client";
import { buildBacklogNodeId } from "../../lib/node-id-parser";

interface AISearchResultsProps {
  query: string;
  onItemClick: (nodeId: string) => void;
}

export function AISearchResults({ query, onItemClick }: AISearchResultsProps) {
  const [resp, setResp] = useState<AISearchResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const trimmed = query.trim();
    if (trimmed === "") {
      setResp(null);
      setError(null);
      return;
    }
    let cancelled = false;
    setLoading(true);
    searchAI({ query: trimmed, entity: "both", limit: 20 })
      .then((r) => {
        if (cancelled) return;
        setResp(r);
        setError(null);
      })
      .catch((err) => {
        if (cancelled) return;
        setError(isApiError(err) ? err.message : String(err));
        setResp(null);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [query]);

  if (query.trim() === "") {
    return (
      <div className="flex flex-col items-center justify-center gap-2 py-8 text-sm text-slate-400">
        <Sparkles className="h-5 w-5" aria-hidden />
        <div>Type to search semantically across backlog items and goals.</div>
      </div>
    );
  }

  if (loading && !resp) {
    return <div className="py-4 text-center text-sm text-slate-400" data-testid="ai-search-loading">Searching…</div>;
  }

  if (error) {
    return (
      <div className="rounded border border-red-500/40 bg-red-900/20 p-3 text-sm text-red-300" data-testid="ai-search-error">
        AI search failed: {error}
      </div>
    );
  }

  if (!resp) {
    return null;
  }

  return (
    <div className="flex flex-col gap-2" data-testid="ai-search-results">
      <div className="flex items-center justify-between text-xs text-slate-400">
        <span>
          {resp.total} result{resp.total === 1 ? "" : "s"}
        </span>
        <span>
          {resp.fallback === "text-search" && "(text-search fallback)"}
          {resp.fallback === "unavailable" && "(AI search unavailable)"}
          {resp.fallback === "none" && `${resp.latencyMs}ms`}
        </span>
      </div>

      {resp.results.length === 0 ? (
        <div className="py-4 text-center text-sm text-slate-400">No matches.</div>
      ) : (
        <ul className="flex flex-col gap-1.5">
          {resp.results.map((r) => (
            <AISearchResultItem key={`${r.entity}:${r.id}`} result={r} onItemClick={onItemClick} />
          ))}
        </ul>
      )}
    </div>
  );
}

interface AISearchResultItemProps {
  result: AISearchResult;
  onItemClick: (nodeId: string) => void;
}

function AISearchResultItem({ result, onItemClick }: AISearchResultItemProps) {
  const title = (result.payload.title as string | undefined) ?? result.id;
  const status = result.payload.status as string | undefined;
  const kind = result.payload.kind as string | undefined;

  const nodeId =
    result.entity === "backlog" && kind
      ? buildBacklogNodeId(kind, result.id)
      : `goal:${result.id}`;

  return (
    <li>
      <button
        type="button"
        onClick={() => onItemClick(nodeId)}
        className="flex w-full flex-col gap-0.5 rounded border border-slate-700/50 bg-slate-900/40 px-2 py-1.5 text-left text-sm hover:bg-slate-800/60"
        data-testid="ai-search-result"
      >
        <div className="flex items-center gap-2">
          <span className="rounded bg-sky-500/20 px-1.5 py-0.5 text-[10px] font-medium uppercase text-sky-300">
            {result.scorePercent}%
          </span>
          <span className="rounded bg-slate-700/50 px-1.5 py-0.5 text-[10px] uppercase text-slate-300">
            {result.entity}
          </span>
          {kind && (
            <span className="rounded bg-slate-700/50 px-1.5 py-0.5 text-[10px] uppercase text-slate-300">
              {kind}
            </span>
          )}
          {status && <span className="text-xs text-slate-400">{status}</span>}
        </div>
        <div className="truncate text-slate-100">{title}</div>
      </button>
    </li>
  );
}
