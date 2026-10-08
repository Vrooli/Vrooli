import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { getProviderHealth } from "../../api/healthStatus";
import {
  listLocalProviders,
  pullModel,
  restartProvider,
  startProvider,
  stopProvider,
} from "../../api/providerLifecycle";
import { Action, type LocalProvider } from "@vrooli/proto-types/audio-tools/v1/provider_lifecycle/provider_lifecycle_pb";
import { CapabilityRow } from "./CapabilityRow";
import { LogsDrawer } from "./LogsDrawer";
import { PullModelModal } from "./PullModelModal";
import { Button } from "../../components/ui/button";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";

const MIN_REFETCH_MS = 5_000;

export function StatusPage() {
  const { t } = useTranslation();
  const qc = useQueryClient();
  const [logsFor, setLogsFor] = useState<string | null>(null);
  const [pullOpen, setPullOpen] = useState(false);
  const [pendingAction, setPendingAction] = useState<{ providerId: string; verb: string } | null>(null);

  const health = useQuery({
    queryKey: ["healthStatus", "providers"],
    queryFn: getProviderHealth,
    refetchInterval: (q) => {
      const ttl = q.state.data?.cacheTtlSeconds ?? 0;
      return Math.max(MIN_REFETCH_MS, Math.floor((ttl / 2) * 1000));
    },
    retry: false,
  });

  const providers = useQuery({
    queryKey: ["providerLifecycle", "list"],
    queryFn: listLocalProviders,
    refetchInterval: MIN_REFETCH_MS * 6,
    retry: false,
  });

  function invalidateAll() {
    void qc.invalidateQueries({ queryKey: ["healthStatus", "providers"] });
    void qc.invalidateQueries({ queryKey: ["providerLifecycle", "list"] });
  }

  const startMut = useMutation({
    mutationFn: (id: string) => startProvider(id),
    onSettled: () => {
      setPendingAction(null);
      invalidateAll();
    },
  });
  const stopMut = useMutation({
    mutationFn: (id: string) => stopProvider(id),
    onSettled: () => {
      setPendingAction(null);
      invalidateAll();
    },
  });
  const restartMut = useMutation({
    mutationFn: (id: string) => restartProvider(id),
    onSettled: () => {
      setPendingAction(null);
      invalidateAll();
    },
  });
  const pullMut = useMutation({
    mutationFn: (model: string) => pullModel(model),
    onSettled: () => {
      setPullOpen(false);
      invalidateAll();
    },
  });

  function triggerStart(id: string) {
    setPendingAction({ providerId: id, verb: "start" });
    startMut.mutate(id);
  }
  function triggerStop(id: string) {
    setPendingAction({ providerId: id, verb: "stop" });
    stopMut.mutate(id);
  }
  function triggerRestart(id: string) {
    setPendingAction({ providerId: id, verb: "restart" });
    restartMut.mutate(id);
  }

  if (health.isLoading) {
    return <p className="text-sm text-app-muted-foreground">{t(strings.status.loadingHealth)}</p>;
  }
  if (health.error || !health.data) {
    return (
      <p className="text-sm text-app-danger">
        {t(strings.status.loadFailed)}
      </p>
    );
  }

  const localByID = new Map<string, LocalProvider>();
  for (const p of providers.data?.providers ?? []) {
    localByID.set(p.providerId, p);
  }

  return (
    <section className="flex flex-col gap-4" aria-label={t(strings.status.pageAriaLabel)}>
      <header className="flex flex-col gap-1">
        <h1 className="text-2xl font-semibold text-app-foreground">{t(strings.status.pageTitle)}</h1>
        <p className="text-sm text-app-muted-foreground">
          {t(strings.status.metaLine, {
            generatedAt: health.data.generatedAt,
            cadenceSeconds: health.data.cacheTtlSeconds,
          })}
        </p>
      </header>
      {health.data.capabilities.length === 0 ? (
        <p className="text-sm text-app-muted-foreground">
          {t(strings.status.noCapabilities)}
        </p>
      ) : (
        <div className="flex flex-col gap-3">
          {health.data.capabilities.map((c) => (
            <CapabilityRow
              key={c.capability}
              capability={c}
              renderProviderActions={(providerId) => {
                const local = localByID.get(providerId);
                if (!local) return null;
                const isPending = pendingAction?.providerId === providerId;
                const actions = new Set(local.supportedActions);
                return (
                  <div className="flex flex-wrap gap-2 pt-2">
                    {actions.has(Action.START) && (
                      <Button size="sm" variant="outline" disabled={isPending} onClick={() => triggerStart(providerId)}>
                        {t(strings.status.actionStart)}
                      </Button>
                    )}
                    {actions.has(Action.STOP) && (
                      <Button size="sm" variant="outline" disabled={isPending} onClick={() => triggerStop(providerId)}>
                        {t(strings.status.actionStop)}
                      </Button>
                    )}
                    {actions.has(Action.RESTART) && (
                      <Button size="sm" variant="outline" disabled={isPending} onClick={() => triggerRestart(providerId)}>
                        {t(strings.status.actionRestart)}
                      </Button>
                    )}
                    {actions.has(Action.PULL_MODEL) && (
                      <Button size="sm" variant="subtle" onClick={() => setPullOpen(true)}>
                        {t(strings.status.actionPullModel)}
                      </Button>
                    )}
                    {actions.has(Action.VIEW_LOGS) && (
                      <Button size="sm" variant="ghost" onClick={() => setLogsFor(providerId)}>
                        {t(strings.status.actionViewLogs)}
                      </Button>
                    )}
                  </div>
                );
              }}
            />
          ))}
        </div>
      )}
      <LogsDrawer
        open={logsFor !== null}
        providerId={logsFor ?? ""}
        onClose={() => setLogsFor(null)}
      />
      <PullModelModal
        open={pullOpen}
        pending={pullMut.isPending}
        onClose={() => setPullOpen(false)}
        onConfirm={(name) => pullMut.mutate(name)}
      />
    </section>
  );
}

export default StatusPage;
