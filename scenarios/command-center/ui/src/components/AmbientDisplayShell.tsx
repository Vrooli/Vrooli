import { useEffect, useState, type ReactNode } from "react";

export interface AmbientDisplayShellProps {
  theme: string; title: string; position: string; status?: ReactNode; legend?: ReactNode;
  samples?: string; paused?: boolean; held?: boolean; progress?: number;
  beatIndex?: number; beatCount?: number; beatProgress?: number; beatDurations?: number[];
  onBeatSelect?: (index: number) => void; children: ReactNode; manageDocumentTheme?: boolean;
}

export function AmbientDisplayShell({ theme, title, position, status, legend, samples = "mark", paused = false, held = false, progress = 0, beatIndex = 0, beatCount = 0, beatProgress = 0, beatDurations = [], onBeatSelect, children, manageDocumentTheme = true }: AmbientDisplayShellProps) {
  const [clock, setClock] = useState(() => new Date().toISOString());
  useEffect(() => { const timer = window.setInterval(() => setClock(new Date().toISOString()), 1000); return () => window.clearInterval(timer); }, []);
  useEffect(() => {
    if (!manageDocumentTheme) return undefined;
    document.documentElement.setAttribute("data-theme", theme);
    return () => document.documentElement.removeAttribute("data-theme");
  }, [manageDocumentTheme, theme]);
  const bounded = Math.max(0, Math.min(1, progress));
  return <div className="cc-shell" data-theme={theme} data-rcl-ambient-shell>
    <div className={beatCount > 0 ? "cc-cycle-rail cc-cycle-rail-segmented" : "cc-cycle-rail"} data-testid="cycle-rail" role={beatCount > 0 ? "group" : "progressbar"} aria-label={beatCount > 0 ? "Room sections" : paused ? "Cycle paused" : held ? "Reading the held section" : "Cycle running"} aria-valuenow={beatCount === 0 ? Math.round(bounded * 100) : undefined} aria-valuemin={beatCount === 0 ? 0 : undefined} aria-valuemax={beatCount === 0 ? 100 : undefined} data-paused={paused || undefined} data-held={held || undefined}>
      {beatCount > 0 ? Array.from({ length: beatCount }, (_, index) => {
        const fill = index < beatIndex ? 1 : index === beatIndex ? Math.max(0, Math.min(1, beatProgress)) : 0;
        const width = beatDurations[index] ?? 1;
        return <button type="button" className="cc-cycle-rail-segment" data-testid={`beat-rail-${index}`} data-active={index === beatIndex || undefined} aria-label={`Show section ${index + 1}`} aria-current={index === beatIndex ? "step" : undefined} onClick={() => onBeatSelect?.(index)} style={{ flexGrow: width }} key={index}><span style={{ transform: `scaleX(${fill.toFixed(3)})` }} /></button>;
      }) : <span data-testid="room-cycle-rail" style={{ transform: `scaleX(${bounded.toFixed(3)})` }} />}
    </div>
    {beatCount > 0 ? <span className="cc-visually-hidden" role="status" aria-live="polite">{paused ? "Cycle paused" : held ? "Reading the held section" : "Cycle running"}</span> : null}
    <header className="cc-eyebrow"><div className="cc-eyebrow-identity"><h1 className="cc-room-title">{title}</h1><span className="cc-eyebrow-line">{position}{paused ? " · PAUSED" : ""}</span></div><div className="cc-eyebrow-status">{status}<time className="cc-clock" dateTime={clock}>{new Date(clock).toLocaleTimeString()}</time></div></header>
    {children}{legend}{samples !== "mark" ? <span className="cc-mode-stamp" data-testid="samples-mode-stamp">samples {samples}</span> : null}
  </div>;
}
