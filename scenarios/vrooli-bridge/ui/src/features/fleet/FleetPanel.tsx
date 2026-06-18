import { timestampDate } from "@bufbuild/protobuf/wkt";
import { ShieldOff } from "lucide-react";

import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { formatDate } from "../../i18n/format";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import { NodeStatus, type Node } from "../../api/nodes";
import { useNodesQuery, useRevokeNodeMutation } from "./queries";

const STATUS_LABEL = {
  [NodeStatus.UNSPECIFIED]: strings.fleet.status.unspecified,
  [NodeStatus.OFFLINE]: strings.fleet.status.offline,
  [NodeStatus.ONLINE]: strings.fleet.status.online,
  [NodeStatus.NEEDS_UPDATE]: strings.fleet.status.needsUpdate,
  [NodeStatus.REVOKED]: strings.fleet.status.revoked,
} as const satisfies Record<NodeStatus, string>;

// Presence is conveyed by BOTH a colored dot and a text label so it never
// depends on color alone (WCAG 1.4.1).
function dotClass(node: Node): string {
  if (node.status === NodeStatus.REVOKED) return "bg-app-danger";
  if (node.status === NodeStatus.NEEDS_UPDATE) return "bg-app-warning";
  return node.online ? "bg-app-success" : "bg-app-muted-foreground";
}

/**
 * FleetPanel is the Phase-1 fleet surface (OT-P0-001/003): the owner's trusted
 * nodes with their live presence (online/offline + status), OS/arch, current
 * revision, last-seen, and an atomic revoke. It handles loading / error / empty
 * explicitly. Phase 5 grows this into the full dashboard (pairing, live job
 * output, run history).
 */
export function FleetPanel() {
  const { t } = useTranslation();
  const nodesQuery = useNodesQuery();
  const revoke = useRevokeNodeMutation();

  const handleRevoke = (node: Node) => {
    const name = node.name || node.id;
    if (window.confirm(t(strings.fleet.revokeConfirm, { name }))) {
      revoke.mutate(node.id);
    }
  };

  return (
    <section
      data-testid={selectors.fleet.panel}
      aria-labelledby="fleet-heading"
      className="rounded-panel border border-app-border bg-app-surface p-4"
    >
      <h3 id="fleet-heading" className="text-sm font-semibold text-app-foreground">
        {t(strings.fleet.title)}
      </h3>
      <p className="mt-1 text-xs text-app-muted-foreground">{t(strings.fleet.description)}</p>

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
        <p data-testid={selectors.fleet.empty} className="mt-3 text-sm text-app-muted-foreground">
          {t(strings.fleet.empty)}
        </p>
      )}

      {nodesQuery.data && nodesQuery.data.length > 0 && (
        <ul data-testid={selectors.fleet.list} className="mt-3 flex flex-col gap-2">
          {nodesQuery.data.map((node) => (
            <li
              key={node.id}
              data-testid={selectors.fleet.row({ id: node.id })}
              className="flex flex-wrap items-center justify-between gap-3 rounded-panel border border-app-border bg-app-background p-3"
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
                  <span className="text-xs text-app-muted-foreground">
                    {t(STATUS_LABEL[node.status])}
                  </span>
                </div>
                <p className="mt-1 text-xs text-app-muted-foreground">
                  {node.os || "?"}/{node.arch || "?"}
                  {node.revision ? ` · ${node.revision.slice(0, 10)}` : ""}
                  {node.lastSeenAt
                    ? ` · ${formatDate(timestampDate(node.lastSeenAt), { dateStyle: "short", timeStyle: "short" })}`
                    : ` · ${t(strings.fleet.neverSeen)}`}
                </p>
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
          ))}
        </ul>
      )}
    </section>
  );
}
