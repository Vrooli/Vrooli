import { useQuery } from "@tanstack/react-query";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { generationClient } from "../../api/generation";
import { errorMessage } from "../../lib/errorMessage";

const PROVIDER_STATUS_QUERY_KEY = ["generation", "provider-status"] as const;
const IMAGE_STATUS_QUERY_KEY = ["generation", "image-backend-status"] as const;

/**
 * GenerationCard reports readiness for both generation backends: the text AI
 * provider chain (Ollama → OpenRouter) for facets, and image-tools for brand
 * images (generate / edit / remove_background). It is a read surface: generation
 * itself is a write action driven from the CLI/wizard where a brand id and the
 * operation are supplied. This card lets a user confirm a backend is reachable —
 * and see a clear model/backend readiness hint — before spending a generation.
 */
export function GenerationCard() {
  const { t } = useTranslation();

  const statusQuery = useQuery({
    queryKey: PROVIDER_STATUS_QUERY_KEY,
    queryFn: () => generationClient.getProviderStatus({}),
  });

  const imageStatusQuery = useQuery({
    queryKey: IMAGE_STATUS_QUERY_KEY,
    queryFn: () => generationClient.getImageBackendStatus({}),
  });

  return (
    <section
      data-testid={selectors.generation.card}
      aria-label={t(strings.generation.title)}
      className="mt-4 rounded-xl border border-white/10 bg-black/20 p-4"
    >
      <h2 className="text-sm font-medium text-slate-400">{t(strings.generation.title)}</h2>
      {statusQuery.isLoading && (
        <p data-testid={selectors.generation.loading} className="mt-2 text-slate-200">
          {t(strings.generation.loading)}
        </p>
      )}
      {statusQuery.error && (
        <p data-testid={selectors.generation.error} className="mt-2 text-red-400">
          {errorMessage(statusQuery.error, t)}
        </p>
      )}
      {statusQuery.data && (
        <p data-testid={selectors.generation.summary} className="mt-2 text-slate-200">
          {statusQuery.data.available
            ? t(strings.generation.summaryAvailable)
            : t(strings.generation.summaryUnavailable)}
        </p>
      )}
      {statusQuery.data && statusQuery.data.providers.length === 0 && (
        <p className="mt-2 text-slate-400">{t(strings.generation.empty)}</p>
      )}
      {statusQuery.data && statusQuery.data.providers.length > 0 && (
        <ul data-testid={selectors.generation.list} className="mt-2 space-y-1 text-sm text-slate-200">
          {statusQuery.data.providers.map((provider) => (
            <li key={provider.name} className="flex items-center justify-between rounded-lg border border-white/10 p-3">
              <span data-testid={selectors.generation.providerName} className="font-medium">
                {provider.name}
              </span>
              <span
                data-testid={selectors.generation.providerStatus}
                className={provider.available ? "text-xs text-emerald-400" : "text-xs text-slate-500"}
              >
                {provider.available
                  ? t(strings.generation.availableLabel)
                  : t(strings.generation.unavailableLabel)}
              </span>
            </li>
          ))}
        </ul>
      )}

      <div data-testid={selectors.generation.imageCard} className="mt-4 border-t border-white/10 pt-4">
        <h3 className="text-sm font-medium text-slate-400">{t(strings.generation.imageTitle)}</h3>
        {imageStatusQuery.isLoading && <p className="mt-2 text-slate-200">{t(strings.generation.imageLoading)}</p>}
        {imageStatusQuery.error && (
          <p className="mt-2 text-red-400">{errorMessage(imageStatusQuery.error, t)}</p>
        )}
        {imageStatusQuery.data && (
          <p data-testid={selectors.generation.imageSummary} className="mt-2 text-slate-200">
            {imageStatusQuery.data.available
              ? t(strings.generation.imageSummaryAvailable)
              : imageStatusQuery.data.detail || t(strings.generation.imageSummaryUnavailable)}
          </p>
        )}
        {imageStatusQuery.data && imageStatusQuery.data.operations.length > 0 && (
          <ul data-testid={selectors.generation.imageList} className="mt-2 space-y-1 text-sm text-slate-200">
            {imageStatusQuery.data.operations.map((op) => (
              <li key={op.operation} className="rounded-lg border border-white/10 p-3">
                <div className="flex items-center justify-between">
                  <span data-testid={selectors.generation.imageOpName} className="font-medium">
                    {op.operation}
                    {op.modelId ? <span className="ml-2 text-xs text-slate-500">{op.modelId} · {op.tier}</span> : null}
                  </span>
                  <span
                    data-testid={selectors.generation.imageOpStatus}
                    className={op.ready ? "text-xs text-emerald-400" : "text-xs text-amber-400"}
                  >
                    {op.ready ? t(strings.generation.imageReadyLabel) : t(strings.generation.imageNotReadyLabel)}
                  </span>
                </div>
                {op.hint ? <p className="mt-1 text-xs text-slate-500">{op.hint}</p> : null}
              </li>
            ))}
          </ul>
        )}
      </div>
    </section>
  );
}
