import { useEffect, useRef, useState } from "react";
import { ChevronDown, Headphones, Loader2, Maximize2, Minimize2, Pause, Play, Repeat, RotateCcw, RotateCw, Volume2, VolumeX } from "lucide-react";
import { useTranslation } from "react-i18next";
import { strings } from "../../../consts/strings";
import { selectors } from "../../../consts/selectors";
import { formatAudioTime } from "../audioAnalysis";
import { formatBytes } from "../format";
import { useMediaPlayer } from "../useMediaPlayer";
import type { PreviewRendererProps } from "../types";
import { ArtifactFooter, ArtifactFrame, PreviewButton, PreviewToolbar, previewSelectClass, previewTextButtonClass } from "./ArtifactChrome";
import { PreviewNotice } from "./shared";

export function VideoPreview(props: PreviewRendererProps) {
  return <VideoPlayer key={`${props.model.previewId}:${props.model.blobHref}`} {...props} />;
}

function VideoPlayer({ model }: PreviewRendererProps) {
  const { t } = useTranslation();
  const c = strings.artifactPreview;
  const a = strings.audioFilePlayer;
  const player = useMediaPlayer<HTMLVideoElement>(model.blobHref);
  const frameRef = useRef<HTMLDivElement>(null);
  const [nativeControls, setNativeControls] = useState(false);
  const [dimensions, setDimensions] = useState({ width: 0, height: 0 });
  const [fullscreen, setFullscreen] = useState(false);
  const [fullscreenFailed, setFullscreenFailed] = useState(false);
  useEffect(() => {
    const frame = frameRef.current;
    const changed = () => { setFullscreen(document.fullscreenElement === frame); };
    document.addEventListener("fullscreenchange", changed);
    return () => {
      document.removeEventListener("fullscreenchange", changed);
      if (document.fullscreenElement === frame) void document.exitFullscreen().catch(() => {});
    };
  }, []);
  const toggleFullscreen = async () => {
    const frame = frameRef.current;
    const media = player.mediaRef.current;
    if (!frame || !media) return;
    setFullscreenFailed(false);
    try {
      if (document.fullscreenElement === frame) await document.exitFullscreen();
      else if (document.fullscreenEnabled) await frame.requestFullscreen();
      else if (media.webkitEnterFullscreen) { media.controls = true; setNativeControls(true); media.webkitEnterFullscreen(); }
      else { setNativeControls(true); setFullscreenFailed(true); }
    } catch { setNativeControls(true); setFullscreenFailed(true); }
  };
  const disabled = player.failed || !model.blobHref;
  const seekDisabled = disabled || !player.duration;
  const playLabel = t(player.playing || player.pendingPlay ? a.pause : a.play);
  const metadata = [dimensions.width ? `${String(dimensions.width)} × ${String(dimensions.height)}` : "", player.duration ? formatAudioTime(player.duration) : "", model.basename.split(".").pop()?.toUpperCase(), formatBytes(model.sizeBytes)].filter(Boolean).join(" · ");
  return <ArtifactFrame testId="file-preview-video" frameRef={frameRef}>
    <PreviewToolbar label={t(c.videoControls)}>
      <span className="truncate px-2 text-xs font-medium text-wc-text-secondary">{t(strings.messagesFileViewer.videoPreview)}</span>
      <PreviewButton label={t(fullscreen ? c.exitFullscreen : c.fullscreen)} disabled={seekDisabled} onClick={() => { void toggleFullscreen(); }}><span>{fullscreen ? <Minimize2 /> : <Maximize2 />}</span></PreviewButton>
    </PreviewToolbar>
    <div className="relative min-h-0 flex-1 overflow-hidden bg-black">
      <video ref={player.mediaRef} src={model.blobHref} controls={nativeControls} playsInline preload="metadata" loop={player.repeat}
        aria-label={t(strings.messagesFileViewer.videoPreview)} className="h-full w-full object-contain" {...player.events}
        onLoadedMetadata={(event) => { player.events.onLoadedMetadata(); setDimensions({ width: event.currentTarget.videoWidth, height: event.currentTarget.videoHeight }); }}
        onClick={() => { if (!nativeControls) void player.togglePlay(); }}
      />
      {!nativeControls && !player.playing && !player.pendingPlay && !disabled && <div className="pointer-events-none absolute inset-0 flex items-center justify-center"><div className="pointer-events-auto"><PreviewButton label={t(a.play)} primary onClick={() => { void player.togglePlay(); }}><Play fill="currentColor" /></PreviewButton></div></div>}
      {player.waiting && <div role="status" className="pointer-events-none absolute inset-0 flex items-center justify-center bg-black/20"><span className="flex items-center gap-2 rounded-lg bg-black/80 p-3 text-sm text-white"><Loader2 className="h-4 w-4 motion-safe:animate-spin" />{t(a.buffering)}</span></div>}
      {player.failed && <div role="alert" className="absolute inset-0 flex items-center justify-center bg-wc-surface-base p-4"><PreviewNotice message={t(strings.messagesFileViewer.codecHint)} /></div>}
    </div>
    <div className="shrink-0 border-t border-wc-default bg-wc-surface-raised px-3 pb-2">
      <input type="range" dir="ltr" aria-label={t(a.seek)} aria-valuetext={formatAudioTime(player.time)} min="0" max={player.duration || 1} step="0.05" value={player.time}
        disabled={seekDisabled} onChange={(event) => { player.seek(Number(event.target.value)); }} className="h-11 w-full accent-wc-accent" data-testid={selectors.artifactPreview.videoSeek} />
      <div dir="ltr" className="flex justify-between font-mono text-[11px] tabular-nums text-wc-text-muted"><span>{formatAudioTime(player.time)}</span><span>{player.duration ? formatAudioTime(player.duration) : "–:––"}</span></div>
      <div role="group" aria-label={t(a.transport)} className="mt-2 flex items-center justify-between gap-1">
        <PreviewButton label={t(c.repeatVideo)} disabled={disabled} pressed={player.repeat} onClick={player.toggleRepeat}><Repeat /></PreviewButton>
        <div className="flex items-center gap-1">
          <PreviewButton label={t(a.back)} disabled={seekDisabled} onClick={() => { player.seek((player.mediaRef.current?.currentTime ?? player.time) - 10); }}><RotateCcw /></PreviewButton>
          <PreviewButton label={playLabel} disabled={disabled} primary onClick={() => { void player.togglePlay(); }} testId={selectors.artifactPreview.videoPlay}>{player.playing || player.pendingPlay ? <Pause fill="currentColor" /> : <Play fill="currentColor" />}</PreviewButton>
          <PreviewButton label={t(a.forward)} disabled={seekDisabled} onClick={() => { player.seek((player.mediaRef.current?.currentTime ?? player.time) + 10); }}><RotateCw /></PreviewButton>
        </div>
        <select aria-label={t(a.speed)} value={player.rate} disabled={disabled} className={`${previewSelectClass} w-14 border-transparent bg-transparent px-0`} onChange={(event) => { player.changeRate(Number(event.target.value)); }}>
          {[0.5, 0.75, 1, 1.25, 1.5, 2].map((rate) => <option key={rate} value={rate}>{rate}×</option>)}
        </select>
      </div>
      {player.playRejected && <p role="alert" className="mt-2 text-xs text-wc-error-text">{t(a.playRejected)}</p>}
      {fullscreenFailed && <p role="status" className="mt-2 text-xs text-wc-text-muted">{t(c.fullscreenFallback)}</p>}
      <details className="group mt-2 border-t border-wc-default">
        <summary className="flex min-h-11 cursor-pointer list-none items-center justify-between text-xs text-wc-text-secondary outline-none focus-visible:ring-2 focus-visible:ring-wc-accent [&::-webkit-details-marker]:hidden">{t(c.playbackOptions)}<ChevronDown className="h-4 w-4 transition group-open:rotate-180" /></summary>
        <div className="flex items-center gap-2">
          <PreviewButton label={t(player.muted ? a.unmute : a.mute)} disabled={disabled} onClick={player.toggleMute}>{player.muted || player.volume === 0 ? <VolumeX /> : <Volume2 />}</PreviewButton>
          <input type="range" aria-label={t(a.volume)} min="0" max="1" step="0.01" value={player.muted ? 0 : player.volume} disabled={disabled} onChange={(event) => { player.changeVolume(Number(event.target.value)); }} className="h-11 min-w-0 flex-1 accent-wc-accent" />
        </div>
        <p className="pb-2 text-[11px] text-wc-text-muted">{t(a.volumeHint)}</p>
      </details>
    </div>
    <ArtifactFooter model={model} metadata={metadata}>
      <button type="button" className={previewTextButtonClass} aria-pressed={nativeControls} onClick={() => { setNativeControls(!nativeControls); }}><Headphones className="h-4 w-4 shrink-0" />{t(a.systemControls)}</button>
    </ArtifactFooter>
  </ArtifactFrame>;
}
