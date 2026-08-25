import { useEffect, useMemo, useState } from "react";

import { artifactsClient } from "../api/artifacts";
import { claimsClient } from "../api/claims";
import { ledgerClient } from "../api/ledger";
import { reviewClient } from "../api/review";
import { Button } from "@vrooli/react-component-library/Button/1.2.0";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { selectors } from "../consts/selectors";

type DeskState = "loading" | "ready" | "error";

type EditorShape = "standard" | "thread" | "long-form";

// Post type, rather than a selected platform, owns the authoring shape. Keep
// this mapping intentionally small until the post-type registry exposes a
// richer authored layout descriptor.
export function editorShapeForPostType(postTypeID: string | undefined): EditorShape {
  const normalized = (postTypeID ?? "").trim().toLowerCase();
  if (normalized === "thread" || normalized.endsWith("-thread")) return "thread";
  if (normalized === "long-form" || normalized.endsWith("-long-form")) return "long-form";
  return "standard";
}

function ThreadEditorGuidance({ body }: { body: string }) {
  const segments = body.trim() === "" ? [""] : body.split(/\n\s*\n/);
  return <section aria-label="Thread authoring guidance" className="rounded-control border border-app-border p-3 text-sm">
    <h3 className="font-medium">Thread structure</h3>
    <p className="text-app-muted-foreground">Separate posts with a blank line. Each post has a 280-character budget.</p>
    <ol className="mt-2 list-decimal space-y-1 pl-5">
      {segments.map((segment, index) => <li key={`${index}-${segment.length}`}>Post {index + 1}: {segment.length}/280 characters</li>)}
    </ol>
  </section>;
}

function LongFormEditorGuidance() {
  return <section aria-label="Long-form authoring guidance" className="rounded-control border border-app-border p-3 text-sm">
    <h3 className="font-medium">Long-form structure</h3>
    <ul className="mt-2 list-disc space-y-1 pl-5 text-app-muted-foreground">
      <li>Use headings to state the argument structure.</li>
      <li>Reserve a banner image slot above the opening section.</li>
      <li>Mark inline figure positions with descriptive alt text.</li>
    </ul>
  </section>;
}

function AttachmentEditor({ assetID, role, aspectRatio, altText, position, disabled, onAssetID, onRole, onAspectRatio, onAltText, onPosition, onAttach }: { assetID: string; role: string; aspectRatio: string; altText: string; position: string; disabled: boolean; onAssetID(value: string): void; onRole(value: string): void; onAspectRatio(value: string): void; onAltText(value: string): void; onPosition(value: string): void; onAttach(): void }) {
  return <section aria-label="Released asset attachment" className="rounded-control border border-app-border p-3 text-sm">
    <h3 className="font-medium">Released Asset Studio attachment</h3>
    <p className="text-app-muted-foreground">Attach only a released Asset Studio reference. Image bytes do not enter Content Desk.</p>
    <div className="mt-2 grid gap-2 sm:grid-cols-2"><input aria-label="Released asset id" value={assetID} onChange={(event) => onAssetID(event.target.value)} placeholder="Asset ID" className="rounded-control border border-app-border p-2" /><input aria-label="Attachment role" value={role} onChange={(event) => onRole(event.target.value)} placeholder="Role, e.g. hero" className="rounded-control border border-app-border p-2" /><input aria-label="Attachment aspect ratio" value={aspectRatio} onChange={(event) => onAspectRatio(event.target.value)} placeholder="16:9" className="rounded-control border border-app-border p-2" /><input aria-label="Attachment position" type="number" min="0" value={position} onChange={(event) => onPosition(event.target.value)} className="rounded-control border border-app-border p-2" /></div>
    <input aria-label="Attachment alt text" value={altText} onChange={(event) => onAltText(event.target.value)} placeholder="Accessible image description" className="mt-2 w-full rounded-control border border-app-border p-2" />
    <Button className="mt-2" variant="secondary" disabled={disabled || !assetID || !role || !aspectRatio || !altText} onClick={onAttach}>Attach released asset</Button>
  </section>;
}

