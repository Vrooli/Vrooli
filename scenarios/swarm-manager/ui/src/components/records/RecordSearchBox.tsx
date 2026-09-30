/**
 * RecordSearchBox — semantic search input + result render.
 */

import { useCallback, useState } from "react";
import { recordsService } from "../../services/records-service";
import type { RecordSearchHit } from "../../types";
import { RecordCard } from "./RecordCard";

interface RecordSearchBoxProps {
  scenario?: string;
}

export function RecordSearchBox({ scenario }: RecordSearchBoxProps) {
  const [query, setQuery] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [hits, setHits] = useState<RecordSearchHit[]>([]);

  const submit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      const q = query.trim();
      if (!q) return;
      setBusy(true);
      setError(null);
      try {
        const results = await recordsService.search(q, { scenario, limit: 10 });
        setHits(results);
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
      } finally {
        setBusy(false);
      }
    },
    [query, scenario],
  );

  return (
    <div className="flex flex-col gap-3">
      <form onSubmit={submit} className="flex items-center gap-2">
        <input
          type="text"
          className="flex-1 rounded border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-100 placeholder-slate-500"
          placeholder="Search records semantically..."
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          data-testid="record-search-input"
        />
        <button
          type="submit"
          disabled={busy || !query.trim()}
          className="rounded bg-emerald-600 px-3 py-2 text-sm font-medium text-white disabled:opacity-50"
          data-testid="record-search-submit"
        >
          {busy ? "Searching…" : "Search"}
        </button>
      </form>

      {error ? (
        <div className="rounded border border-red-700 bg-red-950/40 p-2 text-sm text-red-200">{error}</div>
      ) : null}

      {hits.length > 0 ? (
        <ul className="flex flex-col gap-2" data-testid="record-search-results">
          {hits.map((h) => (
            <li key={h.record.id}>
              <RecordCard record={h.record} score={h.score} />
            </li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}
