import { useState } from "react";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import {
  CheckCircle2,
  CircleSlash,
  HelpCircle,
  Plus,
  PowerOff,
  RefreshCw,
  ServerCog,
  ShieldOff,
  type LucideIcon,
} from "lucide-react";

import { Button } from "@vrooli/react-component-library/Button/2";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { formatDate } from "../../i18n/format";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import { NodeStatus, type Node } from "../../api/nodes";
import { OnboardingStepStatus, type OnboardingOp } from "../../api/onboard";
import { type NodeQueue } from "../../api/queue";
import { NodeManagementPanel } from "./NodeManagementPanel";
import { PendingPairingPanel } from "./PendingPairingPanel";
import { useBridgeFirewallActionMutation, useBridgeReadinessQuery, useFailedOnboardingsQuery, useFleetQueueQuery, useNodesQuery, useOnboardingQuery, useRemoveFailedOnboardingMutation, useRevokeNodeMutation } from "./queries";

const STATUS_LABEL = {
  [NodeStatus.UNSPECIFIED]: strings.fleet.status.unspecified,
  [NodeStatus.OFFLINE]: strings.fleet.status.offline,
  [NodeStatus.ONLINE]: strings.fleet.status.online,
  [NodeStatus.NEEDS_UPDATE]: strings.fleet.status.needsUpdate,
  [NodeStatus.REVOKED]: strings.fleet.status.revoked,
} as const satisfies Record<NodeStatus, string>;

// Health is conveyed by THREE redundant channels — a colored dot, a distinct
// icon, AND a text label — so it never depends on color alone (WCAG 1.4.1).
const STATUS_ICON: Record<NodeStatus, LucideIcon> = {
  [NodeStatus.UNSPECIFIED]: HelpCircle,
  [NodeStatus.OFFLINE]: PowerOff,
  [NodeStatus.ONLINE]: CheckCircle2,
  [NodeStatus.NEEDS_UPDATE]: RefreshCw,
  [NodeStatus.REVOKED]: CircleSlash,
};

const ONBOARDING_STEP_STATUS_KEY = {
  [OnboardingStepStatus.UNSPECIFIED]: strings.fleet.onboard.stepStatus.unspecified,
  [OnboardingStepStatus.STARTED]: strings.fleet.onboard.stepStatus.started,
  [OnboardingStepStatus.OK]: strings.fleet.onboard.stepStatus.ok,
  [OnboardingStepStatus.SKIPPED]: strings.fleet.onboard.stepStatus.skipped,
  [OnboardingStepStatus.FAILED]: strings.fleet.onboard.stepStatus.failed,
} as const satisfies Record<OnboardingStepStatus, string>;

function dotClass(node: Node): string {
  if (node.status === NodeStatus.REVOKED) return "bg-app-danger";
  if (node.status === NodeStatus.NEEDS_UPDATE) return "bg-app-warning";
  return node.online ? "bg-app-success" : "bg-app-muted-foreground";
}

/**
 * Render a node's provenance revision: shorten a long commit sha but PRESERVE the
 * "+dirty" working-tree marker, so a node onboarded from a working tree reads
 * loudly as "e767613fca+dirty" — visibly not a pinned node — rather than being
 * truncated back to a clean-looking sha.
 */
function formatRevision(rev: string): string {
  const dirty = rev.endsWith("+dirty");
  const base = dirty ? rev.slice(0, -"+dirty".length) : rev;
  const shortBase = base.length > 12 ? base.slice(0, 12) : base;
  return dirty ? `${shortBase}+dirty` : shortBase;
}

/** A labeled OS/arch/version/health metadata cell. */
function MetaField({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0">
      <dt className="text-[0.65rem] uppercase tracking-wide text-app-muted-foreground">{label}</dt>
      <dd className="truncate text-xs font-medium text-app-foreground">{value}</dd>
    </div>
  );
}

/** Live per-node job status from the queue overlay (best-effort). */
function JobStatus({ nodeId, queue }: { nodeId: string; queue?: NodeQueue }) {
  const { t } = useTranslation();
  const running = queue?.running ?? 0;
  const queued = queue?.queued ?? 0;
  const busy = running > 0 || queued > 0;
  return (
    <p
      data-testid={selectors.fleet.jobs({ id: nodeId })}
      className="mt-1 text-xs text-app-muted-foreground"
    >
      <span className="font-medium text-app-foreground">{t(strings.fleet.jobsHeading)}: </span>
      {busy ? t(strings.fleet.jobsBusy, { running, queued }) : t(strings.fleet.jobsIdle)}
    </p>
  );
}

