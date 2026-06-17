import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { modelsClient, type Model, type OpDefault } from "../../api/models";
import { errorMessage } from "../../lib/errorMessage";

const DEFAULTS_QUERY_KEY = ["models", "defaults"] as const;
const MODELS_QUERY_KEY = ["models"] as const;

/**
 * ModelDefaultsCard is the per-operation default-model settings surface. It
 * lists every operation with its effective default model and source
 * (ListDefaults), lets the operator pin a different model (SetDefaultModel) or
 * clear the pin back to the built-in default (SetDefaultModel with an empty
 * model id). The per-op model menu is filtered to models that serve that op.
 */
export function ModelDefaultsCard() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();

  const defaultsQuery = useQuery({
    queryKey: DEFAULTS_QUERY_KEY,
    queryFn: () => modelsClient.listDefaults({}),
  });

  const modelsQuery = useQuery({
    queryKey: MODELS_QUERY_KEY,
    queryFn: () => modelsClient.listModels({}),
  });

  const setDefaultMutation = useMutation({
    mutationFn: ({ operation, modelId }: { operation: string; modelId: string }) =>
      modelsClient.setDefaultModel({ operation, modelId }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: DEFAULTS_QUERY_KEY });
    },
  });

  const defaults: OpDefault[] = defaultsQuery.data?.defaults ?? [];
  const models: Model[] = modelsQuery.data?.models ?? [];

  const modelsForOperation = (operation: string): Model[] =>
    models.filter((m) => m.operations.includes(operation));

  return (
    <section
      data-testid={selectors.models.defaults.card}
      aria-label={t(strings.models.defaults.title)}
      className="rounded-xl border border-white/10 bg-black/20 p-4"
    >
      <h2 className="text-sm font-medium text-slate-400">{t(strings.models.defaults.title)}</h2>
      {defaultsQuery.isLoading && (
        <p data-testid={selectors.models.defaults.loading} className="mt-2 text-slate-200">
          {t(strings.models.defaults.loading)}
        </p>
      )}
      {defaultsQuery.error && (
        <p data-testid={selectors.models.defaults.error} className="mt-2 text-red-400">
          {errorMessage(defaultsQuery.error, t)}
        </p>
      )}
      {defaultsQuery.data && defaults.length === 0 && (
        <p data-testid={selectors.models.defaults.empty} className="mt-2 text-slate-200">
          {t(strings.models.defaults.empty)}
        </p>
      )}
      {defaults.length > 0 && (
        <ul
          data-testid={selectors.models.defaults.list}
          className="mt-2 space-y-2 text-sm text-slate-200"
        >
          {defaults.map((d) => {
            const isOverride = d.source === "override";
            const options = modelsForOperation(d.operation);
            const selectId = `default-${d.operation}`;
            return (
              <li
                key={d.operation}
                data-testid={selectors.models.defaults.row}
                className="flex flex-wrap items-end gap-3 rounded-lg border border-white/10 p-3"
              >
                <div data-testid={selectors.models.defaults.operation} className="font-medium">
                  {d.operation}
                </div>
                <div className="flex flex-col gap-1">
                  <label htmlFor={selectId} className="text-xs text-slate-400">
                    {t(strings.models.defaults.modelLabel)}
                  </label>
                  <select
                    id={selectId}
                    data-testid={selectors.models.defaults.select}
                    value={isOverride ? d.modelId : ""}
                    onChange={(e) =>
                      setDefaultMutation.mutate({ operation: d.operation, modelId: e.target.value })
                    }
                    disabled={setDefaultMutation.isPending}
                    className="h-10 rounded-md border border-white/20 bg-white/5 px-3 text-sm text-white focus:outline-none focus:ring-2 focus:ring-white/40"
                  >
                    <option value="" className="bg-slate-900">
                      {t(strings.models.defaults.useDefault)}
                    </option>
                    {options.map((m) => (
                      <option key={m.id} value={m.id} className="bg-slate-900">
                        {m.name}
                      </option>
                    ))}
                  </select>
                </div>
                <div data-testid={selectors.models.defaults.source} className="text-xs text-slate-400">
                  {t(strings.models.defaults.sourceLabel)}{" "}
                  {isOverride
                    ? t(strings.models.defaults.source.override)
                    : t(strings.models.defaults.source.seed)}
                  {" · "}
                  {d.modelId || t(strings.models.defaults.none)}
                </div>
                {isOverride && (
                  <Button
                    data-testid={selectors.models.defaults.clearButton}
                    onClick={() =>
                      setDefaultMutation.mutate({ operation: d.operation, modelId: "" })
                    }
                    disabled={setDefaultMutation.isPending}
                  >
                    {t(strings.models.defaults.clear)}
                  </Button>
                )}
              </li>
            );
          })}
        </ul>
      )}
      {setDefaultMutation.error && (
        <p data-testid={selectors.models.defaults.error} className="mt-2 text-red-400">
          {errorMessage(setDefaultMutation.error, t)}
        </p>
      )}
    </section>
  );
}
