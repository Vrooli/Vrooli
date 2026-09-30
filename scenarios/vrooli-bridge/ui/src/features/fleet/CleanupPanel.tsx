import { useMemo, useState } from "react";

import { Button } from "@vrooli/react-component-library/Button/2";
import { Input } from "@vrooli/react-component-library/Input/1";
import { CleanupStatus } from "@vrooli/proto-types/vrooli-bridge/v1/cleanup/cleanup_pb";
import { sealCleanupPassphrase } from "../../api/cleanup";
import { errorMessage } from "../../lib/errorMessage";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";
import {
  useCleanupQuery,
  useConfirmCleanupMutation,
  useMachinesQuery,
  useNodesQuery,
  useStartCleanupMutation,
} from "./queries";

function statusLabel(status: CleanupStatus): string {
  const labels: Record<CleanupStatus, string> = {
    [CleanupStatus.UNSPECIFIED]: "unspecified",
    [CleanupStatus.QUEUED]: "queued",
    [CleanupStatus.PLANNING]: "planning",
    [CleanupStatus.PLANNED]: "planned",
    [CleanupStatus.CONFIRMED]: "confirmed",
    [CleanupStatus.APPLYING]: "applying",
    [CleanupStatus.COMPLETED]: "completed",
    [CleanupStatus.FAILED]: "failed",
    [CleanupStatus.BLOCKED]: "blocked",
    [CleanupStatus.CANCELED]: "canceled",
  };
  return labels[status];
}

function decodeJSON(value: Uint8Array): string {
  if (value.length === 0) return "";
  return new TextDecoder().decode(value);
}

type CleanupPlanView = {
  remove: unknown[];
  keep: unknown[];
  cannotAttribute: unknown[];
};

function decodePlan(value: Uint8Array): CleanupPlanView | null {
  try {
    const parsed: unknown = JSON.parse(decodeJSON(value));
    if (typeof parsed !== "object" || parsed === null) return null;
    const record = parsed as Record<string, unknown>;
    return {
      remove: Array.isArray(record.remove) ? record.remove : [],
      keep: Array.isArray(record.keep) ? record.keep : [],
      cannotAttribute: Array.isArray(record.cannot_attribute) ? record.cannot_attribute : [],
    };
  } catch {
    return null;
  }
}