/**
 * A failed operation's list entry. The list endpoint intentionally carries only
 * the compact operation record; opening diagnostics fetches GetOnboarding,
 * whose durable append-only event history and failure output survive reloads.
 */
function FailedOnboardingItem({
  op,
  onRetry,
  onRemove,
  removing,
}: {
  op: OnboardingOp;
  onRetry?: (op: OnboardingOp) => void;
  onRemove: (id: string) => void;
  removing: boolean;
}) {
  const { t } = useTranslation();
  const [showDiagnostics, setShowDiagnostics] = useState(false);
  const diagnostics = useOnboardingQuery(showDiagnostics ? op.id : null);
  const detail = diagnostics.data?.op?.failureDetail;
  const events = diagnostics.data?.events ?? [];

  return (
    <li className="rounded-control border border-app-warning/30 bg-app-background/50 p-2 text-xs text-app-muted-foreground">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <span>{t(strings.fleet.onboard.savedFailureDetail, { name: op.nodeName || op.host, host: op.host })}</span>
        <span className="flex flex-wrap gap-2">
          <Button type="button" size="sm" variant="secondary" data-testid={selectors.fleet.onboardViewLogs({ id: op.id })} onClick={() => setShowDiagnostics((open) => !open)}>
            {t(showDiagnostics ? strings.fleet.onboard.hideLogs : strings.fleet.onboard.viewLogs)}
          </Button>
          <Button type="button" size="sm" variant="secondary" data-testid={selectors.fleet.onboardRetry({ id: op.id })} onClick={() => onRetry?.(op)}>
            {t(strings.fleet.onboard.retrySaved)}
          </Button>
          <Button type="button" size="sm" variant="secondary" data-testid={selectors.fleet.onboardRemove({ id: op.id })} onClick={() => onRemove(op.id)} disabled={removing}>
            {t(strings.fleet.onboard.removeSaved)}
          </Button>
        </span>
      </div>
      {showDiagnostics && (
        <div data-testid={selectors.fleet.onboard.failureOutput} className="mt-2 border-t border-app-warning/30 pt-2">
          {diagnostics.isLoading && <p>{t(strings.fleet.onboard.loadingLogs)}</p>}
          {diagnostics.error && <p className="text-app-danger">{errorMessage(diagnostics.error, t)}</p>}
          {diagnostics.data && (
            <>
              {detail && <pre className="max-h-64 overflow-auto whitespace-pre-wrap break-words rounded-control bg-app-surface p-2 font-mono text-[0.65rem] leading-relaxed">{detail}</pre>}
              {events.length > 0 && (
                <ol className="mt-2 flex max-h-64 flex-col gap-1 overflow-auto font-mono text-[0.65rem] leading-relaxed">
                  {events.map((event) => (
                    <li key={`${event.sequence}-${event.stepId}`}>
                      {event.stepId} — {t(ONBOARDING_STEP_STATUS_KEY[event.status])}{event.detail ? ` — ${event.detail}` : ""}
                    </li>
                  ))}
                </ol>
              )}
              {!detail && events.length === 0 && <p>{t(strings.fleet.onboard.noLogs)}</p>}
            </>
          )}
        </div>
      )}
    </li>
  );
}

/**
 * FleetPanel is the fleet-dashboard surface (OT-P0-001/003, OT-P1-005): the
 * owner's trusted nodes with their live presence, OS / arch / version / health
 * (status conveyed by icon + text, never color alone), live per-node job status
 * from the scheduler, and an atomic revoke. Loading / error / empty are handled
 * explicitly. Pairing lives in the sibling `PairNodeForm`; run history lives in
 * the runs feature.
 *
 * `onAddNode` is invoked by the primary "Add a node" call to action — the
 * welcoming empty-state card when the fleet is empty, and the header button once
 * it has nodes. The dashboard wires it to scroll to and focus the Add-a-node
 * wizard; when omitted (isolated renders) the button is inert.
 */
