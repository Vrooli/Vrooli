// DOC: docs/internal/SEAMS.md#capability-registry-seam-api
// DOC: docs/internal/SEAMS.md#connected-scenarios-registry-seam-api-ui
import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { CheckCircle, AlertCircle, Circle, Boxes, Plug, Play, RotateCw } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useCapabilities } from "../hooks/useCapabilities";
import { strings } from "../consts/strings";
import { runCapabilityAction, type CapabilityActionResponse, type CapabilityState, type CapabilityStatus } from "../api/capabilities";
import { getCommercialContext, getConnections, isCommercialContentVisible, removeOpenRouterKey, safeCommercialDestination, testOpenRouterKey } from "../api/monetization";
import { getWebAccessToken } from "../lib/auth";
import { StatusBadge, type StatusTone } from "@vrooli/react-component-library/StatusBadge/1.2.2";
import { IntegrationCard } from "@vrooli/react-component-library/IntegrationCard/0";
import { Button } from "@vrooli/react-component-library/Button/2";

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

function statusTone(status: CapabilityStatus): StatusTone {
  if (status === "available") return "success";
  if (status === "unavailable") return "danger";
  return "neutral";
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

function supportsCredentialActions(actions?: string[]): boolean {
  return Boolean(actions?.some((action) => action === "test" || action === "delete"));
}

function ConnectionActions({ supportedActions }: { supportedActions: string[] }) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [message, setMessage] = useState<string>();
  const testMutation = useMutation({
    mutationFn: testOpenRouterKey,
    onSuccess: (result) => {
      setMessage(result.valid ? t(strings.integrationsPanel.connectionVerified) : t(strings.integrationsPanel.connectionTestFailed));
      void queryClient.invalidateQueries({ queryKey: ["integration-connections"] });
    },
    onError: () => setMessage(t(strings.integrationsPanel.connectionTestUnavailable)),
  });
  const removeMutation = useMutation({
    mutationFn: removeOpenRouterKey,
    onSuccess: () => {
      setMessage(t(strings.integrationsPanel.connectionRemoved)),
      void queryClient.invalidateQueries({ queryKey: ["integration-connections"] });
    },
    onError: () => setMessage(t(strings.integrationsPanel.connectionRemoveFailed)),
  });

  return (
    <div className="flex flex-wrap items-center gap-2">
      {supportedActions.includes("test") && <Button type="button" size="sm" variant="secondary" onClick={() => { setMessage(undefined); testMutation.mutate(); }} disabled={testMutation.isPending || removeMutation.isPending}>
        {testMutation.isPending ? t(strings.integrationsPanel.testingConnection) : t(strings.integrationsPanel.testConnection)}
      </Button>}
      {supportedActions.includes("delete") && <Button type="button" size="sm" variant="ghost" onClick={() => {
        if (window.confirm(t(strings.integrationsPanel.connectionRemoveConfirm))) {
          setMessage(undefined);
          removeMutation.mutate();
        }
      }} disabled={testMutation.isPending || removeMutation.isPending}>
        {removeMutation.isPending ? t(strings.integrationsPanel.removingConnection) : t(strings.integrationsPanel.removeConnection)}
      </Button>}
      {message && <span role="status" className="text-[11px] text-wc-text-muted">{message}</span>}
    </div>
  );
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
        <StatusBadge tone={statusTone(cap.status)}>{cap.status}</StatusBadge>
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
  const connections = useQuery({
    queryKey: ["integration-connections"],
    queryFn: getConnections,
    enabled: open,
    staleTime: 30_000,
    retry: false,
  });
  const commercialCapabilityId = data?.capabilities.find((cap) => cap.status === "unavailable")?.id ?? "";
  const commercial = useQuery({
    queryKey: ["commercial-context", "integrations", commercialCapabilityId],
    queryFn: () => getCommercialContext("integrations", commercialCapabilityId),
    enabled: open && Boolean(getWebAccessToken()),
    staleTime: 60_000,
    retry: false,
  });
  const queryClient = useQueryClient();
  const [pendingCapabilityId, setPendingCapabilityId] = useState<string>();
  const [lastActionResult, setLastActionResult] = useState<CapabilityActionResponse>();
  const [lastActionError, setLastActionError] = useState<{ capabilityId: string; message: string }>();
  const [dismissedCommercialContent, setDismissedCommercialContent] = useState<Set<string>>(() => new Set());
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

  const renderConnectedAccounts = () => (
    <section data-testid="connected-accounts" className="space-y-2">
      <div className="flex items-center justify-between px-1">
        <h3 className="text-xs font-semibold uppercase tracking-wide text-wc-text-muted">{t(strings.integrationsPanel.connectedAccountsHeading)}</h3>
        <span className="text-[11px] text-wc-text-faint">{t(strings.integrationsPanel.managedConnections)}</span>
      </div>
      {connections.isLoading ? <p className="text-[11px] text-wc-text-faint">{t(strings.integrationsPanel.loadingConnectedAccounts)}</p> : connections.isError ? <p data-testid="connected-accounts-error" className="text-[11px] text-amber-200">{t(strings.integrationsPanel.connectionsUnavailableWithRuntime)}</p> : connections.data?.connections.length ? connections.data.connections.map((connection) => (
        <IntegrationCard
          key={connection.id}
          providerName={connection.provider}
          connectionName={connection.connection_name}
          status={connection.status}
          bindings={connection.bindings}
          nextAction={connection.next_action}
          actions={connection.id === "vrooli/openrouter" && supportsCredentialActions(connection.supported_actions) ? <ConnectionActions supportedActions={connection.supported_actions ?? []} /> : undefined}
        />
      )) : <p className="text-[11px] text-wc-text-faint px-1">{t(strings.integrationsPanel.noProviderAccounts)}</p>}
    </section>
  );

  if (isLoading) {
    return (
      <div className="space-y-4">
        {renderConnectedAccounts()}
        <section data-testid="runtime-dependencies" className="space-y-2">
          <h3 className="text-xs font-semibold uppercase tracking-wide text-wc-text-muted">{t(strings.integrationsPanel.runtimeDependenciesHeading)}</h3>
          <p className="text-[11px] text-wc-text-faint">{t(strings.integrationsPanel.checking)}</p>
        </section>
      </div>
    );
  }

  if (isError) {
    return (
      <div className="space-y-4">
        {renderConnectedAccounts()}
        <section data-testid="runtime-dependencies" className="space-y-2">
          <h3 className="text-xs font-semibold uppercase tracking-wide text-wc-text-muted">{t(strings.integrationsPanel.runtimeDependenciesHeading)}</h3>
          <p className="text-[11px] text-red-400">{t(strings.integrationsPanel.loadFailed, { message: error.message })}</p>
        </section>
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
        {renderConnectedAccounts()}
        <p className="text-[11px] text-wc-text-faint">{t(strings.integrationsPanel.noneConfigured)}</p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {capabilities.some((cap) => cap.status === "unavailable") && (
        <section data-testid="needs-attention" className="rounded-lg border border-amber-500/30 bg-amber-500/5 px-3 py-2">
          <h3 className="text-xs font-semibold text-amber-200">{t(strings.integrationsPanel.needsAttentionHeading)}</h3>
          <p className="mt-1 text-[11px] text-amber-100/70">{t(strings.integrationsPanel.needsAttentionDescription)}</p>
        </section>
      )}
      {renderConnectedAccounts()}
      {commercial.data?.content.some((item) => isCommercialContentVisible(item, "integrations") && !dismissedCommercialContent.has(item.content_id)) && (
        <section data-testid="commercial-context" className="rounded-lg border border-wc-default bg-wc-surface-input px-3 py-2">
          {commercial.data.content.filter((item) => isCommercialContentVisible(item, "integrations") && !dismissedCommercialContent.has(item.content_id)).map((item) => (
            <div key={item.content_id} className="space-y-1">
              <div className="flex items-start justify-between gap-2">
                <h3 className="text-xs font-semibold text-wc-text-primary">{item.title}</h3>
                {item.dismissible && <button type="button" className="text-[10px] text-wc-text-faint underline" aria-label={t(strings.integrationsPanel.dismissRecommendation)} onClick={() => setDismissedCommercialContent((current) => new Set(current).add(item.content_id))}>{t(strings.integrationsPanel.dismissRecommendation)}</button>}
              </div>
              <p className="text-[11px] text-wc-text-muted">{item.description}</p>
              {safeCommercialDestination(item.cta_destination) && <a className="text-[11px] text-wc-text-primary underline" href={safeCommercialDestination(item.cta_destination) ?? undefined}>{item.cta_label}</a>}
            </div>
          ))}
          {commercial.data.stale && <p className="mt-1 text-[10px] text-wc-text-faint">{t(strings.integrationsPanel.staleCommercialContext)}</p>}
        </section>
      )}
      {commercial.isError && <p data-testid="commercial-context-error" className="text-[11px] text-wc-text-faint">{t(strings.integrationsPanel.accountRecommendationsUnavailable)}</p>}
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
