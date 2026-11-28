import { useEffect, useMemo, useState } from "react";
import { create } from "@bufbuild/protobuf";
import { SelectPairResponseSchema, TasteCandidateSchema } from "@vrooli/proto-types/music-library/v1/taste/taste_pb";
import { catalogClient, tasteClient } from "../api/taste";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

export function TastePage() {
  const { t } = useTranslation();
  const [context, setContext] = useState("default");
  const [tracks, setTracks] = useState<Array<{ sha256: string; title: string; tags: string[] }>>([]);
  const [profile, setProfile] = useState<Awaited<ReturnType<typeof tasteClient.getProfile>>["profile"]>();
  const [pair, setPair] = useState<Awaited<ReturnType<typeof tasteClient.selectPair>>>();
  const [result, setResult] = useState<Awaited<ReturnType<typeof tasteClient.recommend>>>();
  const [seed, setSeed] = useState("");
  const [attribute, setAttribute] = useState("");
  const [mode, setMode] = useState("boost");
  const [controlValue, setControlValue] = useState("0.5");
  const [correctionId, setCorrectionId] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const randomizePairOrder = (selected: Awaited<ReturnType<typeof tasteClient.selectPair>>) => {
    if (!selected.left || !selected.right || Math.random() >= 0.5) return selected;
    return create(SelectPairResponseSchema, {
      left: selected.right,
      right: selected.left,
      expectedInformation: selected.expectedInformation,
      availableCandidateCount: selected.availableCandidateCount,
      insufficientEvidence: selected.insufficientEvidence,
    });
  };

  useEffect(() => {
    let active = true;
    void Promise.all([tasteClient.getProfile({ context }), catalogClient.listTracks({})]).then(([p, c]) => {
      if (!active) return;
      setProfile(p.profile);
      const catalog = c.tracks.map(track => ({ sha256: track.sha256, title: track.title, tags: track.tags }));
      setTracks(catalog);
      if (catalog.length && catalog[0]) setSeed(catalog[0].sha256);
    }).catch(e => setError(e instanceof Error ? e.message : t(strings.pages.taste.loadError)));
    return () => { active = false; };
  }, [t, context]);

  const tags = useMemo(() => Array.from(new Set(tracks.flatMap(track => track.tags.map(tag => "tag:" + tag.toLowerCase())))).sort(), [tracks]);
  const refresh = async () => {
    setBusy(true); setError("");
    try { setProfile((await tasteClient.getProfile({ context })).profile); }
    catch (e) { setError(e instanceof Error ? e.message : t(strings.pages.taste.loadError)); }
    finally { setBusy(false); }
  };
  const loadPair = async () => {
    setBusy(true); setError("");
    try { setPair(randomizePairOrder(await tasteClient.selectPair({ context }))); }
    catch (e) { setError(e instanceof Error ? e.message : t(strings.pages.taste.pairError)); }
    finally { setBusy(false); }
  };
  const answer = async (choice: string) => {
    if (!pair?.left || !pair.right) return;
    setBusy(true); setError("");
    try {
      const response = await tasteClient.recordJudgment({ context, leftTrackSha256: pair.left.trackSha256, rightTrackSha256: pair.right.trackSha256, choice, reversesJudgmentId: correctionId });
      setProfile(response.profile); setPair(undefined); setCorrectionId("");
    } catch (e) { setError(e instanceof Error ? e.message : t(strings.pages.taste.judgmentError)); }
    finally { setBusy(false); }
  };
  const saveControl = async () => {
    if (!attribute.trim()) return;
    setBusy(true); setError("");
    try {
      const response = await tasteClient.setControl({ context, control: { attribute: attribute.trim(), mode, value: Number(controlValue) } });
      setProfile(response.profile);
    } catch (e) { setError(e instanceof Error ? e.message : t(strings.pages.taste.controlError)); }
    finally { setBusy(false); }
  };
  const controlFor = async (name: string, controlMode: string, value: number) => {
    setBusy(true); setError("");
    try {
      const response = await tasteClient.setControl({ context, control: { attribute: name, mode: controlMode, value } });
      setProfile(response.profile);
    } catch (e) { setError(e instanceof Error ? e.message : t(strings.pages.taste.controlError)); }
    finally { setBusy(false); }
  };
  const recommend = async () => {
    if (!seed) return;
    setBusy(true); setError("");
    try { setResult(await tasteClient.recommend({ context, seedTrackSha256: seed, limit: 10 })); }
    catch (e) { setError(e instanceof Error ? e.message : t(strings.pages.taste.recommendError)); }
    finally { setBusy(false); }
  };

  return <main className="mx-auto w-full max-w-5xl px-4 py-6 sm:px-6 lg:px-8">
    <header className="mb-6">
      <p className="text-sm font-medium text-app-accent">{t(strings.pages.taste.eyebrow)}</p>
      <h1 className="mt-1 text-3xl font-semibold">{t(strings.pages.taste.title)}</h1>
      <p className="mt-2 max-w-3xl text-sm text-app-muted-foreground">{t(strings.pages.taste.description)}</p>
    </header>
    {error && <p role="alert" className="mb-4 rounded-md border border-red-300 bg-red-50 p-3 text-sm text-red-900">{error}</p>}
    <label className="mb-4 block max-w-sm text-sm">{t(strings.pages.taste.context)}<input value={context} maxLength={80} onChange={e => { setContext(e.target.value || "default"); setPair(undefined); setResult(undefined); setCorrectionId(""); }} className="mt-1 w-full rounded-md border bg-app-background p-2" /></label>
    <div className="grid gap-5 lg:grid-cols-2">
      <section className="rounded-xl border border-app-border bg-app-surface p-4 sm:p-5">
        <div className="flex items-start justify-between gap-3">
          <div><h2 className="text-xl font-semibold">{t(strings.pages.taste.profile)}</h2><p className="mt-1 text-sm text-app-muted-foreground">{t(strings.pages.taste.revision, { revision: profile ? String(profile.revision) : "0" })}</p></div>
          <button onClick={() => void refresh()} disabled={busy} className="rounded-md border px-3 py-2 text-sm">{t(strings.pages.taste.refresh)}</button>
        </div>
        {!profile?.weights.length && <p className="mt-4 rounded-md bg-app-surface-muted p-3 text-sm">{t(strings.pages.taste.sparse)}</p>}
        <ul className="mt-3 space-y-2">{profile?.weights.map(weight => <li key={weight.attribute} className="flex items-center justify-between gap-2 rounded-lg border border-app-border p-3"><span className="min-w-0 break-all text-sm">{weight.attribute}</span><span className="shrink-0 text-xs text-app-muted-foreground">{weight.mean.toFixed(2)} ± {weight.variance.toFixed(2)} · {String(weight.evidenceCount)}</span></li>)}</ul>
        {!!profile?.judgments.length && <div className="mt-4 border-t border-app-border pt-4"><h3 className="font-medium">{t(strings.pages.taste.judgmentHistory)}</h3><ul className="mt-2 space-y-2">{profile.judgments.slice(-8).reverse().map(judgment => { const superseded = profile.judgments.some(entry => entry.reversesJudgmentId === judgment.id); return <li key={judgment.id} className="flex flex-wrap items-center justify-between gap-2 rounded-lg border border-app-border p-3 text-xs"><span>{judgment.choice}: {judgment.leftTrackSha256.slice(0, 10)} / {judgment.rightTrackSha256.slice(0, 10)}{superseded ? " · " + t(strings.pages.taste.corrected) : ""}</span><button onClick={() => { const left = tracks.find(track => track.sha256 === judgment.leftTrackSha256); const right = tracks.find(track => track.sha256 === judgment.rightTrackSha256); if (!left || !right) return; const candidate = (track: typeof left) => create(TasteCandidateSchema, { trackSha256: track.sha256, title: track.title, artist: "", genre: "", tags: track.tags }); setPair(create(SelectPairResponseSchema, { left: candidate(left), right: candidate(right), availableCandidateCount: BigInt(tracks.length) })); setCorrectionId(judgment.id); }} disabled={busy || superseded || !tracks.some(track => track.sha256 === judgment.leftTrackSha256) || !tracks.some(track => track.sha256 === judgment.rightTrackSha256)} className="rounded border px-2 py-1">{t(strings.pages.taste.correct)}</button></li>; })}</ul></div>}
        <div className="mt-5 border-t border-app-border pt-4">
          <h3 className="font-medium">{t(strings.pages.taste.controls)}</h3>
          <div className="mt-3 grid gap-2 sm:grid-cols-2">
            <label className="text-sm">{t(strings.pages.taste.attribute)}<input list="taste-tags" value={attribute} onChange={e => setAttribute(e.target.value)} className="mt-1 w-full rounded-md border bg-app-background p-2" /><datalist id="taste-tags">{tags.map(tag => <option key={tag} value={tag} />)}</datalist></label>
            <label className="text-sm">{t(strings.pages.taste.mode)}<select value={mode} onChange={e => setMode(e.target.value)} className="mt-1 w-full rounded-md border bg-app-background p-2"><option value="boost">{t(strings.pages.taste.boost)}</option><option value="suppress">{t(strings.pages.taste.suppress)}</option><option value="pin">{t(strings.pages.taste.pin)}</option><option value="ban">{t(strings.pages.taste.ban)}</option></select></label>
            <label className="text-sm">{t(strings.pages.taste.value)}<input type="number" min="-1" max="1" step="0.1" value={controlValue} onChange={e => setControlValue(e.target.value)} className="mt-1 w-full rounded-md border bg-app-background p-2" /></label>
            <button onClick={() => void saveControl()} disabled={busy || !attribute.trim()} className="self-end rounded-md bg-app-primary px-4 py-2 font-semibold text-app-primary-foreground">{t(strings.pages.taste.saveControl)}</button>
          </div>
        </div>
      </section>
      <section className="rounded-xl border border-app-border bg-app-surface p-4 sm:p-5">
        <h2 className="text-xl font-semibold">{t(strings.pages.taste.compare)}</h2>
        <p className="mt-1 text-sm text-app-muted-foreground">{t(strings.pages.taste.compareHint)}</p>
        {correctionId && <p className="mt-3 rounded-md bg-app-surface-muted p-3 text-sm">{t(strings.pages.taste.correctionPending)}</p>}
        <div className="mt-3 flex flex-wrap gap-2"><button onClick={() => void loadPair()} disabled={busy} className="rounded-md border px-4 py-2 text-sm">{t(strings.pages.taste.nextPair)}</button><button onClick={() => { setBusy(true); setError(""); void tasteClient.selectPair({ context, randomBaseline: true, randomSeed: 1n }).then(randomizePairOrder).then(setPair).catch(e => setError(e instanceof Error ? e.message : t(strings.pages.taste.pairError))).finally(() => setBusy(false)); }} disabled={busy} className="rounded-md border px-4 py-2 text-sm">{t(strings.pages.taste.randomPair)}</button></div>
        {pair?.left && pair.right ? <div className="mt-4 space-y-3">
          <p className="text-xs text-app-muted-foreground">{t(strings.pages.taste.information, { value: pair.expectedInformation.toFixed(2) })}</p>
          <div className="grid gap-3 sm:grid-cols-2">{[pair.left, pair.right].map((candidate, index) => <article key={candidate.trackSha256} className="rounded-lg border border-app-border p-3"><h3 className="font-medium">{candidate.title || candidate.trackSha256.slice(0, 12)}</h3>{(candidate.artist || candidate.genre) && <p className="mt-1 text-xs text-app-muted-foreground">{[candidate.artist, candidate.genre].filter(Boolean).join(" · ")}</p>}<p className="mt-1 break-all text-xs text-app-muted-foreground">{candidate.trackSha256}</p><p className="mt-2 text-xs">{candidate.tags.join(" · ")}</p><button onClick={() => void answer(index === 0 ? "left" : "right")} disabled={busy} className="mt-3 w-full rounded-md bg-app-primary px-3 py-2 text-sm font-semibold text-app-primary-foreground">{index === 0 ? t(strings.pages.taste.preferLeft) : t(strings.pages.taste.preferRight)}</button></article>)}</div>
          <div className="flex flex-wrap gap-2"><button onClick={() => void answer("tie")} disabled={busy} className="rounded-md border px-3 py-2 text-sm">{t(strings.pages.taste.tie)}</button><button onClick={() => void answer("neither")} disabled={busy} className="rounded-md border px-3 py-2 text-sm">{t(strings.pages.taste.neither)}</button><button onClick={() => void answer("skip")} disabled={busy} className="rounded-md border px-3 py-2 text-sm">{t(strings.pages.taste.skip)}</button></div>
        </div> : <p className="mt-4 rounded-md bg-app-surface-muted p-3 text-sm">{t(strings.pages.taste.noPair)}</p>}
      </section>
    </div>
    <section className="mt-5 rounded-xl border border-app-border bg-app-surface p-4 sm:p-5">
      <h2 className="text-xl font-semibold">{t(strings.pages.taste.why)}</h2>
      <p className="mt-1 text-sm text-app-muted-foreground">{t(strings.pages.taste.whyHint)}</p>
      {!tracks.length ? <p className="mt-4 rounded-md bg-app-surface-muted p-3 text-sm">{t(strings.pages.taste.noCatalog)}</p> : <div className="mt-3 flex flex-col gap-2 sm:flex-row"><label className="flex-1 text-sm">{t(strings.pages.taste.seed)}<select value={seed} onChange={e => setSeed(e.target.value)} className="mt-1 w-full rounded-md border bg-app-background p-2">{tracks.map(track => <option key={track.sha256} value={track.sha256}>{track.title} · {track.sha256.slice(0, 10)}</option>)}</select></label><button onClick={() => void recommend()} disabled={busy || !seed} className="self-end rounded-md bg-app-primary px-4 py-2 font-semibold text-app-primary-foreground">{t(strings.pages.taste.findRecommendations)}</button></div>}
      {result && <><p className="mt-4 rounded-md bg-app-surface-muted p-3 text-sm">{result.evidenceState} · {result.recommendations.length}</p>{result.recommendations.length === 0 ? <p className="mt-3 text-sm">{t(strings.pages.taste.noRecommendations)}</p> : <ol className="mt-3 grid gap-3 md:grid-cols-2">{result.recommendations.map(item => <li key={item.candidate?.trackSha256} className="rounded-lg border border-app-border p-3"><h3 className="font-medium">{item.rank}. {item.candidate?.title || item.candidate?.trackSha256.slice(0, 12)}</h3>{(item.candidate?.artist || item.candidate?.genre) && <p className="mt-1 text-xs text-app-muted-foreground">{[item.candidate?.artist, item.candidate?.genre].filter(Boolean).join(" · ")}</p>}<p className="mt-1 text-xs text-app-muted-foreground">{t(strings.pages.taste.score, { score: item.predictedPreference.toFixed(3), uncertainty: item.uncertainty.toFixed(3) })}{item.exploration ? " · " + t(strings.pages.taste.exploration) : ""}</p><ul className="mt-2 space-y-2 text-xs">{item.contributions.map(part => <li key={part.attribute} className="flex flex-wrap items-center justify-between gap-2"><span className="min-w-0 break-all">{part.attribute}: {part.contribution >= 0 ? "+" : ""}{part.contribution.toFixed(3)} <span className="text-app-muted-foreground">{t(strings.pages.taste.featureEvidence, { value: part.featureValue.toFixed(3), confidence: part.featureConfidence.toFixed(2) })} ({part.evidenceSource})</span></span><span className="flex gap-1"><button onClick={() => void controlFor(part.attribute, "boost", 0.5)} disabled={busy} aria-label={t(strings.pages.taste.reinforceAttribute, { attribute: part.attribute })} className="rounded border px-2 py-1">{t(strings.pages.taste.more)}</button><button onClick={() => void controlFor(part.attribute, "suppress", 0.5)} disabled={busy} aria-label={t(strings.pages.taste.suppressAttribute, { attribute: part.attribute })} className="rounded border px-2 py-1">{t(strings.pages.taste.less)}</button><button onClick={() => void controlFor(part.attribute, "ban", 1)} disabled={busy} aria-label={t(strings.pages.taste.banAttribute, { attribute: part.attribute })} className="rounded border px-2 py-1">{t(strings.pages.taste.exclude)}</button></span></li>)}</ul></li>)}</ol>}<h3 className="mt-5 font-semibold">{t(strings.pages.taste.mq2Brief)}</h3><p className="mt-2 whitespace-pre-wrap rounded-lg border border-app-border p-3 text-sm">{result.mq2Brief}</p></>}
    </section>
  </main>;
}