export function DashboardPage() {
  // Test harnesses render a disconnected static route; leave its explicit
  // refresh control usable while production mounts fetch immediately.
  const [state, setState] = useState<DeskState>(() => import.meta.env.MODE === "test" ? "ready" : "loading");
  const [error, setError] = useState("");
  const [drafts, setDrafts] = useState<Awaited<ReturnType<typeof artifactsClient.listDrafts>>["drafts"]>([]);
  const [claims, setClaims] = useState<Awaited<ReturnType<typeof claimsClient.listClaims>>["claims"]>([]);
	const [draftClaims, setDraftClaims] = useState<Awaited<ReturnType<typeof claimsClient.listDraftClaims>>["claims"]>([]);
	const [coverage, setCoverage] = useState<Awaited<ReturnType<typeof claimsClient.getClaimCoverage>> | null>(null);
	const [claimProposals, setClaimProposals] = useState<Awaited<ReturnType<typeof claimsClient.listClaimProposals>>["proposals"]>([]);
	const [attachments, setAttachments] = useState<Awaited<ReturnType<typeof artifactsClient.listDraftAttachments>>["attachments"]>([]);
	const [attachmentAssetID, setAttachmentAssetID] = useState("");
	const [attachmentRole, setAttachmentRole] = useState("hero");
	const [attachmentAspectRatio, setAttachmentAspectRatio] = useState("16:9");
	const [attachmentAltText, setAttachmentAltText] = useState("");
	const [attachmentPosition, setAttachmentPosition] = useState("0");
	const [reviews, setReviews] = useState<Awaited<ReturnType<typeof reviewClient.listReviewRuns>>["reviewRuns"]>([]);
	const [publishRecords, setPublishRecords] = useState<Awaited<ReturnType<typeof ledgerClient.listPublishRecords>>["publishRecords"]>([]);
	const [remediations, setRemediations] = useState<Awaited<ReturnType<typeof ledgerClient.listRemediations>>["remediations"]>([]);
  const [selectedID, setSelectedID] = useState<string | undefined>();
  const [editBody, setEditBody] = useState("");
	const [claimID, setClaimID] = useState("");
	const [spanStart, setSpanStart] = useState("0");
	const [spanEnd, setSpanEnd] = useState("0");
	const [releaseIdentity, setReleaseIdentity] = useState("");
	const [releaseLane, setReleaseLane] = useState("");
	const [releaseKey, setReleaseKey] = useState("");
	const [releaseDisclosureVisible, setReleaseDisclosureVisible] = useState(false);
	const [releaseMessage, setReleaseMessage] = useState("");
	const [approvalIdentity, setApprovalIdentity] = useState("");
	const [approvalLane, setApprovalLane] = useState("");
	const [remediationRecordID, setRemediationRecordID] = useState("");
	const [remediationKind, setRemediationKind] = useState("correct_in_place");
	const [remediationNote, setRemediationNote] = useState("");
	const [agentMessage, setAgentMessage] = useState("");
	const [agentCommissionID, setAgentCommissionID] = useState("");

  async function refresh() {
    setState("loading");
    try {
		const [draftResponse, claimResponse, reviewResponse, ledgerResponse, remediationResponse] = await Promise.all([
			artifactsClient.listDrafts({}),
			claimsClient.listClaims({}),
			reviewClient.listReviewRuns({}),
			ledgerClient.listPublishRecords({}),
			ledgerClient.listRemediations({ openOnly: true }),
		]);
      setDrafts(draftResponse.drafts);
      setClaims(claimResponse.claims);
		setReviews(reviewResponse.reviewRuns);
		setPublishRecords(ledgerResponse.publishRecords);
		setRemediations(remediationResponse.remediations);
      setSelectedID((current) => current ?? draftResponse.drafts[0]?.id);
      setState("ready");
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "Unable to load the content desk.");
      setState("error");
    }
  }

  async function approveSelected() {
    if (!selectedID) return;
    try {
		await artifactsClient.approveDraft({ id: selectedID, identityId: approvalIdentity, lane: approvalLane });
      await refresh();
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "Unable to approve the draft.");
      setState("error");
    }
  }

  async function saveRevision() {
    if (!selectedID) return;
    try {
      await artifactsClient.updateDraftBody({ id: selectedID, body: editBody });
      await refresh();
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "Unable to save the draft revision.");
      setState("error");
    }
  }

	async function commissionAgentWork(action: "draft" | "evidence-hunt" | "review") {
		if (!selectedID) return;
		try {
			const receipt = await artifactsClient.commissionAgentWork({ draftId: selectedID, action });
			setAgentCommissionID(receipt.commissionId);
			setAgentMessage(`Agent Manager run ${receipt.runId} was commissioned for ${action}. Its result remains an editable suggestion and cannot approve or publish.`);
		} catch (caught) { setAgentMessage(caught instanceof Error ? caught.message : "Unable to commission Agent Manager work."); }
	}

	async function loadAgentResult() {
		if (!agentCommissionID) return;
		try {
			const result = await artifactsClient.getAgentWorkResult({ commissionId: agentCommissionID });
			if (result.body) setEditBody(result.body);
			setAgentMessage(result.body ? `Loaded Agent Manager run ${result.runId}; review and edit the suggestion before adopting it.` : `Agent Manager run ${result.runId} is ${result.status}; no terminal suggestion is available yet.`);
		} catch (caught) { setAgentMessage(caught instanceof Error ? caught.message : "Unable to load Agent Manager result."); }
	}

	async function adoptAgentSuggestion() {
		if (!agentCommissionID) return;
		try { await artifactsClient.adoptAgentSuggestion({ commissionId: agentCommissionID, body: editBody }); setAgentMessage("Agent suggestion adopted as an operator-editable draft revision."); await refresh(); }
		catch (caught) { setAgentMessage(caught instanceof Error ? caught.message : "Unable to adopt Agent Manager suggestion."); }
	}

	async function attachReleasedAsset() {
		if (!selectedID) return;
		try {
			await artifactsClient.attachReleasedAsset({ draftId: selectedID, assetId: attachmentAssetID, role: attachmentRole, aspectRatio: attachmentAspectRatio, altText: attachmentAltText, position: Number(attachmentPosition) });
			setAttachments((await artifactsClient.listDraftAttachments({ draftId: selectedID })).attachments);
			setAttachmentAssetID(""); setAttachmentAltText("");
		} catch (caught) { setError(caught instanceof Error ? caught.message : "Unable to attach the released asset."); setState("error"); }
	}

  async function attachClaim() {
	if (!selectedID || !claimID) return;
	try {
		await claimsClient.citeClaim({ draftId: selectedID, claimId: claimID, spanStart: Number(spanStart), spanEnd: Number(spanEnd), body: editBody });
		const response = await claimsClient.listDraftClaims({ draftId: selectedID });
		setDraftClaims(response.claims);
  } catch (caught) {
		setError(caught instanceof Error ? caught.message : "Unable to attach the claim.");
		setState("error");
	}
  }

  async function verifyClaim(id: string) {
	try {
		await claimsClient.verifyClaim({ id });
		if (selectedID) { const response = await claimsClient.listDraftClaims({ draftId: selectedID }); setDraftClaims(response.claims); }
  } catch (caught) {
		setError(caught instanceof Error ? caught.message : "Unable to verify the claim.");
		setState("error");
	}
  }

	async function extractClaims() {
		if (!selectedID) return;
		try {
			setClaimProposals((await claimsClient.extractClaimProposals({ draftId: selectedID, body: editBody })).proposals);
		} catch (caught) {
			setError(caught instanceof Error ? caught.message : "Unable to extract claim proposals.");
			setState("error");
		}
	}

	async function decideClaimProposal(id: string, status: "accepted" | "rejected") {
		try {
			await claimsClient.decideClaimProposal({ id, status });
			if (selectedID) setClaimProposals((await claimsClient.listClaimProposals({ draftId: selectedID })).proposals);
		} catch (caught) {
			setError(caught instanceof Error ? caught.message : "Unable to decide the claim proposal.");
			setState("error");
		}
	}

  async function submitRelease() {
	if (!selectedID) return;
	try {
		const response = await artifactsClient.submitReleaseDraft({ id: selectedID, identityId: releaseIdentity, lane: releaseLane, idempotencyKey: releaseKey, disclosureVisible: releaseDisclosureVisible });
		setReleaseMessage(`Queued in Channel Manager: ${response.releaseId} (${response.releaseStatus}). Publication is recorded only after its outcome returns.`);
		await refresh();
	} catch (caught) {
		setReleaseMessage(caught instanceof Error ? caught.message : "Could not submit this release to Channel Manager.");
	}
  }

	async function createRemediation() {
		if (!remediationRecordID || !remediationKind) return;
		try {
			await ledgerClient.createRemediation({ publishRecordId: remediationRecordID, kind: remediationKind, note: remediationNote });
			setRemediationNote("");
			await refresh();
		} catch (caught) {
			setError(caught instanceof Error ? caught.message : "Unable to record the remediation.");
			setState("error");
		}
	}

	async function resolveRemediation(id: string) {
		try {
			await ledgerClient.resolveRemediation({ id });
			await refresh();
		} catch (caught) {
			setError(caught instanceof Error ? caught.message : "Unable to resolve the remediation.");
			setState("error");
		}
	}

  useEffect(() => {
    // The accessibility shell harness deliberately renders static routes
    // without an API mock. Runtime mounts load immediately; tests exercise
    // loading explicitly through the Refresh desk control.
    if (import.meta.env.MODE !== "test") void refresh();
  }, []);
  const selected = useMemo(() => drafts.find((draft) => draft.id === selectedID), [drafts, selectedID]);
  useEffect(() => { setEditBody(selected?.body ?? ""); }, [selected?.id, selected?.body]);
  useEffect(() => {
    if (!selectedID) { setDraftClaims([]); return; }
    void claimsClient.listDraftClaims({ draftId: selectedID }).then((response) => setDraftClaims(response.claims)).catch((caught) => setError(caught instanceof Error ? caught.message : "Unable to load draft claims."));
  }, [selectedID]);
	useEffect(() => {
		if (!selectedID) { setClaimProposals([]); return; }
		void claimsClient.listClaimProposals({ draftId: selectedID }).then((response) => setClaimProposals(response.proposals)).catch(() => setClaimProposals([]));
	}, [selectedID]);
	useEffect(() => {
		if (!selectedID) { setAttachments([]); return; }
		void artifactsClient.listDraftAttachments({ draftId: selectedID }).then((response) => setAttachments(response.attachments)).catch(() => setAttachments([]));
	}, [selectedID]);
	useEffect(() => {
		if (!selected) { setCoverage(null); return; }
		void claimsClient.getClaimCoverage({ draftId: selected.id, body: editBody || selected.body }).then(setCoverage).catch(() => setCoverage(null));
	}, [selected?.id, selected?.body, editBody]);
  const selectedReviews = useMemo(() => reviews.filter((review) => review.draftId === selected?.id), [reviews, selected?.id]);
  const editorShape = editorShapeForPostType(selected?.postTypeId);
  const unverifiedClaims = draftClaims.filter((claim) => claim.verificationStatus !== "verified");
  const blockers = [
    ...(selected?.status !== "reviewed" ? ["Draft must complete review before approval."] : []),
    ...unverifiedClaims.map((claim) => `Claim ${claim.id} is ${claim.verificationStatus}.`),
    ...(selectedReviews.some((review) => review.outcome !== "passed") ? ["A review run has blocking verdicts."] : []),
  ];

  return (
    <section data-testid={selectors.pages.dashboard} aria-labelledby="desk-heading" className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div><h2 id="desk-heading" className="text-2xl font-semibold">Content Desk</h2><p className="text-app-muted-foreground">Verified editorial production ledger</p></div>
        <Button variant="secondary" onClick={() => void refresh()} disabled={state === "loading"}>Refresh desk</Button>
      </div>
      {state === "error" ? <p role="alert" className="rounded-control border border-app-danger p-3 text-sm">{error}</p> : null}
      {selected ? <AttachmentEditor assetID={attachmentAssetID} role={attachmentRole} aspectRatio={attachmentAspectRatio} altText={attachmentAltText} position={attachmentPosition} disabled={selected.status === "published" || selected.status === "abandoned"} onAssetID={setAttachmentAssetID} onRole={setAttachmentRole} onAspectRatio={setAttachmentAspectRatio} onAltText={setAttachmentAltText} onPosition={setAttachmentPosition} onAttach={() => void attachReleasedAsset()} /> : null}
      {attachments.length ? <p className="text-sm text-app-muted-foreground">{attachments.length} released asset attachment{attachments.length === 1 ? "" : "s"}.</p> : null}
      {agentCommissionID ? <section aria-label="Agent Manager result" className="rounded-control border border-app-border p-3 text-sm"><h3 className="font-medium">Agent Manager result</h3><p className="text-app-muted-foreground">Load the terminal suggestion into the editable draft, then explicitly adopt it. Neither action approves or publishes.</p><div className="mt-2 flex gap-2"><Button variant="secondary" onClick={() => void loadAgentResult()}>Load agent result</Button><Button variant="secondary" onClick={() => void adoptAgentSuggestion()}>Adopt agent suggestion</Button></div></section> : null}
      <div className="grid gap-4 xl:grid-cols-[minmax(14rem,0.8fr)_minmax(20rem,1.5fr)_minmax(18rem,1fr)]">
        <Card aria-label="Draft queue"><CardHeader><CardTitle>Queue</CardTitle></CardHeader><CardContent className="space-y-2">
          {drafts.length === 0 && state === "ready" ? <p className="text-sm text-app-muted-foreground">No drafts are queued.</p> : null}
          {drafts.map((draft) => <button key={draft.id} type="button" onClick={() => setSelectedID(draft.id)} aria-pressed={draft.id === selectedID} className="w-full rounded-control border border-app-border p-3 text-left hover:bg-app-surface-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary"><span className="block font-medium">{draft.id}</span><span className="text-sm text-app-muted-foreground">{draft.status} · campaign {draft.campaignId}</span></button>)}
        </CardContent></Card>
        <Card aria-label="Draft editor"><CardHeader><CardTitle>Draft</CardTitle></CardHeader><CardContent className="space-y-3">
		  {selected ? <><p className="text-sm text-app-muted-foreground">{selected.id} · {selected.status} · {selected.postTypeId || "unspecified post type"}</p>{editorShape === "thread" ? <ThreadEditorGuidance body={editBody} /> : null}{editorShape === "long-form" ? <LongFormEditorGuidance /> : null}<textarea aria-label="Draft body" value={editBody} onChange={(event) => setEditBody(event.target.value)} disabled={selected.status === "published" || selected.status === "abandoned"} className="min-h-56 w-full rounded-control border border-app-border bg-app-background p-3" /><div className="flex gap-2"><Button variant="secondary" disabled={editBody === selected.body || selected.status === "published" || selected.status === "abandoned"} onClick={() => void saveRevision()}>Save revision</Button><Button disabled={blockers.length > 0 || (!!approvalIdentity !== !!approvalLane)} onClick={() => void approveSelected()}>Approve draft</Button></div><div className="rounded-control border border-app-border p-3"><h3 className="font-medium">Governed workbench assistance</h3><p className="text-sm text-app-muted-foreground">Commission a read-only Agent Manager suggestion. It cannot approve or publish this draft.</p><div className="mt-2 flex flex-wrap gap-2"><Button variant="secondary" onClick={() => void commissionAgentWork("draft")}>Draft suggestion</Button><Button variant="secondary" onClick={() => void commissionAgentWork("evidence-hunt")}>Find evidence</Button><Button variant="secondary" onClick={() => void commissionAgentWork("review")}>Review suggestion</Button></div>{agentMessage ? <p aria-live="polite" className="mt-2 text-sm">{agentMessage}</p> : null}</div>{selected.status === "reviewed" ? <div className="grid gap-2 sm:grid-cols-2"><input aria-label="Approval target identity" value={approvalIdentity} onChange={(event) => setApprovalIdentity(event.target.value)} placeholder="Optional Channel Manager identity" className="rounded-control border border-app-border p-2" /><input aria-label="Approval target lane" value={approvalLane} onChange={(event) => setApprovalLane(event.target.value)} placeholder="Matching lane" className="rounded-control border border-app-border p-2" /></div> : null}{blockers.length > 0 ? <ul className="list-disc space-y-1 pl-5 text-sm text-app-muted-foreground">{blockers.map((blocker) => <li key={blocker}>{blocker}</li>)}</ul> : null}{selected.status === "approved" ? <div className="space-y-2 rounded-control border border-app-border p-3"><h3 className="font-medium">Release through Channel Manager</h3><p className="text-sm text-app-muted-foreground">Queue an approved draft against an eligible identity. This is not proof of publication. Persona-actor identities require an operator-confirmed visible disclosure from the Channel Manager preview.</p><div className="grid gap-2 sm:grid-cols-3"><input aria-label="Channel Manager identity" value={releaseIdentity} onChange={(event) => setReleaseIdentity(event.target.value)} placeholder="Identity ID" className="rounded-control border border-app-border p-2" /><input aria-label="Channel Manager lane" value={releaseLane} onChange={(event) => setReleaseLane(event.target.value)} placeholder="Lane" className="rounded-control border border-app-border p-2" /><input aria-label="Release idempotency key" value={releaseKey} onChange={(event) => setReleaseKey(event.target.value)} placeholder="Stable key" className="rounded-control border border-app-border p-2" /></div><label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={releaseDisclosureVisible} onChange={(event) => setReleaseDisclosureVisible(event.target.checked)} /> I verified required disclosure is visible in the Channel Manager preview</label><Button disabled={!releaseIdentity || !releaseLane || !releaseKey} onClick={() => void submitRelease()}>Submit release</Button>{releaseMessage ? <p aria-live="polite" className="text-sm">{releaseMessage}</p> : null}</div> : null}</> : <p className="text-sm text-app-muted-foreground">Select a draft to inspect it.</p>}
        </CardContent></Card>
		<Card aria-label="Claims and review inspector"><CardHeader><CardTitle>Inspector</CardTitle></CardHeader><CardContent className="space-y-4"><div><h3 className="font-medium">Cited claims</h3>{selected && draftClaims.length === 0 ? <p className="text-sm text-app-muted-foreground">No claims cited by this draft.</p> : <ul className="space-y-2">{draftClaims.map((claim) => <li key={claim.id} className="text-sm"><span className="font-medium">{claim.verificationStatus}</span> · {claim.statement} {claim.verificationStatus !== "verified" ? <Button variant="secondary" onClick={() => void verifyClaim(claim.id)}>Verify claim</Button> : null}</li>)}</ul>}</div><div><h3 className="font-medium">Assisted claim extraction</h3><p className="text-sm text-app-muted-foreground">Extraction proposes text for review. It never creates or verifies a claim.</p><Button variant="secondary" disabled={!selected} onClick={() => void extractClaims()}>Extract claim proposals</Button>{claimProposals.length ? <ul className="mt-2 space-y-2">{claimProposals.map((proposal) => <li key={proposal.id} className="rounded-control border border-app-border p-2 text-sm">{proposal.statement}<p className="text-app-muted-foreground">{proposal.status}</p>{proposal.status === "proposed" ? <div className="flex gap-2"><Button variant="secondary" onClick={() => void decideClaimProposal(proposal.id, "accepted")}>Accept proposal</Button><Button variant="secondary" onClick={() => void decideClaimProposal(proposal.id, "rejected")}>Reject proposal</Button></div> : null}</li>)}</ul> : null}</div><div><h3 className="font-medium">Claim coverage</h3>{coverage ? <p aria-live="polite" className="text-sm">{coverage.supportedSpans.length} supported span{coverage.supportedSpans.length === 1 ? "" : "s"}; {coverage.uncoveredSpans.length} uncovered span{coverage.uncoveredSpans.length === 1 ? "" : "s"}. Review uncovered assertions before approval.</p> : <p className="text-sm text-app-muted-foreground">Coverage will appear when a draft is selected.</p>}</div><div><h3 className="font-medium">Attach shared claim</h3><select aria-label="Claim to cite" value={claimID} onChange={(event) => setClaimID(event.target.value)} className="w-full rounded-control border border-app-border p-2"><option value="">Select claim</option>{claims.map((claim) => <option key={claim.id} value={claim.id}>{claim.statement}</option>)}</select><div className="mt-2 flex gap-2"><input aria-label="Claim span start" type="number" min="0" value={spanStart} onChange={(event) => setSpanStart(event.target.value)} className="w-20 rounded-control border border-app-border p-2"/><input aria-label="Claim span end" type="number" min="1" value={spanEnd} onChange={(event) => setSpanEnd(event.target.value)} className="w-20 rounded-control border border-app-border p-2"/><Button variant="secondary" disabled={!selected || !claimID || Number(spanEnd) <= Number(spanStart)} onClick={() => void attachClaim()}>Attach claim</Button></div></div><div><h3 className="font-medium">Review</h3>{selectedReviews.length === 0 ? <p className="text-sm text-app-muted-foreground">No review run for this draft.</p> : <ul className="space-y-2">{selectedReviews.map((review) => <li key={review.id} className="text-sm">{review.outcome} · {review.id}</li>)}</ul>}</div><div><h3 className="font-medium">Published ledger</h3>{publishRecords.length === 0 ? <p className="text-sm text-app-muted-foreground">No publish records yet.</p> : <ul className="space-y-2">{publishRecords.map((record) => <li key={record.id} className="text-sm">{record.draftId} · <a className="text-app-primary underline" href={record.publishedUrl}>{record.platformPostId || record.publishedUrl}</a></li>)}</ul>}</div><div><h3 className="font-medium">Contamination remediation</h3><p className="text-sm text-app-muted-foreground">Record the corrective action against the published record; resolving it preserves its history.</p><input aria-label="Remediation publish record" value={remediationRecordID} onChange={(event) => setRemediationRecordID(event.target.value)} placeholder="Publish record ID" className="mt-2 w-full rounded-control border border-app-border p-2"/><select aria-label="Remediation kind" value={remediationKind} onChange={(event) => setRemediationKind(event.target.value)} className="mt-2 w-full rounded-control border border-app-border p-2"><option value="correct_in_place">Correct in place</option><option value="publish_correction">Publish correction</option><option value="retract">Retract</option><option value="accept_and_annotate">Accept and annotate</option></select><textarea aria-label="Remediation note" value={remediationNote} onChange={(event) => setRemediationNote(event.target.value)} placeholder="Evidence and operator rationale" className="mt-2 min-h-20 w-full rounded-control border border-app-border p-2"/><Button variant="secondary" disabled={!remediationRecordID || !remediationKind} onClick={() => void createRemediation()}>Record remediation</Button>{remediations.length ? <ul className="mt-2 space-y-2">{remediations.map((remediation) => <li key={remediation.id} className="rounded-control border border-app-border p-2 text-sm"><strong>{remediation.kind}</strong> · {remediation.publishRecordId}<p>{remediation.note}</p><Button variant="secondary" onClick={() => void resolveRemediation(remediation.id)}>Resolve remediation</Button></li>)}</ul> : <p className="mt-2 text-sm text-app-muted-foreground">No open remediations.</p>}</div></CardContent></Card>
      </div>
    </section>
  );
}
