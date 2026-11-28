import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { Badge } from "../../../components/ui/badge";
import { Button } from "../../../components/ui/button";
import { ApiErrorState } from "../../../components/composites/ApiErrorState";
import { LoadingRows } from "../../../components/composites/LoadingRows";
import { pushToast } from "../../../components/ui/toast";
import { useTranslation } from "../../../i18n";
import { strings } from "../../../consts/strings";
import { selectors } from "../../../consts/selectors";
import { ClipSource, deleteClip, getClipAudio, listClips, reviewReference, reviewReferenceGroup, type ClipMeta } from "../../../services/corpus";
import { pcm16ToWavBuffer } from "../../../audio-integration/hooks/voice/pcm";

type Translate = (key: string, options?: Record<string, unknown>) => string;

function sourceLabel(t: Translate, source: ClipSource): string {
  switch (source) {
    case ClipSource.FREE_FORM:
      return t(strings.dictationStudio.sourceFreeForm);
    case ClipSource.SCRIPTED:
      return t(strings.dictationStudio.sourceScripted);
    default:
      return t(strings.dictationStudio.sourceUnknown);
  }
}

function captionTrackSrc(text: string, durationMs: number): string {
  const end = new Date(durationMs).toISOString().slice(11, 23);
  const vtt = `WEBVTT\n\n00:00:00.000 --> ${end}\n${text.replace(/-->/g, "—>")}`;
  return `data:text/vtt;charset=utf-8,${encodeURIComponent(vtt)}`;
}

function formatSeconds(durationMs: number): string {
  return `${(durationMs / 1000).toFixed(1)}s`;
}

