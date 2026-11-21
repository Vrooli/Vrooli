import { useId, useMemo } from "react";
import { useFileViewerText } from "../../i18n/context.js";
import { strings } from "../../i18n/strings.js";
import { selectors } from "../../selectors.js";
import { formatAudioTime } from "../audioAnalysis.js";

interface Props {
  peaks: number[];
  duration: number;
  time: number;
  start: number;
  end: number;
  loopA: number | null;
  loopB: number | null;
  loopEnabled: boolean;
  disabled: boolean;
  onSeek: (time: number) => void;
}

export function AudioWaveform({ peaks, duration, time, start, end, loopA, loopB, loopEnabled, disabled, onSeek }: Props) {
  const { t } = useFileViewerText();
  const clipId = useId();
  const span = end - start;
  const position = (at: number) => span > 0 ? Math.max(0, Math.min(1000, (at - start) / span * 1000)) : 0;
  const cursor = position(time);
  const bars = useMemo(() => {
    if (!peaks.length || duration <= 0 || span <= 0) return "";
    const count = 120;
    const path: string[] = [];
    for (let i = 0; i < count; i++) {
      const from = Math.floor((start + i / count * span) / duration * peaks.length);
      const to = Math.max(from + 1, Math.ceil((start + (i + 1) / count * span) / duration * peaks.length));
      let height = 0;
      for (let p = from; p < Math.min(to, peaks.length); p++) height = Math.max(height, peaks[p] ?? 0);
      const half = Math.max(1, height * 49);
      path.push(`M${String(i / count * 1000 + 4)} ${String(64 - half)}v${String(half * 2)}`);
    }
    return path.join("");
  }, [peaks, duration, span, start]);
  const value = Math.max(start, Math.min(end, time));
  return (
    <div dir="ltr">
      <div className="group relative h-28 rounded-xl bg-fv-surface-input/60 focus-within:ring-2 focus-within:ring-fv-accent" data-testid={selectors.audioPreview.waveform}>
        <svg viewBox="0 0 1000 128" preserveAspectRatio="none" className="h-full w-full overflow-hidden rounded-xl" aria-hidden="true">
          <defs><clipPath id={clipId}><rect width={cursor} height="128" /></clipPath></defs>
          {loopA !== null && loopB !== null && loopB > start && loopA < end && (
            <rect x={position(loopA)} width={position(loopB) - position(loopA)} height="128" className={loopEnabled ? "fill-fv-accent/15" : "fill-fv-text-muted/10"} />
          )}
          {bars ? <>
            <path d={bars} stroke="currentColor" strokeWidth="4" strokeLinecap="round" className="text-fv-text-muted/50" />
            <path d={bars} stroke="currentColor" strokeWidth="4" strokeLinecap="round" className="text-fv-accent" clipPath={`url(#${clipId})`} />
          </> : <>
            <path d="M12 64H988" stroke="currentColor" strokeWidth="5" strokeLinecap="round" className="text-fv-text-muted/40" />
            <path d="M12 64H988" stroke="currentColor" strokeWidth="5" strokeLinecap="round" className="text-fv-accent" clipPath={`url(#${clipId})`} />
          </>}
          {loopA !== null && loopA >= start && loopA <= end && <path d={`M${String(position(loopA))} 0V128`} stroke="currentColor" strokeDasharray="4 4" strokeWidth="2" className="text-fv-accent" />}
          {loopB !== null && loopB >= start && loopB <= end && <path d={`M${String(position(loopB))} 0V128`} stroke="currentColor" strokeDasharray="4 4" strokeWidth="2" className="text-fv-accent" />}
          {time >= start && time <= end && <path d={`M${String(cursor)} 8V120`} stroke="currentColor" strokeWidth="2" className="text-fv-text-primary" />}
        </svg>
        <input
          type="range" min={start} max={end || 1} step="0.05" value={value}
          aria-label={t(strings.audioFilePlayer.seek)} aria-valuetext={formatAudioTime(value)}
          disabled={disabled || duration <= 0}
          onChange={(event) => { onSeek(Number(event.target.value)); }}
          className="absolute inset-0 m-0 h-full w-full cursor-pointer opacity-0 disabled:cursor-default"
          data-testid={selectors.audioPreview.seek}
        />
      </div>
      <div className="mt-2 flex justify-between gap-4 font-mono text-xs tabular-nums text-fv-text-secondary">
        <span>{formatAudioTime(time)}</span><span>{duration > 0 ? formatAudioTime(duration) : "–:––"}</span>
      </div>
    </div>
  );
}
