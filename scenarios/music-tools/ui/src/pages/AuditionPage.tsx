import { useEffect, useRef, useState } from "react";
import { readAudio } from "./audition-audio";
import { fromJson } from "@bufbuild/protobuf";
import { ListTakesResponseSchema } from "@vrooli/proto-types/music-tools/v1/composition/composition_pb";
import { musicRestUrl } from "../api/client";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
type Take = {
    id: string;
    job_id?: string;
    provenance?: {
        seed?: string;
        model_id?: string;
        license_lane?: string;
        applied_rung?: string;
        caption_as_sent?: string;
        caption_as_authored?: string;
    };
};
type Finding = {
    choice: "keep" | "reject" | "unreviewed";
    listened: boolean;
    notes: string;
    reviewed_at?: string;
};
const emptyFinding = (): Finding => ({ choice: "unreviewed", listened: false, notes: "" });
export function AuditionPage() {
    const [takes, setTakes] = useState<Take[]>([]);
    const [findings, setFindings] = useState<Record<string, Finding>>({});
    const [reviewer, setReviewer] = useState("");
    const [error, setError] = useState<string | null>(null);
    const [cut, setCut] = useState("");
    const [jobFilter, setJobFilter] = useState("");
    const [query, setQuery] = useState("");
    const [reviewSources, setReviewSources] = useState<Record<string, {
        sha256: string;
        take: Take;
    }>>({});
    const [exporting, setExporting] = useState(false);
    const [audio, setAudio] = useState<Record<string, {
        url: string;
        sha256: string;
    }>>({});
    const [loadingAudio, setLoadingAudio] = useState<string | null>(null);
    const audioRequests = useRef(new Set<AbortController>());
    const retainedAudio = useRef(new Map<string, {
        url: string;
        sha256: string;
        bytes: number;
    }>());
    useEffect(() => () => { for (const request of audioRequests.current)
        request.abort(); for (const source of retainedAudio.current.values())
        URL.revokeObjectURL(source.url); retainedAudio.current.clear(); }, []);
    const players = useRef(new Map<string, HTMLAudioElement>());
    useEffect(() => {
        const controller = new AbortController();
        void fetch(musicRestUrl(`/takes?limit=20${query}`), { signal: controller.signal })
            .then(async (response) => { if (!response.ok)
            throw new Error(`Take list failed (${response.status})`); return response.json(); })
            .then(body => {
            const parsed = fromJson(ListTakesResponseSchema, Array.isArray(body) ? { takes: body } : body);
            setTakes(parsed.takes.map(take => ({ id: take.id, job_id: take.jobId, provenance: take.provenance ? { model_id: take.provenance.modelId, license_lane: take.provenance.licenseLane, caption_as_authored: take.provenance.captionAsAuthored, seed: take.provenance.seed.toString(), applied_rung: take.provenance.appliedRung, caption_as_sent: take.provenance.captionAsSent } : undefined })));
        })
            .catch((failure: unknown) => { if (!controller.signal.aborted)
            setError(failure instanceof Error ? failure.message : "Take list failed"); });
        return () => controller.abort();
    }, [query]);
    function update(id: string, change: Partial<Finding>) {
        if (!findings[id] && Object.keys(findings).length >= 100) {
            setError("Review session limit reached. Export before starting an empty session.");
            return;
        }
        setFindings(previous => ({ ...previous, [id]: { ...(previous[id] ?? emptyFinding()), ...change } }));
    }
    async function loadAudio(take: Take) {
        if (retainedAudio.current.has(take.id) || audioRequests.current.size)
            return;
        if (!reviewSources[take.id] && Object.keys(reviewSources).length >= 100) {
            setError("This review session has 100 candidates. Export your review and start an empty session.");
            return;
        }
        const request = new AbortController();
        audioRequests.current.add(request);
        setLoadingAudio(take.id);
        setError(null);
        try {
            const response = await fetch(musicRestUrl(`/takes/${encodeURIComponent(take.id)}/audio`), { signal: request.signal });
            if (!response.ok)
                throw new Error(`Audio load failed for ${take.id} (${response.status})`);
            const total = [...retainedAudio.current.values()].reduce((sum, source) => sum + source.bytes, 0);
            const bytes = await readAudio(response, Math.min(64 * 1024 * 1024, 256 * 1024 * 1024 - total));
            const hash = await crypto.subtle.digest("SHA-256", bytes);
            const sha256 = Array.from(new Uint8Array(hash), value => value.toString(16).padStart(2, "0")).join("");
            if (request.signal.aborted)
                return;
            const url = URL.createObjectURL(new Blob([bytes], { type: response.headers?.get("Content-Type") ?? "audio/wav" }));
            retainedAudio.current.set(take.id, { url, sha256, bytes: bytes.byteLength });
            setAudio(previous => ({ ...previous, [take.id]: { url, sha256 } }));
            setReviewSources(previous => ({ ...previous, [take.id]: { sha256, take } }));
            if (reviewSources[take.id] && reviewSources[take.id]!.sha256 !== sha256)
                update(take.id, { listened: false, notes: `${findings[take.id]?.notes ?? ""}\nSource bytes changed; listen again before exporting.` });
        }
        catch (failure) {
            if (!request.signal.aborted)
                setError(failure instanceof Error ? failure.message : "Audio load failed");
        }
        finally {
            audioRequests.current.delete(request);
            if (!request.signal.aborted)
                setLoadingAudio(null);
        }
    }
    async function exportReview() {
        setExporting(true);
        setError(null);
        try {
            const reviewed = Object.values(reviewSources).filter(source => { const finding = findings[source.take.id]; return finding?.listened && (finding.choice !== "unreviewed" || finding.notes.trim()); });
            if (!reviewer.trim() || reviewed.length === 0)
                throw new Error("Enter your name and confirm at least one listening review.");
            const rows = reviewed.map(({ take, sha256 }) => ({ take_id: take.id, job_id: take.job_id, sha256, provenance: take.provenance, ...findings[take.id] }));
            const blob = new Blob([JSON.stringify({ version: 1, cut: cut.trim() || "unspecified cut", reviewer: reviewer.trim(), method: "operator browser audition", reviewed_at: new Date().toISOString(), takes: rows, limits: ["Operator attestation; no automated taste score", "Take review does not approve a later final-video mix"] }, null, 2)], { type: "application/json" });
            const url = URL.createObjectURL(blob);
            const link = document.createElement("a");
            link.href = url;
            link.download = `music-audition-${Date.now()}.json`;
            link.click();
            URL.revokeObjectURL(url);
        }
        catch (failure) {
            setError(failure instanceof Error ? failure.message : "Review export failed");
        }
        finally {
            setExporting(false);
        }
    }
    function unloadAudio(id: string) {
        players.current.get(id)?.pause();
        const source = retainedAudio.current.get(id);
        if (source)
            URL.revokeObjectURL(source.url);
        retainedAudio.current.delete(id);
        setAudio(previous => { const next = { ...previous }; delete next[id]; return next; });
    }
    function saveDraft() {
        try {
            const raw = JSON.stringify({ version: 1, reviewer, cut, findings, reviewSources });
            if (raw.length > 1024 * 1024)
                throw new Error("Draft exceeds 1 MiB");
            localStorage.setItem("music-audition-draft-v1", raw);
            setError(null);
        }
        catch {
            setError("Draft could not be saved. Export your listening review before leaving.");
        }
    }
    function restoreDraft() {
        try {
            const raw = localStorage.getItem("music-audition-draft-v1");
            if (!raw || raw.length > 1024 * 1024)
                throw new Error("No supported draft is available.");
            const draft = JSON.parse(raw);
            if (draft.version !== 1 || typeof draft.reviewer !== "string" || typeof draft.cut !== "string" || !draft.findings || !draft.reviewSources)
                throw new Error("Draft is invalid.");
            if (Object.keys(draft.reviewSources).length > 100 || Object.keys(draft.findings).length > 100)
                throw new Error("Draft exceeds candidate limit.");
            for (const [id, source] of Object.entries(draft.reviewSources) as [
                string,
                {
                    sha256?: unknown;
                    take?: Take;
                }
            ][]) {
                const finding = draft.findings[id] ?? emptyFinding();
                draft.findings[id] = finding;
                if (!/^[a-f0-9]{64}$/.test(String(source.sha256)) || source.take?.id !== id || !finding || !["keep", "reject", "unreviewed"].includes(finding.choice) || typeof finding.listened !== "boolean" || typeof finding.notes !== "string")
                    throw new Error("Draft contains invalid source findings.");
            }
            for (const request of audioRequests.current) request.abort();
            audioRequests.current.clear();
            setLoadingAudio(null);
            for (const id of retainedAudio.current.keys()) {
                if (!draft.reviewSources[id]) unloadAudio(id);
            }
            // A restored attestation cannot describe different bytes from an
            // already playable candidate. Unload conflicts and require listening.
            for (const [id, source] of Object.entries(draft.reviewSources) as [string, { sha256: string }][]) {
                const cached = retainedAudio.current.get(id);
                if (cached && cached.sha256 !== source.sha256) {
                    unloadAudio(id);
                    draft.findings[id] = { ...draft.findings[id], listened: false, reviewed_at: undefined };
                }
            }
            setReviewer(draft.reviewer);
            setCut(draft.cut);
            setFindings(draft.findings);
            setReviewSources(draft.reviewSources);
            setError(null);
        }
        catch (failure) {
            setError(failure instanceof Error ? failure.message : "Draft could not be restored.");
        }
    }
    return <section className="flex flex-col gap-4" aria-labelledby="audition-heading">
    <h2 id="audition-heading" className="text-2xl font-semibold">Audition</h2>
    <p className="text-app-muted-foreground">Compare peer takes. Keep/reject records a choice for this cut and retains every source.</p>
    <label>Cut or project<input aria-label="Cut or project" value={cut} onChange={event => setCut(event.target.value)}/></label>
    <label>Job filter<input aria-label="Job filter" value={jobFilter} onChange={event => setJobFilter(event.target.value)}/></label>
    <button type="button" onClick={() => setQuery(jobFilter.trim() ? `&job_id=${encodeURIComponent(jobFilter.trim())}` : "")}>Load candidate batch</button>
    <p>Showing at most 20 candidates. Filter by job to compare another batch. Unload audio to free playback memory; findings retain their source hash.</p>
    <div className="flex gap-3"><button type="button" onClick={saveDraft}>Save review draft</button><button type="button" onClick={restoreDraft}>Restore review draft</button><button type="button" onClick={() => { for (const id of retainedAudio.current.keys())
        unloadAudio(id); }}>Unload all audio</button><button type="button" onClick={() => { for (const request of audioRequests.current)
        request.abort(); audioRequests.current.clear(); setLoadingAudio(null); for (const id of retainedAudio.current.keys())
        unloadAudio(id); setFindings({}); setReviewSources({}); }}>Start empty review session</button></div>
    {error && <p role="alert">{error}</p>}
    <div className="grid gap-3">{takes.map(take => {
            const finding = findings[take.id] ?? emptyFinding();
            return <Card key={take.id} data-testid="take-row"><CardHeader><CardTitle className="font-mono text-base">{take.id}</CardTitle></CardHeader><CardContent className="flex flex-col gap-3">
        <button type="button" className="touch-target" disabled={Boolean(loadingAudio) || Boolean(audio[take.id])} onClick={() => void loadAudio(take)}>Load audio {take.id}</button>
        {audio[take.id] && <button type="button" onClick={() => unloadAudio(take.id)}>Unload audio {take.id}</button>}
        <audio ref={element => { if (element)
                players.current.set(take.id, element);
            else
                players.current.delete(take.id); }} data-testid="take-play" controls preload="metadata" aria-label={`Play ${take.id}`} src={audio[take.id]?.url} onPlay={() => { for (const [id, player] of players.current)
                if (id !== take.id)
                    player.pause(); }} onError={() => { if (!retainedAudio.current.has(take.id))
                return; update(take.id, { listened: false }); setError(`Playback failed for ${take.id}. Check the audio source before reviewing.`); }}/>
        <div data-testid="take-provenance" className="text-sm text-app-muted-foreground">Seed {take.provenance?.seed ?? "—"} · {take.provenance?.caption_as_sent ?? "Prompt not recorded"}</div>
        <span data-testid="take-rung-badge" role="status">rung {take.provenance?.applied_rung ?? "—"}</span>
        <div className="flex gap-3"><button className="touch-target" data-testid="take-keep" type="button" aria-pressed={finding.choice === "keep"} onClick={() => update(take.id, { choice: "keep" })}>Keep for this cut</button><button className="touch-target" data-testid="take-discard" type="button" aria-pressed={finding.choice === "reject"} onClick={() => update(take.id, { choice: "reject" })}>Reject for this cut</button></div>
        <label>Listening notes for {take.id}<textarea aria-label={`Listening notes ${take.id}`} className="w-full rounded border p-2" value={finding.notes} onChange={event => update(take.id, { notes: event.target.value })} placeholder="Groove, energy, seams, ending and fit"/></label>
        <label><input type="checkbox" disabled={!audio[take.id]} checked={finding.listened} onChange={event => update(take.id, { listened: event.target.checked, reviewed_at: event.target.checked ? new Date().toISOString() : undefined })}/> I listened to {take.id} and recorded my finding.</label>
      </CardContent></Card>;
        })}</div>
    <label>Reviewer<input aria-label="Reviewer" className="rounded border p-2" value={reviewer} onChange={event => setReviewer(event.target.value)}/></label>
    <button type="button" className="touch-target" disabled={exporting} onClick={() => void exportReview()}>Export listening review</button>
    <p data-testid="batch-partial-notice" role="status">Completed takes remain usable if a batch ends early.</p>
  </section>;
}
