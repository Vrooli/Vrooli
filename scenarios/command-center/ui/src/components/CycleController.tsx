import { useEffect, useRef, useState, type ReactNode } from "react";

export interface CycleControllerProps { durationSeconds: number; paused?: boolean; onAdvance?: () => void; children?: (progress: number) => ReactNode; }

export function CycleController({ durationSeconds, paused = false, onAdvance, children }: CycleControllerProps) {
  const [progress, setProgress] = useState(0); const progressRef = useRef(0); const last = useRef(performance.now()); progressRef.current = progress;
  useEffect(() => { last.current = performance.now(); const timer = window.setInterval(() => { const now = performance.now(); if (paused) { last.current = now; return; } const next = progressRef.current + Math.max(0, now - last.current) / 1000 / Math.max(.001, durationSeconds); last.current = now; if (next >= 1) { progressRef.current = 0; setProgress(0); onAdvance?.(); } else { progressRef.current = next; setProgress(next); } }, 250); return () => window.clearInterval(timer); }, [durationSeconds, onAdvance, paused]);
  return <div data-rcl-cycle-controller data-paused={paused || undefined} data-progress={progress.toFixed(3)}>{children?.(progress)}</div>;
}
