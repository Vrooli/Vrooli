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

import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { formatDate } from "../../i18n/format";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import { NodeStatus, type Node } from "../../api/nodes";
import { type NodeQueue } from "../../api/queue";
import { useFleetQueueQuery, useNodesQuery, useRevokeNodeMutation } from "./queries";

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
export function FleetPanel({ onAddNode }: { onAddNode?: () => void } = {}) {
  const { t } = useTranslation();
  const nodesQuery = useNodesQuery();
  const queueQuery = useFleetQueueQuery();
  const revoke = useRevokeNodeMutation();
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

                  <p className="mt-1 text-xs text-app-muted-foreground">
                    {node.lastSeenAt
                      ? formatDate(timestampDate(node.lastSeenAt), { dateStyle: "short", timeStyle: "short" })
                      : t(strings.fleet.neverSeen)}
                  </p>

                  <JobStatus nodeId={node.id} queue={queueQuery.data?.get(node.id)} />
                </div>

                {node.status !== NodeStatus.REVOKED && (
                  <Button
                    data-testid={selectors.fleet.revoke({ id: node.id })}
                    size="sm"
                    variant="outline"
                    onClick={() => handleRevoke(node)}
                    disabled={revoke.isPending}
                    aria-label={t(strings.fleet.revoke)}
                  >
                    <ShieldOff aria-hidden="true" className="h-4 w-4" />
                  </Button>
                )}
              </li>
            );
          })}
        </ul>
      )}
    </section>
  );
}
