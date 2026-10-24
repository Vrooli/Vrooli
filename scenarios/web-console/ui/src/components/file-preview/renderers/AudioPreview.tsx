import { useId, useState, type ReactNode } from "react";
import { IconButton } from "@vrooli/react-component-library/IconButton";
import { ChevronDown, Crosshair, Download, ExternalLink, Headphones, Loader2, Music2, Pause, Play, Repeat, RotateCcw, RotateCw, Volume2, VolumeX, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { strings } from "../../../consts/strings";
import { selectors } from "../../../consts/selectors";
import { cn } from "../../../lib/classnames";
import { formatBytes } from "../format";
import { audioWindow, formatAudioTime } from "../audioAnalysis";
import { useAudioAnalysis } from "../useAudioAnalysis";
import { useMediaPlayer } from "../useMediaPlayer";
import type { PreviewRendererProps } from "../types";
import { PreviewNotice } from "./shared";
import { AudioWaveform } from "./AudioWaveform";

const copy = strings.audioFilePlayer;
const selectClass = "h-11 min-w-0 rounded-lg border border-wc-default bg-wc-surface-input px-2 text-xs text-wc-text-secondary outline-none focus-visible:ring-2 focus-visible:ring-wc-accent disabled:opacity-40";
const textButton = "inline-flex min-h-11 items-center justify-center gap-2 rounded-lg border border-wc-default bg-wc-surface-input px-3 text-xs font-medium text-wc-text-secondary transition hover:bg-wc-surface-raised focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-wc-accent disabled:opacity-40";

function PlayerButton({ label, children, onClick, disabled, pressed, primary = false, testId }: {
  label: string; children: ReactNode; onClick: () => void; disabled?: boolean; pressed?: boolean; primary?: boolean; testId?: string;
}) {
  return <IconButton type="button" surface={primary ? "solid" : "ghost"} shape={primary ? "circle" : "rounded"} size={primary ? "lg" : "md"} aria-label={label} title={label}
    aria-pressed={pressed} disabled={disabled} onClick={onClick} data-testid={testId}
    style={{ inlineSize: primary ? "3.5rem" : "2.75rem", blockSize: primary ? "3.5rem" : "2.75rem" }}
    className="shrink-0"
  >{children}</IconButton>;
}

// Keying the full session also tears down transport/analysis when files change.
export function AudioPreview(props: PreviewRendererProps) {
  return <AudioPlayer key={`${props.model.previewId}:${props.model.blobHref}`} model={props.model} />;
}

function AudioPlayer({ model }: Pick<PreviewRendererProps, "model">) {
  const { t } = useTranslation();
  const player = useMediaPlayer<HTMLAudioElement>(model.blobHref);
  const analysis = useAudioAnalysis(model, player.duration);
  const [zoom, setZoom] = useState(1);
  const [windowStart, setWindowStart] = useState(0);
  const [nativeControls, setNativeControls] = useState(false);
  const [artworkFailed, setArtworkFailed] = useState(false);
  const toolsId = useId();
  const view = audioWindow(player.duration, zoom, windowStart);
  const unavailable = player.failed || !model.blobHref;
  const seekDisabled = unavailable || !player.duration;
  const format = model.basename.split(".").pop()?.toUpperCase() ?? model.mimeType;
  const metadata = [player.duration > 0 ? formatAudioTime(player.duration) : null, format, formatBytes(model.sizeBytes)].filter(Boolean).join(" · ");
  const playLabel = t(player.playing || player.pendingPlay ? copy.pause : copy.play);
  const showPlayhead = () => { setWindowStart(Math.max(0, player.time - view.span / 2)); };
  const title = analysis.title || model.basename;

  return (
    <div className="h-full min-w-0 overflow-y-auto bg-wc-surface-base p-3 sm:p-6" data-testid="file-preview-audio">
      <section aria-label={t(strings.messagesFileViewer.audioPreview)} className="mx-auto w-full max-w-2xl overflow-hidden rounded-2xl border border-wc-default bg-wc-surface-raised shadow-sm" data-testid={selectors.audioPreview.player}>
        <div className="flex min-w-0 items-center gap-3 p-4 sm:gap-4 sm:p-5">
          <div className="flex h-14 w-14 shrink-0 items-center justify-center overflow-hidden rounded-xl border border-wc-accent bg-wc-accent/10 text-wc-accent sm:h-16 sm:w-16">
            {analysis.artwork && !artworkFailed
              ? <img src={analysis.artwork} alt="" className="h-full w-full object-cover" onError={() => { setArtworkFailed(true); }} />
              : <Music2 className="h-7 w-7" aria-hidden="true" />}
          </div>
          <div className="min-w-0 flex-1">
            <p className="mb-1 text-[10px] font-semibold uppercase tracking-widest text-wc-text-muted">{t(strings.messagesFileViewer.audioPreview)}</p>
            <h3 className="truncate text-base font-semibold text-wc-text-primary" title={title}>{title}</h3>
            {analysis.artist && <p className="truncate text-xs text-wc-text-secondary" title={analysis.artist}>{analysis.artist}</p>}
            <p className="mt-1 truncate text-xs text-wc-text-muted" title={metadata}>{metadata}</p>
          </div>
        </div>

        <div className="px-4 sm:px-5">
          <AudioWaveform peaks={analysis.peaks} duration={player.duration} time={player.time} start={view.start} end={view.end}
            loopA={player.loopA} loopB={player.loopB} loopEnabled={player.loopEnabled} disabled={seekDisabled} onSeek={player.seek} />
          <p role="status" className="mt-2 min-h-4 text-center text-[11px] text-wc-text-muted">
            {unavailable ? t(copy.unavailable) : player.waiting ? t(copy.buffering) : analysis.loading ? t(copy.loadingWaveform) : !analysis.peaks.length ? t(copy.seekFallback) : zoom > 1 ? `${formatAudioTime(view.start)} – ${formatAudioTime(view.end)}` : t(copy.waveformHint)}
          </p>
          <div className="my-3 flex items-center justify-between gap-1" role="group" aria-label={t(copy.transport)}>
            <PlayerButton label={t(copy.repeat)} pressed={player.repeat} onClick={player.toggleRepeat} disabled={unavailable} testId={selectors.audioPreview.repeat}><Repeat className="h-4 w-4" /></PlayerButton>
            <div className="flex items-center gap-2 sm:gap-4">
              <PlayerButton label={t(copy.back)} onClick={() => { player.seek((player.mediaRef.current?.currentTime ?? player.time) - 10); }} disabled={seekDisabled}>
                <span className="relative"><RotateCcw className="h-6 w-6" /><span className="absolute inset-0 flex items-center justify-center pt-1 text-[8px] font-semibold">10</span></span>
              </PlayerButton>
              <PlayerButton label={playLabel} primary onClick={() => { void player.togglePlay(); }} disabled={unavailable} testId={selectors.audioPreview.play}>
                {player.waiting ? <Loader2 className="h-6 w-6 motion-safe:animate-spin" /> : player.playing || player.pendingPlay ? <Pause className="h-6 w-6" fill="currentColor" /> : <Play className="ml-0.5 h-6 w-6" fill="currentColor" />}
              </PlayerButton>
              <PlayerButton label={t(copy.forward)} onClick={() => { player.seek((player.mediaRef.current?.currentTime ?? player.time) + 10); }} disabled={seekDisabled}>
                <span className="relative"><RotateCw className="h-6 w-6" /><span className="absolute inset-0 flex items-center justify-center pt-1 text-[8px] font-semibold">10</span></span>
              </PlayerButton>
            </div>
            <select aria-label={t(copy.speed)} title={t(copy.speed)} value={player.rate} disabled={unavailable}
              onChange={(event) => { player.changeRate(Number(event.target.value)); }} className={cn(selectClass, "w-14 border-transparent bg-transparent px-0 text-center")}>
              {[0.5, 0.75, 1, 1.25, 1.5, 2].map((rate) => <option key={rate} value={rate}>{rate}×</option>)}
            </select>
          </div>
          {player.failed && <div role="alert" className="mb-4"><PreviewNotice message={t(strings.messagesFileViewer.codecHint)} /></div>}
          {player.playRejected && !player.failed && <p role="alert" className="mb-4 text-sm text-wc-error-text">{t(copy.playRejected)}</p>}
        </div>

        <details className="group border-t border-wc-default" data-testid={selectors.audioPreview.tools}>
          <summary className="flex min-h-12 cursor-pointer list-none items-center justify-between gap-2 px-4 text-xs font-medium text-wc-text-secondary outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-wc-accent sm:px-5 [&::-webkit-details-marker]:hidden" aria-controls={toolsId}>
            <span>{t(copy.tools)}{player.loopEnabled ? ` · ${t(copy.loopActive)}` : ""}</span><ChevronDown className="h-4 w-4 transition group-open:rotate-180" aria-hidden="true" />
          </summary>
          <div id={toolsId} className="space-y-4 px-4 pb-4 sm:px-5">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <label className="flex items-center gap-2 text-xs text-wc-text-secondary">
                {t(copy.zoom)}
                <select aria-label={t(copy.zoom)} value={zoom} disabled={unavailable || !analysis.peaks.length}
                  onChange={(event) => { const next = Number(event.target.value); setZoom(next); setWindowStart(Math.max(0, player.time - player.duration / next / 2)); }} className={selectClass}>
                  {[1, 2, 4, 8].map((scale) => <option key={scale} value={scale}>{scale === 1 ? t(copy.fullTrack) : `${String(scale)}×`}</option>)}
                </select>
              </label>
              <PlayerButton label={t(copy.showPlayhead)} onClick={showPlayhead} disabled={seekDisabled || zoom === 1}><Crosshair className="h-4 w-4" /></PlayerButton>
            </div>
            {zoom > 1 && <div dir="ltr">
              <label className="flex items-center justify-between gap-2 text-xs text-wc-text-secondary" htmlFor={`${toolsId}-window`}><span>{t(copy.timeWindow)}</span><span className="font-mono tabular-nums">{formatAudioTime(view.start)} – {formatAudioTime(view.end)}</span></label>
              <input id={`${toolsId}-window`} type="range" aria-label={t(copy.timeWindow)} min="0" max={player.duration - view.span} step="0.05" value={view.start}
                aria-valuetext={`${formatAudioTime(view.start)} – ${formatAudioTime(view.end)}`} onChange={(event) => { setWindowStart(Number(event.target.value)); }}
                className="h-11 w-full accent-wc-accent" />
            </div>}
            <div className="space-y-2 border-t border-wc-default pt-3">
              <p className="text-xs text-wc-text-secondary">{t(copy.loopHelp)}</p>
              <div className="grid grid-cols-2 gap-2">
                <button type="button" className={textButton} disabled={seekDisabled || player.time >= player.duration - 0.25} onClick={player.markA} aria-label={t(copy.setA)} aria-describedby={`${toolsId}-a`}>{t(copy.pointA)} <span id={`${toolsId}-a`} className="font-mono tabular-nums">{player.loopA === null ? "–:––" : formatAudioTime(player.loopA, true)}</span></button>
                <button type="button" className={textButton} disabled={seekDisabled || player.loopA === null || player.time - player.loopA < 0.25} onClick={player.markB} aria-label={t(copy.setB)} aria-describedby={`${toolsId}-b`}>{t(copy.pointB)} <span id={`${toolsId}-b`} className="font-mono tabular-nums">{player.loopB === null ? "–:––" : formatAudioTime(player.loopB, true)}</span></button>
              </div>
              <div className="flex items-center justify-between gap-2">
                <button type="button" className={cn(textButton, "flex-1", player.loopEnabled && "border-wc-accent bg-wc-accent/10 text-wc-accent")} aria-pressed={player.loopEnabled} disabled={unavailable || player.loopA === null || player.loopB === null} onClick={player.toggleLoop}>
                  <Repeat className="h-4 w-4" />{t(copy.loopSelection)}
                </button>
                <PlayerButton label={t(copy.clearLoop)} onClick={player.clearLoop} disabled={player.loopA === null}><X className="h-4 w-4" /></PlayerButton>
              </div>
            </div>
            <div className="flex items-center gap-2 border-t border-wc-default pt-2">
              <PlayerButton label={t(player.muted ? copy.unmute : copy.mute)} onClick={player.toggleMute} disabled={unavailable}>{player.muted || player.volume === 0 ? <VolumeX className="h-4 w-4" /> : <Volume2 className="h-4 w-4" />}</PlayerButton>
              <input type="range" aria-label={t(copy.volume)} aria-valuetext={`${String(Math.round((player.muted ? 0 : player.volume) * 100))}%`} min="0" max="1" step="0.01" value={player.muted ? 0 : player.volume} disabled={unavailable} onChange={(event) => { player.changeVolume(Number(event.target.value)); }} className="h-11 min-w-0 flex-1 accent-wc-accent" />
            </div>
            <p className="text-[11px] text-wc-text-muted">{t(copy.volumeHint)}</p>
          </div>
        </details>
      </section>

      <div className="mx-auto mt-3 flex w-full max-w-2xl flex-wrap items-start justify-between gap-2">
        {model.blobHref && model.canDownload && <a href={model.blobHref} download={model.basename} className={cn(textButton, "border-transparent bg-transparent")} data-testid="file-preview-download"><Download className="h-4 w-4" />{t(strings.messagesFileViewer.download)}</a>}
        <details className="group min-w-0 max-w-full">
          <summary className={cn(textButton, "cursor-pointer list-none border-transparent bg-transparent [&::-webkit-details-marker]:hidden")}>
            {t(copy.moreActions)}<ChevronDown className="h-3.5 w-3.5 transition group-open:rotate-180" aria-hidden="true" />
          </summary>
          <div className="flex flex-col items-start gap-1 py-2">
            {model.blobHref && <a href={model.blobHref} target="_blank" rel="noreferrer" className={textButton} data-testid="file-preview-open-new-tab"><ExternalLink className="h-4 w-4" />{t(strings.messagesFileViewer.openInNewTab)}</a>}
            <button type="button" className={textButton} onClick={() => { setNativeControls(!nativeControls); }} aria-pressed={nativeControls}><Headphones className="h-4 w-4" />{t(copy.systemControls)}</button>
          </div>
        </details>
      </div>
      <audio ref={player.mediaRef} src={model.blobHref} controls={nativeControls} preload="metadata" loop={player.repeat}
        aria-label={t(copy.systemControls)} className={cn("mx-auto mt-3 w-full max-w-2xl", !nativeControls && "hidden")} {...player.events} />
    </div>
  );
}
