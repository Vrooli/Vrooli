// DOC: docs/internal/SEAMS.md#capability-registry-seam-api
// DOC: docs/internal/SEAMS.md#connected-scenarios-registry-seam-api-ui
import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { CheckCircle, AlertCircle, Circle, Boxes, Plug, Play, RotateCw } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useCapabilities } from "../hooks/useCapabilities";
import { strings } from "../consts/strings";
import { runCapabilityAction, type CapabilityActionResponse, type CapabilityState, type CapabilityStatus } from "../api/capabilities";

interface IntegrationsPanelProps {
  open: boolean;
}

function statusIcon(status: CapabilityStatus) {
  switch (status) {
    case "available":
      return <CheckCircle className="h-4 w-4 text-emerald-500 shrink-0" />;
    case "unavailable":
      return <AlertCircle className="h-4 w-4 text-red-500 shrink-0" />;
    default:
      return <Circle className="h-4 w-4 text-wc-text-faint shrink-0" />;
  }
}

function borderColor(status: CapabilityStatus): string {
  switch (status) {
    case "available":
      return "border-l-emerald-500";
    case "unavailable":
      return "border-l-red-500";
    default:
      return "border-l-wc-default";
  }
}

interface CapabilityCardProps {
  cap: CapabilityState;
  actionPending: boolean;
  actionResult?: CapabilityActionResponse;
  actionError?: string;
  onRunAction: (cap: CapabilityState) => void;
}

function actionIcon(kind?: string) {
  if (kind === "scenario_restart") {
    return <RotateCw className="h-3.5 w-3.5" />;
  }
  return <Play className="h-3.5 w-3.5" />;
}

function supportsBackendAction(cap: CapabilityState): boolean {
  return cap.dependencyKind === "scenario" && (cap.actionKind === "scenario_start" || cap.actionKind === "scenario_restart");
}

function CapabilityCard({ cap, actionPending, actionResult, actionError, onRunAction }: CapabilityCardProps) {
  const { t } = useTranslation();
  const isUnavailable = cap.status === "unavailable";
  const isScenario = cap.dependencyKind === "scenario";
  const canRunAction = supportsBackendAction(cap);
  // For scenario integrations that are unavailable, the message is a
  // CLI-install hint rather than a diagnostic. Treat the badge accordingly.
  const showNotYetBadge = isScenario && cap.status !== "available";

  return (
    <div
      data-testid={`cap-card-${cap.id}`}
      className={`rounded-lg border border-wc-default ${borderColor(cap.status)} border-l-2 bg-wc-surface-input px-3 py-2 space-y-1.5`}
    >
      <div className="flex items-center gap-2">
        {statusIcon(cap.status)}
        <span className="text-xs font-medium text-wc-text-primary">{cap.name}</span>
        <span className="text-[11px] px-1.5 py-0.5 rounded bg-wc-surface text-wc-text-faint">
          {cap.dependencyKind}
        </span>
        {showNotYetBadge && (
          <span
            data-testid={`cap-not-yet-${cap.id}`}
            className="text-[11px] px-1.5 py-0.5 rounded bg-amber-500/10 text-amber-300 border border-amber-500/30"
          >
            {t(strings.integrationsPanel.scenarioNotYetAvailable)}
          </span>
        )}
      </div>

      <p className="text-[11px] text-wc-text-muted">{cap.description}</p>

      {cap.message && (
        <p data-testid={`cap-message-${cap.id}`} className="text-[11px] text-wc-text-faint font-mono">
          {cap.message}
        </p>
      )}

      {(cap.reasonCode || cap.actionLabel || cap.operatorCommand) && (
        <div className="rounded border border-wc-default bg-wc-surface px-2 py-1.5 space-y-1">
          {cap.reasonCode && (
            <p data-testid={`cap-reason-${cap.id}`} className="text-[11px] text-wc-text-muted">
              {t(strings.integrationsPanel.reasonLabel)}{" "}
              <span className="font-mono text-wc-text-primary">{cap.reasonCode}</span>
            </p>
          )}
          {cap.actionLabel && (
            <p data-testid={`cap-action-${cap.id}`} className="text-[11px] text-wc-text-muted">
              {t(strings.integrationsPanel.nextActionLabel)}{" "}
              <span className="text-wc-text-primary">{cap.actionLabel}</span>
            </p>
          )}
          {cap.operatorCommand && (
            <p data-testid={`cap-command-${cap.id}`} className="text-[11px] text-wc-text-faint font-mono break-all">
              {cap.operatorCommand}
            </p>
          )}
          {canRunAction && (
            <button
              type="button"
              data-testid={`cap-run-action-${cap.id}`}
              disabled={actionPending}
              onClick={() => onRunAction(cap)}
              className="inline-flex h-7 items-center gap-1.5 rounded border border-wc-default bg-wc-surface-input px-2 text-[11px] text-wc-text-primary hover:bg-wc-surface disabled:cursor-wait disabled:opacity-60"
            >
              {actionIcon(cap.actionKind)}
              <span>{actionPending ? t(strings.integrationsPanel.actionRunning) : cap.actionLabel}</span>
            </button>
          )}
        </div>
      )}

      {actionResult && (
        <p
          data-testid={`cap-action-result-${cap.id}`}
          className={`text-[11px] ${actionResult.success ? "text-emerald-400" : "text-red-400"}`}
        >
          {actionResult.message || actionResult.status}
        </p>
      )}
      {actionError && (
        <p data-testid={`cap-action-error-${cap.id}`} className="text-[11px] text-red-400">
          {t(strings.integrationsPanel.actionFailed, { message: actionError })}
        </p>
      )}

      {cap.features.length > 0 && (
        <div className="flex flex-wrap gap-1 mt-1">
          {cap.features.map((feature) => (
            <span
              key={feature}
              className={`text-[11px] px-1.5 py-0.5 rounded-full border ${
                isUnavailable
                  ? "border-wc-default text-wc-text-faint line-through"
                  : "border-wc-default text-wc-text-muted"
              }`}
            >
              {feature}
            </span>
          ))}
        </div>
      )}
    </div>
  );
}

