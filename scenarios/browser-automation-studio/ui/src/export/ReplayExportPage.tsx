import {
  type CSSProperties,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import clsx from "clsx";
import ReplayPlayer, {
  type ReplayFrame,
  type ReplayPlayerController,
  type CursorSpeedProfile,
  type CursorPathStyle,
} from "@/domains/exports/replay/ReplayPlayer";
import { MAX_BROWSER_SCALE, MIN_BROWSER_SCALE, resolveReplayStyleFromSpec } from "@/domains/replay-style";
// toNumber is imported and used by the extracted frameMapping.ts module
import type {
  ReplayMovieAsset,
  ReplayMovieSpec,
} from "../types/export";
import { logger } from "../utils/logger";
import "../index.css";

// Import extracted utilities
import type {
  ExportMetadata,
  ExportPreviewPayload,
  FrameTimeline,
  FrameWaiter,
  PresentationBounds,
} from "./types";
import {
  decodeExportPayload,
  ensureBasExportBootstrap,
  resolveBootstrapSpec,
} from "./bootstrap";
import {
  buildTimeline,
  clampProgress,
  computeTotalDuration,
  findFrameForTime,
} from "./timeline";
import {
  mapIntroCardSettings,
  mapOutroCardSettings,
  mapWatermarkSettings,
  toReplayFrame,
} from "./frameMapping";
import { defaultStatusMessage, normalizeStatus } from "./status";

// Re-export types needed by the global augmentation (imported from types.ts)
import "./types";

// Constants (only keeping ones specific to this component)
const PROGRESS_EPSILON = 0.02;
const DEFAULT_TIMEOUT_MS = 6000;
const DEFAULT_BODY_BACKGROUND = "#020617";
const DEFAULT_CANVAS_WIDTH = 1280;
const DEFAULT_CANVAS_HEIGHT = 720;
const SPEC_POLL_INTERVAL_MS = 4000;

const CURSOR_SPEED_PROFILES: CursorSpeedProfile[] = [
  "instant",
  "linear",
  "easeIn",
  "easeOut",
  "easeInOut",
];
const CURSOR_PATH_STYLES: CursorPathStyle[] = [
  "linear",
  "parabolicUp",
  "parabolicDown",
  "cubic",
  "pseudorandom",
];

const asCursorSpeedProfile = (
  value: string | null | undefined,
): CursorSpeedProfile | undefined => {
  if (!value) {
    return undefined;
  }
  const lowered = value.trim().toLowerCase();
  const match = CURSOR_SPEED_PROFILES.find(
    (candidate) => candidate === lowered,
  );
  return match;
};

const asCursorPathStyle = (
  value: string | null | undefined,
): CursorPathStyle | undefined => {
  if (!value) {
    return undefined;
  }
  const trimmed = value.trim().toString();
  const lowered = trimmed.toLowerCase();
  if (lowered === "bezier") {
    return "cubic";
  }
  const match = CURSOR_PATH_STYLES.find((candidate) => candidate === lowered);
  return match;
};

// Initialize basExport bootstrap on module load
ensureBasExportBootstrap();

// Type alias for waiter (used locally in the component)
type Waiter = FrameWaiter;

const ReplayExportPage = () => {
  const [movieSpec, setMovieSpec] = useState<ReplayMovieSpec | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [statusPayload, setStatusPayload] = useState<{
    status: string;
    message: string;
  } | null>(null);
  const [currentFrameIndex, setCurrentFrameIndex] = useState(0);
  const [currentProgress, setCurrentProgress] = useState(0);
  const [controllerSignal, setControllerSignal] = useState(0);
  const [mode, setMode] = useState<"standalone" | "embedded" | "capture">(
    "standalone",
  );
  const [isAwaitingSpec, setIsAwaitingSpec] = useState(false);
  const presentationBoundsRef = useRef<HTMLDivElement | null>(null);
  const [presentationBounds, setPresentationBounds] = useState<PresentationBounds | null>(null);
  const fetchingRef = useRef(false);
  const pendingRetryRef = useRef<number | null>(null);
  const parentOriginRef = useRef<string | null>(null);
  const executionSourceRef = useRef<string | null>(null);
  const readySignalRef = useRef<string | null>(null);

  const controllerRef = useRef<ReplayPlayerController | null>(null);
  const waitersRef = useRef<Waiter[]>([]);
  const timelineRef = useRef<FrameTimeline[]>([]);
  const totalDurationRef = useRef(0);

  const clearPendingRetry = useCallback(() => {
    if (pendingRetryRef.current != null) {
      window.clearTimeout(pendingRetryRef.current);
      pendingRetryRef.current = null;
    }
  }, []);

  const reportStatus = useCallback(
    (status: string, message?: string | null) => {
      clearPendingRetry();
      const normalized = normalizeStatus(status) || "unavailable";
      const trimmedMessage =
        typeof message === "string" && message.trim().length > 0
          ? message.trim()
          : defaultStatusMessage(normalized);
      setStatusPayload({ status: normalized, message: trimmedMessage });
      setLoadError(trimmedMessage);
      setIsAwaitingSpec(false);
      setMovieSpec(null);
    },
    [clearPendingRetry],
  );

  const fetchMovieSpec = useCallback(
    async (executionId: string) => {
      const normalizedId = executionId.trim();
      if (!normalizedId || fetchingRef.current) {
        return;
      }
      fetchingRef.current = true;
      clearPendingRetry();
      setIsAwaitingSpec(true);
      setLoadError(null);
      setStatusPayload(null);
      try {
        const base =
          typeof window !== "undefined" && window.__BAS_EXPORT_API_BASE__
            ? String(window.__BAS_EXPORT_API_BASE__).trim()
            : "";
        const origin =
          base === "" ? window.location.origin : base.replace(/\/$/, "");
        const endpoint = `${origin}/api/v1/executions/${normalizedId}/export`;
        const response = await fetch(endpoint, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Accept: "application/json",
          },
          body: JSON.stringify({ format: "json" }),
        });
        if (!response.ok) {
          const text = await response.text();
          throw new Error(
            text || `Export specification request failed (${response.status})`,
          );
        }
        const preview = (await response.json()) as ExportPreviewPayload;
        const status = normalizeStatus(preview.status);
        const messageText =
          typeof preview.message === "string" ? preview.message.trim() : "";
        const hasFrames =
          preview.package && Array.isArray(preview.package.frames)
            ? preview.package.frames.length > 0
            : false;
        if (status === "pending" && !hasFrames) {
          executionSourceRef.current = normalizedId;
          const pendingMessage = messageText || defaultStatusMessage("pending");
          setStatusPayload({ status: "pending", message: pendingMessage });
          setLoadError(null);
          setMovieSpec(null);
          setIsAwaitingSpec(true);
          if (pendingRetryRef.current == null) {
            pendingRetryRef.current = window.setTimeout(() => {
              pendingRetryRef.current = null;
              void fetchMovieSpec(normalizedId);
            }, SPEC_POLL_INTERVAL_MS);
          }
          return;
        }
        if (status && status !== "ready" && !hasFrames) {
          executionSourceRef.current = normalizedId;
          reportStatus(status, messageText);
          return;
        }
        if (!preview?.package) {
          throw new Error("Replay movie spec missing from export response");
        }
        clearPendingRetry();
        setMovieSpec(preview.package);
        setStatusPayload(null);
        setLoadError(null);
        executionSourceRef.current =
          preview.package.execution?.execution_id ?? normalizedId;
        setIsAwaitingSpec(false);
      } catch (error) {
        const message =
          error instanceof Error
            ? error.message
            : "Failed to fetch replay spec";
        clearPendingRetry();
        reportStatus("error", message);
        logger.error(
          "Replay export fetch failed",
          { component: "ReplayExportPage", executionId: executionId.trim() },
          error,
        );
      } finally {
        fetchingRef.current = false;
      }
    },
    [clearPendingRetry, reportStatus],
  );

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const apiBaseParam = params.get("apiBase");
    if (apiBaseParam) {
      const normalized = apiBaseParam.trim();
      if (normalized) {
        window.__BAS_EXPORT_API_BASE__ = normalized;
      }
    }

    const bootstrap = resolveBootstrapSpec();
    if (!apiBaseParam && bootstrap.apiBase) {
      window.__BAS_EXPORT_API_BASE__ = bootstrap.apiBase;
    }

    const modeParam = params.get("mode");
    if (modeParam === "capture") {
      setMode("capture");
    } else if (modeParam === "embedded") {
      setMode("embedded");
    } else {
      setMode(window.self !== window.top ? "embedded" : "standalone");
    }

    const payloadParam = params.get("payload");
    const executionParam = params.get("executionId") ?? params.get("specId");

    if (payloadParam) {
      const decoded = decodeExportPayload(payloadParam);
      if (!decoded) {
        reportStatus("error", "Invalid replay payload");
        return;
      }
      setMovieSpec(decoded);
      setIsAwaitingSpec(false);
      setLoadError(null);
      setStatusPayload(null);
      if (typeof executionParam === "string" && executionParam.trim()) {
        executionSourceRef.current = executionParam.trim();
      }
      return;
    }

    if (bootstrap.spec) {
      setMovieSpec(bootstrap.spec);
      setIsAwaitingSpec(false);
      setLoadError(null);
      setStatusPayload(null);
      if (bootstrap.executionId) {
        executionSourceRef.current = bootstrap.executionId;
      }
      return;
    }

    if (executionParam) {
      void fetchMovieSpec(executionParam);
      return;
    }

    if (modeParam === "embedded" || window.self !== window.top) {
      setIsAwaitingSpec(true);
      setLoadError(null);
      setStatusPayload(null);
      return;
    }

    reportStatus("error", "Missing replay payload");
  }, [fetchMovieSpec, reportStatus]);

  useEffect(() => {
    if (mode !== "standalone") {
      return;
    }
    const originalStyles = {
      backgroundColor: document.body.style.backgroundColor,
      margin: document.body.style.margin,
      minHeight: document.body.style.minHeight,
      display: document.body.style.display,
      justifyContent: document.body.style.justifyContent,
      alignItems: document.body.style.alignItems,
      padding: document.body.style.padding,
    };
    document.body.style.backgroundColor = DEFAULT_BODY_BACKGROUND;
    document.body.style.margin = "0";
    document.body.style.minHeight = "100vh";
    document.body.style.display = "flex";
    document.body.style.justifyContent = "center";
    document.body.style.alignItems = "center";
    document.body.style.padding = "24px";
    return () => {
      document.body.style.backgroundColor = originalStyles.backgroundColor;
      document.body.style.margin = originalStyles.margin;
      document.body.style.minHeight = originalStyles.minHeight;
      document.body.style.display = originalStyles.display;
      document.body.style.justifyContent = originalStyles.justifyContent;
      document.body.style.alignItems = originalStyles.alignItems;
      document.body.style.padding = originalStyles.padding;
    };
  }, [mode]);

  useEffect(() => {
    if (mode === "capture") {
      return;
    }
    const node = presentationBoundsRef.current;
    if (!node || typeof ResizeObserver === "undefined") {
      return;
    }

    const observer = new ResizeObserver((entries) => {
      const entry = entries[0];
      const width = entry.contentRect.width;
      const height = entry.contentRect.height;
      if (width <= 0 || height <= 0) return;
      setPresentationBounds({ width, height });
    });

    observer.observe(node);
    return () => observer.disconnect();
  }, [mode]);

  const assetMap = useMemo(() => {
    const assets = movieSpec?.assets ?? [];
    const map = new Map<string, ReplayMovieAsset>();
    assets.forEach((asset) => {
      if (asset?.id) {
        map.set(asset.id, asset);
      }
    });
    return map;
  }, [movieSpec?.assets]);

  const replayFrames = useMemo(() => {
    if (!movieSpec?.frames) {
      return [] as ReplayFrame[];
    }
    return movieSpec.frames.map((frame, index) =>
      toReplayFrame(frame, index, assetMap),
    );
  }, [assetMap, movieSpec?.frames]);

  const assetCount = Array.isArray(movieSpec?.assets)
    ? movieSpec.assets.length
    : 0;

  const timeline = useMemo(
    () => buildTimeline(movieSpec?.frames),
    [movieSpec?.frames],
  );
  const totalDurationMs = useMemo(() => {
    const playbackDuration = movieSpec?.playback?.duration_ms;
    if (playbackDuration && playbackDuration > 0) {
      return playbackDuration;
    }
    return computeTotalDuration(movieSpec?.summary, timeline);
  }, [movieSpec?.playback?.duration_ms, movieSpec?.summary, timeline]);

  useEffect(() => {
    timelineRef.current = timeline;
    totalDurationRef.current = totalDurationMs;
  }, [timeline, totalDurationMs]);

  useEffect(() => {
    return () => {
      clearPendingRetry();
    };
  }, [clearPendingRetry]);

  const effectiveCanvasWidth = useMemo(() => {
    const canvasWidth = movieSpec?.presentation?.canvas?.width;
    if (canvasWidth && canvasWidth > 0) {
      return canvasWidth;
    }
    const viewportWidth = movieSpec?.presentation?.viewport?.width;
    if (viewportWidth && viewportWidth > 0) {
      return viewportWidth;
    }
    return DEFAULT_CANVAS_WIDTH;
  }, [
    movieSpec?.presentation?.canvas?.width,
    movieSpec?.presentation?.viewport?.width,
  ]);

  const effectiveCanvasHeight = useMemo(() => {
    const canvasHeight = movieSpec?.presentation?.canvas?.height;
    if (canvasHeight && canvasHeight > 0) {
      return canvasHeight;
    }
    const viewportHeight = movieSpec?.presentation?.viewport?.height;
    if (viewportHeight && viewportHeight > 0) {
      return viewportHeight;
    }
    return DEFAULT_CANVAS_HEIGHT;
  }, [
    movieSpec?.presentation?.canvas?.height,
    movieSpec?.presentation?.viewport?.height,
  ]);

  const registerWaiter = useCallback(
    (targetIndex: number, targetProgress: number) => {
      return new Promise<void>((resolve, reject) => {
        const clampedProgress = clampProgress(targetProgress);
        const cleanup = (waiter: Waiter) => {
          waitersRef.current = waitersRef.current.filter(
            (candidate) => candidate !== waiter,
          );
        };
        const timeoutId = window.setTimeout(() => {
          cleanup(waiter);
          reject(new Error("Timed out waiting for replay state"));
        }, DEFAULT_TIMEOUT_MS);
        const waiter: Waiter = {
          index: targetIndex,
          progress: clampedProgress,
          resolve: () => {
            window.clearTimeout(timeoutId);
            cleanup(waiter);
            resolve();
          },
          reject: (error: Error) => {
            window.clearTimeout(timeoutId);
            cleanup(waiter);
            reject(error);
          },
          timeoutId,
        };
        waitersRef.current.push(waiter);
      });
    },
    [],
  );

  const seekToTime = useCallback(
    async (ms: number) => {
      const controller = controllerRef.current;
      if (!controller) {
        throw new Error("Replay controller not ready");
      }
      const timelineData = timelineRef.current;
      if (!timelineData || timelineData.length === 0) {
        throw new Error("Replay timeline unavailable");
      }
      const total = totalDurationRef.current;
      const clampedMs = Math.min(Math.max(ms, 0), Math.max(total, 0));
      const target = findFrameForTime(clampedMs, timelineData);
      if (
        currentFrameIndex === target.index &&
        Math.abs(currentProgress - target.progress) <= PROGRESS_EPSILON
      ) {
        return;
      }
      const waiterPromise = registerWaiter(target.index, target.progress);
      controller.seek({ frameIndex: target.index, progress: target.progress });
      await waiterPromise;
    },
    [currentFrameIndex, currentProgress, registerWaiter],
  );

  const postToParent = useCallback(
    (message: Record<string, unknown>) => {
      if (mode === "capture") {
        return;
      }
      if (typeof window === "undefined" || window.parent === window) {
        return;
      }
      const targetOrigin = parentOriginRef.current ?? "*";
      try {
        window.parent.postMessage(message, targetOrigin);
      } catch (error) {
        logger.warn(
          "Failed to post message to parent",
          { component: "ReplayExportPage" },
          error,
        );
      }
    },
    [mode],
  );

  useEffect(() => {
    if (mode === "capture") {
      return;
    }
    if (statusPayload) {
      const frames = timelineRef.current.length;
      const assets = Array.isArray(movieSpec?.assets)
        ? movieSpec.assets.length
        : 0;
      const totalDuration = totalDurationRef.current;
      const specId =
        movieSpec?.execution?.execution_id ?? executionSourceRef.current;
      if (statusPayload.status === "pending") {
        postToParent({
          type: "bas:metrics",
          status: statusPayload.status,
          message: statusPayload.message,
          executionId: executionSourceRef.current,
          frames,
          assets,
          totalDurationMs: totalDuration,
          specId,
          canvasWidth: effectiveCanvasWidth,
          canvasHeight: effectiveCanvasHeight,
        });
        return;
      }
      postToParent({
        type: "bas:error",
        status: statusPayload.status,
        message: statusPayload.message,
        executionId: executionSourceRef.current,
        frames,
        assets,
        totalDurationMs: totalDuration,
        specId,
        canvasWidth: effectiveCanvasWidth,
        canvasHeight: effectiveCanvasHeight,
      });
      return;
    }
    if (!loadError) {
      postToParent({
        type: "bas:error-clear",
        executionId: executionSourceRef.current,
        specId:
          movieSpec?.execution?.execution_id ?? executionSourceRef.current,
      });
    }
  }, [
    statusPayload,
    mode,
    postToParent,
    loadError,
    movieSpec,
    effectiveCanvasWidth,
    effectiveCanvasHeight,
  ]);

  useEffect(() => {
    if (mode === "capture") {
      return;
    }
    const executionId = executionSourceRef.current;
    const frames = replayFrames.length || timelineRef.current.length;
    const assets = Array.isArray(movieSpec?.assets)
      ? movieSpec.assets.length
      : 0;
    const specId = movieSpec?.execution?.execution_id ?? executionId;
    postToParent({
      type: "bas:metrics",
      executionId,
      frames,
      assets,
      totalDurationMs: totalDurationRef.current,
      specId,
    });
  }, [mode, postToParent, replayFrames.length, movieSpec, totalDurationMs]);

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }
    const handleMessage = (event: MessageEvent) => {
      const payload = event.data;
      if (!payload || typeof payload !== "object") {
        return;
      }
      const { type } = payload as { type?: unknown };
      if (typeof type !== "string" || !type.startsWith("bas:")) {
        return;
      }
      parentOriginRef.current = event.origin;
      switch (type) {
        case "bas:spec:set": {
          const incoming = payload.spec as ReplayMovieSpec | undefined;
          if (incoming) {
            clearPendingRetry();
            setMovieSpec(incoming);
            setLoadError(null);
            setStatusPayload(null);
            setIsAwaitingSpec(false);
            if (typeof payload.apiBase === "string" && payload.apiBase.trim()) {
              window.__BAS_EXPORT_API_BASE__ = payload.apiBase.trim();
            }
            if (typeof payload.specId === "string" && payload.specId.trim()) {
              executionSourceRef.current = payload.specId.trim();
            } else if (typeof payload.executionId === "string") {
              executionSourceRef.current = payload.executionId.trim() || null;
            }
          }
          break;
        }
        case "bas:spec:set-encoded": {
          if (typeof payload.payload === "string") {
            const decoded = decodeExportPayload(payload.payload);
            if (decoded) {
              clearPendingRetry();
              setMovieSpec(decoded);
              setLoadError(null);
              setStatusPayload(null);
              setIsAwaitingSpec(false);
              if (typeof payload.specId === "string" && payload.specId.trim()) {
                executionSourceRef.current = payload.specId.trim();
              } else if (typeof payload.executionId === "string") {
                executionSourceRef.current = payload.executionId.trim() || null;
              }
            } else {
              reportStatus("error", "Invalid replay payload");
            }
          }
          break;
        }
        case "bas:spec:fetch": {
          if (typeof payload.executionId === "string") {
            void fetchMovieSpec(payload.executionId);
          }
          break;
        }
        case "bas:control:seek": {
          if (
            typeof payload.timeMs === "number" &&
            Number.isFinite(payload.timeMs)
          ) {
            void seekToTime(payload.timeMs);
          }
          break;
        }
        case "bas:control:play": {
          controllerRef.current?.play();
          break;
        }
        case "bas:control:pause": {
          controllerRef.current?.pause();
          break;
        }
        case "bas:control:frame": {
          const frameIndex = payload.frameIndex;
          const progress = payload.progress;
          if (
            typeof frameIndex === "number" &&
            Number.isFinite(frameIndex) &&
            controllerRef.current
          ) {
            const waiterPromise = registerWaiter(
              frameIndex,
              Number(progress) || 0,
            );
            controllerRef.current.seek({
              frameIndex,
              progress: Number.isFinite(progress)
                ? Number(progress)
                : undefined,
            });
            void waiterPromise.catch((error) => {
              logger.warn(
                "Failed to satisfy frame seek request",
                { component: "ReplayExportPage", frameIndex },
                error,
              );
            });
          }
          break;
        }
        default:
          break;
      }
    };
    window.addEventListener("message", handleMessage);
    return () => {
      window.removeEventListener("message", handleMessage);
    };
  }, [
    clearPendingRetry,
    fetchMovieSpec,
    registerWaiter,
    reportStatus,
    seekToTime,
  ]);

  useEffect(() => {
    if (waitersRef.current.length === 0) {
      return;
    }
    waitersRef.current = waitersRef.current.filter((waiter) => {
      const matchesIndex = waiter.index === currentFrameIndex;
      const matchesProgress =
        Math.abs(waiter.progress - currentProgress) <= PROGRESS_EPSILON ||
        (waiter.progress >= 0.98 && currentProgress >= 0.98);
      if (matchesIndex && matchesProgress) {
        waiter.resolve();
        return false;
      }
      return true;
    });
  }, [currentFrameIndex, currentProgress]);

  const handleExposeController = useCallback(
    (controller: ReplayPlayerController | null) => {
      controllerRef.current = controller;
      setControllerSignal((value) => value + 1);
    },
    [],
  );

  useEffect(() => {
    window.basExport = {
      ready: Boolean(
        !loadError && controllerRef.current && replayFrames.length > 0,
      ),
      error: loadError,
      seekTo: seekToTime,
      play: () => {
        controllerRef.current?.play();
      },
      pause: () => {
        controllerRef.current?.pause();
      },
      getViewportRect: () => {
        const controller = controllerRef.current;
        const layout = controller?.getLayout?.();
        const element =
          controller?.getPresentationElement?.() ??
          controller?.getViewportElement();
        if (layout) {
          return {
            x: Math.round(layout.viewportRect.x),
            y: Math.round(layout.viewportRect.y),
            width: Math.round(layout.viewportRect.width),
            height: Math.round(layout.viewportRect.height),
          };
        }
        if (!element) {
          return { x: 0, y: 0, width: 0, height: 0 };
        }
        const rect = element.getBoundingClientRect();
        const scrollX = window.scrollX ?? window.pageXOffset ?? 0;
        const scrollY = window.scrollY ?? window.pageYOffset ?? 0;
        return {
          x: rect.left + scrollX,
          y: rect.top + scrollY,
          width: rect.width,
          height: rect.height,
        };
      },
      getMetadata: () => {
        const controller = controllerRef.current;
        const layout = controller?.getLayout?.();
        const presentationElement = controller?.getPresentationElement?.();
        const viewportElement = controller?.getViewportElement();
        const presentationRect = presentationElement
          ? presentationElement.getBoundingClientRect()
          : null;
        const viewportRect = viewportElement
          ? viewportElement.getBoundingClientRect()
          : null;
        const deviceScale = movieSpec?.presentation?.device_scale_factor;
        const assetCount = Array.isArray(movieSpec?.assets)
          ? movieSpec?.assets.length
          : 0;
        const specId =
          movieSpec?.execution?.execution_id ?? executionSourceRef.current;
        const viewportWidth = layout
          ? Math.max(1, Math.round(layout.viewportRect.width))
          : viewportRect
            ? Math.max(1, Math.round(viewportRect.width))
            : effectiveCanvasWidth;
        const viewportHeight = layout
          ? Math.max(1, Math.round(layout.viewportRect.height))
          : viewportRect
            ? Math.max(1, Math.round(viewportRect.height))
            : effectiveCanvasHeight;
        const canvasWidth = layout
          ? Math.max(1, Math.round(layout.display.width))
          : presentationRect
            ? Math.max(1, Math.round(presentationRect.width))
            : effectiveCanvasWidth;
        const canvasHeight = layout
          ? Math.max(1, Math.round(layout.display.height))
          : presentationRect
            ? Math.max(1, Math.round(presentationRect.height))
            : effectiveCanvasHeight;
        const browserFrameRadius =
          movieSpec?.presentation?.browser_frame?.radius ?? undefined;
        const browserFrame = (() => {
          if (layout) {
            return {
              x: Math.round(layout.viewportRect.x),
              y: Math.round(layout.viewportRect.y),
              width: viewportWidth,
              height: viewportHeight,
              radius: browserFrameRadius,
            } as ExportMetadata["browserFrame"];
          }
          if (presentationRect && viewportRect) {
            return {
              x: Math.round(viewportRect.left - presentationRect.left),
              y: Math.round(viewportRect.top - presentationRect.top),
              width: viewportWidth,
              height: viewportHeight,
              radius: browserFrameRadius,
            } as ExportMetadata["browserFrame"];
          }
          return {
            x: 0,
            y: 0,
            width: viewportWidth,
            height: viewportHeight,
            radius: browserFrameRadius,
          } as ExportMetadata["browserFrame"];
        })();
        return {
          totalDurationMs: totalDurationRef.current,
          frameCount: timelineRef.current.length,
          timeline: [...timelineRef.current],
          width: viewportWidth,
          height: viewportHeight,
          canvasWidth,
          canvasHeight,
          browserFrame,
          assetCount,
          specId,
          deviceScaleFactor: deviceScale ?? 1,
        };
      },
      getCurrentState: () => ({
        frameIndex: currentFrameIndex,
        progress: currentProgress,
      }),
    };
    return () => {
      ensureBasExportBootstrap();
    };
  }, [
    controllerSignal,
    currentFrameIndex,
    currentProgress,
    loadError,
    replayFrames.length,
    seekToTime,
  ]);

  useEffect(() => {
    if (mode === "capture" || loadError || !controllerRef.current) {
      readySignalRef.current = null;
      return;
    }

    const pending = isAwaitingSpec || replayFrames.length === 0;
    const signalToken = `${pending ? "pending" : "ready"}:${replayFrames.length}`;

    if (readySignalRef.current === signalToken) {
      return;
    }

    readySignalRef.current = signalToken;

    postToParent({
      type: "bas:ready",
      frames: replayFrames.length,
      totalDurationMs: totalDurationRef.current,
      executionId: executionSourceRef.current,
      assets: assetCount,
      specId: movieSpec?.execution?.execution_id ?? executionSourceRef.current,
      pending,
      canvasWidth: effectiveCanvasWidth,
      canvasHeight: effectiveCanvasHeight,
    });
  }, [
    controllerSignal,
    isAwaitingSpec,
    loadError,
    mode,
    movieSpec?.execution?.execution_id,
    assetCount,
    effectiveCanvasHeight,
    effectiveCanvasWidth,
    postToParent,
    replayFrames.length,
  ]);

  useEffect(() => {
    if (mode === "capture") {
      return;
    }
    if (typeof window === "undefined" || window.parent === window) {
      return;
    }
    postToParent({
      type: "bas:state",
      frameIndex: currentFrameIndex,
      progress: currentProgress,
      frameId: replayFrames[currentFrameIndex]?.id ?? null,
      executionId: executionSourceRef.current,
    });
  }, [currentFrameIndex, currentProgress, mode, postToParent, replayFrames]);

  const motion = movieSpec?.cursor_motion;
  const watermark = mapWatermarkSettings(movieSpec?.watermark);
  const introCard = mapIntroCardSettings(movieSpec?.intro_card);
  const outroCard = mapOutroCardSettings(movieSpec?.outro_card);
  const styleFromSpec = resolveReplayStyleFromSpec(movieSpec);
  const cursorDefaultSpeedProfile = asCursorSpeedProfile(motion?.speed_profile);
  const cursorDefaultPathStyle = asCursorPathStyle(motion?.path_style);
  const browserFrameWidth = movieSpec?.presentation?.browser_frame?.width;
  const browserScale = browserFrameWidth && effectiveCanvasWidth > 0
    ? Math.min(MAX_BROWSER_SCALE, Math.max(MIN_BROWSER_SCALE, browserFrameWidth / effectiveCanvasWidth))
    : 1;
  const resolvedStyle = {
    ...styleFromSpec,
    browserScale,
  };

  const handleFrameChange = useCallback(
    (_frame: ReplayFrame, index: number) => {
      setCurrentFrameIndex(index);
    },
    [],
  );

  const handleProgressChange = useCallback(
    (index: number, progress: number) => {
      setCurrentFrameIndex(index);
      setCurrentProgress(progress);
    },
    [],
  );

  if (loadError) {
    return (
      <div className="flex min-h-screen w-full items-center justify-center bg-flow-bg p-8 text-flow-text">
        <div className="max-w-md rounded-2xl border border-flow-border bg-flow-node/80 p-8 text-center shadow-[0_20px_60px_rgba(0,0,0,0.35)]">
          <h1 className="text-lg font-semibold text-flow-text">Replay export unavailable</h1>
          <p className="mt-3 text-sm text-flow-text-secondary">{loadError}</p>
        </div>
      </div>
    );
  }

  const showPlaceholder = !movieSpec || replayFrames.length === 0;
  const placeholderMessage = (() => {
    if (statusPayload?.status === "pending") {
      return (
        statusPayload.message ||
        "Replay export pending – timeline frames not captured yet"
      );
    }
    if (isAwaitingSpec) {
      return "Waiting for replay spec…";
    }
    return "Preparing replay…";
  })();

  const containerClassName = clsx(
    "w-full",
    mode === "standalone" ? "mx-auto" : "h-full",
  );

  const containerStyle: CSSProperties = mode === "capture"
    ? {
        width: `${effectiveCanvasWidth}px`,
        height: `${effectiveCanvasHeight}px`,
      }
    : mode === "embedded"
      ? {
          width: "100%",
          maxWidth: "100%",
        }
      : {};

  return (
    <div
      className={containerClassName}
      style={{
        ...containerStyle,
        backgroundColor:
          mode === "standalone" ? undefined : DEFAULT_BODY_BACKGROUND,
      }}
    >
      <div
        ref={presentationBoundsRef}
        className={clsx(
          "relative flex items-center justify-center transition-all duration-500",
          mode === "standalone" ? "w-full" : "h-full w-full",
          {
            "opacity-100": mode !== "standalone" || controllerRef.current,
            "opacity-0": mode === "standalone" && !controllerRef.current,
          },
        )}
      >
        <ReplayPlayer
          frames={replayFrames}
          autoPlay={false}
          loop={false}
          replayStyle={{
            presentation: resolvedStyle.presentation,
            chromeTheme: resolvedStyle.chromeTheme,
            deviceFrameTheme: resolvedStyle.deviceFrameTheme,
            background: resolvedStyle.background,
            cursorTheme: resolvedStyle.cursorTheme,
            cursorInitialPosition: resolvedStyle.cursorInitialPosition,
            cursorScale: resolvedStyle.cursorScale,
            cursorClickAnimation: resolvedStyle.cursorClickAnimation,
            browserScale: resolvedStyle.browserScale,
          }}
          cursorDefaultSpeedProfile={cursorDefaultSpeedProfile}
          cursorDefaultPathStyle={cursorDefaultPathStyle}
          watermark={watermark ?? undefined}
          introCard={introCard ?? undefined}
          outroCard={outroCard ?? undefined}
          onFrameChange={handleFrameChange}
          onFrameProgressChange={handleProgressChange}
          exposeController={handleExposeController}
          presentationMode={mode === "standalone" ? "default" : "export"}
          presentationFit={mode === "capture" ? "none" : "contain"}
          presentationBounds={mode === "capture" ? undefined : presentationBounds ?? undefined}
          allowPointerEditing={mode === "standalone"}
          presentationDimensions={{
            width: effectiveCanvasWidth,
            height: effectiveCanvasHeight,
            deviceScaleFactor:
              movieSpec?.presentation?.device_scale_factor ?? undefined,
          }}
        />

        {showPlaceholder && mode !== "capture" && (
          <div className="pointer-events-none absolute inset-0 flex items-center justify-center bg-black/40">
            <span className="text-xs uppercase tracking-[0.3em] text-flow-text-muted">
              {placeholderMessage}
            </span>
          </div>
        )}
      </div>
    </div>
  );
};

export default ReplayExportPage;
