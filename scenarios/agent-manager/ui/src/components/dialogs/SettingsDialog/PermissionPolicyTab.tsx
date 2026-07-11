import { useState } from "react";
import type { MessageShape } from "@bufbuild/protobuf";
import type {
  DoctorPermissionPolicyResponseSchema,
  PermissionPolicyPlanSchema,
} from "@vrooli/proto-types/agent-manager/v1/api/service_pb";
import { Badge } from "../../ui/badge";
import { Button } from "../../ui/button";
import { Card, CardContent } from "../../ui/card";
import { Checkbox } from "../../ui/checkbox";
import type { usePermissionPolicy } from "../../../hooks/useApi";

type PermissionPolicyHook = ReturnType<typeof usePermissionPolicy>;
type PermissionPolicyPlan = MessageShape<typeof PermissionPolicyPlanSchema>;
type DoctorResponse = MessageShape<typeof DoctorPermissionPolicyResponseSchema>;

function PlanEvidence({ plan }: { plan: PermissionPolicyPlan | null }) {
  if (!plan) return null;
  return (
    <Card className="border-border bg-card/40">
      <CardContent className="space-y-3 py-5 text-sm">
        <div className="flex flex-wrap items-center gap-2">
          <h3 className="font-semibold">Resource projection</h3>
          <Badge variant={plan.hardEnforcementSatisfied ? "default" : "destructive"}>
            {plan.hardEnforcementSatisfied ? "Hard enforcement covered" : "Hard enforcement gap"}
          </Badge>
        </div>
        {plan.missingHardEnforcementRuleIds.length > 0 && (
          <p className="text-destructive">Missing hard enforcement: {plan.missingHardEnforcementRuleIds.join(", ")}</p>
        )}
        <div className="space-y-2">
          {plan.resources.map((resource, index) => (
            <div key={`${resource.runnerType}-${resource.scope}-${index}`} className="border-t border-border pt-2 text-xs">
              <div className="flex flex-wrap items-center gap-2">
                <span className="font-medium">{String(resource.runnerType)} / {resource.scope}</span>
                <Badge variant={resource.status === "planned" || resource.status === "reconciled" ? "secondary" : "outline"}>{resource.status}</Badge>
                {resource.enforcement?.permissions && <span className="text-muted-foreground">{resource.enforcement.permissions}</span>}
                {resource.drift && <span className="text-amber-500">drift detected</span>}
              </div>
              {resource.error && <p className="mt-1 text-destructive">{resource.error}</p>}
              {resource.nativePaths.length > 0 && <p className="mt-1 text-muted-foreground">Native targets: {resource.nativePaths.join(", ")}</p>}
              {resource.changes.length > 0 && <ul className="mt-1 list-disc pl-5 text-muted-foreground">{resource.changes.map((change) => <li key={change}>{change}</li>)}</ul>}
              {resource.enforcement?.caveats.map((caveat) => <p key={caveat} className="mt-1 text-muted-foreground">{caveat}</p>)}
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}

export function PermissionPolicyTab({ policy }: { policy: PermissionPolicyHook }) {
  const [busy, setBusy] = useState<string | null>(null);
  const [authorized, setAuthorized] = useState(false);
  const [notice, setNotice] = useState<string | null>(null);
  const [plan, setPlan] = useState<PermissionPolicyPlan | null>(null);
  const [doctor, setDoctor] = useState<DoctorResponse | null>(null);
  const { data, loading, error } = policy;

  const run = async (name: string, operation: () => Promise<void>) => {
    setBusy(name);
    setNotice(null);
    try {
      await operation();
    } catch (err) {
      setNotice((err as Error).message);
    } finally {
      setBusy(null);
    }
  };

  if (loading && !data) return <p className="text-sm text-muted-foreground">Loading permission policy…</p>;
  if (error && !data) return <div className="rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive">{error}</div>;

  const status = data?.status.status;
  const catalog = data?.catalog.catalog;
  const lastReconcile = data?.status.lastReconcile;

  return (
    <div className="space-y-5">
      <Card className="border-border bg-card/40">
        <CardContent className="space-y-3 py-5 text-sm">
          <div className="flex flex-wrap items-center gap-2">
            <Badge variant={status?.ready ? "default" : "destructive"}>{status?.ready ? "Declared state ready" : "Declared state not ready"}</Badge>
            {lastReconcile && <Badge variant={lastReconcile.success ? "secondary" : "destructive"}>{lastReconcile.success ? "Last reconcile succeeded" : "Last reconcile incomplete"}</Badge>}
          </div>
          <p><span className="text-muted-foreground">Catalog path:</span> <span className="font-mono text-xs break-all">{status?.path || "unavailable"}</span></p>
          <p><span className="text-muted-foreground">Active digest:</span> <span className="font-mono text-xs break-all">{status?.activeDigest || "no active revision"}</span></p>
          {status?.requirement?.required && <p><span className="text-muted-foreground">Required because:</span> {status.requirement.reason}</p>}
          {status?.lastReloadAttempt?.diagnostic && <p className="text-destructive">{status.lastReloadAttempt.diagnostic.code}: {status.lastReloadAttempt.diagnostic.message}</p>}
          {lastReconcile && !lastReconcile.hardEnforcementSatisfied && <p className="text-destructive">Required hard enforcement is not satisfied: {lastReconcile.missingHardEnforcementRuleIds.join(", ")}</p>}
        </CardContent>
      </Card>

      <Card className="border-border bg-card/40">
        <CardContent className="space-y-3 py-5 text-sm">
          <h3 className="font-semibold">Operator actions</h3>
          <p className="text-xs text-muted-foreground">Validate and reload Git-managed declared state. Plan and doctor only query resource-owned adapters. Reconcile writes native files through those adapters only.</p>
          <div className="flex flex-wrap gap-2">
            <Button variant="outline" disabled={busy !== null} onClick={() => void run("validate", async () => { const result = await policy.validate(); setNotice(result.valid ? `Catalog valid: ${result.candidateDigest}` : result.diagnostic?.message || "Catalog is invalid"); })}>{busy === "validate" ? "Validating…" : "Validate"}</Button>
            <Button variant="outline" disabled={busy !== null} onClick={() => void run("reload", async () => { const result = await policy.reload(); setNotice(result.activated ? "Catalog activated" : result.diagnostic?.message || "Catalog was not activated"); })}>{busy === "reload" ? "Reloading…" : "Reload"}</Button>
            <Button variant="outline" disabled={busy !== null} onClick={() => void run("plan", async () => { const result = await policy.plan(); setPlan(result.plan ?? null); setNotice("Projection plan refreshed"); })}>{busy === "plan" ? "Planning…" : "Plan"}</Button>
            <Button variant="outline" disabled={busy !== null} onClick={() => void run("doctor", async () => { const result = await policy.doctor(); setDoctor(result); setPlan(result.plan ?? null); setNotice(result.summary); })}>{busy === "doctor" ? "Checking…" : "Doctor"}</Button>
          </div>
          <label className="flex items-start gap-2 rounded-md border border-amber-500/30 bg-amber-500/10 p-3 text-xs">
            <Checkbox checked={authorized} onCheckedChange={setAuthorized} />
            <span>I am explicitly authorized to reconcile this whole declared permission document through installed resource adapters.</span>
          </label>
          <Button disabled={busy !== null || !authorized} onClick={() => void run("reconcile", async () => { const result = await policy.reconcile(); setAuthorized(false); setPlan(null); setNotice(result.result?.success ? "Reconcile completed" : "Reconcile completed with partial failure"); })}>{busy === "reconcile" ? "Reconciling…" : "Reconcile declared permissions"}</Button>
          {notice && <p className="text-xs text-muted-foreground">{notice}</p>}
        </CardContent>
      </Card>

      {doctor && <p className={doctor.healthy ? "text-sm text-muted-foreground" : "text-sm text-destructive"}>{doctor.summary}</p>}
      <PlanEvidence plan={plan} />

      {catalog && <Card className="border-border bg-card/40"><CardContent className="space-y-3 py-5 text-sm"><h3 className="font-semibold">Declared portable rules</h3><p className="text-xs text-muted-foreground">{catalog.metadata?.catalogId} · scopes: {catalog.targetScopes.join(", ")}</p>{catalog.rules.map((rule) => <div key={rule.id} className="border-t border-border pt-2"><div className="font-medium">{rule.id} <Badge variant="outline">{rule.action}</Badge>{rule.requiresHardEnforcement && <Badge variant="destructive" className="ml-2">hard enforcement required</Badge>}</div><p className="mt-1 font-mono text-xs text-muted-foreground">{rule.matcher?.kind}: {rule.matcher?.pattern}</p><p className="mt-1 text-xs text-muted-foreground">{rule.rationale} · owner: {rule.owner} · scope: {rule.targetScope}</p></div>)}</CardContent></Card>}
    </div>
  );
}
