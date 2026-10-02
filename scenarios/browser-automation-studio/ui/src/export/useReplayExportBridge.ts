import { useCallback, useEffect, useRef, useState, type Dispatch, type MutableRefObject, type SetStateAction } from "react";
import type { ReplayFrame, ReplayPlayerController } from "@/domains/exports/replay/ReplayPlayer";
import { ReplaySpecSchema } from "@vrooli/generated-proto/browser-automation-studio/v1/exports/exports_pb";
import type { ReplaySpec } from "@vrooli/generated-proto/browser-automation-studio/v1/exports/exports_pb";
import { parseProtoStrict } from "@/utils/proto";
import { logger } from "../utils/logger";
import { decodeExportPayload, ensureBasExportBootstrap } from "./bootstrap";
import { clampProgress, findFrameForTime } from "./timeline";
import type { ExportMetadata, FrameTimeline, FrameWaiter } from "./types";
import "./types";

const PROGRESS_EPSILON = 0.02;
const DEFAULT_TIMEOUT_MS = 6000;

interface ReplayExportBridgeOptions {
  mode: "standalone" | "embedded" | "capture";
  statusPayload: { status: string; message: string } | null;
  loadError: string | null;
  movieSpec: ReplaySpec | null;
  replayFrames: ReplayFrame[];
  effectiveCanvasWidth: number;
  effectiveCanvasHeight: number;
  currentFrameIndex: number;
  currentProgress: number;
  isAwaitingSpec: boolean;
  assetCount: number;
  totalDurationMs: number;
  timeline: FrameTimeline[];
  executionSourceRef: MutableRefObject<string | null>;
  fetchMovieSpec: (executionId: string) => Promise<void>;
  clearPendingRetry: () => void;
  reportStatus: (status: string, message?: string | null) => void;
  setMovieSpec: Dispatch<SetStateAction<ReplaySpec | null>>;
  setLoadError: Dispatch<SetStateAction<string | null>>;
  setStatusPayload: Dispatch<SetStateAction<{ status: string; message: string } | null>>;
  setIsAwaitingSpec: Dispatch<SetStateAction<boolean>>;
}

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === "object" && value !== null && !Array.isArray(value);

const readFiniteNumber = (value: unknown): number | null => {
  if (typeof value === "number" && Number.isFinite(value)) return value;
  if (typeof value === "string" && value.trim() !== "") {
    const parsed = Number(value);
    if (Number.isFinite(parsed)) return parsed;
  }
  return null;
};

const parseReplaySpec = (value: unknown): ReplaySpec | null => {
  try {
    return parseProtoStrict<ReplaySpec>(ReplaySpecSchema, value);
  } catch {
    return null;
  }
};

export function useReplayExportBridge({
  mode, statusPayload, loadError, movieSpec, replayFrames,
  effectiveCanvasWidth, effectiveCanvasHeight, currentFrameIndex, currentProgress,
  isAwaitingSpec, assetCount, totalDurationMs, timeline, executionSourceRef,
  fetchMovieSpec, clearPendingRetry, reportStatus, setMovieSpec, setLoadError,
  setStatusPayload, setIsAwaitingSpec,
}: ReplayExportBridgeOptions) {
  const [controllerSignal, setControllerSignal] = useState(0);
  const parentOriginRef = useRef<string | null>(null);
  const readySignalRef = useRef<string | null>(null);
  const controllerRef = useRef<ReplayPlayerController | null>(null);
  type Waiter = FrameWaiter;
  const waitersRef = useRef<Waiter[]>([]);
  const timelineRef = useRef<FrameTimeline[]>([]);
  const totalDurationRef = useRef(0);

  useEffect(() => {
    timelineRef.current = timeline;
    totalDurationRef.current = totalDurationMs;
  }, [timeline, totalDurationMs]);

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
        movieSpec?.execution?.executionId ?? executionSourceRef.current;
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
          movieSpec?.execution?.executionId ?? executionSourceRef.current,
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
    const specId = movieSpec?.execution?.executionId ?? executionId;
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
      const payload = event.data as unknown;
      if (!isRecord(payload)) {
        return;
      }
      const type = payload.type;
      if (typeof type !== "string" || !type.startsWith("bas:")) {
        return;
      }
      parentOriginRef.current = event.origin;
      switch (type) {
        case "bas:spec:set": {
          const incoming = payload.spec;
          const parsedSpec = parseReplaySpec(incoming);
          if (parsedSpec) {
            clearPendingRetry();
            setMovieSpec(parsedSpec);
            setLoadError(null);
            setStatusPayload(null);
            setIsAwaitingSpec(false);
            if (typeof payload.apiBase === "string") {
              const apiBase = payload.apiBase.trim();
              if (apiBase) {
                window.__BAS_EXPORT_API_BASE__ = apiBase;
              }
            }
            if (typeof payload.specId === "string") {
              const specId = payload.specId.trim();
              if (specId) {
                executionSourceRef.current = specId;
              }
            } else if (typeof payload.executionId === "string") {
              executionSourceRef.current = payload.executionId.trim() || null;
            }
          } else if (isRecord(incoming)) {
            reportStatus("error", "Invalid replay spec");
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
              if (typeof payload.specId === "string") {
                const specId = payload.specId.trim();
                if (specId) {
                  executionSourceRef.current = specId;
                }
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
          const timeMs = readFiniteNumber(payload.timeMs);
          if (timeMs != null) {
            void seekToTime(timeMs);
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
          const frameIndex = readFiniteNumber(payload.frameIndex);
          const progress = readFiniteNumber(payload.progress);
          if (frameIndex != null && controllerRef.current) {
            const waiterPromise = registerWaiter(frameIndex, progress ?? 0);
            controllerRef.current.seek({
              frameIndex,
              progress: progress ?? undefined,
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
        const deviceScale = movieSpec?.presentation?.deviceScaleFactor;
        const assetCount = Array.isArray(movieSpec?.assets)
          ? movieSpec?.assets.length
          : 0;
        const specId =
          movieSpec?.execution?.executionId ?? executionSourceRef.current;
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
          movieSpec?.presentation?.browserFrame?.radius ?? undefined;
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
    effectiveCanvasHeight,
    effectiveCanvasWidth,
    loadError,
    movieSpec?.assets,
    movieSpec?.execution?.executionId,
    movieSpec?.presentation?.browserFrame?.radius,
    movieSpec?.presentation?.deviceScaleFactor,
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
      specId: movieSpec?.execution?.executionId ?? executionSourceRef.current,
      pending,
      canvasWidth: effectiveCanvasWidth,
      canvasHeight: effectiveCanvasHeight,
    });
  }, [
    controllerSignal,
    isAwaitingSpec,
    loadError,
    mode,
    movieSpec?.execution?.executionId,
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


  return { handleExposeController, controllerRef };
}
