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
} from "../components/ReplayPlayer";
import {
  mapAssertion,
  mapRegions,
  mapRetryHistory,
  mapTrail,
  resolveUrl,
  toBoundingBox,
  toNumber,
  toPoint,
} from "../utils/executionTypeMappers";
import type {
  ReplayMovieAsset,
  ReplayMovieFrame,
  ReplayMovieSpec,
  ReplayMovieSummary,
} from "../types/export";
import { logger } from "../utils/logger";
import "../index.css";

type FrameTimeline = {
  index: number;
  startMs: number;
  durationMs: number;
};

type Waiter = {
  index: number;
  progress: number;
  resolve: () => void;
  reject: (error: Error) => void;
  timeoutId: number;
};

type ExportMetadata = {
  totalDurationMs: number;
  frameCount: number;
  timeline: FrameTimeline[];
  width: number;
  height: number;
  canvasWidth: number;
  canvasHeight: number;
  browserFrame: {
    x: number;
    y: number;
    width: number;
    height: number;
    radius?: number;
  };
  deviceScaleFactor?: number;
  assetCount: number;
  specId?: string | null;
};

type ExportPreviewPayload = {
  status?: string | null;
  message?: string | null;
  package?: ReplayMovieSpec | null;
  execution_id?: string | null;
};

type BootstrapPayload = {
  payloadJson?: string | null;
  payloadBase64?: string | null;
  apiBase?: string | null;
  executionId?: string | null;
};

declare global {
  interface Window {
    basExport?: {
      ready: boolean;
      error?: string | null;
      seekTo: (ms: number) => Promise<void>;
      play: () => void;
      pause: () => void;
      getViewportRect: () => {
        x: number;
        y: number;
        width: number;
        height: number;
      };
      getMetadata: () => ExportMetadata;
      getCurrentState: () => { frameIndex: number; progress: number };
    };
    __BAS_EXPORT_API_BASE__?: string;
    __BAS_EXPORT_BOOTSTRAP__?: BootstrapPayload | null;
  }
}

const DEFAULT_FRAME_DURATION_MS = 1600;
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

const ensureBasExportBootstrap = () => {
  if (typeof window === "undefined") {
    return;
  }
  if (!window.basExport) {
    window.basExport = {
      ready: false,
      error: "initializing",
      async seekTo() {
        throw new Error("Replay export controller not ready");
      },
      play() {
        throw new Error("Replay export controller not ready");
      },
      pause() {
        throw new Error("Replay export controller not ready");
      },
      getViewportRect() {
        return { x: 0, y: 0, width: 0, height: 0 };
      },
      getMetadata() {
        return {
          totalDurationMs: 0,
          frameCount: 0,
          timeline: [],
          width: 0,
          height: 0,
          canvasWidth: 0,
          canvasHeight: 0,
          browserFrame: {
            x: 0,
            y: 0,
            width: 0,
            height: 0,
            radius: 0,
          },
          assetCount: 0,
          specId: null,
          deviceScaleFactor: 1,
        };
      },
      getCurrentState() {
        return { frameIndex: 0, progress: 0 };
      },
    };
  }
};

ensureBasExportBootstrap();

const addPadding = (base64: string): string => {
  const padding = (4 - (base64.length % 4)) % 4;
  if (padding === 0) {
    return base64;
  }
  return `${base64}${"=".repeat(padding)}`;
};

const decodeExportPayload = (encoded: string): ReplayMovieSpec | null => {
  if (!encoded) {
    return null;
  }
  try {
    const normalized = addPadding(
      encoded.replace(/-/g, "+").replace(/_/g, "/"),
    );
    if (typeof window === "undefined" || typeof window.atob !== "function") {
      return null;
    }
    const json = window.atob(normalized);
    return JSON.parse(json) as ReplayMovieSpec;
  } catch (error) {
    logger.error(
      "Failed to decode replay export payload",
      { component: "ReplayExportPage" },
      error,
    );
    return null;
  }
};

