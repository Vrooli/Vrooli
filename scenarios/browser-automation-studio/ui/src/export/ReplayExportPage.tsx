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
  type CursorSpeedProfile,
  type CursorPathStyle,
} from "@/domains/exports/replay/ReplayPlayer";
import { MAX_BROWSER_SCALE, MIN_BROWSER_SCALE, resolveReplayStyleFromSpec } from "@/domains/replay-style";
// toNumber is imported and used by the extracted frameMapping.ts module
import type { ReplayAsset as ReplayMovieAsset, ReplaySpec as ReplayMovieSpec } from "@vrooli/generated-proto/browser-automation-studio/v1/exports/exports_pb";
import { logger } from "../utils/logger";
import "../index.css";

// Import extracted utilities
import type {
  ExportPreviewPayload,
  PresentationBounds,
} from "./types";
import {
  decodeExportPayload,
  ensureBasExportBootstrap,
  resolveBootstrapSpec,
} from "./bootstrap";
import {
  buildTimeline,
  computeTotalDuration,
} from "./timeline";
import {
  mapIntroCardSettings,
  mapOutroCardSettings,
  mapWatermarkSettings,
  toReplayFrame,
} from "./frameMapping";
import { defaultStatusMessage, normalizeStatus } from "./status";
import { useReplayExportBridge } from "./useReplayExportBridge";

// Re-export types needed by the global augmentation (imported from types.ts)
import "./types";

// Constants (only keeping ones specific to this component)
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

const ReplayExportPage = () => {
  const [movieSpec, setMovieSpec] = useState<ReplayMovieSpec | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [statusPayload, setStatusPayload] = useState<{
    status: string;
    message: string;
  } | null>(null);
  const [currentFrameIndex, setCurrentFrameIndex] = useState(0);
  const [currentProgress, setCurrentProgress] = useState(0);
  const [mode, setMode] = useState<"standalone" | "embedded" | "capture">(
    "standalone",
  );
  const [isAwaitingSpec, setIsAwaitingSpec] = useState(false);
  const presentationBoundsRef = useRef<HTMLDivElement | null>(null);
  const [presentationBounds, setPresentationBounds] = useState<PresentationBounds | null>(null);
  const fetchingRef = useRef(false);
  const pendingRetryRef = useRef<number | null>(null);
  const executionSourceRef = useRef<string | null>(null);

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
          preview.package.execution?.executionId ?? normalizedId;
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
    document.body.style.minHeight = "100%";
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
      if (!entry) return;
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
    const playbackDuration = movieSpec?.playback?.durationMs;
    if (playbackDuration && playbackDuration > 0) {
      return playbackDuration;
    }
    return computeTotalDuration(movieSpec?.summary, timeline);
  }, [movieSpec?.playback?.durationMs, movieSpec?.summary, timeline]);

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

  const { handleExposeController, controllerRef } = useReplayExportBridge({
    mode, statusPayload, loadError, movieSpec, replayFrames,
    effectiveCanvasWidth, effectiveCanvasHeight, currentFrameIndex, currentProgress,
    isAwaitingSpec, assetCount, totalDurationMs, timeline, executionSourceRef,
    fetchMovieSpec, clearPendingRetry, reportStatus, setMovieSpec, setLoadError,
    setStatusPayload, setIsAwaitingSpec,
  });

  const motion = movieSpec?.cursorMotion;
  const watermark = mapWatermarkSettings(movieSpec?.watermark);
  const introCard = mapIntroCardSettings(movieSpec?.introCard);
  const outroCard = mapOutroCardSettings(movieSpec?.outroCard);
  const styleFromSpec = resolveReplayStyleFromSpec(movieSpec);
  const cursorDefaultSpeedProfile = asCursorSpeedProfile(motion?.speedProfile);
  const cursorDefaultPathStyle = asCursorPathStyle(motion?.pathStyle);
  const browserFrameWidth = movieSpec?.presentation?.browserFrame?.width;
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
      <div className="flex min-h-full w-full items-center justify-center bg-flow-bg p-8 text-flow-text">
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
            "opacity-100": mode === "standalone" || controllerRef.current,
            "opacity-0": mode !== "standalone" && !controllerRef.current,
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
          exposeController={mode === "standalone" ? undefined : handleExposeController}
          presentationMode={mode === "standalone" ? "default" : "export"}
          presentationFit={mode === "capture" ? "none" : "contain"}
          presentationBounds={mode === "capture" ? undefined : presentationBounds ?? undefined}
          allowPointerEditing={mode === "standalone"}
          presentationDimensions={{
            width: effectiveCanvasWidth,
            height: effectiveCanvasHeight,
            deviceScaleFactor:
              movieSpec?.presentation?.deviceScaleFactor ?? undefined,
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
