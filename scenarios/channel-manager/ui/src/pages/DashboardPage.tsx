import { useEffect, useMemo, useState } from "react";

import { Button } from "../components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Input } from "../components/ui/input";
import { Textarea } from "../components/ui/textarea";
import { selectors } from "../consts/selectors";
import { assignAutomation, completeAction, configurePortfolio, createIdentity, dispatchBrowserAction, enqueueAction, overview, previewRelease, recordObservation, retireIdentity, startProgram, updateIdentity, type Action, type Overview, type ReleasePreview } from "../api/channelManager";

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
  const [identityPlatform, setIdentityPlatform] = useState("");
  const [identityHandle, setIdentityHandle] = useState("");
  const [identityLabel, setIdentityLabel] = useState("");
  const [identityGoals, setIdentityGoals] = useState("");
  const [identityNotes, setIdentityNotes] = useState("");
  const [identityOwnerRef, setIdentityOwnerRef] = useState("");
  const [identityCredentialRef, setIdentityCredentialRef] = useState("");
  const [d009AcceptanceRef, setD009AcceptanceRef] = useState("");
  const [action, setAction] = useState<Action | null>(null);
  const [message, setMessage] = useState("");
  const [data, setData] = useState<Overview | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState("");
  const [previewCaption, setPreviewCaption] = useState("");
  const [previewPlatform, setPreviewPlatform] = useState("");
  const [previewDisclosure, setPreviewDisclosure] = useState(false);
  const [preview, setPreview] = useState<ReleasePreview | null>(null);
	const [profileKey, setProfileKey] = useState("");
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
		setIdentityPlatform((current) => current || Object.keys(next.platforms ?? {})[0] || "");
		setPreviewPlatform((current) => current || Object.keys(next.platforms ?? {})[0] || "");
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
  const configuredPlatforms = Object.values(data?.platforms ?? {});
  const selectedIdentityPlatform = identityPlatform || configuredPlatforms[0]?.id || "";
  const selectedPreviewPlatform = previewPlatform || configuredPlatforms[0]?.id || "";
	const selectedPreviewType = configuredPlatforms.find((platform) => platform.id === selectedPreviewPlatform)?.post_types?.[0];
  const create = async () => {
    try {
      await createIdentity({ id: identityID, platform_id: selectedIdentityPlatform, handle: identityHandle, display_label: identityLabel, goals: identityGoals.split("\n").map((goal) => goal.trim()).filter(Boolean), notes: identityNotes, owner_ref: identityOwnerRef, purpose: "brand", environment_ref: "operator-attested-environment", credential_ref: identityCredentialRef, status: "draft", lifecycle: "draft", d009_acceptance_ref: d009AcceptanceRef, automation_mode: d009AcceptanceRef ? "operator-gated" : "manual", attestations: { region_locked: true, unique_fingerprint: true } });
      const program = Object.values(data?.programs ?? {}).find((candidate) => candidate.platform_id === selectedIdentityPlatform);
      if (program) await startProgram(identityID, program.id);
      setMessage(program ? "Identity created and warming program started. Complete the queued platform actions manually." : "Identity created. Select a platform warming program before scheduling work.");
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
  const retire = async (id: string) => { try { await retireIdentity(id); setMessage("Identity retired. Existing history remains available for audit."); await refresh(); } catch { setMessage("Could not retire identity. Retry before using the account again."); } };
	const saveIdentityMetadata = async () => {
		const current = data?.identities[identityID];
		if (!current) { setMessage("Select an existing identity before saving metadata."); return; }
		try {
			await updateIdentity(identityID, { ...current, handle: identityHandle, display_label: identityLabel, goals: identityGoals.split("\n").map((goal) => goal.trim()).filter(Boolean), notes: identityNotes, owner_ref: identityOwnerRef, d009_acceptance_ref: d009AcceptanceRef, automation_mode: d009AcceptanceRef ? "operator-gated" : "manual" });
			setMessage("Identity metadata and automation policy saved."); await refresh();
		} catch { setMessage("Could not save identity metadata. Check the lifecycle policy and retry."); }
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
	const renderPreview = async () => { try { setPreview(await previewRelease({ platform_id: selectedPreviewPlatform, caption: previewCaption, post_type: selectedPreviewType?.id, format_kind: selectedPreviewType?.format_kind || "", media_width: 1200, media_height: 1200, disclosure_visible: previewDisclosure, disclosure_in_visible_region: previewDisclosure, first_comment: "" })); } catch { setMessage("Could not render the platform preview."); } };
	const configureAutomation = async () => { try { await assignAutomation(identityID, { consumer_profile_key: profileKey, session_profile_ref: profileRef, workflow_ref: workflowRef, enabled_action_kinds: [automationKind], operator_note: automationNote }); setMessage("BAS profile and workflow assignment saved. Browser dispatch remains limited to this approved action kind."); await refresh(); } catch { setMessage("Could not save the BAS profile assignment. Check the identity and operator decision."); } };
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
            <p><strong>Before activation:</strong> create the account outside Vrooli, place its secret in the credential authority, then record only the authority reference and attest the clean environment.</p>
            <p><strong>Warming:</strong> start the conservative program only after each precondition is checked. Its numbers are speculative operator practice, not a platform promise.</p>
            <label className="block font-medium" htmlFor="identity-id">New identity ID</label>
            <Input id="identity-id" data-testid={selectors.operator.identityInput} value={identityID} onChange={(event) => setIdentityID(event.target.value)} placeholder="x-brand-01" />
            <label className="block font-medium" htmlFor="identity-platform">Platform</label>
			<select id="identity-platform" aria-label="Identity platform" value={selectedIdentityPlatform} onChange={(event) => setIdentityPlatform(event.target.value)} className="min-h-11 w-full rounded-control border border-app-border bg-app-surface px-3 py-2 text-sm"><option value="">Select platform</option>{configuredPlatforms.map((platform) => <option key={platform.id} value={platform.id}>{platform.id}</option>)}</select>
            <Input aria-label="Account handle" value={identityHandle} onChange={(event) => setIdentityHandle(event.target.value)} placeholder="Account handle (optional)" />
            <Input aria-label="Identity display label" value={identityLabel} onChange={(event) => setIdentityLabel(event.target.value)} placeholder="Operator display label" />
            <Textarea aria-label="Identity goals" value={identityGoals} onChange={(event) => setIdentityGoals(event.target.value)} placeholder="One goal per line" />
            <Textarea aria-label="Operator notes" value={identityNotes} onChange={(event) => setIdentityNotes(event.target.value)} placeholder="Private operator notes" />
            <Input aria-label="Identity owner reference" value={identityOwnerRef} onChange={(event) => setIdentityOwnerRef(event.target.value)} placeholder="Owner reference" />
            <Input aria-label="Credential authority reference" value={identityCredentialRef} onChange={(event) => setIdentityCredentialRef(event.target.value)} placeholder="vrooli/channel-manager/identity:credential" />
            <Input aria-label="D-009 acceptance reference" value={d009AcceptanceRef} onChange={(event) => setD009AcceptanceRef(event.target.value)} placeholder="D-009 acceptance reference (needed for BAS)" />
			<Button data-testid={selectors.operator.createButton} onClick={create} disabled={!identityID || !selectedIdentityPlatform}>Create identity and start warming</Button>
			<Button variant="secondary" onClick={() => void saveIdentityMetadata()} disabled={!identityID}>Save identity metadata</Button>
            {message.includes("warming program started") && <p data-testid={selectors.operator.identityReady} role="status">Identity is ready for its first manual action.</p>}
            <Button data-testid={selectors.operator.queueButton} onClick={queue} disabled={!identityID || !message.includes("warming program started")} variant="secondary">Queue manual engagement</Button>
            {currentAction && <p data-testid={selectors.operator.actionReady} role="status">A manual engagement is ready to complete.</p>}
            {!loading && data && Object.keys(data.identities).length === 0 && <p className="text-app-muted-foreground">No identities yet. Add only an account that already exists outside Vrooli.</p>}
            {Object.values(data?.identities ?? {}).map((identity) => <div key={identity.id} className="rounded-control border border-app-border p-2" aria-label={`${identity.id}: ${identity.status}`}><strong>{identity.display_label || identity.id}</strong><span className="ml-2">{identity.platform_id} · {identity.purpose} · {identity.status}</span>{identity.lane_grants?.length ? <span className="ml-2">Lanes: {identity.lane_grants.join(", ")}</span> : <span className="ml-2">No lane grants</span>}{identity.status !== "retired" ? <Button variant="secondary" className="ml-2" onClick={() => void retire(identity.id)}>Retire</Button> : null}</div>)}
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
		  <CardContent className="space-y-3 text-sm"><p>Assign only a scenario-declared profile key plus BAS profile and workflow references—never cookies, tokens, or account credentials. The required note records the operator’s platform-specific acceptance decision.</p><Input aria-label="Declared BAS profile key" value={profileKey} onChange={(event) => setProfileKey(event.target.value)} placeholder="operator-account" /><Input aria-label="BAS session profile reference" value={profileRef} onChange={(event) => setProfileRef(event.target.value)} placeholder="BAS profile reference" /><Input aria-label="BAS workflow reference" value={workflowRef} onChange={(event) => setWorkflowRef(event.target.value)} placeholder="BAS workflow UUID" /><Input aria-label="Approved action kind" value={automationKind} onChange={(event) => setAutomationKind(event.target.value)} placeholder="engage" /><Textarea aria-label="Automation acceptance note" value={automationNote} onChange={(event) => setAutomationNote(event.target.value)} placeholder="Decision record and sanctioned test identity" /><Button variant="secondary" disabled={!identityID || !profileKey || !profileRef || !workflowRef || !automationKind || !automationNote} onClick={() => void configureAutomation()}>Save browser automation gate</Button></CardContent>
		</Card>
		<Card>
		  <CardHeader><CardTitle>Portfolio separation</CardTitle></CardHeader>
		  <CardContent className="space-y-3 text-sm"><p>Future publishes from different identities are deferred to satisfy this operator policy. Existing work is never rewritten.</p><Input aria-label="Minimum cross-identity posting gap in minutes" inputMode="numeric" value={portfolioGap} onChange={(event) => setPortfolioGap(event.target.value)} placeholder="Minimum gap minutes" /><Input aria-label="Portfolio rolling window in minutes" inputMode="numeric" value={portfolioWindow} onChange={(event) => setPortfolioWindow(event.target.value)} placeholder="Window minutes" /><Input aria-label="Maximum publishes per rolling window" inputMode="numeric" value={portfolioCeiling} onChange={(event) => setPortfolioCeiling(event.target.value)} placeholder="Maximum publishes" /><Button variant="secondary" onClick={() => void savePortfolio()}>Save portfolio policy</Button>{data?.portfolio ? <p>Current: {data.portfolio.minimum_post_gap_minutes}m gap · at most {data.portfolio.max_posts_per_window} publish(es) / {data.portfolio.window_minutes}m.</p> : null}</CardContent>
		</Card>
		<Card>
		  <CardHeader><CardTitle>Program provenance and safety</CardTitle></CardHeader>
          <CardContent className="space-y-2 text-sm">
	            {Object.values(data?.programs ?? {}).map((program) => {
				const provenance = program.provenance;
				return <p key={program.id}><strong>{program.id}:</strong> {provenance ? `${provenance.confidence} (${provenance.source_kind}); ${data?.program_support?.[program.id] ?? 0} completed run(s) support this program; revisit ${provenance.revisit_trigger}.` : "provenance is unavailable; do not treat this program as an evidence-backed policy."}</p>;
			})}
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
		  <CardHeader><CardTitle>Identity activity timeline</CardTitle></CardHeader>
		  <CardContent className="space-y-2 text-sm"><p>Immutable, chronological records distinguish queueing, BAS dispatch, manual verification, release completion, and metric delivery. Sensitive browser state is never shown.</p>{(data?.activity_events ?? []).filter((event) => !identityID || event.identity_id === identityID).length ? <ul className="space-y-1">{(data?.activity_events ?? []).filter((event) => !identityID || event.identity_id === identityID).map((event) => <li key={event.id} className="rounded-control border border-app-border p-2"><strong>{event.event_type}</strong> · {new Date(event.occurred_at).toLocaleString()} · {event.actor_type}{event.executor_type ? `/${event.executor_type}` : ""}{event.action_id ? ` · ${event.action_id}` : ""}{event.execution_id ? ` · BAS ${event.execution_id}` : ""}</li>)}</ul> : <p className="text-app-muted-foreground">Select or create an identity to review its recorded activity.</p>}</CardContent>
		</Card>
        <Card>
          <CardHeader><CardTitle>Release preview</CardTitle></CardHeader>
          <CardContent className="space-y-3">
			<label className="block text-sm font-medium" htmlFor="preview-platform">Platform descriptor</label>
			<select id="preview-platform" aria-label="Preview platform" value={selectedPreviewPlatform} onChange={(event) => setPreviewPlatform(event.target.value)} className="min-h-11 w-full rounded-control border border-app-border bg-app-surface px-3 py-2 text-sm">
				<option value="">Select platform</option>{configuredPlatforms.map((platform) => <option key={platform.id} value={platform.id}>{platform.id}</option>)}
			</select>
            <label className="block text-sm font-medium" htmlFor="preview-caption">Caption</label>
            <Textarea id="preview-caption" value={previewCaption} onChange={(event) => setPreviewCaption(event.target.value)} placeholder="Caption to preview" />
            <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={previewDisclosure} onChange={(event) => setPreviewDisclosure(event.target.checked)} /> Disclosure visible in rendered media</label>
            <Button variant="secondary" onClick={() => void renderPreview()}>Render platform preview</Button>
            {preview ? <div aria-live="polite" className="rounded-control border border-app-border p-2 text-sm"><p>{preview.release_allowed ? "Ready for release" : "Release blocked"}{preview.caption_truncated ? " · caption truncated" : ""}{preview.title_truncated ? " · title truncated" : ""}{preview.media_presentation && preview.media_presentation !== "native" ? ` · ${preview.media_presentation}` : ""}</p>{preview.blocking_errors.length ? <ul className="list-disc pl-5">{preview.blocking_errors.map((error) => <li key={error}>{error}</li>)}</ul> : null}</div> : null}
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