const decodeJsonSpec = (
  raw: string | null | undefined,
): ReplayMovieSpec | null => {
  if (!raw) {
    return null;
  }
  try {
    return JSON.parse(raw) as ReplayMovieSpec;
  } catch (error) {
    logger.error(
      "Failed to parse replay export bootstrap payload",
      { component: "ReplayExportPage" },
      error,
    );
    return null;
  }
};

const getBootstrapPayload = (): BootstrapPayload | null => {
  if (typeof window === "undefined") {
    return null;
  }
  const candidate = window.__BAS_EXPORT_BOOTSTRAP__;
  if (!candidate || typeof candidate !== "object") {
    return null;
  }
  const payload = candidate as BootstrapPayload;
  return {
    payloadJson:
      typeof payload.payloadJson === "string" ? payload.payloadJson : undefined,
    payloadBase64:
      typeof payload.payloadBase64 === "string"
        ? payload.payloadBase64
        : undefined,
    apiBase: typeof payload.apiBase === "string" ? payload.apiBase : undefined,
    executionId:
      typeof payload.executionId === "string" ? payload.executionId : undefined,
  };
};

const resolveBootstrapSpec = () => {
  const payload = getBootstrapPayload();
  if (!payload) {
    return {
      spec: null as ReplayMovieSpec | null,
      executionId: null as string | null,
      apiBase: null as string | null,
    };
  }
  let spec = decodeJsonSpec(payload.payloadJson);
  if (!spec && payload.payloadBase64) {
    spec = decodeExportPayload(payload.payloadBase64);
  }
  const executionId = payload.executionId?.trim() || null;
  const apiBase = payload.apiBase?.trim() || null;
  return { spec, executionId, apiBase };
};

const buildTimeline = (
  frames: ReplayMovieFrame[] | undefined | null,
): FrameTimeline[] => {
  if (!frames || frames.length === 0) {
    return [];
  }
  const sorted = [...frames].sort((a, b) => {
    const aStart = toNumber(a.start_offset_ms) ?? 0;
    const bStart = toNumber(b.start_offset_ms) ?? 0;
    return aStart - bStart;
  });
  return sorted.map((frame, index) => {
    const duration = toNumber(frame.duration_ms) ?? DEFAULT_FRAME_DURATION_MS;
    const start =
      toNumber(frame.start_offset_ms) ??
      (index > 0 ? index * DEFAULT_FRAME_DURATION_MS : 0);
    return {
      index,
      startMs: Math.max(0, start),
      durationMs: Math.max(1, duration),
    };
  });
};

const computeTotalDuration = (
  summary: ReplayMovieSummary | undefined,
  timeline: FrameTimeline[],
): number => {
  if (summary?.total_duration_ms && summary.total_duration_ms > 0) {
    return summary.total_duration_ms;
  }
  if (timeline.length === 0) {
    return 0;
  }
  const last = timeline[timeline.length - 1];
  return last.startMs + Math.max(1, last.durationMs);
};

const resolveAssetUrl = (
  asset: ReplayMovieAsset | undefined,
): string | undefined => {
  if (!asset) {
    return undefined;
  }
  const candidates = [asset.source, asset.thumbnail];
  for (const candidate of candidates) {
    if (!candidate || typeof candidate !== "string") {
      continue;
    }
    const resolved = resolveUrl(candidate) ?? candidate;
    if (resolved) {
      return resolved;
    }
  }
  return undefined;
};

