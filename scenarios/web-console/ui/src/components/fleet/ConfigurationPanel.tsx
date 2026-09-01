import { useEffect, useState } from "react";
import FormWizard from "@vrooli/react-component-library/FormWizard/1";
import { GeneratedForm } from "@vrooli/react-component-library/GeneratedForm/1";
import PasswordInput from "@vrooli/react-component-library/PasswordInput/1";
import { AuditTrail } from "@vrooli/react-component-library/AuditTrail/1";
import { VerdictSummary } from "@vrooli/react-component-library/VerdictSummary/1";
import { toGeneratedFields, type OperatorInput } from "@vrooli/react-component-library/ValidationAdapter/1";
import { Button } from "@vrooli/react-component-library/Button/2";
import { answerSecret, createCredentialGrant, getConfiguration, getConfigurationApplyStatus, listCredentialGrants, reapplyConfiguration, resolveConfiguration, revokeCredentialGrant, type ConfigurationQuestion, type Machine, type MachineConfigurationDetail } from "../../api/machines";
import type { CredentialGrant } from "@vrooli/proto-types/vrooli-bridge/v1/credentialgrant/credentialgrant_pb";

export function ConfigurationPanel({ machine, onBack }: { machine: Machine; onBack: () => void }) {
  const [questions, setQuestions] = useState<ConfigurationQuestion[]>([]);
  const [secretValues, setSecretValues] = useState<Record<string, string>>({});
  const [status, setStatus] = useState("Loading target configuration…");
  const [readiness, setReadiness] = useState<unknown>(null);
  const [detail, setDetail] = useState<MachineConfigurationDetail | null>(null);
  const [grants, setGrants] = useState<CredentialGrant[]>([]);
  const [reapplying, setReapplying] = useState(false);
  const [grantIdentity, setGrantIdentity] = useState("");
  const [grantField, setGrantField] = useState("");
  const [granting, setGranting] = useState(false);
  useEffect(() => {
    void Promise.all([getConfiguration(machine.target.id), listCredentialGrants(machine.target.id)]).then(([result, grantResult]) => {
      setQuestions(result.questions);
      setReadiness(result.readiness);
      setDetail(result.detail);
      setGrants(grantResult.grants);
      setStatus(result.questions.length ? "Answer the outstanding questions, then submit." : "No outstanding questions were reported.");
    }).catch((error: unknown) => setStatus(error instanceof Error ? error.message : "The target configuration could not be loaded."));
  }, [machine.target.id]);

  const secrets = questions.filter((question) => question.kind === "secret");
  const regular = questions.filter((question) => question.kind !== "secret");
  const adapted = regular.map((question) => ({ id: question.id, kind: question.kind, label: question.title, description: question.description, required: question.required, defaultValue: question.default, options: question.options?.map((value) => ({ value, label: value })), candidates: question.candidates?.map((candidate) => ({ label: candidate.label, value: candidate.id, status: candidate.status, risk: candidate.risk, remediation: candidate.remediation })), validation: question.validation }));
  const fields = toGeneratedFields(adapted as OperatorInput[]) as never[];
  const submit = async (values: Record<string, unknown>) => {
    setStatus("Submitting answers through the sealed target path…");
    try {
      const nodeId = machine.target.node_id || machine.target.id;
      const secretAnswers = secrets.map((question) => answerSecret({ nodeId, logicalId: question.owner || question.id.split(":")[0] || question.id, field: question.input_id || question.id.split(":")[1] || "value", value: secretValues[question.id] ?? "" }));
      await Promise.all(secretAnswers);
      if (regular.length > 0) {
        await resolveConfiguration(machine.target.id, regular.map((question) => ({ request_id: question.id, value: String(values[question.id] ?? "") })));
      }
      setSecretValues({});
      setStatus("Answers accepted. Refresh the machine to verify readiness and drift.");
    } catch (error: unknown) {
      setStatus(error instanceof Error ? error.message : "The target rejected the configuration answers.");
    }
  };
  const trackApply = async (runId: string) => {
    for (let attempt = 0; attempt < 180; attempt += 1) {
      const current = await getConfigurationApplyStatus(machine.target.id, runId);
      const run = current.result as { status?: string; items?: Array<{ name?: string; outcome?: string }> };
      setStatus(`Re-apply ${run.status ?? "running"}${run.items?.length ? ` · ${run.items.map((item) => `${item.name ?? "step"}: ${item.outcome ?? "pending"}`).join(", ")}` : ""}`);
      if (run.status && !["pending", "applying"].includes(run.status)) {
        const refreshed = await getConfiguration(machine.target.id);
        setDetail(refreshed.detail);
        setReadiness(refreshed.readiness);
        setQuestions(refreshed.questions);
        setStatus(`Re-apply ${run.status}; configuration evidence refreshed.`);
        return;
      }
      await new Promise((resolve) => window.setTimeout(resolve, 1000));
    }
    setStatus("Re-apply is still running; the durable run remains available for refresh.");
  };
  return <section className="flex h-full min-h-0 flex-col p-5" data-testid="machine-configuration-panel" aria-label={`Configuration for ${machine.target.label}`}>
    <div className="flex items-center justify-between gap-3"><div><h2 className="text-lg font-semibold">Configure {machine.target.label}</h2><p className="text-xs text-wc-text-muted">Target: <code>{machine.target.id}</code></p></div><div className="flex gap-2"><Button variant="outline" size="sm" onClick={() => { setReapplying(true); setStatus("Re-applying the desired configuration…"); void reapplyConfiguration(machine.target.id).then(async (result) => { const runId = (result.result as { run_id?: string })?.run_id; if (runId) await trackApply(runId); else setStatus("Re-apply returned no durable run id."); }).catch((error: unknown) => setStatus(error instanceof Error ? error.message : "The configuration could not be re-applied.")).finally(() => setReapplying(false)); }} disabled={reapplying}>{reapplying ? "Applying…" : "Re-apply"}</Button><Button variant="outline" size="sm" onClick={onBack}>Back</Button></div></div>
    <p className="mt-3 text-sm text-wc-text-muted" role="status" data-testid="machine-configuration-status">{status}</p>
    {detail && <div className="mt-3 grid gap-3 md:grid-cols-2">
      <div className="rounded-lg border border-wc-default p-3 text-xs"><h3 className="font-medium">Configuration copies</h3><dl className="mt-2 space-y-1 text-wc-text-muted"><div><dt className="inline">Desired: </dt><dd className="inline text-wc-text-primary">{detail.machine?.desiredProfileId || "not recorded"} {detail.machine?.desiredProfileVersion ? `(${detail.machine.desiredProfileVersion})` : ""}</dd></div><div><dt className="inline">Applied: </dt><dd className="inline text-wc-text-primary">{detail.machine?.appliedProfileId || "not recorded"} {detail.machine?.appliedProfileVersion ? `(${detail.machine.appliedProfileVersion})` : ""}</dd></div></dl>{detail.machine?.desiredSelectionJson && <details className="mt-2"><summary>Desired selection document</summary><pre className="mt-1 max-h-32 overflow-auto whitespace-pre-wrap">{detail.machine.desiredSelectionJson}</pre></details>}{detail.machine?.appliedSelectionJson && <details className="mt-2"><summary>Applied selection document</summary><pre className="mt-1 max-h-32 overflow-auto whitespace-pre-wrap">{detail.machine.appliedSelectionJson}</pre></details>}</div>
      <div className="rounded-lg border border-wc-default p-3"><h3 className="mb-2 text-xs font-medium">Configuration outcome</h3><VerdictSummary pass={detail.readiness?.ready ? 1 : 0} fail={detail.readiness?.ready ? 0 : (detail.readiness?.reasons?.length ?? 1)} unmeasured={detail.readiness ? 0 : 1} /></div>
    </div>}
    {readiness !== null && <details className="mt-3 rounded-lg border border-wc-default p-3 text-xs"><summary>Current readiness facts</summary><pre className="mt-2 max-h-48 overflow-auto whitespace-pre-wrap">{String(JSON.stringify(readiness, null, 2))}</pre></details>}
    {detail && (detail.drift?.length ?? 0) > 0 && <div className="mt-3 rounded-lg border border-amber-300/30 p-3 text-xs text-amber-200"><h3 className="font-medium">Configuration drift</h3>{detail.drift?.map((item) => <p key={`${item.kind}:${item.name}`}>{item.name}: {item.reason}</p>)}</div>}
    {detail && (detail.auditEvents?.length ?? 0) > 0 && <div className="mt-3 rounded-lg border border-wc-default p-3"><h3 className="mb-2 text-xs font-medium">Machine audit trail</h3><AuditTrail entries={(detail.auditEvents ?? []).map((event) => ({ actor: event.actor || "system", action: [event.action, event.detail].filter(Boolean).join(": ") }))} /></div>}
    <div className="mt-3 rounded-lg border border-wc-default p-3" data-testid="machine-credential-grants"><h3 className="text-xs font-medium">Held credentials</h3><p className="mt-1 text-xs text-wc-text-muted">Values stay sealed and are never shown here. This list is delivery metadata from Bridge.</p><form className="mt-2 flex flex-wrap gap-2" onSubmit={(event) => { event.preventDefault(); setGranting(true); void createCredentialGrant({ nodeId: machine.target.node_id || machine.target.id, logicalId: grantIdentity, field: grantField, class: "user_prompt", retention: "durable" }).then((grant) => { setGrants((current) => [...current, grant]); setGrantIdentity(""); setGrantField(""); setStatus("Grant created; Bridge will push the sealed value when the authority has it."); }).catch((error: unknown) => setStatus(error instanceof Error ? error.message : "The credential grant could not be created.")).finally(() => setGranting(false)); }}><input required aria-label="Credential identity" placeholder="namespace/name" value={grantIdentity} onChange={(event) => setGrantIdentity(event.target.value)} className="min-w-0 flex-1 rounded border border-wc-default bg-wc-surface-input px-2 py-1 text-xs" /><input required aria-label="Credential field" placeholder="field" value={grantField} onChange={(event) => setGrantField(event.target.value)} className="min-w-0 flex-1 rounded border border-wc-default bg-wc-surface-input px-2 py-1 text-xs" /><Button size="sm" type="submit" disabled={granting}>{granting ? "Granting…" : "Grant and push"}</Button></form>{grants.length === 0 ? <p className="mt-2 text-xs text-wc-text-muted">No credential grants are held by this target.</p> : <div className="mt-2 space-y-2">{grants.map((grant) => <div key={grant.id} className="flex items-center justify-between gap-2 text-xs"><span>{grant.logicalId}:{grant.field} · generation {grant.generation.toString()} · {grant.receiptAccepted && grant.ackedGeneration >= grant.generation ? `received${grant.receiptAt ? ` ${new Date(Number(grant.receiptAt.seconds) * 1000).toLocaleString()}` : ""}` : grant.receiptReason ? `refused: ${grant.receiptReason}` : "pending"}</span><Button variant="outline" size="sm" onClick={() => { void revokeCredentialGrant(grant.id).then(() => setGrants((current) => current.filter((item) => item.id !== grant.id))).catch((error: unknown) => setStatus(error instanceof Error ? error.message : "The credential grant could not be revoked.")); }}>Revoke</Button></div>)}</div>}</div>
    {questions.length > 0 && <div className="mt-4 min-h-0 overflow-auto"><FormWizard draftKey={`machine-${machine.target.id}`} steps={[{ id: "configuration", title: "Configuration questions", content: <div>{secrets.map((question) => <PasswordInput key={question.id} name={question.id} label={question.title} value={secretValues[question.id] ?? ""} onChange={(value) => setSecretValues((current) => ({ ...current, [question.id]: value }))} autoComplete="new-password" />)}{regular.length > 0 && <GeneratedForm fields={fields} onSubmit={submit} submitLabel="Submit answers" />}{secrets.length > 0 && <Button className="mt-3" onClick={() => { void submit({}); }}>Submit answers</Button>}</div> }]} /> </div>}
  </section>;
}