interface GroupProps {
  testId: string;
  icon: React.ReactNode;
  heading: string;
  description: string;
  items: CapabilityState[];
  pendingCapabilityId?: string;
  lastActionResult?: CapabilityActionResponse;
  lastActionError?: { capabilityId: string; message: string };
  onRunAction: (cap: CapabilityState) => void;
}

function IntegrationsGroup({
  testId,
  icon,
  heading,
  description,
  items,
  pendingCapabilityId,
  lastActionResult,
  lastActionError,
  onRunAction,
}: GroupProps) {
  if (items.length === 0) return null;
  return (
    <section data-testid={testId} className="space-y-2">
      <header className="flex items-center gap-2 px-1">
        <span className="text-wc-text-muted">{icon}</span>
        <h3 className="text-xs font-semibold uppercase tracking-wide text-wc-text-muted">{heading}</h3>
      </header>
      <p className="text-[11px] text-wc-text-faint px-1">{description}</p>
      <div className="flex flex-col gap-2">
        {items.map((cap) => (
          <CapabilityCard
            key={cap.id}
            cap={cap}
            actionPending={pendingCapabilityId === cap.id}
            actionResult={lastActionResult?.capabilityId === cap.id ? lastActionResult : undefined}
            actionError={lastActionError?.capabilityId === cap.id ? lastActionError.message : undefined}
            onRunAction={onRunAction}
          />
        ))}
      </div>
    </section>
  );
}

export default function IntegrationsPanel({ open }: IntegrationsPanelProps) {
  const { t } = useTranslation();
  const { data, isLoading, isError, error } = useCapabilities(open);
  const queryClient = useQueryClient();
  const [pendingCapabilityId, setPendingCapabilityId] = useState<string>();
  const [lastActionResult, setLastActionResult] = useState<CapabilityActionResponse>();
  const [lastActionError, setLastActionError] = useState<{ capabilityId: string; message: string }>();
  const actionMutation = useMutation({
    mutationFn: (cap: CapabilityState) => {
      if (!cap.actionKind) throw new Error("action kind missing");
      return runCapabilityAction(cap.id, cap.actionKind);
    },
    onMutate: (cap) => {
      setPendingCapabilityId(cap.id);
      setLastActionResult(undefined);
      setLastActionError(undefined);
    },
    onSuccess: (result) => {
      setLastActionResult(result);
      queryClient.setQueryData(["capabilities"], {
        capabilities: result.capabilities,
        timestamp: result.timestamp,
      });
    },
    onError: (err, cap) => {
      setLastActionError({ capabilityId: cap.id, message: err instanceof Error ? err.message : String(err) });
    },
    onSettled: () => {
      setPendingCapabilityId(undefined);
      queryClient.invalidateQueries({ queryKey: ["capabilities"] });
    },
  });

  const handleRunAction = (cap: CapabilityState) => {
    if (supportsBackendAction(cap) && !actionMutation.isPending) {
      actionMutation.mutate(cap);
    }
  };

  if (isLoading) {
    return (
      <div className="space-y-4">
        <p className="text-[11px] text-wc-text-faint">{t(strings.integrationsPanel.checking)}</p>
      </div>
    );
  }

  if (isError) {
    return (
      <div className="space-y-4">
        <p className="text-[11px] text-red-400">{t(strings.integrationsPanel.loadFailed, { message: error.message })}</p>
      </div>
    );
  }

  const capabilities = data?.capabilities ?? [];
  const activeCount = capabilities.filter((c) => c.status === "available").length;
  const scenarios = capabilities.filter((c) => c.dependencyKind === "scenario");
  const resources = capabilities.filter((c) => c.dependencyKind !== "scenario");

  if (capabilities.length === 0) {
    return (
      <div className="space-y-3">
        <p className="text-[11px] text-wc-text-faint">{t(strings.integrationsPanel.noneConfigured)}</p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between px-1">
        <span className="text-[11px] text-wc-text-faint">
          {t(strings.integrationsPanel.activeCount, { active: activeCount, total: capabilities.length })}
        </span>
      </div>

      <IntegrationsGroup
        testId="integrations-group-scenarios"
        icon={<Boxes className="h-4 w-4" />}
        heading={t(strings.integrationsPanel.connectedScenariosHeading)}
        description={t(strings.integrationsPanel.connectedScenariosDescription)}
        items={scenarios}
        pendingCapabilityId={pendingCapabilityId}
        lastActionResult={lastActionResult}
        lastActionError={lastActionError}
        onRunAction={handleRunAction}
      />

      <IntegrationsGroup
        testId="integrations-group-resources"
        icon={<Plug className="h-4 w-4" />}
        heading={t(strings.integrationsPanel.localResourcesHeading)}
        description={t(strings.integrationsPanel.localResourcesDescription)}
        items={resources}
        pendingCapabilityId={pendingCapabilityId}
        lastActionResult={lastActionResult}
        lastActionError={lastActionError}
        onRunAction={handleRunAction}
      />
    </div>
  );
}
