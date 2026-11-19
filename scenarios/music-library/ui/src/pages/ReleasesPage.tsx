import { useEffect, useState } from "react";
import { getRelease, listReleases, recordReleaseRightsDecision, recordReleaseReview, saveReleaseDraft, type ReleaseAssessment, type ReleasePage, type ReleaseTrackRights } from "../api/releases";
import { sourceAudioUrl } from "../api/decompositions";

const controlClass = "mt-1 w-full rounded-md border border-app-border bg-app-background p-2";

export function ReleasesPage() {
  const [releaseID, setReleaseID] = useState("");
  const [title, setTitle] = useState("");
  const [jaTitle, setJaTitle] = useState("");
  const [artistOwner, setArtistOwner] = useState("");
  const [artistSubjectID, setArtistSubjectID] = useState("");
  const [artistSubjectVersion, setArtistSubjectVersion] = useState("");
  const [hashes, setHashes] = useState("");
  const [creditsJSON, setCreditsJSON] = useState("[]");
  const [assetsJSON, setAssetsJSON] = useState("[]");
  const [rightsJSON, setRightsJSON] = useState("[]");
  const [derivationsJSON, setDerivationsJSON] = useState("[]");
  const [reviewNote, setReviewNote] = useState("");
  const [reviewDecision, setReviewDecision] = useState<"approved" | "changes_requested">("approved");
  const [heardAllTracks, setHeardAllTracks] = useState(false);
  const [reviewedSourceRights, setReviewedSourceRights] = useState(false);
  const [assessment, setAssessment] = useState<ReleaseAssessment>();
  const [page, setPage] = useState<ReleasePage>();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const trackHashes = hashes.split(/\r?\n/).map(hash => hash.trim()).filter(Boolean);

  useEffect(() => {
    let active = true;
    void listReleases().then(result => { if (active) setPage(result); })
      .catch(cause => { if (active) setError(cause instanceof Error ? cause.message : "Could not list release drafts."); });
    return () => { active = false; };
  }, []);

  const run = async (work: () => Promise<void>) => {
    setBusy(true); setError("");
    try { await work(); } catch (cause) { setError(cause instanceof Error ? cause.message : "Request failed"); }
    finally { setBusy(false); }
  };

  const save = () => run(async () => {
    const id = releaseID.trim();
    const cleanTitle = title.trim();
    if (!id || !cleanTitle || trackHashes.length < 1) throw new Error("Enter a release ID, title, and one or more exact track hashes.");
    const current = assessment?.edition.release_id === id ? assessment.edition.revision : 0;
    const localizedMetadata: Record<string, { title: string }> = { en: { title: cleanTitle } };
    if (jaTitle.trim()) localizedMetadata.ja = { title: jaTitle.trim() };
    const created = await saveReleaseDraft({
      release_id: id, revision: current + 1, kind: "ep", status: "draft", title: cleanTitle, default_locale: "en",
      artist_owner: artistOwner.trim(), artist_subject_id: artistSubjectID.trim(), artist_subject_version: artistSubjectVersion.trim(),
      localized_metadata: localizedMetadata,
      tracks: trackHashes.map((track_sha256, index) => ({ order: index + 1, track_sha256 })),
      credits: parseArray(creditsJSON, "credits"), assets: parseArray(assetsJSON, "assets"),
      track_rights: parseTrackRights(rightsJSON), derivations: parseArray(derivationsJSON, "derivations"),
    });
    setAssessment(await getRelease(created.edition.release_id, created.edition.revision));
    setPage(await listReleases());
  });

  const load = () => run(async () => {
    if (!releaseID.trim()) throw new Error("Enter a release ID to load.");
    const next = await getRelease(releaseID.trim());
    populate(next);
  });

  const populate = (next: ReleaseAssessment) => {
    setAssessment(next);
    setHeardAllTracks(false);
    setReviewedSourceRights(false);
    setTitle(next.edition.title);
    setJaTitle(next.edition.localized_metadata.ja?.title ?? "");
    setArtistOwner(next.edition.artist_owner ?? "");
    setArtistSubjectID(next.edition.artist_subject_id ?? "");
    setArtistSubjectVersion(next.edition.artist_subject_version ?? "");
    setHashes(next.edition.tracks.map(track => track.track_sha256).join("\n"));
    setCreditsJSON(JSON.stringify(next.edition.credits, null, 2));
    setAssetsJSON(JSON.stringify(next.edition.assets, null, 2));
    setRightsJSON(JSON.stringify(next.edition.track_rights, null, 2));
    setDerivationsJSON(JSON.stringify(next.edition.derivations, null, 2));
  };

  const review = () => run(async () => {
    if (!assessment?.eligibility.ready_for_operator_judgment) throw new Error("This exact revision is not ready for operator review.");
    if (!heardAllTracks) throw new Error("Confirm that you listened to every ordered track before recording the batch decision.");
    await recordReleaseReview(assessment.edition.release_id, {
      review_id: `review-${crypto.randomUUID()}`,
      revision: assessment.edition.revision,
      decision: reviewDecision,
      heard_all_tracks: heardAllTracks,
      reviewed_track_sha256s: assessment.edition.tracks.map(track => track.track_sha256),
      note: reviewNote.trim(),
    });
    setAssessment(await getRelease(assessment.edition.release_id, assessment.edition.revision));
    setReviewNote("");
    setHeardAllTracks(false);
  });

  const confirmRights = () => run(async () => {
    if (!assessment) throw new Error("Load a release revision first.");
    if (!reviewedSourceRights) throw new Error("Confirm that you inspected each pinned source receipt and its use terms.");
    const { edition } = assessment;
    const rights = edition.track_rights ?? [];
    if (rights.length !== edition.tracks.length) throw new Error("Every exact track hash needs a source license and receipt reference before rights can be recorded.");
    const rightsByHash = new Map(rights.map(item => [item.track_sha256, item]));
    if (rightsByHash.size !== edition.tracks.length || edition.tracks.some(track => !rightsByHash.has(track.track_sha256))) throw new Error("Rights entries must cover each exact track hash once.");
    const normalized = edition.tracks.map(track => {
      const item = rightsByHash.get(track.track_sha256);
      if (!item) throw new Error("A track is missing its rights entry.");
      const sourceLicense = item.source_license?.trim();
      const evidenceRef = item.evidence_ref?.trim();
      if (!sourceLicense || !evidenceRef || !["personal_only", "commercial", "blocked"].includes(item.usage_scope)) throw new Error("Rights entries must match an exact track hash and include its source license, use scope, and receipt reference.");
      return { item, sourceLicense, evidenceRef, usageScope: item.usage_scope as "personal_only" | "commercial" | "blocked" };
    });
    for (const { item, sourceLicense, evidenceRef, usageScope } of normalized) {
      await recordReleaseRightsDecision(edition.release_id, {
        decision_id: `rights-${crypto.randomUUID()}`, revision: edition.revision, track_sha256: item.track_sha256,
        source_license: sourceLicense, usage_scope: usageScope,
        decision: usageScope === "blocked" ? "denied" : "confirmed", evidence_ref: evidenceRef, note: item.note ?? "",
      });
    }
    setAssessment(await getRelease(edition.release_id, edition.revision));
    setReviewedSourceRights(false);
  });

  return <main className="mx-auto w-full max-w-5xl px-4 py-6 sm:px-6 lg:px-8">
    <header className="mb-6">
      <p className="text-sm font-medium text-app-accent">MQ6 · local release catalog</p>
      <h1 className="mt-1 text-3xl font-semibold">Release workbench</h1>
      <p className="mt-2 max-w-3xl text-sm text-app-muted-foreground">Build an immutable draft from exact catalog hashes. This editor does not generate, copy, or publish audio. An evidence check marks what exists and what still needs the canonical artist, artwork, rights, and lineage owners.</p>
    </header>
    {error && <p role="alert" className="mb-4 rounded-md border border-red-300 bg-red-50 p-3 text-sm text-red-900">{error}</p>}
    <section className="mb-5 rounded-xl border border-app-border bg-app-surface p-4 sm:p-5">
      <div className="flex flex-wrap items-center justify-between gap-2"><h2 className="text-xl font-semibold">Release catalog</h2><button disabled={busy} onClick={() => void run(async () => setPage(await listReleases()))} className="rounded-md border px-3 py-2 text-sm">Refresh list</button></div>
      {!page?.items.length ? <p className="mt-3 text-sm text-app-muted-foreground">No release drafts exist yet.</p> : <ul className="mt-3 grid gap-2 sm:grid-cols-2">{page.items.map(item => <li key={item.release_id} className="flex items-center justify-between gap-2 rounded-lg border border-app-border p-3"><span className="min-w-0"><span className="block truncate font-medium">{item.title}</span><span className="block break-all text-xs text-app-muted-foreground">{item.release_id} · revision {item.revision} · {item.track_count} tracks</span></span><button disabled={busy} onClick={() => { setReleaseID(item.release_id); void run(async () => populate(await getRelease(item.release_id, item.revision))); }} className="shrink-0 rounded border px-3 py-2 text-sm">Open</button></li>)}</ul>}
      {!!page?.next_cursor && <button disabled={busy} onClick={() => void run(async () => setPage(await listReleases(page.next_cursor)))} className="mt-3 rounded-md border px-3 py-2 text-sm">Next page</button>}
    </section>
    <div className="grid gap-5 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
      <section className="rounded-xl border border-app-border bg-app-surface p-4 sm:p-5">
        <div className="flex items-center justify-between gap-2"><h2 className="text-xl font-semibold">Draft an EP</h2><button disabled={busy || !releaseID.trim()} onClick={() => void load()} className="rounded-md border px-3 py-2 text-sm">Load latest</button></div>
        <div className="mt-4 grid gap-3">
          <label className="text-sm">Release ID<input value={releaseID} onChange={event => setReleaseID(event.target.value)} className={controlClass} maxLength={160} /></label>
          <label className="text-sm">Release title<input value={title} onChange={event => setTitle(event.target.value)} className={controlClass} maxLength={512} /></label>
          <label className="text-sm">Japanese title (leave empty until operator-written)<input value={jaTitle} onChange={event => setJaTitle(event.target.value)} className={controlClass} maxLength={512} /></label>
          <fieldset className="grid gap-3 rounded-lg border border-app-border p-3 sm:grid-cols-3"><legend className="px-1 text-sm font-medium">Pinned fictional artist subject</legend>
            <label className="text-xs">Owner<input value={artistOwner} onChange={event => setArtistOwner(event.target.value)} className={controlClass} /></label>
            <label className="text-xs">Subject ID<input value={artistSubjectID} onChange={event => setArtistSubjectID(event.target.value)} className={controlClass} /></label>
            <label className="text-xs">Version<input value={artistSubjectVersion} onChange={event => setArtistSubjectVersion(event.target.value)} className={controlClass} /></label>
          </fieldset>
          <label className="text-sm">Ordered track SHA-256 values, one per line<textarea value={hashes} onChange={event => setHashes(event.target.value)} rows={7} spellCheck={false} className={controlClass + " font-mono text-xs"} placeholder="Paste exact catalog hashes in release order" /></label>
          <label className="text-sm">Credits JSON<textarea value={creditsJSON} onChange={event => setCreditsJSON(event.target.value)} rows={4} spellCheck={false} className={controlClass + " font-mono text-xs"} /></label>
          <label className="text-sm">Existing owner artwork references JSON<textarea value={assetsJSON} onChange={event => setAssetsJSON(event.target.value)} rows={4} spellCheck={false} className={controlClass + " font-mono text-xs"} /></label>
          <label className="text-sm">Operator rights decisions JSON<textarea value={rightsJSON} onChange={event => setRightsJSON(event.target.value)} rows={4} spellCheck={false} className={controlClass + " font-mono text-xs"} /></label>
          <label className="text-sm">Existing transformation lineage JSON<textarea value={derivationsJSON} onChange={event => setDerivationsJSON(event.target.value)} rows={4} spellCheck={false} className={controlClass + " font-mono text-xs"} /></label>
          <p className="text-xs text-app-muted-foreground">The EP assessment requires four to six distinct exact hashes. Every track hash must already exist in the local catalog. Nested JSON is evidence input, not proof: artist and asset adapters must resolve their owner references before operator review.</p>
          <button disabled={busy} onClick={() => void save()} className="rounded-md bg-app-primary px-4 py-2 font-semibold text-app-primary-foreground">Save next immutable draft revision</button>
        </div>
      </section>
      <section className="rounded-xl border border-app-border bg-app-surface p-4 sm:p-5">
        <h2 className="text-xl font-semibold">Eligibility evidence</h2>
        {!assessment ? <p className="mt-4 rounded-md bg-app-surface-muted p-3 text-sm">Load or save a release to see supported properties and unresolved owner evidence.</p> : <>
          <p className="mt-2 break-all text-xs text-app-muted-foreground">{assessment.edition.release_id} · revision {assessment.edition.revision} · {assessment.edition.tracks.length} ordered tracks</p>
          <ul className="mt-4 space-y-2">{assessment.eligibility.checks.map(check => <li key={check.property} className="rounded-lg border border-app-border p-3">
            <div className="flex flex-wrap items-center justify-between gap-2"><span className="font-medium">{check.property.replace(/_/g, " ")}</span><span className={check.status === "supported" ? "text-xs text-green-700" : check.status === "blocked" ? "text-xs text-red-700" : "text-xs text-amber-700"}>{check.status}</span></div>
            {!!check.evidence?.length && <p className="mt-1 break-all font-mono text-xs text-app-muted-foreground">{check.evidence.join(" · ")}</p>}
            {!!check.missing?.length && <ul className="mt-1 list-disc pl-5 text-xs text-app-muted-foreground">{check.missing.map(item => <li key={item}>{item}</li>)}</ul>}
          </li>)}</ul>
          <p className="mt-4 rounded-lg bg-app-surface-muted p-3 text-sm">{assessment.eligibility.showcase_ready ? "Ready for operator EP judgment; this still does not mean the operator has approved it." : assessment.eligibility.ready_for_operator_judgment ? "Evidence packet is ready for the operator's one-batch judgment." : "Not ready for operator judgment. Draft fields and owner proof must be complete first."}</p>
          <section className="mt-5 rounded-lg border border-app-border p-4">
            <h3 className="text-lg font-semibold">Exact-hash rights decisions</h3>
            <p className="mt-1 text-sm text-app-muted-foreground">Draft fields are claims, not proof. A durable rights decision is accepted only for a track on this exact revision and when its evidence reference is the receipt path pinned to the liked catalog recording.</p>
            <ul className="mt-3 space-y-2">{assessment.edition.track_rights.map(item => <li key={item.track_sha256} className="rounded-md border border-app-border p-3 text-xs">
              <code className="block break-all">{item.track_sha256}</code><p className="mt-1">{item.decision_status} {item.decision_id ? `· ${item.decision_id}` : "· no operator decision"}</p>
              <p>License: {item.source_license || "missing"} · use: {item.usage_scope}</p><p className="break-all">Receipt: {item.evidence_ref || "missing"}</p>
            </li>)}</ul>
            <label className="mt-4 flex items-start gap-2 text-sm"><input type="checkbox" checked={reviewedSourceRights} onChange={event => setReviewedSourceRights(event.target.checked)} className="mt-1" /><span>I inspected each pinned source receipt and confirm the listed license and usage scope for this exact release revision.</span></label>
            <button disabled={busy || !reviewedSourceRights || !assessment.edition.track_rights.length} onClick={() => void confirmRights()} className="mt-3 rounded-md border px-4 py-2 text-sm font-semibold">Record exact-hash rights decisions</button>
            {!!assessment.edition.rights_decisions?.length && <ul className="mt-3 space-y-1">{assessment.edition.rights_decisions.map(item => <li key={item.decision_id} className="text-xs">{item.decision} · {item.track_sha256} · {item.source_license} · {item.usage_scope} · {item.reviewer_source}</li>)}</ul>}
          </section>
          <section className="mt-5 rounded-lg border border-app-border p-4">
            <h3 className="text-lg font-semibold">One-batch listening room</h3>
            <p className="mt-1 text-sm text-app-muted-foreground">Audio is loaded only when you press play. These streams resolve from the existing catalog by the exact hashes pinned to revision {assessment.edition.revision}; no audio is copied into the release catalog.</p>
            <ol className="mt-3 space-y-3">{assessment.edition.tracks.map(track => <li key={`${track.order}-${track.track_sha256}`} className="rounded-md border border-app-border p-3">
              <p className="text-sm font-medium">{String(track.order).padStart(2, "0")}. {track.title_override || `Track ${track.order}`}</p>
              <code className="mt-1 block break-all text-xs text-app-muted-foreground">SHA-256 {track.track_sha256}</code>
              <audio controls preload="none" className="mt-2 w-full" src={sourceAudioUrl(track.track_sha256)} aria-label={`Listen to release track ${track.order}, exact hash ${track.track_sha256}`} />
            </li>)}</ol>
            <label className="mt-4 flex items-start gap-2 text-sm"><input type="checkbox" checked={heardAllTracks} onChange={event => setHeardAllTracks(event.target.checked)} disabled={!assessment.eligibility.ready_for_operator_judgment} className="mt-1" /><span>I listened to every track in this exact ordered revision and am recording one decision for the assembled EP.</span></label>
            <label className="mt-4 block text-sm">Batch decision<select value={reviewDecision} onChange={event => setReviewDecision(event.target.value as "approved" | "changes_requested")} disabled={!assessment.eligibility.ready_for_operator_judgment} className={controlClass}><option value="approved">Approve this EP version</option><option value="changes_requested">Request changes</option></select></label>
            <label className="mt-3 block text-sm">Operator note<textarea value={reviewNote} onChange={event => setReviewNote(event.target.value)} maxLength={4000} rows={3} disabled={!assessment.eligibility.ready_for_operator_judgment} className={controlClass} /></label>
            <button disabled={busy || !assessment.eligibility.ready_for_operator_judgment || !heardAllTracks} onClick={() => void review()} className="mt-3 rounded-md bg-app-primary px-4 py-2 font-semibold text-app-primary-foreground">Record one-batch operator decision</button>
            {!!assessment.reviews.length && <ul className="mt-4 space-y-2">{assessment.reviews.map(item => <li key={item.review_id} className="rounded-md bg-app-surface-muted p-3 text-sm"><strong>{item.decision}</strong> · revision {item.revision} · {new Date(item.created_at).toLocaleString()}<p className="mt-1">{item.note || "No note recorded."}</p></li>)}</ul>}
          </section>
        </>}
      </section>
    </div>
  </main>;
}

