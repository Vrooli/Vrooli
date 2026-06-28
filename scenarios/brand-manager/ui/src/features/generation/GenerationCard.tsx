import { useQuery } from "@tanstack/react-query";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { generationClient } from "../../api/generation";
import { errorMessage } from "../../lib/errorMessage";

const PROVIDER_STATUS_QUERY_KEY = ["generation", "provider-status"] as const;

/**
 * GenerationCard reports AI provider availability (Ollama → OpenRouter). It is a
 * read surface: generation itself (text facets, images) is a write action driven
 * from the CLI/wizard where a brand id and element selection are supplied. This
 * card lets a user confirm a provider is reachable before spending a generation.
 * Mirrors the canonical AssetsCard structure but wired to GenerationService.
 */
export function GenerationCard() {
  const { t } = useTranslation();

  const statusQuery = useQuery({
    queryKey: PROVIDER_STATUS_QUERY_KEY,
    queryFn: () => generationClient.getProviderStatus({}),
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
    </section>
  );
}
