import { useCallback, useEffect, useRef, useState } from "react";
import { useMutation } from "@tanstack/react-query";

import { findingsClient, liveSearchClient } from "../../api/clients";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { errorMessage } from "../../lib/errorMessage";
import { recordLiveSearchHealth } from "../../lib/liveSearchHealth";
import { recordSearch, type SearchMode } from "../../lib/searchHistory";
import { useTranslation } from "../../i18n";
import type { SearchResponse } from "@vrooli/proto-types/web-search/v1/livesearch/livesearch_pb";
import type { SearchFindingsResponse } from "@vrooli/proto-types/web-search/v1/findings/findings_pb";
import { FindingHitList } from "./FindingHitList";
import { SnippetCard } from "./SnippetCard";
import { SynthesisBlock } from "./SynthesisBlock";

const RESULT_LIMIT = 20;

type LiveResult = { kind: "live"; data: SearchResponse };
type LearningsResult = { kind: "learnings"; data: SearchFindingsResponse };
type Result = LiveResult | LearningsResult;

/**
 * Unified search surface. Toggles between a live web search (LiveSearchService,
 * with an optional cited synthesis pass) and the curated learnings corpus
 * (FindingsService.SearchFindings). Both modes share one query box; the
 * executed query is recorded into history and (for live) the response signals
 * are published to the Operations panel.
 */
export interface ReplayRequest {
  query: string;
  mode: SearchMode;
  /** Bumped on every replay click so identical (query, mode) still re-runs. */
  nonce: number;
}

export function SearchPanel({ replay }: { replay?: ReplayRequest | null }) {
  const { t } = useTranslation();
  const [query, setQuery] = useState("");
  const [mode, setMode] = useState<SearchMode>("live");
  const [synthesize, setSynthesize] = useState(false);

  const mutation = useMutation<Result, unknown, { query: string; mode: SearchMode }>({
    mutationFn: async ({ query: q, mode: m }) => {
      if (m === "live") {
        const data = await liveSearchClient.search({
          query: q,
          limit: RESULT_LIMIT,
          synthesize,
        });
        recordLiveSearchHealth({
          query: q,
          cached: data.cached,
          degraded: data.degraded,
          degradedReason: data.degradedReason,
          resultCount: data.results.length,
          at: Date.now(),
        });
        return { kind: "live", data };
      }
      const data = await findingsClient.searchFindings({
        query: q,
        limit: RESULT_LIMIT,
        includeArchived: false,
      });
      return { kind: "learnings", data };
    },
    onSuccess: (_data, vars) => {
      recordSearch(vars.query, vars.mode);
    },
  });

  const { mutate } = mutation;
  const runSearch = useCallback(
    (q: string, m: SearchMode) => {
      const trimmed = q.trim();
      if (!trimmed) return;
      mutate({ query: trimmed, mode: m });
    },
    [mutate],
  );

  const handleSubmit = (event: React.FormEvent) => {
    event.preventDefault();
    runSearch(query, mode);
  };

  // Replay a query selected from the history panel: sync the controls, then
  // run it. A ref guards against re-running on unrelated renders — we only act
  // when a *new* replay (higher nonce) arrives, so the effect can legitimately
  // depend on the whole `replay`/`runSearch` closure.
  const lastReplayNonce = useRef<number | undefined>(undefined);
  useEffect(() => {
    if (!replay || replay.nonce === lastReplayNonce.current) return;
    lastReplayNonce.current = replay.nonce;
    setQuery(replay.query);
    setMode(replay.mode);
    runSearch(replay.query, replay.mode);
  }, [replay, runSearch]);

  const result = mutation.data;

  return (
    <section className="flex flex-col gap-4">
      <form
        data-testid={selectors.search.form}
        onSubmit={handleSubmit}
        className="flex flex-col gap-3"
      >
        <div className="flex gap-2">
          <Input
            data-testid={selectors.search.input}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder={t(strings.search.placeholder)}
            aria-label={t(strings.search.placeholder)}
          />
          <Button data-testid={selectors.search.submit} type="submit">
            {t(strings.search.submit)}
          </Button>
        </div>

        <div className="flex flex-wrap items-center gap-3 text-sm">
          <span className="text-app-muted-foreground">{t(strings.search.modeLabel)}:</span>
          <div role="radiogroup" aria-label={t(strings.search.modeLabel)} className="flex gap-2">
            <button
              type="button"
              role="radio"
              aria-checked={mode === "live"}
              data-testid={selectors.search.modeLive}
              onClick={() => setMode("live")}
              className={
                mode === "live"
                  ? "rounded-control bg-app-primary px-3 py-1 text-app-primary-foreground"
                  : "rounded-control border border-app-border px-3 py-1 text-app-foreground hover:bg-app-surface-muted"
              }
            >
              {t(strings.search.modeLive)}
            </button>
            <button
              type="button"
              role="radio"
              aria-checked={mode === "learnings"}
              data-testid={selectors.search.modeLearnings}
              onClick={() => setMode("learnings")}
              className={
                mode === "learnings"
                  ? "rounded-control bg-app-primary px-3 py-1 text-app-primary-foreground"
                  : "rounded-control border border-app-border px-3 py-1 text-app-foreground hover:bg-app-surface-muted"
              }
            >
              {t(strings.search.modeLearnings)}
            </button>
          </div>

          {mode === "live" && (
            <label className="ms-auto flex items-center gap-2 text-app-foreground">
              <input
                type="checkbox"
                data-testid={selectors.search.synthesizeToggle}
                checked={synthesize}
                onChange={(e) => setSynthesize(e.target.checked)}
                className="h-4 w-4 accent-app-primary"
              />
              {t(strings.search.synthesizeLabel)}
            </label>
          )}
        </div>
      </form>

      {mutation.isPending && (
        <p data-testid={selectors.search.loading} className="text-sm text-app-muted-foreground">
          {t(strings.search.loading)}
        </p>
      )}
      {mutation.error != null && (
        <p data-testid={selectors.search.error} className="text-sm text-app-danger">
          {t(strings.search.error, { message: errorMessage(mutation.error, t) })}
        </p>
      )}

      {result?.kind === "live" && <LiveResults response={result.data} />}
      {result?.kind === "learnings" && <LearningsResults response={result.data} />}
    </section>
  );
}