function parseArray(raw: string, label: string): Array<Record<string, unknown>> {
  let parsed: unknown;
  try { parsed = JSON.parse(raw); } catch { throw new Error(label + " must be valid JSON."); }
  if (!Array.isArray(parsed) || !parsed.every((value): value is Record<string, unknown> => Boolean(value) && typeof value === "object" && !Array.isArray(value))) throw new Error(label + " must be a JSON array of objects.");
  return parsed;
}

function parseTrackRights(raw: string): ReleaseTrackRights[] {
  return parseArray(raw, "track rights").map(value => {
    const trackHash = typeof value.track_sha256 === "string" ? value.track_sha256.trim() : "";
    const sourceLicense = typeof value.source_license === "string" ? value.source_license.trim() : "";
    const usageScope = value.usage_scope;
    const evidenceRef = typeof value.evidence_ref === "string" ? value.evidence_ref.trim() : "";
    if (!/^[0-9a-f]{64}$/.test(trackHash)) throw new Error("Each rights entry needs an exact lowercase SHA-256 track hash.");
    if (!(["unknown", "personal_only", "commercial", "blocked"] as const).includes(usageScope as ReleaseTrackRights["usage_scope"])) throw new Error("Each rights entry needs a recognized use scope.");
    return { track_sha256: trackHash, source_license: sourceLicense, usage_scope: usageScope as ReleaseTrackRights["usage_scope"], decision_status: "review_required", evidence_ref: evidenceRef, note: typeof value.note === "string" ? value.note.trim() : "" };
  });
}
