import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { UseQueryResult } from "@tanstack/react-query";

import { Panel } from "../../components/ui/panel";
import { Button } from "../../components/ui/button";
import { Badge } from "../../components/ui/badge";
import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";
import { ApiErrorState } from "../../components/composites/ApiErrorState";
import { LoadingRows } from "../../components/composites/LoadingRows";
import { pushToast } from "../../components/ui/toast";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";
import { selectors } from "../../consts/selectors";
import {
  getEngineSwitchImpact,
  listEngines,
  setEngine,
  type Engine,
  type EngineSwitchImpact,
} from "../../services/sttEngines";
import { listModels, type ModelRow } from "../../services/models";
import type { Result } from "../../services/result";
import {
  commercialLabelKey,
  commercialTone,
  languageSummary,
  readinessLabelKey,
  readinessTone,
} from "../models/modelDisplay";
import { OverlapStallGuard } from "./OverlapStallGuard";
import { CopyableCommand } from "../../components/composites/CopyableCommand";

interface PendingSwitch {
  target: Engine;
  fromEngineId: string;
  impact: EngineSwitchImpact;
}

export function StreamConfigPage() {
  const { t } = useTranslation();
  const qc = useQueryClient();
  const engines = useQuery({ queryKey: ["stt", "engines"], queryFn: listEngines });

  // Registry facts for the picker. Fetched independently so a registry outage
  // never blocks engine selection — each row carries its own loading/error UI.
  const models = useQuery({
    queryKey: ["models", "stt"],
    queryFn: () => listModels({ capability: "stt" }),
  });

  const [pending, setPending] = useState<PendingSwitch | null>(null);

  const impactMut = useMutation({
    mutationFn: async (target: Engine) => {
      const active = (engines.data ?? []).find((e) => e.isActive);
      const fromEngineId = active?.id ?? "";
      const impact = await getEngineSwitchImpact(fromEngineId);
      return { target, fromEngineId, impact } satisfies PendingSwitch;
    },
    onSuccess: (p) => setPending(p),
  });

  const switchMut = useMutation({
    mutationFn: (engineId: string) => setEngine(engineId),
    onSuccess: (_data, engineId) => {
      const target = (engines.data ?? []).find((e) => e.id === engineId);
      setPending(null);
      void qc.invalidateQueries({ queryKey: ["stt", "engines"] });
      pushToast({
        title: t(strings.streamConfigAdmin.switchSuccess, { engine: target?.displayName ?? engineId }),
      });
    },
  });

  if (engines.isPending) {
    return (
      <div className="flex flex-col gap-4">
        <PageHeader
          title={t(strings.streamConfigAdmin.pageTitle)}
          description={t(strings.streamConfigAdmin.pageDescription)}
        />
        <LoadingRows />
      </div>
    );
  }

  if (engines.isError) {
    return (
      <div className="flex flex-col gap-4">
        <PageHeader
          title={t(strings.streamConfigAdmin.pageTitle)}
          description={t(strings.streamConfigAdmin.pageDescription)}
        />
        <ApiErrorState error={engines.error} onRetry={() => void engines.refetch()} />
      </div>
    );
  }

  const list = engines.data;

  return (
    <div className="flex flex-col gap-4">
      <PageHeader
        title={t(strings.streamConfigAdmin.pageTitle)}
        description={t(strings.streamConfigAdmin.pageDescription)}
      />

      <Panel title={t(strings.streamConfigAdmin.enginesTitle)}>
        <p className="px-4 pt-3 text-xs text-app-muted-foreground">
          {t(strings.streamConfigAdmin.enginesHelp)}
        </p>
        <ul
          className="flex flex-col gap-2 p-4"
          data-testid={selectors.streamConfig.enginePicker}
        >
          {list.map((engine) => {
            const disabled = !engine.available;
            return (
              <li
                key={engine.id}
                data-testid={selectors.streamConfig.engineRow({ id: engine.id })}
                className={`flex flex-col gap-3 rounded-control border px-3 py-2 ${
                  engine.isActive ? "border-app-primary" : "border-app-border"
                } ${disabled ? "opacity-60" : ""}`}
              >
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div className="flex min-w-0 flex-col gap-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="text-sm font-medium text-app-foreground">{engine.displayName}</span>
                      <Badge variant={engine.nativeStreaming ? "info" : "neutral"}>
                        {engine.nativeStreaming
                          ? t(strings.streamConfigAdmin.badgeNativeStreaming)
                          : t(strings.streamConfigAdmin.badgeBatch)}
                      </Badge>
                      {engine.isActive ? (
                        <Badge variant="success">{t(strings.streamConfigAdmin.engineActive)}</Badge>
                      ) : null}
                    </div>
                    <span className="font-mono text-xs text-app-muted-foreground">{engine.kind}</span>
                    {disabled ? (
                      <span className="text-xs text-app-danger">
                        {t(strings.streamConfigAdmin.engineUnavailable)}
                      </span>
                    ) : null}
                  </div>
                  <Button
                    type="button"
                    size="sm"
                    variant={engine.isActive ? "ghost" : "default"}
                    disabled={disabled || engine.isActive || impactMut.isPending || switchMut.isPending}
                    data-testid={selectors.streamConfig.engineSelect({ id: engine.id })}
                    onClick={() => impactMut.mutate(engine)}
                  >
                    {engine.isActive
                      ? t(strings.streamConfigAdmin.engineActive)
                      : t(strings.streamConfigAdmin.selectEngine)}
                  </Button>
                </div>
                <EngineModelFacts engineId={engine.id} modelsQuery={models} />
              </li>
            );
          })}
        </ul>
      </Panel>

      <OverlapStallGuard />

      {pending ? (
        <Panel
          title={t(strings.streamConfigAdmin.switchPromptTitle)}
          data-testid={selectors.streamConfig.switchPrompt}
        >
          <div className="flex flex-col gap-3 px-4 py-3 text-sm">
            {pending.impact.resource === "" ? (
              <p className="text-app-muted-foreground">
                {t(strings.streamConfigAdmin.switchNoResource)}
              </p>
            ) : !pending.impact.consumersKnown ? (
              <p className="text-app-warning">
                {t(strings.streamConfigAdmin.switchConsumersUnknown, { resource: pending.impact.resource })}
              </p>
            ) : pending.impact.consumers.length === 0 && pending.impact.safeToStop ? (
              <>
                <p className="text-app-muted-foreground">
                  {t(strings.streamConfigAdmin.switchSafeToStop, { resource: pending.impact.resource })}
                </p>
                {pending.impact.stopCommand ? (
                  <div className="flex flex-col gap-1">
                    <span className="text-xs text-app-muted-foreground">
                      {t(strings.streamConfigAdmin.stopCommandLabel)}
                    </span>
                    <CopyableCommand
                      command={pending.impact.stopCommand}
                      copyLabel={t(strings.streamConfigAdmin.copyCommand)}
                      copiedLabel={t(strings.streamConfigAdmin.copied)}
                    />
                  </div>
                ) : null}
              </>
            ) : (
              <>
                <p className="text-app-foreground">
                  {t(strings.streamConfigAdmin.switchConsumersHeader, { resource: pending.impact.resource })}
                </p>
                <ul className="flex flex-col gap-1">
                  {pending.impact.consumers.map((c) => (
                    <li key={c.scenario} className="flex items-center gap-2 text-xs">
                      <span className="text-app-foreground">{c.displayName}</span>
                      <span className="font-mono text-app-muted-foreground">{c.scenario}</span>
                      {c.required ? (
                        <Badge variant="warning">{t(strings.streamConfigAdmin.consumerRequired)}</Badge>
                      ) : null}
                    </li>
                  ))}
                </ul>
                {pending.impact.stopCommand ? (
                  <div className="flex flex-col gap-1">
                    <span className="text-xs text-app-muted-foreground">
                      {t(strings.streamConfigAdmin.stopCommandHintConsumers)}
                    </span>
                    <CopyableCommand
                      command={pending.impact.stopCommand}
                      copyLabel={t(strings.streamConfigAdmin.copyCommand)}
                      copiedLabel={t(strings.streamConfigAdmin.copied)}
                    />
                  </div>
                ) : null}
              </>
            )}
            <div className="flex gap-2">
              <Button
                type="button"
                data-testid={selectors.streamConfig.confirmSwitch}
                disabled={switchMut.isPending}
                onClick={() => switchMut.mutate(pending.target.id)}
              >
                {t(strings.streamConfigAdmin.confirmSwitch)}
              </Button>
              <Button
                type="button"
                variant="ghost"
                data-testid={selectors.streamConfig.cancelSwitch}
                onClick={() => setPending(null)}
              >
                {t(strings.streamConfigAdmin.cancelSwitch)}
              </Button>
            </div>
          </div>
        </Panel>
      ) : null}
    </div>
  );
}

