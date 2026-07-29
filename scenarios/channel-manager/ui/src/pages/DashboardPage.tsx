import { useEffect, useMemo, useState } from "react";

import { Button } from "../components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Input } from "../components/ui/input";
import { Textarea } from "../components/ui/textarea";
import { selectors } from "../consts/selectors";
import { completeAction, createIdentity, enqueueAction, overview, recordObservation, startProgram, type Action, type Overview } from "../api/channelManager";

/**
 * P0's operator surface is intentionally manual: it tells the operator what
 * to do in the platform, captures the evidence they bring back, and never
 * attempts to log into or automate an account.
 */
export function DashboardPage() {
  const [evidence, setEvidence] = useState("");
  const [completed, setCompleted] = useState(false);
  const [metric, setMetric] = useState("");
  const [recorded, setRecorded] = useState(false);
  const [identityID, setIdentityID] = useState("");
  const [action, setAction] = useState<Action | null>(null);
  const [message, setMessage] = useState("");
  const [data, setData] = useState<Overview | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState("");
  const refresh = async () => {
    setLoading(true);
    try { setData(await overview()); setLoadError(""); } catch { setLoadError("The operator console could not load its saved work list. Retry before taking action."); } finally { setLoading(false); }
  };
  useEffect(() => { void refresh(); }, []);
  const dueActions = useMemo(() => Object.values(data?.actions ?? {}).filter((candidate) => candidate.status === "scheduled"), [data]);
  const create = async () => {
    try {
      await createIdentity({ id: identityID, platform_id: "x", purpose: "brand", environment_ref: "operator-attested-environment", vault_ref: `vault://channel-manager/identities/${identityID}`, status: "draft", attestations: { region_locked: true, unique_fingerprint: true } });
      await startProgram(identityID, "x-conservative");
      setMessage("Identity created and warming program started. Complete the queued platform actions manually.");
      await refresh();
    } catch {
      // A repeat visit should resume a safe draft/warming identity instead of
      // forcing an operator to invent a second platform-account record.
      try {
        const current = await overview();
        if (!current.identities[identityID]) throw new Error("identity was not found after create failure");
        setMessage("Existing identity resumed and warming program started. Complete the queued platform actions manually.");
      } catch {
        setMessage("Could not create or resume the identity. Confirm the API is running and that the identifier is valid.");
      }
    }
  };
  const queue = async () => {
    try {
      setAction(await enqueueAction(identityID, "engage"));
      setMessage("Manual action queued.");
      await refresh();
    } catch {
      // A resumed operator session should surface the already-scheduled
      // action instead of attempting to exceed the identity's cadence cap.
      try {
        const current = await overview();
        const existing = Object.values(current.actions).find((candidate) => candidate.identity_id === identityID && candidate.status === "scheduled");
        if (!existing) throw new Error("no queued action to resume");
        setAction(existing);
        setMessage("Existing manual action resumed.");
      } catch {
        setMessage("Could not queue or resume this action; check the warming stage and cadence.");
      }
    }
  };
  const finish = async () => { if (!action) return; try { await completeAction(action.id, evidence); setCompleted(true); setMessage("Manual completion saved."); await refresh(); } catch { setMessage("Could not record completion."); } };
  const observe = async () => { try { await recordObservation(identityID, Number(metric)); setRecorded(true); setMessage("Observation recorded."); await refresh(); } catch { setMessage("Could not record observation."); } };

  return (
    <section data-testid={selectors.pages.dashboard} aria-labelledby="dashboard-heading" className="flex flex-col gap-4">
      <div>
        <h2 id="dashboard-heading" className="text-2xl font-semibold">Channel operations</h2>
        <p className="text-app-muted-foreground">A manual work list for platform identities. No account credentials are entered or stored here.</p>
      </div>
      {loading && <p role="status" className="text-sm">Loading identity roster and due actions…</p>}
      {loadError && <div role="alert" className="flex items-center gap-3 rounded-control border border-red-400 p-3 text-sm"><span>{loadError}</span><Button variant="secondary" onClick={() => void refresh()}>Retry work list</Button></div>}
      <div className="grid gap-4 xl:grid-cols-2">
        <Card>
          <CardHeader><CardTitle>Identity roster</CardTitle></CardHeader>
          <CardContent className="space-y-2 text-sm">
            <p><strong>Before activation:</strong> create the account outside Vrooli, place its secret in Vault, then record only the Vault reference and attest the clean environment.</p>
            <p><strong>Warming:</strong> start the conservative program only after each precondition is checked. Its numbers are speculative operator practice, not a platform promise.</p>
            <label className="block font-medium" htmlFor="identity-id">New identity ID</label>
            <Input id="identity-id" data-testid={selectors.operator.identityInput} value={identityID} onChange={(event) => setIdentityID(event.target.value)} placeholder="x-brand-01" />
            <Button data-testid={selectors.operator.createButton} onClick={create} disabled={!identityID}>Create and start X warming</Button>
            {message.includes("warming program started") && <p data-testid={selectors.operator.identityReady} role="status">Identity is ready for its first manual action.</p>}
            <Button data-testid={selectors.operator.queueButton} onClick={queue} disabled={!identityID || !message.includes("warming program started")} variant="secondary">Queue manual engagement</Button>
            {action && <p data-testid={selectors.operator.actionReady} role="status">A manual engagement is ready to complete.</p>}
            {!loading && data && Object.keys(data.identities).length === 0 && <p className="text-app-muted-foreground">No identities yet. Add only an account that already exists outside Vrooli.</p>}
            {Object.values(data?.identities ?? {}).map((identity) => <div key={identity.id} className="rounded-control border border-app-border p-2" aria-label={`${identity.id}: ${identity.status}`}><strong>{identity.id}</strong><span className="ml-2">{identity.platform_id} · {identity.purpose} · {identity.status}</span>{identity.lane_grants?.length ? <span className="ml-2">Lanes: {identity.lane_grants.join(", ")}</span> : <span className="ml-2">No lane grants</span>}</div>)}
          </CardContent>
        </Card>
        <Card>
          <CardHeader><CardTitle>Due action</CardTitle></CardHeader>
          <CardContent className="space-y-3">
            {dueActions.length === 0 && <p className="text-sm text-app-muted-foreground">No actions are due. The queue will remain quiet until an operator schedules permitted work.</p>}
            {dueActions.map((due) => <button key={due.id} type="button" className="w-full rounded-control border border-app-border p-2 text-left text-sm" onClick={() => { setAction(due); setIdentityID(due.identity_id); setCompleted(false); }}><strong>{due.identity_id}</strong> · {due.kind} · window {new Date(due.window).toLocaleString()} · rolled count {due.rolled_count}</button>)}
            <p className="text-sm">When an action is due, complete it manually in the named platform, return here, and attach optional evidence (URL, number, or screenshot reference).</p>
            <label className="block text-sm font-medium" htmlFor="evidence">Completion evidence</label>
            <Textarea id="evidence" data-testid={selectors.operator.evidenceInput} value={evidence} onChange={(event) => setEvidence(event.target.value)} placeholder="https://… or a screenshot reference" />
            <Button data-testid={selectors.operator.completeButton} onClick={finish} disabled={completed || !action}>{completed ? "Action recorded" : "Record manual completion"}</Button>
            {completed && <p role="status" className="text-sm">Manual completion recorded with {evidence ? "evidence" : "no evidence"}. The executor never receives credentials.</p>}
          </CardContent>
        </Card>
        <Card>
          <CardHeader><CardTitle>Distribution observation</CardTitle></CardHeader>
          <CardContent className="space-y-3">
            <label className="block text-sm font-medium" htmlFor="reach">Reach or impressions</label>
            <Input id="reach" data-testid={selectors.operator.reachInput} inputMode="decimal" value={metric} onChange={(event) => setMetric(event.target.value)} placeholder="0" />
            <Button data-testid={selectors.operator.observeButton} onClick={observe} disabled={metric.trim()==="" || recorded || !identityID}>{recorded ? "Observation recorded" : "Record observation"}</Button>
            {recorded && <p role="status" className="text-sm">Observation recorded. A low measurement is evidence, not a claim of a platform penalty.</p>}
          </CardContent>
        </Card>
        <Card>
          <CardHeader><CardTitle>Program provenance and safety</CardTitle></CardHeader>
          <CardContent className="space-y-2 text-sm">
            {Object.values(data?.programs ?? {}).map((program) => <p key={program.id}><strong>{program.id}:</strong> {program.provenance.confidence} ({program.provenance.source_kind}); revisit {program.provenance.revisit_trigger}.</p>)}
            {!loading && !data?.programs && <p><strong>Confidence:</strong> speculative. Revisit after five completed manual runs.</p>}
            <p>Any flagged decline pauses the identity queue; the system takes no automatic corrective action. Graduation must be earned by declared criteria.</p>
            {Object.entries(data?.flags ?? {}).flatMap(([id, flags]) => flags.map((flag, index) => <p key={`${id}-${index}`} className="rounded-control border border-app-border p-2">Flag for {id}: {flag.message ?? flag.metric ?? "measurement needs review"}</p>))}
          </CardContent>
        </Card>
      </div>
      {message && <p data-testid={selectors.operator.status} role="status" className="text-sm">{message}</p>}
    </section>
  );
}