const toReplayFrame = (
  frame: ReplayMovieFrame,
  index: number,
  assetMap: Map<string, ReplayMovieAsset>,
): ReplayFrame => {
  const screenshotId =
    typeof frame.screenshot_asset_id === "string"
      ? frame.screenshot_asset_id
      : undefined;
  const asset = screenshotId ? assetMap.get(screenshotId) : undefined;
  const screenshotUrl = resolveAssetUrl(asset);
  const durationMs = toNumber(frame.duration_ms) ?? DEFAULT_FRAME_DURATION_MS;
  const holdMs = toNumber(frame.hold_ms) ?? 0;
  const totalDurationMs = durationMs + holdMs;
  const boundingBox = toBoundingBox(frame.element_bounding_box);
  const focusedElementBox = frame.focused_element?.boundingBox;
  const focusedBoundingBox = toBoundingBox(focusedElementBox);
  const clickPosition = toPoint(frame.click_position);
  const cursorTrail = mapTrail(
    frame.cursor_trail ?? frame.normalized_cursor_trail,
  );
  const retry = frame.resilience;

  return {
    id: frame.index != null ? String(frame.index) : `frame-${index}`,
    stepIndex: toNumber(frame.step_index) ?? index,
    nodeId: typeof frame.node_id === "string" ? frame.node_id : undefined,
    stepType: typeof frame.step_type === "string" ? frame.step_type : undefined,
    status: typeof frame.status === "string" ? frame.status : undefined,
    success: (frame.status ?? "").toLowerCase() !== "failed",
    durationMs,
    totalDurationMs,
    progress: 0,
    finalUrl: typeof frame.final_url === "string" ? frame.final_url : undefined,
    error: typeof frame.error === "string" ? frame.error : undefined,
    extractedDataPreview: frame.dom_snapshot_preview,
    consoleLogCount: toNumber(frame.console_log_count),
    networkEventCount: toNumber(frame.network_event_count),
    screenshot: screenshotUrl
      ? {
          artifactId: screenshotId ?? `artifact-${index}`,
          url: screenshotUrl,
          thumbnailUrl: resolveAssetUrl(asset),
          width: toNumber(asset?.width),
          height: toNumber(asset?.height),
          contentType:
            typeof asset?.type === "string" ? asset?.type : undefined,
          sizeBytes: toNumber(asset?.size_bytes),
        }
      : undefined,
    highlightRegions: mapRegions(frame.highlight_regions),
    maskRegions: mapRegions(frame.mask_regions),
    focusedElement: focusedBoundingBox
      ? {
          selector:
            typeof frame.focused_element?.selector === "string"
              ? frame.focused_element.selector
              : undefined,
          boundingBox: focusedBoundingBox,
        }
      : null,
    elementBoundingBox: boundingBox ?? null,
    clickPosition: clickPosition ?? null,
    cursorTrail,
    zoomFactor: toNumber(frame.zoom_factor),
    assertion: mapAssertion(frame.assertion),
    retryAttempt: toNumber(retry?.attempt),
    retryMaxAttempts: toNumber(retry?.max_attempts),
    retryConfigured: toNumber(retry?.configured_retries),
    retryDelayMs: toNumber(retry?.delay_ms),
    retryBackoffFactor: toNumber(retry?.backoff_factor),
    retryHistory: mapRetryHistory(retry?.history),
    domSnapshotPreview: frame.dom_snapshot_preview,
    domSnapshotHtml: frame.dom_snapshot_html,
    domSnapshotArtifactId: undefined,
  };
};

const findFrameForTime = (
  ms: number,
  timeline: FrameTimeline[],
): { index: number; progress: number } => {
  if (timeline.length === 0) {
    return { index: 0, progress: 0 };
  }
  const clamped = Math.max(0, ms);
  for (let i = timeline.length - 1; i >= 0; i -= 1) {
    const entry = timeline[i];
    if (clamped >= entry.startMs || i === 0) {
      const elapsed = clamped - entry.startMs;
      const progress =
        entry.durationMs > 0
          ? Math.min(Math.max(elapsed / entry.durationMs, 0), 1)
          : 1;
      return { index: entry.index, progress };
    }
  }
  const last = timeline[timeline.length - 1];
  return { index: last.index, progress: 1 };
};

const clampProgress = (value: number): number => {
  if (!Number.isFinite(value)) {
    return 0;
  }
  if (value <= 0) {
    return 0;
  }
  if (value >= 1) {
    return 1;
  }
  return value;
};

