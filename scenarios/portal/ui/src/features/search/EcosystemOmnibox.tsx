import { useEffect, useMemo, useState } from "react";

import { suggest, type SuggestResponse } from "../../api/search";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";

const debounceMs = 250;
const minQueryLength = 2;

export function EcosystemOmnibox() {
  const { t } = useTranslation();
  const [draft, setDraft] = useState("");
  const [response, setResponse] = useState<SuggestResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const query = draft.trim();
  const hits = response?.hits ?? [];

  useEffect(() => {
    if (query.length < minQueryLength) {
      setResponse(null);
      setError("");
      setLoading(false);
      return;
    }
    let canceled = false;
    setLoading(true);
    const timer = window.setTimeout(() => {
      void suggest({ query, limit: 5 })
        .then((next) => {
          if (!canceled) {
            setResponse(next);
            setError("");
          }
        })
        .catch((err: unknown) => {
          if (!canceled) {
            setResponse(null);
            setError(errorMessage(err, t));
          }
        })
        .finally(() => {
          if (!canceled) {
            setLoading(false);
          }
        });
    }, debounceMs);
    return () => {
      canceled = true;
      window.clearTimeout(timer);
    };
  }, [query, t]);

  const statusText = useMemo(() => {
    if (loading) {
      return t(strings.search.omnibox.loading);
    }
    if (error) {
      return error;
    }
    if (response?.degraded) {
      return response.reason || t(strings.search.omnibox.degraded);
    }
    if (query.length >= minQueryLength && hits.length === 0) {
      return t(strings.search.omnibox.empty);
    }
    return "";
  }, [error, hits.length, loading, query.length, response, t]);

  return (
    <section
      data-testid={selectors.search.omnibox}
      aria-labelledby="ecosystem-omnibox-heading"
      className="flex flex-col gap-3 rounded-panel border border-app-border bg-app-surface p-4"
    >
      <div className="flex flex-col gap-1">
        <h3 id="ecosystem-omnibox-heading" className="text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.search.omnibox.title)}
        </h3>
        <p className="text-sm text-app-muted-foreground">{t(strings.search.omnibox.description)}</p>
      </div>
      <label className="flex flex-col gap-2 text-sm font-medium">
        <span>{t(strings.search.omnibox.inputLabel)}</span>
        <textarea
          data-testid={selectors.search.input}
          value={draft}
          onChange={(event) => setDraft(event.target.value)}
          rows={3}
          className="min-h-24 rounded-control border border-app-border bg-app-background px-3 py-2 text-app-foreground outline-none focus-visible:ring-2 focus-visible:ring-app-primary"
          placeholder={t(strings.search.omnibox.placeholder)}
        />
      </label>
      {statusText ? (
        <p data-testid={selectors.search.status} className="text-sm text-app-muted-foreground">
          {statusText}
        </p>
      ) : null}
      {hits.length > 0 ? (
        <div data-testid={selectors.search.results} className="flex flex-col gap-2">
          {hits.map((hit, index) => (
            <button
              key={`${hit.providerId}-${hit.path}-${index}`}
              type="button"
              data-testid={selectors.search.result({ index })}
              onClick={() => setDraft((current) => `${current.trim()} ${hit.path || hit.title}`.trim())}
              className="rounded-control border border-app-border bg-app-background px-3 py-2 text-left hover:bg-app-surface-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary"
            >
              <span className="flex flex-wrap items-center gap-2">
                <span className="rounded-control bg-app-primary/10 px-2 py-0.5 text-xs font-medium text-app-primary">
                  {hit.type || hit.providerId}
                </span>
                <span className="font-medium text-app-foreground">{hit.title || hit.path}</span>
              </span>
              {hit.snippet ? (
                <span className="mt-1 block text-sm text-app-muted-foreground">{hit.snippet}</span>
              ) : null}
              {hit.path ? <span className="mt-1 block text-xs text-app-muted-foreground">{hit.path}</span> : null}
            </button>
          ))}
        </div>
      ) : null}
    </section>
  );
}