function PlanSection({ title, emptyLabel, entries }: { title: string; emptyLabel: string; entries: unknown[] }) {
  return (
    <div className="mt-2">
      <p className="text-xs font-semibold text-app-foreground">{title} ({entries.length})</p>
      {entries.length === 0 ? (
        <p className="mt-1 text-[0.65rem] text-app-muted-foreground">{emptyLabel}</p>
      ) : (
        <ul className="mt-1 space-y-1">
          {entries.map((entry, index) => (
            <li key={`${title}-${index}`} className="rounded-control bg-app-background p-2 text-[0.65rem] text-app-foreground">
              <pre className="whitespace-pre-wrap break-words">{JSON.stringify(entry, null, 2)}</pre>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

/**
 * Bridge's owner-facing cleanup view. Planning is always available as a
 * read-only action; applying requires the exact target, the frozen plan hash,
 * and a passphrase sealed in this browser to the node's published key. The
 * plaintext passphrase is never put in a request body or durable state.
 */
export function CleanupPanel() {
  const { t } = useTranslation();
  const nodesQuery = useNodesQuery();
  const machinesQuery = useMachinesQuery();
  const start = useStartCleanupMutation();
  const confirm = useConfirmCleanupMutation();
  const [operationID, setOperationID] = useState<string | null>(null);
  const [passphrase, setPassphrase] = useState("");
  const [sealError, setSealError] = useState<string | null>(null);
  const operationQuery = useCleanupQuery(operationID);
  const operation = operationQuery.data?.operation;
  const plan = useMemo(() => decodePlan(operation?.planJson ?? new Uint8Array()), [operation?.planJson]);

  const targets = useMemo(() => {
    const result = new Map<string, { machineID: string; target: string }>();
    for (const machine of machinesQuery.data ?? []) {
      for (const lineage of machine.nodeLineage) {
        if (!lineage.current) continue;
        const locator = machine.locators.find((candidate) => candidate.kind === "hostname") ?? machine.locators[0];
        result.set(lineage.nodeId, { machineID: machine.id, target: locator?.value || lineage.nodeId });
      }
    }
    return result;
  }, [machinesQuery.data]);

  const startCleanup = (nodeID: string) => {
    const target = targets.get(nodeID);
    const node = nodesQuery.data?.find((candidate) => candidate.id === nodeID);
    if (!target || !node) return;
    start.mutate({ machineId: target.machineID, nodeId: nodeID, target: target.target || node.name, scope: "all" }, {
      onSuccess: (response) => {
        const id = response.operation?.id;
        if (id) setOperationID(id);
      },
    });
  };

  const confirmCleanup = async () => {
    if (!operation || passphrase.trim() === "") return;
    try {
      const aad = [
        "vrooli-cleanup-context-v1",
        operation.machineId,
        operation.nodeId,
        operation.target,
        operation.scope,
        operation.planHash,
        operation.id,
        operation.operatorId,
      ];
      const sealed = await sealCleanupPassphrase(operation.sealingPublicKey, passphrase, aad);
      setSealError(null);
      setPassphrase("");
      confirm.mutate({ id: operation.id, target: operation.target, planHash: operation.planHash, sealedPassphrase: sealed, capability: new Uint8Array(), operatorId: operation.operatorId });
    } catch (error) {
      setSealError(error instanceof Error ? error.message : "could not seal the passphrase to the node");
      setPassphrase("");
    }
  };

  return (
    <section className="rounded-panel border border-app-border bg-app-surface p-4" aria-labelledby="cleanup-heading">
      <h3 id="cleanup-heading" className="text-sm font-semibold text-app-foreground">Managed cleanup</h3>
      <p className="mt-1 text-xs text-app-muted-foreground">Read-only inventory and frozen-plan review come first. Apply removes only artifacts attributed to Vrooli; the passphrase is sealed locally to the node.</p>
      <div className="mt-3 flex flex-col gap-2">
        {(nodesQuery.data ?? []).map((node) => {
          const target = targets.get(node.id);
          return (
            <div key={node.id} className="flex flex-wrap items-center justify-between gap-2 rounded-control border border-app-border bg-app-background p-2">
              <span className="text-xs text-app-foreground">{node.name || node.id}</span>
              <Button type="button" size="sm" variant="secondary" onClick={() => startCleanup(node.id)} disabled={!target || start.isPending}>
                Preview cleanup
              </Button>
            </div>
          );
        })}
      </div>
      {start.error && <p className="mt-2 text-xs text-app-danger">{errorMessage(start.error, t)}</p>}
      {operationQuery.error && <p className="mt-2 text-xs text-app-danger">{errorMessage(operationQuery.error, t)}</p>}
      {operation && (
        <div className="mt-3 rounded-control border border-app-warning/40 bg-app-warning/10 p-3">
          <p className="text-xs text-app-foreground">Operation {operation.id} — {statusLabel(operation.status)} — target {operation.target}</p>
          {operation.planHash && <p className="mt-1 break-all font-mono text-[0.65rem] text-app-muted-foreground">plan hash: {operation.planHash}</p>}
          {plan ? (
            <div className="mt-2 max-h-96 overflow-auto rounded-control bg-app-background p-2">
              <PlanSection title={t(strings.fleet.machines.cleanupPlanRemove)} emptyLabel={t(strings.fleet.machines.cleanupPlanEmpty)} entries={plan.remove} />
              <PlanSection title={t(strings.fleet.machines.cleanupPlanKeep)} emptyLabel={t(strings.fleet.machines.cleanupPlanEmpty)} entries={plan.keep} />
              <PlanSection title={t(strings.fleet.machines.cleanupPlanCannotAttribute)} emptyLabel={t(strings.fleet.machines.cleanupPlanEmpty)} entries={plan.cannotAttribute} />
            </div>
          ) : operation.planJson.length > 0 ? (
            <pre className="mt-2 max-h-64 overflow-auto whitespace-pre-wrap break-words rounded-control bg-app-background p-2 text-[0.65rem] text-app-foreground">{decodeJSON(operation.planJson)}</pre>
          ) : null}
          {operation.status === CleanupStatus.PLANNED && (
            <div className="mt-3 flex flex-col gap-2">
              <label className="text-xs text-app-foreground" htmlFor="cleanup-passphrase">Break-glass passphrase</label>
              <Input id="cleanup-passphrase" type="password" value={passphrase} onChange={(event) => setPassphrase(event.target.value)} autoComplete="off" />
              <Button type="button" size="sm" onClick={() => void confirmCleanup()} disabled={passphrase.trim() === "" || confirm.isPending}>Confirm and apply frozen plan</Button>
            </div>
          )}
          {sealError && <p className="mt-2 text-xs text-app-danger">{sealError}</p>}
          {confirm.error && <p className="mt-2 text-xs text-app-danger">{errorMessage(confirm.error, t)}</p>}
          {operation.receiptJson.length > 0 && <pre className="mt-2 max-h-64 overflow-auto whitespace-pre-wrap break-words rounded-control bg-app-background p-2 text-[0.65rem] text-app-foreground">{decodeJSON(operation.receiptJson)}</pre>}
        </div>
      )}
      <p className="mt-3 text-[0.65rem] text-app-muted-foreground">{t(strings.fleet.description)}</p>
    </section>
  );
}