const normalizeStatus = (value: string | null | undefined): string => {
  if (!value) {
    return "";
  }
  return value.trim().toLowerCase();
};

const defaultStatusMessage = (status: string): string => {
  switch (status) {
    case "pending":
      return "Replay export pending – timeline frames not captured yet";
    case "unavailable":
      return "Replay export unavailable – execution did not capture any timeline frames";
    default:
      return "Replay export unavailable";
  }
};

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
        const element =
          controller?.getPresentationElement?.() ??
          controller?.getViewportElement();
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
        const viewportWidth = viewportRect
          ? Math.max(1, Math.round(viewportRect.width))
          : effectiveCanvasWidth;
        const viewportHeight = viewportRect
          ? Math.max(1, Math.round(viewportRect.height))
          : effectiveCanvasHeight;
        const canvasWidth = presentationRect
          ? Math.max(1, Math.round(presentationRect.width))
          : effectiveCanvasWidth;
        const canvasHeight = presentationRect
          ? Math.max(1, Math.round(presentationRect.height))
          : effectiveCanvasHeight;
        const browserFrameRadius =
          movieSpec?.presentation?.browser_frame?.radius ?? undefined;
        const browserFrame = (() => {
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

  const decor = movieSpec?.decor ?? {};
  const motion = movieSpec?.cursor_motion;
  const chromeTheme = (decor.chrome_theme ?? "aurora") as any;
  const backgroundTheme = (decor.background_theme ?? "aurora") as any;
  const cursorTheme = (decor.cursor_theme ?? "white") as any;
  const cursorInitialPosition = (decor.cursor_initial_position ??
    motion?.initial_position ??
    movieSpec?.cursor?.initial_position ??
    "center") as any;
  const cursorClickAnimation = (decor.cursor_click_animation ??
    motion?.click_animation ??
    movieSpec?.cursor?.click_animation ??
    "none") as any;
  const cursorScale =
    decor.cursor_scale ?? motion?.cursor_scale ?? movieSpec?.cursor?.scale ?? 1;
  const cursorDefaultSpeedProfile = asCursorSpeedProfile(motion?.speed_profile);
  const cursorDefaultPathStyle = asCursorPathStyle(motion?.path_style);

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
      <div className="flex min-h-screen w-full items-center justify-center bg-slate-950 p-8 text-slate-200">
        <div className="max-w-md rounded-2xl border border-slate-800 bg-slate-900/60 p-8 text-center shadow-[0_20px_60px_rgba(15,23,42,0.45)]">
          <h1 className="text-lg font-semibold text-slate-100">
            Replay export unavailable
          </h1>
          <p className="mt-3 text-sm text-slate-300/80">{loadError}</p>
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
        className={clsx(
          "relative transition-all duration-500",
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
          chromeTheme={chromeTheme}
          backgroundTheme={backgroundTheme}
          cursorTheme={cursorTheme}
          cursorInitialPosition={cursorInitialPosition}
          cursorScale={cursorScale}
          cursorClickAnimation={cursorClickAnimation}
          cursorDefaultSpeedProfile={cursorDefaultSpeedProfile}
          cursorDefaultPathStyle={cursorDefaultPathStyle}
          onFrameChange={handleFrameChange}
          onFrameProgressChange={handleProgressChange}
          exposeController={handleExposeController}
          presentationMode={mode === "standalone" ? "default" : "export"}
          allowPointerEditing={mode === "standalone"}
          presentationDimensions={{
            width: effectiveCanvasWidth,
            height: effectiveCanvasHeight,
            deviceScaleFactor:
              movieSpec?.presentation?.device_scale_factor ?? undefined,
          }}
        />

        {showPlaceholder && mode !== "capture" && (
          <div className="pointer-events-none absolute inset-0 flex items-center justify-center bg-slate-950/40">
            <span className="text-xs uppercase tracking-[0.3em] text-slate-300/80">
              {placeholderMessage}
            </span>
          </div>
        )}
      </div>
    </div>
  );
};

export default ReplayExportPage;
