import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import { useApplyMachinePolicyMutation, useArchiveMachineMutation, useMachineDetailQuery, useMachinesQuery, useMachineTrustQuery, useRemoveMachineMutation, useRequestMachineCleanupMutation, useReviewMachineHostKeyMutation, useRevokeMachineNodeMutation } from "./queries";

const POLICY_PROFILES = ["managed-connection", "presence", "deployment-target", "production-runtime", "development-runner", "custom"] as const;

/**
 * Operator controls for durable Machine intent. These never infer that an
 * archive or deletion has revoked a Node or cleaned SSH access: each effect is
 * a separate, auditable request made deliberately by the owner.
 */
export function MachineLifecyclePanel() {
  const { t } = useTranslation();
  const [selectedID, setSelectedID] = useState<string | null>(null);
  const [profileID, setProfileID] = useState("managed-connection");
  const [reason, setReason] = useState("");
  const [confirmRemoval, setConfirmRemoval] = useState(false);
  const [replacementFingerprint, setReplacementFingerprint] = useState("");
  const machines = useMachinesQuery();
  const detail = useMachineDetailQuery(selectedID);
  const trust = useMachineTrustQuery(selectedID);
  const archive = useArchiveMachineMutation();
  const remove = useRemoveMachineMutation();
  const revoke = useRevokeMachineNodeMutation();
  const cleanup = useRequestMachineCleanupMutation();
  const policy = useApplyMachinePolicyMutation();
  const reviewHostKey = useReviewMachineHostKeyMutation();
  const busy = archive.isPending || remove.isPending || revoke.isPending || cleanup.isPending || policy.isPending || reviewHostKey.isPending;
  const error = archive.error ?? remove.error ?? revoke.error ?? cleanup.error ?? policy.error ?? reviewHostKey.error ?? detail.error ?? trust.error;
  const selectedMachine = machines.data?.find((machine) => machine.id === selectedID);

  return <section data-testid={selectors.fleet.machineLifecycle} aria-labelledby="machine-lifecycle-heading" className="rounded-panel border border-app-border bg-app-surface p-4">
    <h3 id="machine-lifecycle-heading" className="text-sm font-semibold text-app-foreground">{t(strings.fleet.machines.heading)}</h3>
    <p className="mt-1 text-xs text-app-muted-foreground">{t(strings.fleet.machines.description)}</p>
    {machines.isLoading && <p className="mt-3 text-xs text-app-muted-foreground">{t(strings.fleet.machines.loading)}</p>}
    {(machines.error || error) && <p role="alert" className="mt-3 text-xs text-app-danger">{errorMessage(machines.error ?? error, t)}</p>}
    {machines.data?.length === 0 && <p className="mt-3 text-xs text-app-muted-foreground">{t(strings.fleet.machines.empty)}</p>}
    <ul className="mt-3 flex flex-col gap-2">
      {machines.data?.map((machine) => <li key={machine.id} data-testid={selectors.fleet.machineLifecycleRow({ id: machine.id })} className="rounded-control border border-app-border bg-app-background p-3">
        <p className="break-all text-xs font-medium text-app-foreground">{machine.id}</p>
        <p className="mt-1 text-xs text-app-muted-foreground">{t(strings.fleet.machines.state, { state: machine.lifecycle || t(strings.fleet.unknownValue) })}</p>
        <div className="mt-2 flex flex-wrap gap-2">
          <Button size="sm" type="button" variant="outline" disabled={busy} onClick={() => setSelectedID(machine.id)}>{t(strings.fleet.machines.details)}</Button>
          <Button size="sm" type="button" variant="outline" disabled={busy} onClick={() => archive.mutate(machine)}>{t(strings.fleet.machines.archive)}</Button>
          <Button size="sm" type="button" variant="outline" disabled={busy} onClick={() => revoke.mutate(machine.id)}>{t(strings.fleet.machines.revokeNode)}</Button>
          <Button size="sm" type="button" variant="outline" disabled={busy} onClick={() => cleanup.mutate(machine.id)}>{t(strings.fleet.machines.cleanup)}</Button>
          <Button size="sm" type="button" variant="outline" disabled={busy} onClick={() => { if (window.confirm(t(strings.fleet.machines.removeConfirm, { id: machine.id }))) remove.mutate(machine); }}>{t(strings.fleet.machines.remove)}</Button>
        </div>
      </li>)}
    </ul>
    {selectedID && <section data-testid="machine-detail" className="mt-3 rounded-control border border-app-border bg-app-background p-3" aria-live="polite">
      <div className="flex items-center justify-between gap-2"><h4 className="text-sm font-semibold text-app-foreground">{t(strings.fleet.machines.detailHeading, { id: selectedID })}</h4><Button size="sm" type="button" variant="outline" onClick={() => setSelectedID(null)}>{t(strings.fleet.machines.closeDetails)}</Button></div>
      {detail.isLoading && <p className="mt-2 text-xs text-app-muted-foreground">{t(strings.fleet.machines.loading)}</p>}
      {detail.data && <>
        <p className="mt-2 text-xs text-app-muted-foreground">{t(strings.fleet.machines.attempts, { count: detail.data.enrollmentAttempts.length })}</p>
        <p className="mt-1 break-all text-xs text-app-muted-foreground">{detail.data.currentNode ? t(strings.fleet.machines.currentNode, { id: detail.data.currentNode.nodeId, online: detail.data.currentNode.online ? t(strings.fleet.onlineLabel) : t(strings.fleet.offlineLabel) }) : t(strings.fleet.machines.noCurrentNode)}</p>
        <p className="mt-1 text-xs text-app-muted-foreground">{t(strings.fleet.machines.auditEvents, { count: detail.data.auditEvents.length })}</p>
        <p className="mt-1 text-xs text-app-muted-foreground">{t(strings.fleet.machines.cleanupStatus, { statuses: detail.data.cleanupTombstones.length === 0 ? t(strings.fleet.machines.noCleanup) : detail.data.cleanupTombstones.map((cleanup) => cleanup.status).join(", ") })}</p>
        {detail.data.readiness && <p className={detail.data.readiness.ready ? "mt-1 text-xs text-app-success" : "mt-1 text-xs text-app-warning"}>{detail.data.readiness.ready ? t(strings.fleet.machines.ready) : t(strings.fleet.machines.notReady, { reasons: detail.data.readiness.reasons.join(", ") })}</p>}
      </>}
      {trust.data && <div className="mt-3 border-t border-app-border pt-3 text-xs text-app-muted-foreground"><p>{t(strings.fleet.machines.hostTrust, { state: trust.data.hostKeyState || t(strings.fleet.unknownValue) })}</p><p className="mt-1 break-all">{t(strings.fleet.machines.hostFingerprint, { fingerprint: trust.data.hostKeyFingerprint || t(strings.fleet.unknownValue) })}</p><p className="mt-1 break-all">{t(strings.fleet.machines.clientFingerprint, { fingerprint: trust.data.clientKeyFingerprint || t(strings.fleet.unknownValue) })}</p><label className="mt-2 block" htmlFor="machine-host-key">{t(strings.fleet.machines.replacementHostKey)}</label><input id="machine-host-key" value={replacementFingerprint} onChange={(event) => setReplacementFingerprint(event.target.value)} className="mt-1 w-full rounded-control border border-app-border bg-app-surface px-2 py-1 text-app-foreground"/><Button className="mt-2" size="sm" type="button" variant="outline" disabled={busy || replacementFingerprint.trim() === ""} onClick={() => reviewHostKey.mutate({ machineID: selectedID, fingerprint: replacementFingerprint.trim() })}>{t(strings.fleet.machines.reviewHostKey)}</Button></div>}
      {selectedMachine && <form className="mt-3 border-t border-app-border pt-3" onSubmit={(event) => { event.preventDefault(); policy.mutate({ machine: selectedMachine, profileID, reason, confirmRemoval }); }}><label className="block text-xs text-app-muted-foreground" htmlFor="machine-policy">{t(strings.fleet.machines.policy)}</label><select id="machine-policy" value={profileID} onChange={(event) => setProfileID(event.target.value)} className="mt-1 w-full rounded-control border border-app-border bg-app-surface px-2 py-1 text-app-foreground">{POLICY_PROFILES.map((profile) => <option key={profile} value={profile}>{profile}</option>)}</select><label className="mt-2 block text-xs text-app-muted-foreground" htmlFor="machine-policy-reason">{t(strings.fleet.machines.policyReason)}</label><input id="machine-policy-reason" value={reason} onChange={(event) => setReason(event.target.value)} className="mt-1 w-full rounded-control border border-app-border bg-app-surface px-2 py-1 text-app-foreground"/><label className="mt-2 flex items-center gap-2 text-xs text-app-muted-foreground" htmlFor="machine-confirm-removal"><input id="machine-confirm-removal" type="checkbox" checked={confirmRemoval} onChange={(event) => setConfirmRemoval(event.target.checked)}/>{t(strings.fleet.machines.confirmRemoval)}</label><Button className="mt-2" size="sm" type="submit" disabled={busy}>{t(strings.fleet.machines.applyPolicy)}</Button></form>}
    </section>}
  </section>;
}
import { useState } from "react";