// CorpusListView lists stored corpus clips (newest first, from ListClips)
// and supports per-clip deletion.
export function CorpusListView() {
  const { t } = useTranslation();
  const tr = t as unknown as Translate;
  const qc = useQueryClient();
  const clips = useQuery({ queryKey: ["corpus", "clips"], queryFn: () => listClips() });
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [referenceText, setReferenceText] = useState("");
  const [reviewer, setReviewer] = useState("");
  const [protocol, setProtocol] = useState("listen-and-correct-v1");
  const [audioUrl, setAudioUrl] = useState<string | null>(null);
  const [audioCompleted, setAudioCompleted] = useState(false);
  const [groupReview, setGroupReview] = useState(false);
  const [playedVariantIds, setPlayedVariantIds] = useState<string[]>([]);
  const [sourceTranscriptVerified, setSourceTranscriptVerified] = useState(false);
  const selected = clips.data?.find((clip) => clip.id === selectedId) ?? null;
  const sourceVariants = useMemo(() => selected?.sourceId ? (clips.data ?? []).filter((clip) => clip.sourceId === selected.sourceId) : [], [selected?.sourceId, clips.data]);
  useEffect(() => {
    let active = true;
    setSourceTranscriptVerified(false);
    if (!selected?.manifestTranscriptSha256 || !selected.lineageManifestSha256 || sourceVariants.length < 2 || !sourceVariants.every((clip) => clip.lineageManifestSha256 === selected.lineageManifestSha256 && clip.manifestTranscriptSha256 === selected.manifestTranscriptSha256)) return () => { active = false; };
    void crypto.subtle.digest("SHA-256", new TextEncoder().encode(selected.referenceText)).then((digest) => {
      const actual = Array.from(new Uint8Array(digest), (byte) => byte.toString(16).padStart(2, "0")).join("");
      if (active) setSourceTranscriptVerified(actual === selected.manifestTranscriptSha256);
    });
    return () => { active = false; };
  }, [selected, sourceVariants]);
  const groupCompatible = !!selected && sourceTranscriptVerified && sourceVariants.length > 1 && sourceVariants.every((clip) => clip.corpusRevision === selected.corpusRevision && clip.speakerPseudonym === selected.speakerPseudonym && clip.accentGroup === selected.accentGroup && clip.referenceText === selected.referenceText);
  const allVariantsPlayed = groupCompatible && sourceVariants.every((clip) => playedVariantIds.includes(clip.id));
  const audio = useQuery({
    queryKey: ["corpus", "clip-audio", selectedId],
    queryFn: () => selectedId ? getClipAudio(selectedId) : Promise.reject(new Error("No clip selected")),
    enabled: !!selectedId && !groupReview,
    staleTime: 60_000,
  });
  useEffect(() => {
    if (groupReview || !audio.data || !selected) return;
    const bytes = audio.data.audio;
    const blob = selected.format === "pcm_s16le" || selected.format === "pcm"
      ? new Blob([pcm16ToWavBuffer(new Int16Array(bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength)), selected.sampleRateHz)], { type: "audio/wav" })
      : new Blob([bytes.slice().buffer], { type: selected.format === "wav" ? "audio/wav" : `audio/${selected.format || "octet-stream"}` });
    const url = URL.createObjectURL(blob);
    setAudioUrl(url);
    return () => URL.revokeObjectURL(url);
  }, [audio.data, selected, groupReview]);

  const del = useMutation({
    mutationFn: (id: string) => deleteClip(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["corpus", "clips"] });
      pushToast({ title: tr(strings.dictationStudio.clipDeleted) });
    },
  });
  const review = useMutation({
    mutationFn: async () => {
      if (!selected) throw new Error("Select a clip first");
      const digest = await crypto.subtle.digest("SHA-256", new TextEncoder().encode(selected.referenceText));
      const expectedReferenceSha256 = Array.from(new Uint8Array(digest), (byte) => byte.toString(16).padStart(2, "0")).join("");
      if (groupReview) {
        return reviewReferenceGroup({
          members: sourceVariants.map((clip) => ({ id: clip.id, expectedReferenceSha256 })),
          referenceText, reviewer, protocol: "listen-and-correct-each-source-variant-v1",
        });
      }
      return reviewReference({ id: selected.id, expectedReferenceSha256, referenceText, reviewer, protocol });
    },
    onSuccess: async (result) => {
      await qc.invalidateQueries({ queryKey: ["corpus", "clips"] });
      setSelectedId(null);
      pushToast({ title: "affectedCount" in result ? `Direct human review saved for ${result.affectedCount} clips.` : "Reference review saved." });
    },
  });

  if (clips.isPending) {
    return <LoadingRows rows={3} label={tr(strings.dictationStudio.corpusTitle)} />;
  }
  if (clips.isError) {
    return (
      <ApiErrorState
        error={clips.error}
        title={tr(strings.dictationStudio.corpusError)}
        onRetry={() => void clips.refetch()}
      />
    );
  }

  const list: ClipMeta[] = clips.data;
  if (list.length === 0) {
    return <p className="text-sm text-app-muted-foreground">{tr(strings.dictationStudio.corpusEmpty)}</p>;
  }

  return (
    <>
    <ul className="flex flex-col gap-2" data-testid={selectors.dictationStudio.corpusList}>
      {list.map((clip) => (
        <li
          key={clip.id}
          data-testid={selectors.dictationStudio.clipRow({ id: clip.id })}
          className="flex items-start justify-between gap-3 rounded-control border border-app-border px-3 py-2"
        >
          <div className="flex min-w-0 flex-col gap-1">
            <p className="truncate text-sm text-app-foreground">{clip.referenceText}</p>
            <div className="flex flex-wrap items-center gap-2">
              <Badge variant="neutral">{sourceLabel(tr, clip.source)}</Badge>
              <span className="text-xs text-app-muted-foreground">{formatSeconds(clip.durationMs)}</span>
              {clip.tags.map((tag) => (
                <Badge key={tag} variant="info">
                  {tag}
                </Badge>
              ))}
              {clip.referenceReview ? <Badge variant="success">{t(strings.dictationStudio.reviewedStatus)}</Badge> : <Badge variant="neutral">{t(strings.dictationStudio.unreviewedStatus)}</Badge>}
            </div>
          </div>
          <div className="flex gap-1">
          <Button type="button" variant="ghost" size="sm" onClick={() => { setSelectedId(clip.id); setReferenceText(clip.referenceText); setAudioCompleted(false); setGroupReview(false); setPlayedVariantIds([]); review.reset(); }}>
            {t(strings.dictationStudio.reviewButton)}
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            data-testid={selectors.dictationStudio.clipDelete({ id: clip.id })}
            disabled={del.isPending}
            onClick={() => del.mutate(clip.id)}
          >
            {t(strings.dictationStudio.deleteClip)}
          </Button>
          </div>
        </li>
      ))}
    </ul>
    {selected && <section aria-label={t(strings.dictationStudio.reviewTranscript)} className="mt-4 rounded-control border border-app-border p-4">
      <h3 className="text-base font-semibold">{t(strings.dictationStudio.reviewReferenceTitle)}</h3>
      <p className="text-sm text-app-muted-foreground">{t(strings.dictationStudio.reviewInstructions)}</p>
      {groupCompatible && <label className="mt-3 flex items-center gap-2 text-sm"><input type="checkbox" checked={groupReview} onChange={(event) => { setGroupReview(event.target.checked); setPlayedVariantIds([]); if (event.target.checked) setAudioUrl(null); }} />{t(strings.dictationStudio.reviewAllVariants, { count: sourceVariants.length })}</label>}
      {groupReview && groupCompatible ? <div className="my-3 space-y-3">
        <p className="text-sm font-medium">{t(strings.dictationStudio.reviewAllInstructions, { count: sourceVariants.length })}</p>
        {sourceVariants.map((clip) => <VariantReviewAudio key={clip.id} clip={clip} onCompleted={() => setPlayedVariantIds((ids) => ids.includes(clip.id) ? ids : [...ids, clip.id])} />)}
      </div> : <>
        {audio.isPending ? <p>{t(strings.dictationStudio.loadingAudio)}</p> : audio.isError ? <p role="alert">{t(strings.dictationStudio.clipAudioError)}</p> : audioUrl && <audio controls preload="metadata" src={audioUrl} onEnded={() => setAudioCompleted(true)} className="my-3 w-full" data-testid={selectors.dictationStudio.reviewAudio}><track kind="captions" srcLang="en" label={t(strings.dictationStudio.transcriptCaptionTrackLabel)} src={captionTrackSrc(selected.referenceText, selected.durationMs)} /></audio>}
        {!audioCompleted && <p className="text-xs text-app-muted-foreground">{t(strings.dictationStudio.playFullClip)}</p>}
      </>}
      <label className="block text-sm">{t(strings.dictationStudio.referenceTranscriptLabel)}<textarea className="mt-1 w-full rounded border bg-app-background p-2" rows={4} value={referenceText} onChange={(event) => setReferenceText(event.target.value)} /></label>
      <label className="mt-3 block text-sm">{t(strings.dictationStudio.reviewerLabel)}<input className="mt-1 w-full rounded border bg-app-background p-2" required value={reviewer} onChange={(event) => setReviewer(event.target.value)} /></label>
      {groupReview ? <p className="mt-3 text-xs text-app-muted-foreground">{t(strings.dictationStudio.groupReviewProtocol)}</p> : <label className="mt-3 block text-sm">{t(strings.dictationStudio.reviewProtocolLabel)}<input className="mt-1 w-full rounded border bg-app-background p-2" required value={protocol} onChange={(event) => setProtocol(event.target.value)} /></label>}
      {review.isError && <p role="alert" className="mt-2 text-sm">{review.error.message.includes("aborted") ? t(strings.dictationStudio.staleReviewError) : review.error.message}</p>}
      <div className="mt-3 flex gap-2"><Button type="button" disabled={review.isPending || !reviewer.trim() || (groupReview ? !allVariantsPlayed : !protocol.trim() || !audioCompleted || audio.isPending || audio.isError)} onClick={() => review.mutate()}>{groupReview ? t(strings.dictationStudio.submitGroupReview, { count: sourceVariants.length }) : t(strings.dictationStudio.submitReview)}</Button><Button type="button" variant="ghost" onClick={() => setSelectedId(null)}>{t(strings.dictationStudio.closeReview)}</Button></div>
    </section>}
    </>
  );
}