export function FleetPanel({
  onAddNode,
  onRetryOnboarding,
}: {
  onAddNode?: () => void;
  onRetryOnboarding?: (op: OnboardingOp) => void;
} = {}) {
  const { t } = useTranslation();
  const nodesQuery = useNodesQuery();
  const queueQuery = useFleetQueueQuery();
  const failedOnboardingsQuery = useFailedOnboardingsQuery();
  const readinessQuery = useBridgeReadinessQuery();
  const firewallAction = useBridgeFirewallActionMutation();
  const readinessCandidate = readinessQuery.data?.last_candidate;
  const readinessCandidateIP = readinessCandidate?.source_ip;
  const removeFailedOnboarding = useRemoveFailedOnboardingMutation();
  const revoke = useRevokeNodeMutation();
  const [selectedNode, setSelectedNode] = useState<Node | null>(null);
  const hasNodes = (nodesQuery.data?.length ?? 0) > 0;

  const handleRevoke = (node: Node) => {
    const name = node.name || node.id;
    if (window.confirm(t(strings.fleet.revokeConfirm, { name }))) {
      revoke.mutate(node.id);
    }
  };

  const unknown = t(strings.fleet.unknownValue);

  return (
    <section
      data-testid={selectors.fleet.panel}
      aria-labelledby="fleet-heading"
      className="rounded-panel border border-app-border bg-app-surface p-4"
    >
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="min-w-0">
          <h3 id="fleet-heading" className="text-sm font-semibold text-app-foreground">
            {t(strings.fleet.title)}
          </h3>
          <p className="mt-1 text-xs text-app-muted-foreground">{t(strings.fleet.description)}</p>
        </div>
        {hasNodes && (
          <Button
            type="button"
            size="sm"
            data-testid={selectors.fleet.onboard.addNode}
            onClick={onAddNode}
          >
            <Plus aria-hidden="true" className="mr-1 h-4 w-4" />
            {t(strings.fleet.onboard.addNode)}
          </Button>
        )}
      </div>

      {readinessQuery.data && (
        <section data-testid={selectors.fleet.bridgeReadiness} className="mt-3 rounded-control border border-app-border bg-app-background p-3">
          <h4 className="text-sm font-semibold text-app-foreground">{t(strings.fleet.bridgeReadinessHeading)}</h4>
          <p className={readinessQuery.data.status === "ready" ? "mt-1 text-xs text-app-success" : "mt-1 text-xs text-app-danger"}>
            {t(readinessQuery.data.status === "ready" ? strings.fleet.bridgeReadinessReady : readinessQuery.data.status === "candidate_blocked" ? strings.fleet.bridgeReadinessCandidateBlocked : strings.fleet.bridgeReadinessNotReady)}
          </p>
          <p className="mt-1 break-all text-xs text-app-muted-foreground">
            {t(strings.fleet.bridgeReadinessEndpoint, { endpoint: readinessQuery.data.endpoint, source: readinessQuery.data.endpoint_source })}
          </p>
          <p className="mt-1 text-xs text-app-muted-foreground">
            {t(strings.fleet.bridgeReadinessMode, { mode: readinessQuery.data.reachability_mode })}
          </p>
          {readinessQuery.data.firewall ? (
            <p className="mt-1 text-xs text-app-muted-foreground">
              {t(strings.fleet.bridgeReadinessFirewall, { state: readinessQuery.data.firewall.broker_available ? readinessQuery.data.firewall.broker_status ?? "available" : "unavailable" })}
            </p>
          ) : null}
          {readinessQuery.data.status === "candidate_blocked" && readinessCandidate && readinessCandidateIP ? (
            <div className="mt-2 rounded-control border border-app-warning/40 bg-app-warning/10 p-2 text-xs text-app-foreground">
              <p>{t(strings.fleet.bridgeReadinessRemediation, { host: readinessCandidate.host, source: readinessCandidateIP })}</p>
              {readinessQuery.data.firewall?.broker_available ? (
                <div className="mt-2 flex flex-wrap gap-2">
                  <Button size="sm" type="button" variant="secondary" disabled={firewallAction.isPending} onClick={() => firewallAction.mutate({ action: "preview", candidateIP: readinessCandidateIP, confirm: false })}>{t(strings.fleet.bridgeReadinessPreview)}</Button>
                  <Button size="sm" type="button" disabled={firewallAction.isPending} onClick={() => firewallAction.mutate({ action: "verify", candidateIP: readinessCandidateIP, confirm: false })}>{t(strings.fleet.bridgeReadinessVerify)}</Button>
                  <Button size="sm" type="button" disabled={firewallAction.isPending} onClick={() => {
                    if (window.confirm(t(strings.fleet.bridgeReadinessAllowConfirm, { source: readinessCandidateIP }))) firewallAction.mutate({ action: "allow", candidateIP: readinessCandidateIP, confirm: true });
                  }}>{t(strings.fleet.bridgeReadinessAllow)}</Button>
                  {readinessQuery.data.firewall.rule_found ? <Button size="sm" type="button" variant="secondary" disabled={firewallAction.isPending} onClick={() => {
                    if (window.confirm(t(strings.fleet.bridgeReadinessRevokeConfirm, { source: readinessCandidateIP }))) firewallAction.mutate({ action: "revoke", candidateIP: readinessCandidateIP, confirm: true });
                  }}>{t(strings.fleet.bridgeReadinessRevoke)}</Button> : null}
                </div>
              ) : <p className="mt-1 text-app-muted-foreground">{t(strings.fleet.bridgeReadinessBrokerUnavailable)}</p>}
              {firewallAction.data ? <p className="mt-1 text-app-muted-foreground">{t(strings.fleet.bridgeReadinessActionResult, { status: firewallAction.data.status })}</p> : null}
              {firewallAction.error ? <p className="mt-1 text-app-danger">{errorMessage(firewallAction.error, t)}</p> : null}
            </div>
          ) : null}
        </section>
      )}

      <PendingPairingPanel />

      {nodesQuery.isLoading && (
        <p data-testid={selectors.fleet.loading} className="mt-3 text-sm text-app-muted-foreground">
          {t(strings.fleet.loading)}
        </p>
      )}

      {nodesQuery.error && (
        <p data-testid={selectors.fleet.error} className="mt-3 text-sm text-app-danger">
          {errorMessage(nodesQuery.error, t)}
        </p>
      )}

      {nodesQuery.data && nodesQuery.data.length === 0 && (
        <div
          data-testid={selectors.fleet.empty}
          className="mt-3 flex flex-col items-center gap-3 rounded-panel border border-dashed border-app-border bg-app-background p-8 text-center"
        >
          <span className="flex h-12 w-12 items-center justify-center rounded-pill bg-app-surface-muted">
            <ServerCog aria-hidden="true" className="h-6 w-6 text-app-muted-foreground" />
          </span>
          <div className="flex flex-col gap-1">
            <p className="text-base font-semibold text-app-foreground">{t(strings.fleet.emptyHeading)}</p>
            <p className="max-w-sm text-sm text-app-muted-foreground">{t(strings.fleet.empty)}</p>
          </div>
          <Button type="button" data-testid={selectors.fleet.onboard.addNode} onClick={onAddNode}>
            <Plus aria-hidden="true" className="mr-1 h-4 w-4" />
            {t(strings.fleet.onboard.addNode)}
          </Button>
        </div>
      )}

      {nodesQuery.data && nodesQuery.data.length > 0 && (
        <ul data-testid={selectors.fleet.list} className="mt-3 flex flex-col gap-2">
          {nodesQuery.data.map((node) => {
            const StatusIcon = STATUS_ICON[node.status];
            return (
              <li
                key={node.id}
                data-testid={selectors.fleet.row({ id: node.id })}
                className="flex flex-wrap items-start justify-between gap-3 rounded-panel border border-app-border bg-app-background p-3"
              >
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <span
                      className={["inline-block h-2 w-2 shrink-0 rounded-pill", dotClass(node)].join(" ")}
                      role="img"
                      aria-label={node.online ? t(strings.fleet.onlineLabel) : t(strings.fleet.offlineLabel)}
                    />
                    <span className="truncate text-sm font-medium text-app-foreground">
                      {node.name || node.id}
                    </span>
                    <span className="inline-flex items-center gap-1 text-xs text-app-muted-foreground">
                      <StatusIcon aria-hidden="true" className="h-3.5 w-3.5" />
                      {t(STATUS_LABEL[node.status])}
                    </span>
                  </div>

                  <dl className="mt-2 grid grid-cols-2 gap-x-4 gap-y-1 sm:grid-cols-4">
                    <MetaField label={t(strings.fleet.osLabel)} value={node.os || unknown} />
                    <MetaField label={t(strings.fleet.archLabel)} value={node.arch || unknown} />
                    <MetaField
                      label={t(strings.fleet.versionLabel)}
                      value={node.revision ? formatRevision(node.revision) : unknown}
                    />
                    <MetaField label={t(strings.fleet.healthLabel)} value={t(STATUS_LABEL[node.status])} />
                  </dl>

                  <dl className="mt-2 grid grid-cols-2 gap-x-4 gap-y-1 sm:grid-cols-5">
                    <MetaField label={t(strings.fleet.readiness.registry)} value={node.registryRecordPresent ? t(strings.fleet.readiness.present) : t(strings.fleet.readiness.missing)} />
                    <MetaField label={t(strings.fleet.readiness.heartbeat)} value={node.heartbeatFresh ? t(strings.fleet.readiness.fresh, { age: node.heartbeatAgeSeconds }) : t(strings.fleet.readiness.stale, { age: node.heartbeatAgeSeconds })} />
                    <MetaField label={t(strings.fleet.readiness.channel)} value={node.channelHeld ? t(strings.fleet.readiness.held) : t(strings.fleet.readiness.absent)} />
                    <MetaField label={t(strings.fleet.readiness.protocol)} value={node.protocolCompatible ? t(strings.fleet.readiness.compatible) : t(strings.fleet.readiness.updateRequired)} />
                    <MetaField label={t(strings.fleet.readiness.dispatch)} value={node.dispatchable ? t(strings.fleet.readiness.ready) : t(strings.fleet.readiness.blocked)} />
                  </dl>

                  <p className="mt-1 text-xs text-app-muted-foreground">
                    {node.lastSeenAt
                      ? formatDate(timestampDate(node.lastSeenAt), { dateStyle: "short", timeStyle: "short" })
                      : t(strings.fleet.neverSeen)}
                  </p>

                  {node.configurationState && (
                    <p data-testid={`fleet-node-configuration-${node.id}`} className="mt-1 text-xs text-app-muted-foreground">
                      <span className="font-medium text-app-foreground">Configuration:</span> {node.configurationState}
                      {node.configurationUnmet.length > 0 && ` · Unmet: ${node.configurationUnmet.join(", ")}`}
                    </p>
                  )}

                  <JobStatus nodeId={node.id} queue={queueQuery.data?.get(node.id)} />
                </div>

                <div className="flex shrink-0 gap-2">
                  <Button type="button" size="sm" variant="secondary" onClick={() => setSelectedNode(node)}>{t(strings.fleet.management.details)}</Button>
                  {node.status !== NodeStatus.REVOKED && (
                    <Button data-testid={selectors.fleet.revoke({ id: node.id })} size="sm" variant="secondary" onClick={() => handleRevoke(node)} disabled={revoke.isPending} aria-label={t(strings.fleet.revoke)}><ShieldOff aria-hidden="true" className="h-4 w-4" /></Button>
                  )}
                </div>
              </li>
            );
          })}
        </ul>
      )}

      {selectedNode && <div data-testid={selectors.fleet.management} className="mt-4"><NodeManagementPanel node={selectedNode} onClose={() => setSelectedNode(null)} /></div>}

      {failedOnboardingsQuery.data && failedOnboardingsQuery.data.length > 0 && (
        <section data-testid={selectors.fleet.failedOnboardings} className="mt-3 rounded-panel border border-app-warning/40 bg-app-warning/10 p-3">
          <h4 className="text-sm font-semibold text-app-foreground">{t(strings.fleet.onboard.savedFailuresHeading)}</h4>
          <ul className="mt-2 flex flex-col gap-2">
            {failedOnboardingsQuery.data.map((op) => (
              <FailedOnboardingItem key={op.id} op={op} onRetry={onRetryOnboarding} onRemove={(id) => removeFailedOnboarding.mutate(id)} removing={removeFailedOnboarding.isPending} />
            ))}
          </ul>
        </section>
      )}
    </section>
  );
}
