import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { Panel } from "../../components/ui/panel";
import { Button } from "../../components/ui/button";
import { Badge } from "../../components/ui/badge";
import { PageHeader } from "../../components/composites/PageHeader";
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
import { OverlapStallGuard } from "./OverlapStallGuard";

interface PendingSwitch {
  target: Engine;
  fromEngineId: string;
  impact: EngineSwitchImpact;
}

function CopyableCommand({ command, copyLabel, copiedLabel }: {
  command: string;
  copyLabel: string;
  copiedLabel: string;
}) {
  const [copied, setCopied] = useState(false);
  return (
    <div className="flex items-center gap-2">
      <code className="flex-1 overflow-x-auto rounded-control border border-app-border bg-app-surface-muted px-2 py-1 font-mono text-xs">
        {command}
      </code>
      <Button
        type="button"
        variant="ghost"
        size="sm"
        onClick={() => {
          // The UI NEVER runs the command; it only places it on the clipboard.
          // Older WebViews / jsdom omit navigator.clipboard despite the
          // lib.dom type asserting it is always present.
          // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
          if (navigator.clipboard) void navigator.clipboard.writeText(command);
          setCopied(true);
        }}
      >
        {copied ? copiedLabel : copyLabel}
      </Button>
    </div>
  );
}

export function StreamConfigPage() {
  const { t } = useTranslation();
  const qc = useQueryClient();
  const engines = useQuery({ queryKey: ["stt", "engines"], queryFn: listEngines });

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
                className={`flex items-center justify-between gap-3 rounded-control border px-3 py-2 ${
                  engine.isActive ? "border-app-primary" : "border-app-border"
                } ${disabled ? "opacity-60" : ""}`}
              >
                <div className="flex flex-col gap-1">
                  <div className="flex items-center gap-2">
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
