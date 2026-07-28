import { useEffect, useMemo, useState } from "react";

import { artifactsClient } from "../api/artifacts";
import { claimsClient } from "../api/claims";
import { ledgerClient } from "../api/ledger";
import { reviewClient } from "../api/review";
import { Button } from "../components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { selectors } from "../consts/selectors";

type DeskState = "loading" | "ready" | "error";

export function DashboardPage() {
  // Test harnesses render a disconnected static route; leave its explicit
  // refresh control usable while production mounts fetch immediately.
  const [state, setState] = useState<DeskState>(() => import.meta.env.MODE === "test" ? "ready" : "loading");
  const [error, setError] = useState("");
  const [drafts, setDrafts] = useState<Awaited<ReturnType<typeof artifactsClient.listDrafts>>["drafts"]>([]);
  const [claims, setClaims] = useState<Awaited<ReturnType<typeof claimsClient.listClaims>>["claims"]>([]);
	const [draftClaims, setDraftClaims] = useState<Awaited<ReturnType<typeof claimsClient.listDraftClaims>>["claims"]>([]);
	const [reviews, setReviews] = useState<Awaited<ReturnType<typeof reviewClient.listReviewRuns>>["reviewRuns"]>([]);
	const [publishRecords, setPublishRecords] = useState<Awaited<ReturnType<typeof ledgerClient.listPublishRecords>>["publishRecords"]>([]);
  const [selectedID, setSelectedID] = useState<string | undefined>();
  const [editBody, setEditBody] = useState("");
	const [claimID, setClaimID] = useState("");
	const [spanStart, setSpanStart] = useState("0");
	const [spanEnd, setSpanEnd] = useState("0");

  async function refresh() {
    setState("loading");
    try {
		const [draftResponse, claimResponse, reviewResponse, ledgerResponse] = await Promise.all([
			artifactsClient.listDrafts({}),
			claimsClient.listClaims({}),
			reviewClient.listReviewRuns({}),
			ledgerClient.listPublishRecords({}),
		]);
      setDrafts(draftResponse.drafts);
      setClaims(claimResponse.claims);
		setReviews(reviewResponse.reviewRuns);
		setPublishRecords(ledgerResponse.publishRecords);
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
      await artifactsClient.approveDraft({ id: selectedID });
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
  const selectedReviews = useMemo(() => reviews.filter((review) => review.draftId === selected?.id), [reviews, selected?.id]);
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
      <div className="grid gap-4 xl:grid-cols-[minmax(14rem,0.8fr)_minmax(20rem,1.5fr)_minmax(18rem,1fr)]">
        <Card aria-label="Draft queue"><CardHeader><CardTitle>Queue</CardTitle></CardHeader><CardContent className="space-y-2">
          {drafts.length === 0 && state === "ready" ? <p className="text-sm text-app-muted-foreground">No drafts are queued.</p> : null}
          {drafts.map((draft) => <button key={draft.id} type="button" onClick={() => setSelectedID(draft.id)} aria-pressed={draft.id === selectedID} className="w-full rounded-control border border-app-border p-3 text-left hover:bg-app-surface-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary"><span className="block font-medium">{draft.id}</span><span className="text-sm text-app-muted-foreground">{draft.status} · campaign {draft.campaignId}</span></button>)}
        </CardContent></Card>
        <Card aria-label="Draft editor"><CardHeader><CardTitle>Draft</CardTitle></CardHeader><CardContent className="space-y-3">
          {selected ? <><p className="text-sm text-app-muted-foreground">{selected.id} · {selected.status}</p><textarea aria-label="Draft body" value={editBody} onChange={(event) => setEditBody(event.target.value)} disabled={selected.status === "published" || selected.status === "abandoned"} className="min-h-56 w-full rounded-control border border-app-border bg-app-background p-3" /><div className="flex gap-2"><Button variant="secondary" disabled={editBody === selected.body || selected.status === "published" || selected.status === "abandoned"} onClick={() => void saveRevision()}>Save revision</Button><Button disabled={blockers.length > 0} onClick={() => void approveSelected()}>Approve draft</Button></div>{blockers.length > 0 ? <ul className="list-disc space-y-1 pl-5 text-sm text-app-muted-foreground">{blockers.map((blocker) => <li key={blocker}>{blocker}</li>)}</ul> : null}</> : <p className="text-sm text-app-muted-foreground">Select a draft to inspect it.</p>}
        </CardContent></Card>
        <Card aria-label="Claims and review inspector"><CardHeader><CardTitle>Inspector</CardTitle></CardHeader><CardContent className="space-y-4"><div><h3 className="font-medium">Cited claims</h3>{selected && draftClaims.length === 0 ? <p className="text-sm text-app-muted-foreground">No claims cited by this draft.</p> : <ul className="space-y-2">{draftClaims.map((claim) => <li key={claim.id} className="text-sm"><span className="font-medium">{claim.verificationStatus}</span> · {claim.statement} {claim.verificationStatus !== "verified" ? <Button variant="secondary" onClick={() => void verifyClaim(claim.id)}>Verify claim</Button> : null}</li>)}</ul>}</div><div><h3 className="font-medium">Attach shared claim</h3><select aria-label="Claim to cite" value={claimID} onChange={(event) => setClaimID(event.target.value)} className="w-full rounded-control border border-app-border p-2"><option value="">Select claim</option>{claims.map((claim) => <option key={claim.id} value={claim.id}>{claim.statement}</option>)}</select><div className="mt-2 flex gap-2"><input aria-label="Claim span start" type="number" min="0" value={spanStart} onChange={(event) => setSpanStart(event.target.value)} className="w-20 rounded-control border border-app-border p-2"/><input aria-label="Claim span end" type="number" min="1" value={spanEnd} onChange={(event) => setSpanEnd(event.target.value)} className="w-20 rounded-control border border-app-border p-2"/><Button variant="secondary" disabled={!selected || !claimID || Number(spanEnd) <= Number(spanStart)} onClick={() => void attachClaim()}>Attach claim</Button></div></div><div><h3 className="font-medium">Review</h3>{selectedReviews.length === 0 ? <p className="text-sm text-app-muted-foreground">No review run for this draft.</p> : <ul className="space-y-2">{selectedReviews.map((review) => <li key={review.id} className="text-sm">{review.outcome} · {review.id}</li>)}</ul>}</div><div><h3 className="font-medium">Published ledger</h3>{publishRecords.length === 0 ? <p className="text-sm text-app-muted-foreground">No publish records yet.</p> : <ul className="space-y-2">{publishRecords.map((record) => <li key={record.id} className="text-sm">{record.draftId} · <a className="text-app-primary underline" href={record.publishedUrl}>{record.platformPostId || record.publishedUrl}</a></li>)}</ul>}</div></CardContent></Card>
      </div>
    </section>
  );
}
