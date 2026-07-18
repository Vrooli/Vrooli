import { useEffect, useState } from "react";
import { ShieldOff, X } from "lucide-react";

import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { NodeStatus, type Node } from "../../api/nodes";
import { useRemoveNodeMutation, useRevokeNodeMutation, useUpdateNodeMutation } from "./queries";

const split = (value: string) => value.split(",").map((part) => part.trim()).filter(Boolean);

/** Contextual control surface for one trusted node. Destructive actions stay
 * explicit: revocation severs trust; deleting a separate Machine record is a
 * deliberate lifecycle operation in the machine-management area. */
export function NodeManagementPanel({ node, onClose }: { node: Node; onClose: () => void }) {
  const { t } = useTranslation();
  const update = useUpdateNodeMutation();
  const revoke = useRevokeNodeMutation();
  const removeNode = useRemoveNodeMutation();
  const [name, setName] = useState(node.name);
  const [endpoint, setEndpoint] = useState(node.endpoint);
  const [capabilities, setCapabilities] = useState(node.capabilities.join(", "));
  const [scopes, setScopes] = useState(node.scopes.join(", "));
  const [revision, setRevision] = useState(node.revision);

  useEffect(() => {
    setName(node.name); setEndpoint(node.endpoint); setCapabilities(node.capabilities.join(", "));
    setScopes(node.scopes.join(", ")); setRevision(node.revision);
  }, [node]);

  const save = () => update.mutate({ ...node, name, endpoint, capabilities: split(capabilities), scopes: split(scopes), revision });
  const revokeNode = () => {
    if (window.confirm(t(strings.fleet.revokeConfirm, { name: node.name || node.id }))) revoke.mutate(node.id, { onSuccess: onClose });
  };
  const remove = () => {
    if (window.confirm(t(strings.fleet.management.removeConfirm))) removeNode.mutate(node.id, { onSuccess: onClose });
  };

  return <aside aria-label={t(strings.fleet.management.heading, { name: node.name || node.id })} className="rounded-sheet border border-app-border bg-app-surface-raised p-5 shadow-lg">
    <div className="flex items-start justify-between gap-4">
      <div><p className="text-xs font-medium uppercase tracking-wide text-app-muted-foreground">{node.os || t(strings.fleet.unknownValue)} · {node.arch || t(strings.fleet.unknownValue)}</p><h3 className="mt-1 text-lg font-semibold">{t(strings.fleet.management.heading, { name: node.name || node.id })}</h3><p className="mt-1 text-sm text-app-muted-foreground">{t(strings.fleet.management.overview)}</p></div>
      <Button type="button" size="sm" variant="outline" onClick={onClose} aria-label={t(strings.fleet.management.close)}><X className="h-4 w-4" /></Button>
    </div>
    <div className="mt-5 grid gap-3 sm:grid-cols-2">
      <label className="text-xs text-app-muted-foreground">{t(strings.fleet.onboard.nameLabel)}<Input className="mt-1" value={name} onChange={(event) => setName(event.target.value)} /></label>
      <label className="text-xs text-app-muted-foreground">{t(strings.fleet.management.endpoint)}<Input className="mt-1" value={endpoint} onChange={(event) => setEndpoint(event.target.value)} /></label>
      <label className="text-xs text-app-muted-foreground">{t(strings.fleet.onboard.capabilitiesLabel)}<Input className="mt-1" value={capabilities} onChange={(event) => setCapabilities(event.target.value)} /></label>
      <label className="text-xs text-app-muted-foreground">{t(strings.fleet.management.scopes)}<Input className="mt-1" value={scopes} onChange={(event) => setScopes(event.target.value)} /></label>
      <label className="text-xs text-app-muted-foreground sm:col-span-2">{t(strings.fleet.onboard.revisionLabel)}<Input className="mt-1" value={revision} onChange={(event) => setRevision(event.target.value)} /></label>
    </div>
    <div className="mt-4 flex flex-wrap gap-2"><Button type="button" onClick={save} disabled={update.isPending}>{t(update.isPending ? strings.fleet.management.saving : strings.fleet.management.save)}</Button></div>
    <div className="mt-6 border-t border-app-border pt-4"><h4 className="text-sm font-semibold">{t(strings.fleet.management.revokeHeading)}</h4><p className="mt-1 max-w-2xl text-xs text-app-muted-foreground">{t(strings.fleet.management.revokeDescription)}</p><Button className="mt-3" type="button" variant="outline" disabled={revoke.isPending} onClick={revokeNode}><ShieldOff className="mr-1 h-4 w-4" />{t(strings.fleet.management.revoke)}</Button></div>
    <div className="mt-5 rounded-control bg-app-surface-muted p-3"><h4 className="text-sm font-semibold">{t(strings.fleet.management.removeHeading)}</h4><p className="mt-1 text-xs text-app-muted-foreground">{t(strings.fleet.management.removeDescription)}</p>{node.status !== NodeStatus.REVOKED ? <p className="mt-2 text-xs text-app-warning">{t(strings.fleet.management.removeBlocked)}</p> : <Button className="mt-3" type="button" variant="outline" disabled={removeNode.isPending} onClick={remove}>{t(strings.fleet.management.removeAction)}</Button>}</div>
  </aside>;
}
