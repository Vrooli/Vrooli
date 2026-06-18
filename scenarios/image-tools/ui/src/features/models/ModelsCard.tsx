import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { CommercialUse, modelsClient, type Model } from "../../api/models";
import { errorMessage } from "../../lib/errorMessage";
import { hardwareFitChips } from "./hardwareFit";

const MODELS_QUERY_KEY = ["models"] as const;

/** Map each CommercialUse enum value to its label key (every leaf referenced). */
const COMMERCIAL_LABEL: Record<CommercialUse, (typeof strings.models.commercial)[keyof typeof strings.models.commercial]> = {
  [CommercialUse.UNSPECIFIED]: strings.models.commercial.unspecified,
  [CommercialUse.YES]: strings.models.commercial.yes,
  [CommercialUse.NO]: strings.models.commercial.no,
  [CommercialUse.CONDITIONAL]: strings.models.commercial.conditional,
};

/** Per-install feedback surfaced inline on the model that launched it. */
interface InstallNotice {
  id: string;
  jobId: string;
  eta: number;
  alreadyInstalled: boolean;
}

/**
 * ModelsCard is the model-management surface: it lists each registered model
 * (id, name, tier, backend, size, hardware-fit, license/capability labels,
 * install state) and lets the operator install on-opt-in (InstallModel →
 * durable job), enable/disable (SetModelEnabled), and remove (RemoveModel).
 * Custom entries carry a [custom] badge. Loading / empty / error states
 * mirror the other read-oriented cards.
 */