function VariantReviewAudio({ clip, onCompleted }: { clip: ClipMeta; onCompleted: () => void }) {
  const { t } = useTranslation();
  const audio = useQuery({ queryKey: ["corpus", "clip-audio", clip.id], queryFn: () => getClipAudio(clip.id), staleTime: 60_000 });
  const [url, setUrl] = useState<string | null>(null);
  const [completed, setCompleted] = useState(false);
  useEffect(() => {
    if (!audio.data) return;
    const bytes = audio.data.audio;
    const blob = clip.format === "pcm_s16le" || clip.format === "pcm"
      ? new Blob([pcm16ToWavBuffer(new Int16Array(bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength)), clip.sampleRateHz)], { type: "audio/wav" })
      : new Blob([bytes.slice().buffer], { type: clip.format === "wav" ? "audio/wav" : `audio/${clip.format || "octet-stream"}` });
    const next = URL.createObjectURL(blob); setUrl(next); return () => URL.revokeObjectURL(next);
  }, [audio.data, clip]);
  return <div className="rounded border p-2"><p className="text-sm">{clip.noiseCondition} · {clip.snrDb ? `${clip.snrDb} dB` : t(strings.dictationStudio.cleanNoiseCondition)} · {clip.id}</p>{audio.isPending ? <p>{t(strings.dictationStudio.loadingAudio)}</p> : audio.isError ? <p role="alert">{t(strings.dictationStudio.variantAudioError)}</p> : url && <audio controls preload="metadata" src={url} onEnded={() => { setCompleted(true); onCompleted(); }} className="w-full" data-testid={selectors.dictationStudio.variantReviewAudio({ id: clip.id })}><track kind="captions" srcLang="en" label={t(strings.dictationStudio.transcriptCaptionTrackLabel)} src={captionTrackSrc(clip.referenceText, clip.durationMs)} /></audio>}{completed && <span className="text-xs">{t(strings.dictationStudio.fullClipPlayed)}</span>}</div>;
}
