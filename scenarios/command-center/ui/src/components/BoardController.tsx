import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { useLocation, useNavigate, useSearchParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { useGamepad } from "@vrooli/iframe-bridge/react";
import { useBoardKeyboard } from "../hooks/useBoardKeyboard";
import { fetchBoard, fetchBoardSettings } from "../lib/api";
import { BoardContext, BoardProgressContext, parseSamples, type BoardControllerValue, type BoardIntent, type BoardProgress, type SamplesMode } from "../lib/boardContext";
import { beatPositionAtProgress, buildBeatDurations, progressAtBeat, remapProgress, roomNavigationSuffix, tickCycle } from "../lib/cycle";


const IDLE_RESUME_MS = 20_000;
/** A held beat waits at its end for at most this long before the cycle moves on regardless. */
const MAX_HOLD_MS = 90_000;
/** Backstop margin past a beat's reading time and the max hold, after which a stuck beat is released and skipped. */
const BEAT_STALL_MARGIN_MS = 30_000;
const CONTROLS_HIDE_MS = 4_000;
const TRANSITION_MS = 900;

/** Routes that are operator surfaces, not kiosk board rooms: the auto-cycle must
 *  never evict them, and board keyboard shortcuts must not fire while one is open. */

export function BoardController({ children }: { children: ReactNode }) {
  const gamepadRef = useRef<HTMLDivElement>(null);
  const navigate = useNavigate();
  const location = useLocation();
  const [searchParams, setSearchParams] = useSearchParams();
  const { data: board } = useQuery({ queryKey: ["board-shape"], queryFn: fetchBoard, staleTime: 30_000 });
  const { data: boardSettings } = useQuery({ queryKey: ["board-settings"], queryFn: fetchBoardSettings ?? (async () => ({ cycleSeconds: 60, transition: "crossfade", rooms: [] })), staleTime: 30_000 });
  const rooms = useMemo(() => board?.rooms ?? [], [board]);
  const currentRoom = useMemo(() => rooms.find((room) => location.pathname === `/${room.id}`), [location.pathname, rooms]);
  const [controlsVisible, setControlsVisible] = useState(false);
  const [helpVisible, setHelpVisible] = useState(false);
  const [acknowledgement, setAcknowledgement] = useState("Ready");
  const [pausedUntil, setPausedUntil] = useState(0);
  const [progress, setProgress] = useState(0);
  const [held, setHeld] = useState(false);
  const [transitioning, setTransitioning] = useState(false);
  const cycleSeconds = Math.max(5, Number(searchParams.get("cycle") ?? boardSettings?.cycleSeconds ?? 60) || 60);
  const [reading, setReading] = useState<{ roomId: string; seconds: number[] } | null>(null);
  const readingSeconds = reading && reading.roomId === currentRoom?.id ? reading.seconds : undefined;
  const beatDurations = useMemo(() => buildBeatDurations(currentRoom?.beats ?? [], cycleSeconds, readingSeconds), [currentRoom, cycleSeconds, readingSeconds]);
  const beatPosition = useMemo(() => beatPositionAtProgress(progress, beatDurations), [beatDurations, progress]);
  const roomDwellSeconds = useMemo(() => (beatDurations.length ? beatDurations.reduce((sum, duration) => sum + duration, 0) : cycleSeconds), [beatDurations, cycleSeconds]);
  const samples = parseSamples(searchParams.get("samples"));
  const touchStart = useRef<{ x: number; y: number; at: number } | null>(null);
  const lastInputAt = useRef(Date.now());
  const cycleStartedAt = useRef(Date.now());
  const pathRef = useRef(location.pathname);
  const restoredPathRef = useRef<string | null>(null);
  const restoringBeatRef = useRef<{ path: string; index: number } | null>(null);
  const transitionTimersRef = useRef<number[]>([]);
  const progressRef = useRef(0);
  const holdsRef = useRef(new Set<string>());
  const heldSinceRef = useRef<number | null>(null);
  const beatStallRef = useRef({ index: -1, at: 0 });

  const showProgress = useCallback((value: number) => {
    progressRef.current = value;
    setProgress(value);
  }, []);

  const holdBeat = useCallback((id: string, holding: boolean) => {
    if (holding) holdsRef.current.add(id);
    else holdsRef.current.delete(id);
  }, []);

  const reportReadingSeconds = useCallback((roomId: string, seconds: number[]) => {
    setReading((current) => (current?.roomId === roomId && current.seconds.length === seconds.length && current.seconds.every((value, index) => Math.abs(value - (seconds[index] ?? 0)) < 0.05) ? current : { roomId, seconds }));
  }, []);

  // Reading time follows the data. When it changes inside a room, stay on the same beat at the same point through it.
  const durationsRef = useRef<{ roomId?: string; durations: number[] }>({ durations: [] });
  useLayoutEffect(() => {
    const previous = durationsRef.current;
    durationsRef.current = { roomId: currentRoom?.id, durations: beatDurations };
    if (previous.roomId !== currentRoom?.id || previous.durations === beatDurations) return;
    const next = remapProgress(progressRef.current, previous.durations, beatDurations);
    cycleStartedAt.current = Date.now() - next * roomDwellSeconds * 1000;
    showProgress(next);
  }, [beatDurations, currentRoom?.id, roomDwellSeconds, showProgress]);

  const cancelTransition = useCallback(() => {
    transitionTimersRef.current.forEach((timer) => window.clearTimeout(timer));
    transitionTimersRef.current = [];
  }, []);

  const setSamples = useCallback((mode: SamplesMode) => {
    const next = new URLSearchParams(searchParams);
    next.set("samples", mode);
    setSearchParams(next, { replace: true });
    setAcknowledgement(`Samples: ${mode}`);
  }, [searchParams, setSearchParams]);

  const goTo = useCallback((path: string) => {
    if (path === pathRef.current) return;
    cancelTransition();
    setTransitioning(true);
    const isRoom = rooms.some((room) => path === `/${room.id}`);
    const suffix = isRoom ? roomNavigationSuffix(location.search) : location.search;
    // Rooms replace history so kiosk cycling never fills the back stack; operator
    // sub-pages (focus, open-loop, settings) push so the board stays one step back.
    const navigationTimer = window.setTimeout(() => navigate(`${path}${suffix}`, { replace: isRoom }), TRANSITION_MS / 2);
    const completionTimer = window.setTimeout(() => setTransitioning(false), TRANSITION_MS);
    transitionTimersRef.current = [navigationTimer, completionTimer];
  }, [cancelTransition, location.search, navigate, rooms]);

  const navigateRoom = useCallback((delta: number) => {
    if (!rooms.length) return;
    const current = rooms.findIndex((room) => `/${room.id}` === pathRef.current);
    const index = (Math.max(0, current) + delta + rooms.length) % rooms.length;
    const next = rooms[index];
    if (next) goTo(`/${next.id}`);
  }, [goTo, rooms]);

  const seekCycle = useCallback((nextProgress: number) => {
    const bounded = Math.max(0, Math.min(0.999, nextProgress));
    cycleStartedAt.current = Date.now() - bounded * roomDwellSeconds * 1000;
    showProgress(bounded);
  }, [roomDwellSeconds, showProgress]);

  const selectBeat = useCallback((index: number) => {
    if (!beatDurations.length || !currentRoom?.beats?.[index]) return;
    seekCycle(progressAtBeat(index, beatDurations));
    const next = new URLSearchParams(searchParams);
    next.set("beat", String(index));
    setSearchParams(next, { replace: true });
  }, [beatDurations, currentRoom, searchParams, seekCycle, setSearchParams]);

  const dispatch = useCallback((intent: BoardIntent) => {
    const now = Date.now();
    lastInputAt.current = now;
    setControlsVisible(true);
    if (intent === "reveal-controls") return;
    cycleStartedAt.current = now;
    setPausedUntil((value) => (value > now ? value : now + IDLE_RESUME_MS));
    switch (intent) {
      case "page-next":
      case "navigate-right":
        navigateRoom(1);
        setAcknowledgement("Next room");
        break;
      case "page-prev":
      case "navigate-left":
        navigateRoom(-1);
        setAcknowledgement("Previous room");
        break;
      case "navigate-beat-prev":
      case "navigate-beat-next":
        if (beatDurations.length) {
          const delta = intent === "navigate-beat-next" ? 1 : -1;
          selectBeat((beatPosition.index + delta + beatDurations.length) % beatDurations.length);
        }
        setAcknowledgement(intent === "navigate-beat-next" ? "Next section" : "Previous section");
        break;
      case "pause-cycle":
      case "select":
        setPausedUntil((value) => {
          const paused = value > now;
          setAcknowledgement(paused ? "Cycle resumed" : "Cycle paused");
          return paused ? 0 : Number.MAX_SAFE_INTEGER;
        });
        break;
      case "toggle-fullscreen":
        if (document.fullscreenElement) {
          void document.exitFullscreen().then(() => setAcknowledgement("Fullscreen off")).catch(() => setAcknowledgement("Fullscreen unavailable here"));
        } else {
          void document.documentElement.requestFullscreen().then(() => setAcknowledgement("Fullscreen on")).catch(() => setAcknowledgement("Fullscreen unavailable here"));
        }
        break;
      case "show-help":
      case "menu":
        setHelpVisible((visible) => !visible);
        setAcknowledgement("Help");
        break;
      case "back":
        setHelpVisible(false);
        setControlsVisible(false);
        setAcknowledgement("Controls hidden");
        break;
      case "inspect":
        setAcknowledgement("Reading inspected");
        break;
      default:
        setAcknowledgement(`${intent} acknowledged`);
    }
  }, [beatDurations.length, beatPosition.index, navigateRoom, selectBeat]);

  useEffect(() => {
    pathRef.current = location.pathname;
    restoredPathRef.current = null;
    restoringBeatRef.current = null;
    cycleStartedAt.current = Date.now();
    heldSinceRef.current = null;
    beatStallRef.current = { index: -1, at: Date.now() };
    setHeld(false);
    showProgress(0);
  }, [location.pathname, showProgress]);

  useEffect(() => cancelTransition, [cancelTransition]);

  useEffect(() => {
    if (!currentRoom || !beatDurations.length || restoredPathRef.current === location.pathname) return;
    const requested = Number.parseInt(searchParams.get("beat") ?? "0", 10);
    const index = Number.isFinite(requested) ? Math.max(0, Math.min(beatDurations.length - 1, requested)) : 0;
    restoringBeatRef.current = { path: location.pathname, index };
    seekCycle(progressAtBeat(index, beatDurations));
    restoredPathRef.current = location.pathname;
  }, [beatDurations, currentRoom, location.pathname, searchParams, seekCycle]);

  useEffect(() => {
    if (!beatDurations.length || restoredPathRef.current !== location.pathname) return;
    const pending = restoringBeatRef.current;
    if (pending?.path === location.pathname && pending.index !== beatPosition.index) return;
    if (pending?.path === location.pathname) restoringBeatRef.current = null;
    if (searchParams.get("beat") === String(beatPosition.index)) return;
    const next = new URLSearchParams(searchParams);
    next.set("beat", String(beatPosition.index));
    setSearchParams(next, { replace: true });
  }, [beatDurations.length, beatPosition.index, location.pathname, searchParams, setSearchParams]);

  useEffect(() => {
    if (location.pathname === "/settings") return;
    const requestedRoom = searchParams.get("room");
    if (requestedRoom && rooms.some((room) => room.id === requestedRoom) && location.pathname !== `/${requestedRoom}`) {
      navigate(`/${requestedRoom}${location.search}`, { replace: true });
    }
  }, [location.pathname, location.search, navigate, rooms, searchParams]);

  useEffect(() => {
    if (searchParams.get("fullscreen") !== "1") return;
    void document.documentElement.requestFullscreen().catch(() => setAcknowledgement("Fullscreen needs one interaction on this browser"));
  }, [searchParams]);

  useEffect(() => {
    let sentinel: WakeLockSentinel | null = null;
    if ("wakeLock" in navigator) {
      navigator.wakeLock.request("screen").then((lock) => { sentinel = lock; }).catch(() => undefined);
    }
    return () => { void sentinel?.release(); };
  }, []);

  useBoardKeyboard(location.pathname, rooms, dispatch, goTo);

  useGamepad(gamepadRef, action => {
    // Board-level controls handle the focused board surface. Actual buttons
    // retain normal selection; bumpers keep room switching available throughout.
    if (document.activeElement !== gamepadRef.current && action !== "page-next" && action !== "page-prev") return false;
    dispatch(action);
    return true;
  });
  useEffect(() => {
    if (document.activeElement === document.body) gamepadRef.current?.focus();
  }, []);

  useEffect(() => {
    const onTouchStart = (event: TouchEvent) => {
      const point = event.changedTouches[0];
      if (!point) return;
      touchStart.current = { x: point.clientX, y: point.clientY, at: Date.now() };
      dispatch("reveal-controls");
    };
    const onTouchEnd = (event: TouchEvent) => {
      const start = touchStart.current;
      const point = event.changedTouches[0];
      touchStart.current = null;
      if (!point || !start) return;
      const dx = point.clientX - start.x;
      const dy = point.clientY - start.y;
      const elapsed = Date.now() - start.at;
      if (elapsed >= 400 && Math.abs(dx) < 30 && Math.abs(dy) < 30) dispatch("pause-cycle");
      else if (Math.abs(dx) > 48 && Math.abs(dx) > Math.abs(dy)) dispatch(dx < 0 ? "page-next" : "page-prev");
    };
    window.addEventListener("touchstart", onTouchStart, { passive: true });
    window.addEventListener("touchend", onTouchEnd, { passive: true });
    return () => {
      window.removeEventListener("touchstart", onTouchStart);
      window.removeEventListener("touchend", onTouchEnd);
    };
  }, [dispatch]);

  useEffect(() => {
    // The clock is the animation: progress follows wall time every frame, so the
    // rail moves smoothly and a held beat waits at its segment's end.
    let frame = 0;
    let controlsCheckedAt = Number.NEGATIVE_INFINITY;
    const step = () => {
      // The cycle belongs to the kiosk rooms. On an operator surface (settings)
      // it neither advances progress nor navigates, so the page never self-evicts.
      if (pathRef.current === "/settings") {
        frame = window.requestAnimationFrame(step);
        return;
      }
      const now = Date.now();
      if (now - controlsCheckedAt >= 250) {
        controlsCheckedAt = now;
        setControlsVisible((visible) => visible && now - lastInputAt.current < CONTROLS_HIDE_MS);
      }
      const dwellMs = roomDwellSeconds * 1000;
      if (pausedUntil <= now && dwellMs > 0) {
        const tick = tickCycle({ now, startedAt: cycleStartedAt.current, dwellMs, progress: progressRef.current, durations: beatDurations, holding: holdsRef.current.size > 0, heldSince: heldSinceRef.current, maxHoldMs: MAX_HOLD_MS });
        heldSinceRef.current = tick.heldSince;
        if (tick.held) cycleStartedAt.current = now - tick.progress * dwellMs;
        showProgress(tick.progress);
        setHeld(tick.held);
        const displayIndex = beatDurations.length ? beatPositionAtProgress(tick.progress, beatDurations).index : 0;
        if (displayIndex !== beatStallRef.current.index) {
          beatStallRef.current = { index: displayIndex, at: now };
        } else {
          const beatMs = (beatDurations[displayIndex] ?? 0) * 1000;
          if (now - beatStallRef.current.at > beatMs + MAX_HOLD_MS + BEAT_STALL_MARGIN_MS) {
            // Whatever is holding it, a beat the viewer has been stuck on well
            // past its reading time is released and skipped, not left frozen.
            console.warn("[command-center] beat stalled; releasing and skipping", { index: displayIndex, holds: [...holdsRef.current] });
            holdsRef.current.clear();
            heldSinceRef.current = null;
            beatStallRef.current = { index: -1, at: now };
            if (beatDurations.length > 1) selectBeat((displayIndex + 1) % beatDurations.length);
            else if (rooms.length) navigateRoom(1);
          }
        }
        if (tick.navigate && rooms.length && !document.hidden) {
          cycleStartedAt.current = now;
          navigateRoom(1);
        }
      }
      frame = window.requestAnimationFrame(step);
    };
    frame = window.requestAnimationFrame(step);
    return () => window.cancelAnimationFrame(frame);
  }, [beatDurations, navigateRoom, pausedUntil, roomDwellSeconds, rooms.length, selectBeat, showProgress]);

  const paused = pausedUntil > Date.now();
  const beatIndex = beatPosition.index;
  // The controller value excludes the per-frame progress so the clock does not
  // re-render every room surface; progress has its own context.
  const value = useMemo<BoardControllerValue>(() => ({ rooms, board, samples, paused, controlsVisible, helpVisible, acknowledgement, cycleSeconds, beatIndex, beatDurations, transitioning, dispatch, setSamples, goTo, seekCycle, selectBeat, holdBeat, reportReadingSeconds }), [rooms, board, samples, paused, controlsVisible, helpVisible, acknowledgement, cycleSeconds, beatIndex, beatDurations, transitioning, dispatch, setSamples, goTo, seekCycle, selectBeat, holdBeat, reportReadingSeconds]);
  const progressValue = useMemo<BoardProgress>(() => ({ progress, beatProgress: beatPosition.progress, held }), [progress, beatPosition.progress, held]);

  return (
    <BoardContext.Provider value={value}>
      <BoardProgressContext.Provider value={progressValue}>
        <div ref={gamepadRef} tabIndex={0} aria-label="Board controls" data-board-root data-samples-mode={samples} data-paused={paused || undefined} onPointerDown={() => dispatch("reveal-controls")}>
          {children}
        </div>
      </BoardProgressContext.Provider>
    </BoardContext.Provider>
  );
}