function LiveResults({ response }: { response: SearchResponse }) {
  const { t } = useTranslation();

  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center gap-2 text-xs text-app-muted-foreground">
        <span data-testid={selectors.search.cached}>
          {response.cached ? t(strings.search.cached) : t(strings.search.fresh)}
        </span>
      </div>
      {response.degraded && (
        <p data-testid={selectors.search.degraded} className="text-sm text-app-warning">
          {t(strings.search.degraded, {
            reason: response.degradedReason || t(strings.errors.unavailable),
          })}
        </p>
      )}
      {response.synthesis && <SynthesisBlock synthesis={response.synthesis} />}
      {response.results.length === 0 ? (
        <p data-testid={selectors.search.empty} className="text-sm text-app-muted-foreground">
          {t(strings.search.empty)}
        </p>
      ) : (
        <ul data-testid={selectors.search.results} className="space-y-2">
          {response.results.map((r, i) => (
            <SnippetCard key={`${r.url}/${i}`} result={r} />
          ))}
        </ul>
      )}
    </div>
  );
}

function LearningsResults({ response }: { response: SearchFindingsResponse }) {
  const { t } = useTranslation();

  return (
    <div className="flex flex-col gap-3">
      {response.method && (
        <p data-testid={selectors.search.method} className="text-xs text-app-muted-foreground">
          {t(strings.search.method, { method: response.method })}
        </p>
      )}
      {response.hits.length === 0 ? (
        <p data-testid={selectors.search.empty} className="text-sm text-app-muted-foreground">
          {t(strings.search.emptyLearnings)}
        </p>
      ) : (
        <FindingHitList hits={response.hits} />
      )}
    </div>
  );
}
