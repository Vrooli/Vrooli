import { useEffect, useState, type ReactNode } from "react";

export interface AmbientDisplayShellProps {
  theme: string;
  title: string;
  position: string;
  status?: ReactNode;
  legend?: ReactNode;
  samples?: string;
  paused?: boolean;
  progress?: number;
  cycleSeconds?: number;
  beatIndex?: number;
  beatCount?: number;
  beatProgress?: number;
  children: ReactNode;
}

export function AmbientDisplayShell({
  theme,
  title,
  position,
  status,
  legend,
  samples = "mark",
  paused = false,
  progress = 0,
  cycleSeconds = 60,
  beatIndex = 0,
  beatCount = 0,
  beatProgress = 0,
  children,
}: AmbientDisplayShellProps) {
  const [clock, setClock] = useState(() => new Date().toISOString());
  useEffect(() => {
    const timer = window.setInterval(() => setClock(new Date().toISOString()), 1000);
    return () => window.clearInterval(timer);
  }, []);
  useEffect(() => {
    document.documentElement.setAttribute("data-theme", theme);
    return () => document.documentElement.removeAttribute("data-theme");
  }, [theme]);
  const bounded = Math.max(0, Math.min(1, progress));
  return (
    <div className="cc-shell" data-theme={theme} data-rcl-ambient-shell>
      <div
        className={beatCount > 0 ? "cc-cycle-rail cc-cycle-rail-segmented" : "cc-cycle-rail"}
        data-testid="cycle-rail"
        role="progressbar"
        aria-label={paused ? "Cycle paused" : "Cycle running"}
        aria-valuenow={Math.round(bounded * 100)}
        aria-valuemin={0}
        aria-valuemax={100}
        data-paused={paused || undefined}
      >
        {beatCount > 0 ? (
          Array.from({ length: beatCount }, (_, index) => {
            const fill =
              index < beatIndex
                ? 1
                : index === beatIndex
                  ? Math.max(0, Math.min(1, beatProgress))
                  : 0;
            return (
              <span
                className="cc-cycle-rail-segment"
                data-testid={`beat-rail-${index}`}
                data-active={index === beatIndex || undefined}
                key={index}
              >
                <span style={{ transform: `scaleX(${fill.toFixed(3)})` }} />
              </span>
            );
          })
        ) : (
          <span
            data-testid="room-cycle-rail"
            style={{ transform: `scaleX(${bounded.toFixed(3)})` }}
          />
        )}
      </div>
      <header className="cc-eyebrow">
        <div className="cc-eyebrow-identity">
          <h1 className="cc-room-title">{title}</h1>
          <span className="cc-eyebrow-line">
            {position}
            {paused ? " · PAUSED" : ` · CYCLE ${cycleSeconds}S`}
          </span>
        </div>
        <div className="cc-eyebrow-status">
          {status}
          <time className="cc-clock" dateTime={clock}>
            {new Date(clock).toLocaleTimeString()}
          </time>
        </div>
      </header>
      {children}
      {legend}
      {samples !== "mark" ? (
        <span className="cc-mode-stamp" data-testid="samples-mode-stamp">
          samples {samples}
        </span>
      ) : null}
    </div>
  );
}
