import { createContext, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { useLocation, useNavigate, useSearchParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { GamepadInputManager, type GamepadAction } from "@vrooli/iframe-bridge/spatial";
import { fetchBoard } from "../lib/api";
import { useBoardKeyboard } from "../hooks/useBoardKeyboard";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Select } from "@vrooli/react-component-library/Select/1.2.1";

export type BoardIntent = GamepadAction | "pause-cycle" | "reveal-controls" | "show-help" | "inspect";
export type SamplesMode = "hide" | "mark" | "full";
interface BoardControllerValue { samples: SamplesMode; paused: boolean; controlsVisible: boolean; dispatch: (intent: BoardIntent) => void; }
const BoardContext = createContext<BoardControllerValue | null>(null);

export function useBoardController(): BoardControllerValue {
  const value = useContext(BoardContext);
  if (!value) throw new Error("useBoardController must be used inside BoardController");
  return value;
}

export function BoardController({ children }: { children: ReactNode }) {
  const navigate = useNavigate();
  const location = useLocation();
  const [searchParams, setSearchParams] = useSearchParams();
  const { data } = useQuery({ queryKey: ["board-shape"], queryFn: fetchBoard, staleTime: 30_000 });
  const rooms = data?.rooms ?? [];
  const [controlsVisible, setControlsVisible] = useState(false);
  const [acknowledgement, setAcknowledgement] = useState("Ready");
  const [pausedUntil, setPausedUntil] = useState(0);
  const [progress, setProgress] = useState(0);
  const cycleSeconds = Math.max(5, Number(searchParams.get("cycle") ?? 60) || 60);
  const samples = parseSamples(searchParams.get("samples"));
  const touchStart = useRef<{ x: number; y: number; at: number } | null>(null);
  const lastInputAt = useRef(Date.now());
  const cycleStartedAt = useRef(Date.now());

  const setParam = (key: string, value: string) => {
    const next = new URLSearchParams(searchParams);
    next.set(key, value);
    setSearchParams(next, { replace: true });
  };
  const navigateRoom = (delta: number) => {
    if (!rooms.length) return;
    const current = rooms.findIndex((room) => `/${room.id}` === location.pathname);
    const index = Math.max(0, current) + delta;
    const next = rooms[(index + rooms.length) % rooms.length];
    if (next) navigate(`/${next.id}${location.search}`);
  };
  const dispatch = (intent: BoardIntent) => {
    const now = Date.now();
    lastInputAt.current = now;
    cycleStartedAt.current = now;
    setControlsVisible(true);
    setPausedUntil(now + 20_000);
    setAcknowledgement(intent === "show-help" ? "Help shown" : intent === "inspect" ? "Inspection acknowledged" : `${intent} acknowledged`);
    if (intent === "page-next" || intent === "navigate-right") navigateRoom(1);
    if (intent === "page-prev" || intent === "navigate-left") navigateRoom(-1);
    if (intent === "pause-cycle" || intent === "select") setPausedUntil((value) => value > now ? 0 : now + 20_000);
    if (intent === "menu" || intent === "reveal-controls") setControlsVisible(true);
    if (intent === "back") setControlsVisible(false);
    if (intent === "show-help") setControlsVisible(true);
  };

  useEffect(() => {
    const requestedRoom = searchParams.get("room");
    if (requestedRoom && rooms.some((room) => room.id === requestedRoom) && location.pathname !== `/${requestedRoom}`) {
      navigate(`/${requestedRoom}${location.search}`, { replace: true });
    }
  }, [location.pathname, location.search, navigate, rooms, searchParams]);

  useEffect(() => {
    if (searchParams.get("fullscreen") !== "1" || !document.documentElement.requestFullscreen) return;
    void document.documentElement.requestFullscreen().catch(() => dispatch("show-help"));
  }, [searchParams]);

  useBoardKeyboard({ rooms, search: location.search, navigate, dispatch });

  useEffect(() => {
    const manager = new GamepadInputManager({ onAction: dispatch });
    manager.start();
    return () => manager.dispose();
  }, [rooms.length, location.pathname, location.search]);

  useEffect(() => {
    const onTouchStart = (event: TouchEvent) => { const point = event.changedTouches[0]; if (!point) return; touchStart.current = { x: point.clientX, y: point.clientY, at: Date.now() }; dispatch("reveal-controls"); };
    const onTouchEnd = (event: TouchEvent) => {
      const start = touchStart.current; const point = event.changedTouches[0];
      if (!point) return;
      if (!start) return;
      const dx = point.clientX - start.x; const dy = point.clientY - start.y; const elapsed = Date.now() - start.at;
      if (elapsed >= 400 && Math.abs(dx) < 30 && Math.abs(dy) < 30) dispatch("pause-cycle");
      else if (Math.abs(dx) > 48 && Math.abs(dx) > Math.abs(dy)) dispatch(dx < 0 ? "page-next" : "page-prev");
      touchStart.current = null;
    };
    window.addEventListener("touchstart", onTouchStart, { passive: true });
    window.addEventListener("touchend", onTouchEnd, { passive: true });
    return () => { window.removeEventListener("touchstart", onTouchStart); window.removeEventListener("touchend", onTouchEnd); };
  });

  useEffect(() => {
    const timer = window.setInterval(() => {
      const now = Date.now();
      setControlsVisible((visible) => visible && now - lastInputAt.current < 4_000);
      const paused = pausedUntil > now;
      setProgress(paused ? 0 : Math.min(1, (now - cycleStartedAt.current) / (cycleSeconds * 1000)));
      if (!paused && now - cycleStartedAt.current >= cycleSeconds * 1000 && rooms.length && !document.hidden) {
        cycleStartedAt.current = now;
        navigateRoom(1);
      }
    }, 250);
    return () => window.clearInterval(timer);
  }, [cycleSeconds, pausedUntil, rooms.length, location.pathname, location.search]);

  const value = useMemo(() => ({ samples, paused: pausedUntil > Date.now(), controlsVisible, dispatch }), [controlsVisible, pausedUntil, samples]);
  return <BoardContext.Provider value={value}>
    <div data-board-root data-samples-mode={samples} onPointerDown={() => dispatch("reveal-controls")}>
      <div className="cc-cycle-rail" data-testid="cycle-rail" aria-label={pausedUntil > Date.now() ? "Cycle paused" : "Cycle running"} data-experience-surface="cycle-rail" data-experience-state="static"><span data-testid="room-cycle-rail" style={{ transform: `scaleX(${progress})` }} /></div>
      {!controlsVisible ? <div className="cc-controls-silent" data-testid="control-bar-controls" data-experience-surface="controls" data-experience-state="static" aria-hidden="true" /> : null}
      {children}
      {!controlsVisible ? <output className="cc-acknowledgement-silent" data-testid="control-bar-acknowledgement" data-experience-surface="acknowledgement" data-experience-state="ready" aria-live="polite">{acknowledgement}</output> : null}
      {controlsVisible ? <ControlBar paused={pausedUntil > Date.now()} samples={samples} acknowledgement={acknowledgement} onIntent={dispatch} onSamples={(mode) => setParam("samples", mode)} /> : null}
    </div>
  </BoardContext.Provider>;
}

function ControlBar({ paused, samples, acknowledgement, onIntent, onSamples }: { paused: boolean; samples: SamplesMode; acknowledgement: string; onIntent: (intent: BoardIntent) => void; onSamples: (mode: SamplesMode) => void }) {
  return <nav className="cc-control-bar" aria-label="Board controls" data-testid="control-bar-controls" data-experience-surface="controls" data-experience-state="ready">
    <Button type="button" variant="ghost" size="sm" onClick={() => onIntent("page-prev")}>Previous</Button>
    <Button type="button" variant="ghost" size="sm" onClick={() => onIntent("pause-cycle")}>{paused ? "Resume" : "Pause"}</Button>
    <Button type="button" variant="ghost" size="sm" onClick={() => onIntent("page-next")}>Next</Button>
    <Button type="button" variant="ghost" size="sm" onClick={() => onIntent("show-help")}>Help</Button>
    <label>Samples <Select value={samples} onChange={(event) => onSamples(parseSamples(event.target.value))} options={[{ value: "hide", label: "Hide" }, { value: "mark", label: "Mark" }, { value: "full", label: "Full" }]} /></label>
    <output data-testid="control-bar-acknowledgement" data-experience-surface="acknowledgement" data-experience-state="ready" aria-live="polite">{acknowledgement}</output>
  </nav>;
}

function parseSamples(value: string | null): SamplesMode { return value === "hide" || value === "full" ? value : "mark"; }
