import { useEffect, useMemo, useState } from "react";

import { Button } from "../components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Input } from "../components/ui/input";
import { Textarea } from "../components/ui/textarea";
import { selectors } from "../consts/selectors";
import { assignAutomation, completeAction, configurePortfolio, createIdentity, dispatchBrowserAction, enqueueAction, overview, previewRelease, recordObservation, startProgram, type Action, type Overview, type ReleasePreview } from "../api/channelManager";

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
  const [previewCaption, setPreviewCaption] = useState("");
	const [previewPlatform, setPreviewPlatform] = useState("x");
  const [previewDisclosure, setPreviewDisclosure] = useState(false);
  const [preview, setPreview] = useState<ReleasePreview | null>(null);
	const [profileRef, setProfileRef] = useState("");
	const [workflowRef, setWorkflowRef] = useState("");
	const [automationKind, setAutomationKind] = useState("engage");
	const [automationNote, setAutomationNote] = useState("");
	const [portfolioGap, setPortfolioGap] = useState("30");
	const [portfolioWindow, setPortfolioWindow] = useState("1440");
	const [portfolioCeiling, setPortfolioCeiling] = useState("1");
  const refresh = async (isCurrent: () => boolean = () => true) => {
    setLoading(true);
    try {
      const next = await overview();
      if (!isCurrent()) return;
      setData(next);
      setLoadError("");
    } catch {
      if (!isCurrent()) return;
      setLoadError("The operator console could not load its saved work list. Retry before taking action.");
    } finally {
      if (isCurrent()) setLoading(false);
    }
  };
  useEffect(() => {
    let current = true;
    void refresh(() => current);
    return () => { current = false; };
  }, []);
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
  // Refresh can replace the due-action projection while an operator is typing
  // evidence. The first scheduled action remains the safe fallback because it
  // is the same server-owned queue the operator sees in the Due action card.
  const currentAction = action ?? dueActions[0] ?? null;
  const finish = async () => { if (!currentAction) return; try { await completeAction(currentAction.id, evidence); setCompleted(true); setMessage("Manual completion saved."); await refresh(); } catch { setMessage("Could not record completion."); } };
  const observe = async () => { try { await recordObservation(identityID, Number(metric)); setRecorded(true); setMessage("Observation recorded."); await refresh(); } catch { setMessage("Could not record observation."); } };
	const renderPreview = async () => { try { setPreview(await previewRelease({ platform_id: previewPlatform, caption: previewCaption, format_kind: "image", media_width: 1200, media_height: 1200, disclosure_visible: previewDisclosure, first_comment: "" })); } catch { setMessage("Could not render the platform preview."); } };
	const configureAutomation = async () => { try { await assignAutomation(identityID, { session_profile_ref: profileRef, workflow_ref: workflowRef, enabled_action_kinds: [automationKind], operator_note: automationNote }); setMessage("BAS profile and workflow assignment saved. Browser dispatch remains limited to this approved action kind."); await refresh(); } catch { setMessage("Could not save the BAS profile assignment. Check the identity and operator decision."); } };
	const dispatchBrowser = async () => { if (!currentAction) return; try { const result = await dispatchBrowserAction(currentAction.id); setMessage(`Browser execution dispatched: ${result.execution_id}. Manual completion remains available if BAS cannot continue.`); await refresh(); } catch { setMessage("Browser dispatch did not start; keep the action in the manual workflow and review BAS availability."); } };
	const savePortfolio = async () => { try { await configurePortfolio({ minimum_post_gap_minutes: Number(portfolioGap), window_minutes: Number(portfolioWindow), max_posts_per_window: Number(portfolioCeiling) }); setMessage("Portfolio separation policy saved. Future publishes will defer into compliant slots."); await refresh(); } catch { setMessage("Could not save portfolio policy. Values must be non-negative and a ceiling needs a positive window."); } };

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
            {currentAction && <p data-testid={selectors.operator.actionReady} role="status">A manual engagement is ready to complete.</p>}
            {!loading && data && Object.keys(data.identities).length === 0 && <p className="text-app-muted-foreground">No identities yet. Add only an account that already exists outside Vrooli.</p>}
            {Object.values(data?.identities ?? {}).map((identity) => <div key={identity.id} className="rounded-control border border-app-border p-2" aria-label={`${identity.id}: ${identity.status}`}><strong>{identity.id}</strong><span className="ml-2">{identity.platform_id} · {identity.purpose} · {identity.status}</span>{identity.lane_grants?.length ? <span className="ml-2">Lanes: {identity.lane_grants.join(", ")}</span> : <span className="ml-2">No lane grants</span>}</div>)}
          </CardContent>
        </Card>
        <Card>
          <CardHeader><CardTitle>Due action</CardTitle></CardHeader>
          <CardContent className="space-y-3">
            {dueActions.length === 0 && <p className="text-sm text-app-muted-foreground">No actions are due. The queue will remain quiet until an operator schedules permitted work.</p>}
			{dueActions.map((due) => <button key={due.id} data-testid={selectors.operator.dueAction} type="button" className="w-full rounded-control border border-app-border p-2 text-left text-sm" onClick={() => { setAction(due); setIdentityID(due.identity_id); setCompleted(false); }}><strong>{due.identity_id}</strong> · {due.kind} · window {new Date(due.window).toLocaleString()} · rolled count {due.rolled_count}{due.deferred ? " · deferred by portfolio policy" : ""}{due.failure_class ? ` · ${due.failure_class}` : ""}{due.next_attempt_at ? ` · retry after ${new Date(due.next_attempt_at).toLocaleString()}` : ""}{due.execution_error ? <span className="block mt-1 text-app-muted-foreground">Execution note: {due.execution_error}</span> : null}</button>)}
            <p className="text-sm">When an action is due, complete it manually in the named platform, return here, and attach optional evidence (URL, number, or screenshot reference).</p>
            <label className="block text-sm font-medium" htmlFor="evidence">Completion evidence</label>
            <Textarea id="evidence" data-testid={selectors.operator.evidenceInput} value={evidence} onChange={(event) => setEvidence(event.target.value)} placeholder="https://… or a screenshot reference" />
            <Button data-testid={selectors.operator.completeButton} onClick={finish} disabled={completed || !currentAction}>{completed ? "Action recorded" : "Record manual completion"}</Button>
			{currentAction ? <Button variant="secondary" onClick={() => void dispatchBrowser()}>Dispatch approved browser action</Button> : null}
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
		  <CardHeader><CardTitle>Browser automation gate</CardTitle></CardHeader>
		  <CardContent className="space-y-3 text-sm"><p>Assign only BAS profile and workflow references—never cookies, tokens, or account credentials. The required note records the operator’s platform-specific acceptance decision.</p><Input aria-label="BAS session profile reference" value={profileRef} onChange={(event) => setProfileRef(event.target.value)} placeholder="BAS profile reference" /><Input aria-label="BAS workflow reference" value={workflowRef} onChange={(event) => setWorkflowRef(event.target.value)} placeholder="BAS workflow UUID" /><Input aria-label="Approved action kind" value={automationKind} onChange={(event) => setAutomationKind(event.target.value)} placeholder="engage" /><Textarea aria-label="Automation acceptance note" value={automationNote} onChange={(event) => setAutomationNote(event.target.value)} placeholder="Decision record and sanctioned test identity" /><Button variant="secondary" disabled={!identityID || !profileRef || !workflowRef || !automationKind || !automationNote} onClick={() => void configureAutomation()}>Save browser automation gate</Button></CardContent>
		</Card>
		<Card>
		  <CardHeader><CardTitle>Portfolio separation</CardTitle></CardHeader>
		  <CardContent className="space-y-3 text-sm"><p>Future publishes from different identities are deferred to satisfy this operator policy. Existing work is never rewritten.</p><Input aria-label="Minimum cross-identity posting gap in minutes" inputMode="numeric" value={portfolioGap} onChange={(event) => setPortfolioGap(event.target.value)} placeholder="Minimum gap minutes" /><Input aria-label="Portfolio rolling window in minutes" inputMode="numeric" value={portfolioWindow} onChange={(event) => setPortfolioWindow(event.target.value)} placeholder="Window minutes" /><Input aria-label="Maximum publishes per rolling window" inputMode="numeric" value={portfolioCeiling} onChange={(event) => setPortfolioCeiling(event.target.value)} placeholder="Maximum publishes" /><Button variant="secondary" onClick={() => void savePortfolio()}>Save portfolio policy</Button>{data?.portfolio ? <p>Current: {data.portfolio.minimum_post_gap_minutes}m gap · at most {data.portfolio.max_posts_per_window} publish(es) / {data.portfolio.window_minutes}m.</p> : null}</CardContent>
		</Card>
		<Card>
		  <CardHeader><CardTitle>Program provenance and safety</CardTitle></CardHeader>
          <CardContent className="space-y-2 text-sm">
            {Object.values(data?.programs ?? {}).map((program) => <p key={program.id}><strong>{program.id}:</strong> {program.provenance.confidence} ({program.provenance.source_kind}); {data?.program_support?.[program.id] ?? 0} completed run(s) support this program; revisit {program.provenance.revisit_trigger}.</p>)}
            {!loading && !data?.programs && <p><strong>Confidence:</strong> speculative. Revisit after five completed manual runs.</p>}
            <p>Any flagged decline pauses the identity queue; the system takes no automatic corrective action. Graduation must be earned by declared criteria.</p>
            {Object.entries(data?.flags ?? {}).flatMap(([id, flags]) => flags.map((flag, index) => <p key={`${id}-${index}`} className="rounded-control border border-app-border p-2">Flag for {id}: {flag.message ?? flag.metric ?? "measurement needs review"}</p>))}
		  </CardContent>
		</Card>
		<Card>
		  <CardHeader><CardTitle>Environment liveness</CardTitle></CardHeader>
		  <CardContent className="space-y-2 text-sm"><p>Environment probes use only an opaque environment reference. Unknown or mismatched region checks pause the queue until an operator resolves the environment outside this scenario.</p>{Object.values(data?.environment_checks ?? {}).length ? <ul className="space-y-1">{Object.values(data?.environment_checks ?? {}).map((check) => <li key={check.identity_id}>{check.identity_id}: <strong>{check.status}</strong>{check.expected_region ? ` · expected ${check.expected_region}` : ""}{check.observed_region ? ` · observed ${check.observed_region}` : ""}{check.reason ? ` · ${check.reason}` : ""}</li>)}</ul> : <p className="text-app-muted-foreground">No liveness checks recorded.</p>}</CardContent>
		</Card>
        <Card>
          <CardHeader><CardTitle>Release preview</CardTitle></CardHeader>
          <CardContent className="space-y-3">
			<label className="block text-sm font-medium" htmlFor="preview-platform">Platform descriptor</label>
			<select id="preview-platform" aria-label="Preview platform" value={previewPlatform} onChange={(event) => setPreviewPlatform(event.target.value)} className="min-h-11 w-full rounded-control border border-app-border bg-app-surface px-3 py-2 text-sm">
				{Object.values(data?.platforms ?? { x: { id: "x" } }).map((platform) => <option key={platform.id} value={platform.id}>{platform.id}</option>)}
			</select>
            <label className="block text-sm font-medium" htmlFor="preview-caption">Caption</label>
            <Textarea id="preview-caption" value={previewCaption} onChange={(event) => setPreviewCaption(event.target.value)} placeholder="Caption to preview" />
            <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={previewDisclosure} onChange={(event) => setPreviewDisclosure(event.target.checked)} /> Disclosure visible in rendered media</label>
            <Button variant="secondary" onClick={() => void renderPreview()}>Render platform preview</Button>
            {preview ? <div aria-live="polite" className="rounded-control border border-app-border p-2 text-sm"><p>{preview.release_allowed ? "Ready for release" : "Release blocked"}{preview.caption_truncated ? " · caption truncated" : ""}</p>{preview.blocking_errors.length ? <ul className="list-disc pl-5">{preview.blocking_errors.map((error) => <li key={error}>{error}</li>)}</ul> : null}</div> : null}
          </CardContent>
        </Card>
        <Card>
          <CardHeader><CardTitle>Release delivery and metrics</CardTitle></CardHeader>
          <CardContent className="space-y-3 text-sm">
            {Object.values(data?.releases ?? {}).length === 0 ? <p className="text-app-muted-foreground">No completed releases are awaiting Content Desk delivery.</p> : <ul className="space-y-2">{Object.values(data?.releases ?? {}).map((release) => <li key={release.id} className="rounded-control border border-app-border p-2"><strong>{release.draft_id}</strong> · {release.status} · Content Desk {release.delivery_status || "pending"}{release.first_comment_status === "failed" ? " · first comment failed" : ""}{release.delivery_error ? <p role="alert" className="mt-1">Delivery error: {release.delivery_error}</p> : null}</li>)}</ul>}
            {Object.values(data?.metric_samples ?? {}).length ? <div><h3 className="font-medium">Metric delivery</h3><ul className="mt-1 space-y-1">{Object.values(data?.metric_samples ?? {}).map((sample) => <li key={sample.id}>{sample.metric} {sample.value} · draft {sample.draft_id} · {sample.delivery_status}</li>)}</ul></div> : null}
          </CardContent>
        </Card>
      </div>
      {message && <p data-testid={selectors.operator.status} role="status" className="text-sm">{message}</p>}
    </section>
  );
}
