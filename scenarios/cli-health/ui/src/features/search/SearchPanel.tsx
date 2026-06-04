import { useState } from "react";
import { useMutation } from "@tanstack/react-query";

import { searchClient } from "../../api/clients";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { Mode, type SearchResponse } from "@vrooli/proto-types/cli-health/v1/search/search_pb";

// lowConfidenceThreshold is the display cutoff below which an AI result is
// flagged as a weak match — the human judges the ambiguous band the server-side
// relevance floor intentionally keeps (WS2).
const lowConfidenceThreshold = 0.55;

const modeLabel = (mode: Mode): string => {
  switch (mode) {
    case Mode.AI:
      return "AI";
    case Mode.TEXT:
      return "TEXT";
    default:
      return "UNSPECIFIED";
  }
};

export function SearchPanel() {
  const { t } = useTranslation();
  const [query, setQuery] = useState("");
  const [mode, setMode] = useState<Mode>(Mode.UNSPECIFIED);

  const mutation = useMutation<SearchResponse>({
    mutationFn: async () => searchClient.search({ query, limit: 20, mode }),
  });

  const handleSubmit = (event: React.FormEvent) => {
    event.preventDefault();
    if (!query.trim()) return;
    mutation.mutate();
  };

  const results = mutation.data?.results ?? [];

  return (
    <section
      data-testid={selectors.search.card}
      className="mt-6 rounded-xl border border-white/10 bg-black/20 p-4"
    >
      <h2 className="text-lg font-semibold">{t(strings.search.title)}</h2>
      <p className="mt-1 text-sm text-app-muted-foreground">{t(strings.search.description)}</p>
      <form onSubmit={handleSubmit} className="mt-3 flex flex-col gap-2">
        <Input
          data-testid={selectors.search.input}
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder={t(strings.search.placeholder)}
        />
        <div className="flex items-center gap-2 text-sm">
          <span className="text-app-muted-foreground">{t(strings.search.modeLabel)}:</span>
          <button
            type="button"
            data-testid={selectors.search.modeAi}
            aria-pressed={mode === Mode.AI || mode === Mode.UNSPECIFIED}
            onClick={() => setMode(Mode.UNSPECIFIED)}
            className={
              mode !== Mode.TEXT
                ? "rounded-control bg-app-primary px-3 py-1 text-app-primary-foreground"
                : "rounded-control border border-app-border px-3 py-1"
            }
          >
            {t(strings.search.modeAi)}
          </button>
          <button
            type="button"
            data-testid={selectors.search.modeText}
            aria-pressed={mode === Mode.TEXT}
            onClick={() => setMode(Mode.TEXT)}
            className={
              mode === Mode.TEXT
                ? "rounded-control bg-app-primary px-3 py-1 text-app-primary-foreground"
                : "rounded-control border border-app-border px-3 py-1"
            }
          >
            {t(strings.search.modeText)}
          </button>
          <Button data-testid={selectors.search.submit} type="submit" className="ms-auto">
            {t(strings.search.submit)}
          </Button>
        </div>
      </form>

      {mutation.isPending && (
        <p data-testid={selectors.search.loading} className="mt-3 text-sm text-app-muted-foreground">
          {t(strings.search.loading)}
        </p>
      )}
      {mutation.error && (
        <p data-testid={selectors.search.error} className="mt-3 text-sm text-red-400">
          {t(strings.search.error, { message: mutation.error.message })}
        </p>
      )}
      {mutation.data && (
        <>
          <p
            data-testid={selectors.search.modeUsed}
            className="mt-3 text-xs text-app-muted-foreground"
          >
            {t(strings.search.modeUsed, { mode: modeLabel(mutation.data.modeUsed) })}
          </p>
          {mutation.data.reranker && mutation.data.reranker !== "none" && (
            <p
              data-testid={selectors.search.reranker}
              className="mt-1 text-xs text-app-muted-foreground"
            >
              {t(strings.search.reranker, { reranker: mutation.data.reranker })}
            </p>
          )}
          {results.length === 0 ? (
            <p data-testid={selectors.search.empty} className="mt-2 text-sm">
              {t(strings.search.noResults)}
            </p>
          ) : (
            <ul data-testid={selectors.search.results} className="mt-2 space-y-2">
              {results.map((r, i) => (
                <li
                  key={`${r.origin}/${r.group}/${r.name}/${i}`}
                  data-testid={selectors.search.result}
                  className="rounded-md border border-app-border bg-app-surface-muted p-3"
                >
                  <p className="font-mono text-sm">
                    {r.origin} {r.group} {r.name}
                    {r.score < lowConfidenceThreshold && (
                      <span
                        data-testid={selectors.search.weakMatch}
                        className="ms-2 rounded-control border border-app-border px-1.5 py-0.5 text-xs text-app-muted-foreground align-middle"
                      >
                        {t(strings.search.weakMatch)}
                      </span>
                    )}
                  </p>
                  {r.description && (
                    <p className="mt-1 text-sm text-app-muted-foreground">{r.description}</p>
                  )}
                  <p className="mt-1 text-xs text-app-muted-foreground">
                    {t(strings.search.resultScore, { score: r.score.toFixed(3) })} ·{" "}
                    {t(strings.search.resultSource, { source: r.source })}
                  </p>
                </li>
              ))}
            </ul>
          )}
        </>
      )}
    </section>
  );
}
