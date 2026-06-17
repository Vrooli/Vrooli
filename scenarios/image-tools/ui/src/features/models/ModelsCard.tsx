import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { modelsClient, type Model } from "../../api/models";
import { errorMessage } from "../../lib/errorMessage";

const MODELS_QUERY_KEY = ["models"] as const;

/**
 * ModelsCard is the declarative model-registry surface: it lists each
 * registered model (id, name, tier, backend, enabled) and lets the
 * operator toggle a model on/off via SetModelEnabled. Loading / empty /
 * error states mirror the other read-oriented cards.
 */
export function ModelsCard() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();

  const modelsQuery = useQuery({
    queryKey: MODELS_QUERY_KEY,
    queryFn: () => modelsClient.listModels({}),
  });

  const setEnabledMutation = useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) =>
      modelsClient.setModelEnabled({ id, enabled }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: MODELS_QUERY_KEY });
    },
  });

  const models: Model[] = modelsQuery.data?.models ?? [];

  return (
    <section
      data-testid={selectors.models.card}
      aria-label={t(strings.models.title)}
      className="mt-4 rounded-xl border border-white/10 bg-black/20 p-4"
    >
      <h2 className="text-sm font-medium text-slate-400">{t(strings.models.title)}</h2>
      {modelsQuery.isLoading && (
        <p data-testid={selectors.models.loading} className="mt-2 text-slate-200">
          {t(strings.models.loading)}
        </p>
      )}
      {modelsQuery.error && (
        <p data-testid={selectors.models.error} className="mt-2 text-red-400">
          {errorMessage(modelsQuery.error, t)}
        </p>
      )}
      {modelsQuery.data && models.length === 0 && (
        <p data-testid={selectors.models.empty} className="mt-2 text-slate-200">
          {t(strings.models.empty)}
        </p>
      )}
      {models.length > 0 && (
        <ul data-testid={selectors.models.list} className="mt-2 space-y-1 text-sm text-slate-200">
          {models.map((model) => (
            <li key={model.id} className="rounded-lg border border-white/10 p-3">
              <div data-testid={selectors.models.name} className="font-medium">
                {model.name}
              </div>
              <div className="mt-1 text-xs text-slate-500">{model.id}</div>
              <div data-testid={selectors.models.tier} className="mt-1 text-xs text-slate-400">
                {t(strings.models.tierLabel)} {model.tier}
              </div>
              <div data-testid={selectors.models.backend} className="mt-1 text-xs text-slate-400">
                {t(strings.models.backendLabel)} {model.backend}
              </div>
              <div data-testid={selectors.models.enabledState} className="mt-1 text-xs text-slate-400">
                {model.enabled ? t(strings.models.enabledLabel) : t(strings.models.disabledLabel)}
              </div>
              <Button
                data-testid={selectors.models.toggleButton}
                className="mt-2"
                aria-pressed={model.enabled}
                onClick={() =>
                  setEnabledMutation.mutate({ id: model.id, enabled: !model.enabled })
                }
                disabled={setEnabledMutation.isPending}
              >
                {model.enabled ? t(strings.models.disable) : t(strings.models.enable)}
              </Button>
            </li>
          ))}
        </ul>
      )}
      {setEnabledMutation.error && (
        <p data-testid={selectors.models.error} className="mt-2 text-red-400">
          {errorMessage(setEnabledMutation.error, t)}
        </p>
      )}
    </section>
  );
}