export function ModelsCard() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [installNotice, setInstallNotice] = useState<InstallNotice | null>(null);

  const modelsQuery = useQuery({
    queryKey: MODELS_QUERY_KEY,
    queryFn: () => modelsClient.listModels({}),
  });

  const invalidate = () => void queryClient.invalidateQueries({ queryKey: MODELS_QUERY_KEY });

  const setEnabledMutation = useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) =>
      modelsClient.setModelEnabled({ id, enabled }),
    onSuccess: invalidate,
  });

  const installMutation = useMutation({
    mutationFn: (id: string) => modelsClient.installModel({ id }),
    onSuccess: (res, id) => {
      setInstallNotice({
        id,
        jobId: res.jobId,
        eta: res.etaSeconds,
        alreadyInstalled: res.alreadyInstalled,
      });
      invalidate();
    },
  });

  const removeMutation = useMutation({
    mutationFn: (id: string) => modelsClient.removeModel({ id }),
    onSuccess: invalidate,
  });

  const models: Model[] = modelsQuery.data?.models ?? [];
  const mutationError =
    setEnabledMutation.error ?? installMutation.error ?? removeMutation.error;

  return (
    <section
      data-testid={selectors.models.card}
      aria-label={t(strings.models.title)}
      className="mt-4 rounded-xl border border-app-border bg-app-surface p-4"
    >
      <h2 className="text-sm font-medium text-app-muted-foreground">{t(strings.models.title)}</h2>
      {modelsQuery.isLoading && (
        <p data-testid={selectors.models.loading} className="mt-2 text-app-foreground">
          {t(strings.models.loading)}
        </p>
      )}
      {modelsQuery.error && (
        <p data-testid={selectors.models.error} className="mt-2 text-app-danger">
          {errorMessage(modelsQuery.error, t)}
        </p>
      )}
      {modelsQuery.data && models.length === 0 && (
        <p data-testid={selectors.models.empty} className="mt-2 text-app-foreground">
          {t(strings.models.empty)}
        </p>
      )}
      {models.length > 0 && (
        <ul data-testid={selectors.models.list} className="mt-2 space-y-2 text-sm text-app-foreground">
          {models.map((model) => {
            const installed = model.install?.installed ?? false;
            const labels = model.capabilityLabels;
            return (
              <li key={model.id} className="rounded-lg border border-app-border p-3">
                <div className="flex flex-wrap items-center gap-2">
                  <span data-testid={selectors.models.name} className="font-medium">
                    {model.name}
                  </span>
                  {model.custom && (
                    <span
                      data-testid={selectors.models.customBadge}
                      className="rounded bg-app-primary/20 px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-app-primary"
                    >
                      {t(strings.models.customBadge)}
                    </span>
                  )}
                  {labels?.nsfwCapable && (
                    <span
                      data-testid={selectors.models.nsfwBadge}
                      className="rounded bg-app-warning/20 px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-app-warning"
                    >
                      {t(strings.models.nsfwBadge)}
                    </span>
                  )}
                </div>
                <div className="mt-1 text-xs text-app-muted-foreground">{model.id}</div>

                <dl className="mt-2 grid grid-cols-2 gap-x-4 gap-y-1 text-xs text-app-muted-foreground">
                  <div data-testid={selectors.models.tier}>
                    <dt className="inline text-app-muted-foreground">{t(strings.models.tierLabel)} </dt>
                    <dd className="inline">{model.tier}</dd>
                  </div>
                  <div data-testid={selectors.models.backend}>
                    <dt className="inline text-app-muted-foreground">{t(strings.models.backendLabel)} </dt>
                    <dd className="inline">{model.backend}</dd>
                  </div>
                  <div data-testid={selectors.models.operations}>
                    <dt className="inline text-app-muted-foreground">{t(strings.models.operationsLabel)} </dt>
                    <dd className="inline">{model.operations.join(", ")}</dd>
                  </div>
                  <div data-testid={selectors.models.size}>
                    <dt className="inline text-app-muted-foreground">{t(strings.models.sizeLabel)} </dt>
                    <dd className="inline">{t(strings.models.sizeValue, { mb: model.sizeMbApprox })}</dd>
                  </div>
                  {labels && (
                    <div data-testid={selectors.models.license}>
                      <dt className="inline text-app-muted-foreground">{t(strings.models.licenseLabel)} </dt>
                      <dd className="inline">{labels.license || "—"}</dd>
                    </div>
                  )}
                  {labels && (
                    <div data-testid={selectors.models.commercial}>
                      <dt className="inline text-app-muted-foreground">{t(strings.models.commercialLabel)} </dt>
                      <dd className="inline">{t(COMMERCIAL_LABEL[labels.commercialUse])}</dd>
                    </div>
                  )}
                </dl>

                {model.hardware && (
                  <div
                    data-testid={selectors.models.hardware}
                    className="mt-2 flex flex-wrap gap-1.5"
                    aria-label={t(strings.models.hardwareLabel)}
                  >
                    {hardwareFitChips(model.hardware).map((chip) => (
                      <span
                        key={chip.key}
                        className="rounded border border-app-border bg-app-surface-muted px-1.5 py-0.5 text-[10px] text-app-foreground"
                      >
                        {t(chip.key, chip.values)}
                      </span>
                    ))}
                  </div>
                )}

                <div
                  data-testid={selectors.models.installState}
                  className="mt-2 text-xs text-app-muted-foreground"
                >
                  {installed
                    ? t(strings.models.installState.installed)
                    : t(strings.models.installState.notInstalled)}
                </div>

                <div className="mt-2 flex flex-wrap gap-2">
                  {!installed && (
                    <Button
                      data-testid={selectors.models.installButton}
                      onClick={() => installMutation.mutate(model.id)}
                      disabled={installMutation.isPending}
                    >
                      {installMutation.isPending && installMutation.variables === model.id
                        ? t(strings.models.installing)
                        : t(strings.models.install)}
                    </Button>
                  )}
                  <Button
                    data-testid={selectors.models.toggleButton}
                    aria-pressed={model.enabled}
                    onClick={() =>
                      setEnabledMutation.mutate({ id: model.id, enabled: !model.enabled })
                    }
                    disabled={setEnabledMutation.isPending}
                  >
                    {model.enabled ? t(strings.models.disable) : t(strings.models.enable)}
                  </Button>
                  <Button
                    data-testid={selectors.models.removeButton}
                    onClick={() => removeMutation.mutate(model.id)}
                    disabled={removeMutation.isPending}
                  >
                    {removeMutation.isPending && removeMutation.variables === model.id
                      ? t(strings.models.removing)
                      : t(strings.models.remove)}
                  </Button>
                  <span
                    data-testid={selectors.models.enabledState}
                    className="self-center text-xs text-app-muted-foreground"
                  >
                    {model.enabled
                      ? t(strings.models.enabledLabel)
                      : t(strings.models.disabledLabel)}
                  </span>
                </div>

                {installNotice?.id === model.id && (
                  <p data-testid={selectors.models.installNotice} className="mt-2 text-xs text-app-success">
                    {installNotice.alreadyInstalled
                      ? t(strings.models.alreadyInstalled)
                      : t(strings.models.installStarted, {
                          jobId: installNotice.jobId,
                          eta: installNotice.eta,
                        })}
                  </p>
                )}
              </li>
            );
          })}
        </ul>
      )}
      {mutationError && (
        <p data-testid={selectors.models.error} className="mt-2 text-app-danger">
          {errorMessage(mutationError, t)}
        </p>
      )}
    </section>
  );
}