interface EngineModelFactsProps {
  engineId: string;
  modelsQuery: UseQueryResult<Result<ModelRow[]>>;
}

/**
 * Registry facts for one engine. Rendered inside the engine row so the model a
 * user is about to run alongside an engine is never hidden — including the
 * honest "nothing registered" case. The section owns its loading and error
 * states so engine selection stays usable while the registry is unavailable.
 */
function EngineModelFacts({ engineId, modelsQuery }: EngineModelFactsProps) {
  const { t } = useTranslation();
  const dash = t(strings.common.dash);

  const heading = (
    <span className="text-[11px] font-medium uppercase tracking-wide text-app-muted-foreground">
      {t(strings.streamConfigAdmin.modelsTitle)}
    </span>
  );

  if (modelsQuery.isPending) {
    return (
      <div className="flex min-w-0 flex-col gap-1 border-t border-app-border pt-2">
        {heading}
        <span className="text-xs text-app-muted-foreground">
          {t(strings.streamConfigAdmin.modelsLoading)}
        </span>
      </div>
    );
  }

  const result = modelsQuery.data;
  if (!result || !result.ok) {
    return (
      <div className="flex min-w-0 flex-col gap-1 border-t border-app-border pt-2">
        {heading}
        <div className="flex flex-wrap items-center gap-2 text-xs text-app-danger">
          <span>{t(strings.streamConfigAdmin.modelsLoadError)}</span>
          <Button
            type="button"
            variant="link"
            size="sm"
            className="h-auto p-0 text-xs"
            onClick={() => void modelsQuery.refetch()}
          >
            {t(strings.streamConfigAdmin.modelsRetry)}
          </Button>
        </div>
      </div>
    );
  }

  // Defaults lead the list so the engine's resolved pick is the first thing a
  // reader sees; the badge still marks only the first default when the registry
  // reports several.
  const engineModels = result.data
    .filter((model) => model.engineIds.includes(engineId))
    .sort((a, b) => Number(b.isDefault) - Number(a.isDefault));
  const defaultId = engineModels.find((model) => model.isDefault)?.id;

  if (engineModels.length === 0) {
    return (
      <div className="flex min-w-0 flex-col gap-1 border-t border-app-border pt-2">
        {heading}
        <span className="text-xs text-app-muted-foreground">
          {t(strings.streamConfigAdmin.modelsEmpty)}
        </span>
      </div>
    );
  }

  return (
    <div className="flex min-w-0 flex-col gap-1 border-t border-app-border pt-2">
      {heading}
      <ul className="flex min-w-0 flex-col gap-1.5">
        {engineModels.map((model) => {
          const languages = languageSummary(model.languages);
          return (
            <li
              key={model.id}
              className="flex min-w-0 flex-col gap-1 rounded-control bg-app-surface-muted/50 px-2 py-1.5"
            >
              <div className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1">
                <span className="min-w-0 break-words text-xs font-medium text-app-foreground">
                  {model.name || model.id}
                </span>
                {model.id === defaultId ? (
                  <Badge variant="primary">{t(strings.models.defaultBadge)}</Badge>
                ) : null}
                <Badge variant={readinessTone(model.readiness)}>
                  {t(readinessLabelKey(model.readiness))}
                </Badge>
                <Badge variant={commercialTone(model.commercialUse)}>
                  {t(commercialLabelKey(model.commercialUse))}
                </Badge>
              </div>
              <div className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1 text-[11px] text-app-muted-foreground">
                <span className="min-w-0 break-all font-mono">{model.id}</span>
                <span className="break-words">
                  {t(strings.streamConfigAdmin.modelRuntimeLabel)}
                  {": "}
                  {model.runtime || dash}
                </span>
                <span className="break-words">
                  {t(strings.streamConfigAdmin.modelSizeLabel)}
                  {": "}
                  {model.sizeMbApprox > 0
                    ? t(strings.models.sizeMb, { mb: model.sizeMbApprox })
                    : dash}
                </span>
                <span className="break-words">
                  {t(strings.streamConfigAdmin.modelLanguagesLabel)}
                  {": "}
                  {languages.shown.length === 0 ? (
                    dash
                  ) : (
                    <>
                      {languages.shown.join(", ")}
                      {languages.overflow > 0 ? (
                        <span className="ml-1">
                          {t(strings.models.languageOverflow, { count: languages.overflow })}
                        </span>
                      ) : null}
                    </>
                  )}
                </span>
              </div>
            </li>
          );
        })}
      </ul>
    </div>
  );
}
