import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { useLocation, useNavigate, useSearchParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { GamepadInputManager } from "@vrooli/iframe-bridge/spatial";
import { emitShortcutIntent } from "@vrooli/iframe-bridge";
import { fetchBoard } from "../lib/api";
import { BoardContext, parseSamples, type BoardControllerValue, type BoardIntent, type SamplesMode } from "../lib/boardContext";


const IDLE_RESUME_MS = 20_000;
const CONTROLS_HIDE_MS = 4_000;
const TRANSITION_MS = 900;

const KEY_INTENTS: Record<string, BoardIntent> = {
  arrowright: "navigate-right",
  arrowleft: "navigate-left",
  arrowup: "navigate-up",
  arrowdown: "navigate-down",
  " ": "pause-cycle",
  f: "toggle-fullscreen",
  "?": "show-help",
  h: "show-help",
  enter: "inspect",
  escape: "back",
};

export function BoardController({ children }: { children: ReactNode }) {
  const navigate = useNavigate();
  const location = useLocation();
  const [searchParams, setSearchParams] = useSearchParams();
  const { data: board } = useQuery({ queryKey: ["board-shape"], queryFn: fetchBoard, staleTime: 30_000 });
  const rooms = useMemo(() => board?.rooms ?? [], [board]);
  const [controlsVisible, setControlsVisible] = useState(false);
  const [helpVisible, setHelpVisible] = useState(false);
  const [acknowledgement, setAcknowledgement] = useState("Ready");
  const [pausedUntil, setPausedUntil] = useState(0);
  const [progress, setProgress] = useState(0);
  const [transitioning, setTransitioning] = useState(false);
  const cycleSeconds = Math.max(5, Number(searchParams.get("cycle") ?? 60) || 60);
  const samples = parseSamples(searchParams.get("samples"));
  const touchStart = useRef<{ x: number; y: number; at: number } | null>(null);
  const lastInputAt = useRef(Date.now());
  const cycleStartedAt = useRef(Date.now());
  const pathRef = useRef(location.pathname);

  const setSamples = useCallback((mode: SamplesMode) => {
    const next = new URLSearchParams(searchParams);
    next.set("samples", mode);
    setSearchParams(next, { replace: true });
    setAcknowledgement(`Samples: ${mode}`);
  }, [searchParams, setSearchParams]);

  const goTo = useCallback((path: string) => {
    if (path === pathRef.current) return;
    setTransitioning(true);
    window.setTimeout(() => navigate(`${path}${location.search}`), TRANSITION_MS / 2);
    window.setTimeout(() => setTransitioning(false), TRANSITION_MS);
  }, [location.search, navigate]);

  const navigateRoom = useCallback((delta: number) => {
    if (!rooms.length) return;
    const current = rooms.findIndex((room) => `/${room.id}` === pathRef.current);
    const index = (Math.max(0, current) + delta + rooms.length) % rooms.length;
    const next = rooms[index];
    if (next) goTo(`/${next.id}`);
  }, [goTo, rooms]);

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
  }, [navigateRoom]);

  useEffect(() => {
    pathRef.current = location.pathname;
  }, [location.pathname]);

  useEffect(() => {
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

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      const key = event.key.toLowerCase();
      if (/^[1-9]$/.test(key)) {
        const room = rooms[Number(key) - 1];
        if (room) {
          dispatch("reveal-controls");
          goTo(`/${room.id}`);
        }
        return;
      }
      const intent = KEY_INTENTS[key];
      if (!intent) {
        dispatch("reveal-controls");
        return;
      }
      event.preventDefault();
      emitShortcutIntent({ action: `command-center.${intent}`, outcome: "handled", chord: event.key, source: "keyboard" });
      dispatch(intent);
    };
    const onPointerMove = () => dispatch("reveal-controls");
    window.addEventListener("keydown", onKey);
    window.addEventListener("pointermove", onPointerMove, { passive: true });
    return () => {
      window.removeEventListener("keydown", onKey);
      window.removeEventListener("pointermove", onPointerMove);
    };
  }, [dispatch, goTo, rooms]);

  useEffect(() => {
    const manager = new GamepadInputManager({ onAction: dispatch });
    manager.start();
    return () => manager.dispose();
  }, [dispatch]);

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
    const timer = window.setInterval(() => {
      const now = Date.now();
      setControlsVisible((visible) => visible && now - lastInputAt.current < CONTROLS_HIDE_MS);
      const paused = pausedUntil > now;
      setProgress(paused ? Math.min(1, (now - cycleStartedAt.current) / (cycleSeconds * 1000)) : Math.min(1, (now - cycleStartedAt.current) / (cycleSeconds * 1000)));
      if (!paused && now - cycleStartedAt.current >= cycleSeconds * 1000 && rooms.length && !document.hidden) {
        cycleStartedAt.current = now;
        navigateRoom(1);
      }
    }, 250);
    return () => window.clearInterval(timer);
  }, [cycleSeconds, navigateRoom, pausedUntil, rooms.length]);

  const paused = pausedUntil > Date.now();
  const value = useMemo<BoardControllerValue>(() => ({ rooms, board, samples, paused, controlsVisible, helpVisible, acknowledgement, progress, cycleSeconds, transitioning, dispatch, setSamples, goTo }), [rooms, board, samples, paused, controlsVisible, helpVisible, acknowledgement, progress, cycleSeconds, transitioning, dispatch, setSamples, goTo]);

  return (
    <BoardContext.Provider value={value}>
      <div data-board-root data-samples-mode={samples} data-paused={paused || undefined} onPointerDown={() => dispatch("reveal-controls")}>
        {children}
      </div>
    </BoardContext.Provider>
  );
}
