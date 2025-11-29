import {
  useState,
  useEffect,
  useMemo,
  useCallback,
  useRef,
  useId,
  type CSSProperties,
  type MutableRefObject,
  type ReactNode,
} from "react";
import {
  Activity,
  Pause,
  RotateCw,
  X,
  Square,
  Terminal,
  Image,
  Clock,
  CheckCircle,
  XCircle,
  Loader,
  PlayCircle,
  AlertTriangle,
  ChevronDown,
  Check,
  ListTree,
  Download,
  Clapperboard,
  Film,
  FileJson,
  FolderOutput,
  Pencil,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { format } from "date-fns";
import clsx from "clsx";
import type {
  ReplayFrame,
  ReplayChromeTheme,
  ReplayBackgroundTheme,
  ReplayCursorTheme,
  ReplayCursorInitialPosition,
  ReplayCursorClickAnimation,
} from "./ReplayPlayer";
import { useExecutionStore } from "@stores/executionStore";
import type {
  Execution,
  TimelineFrame,
  TimelineArtifact,
} from "@stores/executionStore";
import { useWorkflowStore } from "@stores/workflowStore";
import { useExportStore, type Export } from "@stores/exportStore";
import { ExportSuccessPanel } from "./ExportSuccessPanel";
import type { Screenshot, LogEntry } from "@stores/executionEventProcessor";
import { toast } from "react-hot-toast";
import { ResponsiveDialog } from "@shared/layout";
import { getConfig } from "@/config";
import { logger } from "@utils/logger";
import type {
  ReplayMovieSpec,
  ReplayMovieFrameRect,
  ReplayMoviePresentation,
} from "@/types/export";
import {
  toNumber,
  toBoundingBox,
  toPoint,
  mapTrail,
  mapRegions,
  mapRetryHistory,
  mapAssertion,
  resolveUrl,
} from "@utils/executionTypeMappers";
import ExecutionHistory from "./ExecutionHistory";
import { selectors } from "@constants/selectors";
// Unsplash assets (IDs: m_7p45JfXQo, Tn29N3Hpf2E, KfFmwa7m5VQ) licensed for free use
const ABSOLUTE_URL_PATTERN = /^[a-zA-Z][a-zA-Z\d+.-]*:/;
const MOVIE_SPEC_POLL_INTERVAL_MS = 4000;

const stripApiSuffix = (value?: string | null): string | null => {
  if (!value) {
    return null;
  }

  try {
    const parsed = new URL(value);
    const cleanedPath = parsed.pathname
      .replace(/\/api\/v1\/?$/i, "")
      .replace(/\/+$/, "");
    return cleanedPath ? `${parsed.origin}${cleanedPath}` : parsed.origin;
  } catch {
    const cleaned = value.replace(/\/api\/v1\/?$/i, "").replace(/\/+$/, "");
    return cleaned.length > 0 ? cleaned : null;
  }
};

const withAssetBasePath = (value: string) => {
  if (!value || ABSOLUTE_URL_PATTERN.test(value)) {
    return value;
  }
  const base = import.meta.env.BASE_URL || "/";
  if (base === "/" || value.startsWith(base)) {
    return value;
  }
  const normalizedBase = base.endsWith("/") ? base : `${base}/`;
  const normalizedValue = value.startsWith("/") ? value.slice(1) : value;
  return `${normalizedBase}${normalizedValue}`;
};

const normalizePreviewStatus = (value?: string | null): string => {
  if (!value) {
    return "";
  }
  return value.trim().toLowerCase();
};

const describePreviewStatusMessage = (
  status: string,
  fallback?: string | null,
  metrics?: { capturedFrames?: number; assetCount?: number },
): string => {
  if (fallback && fallback.trim().length > 0) {
    return fallback.trim();
  }
  const capturedFrames = metrics?.capturedFrames ?? 0;
  switch (status) {
    case "pending":
      if (capturedFrames > 0) {
        return `Replay export pending – ${formatCapturedLabel(capturedFrames, "frame")} captured so far`;
      }
      return "Replay export pending – timeline frames not captured yet";
    case "unavailable":
      if (capturedFrames > 0) {
        return `Replay export unavailable – ${formatCapturedLabel(capturedFrames, "frame")} captured but not yet playable`;
      }
      return "Replay export unavailable – execution did not capture any timeline frames";
    default:
      return "Replay export unavailable";
  }
};

const resolveBackgroundAsset = (relativePath: string) => {
  const url = new URL(relativePath, import.meta.url);
  return withAssetBasePath(url.pathname || url.href);
};

const geometricPrismUrl = resolveBackgroundAsset(
  "../assets/replay-backgrounds/geometric-prism.jpg",
);
const geometricOrbitUrl = resolveBackgroundAsset(
  "../assets/replay-backgrounds/geometric-orbit.jpg",
);
const geometricMosaicUrl = resolveBackgroundAsset(
  "../assets/replay-backgrounds/geometric-mosaic.jpg",
);

interface ActiveExecutionProps {
  execution: Execution;
  onClose?: () => void;
  showExecutionSwitcher?: boolean;
}

type ViewerTab = "replay" | "screenshots" | "logs" | "executions";

const HEARTBEAT_WARN_SECONDS = 8;
const HEARTBEAT_STALL_SECONDS = 15;

const REPLAY_CHROME_OPTIONS: Array<{
  id: ReplayChromeTheme;
  label: string;
  subtitle: string;
}> = [
  { id: "aurora", label: "Aurora", subtitle: "macOS-inspired chrome" },
  { id: "chromium", label: "Chromium", subtitle: "Modern minimal controls" },
  { id: "midnight", label: "Midnight", subtitle: "Gradient showcase frame" },
  { id: "minimal", label: "Minimal", subtitle: "Hide browser chrome" },
];

const CURSOR_SCALE_MIN = 0.6;
const CURSOR_SCALE_MAX = 1.8;

type ExportFormat = "json" | "mp4" | "gif";

type SaveFilePickerOptions = {
  suggestedName?: string;
  types?: Array<{
    description?: string;
    accept: Record<string, string[]>;
  }>;
};

type FileSystemWritableFileStream = {
  write: (data: BlobPart) => Promise<void>;
  close: () => Promise<void>;
};

type FileSystemFileHandle = {
  createWritable: () => Promise<FileSystemWritableFileStream>;
};

interface ExecutionExportPreviewResponse {
  execution_id?: string;
  spec_id?: string;
  status?: string;
  message?: string;
  captured_frame_count?: number;
  available_asset_count?: number;
  total_duration_ms?: number;
  package?: ReplayMovieSpec;
}

interface ExecutionExportPreview {
  executionId: string;
  specId?: string;
  status: string;
  message?: string;
  capturedFrameCount: number;
  availableAssetCount: number;
  totalDurationMs: number;
  package?: ReplayMovieSpec;
}

type ExportDimensionPreset = "spec" | "1080p" | "720p" | "custom";

interface ExportPreviewMetrics {
  capturedFrames: number;
  assetCount: number;
  totalDurationMs: number;
}

const EXPORT_EXTENSIONS: Record<ExportFormat, string> = {
  json: "json",
  mp4: "mp4",
  gif: "gif",
};

const DIMENSION_PRESET_CONFIG: Record<
  "1080p" | "720p",
  { width: number; height: number; label: string }
> = {
  "1080p": { width: 1920, height: 1080, label: "1080p (Full HD)" },
  "720p": { width: 1280, height: 720, label: "720p (HD)" },
};

const DEFAULT_EXPORT_WIDTH = 1280;
const DEFAULT_EXPORT_HEIGHT = 720;

const EXPORT_FORMAT_OPTIONS: Array<{
  id: ExportFormat;
  label: string;
  description: string;
  icon: LucideIcon;
  badge?: string;
  disabled?: boolean;
}> = [
  {
    id: "mp4",
    label: "MP4 Video",
    description: "1080p marketing reel (server render)",
    icon: Clapperboard,
    badge: "Default",
  },
  {
    id: "gif",
    label: "Animated GIF",
    description: "Looped shareable highlight",
    icon: Film,
    badge: "Guide",
  },
  {
    id: "json",
    label: "JSON Package",
    description: "Raw replay bundle for tooling",
    icon: FileJson,
    badge: "Data",
  },
];

const sanitizeFileStem = (value: string, fallback: string): string => {
  const trimmed = value.trim();
  if (!trimmed) {
    return fallback;
  }
  const sanitized = trimmed.replace(/[^a-zA-Z0-9-_]/g, "-");
  if (!sanitized) {
    return fallback;
  }
  return sanitized;
};

const coerceMetricNumber = (value: unknown): number => {
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }
  if (typeof value === "string") {
    const parsed = Number.parseFloat(value);
    if (Number.isFinite(parsed)) {
      return parsed;
    }
  }
  return 0;
};

const formatCapturedLabel = (count: number, noun: string): string => {
  const rounded = Math.round(count);
  const suffix = rounded === 1 ? "" : "s";
  return `${rounded} ${noun}${suffix}`;
};

const REPLAY_BACKGROUND_OPTIONS: Array<{
  id: ReplayBackgroundTheme;
  label: string;
  subtitle: string;
  previewStyle: CSSProperties;
  previewNode?: ReactNode;
  kind: "abstract" | "solid" | "minimal" | "geometric";
}> = [
  {
    id: "aurora",
    label: "Aurora Glow",
    subtitle: "Iridescent gradient wash",
    previewStyle: {
      backgroundImage:
        "linear-gradient(135deg, rgba(56,189,248,0.7), rgba(129,140,248,0.7))",
    },
    kind: "abstract",
  },
  {
    id: "sunset",
    label: "Sunset Bloom",
    subtitle: "Fuchsia → amber ambience",
    previewStyle: {
      backgroundImage:
        "linear-gradient(135deg, rgba(244,114,182,0.9), rgba(251,191,36,0.88))",
      backgroundColor: "#43112d",
    },
    kind: "abstract",
  },
  {
    id: "ocean",
    label: "Ocean Depths",
    subtitle: "Cerulean blue gradient",
    previewStyle: {
      backgroundImage:
        "linear-gradient(135deg, rgba(14,165,233,0.78), rgba(30,64,175,0.82))",
    },
    kind: "abstract",
  },
  {
    id: "nebula",
    label: "Nebula Drift",
    subtitle: "Cosmic violet haze",
    previewStyle: {
      backgroundImage:
        "linear-gradient(135deg, rgba(147,51,234,0.78), rgba(99,102,241,0.78))",
    },
    kind: "abstract",
  },
  {
    id: "grid",
    label: "Tech Grid",
    subtitle: "Futuristic lattice backdrop",
    previewStyle: {
      backgroundColor: "#0f172a",
      backgroundImage:
        "linear-gradient(rgba(96,165,250,0.34) 1px, transparent 1px), linear-gradient(90deg, rgba(96,165,250,0.31) 1px, transparent 1px)",
      backgroundSize: "14px 14px",
    },
    kind: "abstract",
  },
  {
    id: "charcoal",
    label: "Charcoal",
    subtitle: "Deep neutral tone",
    previewStyle: {
      backgroundColor: "#0f172a",
    },
    kind: "solid",
  },
  {
    id: "steel",
    label: "Steel Slate",
    subtitle: "Cool slate finish",
    previewStyle: {
      backgroundColor: "#1f2937",
    },
    kind: "solid",
  },
  {
    id: "emerald",
    label: "Evergreen",
    subtitle: "Saturated green solid",
    previewStyle: {
      backgroundColor: "#064e3b",
    },
    kind: "solid",
  },
  {
    id: "none",
    label: "No Background",
    subtitle: "Edge-to-edge browser",
    previewStyle: {
      backgroundColor: "transparent",
      backgroundImage:
        "linear-gradient(45deg, rgba(148,163,184,0.35) 25%, transparent 25%, transparent 50%, rgba(148,163,184,0.35) 50%, rgba(148,163,184,0.35) 75%, transparent 75%, transparent)",
      backgroundSize: "10px 10px",
    },
    kind: "minimal",
  },
  {
    id: "geoPrism",
    label: "Prismatic Peaks",
    subtitle: "Layered neon triangles",
    previewStyle: {
      backgroundColor: "#0f172a",
    },
    previewNode: (
      <span className="absolute inset-0 overflow-hidden">
        <img
          src={geometricPrismUrl}
          alt=""
          loading="lazy"
          className="h-full w-full object-cover"
        />
        <span className="absolute inset-0 bg-gradient-to-br from-cyan-400/30 via-transparent to-indigo-500/24 mix-blend-screen" />
        <span className="absolute inset-0 bg-slate-950/45" />
      </span>
    ),
    kind: "geometric",
  },
  {
    id: "geoOrbit",
    label: "Orbital Glow",
    subtitle: "Concentric energy orbits",
    previewStyle: {
      backgroundColor: "#0b1120",
    },
    previewNode: (
      <span className="absolute inset-0 overflow-hidden">
        <img
          src={geometricOrbitUrl}
          alt=""
          loading="lazy"
          className="h-full w-full object-cover"
        />
        <span className="absolute inset-0 bg-gradient-to-br from-sky-300/28 via-transparent to-amber-300/20 mix-blend-screen" />
        <span className="absolute inset-0 bg-slate-950/45" />
      </span>
    ),
    kind: "geometric",
  },
  {
    id: "geoMosaic",
    label: "Isometric Mosaic",
    subtitle: "Staggered tile lattice",
    previewStyle: {
      backgroundColor: "#0b1526",
    },
    previewNode: (
      <span className="absolute inset-0 overflow-hidden">
        <img
          src={geometricMosaicUrl}
          alt=""
          loading="lazy"
          className="h-full w-full object-cover"
        />
        <span className="absolute inset-0 bg-gradient-to-tr from-sky-400/28 via-transparent to-indigo-400/22 mix-blend-screen" />
        <span className="absolute inset-0 bg-slate-950/45" />
      </span>
    ),
    kind: "geometric",
  },
];

const BACKGROUND_GROUP_ORDER: Array<{
  id: (typeof REPLAY_BACKGROUND_OPTIONS)[number]["kind"];
  label: string;
}> = [
  { id: "abstract", label: "Abstract" },
  { id: "solid", label: "Solid" },
  { id: "minimal", label: "Minimal" },
  { id: "geometric", label: "Geometric" },
];

type BackgroundOption = (typeof REPLAY_BACKGROUND_OPTIONS)[number];

type CursorOption = {
  id: ReplayCursorTheme;
  label: string;
  subtitle: string;
  group: "hidden" | "halo" | "arrow" | "hand";
  preview: ReactNode;
};

const ARROW_CURSOR_PATH =
  "M6 3L6 22L10.4 18.1L13.1 26.4L15.9 25.2L13.1 17.5L22 17.5L6 3Z";

const HAND_POINTER_PATHS = [
  "M22 14a8 8 0 0 1-8 8",
  "M18 11v-1a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v0",
  "M14 10V9a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v1",
  "M10 9.5V4a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v10",
  "M18 11a2 2 0 1 1 4 0v3a8 8 0 0 1-8 8h-2c-2.8 0-4.5-.86-5.99-2.34l-3.6-3.6a2 2 0 0 1 2.83-2.82L7 15",
];

const REPLAY_CURSOR_OPTIONS: CursorOption[] = [
  {
    id: "disabled",
    group: "hidden",
    label: "Hidden",
    subtitle: "No virtual cursor overlay",
    preview: (
      <span className="inline-flex h-7 w-7 items-center justify-center rounded-full border border-white/15 text-[10px] uppercase tracking-[0.18em] text-slate-400">
        Off
      </span>
    ),
  },
  {
    id: "white",
    group: "halo",
    label: "Soft White",
    subtitle: "Clean highlight for dark scenes",
    preview: (
      <span className="relative inline-flex h-7 w-7 items-center justify-center rounded-full border-2 border-white/85 bg-white/90 shadow-[0_8px_20px_rgba(148,163,184,0.4)]">
        <span className="h-1.5 w-1.5 rounded-full bg-slate-500/60" />
      </span>
    ),
  },
  {
    id: "black",
    group: "halo",
    label: "Carbon Dark",
    subtitle: "High contrast for bright scenes",
    preview: (
      <span className="relative inline-flex h-7 w-7 items-center justify-center rounded-full border-2 border-black bg-slate-900 shadow-[0_8px_20px_rgba(15,23,42,0.55)]">
        <span className="h-2 w-2 rounded-full bg-white/80" />
      </span>
    ),
  },
  {
    id: "aura",
    group: "halo",
    label: "Aura Glow",
    subtitle: "Brand accent trail",
    preview: (
      <span className="relative inline-flex h-7 w-7 items-center justify-center rounded-full border-2 border-cyan-200/80 bg-gradient-to-br from-sky-400 via-emerald-300 to-violet-400 shadow-[0_10px_22px_rgba(56,189,248,0.45)]">
        <span className="absolute -inset-0.5 rounded-full border border-cyan-300/50 opacity-70" />
      </span>
    ),
  },
  {
    id: "arrowLight",
    group: "arrow",
    label: "Classic Light",
    subtitle: "OS-style white arrow",
    preview: (
      <span className="relative inline-flex h-7 w-7 items-center justify-center text-white">
        <svg viewBox="0 0 32 32" className="h-6 w-6">
          <path
            d={ARROW_CURSOR_PATH}
            fill="rgba(255,255,255,0.95)"
            stroke="rgba(15,23,42,0.85)"
            strokeWidth={1.4}
            strokeLinejoin="round"
          />
        </svg>
      </span>
    ),
  },
  {
    id: "arrowDark",
    group: "arrow",
    label: "Noir Precision",
    subtitle: "Deep slate pointer with halo",
    preview: (
      <span className="relative inline-flex h-7 w-7 items-center justify-center text-white">
        <svg viewBox="0 0 32 32" className="h-6 w-6">
          <path
            d={ARROW_CURSOR_PATH}
            fill="rgba(30,41,59,0.95)"
            stroke="rgba(226,232,240,0.9)"
            strokeWidth={1.3}
            strokeLinejoin="round"
          />
        </svg>
      </span>
    ),
  },
  {
    id: "arrowNeon",
    group: "arrow",
    label: "Neon Signal",
    subtitle: "Gradient arrow with glow",
    preview: (
      <span className="relative inline-flex h-7 w-7 items-center justify-center text-white">
        <svg viewBox="0 0 32 32" className="h-6 w-6">
          <defs>
            <linearGradient
              id="cursor-neon-preview"
              x1="0%"
              y1="0%"
              x2="100%"
              y2="100%"
            >
              <stop offset="0%" stopColor="#38bdf8" />
              <stop offset="45%" stopColor="#34d399" />
              <stop offset="100%" stopColor="#a855f7" />
            </linearGradient>
          </defs>
          <path
            d={ARROW_CURSOR_PATH}
            fill="url(#cursor-neon-preview)"
            stroke="rgba(191,219,254,0.9)"
            strokeWidth={1.2}
            strokeLinejoin="round"
          />
        </svg>
        <span className="pointer-events-none absolute -inset-1 rounded-full bg-cyan-300/25 blur-md" />
      </span>
    ),
  },
  {
    id: "handNeutral",
    group: "hand",
    label: "Pointer Neutral",
    subtitle: "Classic hand cursor outline",
    preview: (
      <span className="relative inline-flex h-7 w-7 items-center justify-center">
        <svg viewBox="0 0 24 24" className="h-6 w-6" fill="none">
          {HAND_POINTER_PATHS.map((path, index) => (
            <path
              key={`hand-neutral-${index}`}
              d={path}
              stroke="rgba(241,245,249,0.92)"
              strokeWidth={1.7}
              strokeLinecap="round"
              strokeLinejoin="round"
            />
          ))}
        </svg>
      </span>
    ),
  },
  {
    id: "handAura",
    group: "hand",
    label: "Pointer Aura",
    subtitle: "Gradient hand with halo",
    preview: (
      <span className="relative inline-flex h-7 w-7 items-center justify-center">
        <svg viewBox="0 0 24 24" className="h-6 w-6" fill="none">
          <defs>
            <linearGradient
              id="cursor-hand-preview"
              x1="10%"
              y1="5%"
              x2="80%"
              y2="95%"
            >
              <stop offset="0%" stopColor="#38bdf8" />
              <stop offset="50%" stopColor="#34d399" />
              <stop offset="100%" stopColor="#a855f7" />
            </linearGradient>
          </defs>
          {HAND_POINTER_PATHS.map((path, index) => (
            <path
              key={`hand-aura-${index}`}
              d={path}
              stroke="url(#cursor-hand-preview)"
              strokeWidth={1.7}
              strokeLinecap="round"
              strokeLinejoin="round"
            />
          ))}
        </svg>
        <span className="pointer-events-none absolute h-8 w-8 rounded-full bg-cyan-400/25 blur-lg" />
      </span>
    ),
  },
];

const CURSOR_GROUP_ORDER: Array<{ id: CursorOption["group"]; label: string }> =
  [
    { id: "hidden", label: "Hidden" },
    { id: "halo", label: "Halo Cursors" },
    { id: "arrow", label: "Arrowhead Cursors" },
    { id: "hand", label: "Pointing Hands" },
  ];

const REPLAY_CURSOR_CLICK_ANIMATION_OPTIONS: Array<{
  id: ReplayCursorClickAnimation;
  label: string;
  subtitle: string;
  preview: ReactNode;
}> = [
  {
    id: "none",
    label: "None",
    subtitle: "No click highlight",
    preview: (
      <span className="inline-flex h-7 w-7 items-center justify-center rounded-full border border-white/15 text-[10px] uppercase tracking-[0.18em] text-slate-400">
        Off
      </span>
    ),
  },
  {
    id: "pulse",
    label: "Pulse",
    subtitle: "Radial glow on click",
    preview: (
      <span className="relative inline-flex h-7 w-7 items-center justify-center">
        <span className="absolute h-6 w-6 rounded-full border border-sky-300/60 bg-sky-400/20" />
        <span className="absolute h-10 w-10 rounded-full border border-sky-400/30" />
        <span className="relative h-2 w-2 rounded-full bg-white/80" />
      </span>
    ),
  },
  {
    id: "ripple",
    label: "Ripple",
    subtitle: "Expanding ring accent",
    preview: (
      <span className="relative inline-flex h-7 w-7 items-center justify-center">
        <span className="absolute h-5 w-5 rounded-full border border-violet-300/70" />
        <span className="absolute h-9 w-9 rounded-full border border-violet-400/40" />
        <span className="relative h-1.5 w-1.5 rounded-full bg-violet-200" />
      </span>
    ),
  },
];

type ClickAnimationOption =
  (typeof REPLAY_CURSOR_CLICK_ANIMATION_OPTIONS)[number];

const REPLAY_CURSOR_POSITIONS: Array<{
  id: ReplayCursorInitialPosition;
  label: string;
  subtitle: string;
}> = [
  { id: "center", label: "Center", subtitle: "Start replay from the middle" },
  {
    id: "top-left",
    label: "Top Left",
    subtitle: "Anchor to the navigation corner",
  },
  { id: "top-right", label: "Top Right", subtitle: "Anchor to utility corner" },
  {
    id: "bottom-left",
    label: "Bottom Left",
    subtitle: "Anchor to lower control area",
  },
  {
    id: "bottom-right",
    label: "Bottom Right",
    subtitle: "Anchor to lower action edge",
  },
  {
    id: "random",
    label: "Randomized",
    subtitle: "Fresh placement each replay",
  },
];

const isReplayChromeTheme = (
  value: string | null | undefined,
): value is ReplayChromeTheme =>
  Boolean(value && REPLAY_CHROME_OPTIONS.some((option) => option.id === value));

const isReplayBackgroundTheme = (
  value: string | null | undefined,
): value is ReplayBackgroundTheme =>
  Boolean(
    value && REPLAY_BACKGROUND_OPTIONS.some((option) => option.id === value),
  );

const isReplayCursorTheme = (
  value: string | null | undefined,
): value is ReplayCursorTheme =>
  Boolean(value && REPLAY_CURSOR_OPTIONS.some((option) => option.id === value));

const isReplayCursorInitialPosition = (
  value: string | null | undefined,
): value is ReplayCursorInitialPosition =>
  Boolean(
    value && REPLAY_CURSOR_POSITIONS.some((option) => option.id === value),
  );

const isReplayCursorClickAnimation = (
  value: string | null | undefined,
): value is ReplayCursorClickAnimation =>
  Boolean(
    value &&
      REPLAY_CURSOR_CLICK_ANIMATION_OPTIONS.some(
        (option) => option.id === value,
      ),
  );

function ActiveExecutionViewer({
  execution,
  onClose,
  showExecutionSwitcher = false,
}: ActiveExecutionProps) {
  const refreshTimeline = useExecutionStore((state) => state.refreshTimeline);
  const stopExecution = useExecutionStore((state) => state.stopExecution);
  const startExecution = useExecutionStore((state) => state.startExecution);
  const loadExecution = useExecutionStore((state) => state.loadExecution);
  const loadExecutions = useExecutionStore((state) => state.loadExecutions);
  const workflowName = useCurrentWorkflowName();
  const [activeTab, setActiveTab] = useState<ViewerTab>("replay");
  const [hasAutoSwitchedToReplay, setHasAutoSwitchedToReplay] =
    useState<boolean>(
      Boolean(execution.timeline && execution.timeline.length > 0),
    );
  const [selectedScreenshot, setSelectedScreenshot] =
    useState<Screenshot | null>(null);
  const [hasAutoOpenedScreenshots, setHasAutoOpenedScreenshots] =
    useState(false);
  const [, setHeartbeatTick] = useState(0);
  const [isStopping, setIsStopping] = useState(false);
  const [isRestarting, setIsRestarting] = useState(false);
  const [isSwitchingExecution, setIsSwitchingExecution] = useState(false);
  const [replayChromeTheme, setReplayChromeTheme] = useState<ReplayChromeTheme>(
    () => {
      if (typeof window === "undefined") {
        return "aurora";
      }
      const stored = window.localStorage.getItem(
        "browserAutomation.replayChromeTheme",
      );
      return isReplayChromeTheme(stored) ? stored : "aurora";
    },
  );
  const [replayBackgroundTheme, setReplayBackgroundTheme] =
    useState<ReplayBackgroundTheme>(() => {
      if (typeof window === "undefined") {
        return "aurora";
      }
      const stored = window.localStorage.getItem(
        "browserAutomation.replayBackgroundTheme",
      );
      return isReplayBackgroundTheme(stored) ? stored : "aurora";
    });
  const [replayCursorTheme, setReplayCursorTheme] = useState<ReplayCursorTheme>(
    () => {
      if (typeof window === "undefined") {
        return "white";
      }
      const stored = window.localStorage.getItem(
        "browserAutomation.replayCursorTheme",
      );
      return isReplayCursorTheme(stored) ? stored : "white";
    },
  );
  const [replayCursorInitialPosition, setReplayCursorInitialPosition] =
    useState<ReplayCursorInitialPosition>(() => {
      if (typeof window === "undefined") {
        return "center";
      }
      const stored = window.localStorage.getItem(
        "browserAutomation.replayCursorInitialPosition",
      );
      return isReplayCursorInitialPosition(stored) ? stored : "center";
    });
  const [replayCursorClickAnimation, setReplayCursorClickAnimation] =
    useState<ReplayCursorClickAnimation>(() => {
      if (typeof window === "undefined") {
        return "none";
      }
      const stored = window.localStorage.getItem(
        "browserAutomation.replayCursorClickAnimation",
      );
      return isReplayCursorClickAnimation(stored) ? stored : "none";
    });
  const [replayCursorScale, setReplayCursorScale] = useState<number>(() => {
    if (typeof window === "undefined") {
      return 1;
    }
    const stored = window.localStorage.getItem(
      "browserAutomation.replayCursorScale",
    );
    if (!stored) {
      return 1;
    }
    const parsed = Number.parseFloat(stored);
    if (Number.isFinite(parsed)) {
      return Math.min(CURSOR_SCALE_MAX, Math.max(CURSOR_SCALE_MIN, parsed));
    }
    return 1;
  });
  const [isBackgroundMenuOpen, setIsBackgroundMenuOpen] = useState(false);
  const [isCursorMenuOpen, setIsCursorMenuOpen] = useState(false);
  const [isCursorPositionMenuOpen, setIsCursorPositionMenuOpen] =
    useState(false);
  const [isCursorClickAnimationMenuOpen, setIsCursorClickAnimationMenuOpen] =
    useState(false);
  const [isCustomizationCollapsed, setIsCustomizationCollapsed] =
    useState(true);
  const screenshotRefs = useRef<Record<string, HTMLDivElement | null>>({});
  const backgroundSelectorRef = useRef<HTMLDivElement | null>(null);
  const cursorSelectorRef = useRef<HTMLDivElement | null>(null);
  const cursorPositionSelectorRef = useRef<HTMLDivElement | null>(null);
  const cursorClickAnimationSelectorRef = useRef<HTMLDivElement | null>(null);
  const preloadedWorkflowRef = useRef<string | null>(null);
  const [isExportDialogOpen, setIsExportDialogOpen] = useState(false);
  const [exportFormat, setExportFormat] = useState<ExportFormat>("mp4");
  const [exportFileStem, setExportFileStem] = useState<string>(
    () => `browser-automation-replay-${execution.id.slice(0, 8)}`,
  );
  const [useNativeFilePicker, setUseNativeFilePicker] = useState(false);
  const [dimensionPreset, setDimensionPreset] =
    useState<ExportDimensionPreset>("spec");
  const [customWidthInput, setCustomWidthInput] = useState<string>(() =>
    String(DEFAULT_EXPORT_WIDTH),
  );
  const [customHeightInput, setCustomHeightInput] = useState<string>(() =>
    String(DEFAULT_EXPORT_HEIGHT),
  );
  const [exportPreview, setExportPreview] =
    useState<ExecutionExportPreview | null>(null);
  const [exportPreviewExecutionId, setExportPreviewExecutionId] = useState<
    string | null
  >(null);
  const [isExportPreviewLoading, setIsExportPreviewLoading] = useState(false);
  const [exportPreviewError, setExportPreviewError] = useState<string | null>(
    null,
  );
  const [previewMetrics, setPreviewMetrics] = useState<ExportPreviewMetrics>({
    capturedFrames: 0,
    assetCount: 0,
    totalDurationMs: 0,
  });
  const [activeSpecId, setActiveSpecId] = useState<string | null>(null);
  const [movieSpec, setMovieSpec] = useState<ReplayMovieSpec | null>(null);
  const [isMovieSpecLoading, setIsMovieSpecLoading] = useState(false);
  const [movieSpecError, setMovieSpecError] = useState<string | null>(null);
  const [composerApiBase, setComposerApiBase] = useState<string | null>(() => {
    if (typeof window === "undefined") {
      return null;
    }
    const existing = (
      window as typeof window & {
        __BAS_EXPORT_API_BASE__?: unknown;
      }
    ).__BAS_EXPORT_API_BASE__;
    return typeof existing === "string" && existing.trim().length > 0
      ? existing.trim()
      : null;
  });
  const composerRef = useRef<HTMLIFrameElement | null>(null);
  const composerWindowRef = useRef<Window | null>(null);
  const composerOriginRef = useRef<string | null>(null);
  const previewComposerRef = useRef<HTMLIFrameElement | null>(null);
  const composerPreviewWindowRef = useRef<Window | null>(null);
  const composerPreviewOriginRef = useRef<string | null>(null);
  const [isComposerReady, setIsComposerReady] = useState(false);
  const [isPreviewComposerReady, setIsPreviewComposerReady] = useState(false);
  const [previewComposerError, setPreviewComposerError] = useState<
    string | null
  >(null);
  const composerFrameStateRef = useRef({ frameIndex: 0, progress: 0 });
  const movieSpecRetryTimeoutRef = useRef<number | null>(null);
  const movieSpecAbortControllerRef = useRef<AbortController | null>(null);
  const [isExporting, setIsExporting] = useState(false);
  const [showExportSuccess, setShowExportSuccess] = useState(false);
  const [lastCreatedExport, setLastCreatedExport] = useState<Export | null>(null);
  const { createExport } = useExportStore();
  const supportsFileSystemAccess =
    typeof window !== "undefined" &&
    typeof (window as typeof window & { showSaveFilePicker?: unknown })
      .showSaveFilePicker === "function";

  useEffect(() => {
    if (composerApiBase) {
      return;
    }

    let cancelled = false;

    (async () => {
      try {
        const configData = await getConfig();
        if (cancelled) {
          return;
        }
        const derived = stripApiSuffix(configData.API_URL);
        if (derived) {
          setComposerApiBase((current) =>
            current && current.length > 0 ? current : derived,
          );
        }
      } catch (error) {
        if (!cancelled) {
          logger.warn(
            "Failed to derive API base for replay composer",
            { component: "ExecutionViewer", executionId: execution.id },
            error,
          );
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [composerApiBase, execution.id]);

  useEffect(() => {
    if (typeof window === "undefined" || !composerApiBase) {
      return;
    }
    (
      window as typeof window & { __BAS_EXPORT_API_BASE__?: string }
    ).__BAS_EXPORT_API_BASE__ = composerApiBase;
  }, [composerApiBase]);

  const specCanvasWidth =
    movieSpec?.presentation?.canvas?.width ??
    movieSpec?.presentation?.viewport?.width ??
    DEFAULT_EXPORT_WIDTH;
  const specCanvasHeight =
    movieSpec?.presentation?.canvas?.height ??
    movieSpec?.presentation?.viewport?.height ??
    DEFAULT_EXPORT_HEIGHT;

  const defaultExportFileStem = useMemo(
    () => `browser-automation-replay-${execution.id.slice(0, 8)}`,
    [execution.id],
  );

  const composerUrl = useMemo(() => {
    const embedOrigin =
      typeof window !== "undefined"
        ? window.location.origin
        : "http://localhost";
    const url = new URL("/export/composer.html", embedOrigin);
    url.searchParams.set("mode", "embedded");
    url.searchParams.set("executionId", execution.id);
    url.searchParams.set("apiBase", composerApiBase ?? embedOrigin);
    return url.toString();
  }, [composerApiBase, execution.id]);

  const composerPreviewUrl = useMemo(() => {
    const embedOrigin =
      typeof window !== "undefined"
        ? window.location.origin
        : "http://localhost";
    const url = new URL("/export/composer.html", embedOrigin);
    url.searchParams.set("mode", "embedded");
    url.searchParams.set("executionId", execution.id);
    url.searchParams.set("apiBase", composerApiBase ?? embedOrigin);
    url.searchParams.set("context", "preview");
    return url.toString();
  }, [composerApiBase, execution.id]);

  useEffect(() => {
    if (typeof window === "undefined") {
      composerOriginRef.current = null;
      return;
    }
    try {
      composerOriginRef.current = new URL(
        composerUrl,
        window.location.href,
      ).origin;
    } catch (error) {
      composerOriginRef.current = null;
      logger.warn(
        "Failed to derive composer origin",
        { component: "ExecutionViewer", executionId: execution.id },
        error,
      );
    }
  }, [composerUrl, execution.id]);

  useEffect(() => {
    if (typeof window === "undefined") {
      composerPreviewOriginRef.current = null;
      return;
    }
    try {
      composerPreviewOriginRef.current = new URL(
        composerPreviewUrl,
        window.location.href,
      ).origin;
    } catch (error) {
      composerPreviewOriginRef.current = null;
      logger.warn(
        "Failed to derive preview composer origin",
        { component: "ExecutionViewer", executionId: execution.id },
        error,
      );
    }
  }, [composerPreviewUrl, execution.id]);

  const getSanitizedFileStem = useCallback(
    () => sanitizeFileStem(exportFileStem, defaultExportFileStem),
    [exportFileStem, defaultExportFileStem],
  );

  const buildOutputFileName = useCallback(
    (extension: string) => {
      const stem = getSanitizedFileStem();
      return `${stem}.${extension}`;
    },
    [getSanitizedFileStem],
  );

  const finalFileName = useMemo(
    () => buildOutputFileName(EXPORT_EXTENSIONS[exportFormat]),
    [buildOutputFileName, exportFormat],
  );

  const heartbeatTimestamp = execution.lastHeartbeat?.timestamp?.valueOf();
  const executionError = execution.error ?? undefined;

  useEffect(() => {
    if (execution.status !== "running" || !heartbeatTimestamp) {
      return;
    }
    const interval = window.setInterval(() => {
      setHeartbeatTick((tick) => tick + 1);
    }, 1000);
    return () => {
      window.clearInterval(interval);
    };
  }, [execution.status, heartbeatTimestamp]);

  useEffect(() => {
    setHasAutoSwitchedToReplay(
      Boolean(execution.timeline && execution.timeline.length > 0),
    );
    setActiveTab("replay");
    setSelectedScreenshot(null);
  }, [execution.id]);

  useEffect(() => {
    setIsExportDialogOpen(false);
    setExportPreview(null);
    setExportPreviewExecutionId(null);
    setExportPreviewError(null);
    setExportFileStem(defaultExportFileStem);
    setUseNativeFilePicker(false);
  }, [defaultExportFileStem]);

  useEffect(() => {
    setCustomWidthInput(String(specCanvasWidth));
    setCustomHeightInput(String(specCanvasHeight));
    setDimensionPreset("spec");
  }, [execution.id, specCanvasWidth, specCanvasHeight]);

  const clearMovieSpecRetryTimeout = useCallback(() => {
    if (movieSpecRetryTimeoutRef.current != null) {
      window.clearTimeout(movieSpecRetryTimeoutRef.current);
      movieSpecRetryTimeoutRef.current = null;
    }
  }, []);

  useEffect(() => {
    return () => {
      clearMovieSpecRetryTimeout();
      if (movieSpecAbortControllerRef.current) {
        movieSpecAbortControllerRef.current.abort();
        movieSpecAbortControllerRef.current = null;
      }
    };
  }, [clearMovieSpecRetryTimeout]);

  useEffect(() => {
    let isCancelled = false;

    if (movieSpecAbortControllerRef.current) {
      movieSpecAbortControllerRef.current.abort();
      movieSpecAbortControllerRef.current = null;
    }
    clearMovieSpecRetryTimeout();

    async function fetchMovieSpec() {
      if (isCancelled) {
        return;
      }

      if (movieSpecAbortControllerRef.current) {
        movieSpecAbortControllerRef.current.abort();
      }
      const abortController = new AbortController();
      movieSpecAbortControllerRef.current = abortController;

      let shouldRetry = false;
      try {
        setIsMovieSpecLoading(true);
        const configData = await getConfig();
        const response = await fetch(
          `${configData.API_URL}/executions/${execution.id}/export`,
          {
            method: "POST",
            headers: {
              "Content-Type": "application/json",
              Accept: "application/json",
            },
            body: JSON.stringify({ format: "json" }),
            signal: abortController.signal,
          },
        );
        if (!response.ok) {
          const text = await response.text();
          throw new Error(text || `Export request failed (${response.status})`);
        }
        const raw = (await response.json()) as ExecutionExportPreviewResponse;
        if (isCancelled) {
          return;
        }
        const status = normalizePreviewStatus(raw.status);
        const capturedFramesRaw =
          raw.captured_frame_count ??
          raw.package?.summary?.frame_count ??
          raw.package?.frames?.length ??
          0;
        const availableAssetRaw =
          raw.available_asset_count ?? raw.package?.assets?.length ?? 0;
        const totalDurationRaw =
          raw.total_duration_ms ??
          raw.package?.summary?.total_duration_ms ??
          raw.package?.playback?.duration_ms ??
          0;
        const specId =
          (typeof raw.spec_id === "string" && raw.spec_id.trim()) ||
          raw.execution_id ||
          execution.id;
        const metrics: ExportPreviewMetrics = {
          capturedFrames: coerceMetricNumber(capturedFramesRaw),
          assetCount: coerceMetricNumber(availableAssetRaw),
          totalDurationMs: coerceMetricNumber(totalDurationRaw),
        };
        setPreviewMetrics(metrics);
        setActiveSpecId(specId);
        const message = describePreviewStatusMessage(
          status,
          raw.message,
          metrics,
        );
        if (status && status !== "ready") {
          setMovieSpec(null);
          setMovieSpecError(message);
          shouldRetry = status === "pending";
          return;
        }
        if (!raw.package) {
          setMovieSpec(null);
          setMovieSpecError(
            "Replay export unavailable – missing export package",
          );
          return;
        }
        clearMovieSpecRetryTimeout();
        setMovieSpec(raw.package);
        setMovieSpecError(null);
      } catch (error) {
        if (isCancelled) {
          return;
        }
        if (error instanceof DOMException && error.name === "AbortError") {
          return;
        }
        const message =
          error instanceof Error ? error.message : "Failed to load replay spec";
        setMovieSpecError(message);
        setMovieSpec(null);
        logger.error(
          "Failed to load replay movie spec",
          { component: "ExecutionViewer", executionId: execution.id },
          error,
        );
        shouldRetry = execution.status === "running";
      } finally {
        if (!isCancelled) {
          setIsMovieSpecLoading(false);
        }
        if (!isCancelled) {
          movieSpecAbortControllerRef.current = null;
        }
        if (!isCancelled && shouldRetry) {
          clearMovieSpecRetryTimeout();
          movieSpecRetryTimeoutRef.current = window.setTimeout(() => {
            void fetchMovieSpec();
          }, MOVIE_SPEC_POLL_INTERVAL_MS);
        }
      }
    }

    setMovieSpecError(null);
    setMovieSpec(null);
    setIsMovieSpecLoading(true);
    void fetchMovieSpec();

    return () => {
      isCancelled = true;
      clearMovieSpecRetryTimeout();
      if (movieSpecAbortControllerRef.current) {
        movieSpecAbortControllerRef.current.abort();
        movieSpecAbortControllerRef.current = null;
      }
    };
  }, [
    execution.id,
    execution.status,
    execution.timeline?.length,
    clearMovieSpecRetryTimeout,
  ]);

  useEffect(() => {
    if (exportFormat !== "json" && useNativeFilePicker) {
      setUseNativeFilePicker(false);
    }
  }, [exportFormat, useNativeFilePicker]);

  useEffect(() => {
    if (!showExecutionSwitcher || !execution.workflowId) {
      return;
    }
    if (preloadedWorkflowRef.current === execution.workflowId) {
      return;
    }
    preloadedWorkflowRef.current = execution.workflowId;
    void loadExecutions(execution.workflowId);
  }, [execution.workflowId, loadExecutions, showExecutionSwitcher]);

  useEffect(() => {
    if (!showExecutionSwitcher && activeTab === "executions") {
      setActiveTab("replay");
    }
  }, [showExecutionSwitcher, activeTab]);

  useEffect(() => {
    if (!isExportDialogOpen) {
      return;
    }
    if (exportPreview && exportPreviewExecutionId === execution.id) {
      return;
    }
    let isCancelled = false;
    setIsExportPreviewLoading(true);
    setExportPreviewError(null);
    void (async () => {
      try {
        const configData = await getConfig();
        const response = await fetch(
          `${configData.API_URL}/executions/${execution.id}/export`,
          {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ format: "json" }),
          },
        );
        if (!response.ok) {
          const text = await response.text();
          throw new Error(text || `Export request failed (${response.status})`);
        }
        const raw = (await response.json()) as ExecutionExportPreviewResponse;
        if (isCancelled) {
          return;
        }
        const executionIdForPreview = raw.execution_id ?? execution.id;
        const specId =
          (typeof raw.spec_id === "string" && raw.spec_id.trim()) ||
          executionIdForPreview;
        const status = normalizePreviewStatus(raw.status) || "unknown";
        const capturedFramesRaw =
          raw.captured_frame_count ??
          raw.package?.summary?.frame_count ??
          raw.package?.frames?.length ??
          0;
        const availableAssetRaw =
          raw.available_asset_count ?? raw.package?.assets?.length ?? 0;
        const totalDurationRaw =
          raw.total_duration_ms ??
          raw.package?.summary?.total_duration_ms ??
          raw.package?.playback?.duration_ms ??
          0;
        const metrics: ExportPreviewMetrics = {
          capturedFrames: coerceMetricNumber(capturedFramesRaw),
          assetCount: coerceMetricNumber(availableAssetRaw),
          totalDurationMs: coerceMetricNumber(totalDurationRaw),
        };
        const parsed: ExecutionExportPreview = {
          executionId: executionIdForPreview,
          specId,
          status,
          message: describePreviewStatusMessage(status, raw.message, metrics),
          capturedFrameCount: metrics.capturedFrames,
          availableAssetCount: metrics.assetCount,
          totalDurationMs: metrics.totalDurationMs,
          package: raw.package,
        };
        setExportPreview(parsed);
        setPreviewMetrics(metrics);
        setActiveSpecId(specId);
        setExportPreviewExecutionId(parsed.executionId);
      } catch (error) {
        if (isCancelled) {
          return;
        }
        const message =
          error instanceof Error
            ? error.message
            : "Failed to prepare export preview";
        setExportPreviewError(message);
        setExportPreview(null);
      } finally {
        if (!isCancelled) {
          setIsExportPreviewLoading(false);
        }
      }
    })();
    return () => {
      isCancelled = true;
    };
  }, [
    execution.id,
    exportPreview,
    exportPreviewExecutionId,
    isExportDialogOpen,
  ]);

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }
    try {
      window.localStorage.setItem(
        "browserAutomation.replayChromeTheme",
        replayChromeTheme,
      );
    } catch (err) {
      logger.warn(
        "Failed to persist replay chrome theme",
        { component: "ExecutionViewer" },
        err,
      );
    }
  }, [replayChromeTheme]);

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }
    try {
      window.localStorage.setItem(
        "browserAutomation.replayBackgroundTheme",
        replayBackgroundTheme,
      );
    } catch (err) {
      logger.warn(
        "Failed to persist replay background theme",
        { component: "ExecutionViewer" },
        err,
      );
    }
  }, [replayBackgroundTheme]);

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }
    try {
      window.localStorage.setItem(
        "browserAutomation.replayCursorTheme",
        replayCursorTheme,
      );
    } catch (err) {
      logger.warn(
        "Failed to persist replay cursor theme",
        { component: "ExecutionViewer" },
        err,
      );
    }
  }, [replayCursorTheme]);

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }
    try {
      window.localStorage.setItem(
        "browserAutomation.replayCursorInitialPosition",
        replayCursorInitialPosition,
      );
    } catch (err) {
      logger.warn(
        "Failed to persist replay cursor initial position",
        { component: "ExecutionViewer" },
        err,
      );
    }
  }, [replayCursorInitialPosition]);

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }
    try {
      window.localStorage.setItem(
        "browserAutomation.replayCursorClickAnimation",
        replayCursorClickAnimation,
      );
    } catch (err) {
      logger.warn(
        "Failed to persist replay cursor click animation",
        { component: "ExecutionViewer" },
        err,
      );
    }
  }, [replayCursorClickAnimation]);

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }
    try {
      window.localStorage.setItem(
        "browserAutomation.replayCursorScale",
        replayCursorScale.toFixed(2),
      );
    } catch (err) {
      logger.warn(
        "Failed to persist replay cursor scale",
        { component: "ExecutionViewer" },
        err,
      );
    }
  }, [replayCursorScale]);

  useEffect(() => {
    if (!isBackgroundMenuOpen) {
      return;
    }
    if (typeof document === "undefined") {
      return;
    }
    const handlePointerDown = (event: MouseEvent) => {
      const target = event.target as Node | null;
      if (!target) {
        return;
      }
      if (
        backgroundSelectorRef.current &&
        !backgroundSelectorRef.current.contains(target)
      ) {
        setIsBackgroundMenuOpen(false);
      }
    };
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setIsBackgroundMenuOpen(false);
      }
    };
    document.addEventListener("mousedown", handlePointerDown);
    document.addEventListener("keydown", handleKeyDown);
    return () => {
      document.removeEventListener("mousedown", handlePointerDown);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [isBackgroundMenuOpen]);

  useEffect(() => {
    if (!isCursorMenuOpen) {
      return;
    }
    if (typeof document === "undefined") {
      return;
    }
    const handlePointerDown = (event: MouseEvent) => {
      const target = event.target as Node | null;
      if (!target) {
        return;
      }
      if (
        cursorSelectorRef.current &&
        !cursorSelectorRef.current.contains(target)
      ) {
        setIsCursorMenuOpen(false);
      }
    };
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setIsCursorMenuOpen(false);
      }
    };
    document.addEventListener("mousedown", handlePointerDown);
    document.addEventListener("keydown", handleKeyDown);
    return () => {
      document.removeEventListener("mousedown", handlePointerDown);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [isCursorMenuOpen]);

  useEffect(() => {
    if (!isCursorPositionMenuOpen) {
      return;
    }
    if (typeof document === "undefined") {
      return;
    }
    const handlePointerDown = (event: MouseEvent) => {
      const target = event.target as Node | null;
      if (!target) {
        return;
      }
      if (
        cursorPositionSelectorRef.current &&
        !cursorPositionSelectorRef.current.contains(target)
      ) {
        setIsCursorPositionMenuOpen(false);
      }
    };
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setIsCursorPositionMenuOpen(false);
      }
    };
    document.addEventListener("mousedown", handlePointerDown);
    document.addEventListener("keydown", handleKeyDown);
    return () => {
      document.removeEventListener("mousedown", handlePointerDown);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [isCursorPositionMenuOpen]);

  useEffect(() => {
    if (!isCursorClickAnimationMenuOpen) {
      return;
    }
    if (typeof document === "undefined") {
      return;
    }
    const handlePointerDown = (event: MouseEvent) => {
      const target = event.target as Node | null;
      if (!target) {
        return;
      }
      if (
        cursorClickAnimationSelectorRef.current &&
        !cursorClickAnimationSelectorRef.current.contains(target)
      ) {
        setIsCursorClickAnimationMenuOpen(false);
      }
    };
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setIsCursorClickAnimationMenuOpen(false);
      }
    };
    document.addEventListener("mousedown", handlePointerDown);
    document.addEventListener("keydown", handleKeyDown);
    return () => {
      document.removeEventListener("mousedown", handlePointerDown);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [isCursorClickAnimationMenuOpen]);

  useEffect(() => {
    if (
      !hasAutoSwitchedToReplay &&
      execution.timeline &&
      execution.timeline.length > 0
    ) {
      setActiveTab("replay");
      setHasAutoSwitchedToReplay(true);
    }
  }, [execution.timeline, hasAutoSwitchedToReplay]);

  const derivedHeartbeatTimestamp = useMemo(() => {
    if (execution.lastHeartbeat?.timestamp) {
      return execution.lastHeartbeat.timestamp;
    }
    const frames = execution.timeline ?? [];
    for (let idx = frames.length - 1; idx >= 0; idx -= 1) {
      const frame = frames[idx];
      const rawTimestamp =
        frame.completed_at ||
        frame.completedAt ||
        frame.started_at ||
        frame.startedAt;
      if (rawTimestamp) {
        const parsed = new Date(rawTimestamp);
        if (!Number.isNaN(parsed.getTime())) {
          return parsed;
        }
      }
    }
    return undefined;
  }, [execution.lastHeartbeat, execution.timeline]);

  const heartbeatAgeSeconds = useMemo(() => {
    if (!derivedHeartbeatTimestamp) {
      return null;
    }
    const age = (Date.now() - derivedHeartbeatTimestamp.getTime()) / 1000;
    return age < 0 ? 0 : age;
  }, [derivedHeartbeatTimestamp]);

  const inStepSeconds =
    execution.lastHeartbeat?.elapsedMs != null
      ? Math.max(0, execution.lastHeartbeat.elapsedMs / 1000)
      : null;

  const formatSeconds = (value: number) => {
    if (Number.isNaN(value) || !Number.isFinite(value)) {
      return "0s";
    }
    if (value >= 10) {
      return `${Math.round(value)}s`;
    }
    return `${value.toFixed(1)}s`;
  };

  const heartbeatAgeLabel =
    heartbeatAgeSeconds == null
      ? null
      : heartbeatAgeSeconds < 0.75
        ? "just now"
        : `${formatSeconds(heartbeatAgeSeconds)} ago`;

  const inStepLabel =
    inStepSeconds != null ? formatSeconds(inStepSeconds) : null;

  type HeartbeatState = "idle" | "awaiting" | "healthy" | "delayed" | "stalled";

  const heartbeatState: HeartbeatState = useMemo(() => {
    const isRunning = execution.status === "running";
    const hasHeartbeat = Boolean(
      execution.lastHeartbeat || derivedHeartbeatTimestamp,
    );
    if (!hasHeartbeat) {
      return isRunning ? "awaiting" : "idle";
    }
    if (heartbeatAgeSeconds == null) {
      return "awaiting";
    }
    if (heartbeatAgeSeconds >= HEARTBEAT_STALL_SECONDS) {
      return "stalled";
    }
    if (heartbeatAgeSeconds >= HEARTBEAT_WARN_SECONDS) {
      return "delayed";
    }
    return "healthy";
  }, [
    execution.status,
    execution.lastHeartbeat,
    heartbeatAgeSeconds,
    derivedHeartbeatTimestamp,
  ]);

  const heartbeatDescriptor = useMemo(() => {
    const baseDescriptor = (() => {
      switch (heartbeatState) {
        case "idle":
          if (execution.status === "running") {
            return null;
          }
          return {
            tone: "awaiting" as const,
            iconClass: "text-amber-400",
            textClass: "text-amber-200/90",
            label: "No heartbeats recorded for this run",
          };
        case "awaiting":
          return {
            tone: "awaiting" as const,
            iconClass: "text-amber-400",
            textClass: "text-amber-200/90",
            label: "Awaiting first heartbeat…",
          };
        case "healthy":
          return {
            tone: "healthy" as const,
            iconClass: "text-blue-400",
            textClass: "text-blue-200",
            label: `Heartbeat ${heartbeatAgeLabel ?? "just now"}`,
          };
        case "delayed":
          return {
            tone: "delayed" as const,
            iconClass: "text-amber-400",
            textClass: "text-amber-200",
            label: `Heartbeat delayed (${formatSeconds(heartbeatAgeSeconds ?? 0)} since last update)`,
          };
        case "stalled":
          return {
            tone: "stalled" as const,
            iconClass: "text-red-400",
            textClass: "text-red-200",
            label: `Heartbeat stalled (${formatSeconds(heartbeatAgeSeconds ?? 0)} without update)`,
          };
        default:
          return null;
      }
    })();

    if (!baseDescriptor) {
      return null;
    }

    const labelWithSource = execution.lastHeartbeat
      ? baseDescriptor.label
      : `${baseDescriptor.label} (timeline activity)`;

    if (execution.status !== "running") {
      return {
        ...baseDescriptor,
        label: `${labelWithSource} • Final heartbeat snapshot`,
      };
    }

    return {
      ...baseDescriptor,
      label: labelWithSource,
    };
  }, [
    heartbeatState,
    heartbeatAgeLabel,
    heartbeatAgeSeconds,
    execution.status,
    execution.lastHeartbeat,
  ]);

  const statusMessage = useMemo(() => {
    const label =
      typeof execution.currentStep === "string"
        ? execution.currentStep.trim()
        : "";
    if (label.length > 0) {
      return label;
    }
    switch (execution.status) {
      case "pending":
        return "Pending...";
      case "running":
        return "Running...";
      case "completed":
        return "Completed";
      case "failed":
        return execution.error ? "Failed" : "Failed";
      case "cancelled":
        return "Cancelled";
      default:
        return "Initializing...";
    }
  }, [execution.currentStep, execution.status, execution.error]);

  const isRunning = execution.status === "running";
  const isFailed = execution.status === "failed";
  const isCancelled = execution.status === "cancelled";
  const canRestart =
    Boolean(execution.workflowId) && execution.status !== "running";

  const timelineForReplay = useMemo(() => {
    return (execution.timeline ?? []).filter((frame) => {
      if (!frame) {
        return false;
      }
      const type = (
        typeof frame.step_type === "string"
          ? frame.step_type
          : typeof frame.stepType === "string"
            ? frame.stepType
            : ""
      ).toLowerCase();
      return type !== "screenshot";
    });
  }, [execution.timeline]);

  const replayFrames = useMemo<ReplayFrame[]>(() => {
    return timelineForReplay.map((frame: TimelineFrame, index: number) => {
      const screenshotData = frame?.screenshot ?? undefined;
      const screenshotUrl = resolveUrl(screenshotData?.url);
      const thumbnailUrl = resolveUrl(screenshotData?.thumbnail_url);

      const focused = frame?.focused_element ?? frame?.focusedElement;
      const focusedRaw = (focused ?? undefined) as
        | Record<string, unknown>
        | undefined;
      const focusedBoundingBox = toBoundingBox(
        (focusedRaw?.bounding_box as unknown) ??
          (focusedRaw?.boundingBox as unknown),
      );
      const normalizedFocusedBoundingBox = focusedBoundingBox ?? undefined;
      const totalDuration = toNumber(
        frame?.total_duration_ms ?? frame?.totalDurationMs,
      );
      const retryAttempt = toNumber(
        frame?.retry_attempt ?? frame?.retryAttempt,
      );
      const retryMaxAttempts = toNumber(
        frame?.retry_max_attempts ?? frame?.retryMaxAttempts,
      );
      const retryConfigured = toNumber(
        frame?.retry_configured ?? frame?.retryConfigured,
      );
      const retryDelayMs = toNumber(
        frame?.retry_delay_ms ?? frame?.retryDelayMs,
      );
      const retryBackoffFactor = toNumber(
        frame?.retry_backoff_factor ?? frame?.retryBackoffFactor,
      );
      const retryHistory = mapRetryHistory(
        frame?.retry_history ?? frame?.retryHistory,
      );
      const domSnapshotArtifact = Array.isArray(frame?.artifacts)
        ? frame.artifacts.find(
            (artifact: TimelineArtifact) => artifact?.type === "dom_snapshot",
          )
        : undefined;
      const domSnapshotHtml =
        domSnapshotArtifact?.payload &&
        typeof domSnapshotArtifact.payload === "object"
          ? (() => {
              const payload = domSnapshotArtifact.payload as Record<
                string,
                unknown
              >;
              const html = payload?.html;
              return typeof html === "string" ? html : undefined;
            })()
          : undefined;
      const domSnapshotPreview =
        typeof (frame?.dom_snapshot_preview ?? frame?.domSnapshotPreview) ===
        "string"
          ? (frame.dom_snapshot_preview ?? frame.domSnapshotPreview)
          : undefined;
      const domSnapshotArtifactId =
        typeof (
          frame?.dom_snapshot_artifact_id ?? frame?.domSnapshotArtifactId
        ) === "string"
          ? (frame.dom_snapshot_artifact_id ?? frame.domSnapshotArtifactId)
          : typeof domSnapshotArtifact?.id === "string"
            ? domSnapshotArtifact.id
            : undefined;

      const mappedFrame: ReplayFrame = {
        id:
          screenshotData?.artifact_id ||
          frame?.timeline_artifact_id ||
          `frame-${index}`,
        stepIndex:
          typeof frame?.step_index === "number" ? frame.step_index : index,
        nodeId: typeof frame?.node_id === "string" ? frame.node_id : undefined,
        stepType:
          typeof frame?.step_type === "string" ? frame.step_type : undefined,
        status: typeof frame?.status === "string" ? frame.status : undefined,
        success: Boolean(frame?.success),
        durationMs: toNumber(frame?.duration_ms ?? frame?.durationMs),
        totalDurationMs: totalDuration,
        progress: toNumber(frame?.progress),
        finalUrl:
          typeof frame?.final_url === "string"
            ? frame.final_url
            : typeof frame?.finalUrl === "string"
              ? frame.finalUrl
              : undefined,
        error: typeof frame?.error === "string" ? frame.error : undefined,
        extractedDataPreview:
          frame?.extracted_data_preview ?? frame?.extractedDataPreview,
        consoleLogCount: toNumber(
          frame?.console_log_count ?? frame?.consoleLogCount,
        ),
        networkEventCount: toNumber(
          frame?.network_event_count ?? frame?.networkEventCount,
        ),
        screenshot: screenshotData
          ? {
              artifactId: screenshotData.artifact_id || `artifact-${index}`,
              url: screenshotUrl,
              thumbnailUrl,
              width: toNumber(screenshotData.width),
              height: toNumber(screenshotData.height),
              contentType:
                typeof screenshotData.content_type === "string"
                  ? screenshotData.content_type
                  : undefined,
              sizeBytes: toNumber(screenshotData.size_bytes),
            }
          : undefined,
        highlightRegions: mapRegions(
          frame?.highlight_regions ?? frame?.highlightRegions,
        ),
        maskRegions: mapRegions(frame?.mask_regions ?? frame?.maskRegions),
        focusedElement:
          focused || normalizedFocusedBoundingBox
            ? {
                selector:
                  typeof focused?.selector === "string"
                    ? focused.selector
                    : undefined,
                boundingBox: normalizedFocusedBoundingBox,
              }
            : null,
        elementBoundingBox:
          toBoundingBox(
            frame?.element_bounding_box ?? frame?.elementBoundingBox,
          ) ?? null,
        clickPosition:
          toPoint(frame?.click_position ?? frame?.clickPosition) ?? null,
        cursorTrail: mapTrail(frame?.cursor_trail ?? frame?.cursorTrail),
        zoomFactor: toNumber(frame?.zoom_factor ?? frame?.zoomFactor),
        assertion: mapAssertion(frame?.assertion) ?? undefined,
        retryAttempt,
        retryMaxAttempts,
        retryConfigured,
        retryDelayMs,
        retryBackoffFactor,
        retryHistory,
        domSnapshotHtml,
        domSnapshotPreview,
        domSnapshotArtifactId,
      };

      const hasScreenshot = Boolean(
        mappedFrame.screenshot?.url || mappedFrame.screenshot?.thumbnailUrl,
      );
      return hasScreenshot
        ? mappedFrame
        : { ...mappedFrame, screenshot: undefined };
    });
  }, [timelineForReplay]);

  const decoratedMovieSpec = useMemo<ReplayMovieSpec | null>(() => {
    if (!movieSpec) {
      return null;
    }
    const cursorSpec = movieSpec.cursor ?? {};
    const decor = movieSpec.decor ?? {};
    const motion = movieSpec.cursor_motion ?? {};
    return {
      ...movieSpec,
      cursor: {
        ...cursorSpec,
        scale: replayCursorScale,
        initial_position: replayCursorInitialPosition,
        click_animation: replayCursorClickAnimation,
      },
      decor: {
        ...decor,
        chrome_theme: replayChromeTheme,
        background_theme: replayBackgroundTheme,
        cursor_theme: replayCursorTheme,
        cursor_initial_position: replayCursorInitialPosition,
        cursor_click_animation: replayCursorClickAnimation,
        cursor_scale: replayCursorScale,
      },
      cursor_motion: {
        ...motion,
        initial_position: replayCursorInitialPosition,
        click_animation: replayCursorClickAnimation,
        cursor_scale: replayCursorScale,
      },
    };
  }, [
    movieSpec,
    replayBackgroundTheme,
    replayChromeTheme,
    replayCursorTheme,
    replayCursorInitialPosition,
    replayCursorClickAnimation,
    replayCursorScale,
  ]);

  const dimensionPresetOptions = useMemo(
    () => [
      {
        id: "spec" as ExportDimensionPreset,
        label: `Scenario (${specCanvasWidth}×${specCanvasHeight})`,
        width: specCanvasWidth,
        height: specCanvasHeight,
        description: "Match recorded canvas",
      },
      {
        id: "1080p" as ExportDimensionPreset,
        label: DIMENSION_PRESET_CONFIG["1080p"].label,
        width: DIMENSION_PRESET_CONFIG["1080p"].width,
        height: DIMENSION_PRESET_CONFIG["1080p"].height,
        description: "Great for polished exports",
      },
      {
        id: "720p" as ExportDimensionPreset,
        label: DIMENSION_PRESET_CONFIG["720p"].label,
        width: DIMENSION_PRESET_CONFIG["720p"].width,
        height: DIMENSION_PRESET_CONFIG["720p"].height,
        description: "Smaller file size",
      },
      {
        id: "custom" as ExportDimensionPreset,
        label: "Custom size",
        width: Number.parseInt(customWidthInput, 10) || DEFAULT_EXPORT_WIDTH,
        height: Number.parseInt(customHeightInput, 10) || DEFAULT_EXPORT_HEIGHT,
        description: "Set explicit pixel dimensions",
      },
    ],
    [customHeightInput, customWidthInput, specCanvasHeight, specCanvasWidth],
  );

  const selectedDimensions = useMemo(() => {
    switch (dimensionPreset) {
      case "custom": {
        const parsedWidth = Number.parseInt(customWidthInput, 10);
        const parsedHeight = Number.parseInt(customHeightInput, 10);
        return {
          width:
            Number.isFinite(parsedWidth) && parsedWidth > 0
              ? parsedWidth
              : DEFAULT_EXPORT_WIDTH,
          height:
            Number.isFinite(parsedHeight) && parsedHeight > 0
              ? parsedHeight
              : DEFAULT_EXPORT_HEIGHT,
        };
      }
      case "1080p":
      case "720p": {
        const preset = DIMENSION_PRESET_CONFIG[dimensionPreset];
        return { width: preset.width, height: preset.height };
      }
      case "spec":
      default:
        return { width: specCanvasWidth, height: specCanvasHeight };
    }
  }, [
    customHeightInput,
    customWidthInput,
    dimensionPreset,
    specCanvasHeight,
    specCanvasWidth,
  ]);

  const preparedMovieSpec = useMemo<ReplayMovieSpec | null>(() => {
    const base = decoratedMovieSpec ?? movieSpec;
    if (!base) {
      return null;
    }
    const cloned = JSON.parse(JSON.stringify(base)) as ReplayMovieSpec;
    const width = selectedDimensions.width;
    const height = selectedDimensions.height;

    const basePresentation: ReplayMoviePresentation | undefined =
      cloned.presentation ?? base.presentation;

    const baseCanvasWidth =
      basePresentation?.canvas?.width && basePresentation.canvas.width > 0
        ? basePresentation.canvas.width
        : width;
    const baseCanvasHeight =
      basePresentation?.canvas?.height && basePresentation.canvas.height > 0
        ? basePresentation.canvas.height
        : height;

    const scaleX = baseCanvasWidth > 0 ? width / baseCanvasWidth : 1;
    const scaleY = baseCanvasHeight > 0 ? height / baseCanvasHeight : 1;

    const scaleDimensions = (
      dims?: { width?: number; height?: number },
      fallback?: { width: number; height: number },
    ): { width: number; height: number } => {
      const fallbackWidth = fallback?.width ?? baseCanvasWidth;
      const fallbackHeight = fallback?.height ?? baseCanvasHeight;
      const sourceWidth =
        dims?.width && dims.width > 0 ? dims.width : fallbackWidth;
      const sourceHeight =
        dims?.height && dims.height > 0 ? dims.height : fallbackHeight;
      return {
        width: Math.round(sourceWidth * scaleX),
        height: Math.round(sourceHeight * scaleY),
      };
    };

    const scaleFrameRect = (
      rect?: ReplayMovieFrameRect | null,
    ): ReplayMovieFrameRect => {
      const source: ReplayMovieFrameRect = rect
        ? { ...rect }
        : {
            x: 0,
            y: 0,
            width: basePresentation?.browser_frame?.width ?? baseCanvasWidth,
            height: basePresentation?.browser_frame?.height ?? baseCanvasHeight,
            radius: basePresentation?.browser_frame?.radius ?? 24,
          };
      return {
        x: Math.round((source.x ?? 0) * scaleX),
        y: Math.round((source.y ?? 0) * scaleY),
        width: Math.round((source.width ?? baseCanvasWidth) * scaleX),
        height: Math.round((source.height ?? baseCanvasHeight) * scaleY),
        radius: source.radius ?? basePresentation?.browser_frame?.radius ?? 24,
      };
    };

    const presentation: ReplayMoviePresentation = {
      canvas: {
        width,
        height,
      },
      viewport: scaleDimensions(basePresentation?.viewport, {
        width: basePresentation?.viewport?.width ?? baseCanvasWidth,
        height: basePresentation?.viewport?.height ?? baseCanvasHeight,
      }),
      browser_frame: scaleFrameRect(basePresentation?.browser_frame),
      device_scale_factor:
        basePresentation?.device_scale_factor &&
        basePresentation.device_scale_factor > 0
          ? basePresentation.device_scale_factor
          : 1,
    };

    cloned.presentation = presentation;

    if (Array.isArray(cloned.frames)) {
      cloned.frames = cloned.frames.map((frame) => ({
        ...frame,
        viewport: scaleDimensions(frame.viewport, {
          width:
            frame.viewport?.width ??
            basePresentation?.viewport?.width ??
            baseCanvasWidth,
          height:
            frame.viewport?.height ??
            basePresentation?.viewport?.height ??
            baseCanvasHeight,
        }),
      }));
    }
    return cloned;
  }, [
    decoratedMovieSpec,
    movieSpec,
    selectedDimensions.height,
    selectedDimensions.width,
  ]);

  const activeMovieSpec = preparedMovieSpec ?? decoratedMovieSpec ?? movieSpec;

  useEffect(() => {
    if (!activeMovieSpec) {
      return;
    }
    setPreviewMetrics((current) => {
      const frameCount =
        activeMovieSpec.summary?.frame_count ??
        activeMovieSpec.frames?.length ??
        current.capturedFrames;
      const assetCount =
        Array.isArray(activeMovieSpec.assets) &&
        activeMovieSpec.assets.length >= 0
          ? activeMovieSpec.assets.length
          : current.assetCount;
      const totalDuration =
        activeMovieSpec.summary?.total_duration_ms ??
        activeMovieSpec.playback?.duration_ms ??
        current.totalDurationMs;
      return {
        capturedFrames: frameCount,
        assetCount,
        totalDurationMs: totalDuration,
      };
    });
    const specIdFromSpec = activeMovieSpec.execution?.execution_id;
    if (specIdFromSpec && specIdFromSpec !== activeSpecId) {
      setActiveSpecId(specIdFromSpec);
    }
  }, [activeMovieSpec, activeSpecId]);

  const firstFramePreviewUrl = useMemo(() => {
    const frame = replayFrames[0];
    const screenshotUrl = frame?.screenshot?.url
      ? (resolveUrl(frame.screenshot.url) ?? frame.screenshot.url)
      : null;
    if (screenshotUrl) {
      return screenshotUrl;
    }
    const specFrame = activeMovieSpec?.frames?.[0];
    if (specFrame?.screenshot_asset_id && activeMovieSpec?.assets) {
      const asset = activeMovieSpec.assets.find(
        (candidate) => candidate?.id === specFrame.screenshot_asset_id,
      );
      if (asset?.source) {
        return resolveUrl(asset.source) ?? asset.source;
      }
    }
    return null;
  }, [activeMovieSpec, replayFrames]);

  const firstFrameLabel = useMemo(() => {
    const frame = replayFrames[0];
    if (!frame) {
      return null;
    }
    return (
      frame.nodeId ||
      frame.stepType ||
      (typeof frame.stepIndex === "number"
        ? `Step ${frame.stepIndex + 1}`
        : null)
    );
  }, [replayFrames]);

  const hasTimeline = replayFrames.length > 0;

  const exportDialogTitleId = useId();
  const exportDialogDescriptionId = useId();
  const exportSummary =
    exportPreview?.package?.summary ?? activeMovieSpec?.summary ?? null;
  const normalizedExportStatus = exportPreview
    ? normalizePreviewStatus(exportPreview.status)
    : movieSpec
      ? "ready"
      : "";
  const exportStatusMessage = isExportPreviewLoading
    ? "Preparing replay data…"
    : describePreviewStatusMessage(
        normalizedExportStatus,
        exportPreviewError ?? exportPreview?.message,
        previewMetrics,
      );
  const movieSpecFrameCount =
    movieSpec?.frames != null ? movieSpec.frames.length : undefined;
  const previewFrameCount =
    previewMetrics.capturedFrames || previewMetrics.capturedFrames === 0
      ? previewMetrics.capturedFrames
      : undefined;
  const estimatedFrameCount =
    exportSummary?.frame_count ??
    movieSpecFrameCount ??
    previewFrameCount ??
    replayFrames.length;
  const previewDurationMs =
    previewMetrics.totalDurationMs || previewMetrics.totalDurationMs === 0
      ? previewMetrics.totalDurationMs
      : undefined;
  const specDurationMs = activeMovieSpec?.playback?.duration_ms;
  const estimatedTotalDurationMs =
    exportSummary?.total_duration_ms ??
    specDurationMs ??
    previewDurationMs ??
    null;
  const estimatedDurationSeconds =
    estimatedTotalDurationMs != null ? estimatedTotalDurationMs / 1000 : null;
  const isBinaryExport = exportFormat !== "json";

  const timelineScreenshots = useMemo(() => {
    const frames = execution.timeline ?? [];
    const items: Screenshot[] = [];
    frames.forEach((frame: TimelineFrame, index: number) => {
      const resolved = resolveUrl(frame?.screenshot?.url);
      if (!resolved) {
        return;
      }
      items.push({
        id: frame?.screenshot?.artifact_id || `timeline-${index}`,
        url: resolved,
        stepName:
          frame?.node_id ||
          frame?.step_type ||
          (typeof frame?.step_index === "number"
            ? `Step ${frame.step_index + 1}`
            : "Step"),
        timestamp: frame?.started_at ? new Date(frame.started_at) : new Date(),
      });
    });
    return items;
  }, [execution.timeline]);

  const backgroundOptionsByGroup = useMemo(() => {
    const base: Record<BackgroundOption["kind"], BackgroundOption[]> = {
      abstract: [],
      solid: [],
      minimal: [],
      geometric: [],
    };
    for (const option of REPLAY_BACKGROUND_OPTIONS) {
      base[option.kind].push(option);
    }
    return base;
  }, []);

  const cursorOptionsByGroup = useMemo(() => {
    const base: Record<CursorOption["group"], CursorOption[]> = {
      hidden: [],
      halo: [],
      arrow: [],
      hand: [],
    };
    for (const option of REPLAY_CURSOR_OPTIONS) {
      base[option.group].push(option);
    }
    return base;
  }, []);

  const selectedChromeOption = useMemo(() => {
    return (
      REPLAY_CHROME_OPTIONS.find((option) => option.id === replayChromeTheme) ||
      REPLAY_CHROME_OPTIONS[0]
    );
  }, [replayChromeTheme]);

  const selectedBackgroundOption = useMemo<BackgroundOption>(() => {
    return (
      REPLAY_BACKGROUND_OPTIONS.find(
        (option) => option.id === replayBackgroundTheme,
      ) || REPLAY_BACKGROUND_OPTIONS[0]
    );
  }, [replayBackgroundTheme]);

  const selectedCursorOption = useMemo<CursorOption>(() => {
    return (
      REPLAY_CURSOR_OPTIONS.find((option) => option.id === replayCursorTheme) ||
      REPLAY_CURSOR_OPTIONS[0]
    );
  }, [replayCursorTheme]);

  const selectedCursorPositionOption = useMemo(() => {
    return (
      REPLAY_CURSOR_POSITIONS.find(
        (option) => option.id === replayCursorInitialPosition,
      ) || REPLAY_CURSOR_POSITIONS[0]
    );
  }, [replayCursorInitialPosition]);

  const selectedCursorClickAnimationOption =
    useMemo<ClickAnimationOption>(() => {
      return (
        REPLAY_CURSOR_CLICK_ANIMATION_OPTIONS.find(
          (option) => option.id === replayCursorClickAnimation,
        ) || REPLAY_CURSOR_CLICK_ANIMATION_OPTIONS[0]
      );
    }, [replayCursorClickAnimation]);

  const postToComposer = useCallback(
    (message: Record<string, unknown>) => {
      const targets: Array<{
        windowRef: MutableRefObject<Window | null>;
        originRef: MutableRefObject<string | null>;
        url: string;
        label: string;
      }> = [
        {
          windowRef: composerWindowRef,
          originRef: composerOriginRef,
          url: composerUrl,
          label: "composer",
        },
      ];
      if (composerPreviewWindowRef.current) {
        targets.push({
          windowRef: composerPreviewWindowRef,
          originRef: composerPreviewOriginRef,
          url: composerPreviewUrl,
          label: "preview-composer",
        });
      }

      targets.forEach(({ windowRef, originRef, url, label }) => {
        const targetWindow = windowRef.current;
        if (!targetWindow) {
          return;
        }
        let targetOrigin = originRef.current;
        if (!targetOrigin && typeof window !== "undefined") {
          try {
            targetOrigin = new URL(url, window.location.href).origin;
            originRef.current = targetOrigin;
          } catch (error) {
            logger.warn(
              "Failed to resolve composer origin",
              {
                component: "ExecutionViewer",
                executionId: execution.id,
                context: label,
              },
              error,
            );
          }
        }
        try {
          targetWindow.postMessage(message, targetOrigin ?? "*");
        } catch (error) {
          logger.warn(
            "Failed to post message to composer",
            { component: "ExecutionViewer", context: label },
            error,
          );
        }
      });
    },
    [composerUrl, composerPreviewUrl, execution.id],
  );

  const sendSpecToComposer = useCallback(
    (spec: ReplayMovieSpec | null) => {
      if (!spec) {
        return;
      }
      const specId =
        spec.execution?.execution_id ?? activeSpecId ?? execution.id;
      const summaryFrames =
        spec.summary?.frame_count ?? spec.frames?.length ?? 0;
      const summaryDuration =
        spec.summary?.total_duration_ms ?? spec.playback?.duration_ms ?? 0;
      const summaryAssets = Array.isArray(spec.assets) ? spec.assets.length : 0;
      postToComposer({
        type: "bas:spec:set",
        spec,
        executionId: execution.id,
        specId,
        apiBase: composerApiBase ?? undefined,
        summary: {
          frames: summaryFrames,
          totalDurationMs: summaryDuration,
          assets: summaryAssets,
        },
      });
    },
    [activeSpecId, composerApiBase, execution.id, postToComposer],
  );

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }
    const handleMessage = (event: MessageEvent) => {
      const fromComposer = event.source === composerWindowRef.current;
      const fromPreview = event.source === composerPreviewWindowRef.current;
      if (!fromComposer && !fromPreview) {
        return;
      }
      const payload = event.data;
      if (!payload || typeof payload !== "object") {
        return;
      }
      const { type } = payload as { type?: unknown };
      if (typeof type !== "string" || !type.startsWith("bas:")) {
        return;
      }
      if (type === "bas:ready") {
        if (fromComposer) {
          setIsComposerReady(true);
          setMovieSpecError(null);
          composerOriginRef.current = event.origin || composerOriginRef.current;
        } else {
          setIsPreviewComposerReady(true);
          setPreviewComposerError(null);
          composerPreviewOriginRef.current =
            event.origin || composerPreviewOriginRef.current;
        }
        if (preparedMovieSpec) {
          sendSpecToComposer(preparedMovieSpec);
        }
      } else if (type === "bas:state") {
        if (fromComposer) {
          const data = payload as { frameIndex?: unknown; progress?: unknown };
          const frameIndex =
            typeof data.frameIndex === "number" ? data.frameIndex : 0;
          const progress =
            typeof data.progress === "number" ? data.progress : 0;
          composerFrameStateRef.current = { frameIndex, progress };
        }
      } else if (type === "bas:error") {
        const data = payload as {
          message?: unknown;
          status?: unknown;
          frames?: unknown;
          assets?: unknown;
          totalDurationMs?: unknown;
          specId?: unknown;
        };
        const status = normalizePreviewStatus(
          typeof data.status === "string" ? data.status : undefined,
        );
        const message = describePreviewStatusMessage(
          status,
          typeof data.message === "string" ? data.message : undefined,
          {
            capturedFrames:
              typeof data.frames !== "undefined"
                ? coerceMetricNumber(data.frames)
                : previewMetrics.capturedFrames,
            assetCount:
              typeof data.assets !== "undefined"
                ? coerceMetricNumber(data.assets)
                : previewMetrics.assetCount,
          },
        );
        if (fromComposer) {
          setIsComposerReady(false);
          setMovieSpecError(message);
        } else {
          setIsPreviewComposerReady(false);
          setPreviewComposerError(message);
        }
        const framesValue =
          typeof data.frames !== "undefined"
            ? coerceMetricNumber(data.frames)
            : null;
        const assetsValue =
          typeof data.assets !== "undefined"
            ? coerceMetricNumber(data.assets)
            : null;
        const durationValue =
          typeof data.totalDurationMs !== "undefined"
            ? coerceMetricNumber(data.totalDurationMs)
            : null;
        if (
          framesValue !== null ||
          assetsValue !== null ||
          durationValue !== null
        ) {
          setPreviewMetrics((current) => {
            const next = { ...current };
            let changed = false;
            if (framesValue !== null) {
              next.capturedFrames = framesValue;
              changed = true;
            }
            if (assetsValue !== null) {
              next.assetCount = assetsValue;
              changed = true;
            }
            if (durationValue !== null) {
              next.totalDurationMs = durationValue;
              changed = true;
            }
            return changed ? next : current;
          });
        }
        if (typeof data.specId === "string" && data.specId.trim()) {
          setActiveSpecId(data.specId.trim());
        }
      } else if (type === "bas:error-clear") {
        if (fromComposer) {
          setMovieSpecError(null);
        } else {
          setPreviewComposerError(null);
        }
      } else if (type === "bas:metrics") {
        if (!fromComposer) {
          return;
        }
        const data = payload as {
          frames?: unknown;
          assets?: unknown;
          totalDurationMs?: unknown;
          specId?: unknown;
        };
        const framesValue =
          typeof data.frames !== "undefined"
            ? coerceMetricNumber(data.frames)
            : null;
        const assetsValue =
          typeof data.assets !== "undefined"
            ? coerceMetricNumber(data.assets)
            : null;
        const durationValue =
          typeof data.totalDurationMs !== "undefined"
            ? coerceMetricNumber(data.totalDurationMs)
            : null;
        if (
          framesValue !== null ||
          assetsValue !== null ||
          durationValue !== null
        ) {
          setPreviewMetrics((current) => {
            const next = { ...current };
            let changed = false;
            if (framesValue !== null) {
              next.capturedFrames = framesValue;
              changed = true;
            }
            if (assetsValue !== null) {
              next.assetCount = assetsValue;
              changed = true;
            }
            if (durationValue !== null) {
              next.totalDurationMs = durationValue;
              changed = true;
            }
            return changed ? next : current;
          });
        }
        if (typeof data.specId === "string" && data.specId.trim()) {
          setActiveSpecId(data.specId.trim());
        }
      }
    };
    window.addEventListener("message", handleMessage);
    return () => {
      window.removeEventListener("message", handleMessage);
    };
  }, [preparedMovieSpec, previewMetrics, sendSpecToComposer]);

  useEffect(() => {
    setIsComposerReady(false);
    composerFrameStateRef.current = { frameIndex: 0, progress: 0 };
    setHasAutoOpenedScreenshots(false);
  }, [execution.id]);

  useEffect(() => {
    if (!isComposerReady || !preparedMovieSpec) {
      return;
    }
    sendSpecToComposer(preparedMovieSpec);
  }, [isComposerReady, preparedMovieSpec, sendSpecToComposer]);

  useEffect(() => {
    if (!isCustomizationCollapsed) {
      return;
    }
    setIsBackgroundMenuOpen(false);
    setIsCursorMenuOpen(false);
    setIsCursorPositionMenuOpen(false);
    setIsCursorClickAnimationMenuOpen(false);
  }, [isCustomizationCollapsed]);

  const handleBackgroundSelect = useCallback(
    (option: BackgroundOption) => {
      setReplayBackgroundTheme(option.id);
      setIsBackgroundMenuOpen(false);
    },
    [setReplayBackgroundTheme],
  );

  const handleCursorThemeSelect = useCallback((value: ReplayCursorTheme) => {
    setReplayCursorTheme(value);
    setIsCursorMenuOpen(false);
  }, []);

  const handleCursorPositionSelect = useCallback(
    (value: ReplayCursorInitialPosition) => {
      setReplayCursorInitialPosition(value);
      setIsCursorPositionMenuOpen(false);
    },
    [],
  );

  const handleCursorClickAnimationSelect = useCallback(
    (value: ReplayCursorClickAnimation) => {
      setReplayCursorClickAnimation(value);
      setIsCursorClickAnimationMenuOpen(false);
    },
    [],
  );

  const handleCursorScaleChange = useCallback((value: number) => {
    if (!Number.isFinite(value)) {
      return;
    }
    const clamped = Math.min(
      CURSOR_SCALE_MAX,
      Math.max(CURSOR_SCALE_MIN, value),
    );
    setReplayCursorScale(clamped);
  }, []);

  const screenshots =
    timelineScreenshots.length > 0
      ? timelineScreenshots
      : execution.screenshots;

  useEffect(() => {
    if (screenshots.length === 0) {
      return;
    }
    const alreadySelected =
      selectedScreenshot &&
      screenshots.some((shot) => shot.id === selectedScreenshot.id);
    if (!alreadySelected) {
      setSelectedScreenshot(screenshots[screenshots.length - 1]);
    }
    if (
      screenshots.length > 0 &&
      !hasAutoOpenedScreenshots &&
      activeTab !== "screenshots"
    ) {
      setActiveTab("screenshots");
      setHasAutoOpenedScreenshots(true);
    }
  }, [screenshots, selectedScreenshot, hasAutoOpenedScreenshots, activeTab]);

  useEffect(() => {
    if (!selectedScreenshot) {
      return;
    }
    const element = screenshotRefs.current[selectedScreenshot.id];
    if (element) {
      element.scrollIntoView({
        behavior: "smooth",
        block: "start",
        inline: "nearest",
      });
    }
  }, [selectedScreenshot]);

  useEffect(() => {
    if (
      activeTab === "screenshots" &&
      screenshots.length === 0 &&
      replayFrames.length > 0
    ) {
      setActiveTab("replay");
    }
    if (activeTab === "screenshots") {
      setHasAutoOpenedScreenshots(true);
    }
  }, [activeTab, screenshots.length, replayFrames.length]);

  useEffect(() => {
    let interval: number | undefined;
    if (execution.status === "running") {
      interval = window.setInterval(() => {
        void refreshTimeline(execution.id);
      }, 2000);
    } else if (
      execution.status === "completed" ||
      execution.status === "failed" ||
      execution.status === "cancelled"
    ) {
      void refreshTimeline(execution.id);
    }

    return () => {
      if (interval) {
        window.clearInterval(interval);
      }
    };
  }, [execution.status, execution.id, refreshTimeline]);

  const handleStop = useCallback(async () => {
    if (!isRunning || isStopping) {
      return;
    }
    setIsStopping(true);
    try {
      await stopExecution(execution.id);
      toast.success("Execution stopped");
      await refreshTimeline(execution.id);
    } catch (error) {
      const message =
        error instanceof Error ? error.message : "Failed to stop execution";
      toast.error(message);
    } finally {
      setIsStopping(false);
    }
  }, [execution.id, isRunning, isStopping, stopExecution, refreshTimeline]);

  const handleRestart = useCallback(async () => {
    if (!canRestart || isRestarting) {
      return;
    }
    if (!execution.workflowId) {
      toast.error("Workflow identifier missing for restart");
      return;
    }
    setIsRestarting(true);
    try {
      await startExecution(execution.workflowId);
      toast.success("Workflow restarted");
    } catch (error) {
      const message =
        error instanceof Error ? error.message : "Failed to restart workflow";
      toast.error(message);
    } finally {
      setIsRestarting(false);
    }
  }, [canRestart, execution.workflowId, isRestarting, startExecution]);

  const handleExecutionSwitch = useCallback(
    async (candidate: { id: string }) => {
      if (!candidate?.id) {
        return;
      }
      if (candidate.id === execution.id) {
        setActiveTab("replay");
        return;
      }
      setIsSwitchingExecution(true);
      try {
        await loadExecution(candidate.id);
        setActiveTab("replay");
      } catch (error) {
        const message =
          error instanceof Error ? error.message : "Failed to load execution";
        toast.error(message);
      } finally {
        setIsSwitchingExecution(false);
      }
    },
    [execution.id, loadExecution],
  );

  const handleOpenExportDialog = useCallback(() => {
    if (replayFrames.length === 0) {
      toast.error("Replay not ready to export yet");
      return;
    }
    setExportFormat("mp4");
    setExportFileStem(defaultExportFileStem);
    setUseNativeFilePicker(false);
    setIsExportDialogOpen(true);
  }, [defaultExportFileStem, replayFrames.length]);

  const handleCloseExportDialog = useCallback(() => {
    setIsExportDialogOpen(false);
  }, []);

  const handleConfirmExport = useCallback(async () => {
    if (exportFormat !== "json") {
      if (replayFrames.length === 0) {
        toast.error("Replay not ready to export yet");
        return;
      }

      setIsExporting(true);
      try {
        const { API_URL } = await getConfig();
        const baselineSpec = exportPreview?.package ?? movieSpec;
        const exportSpec = preparedMovieSpec ?? baselineSpec;
        const hasExportFrames =
          exportSpec && Array.isArray(exportSpec.frames)
            ? exportSpec.frames.length > 0
            : false;
        const requestPayload: Record<string, unknown> = {
          format: exportFormat,
          file_name: finalFileName,
        };
        if (hasExportFrames && exportSpec) {
          requestPayload.movie_spec = exportSpec;
        } else {
          requestPayload.overrides = {
            theme_preset: {
              chrome_theme: replayChromeTheme,
              background_theme: replayBackgroundTheme,
            },
            cursor_preset: {
              theme: replayCursorTheme,
              initial_position: replayCursorInitialPosition,
              scale: Number.isFinite(replayCursorScale) ? replayCursorScale : 1,
              click_animation: replayCursorClickAnimation,
            },
          } satisfies Record<string, unknown>;
        }

        const response = await fetch(
          `${API_URL}/executions/${execution.id}/export`,
          {
            method: "POST",
            headers: {
              "Content-Type": "application/json",
              Accept: exportFormat === "gif" ? "image/gif" : "video/mp4",
            },
            body: JSON.stringify(requestPayload),
          },
        );

        if (!response.ok) {
          const text = await response.text();
          throw new Error(text || `Export request failed (${response.status})`);
        }

        const blob = await response.blob();
        const downloadUrl = URL.createObjectURL(blob);
        const anchor = document.createElement("a");
        anchor.href = downloadUrl;
        anchor.download = finalFileName;
        document.body.appendChild(anchor);
        anchor.click();
        document.body.removeChild(anchor);
        URL.revokeObjectURL(downloadUrl);

        // Create export record in the library
        const exportName = exportFileStem || `${workflowName} Export`;
        const createdExport = await createExport({
          executionId: execution.id,
          workflowId: execution.workflowId,
          name: exportName,
          format: exportFormat as 'mp4' | 'gif' | 'json' | 'html',
          settings: {
            chromeTheme: replayChromeTheme,
            backgroundTheme: replayBackgroundTheme,
            cursorTheme: replayCursorTheme,
            cursorScale: replayCursorScale,
          },
          fileSizeBytes: blob.size,
          durationMs: exportPreview?.totalDurationMs,
          frameCount: exportPreview?.capturedFrameCount,
        });

        if (createdExport) {
          setLastCreatedExport(createdExport);
          setShowExportSuccess(true);
        }
        toast.success("Replay export ready");
        setIsExportDialogOpen(false);
      } catch (error) {
        const message =
          error instanceof Error ? error.message : "Failed to export replay";
        toast.error(message);
      } finally {
        setIsExporting(false);
      }
      return;
    }

    if (replayFrames.length === 0) {
      toast.error("Replay not ready to export yet");
      return;
    }

    if (useNativeFilePicker && !supportsFileSystemAccess) {
      toast.error(
        "This browser does not support choosing a destination. Disable “Choose save location”.",
      );
      return;
    }

    setIsExporting(true);
    try {
      const payload = preparedMovieSpec ?? exportPreview?.package ?? movieSpec;
      if (!payload) {
        toast.error("Replay not ready to export yet");
        setIsExporting(false);
        return;
      }

      const blob = new Blob([JSON.stringify(payload, null, 2)], {
        type: "application/json",
      });

      if (useNativeFilePicker && supportsFileSystemAccess) {
        try {
          const picker = await (
            window as typeof window & {
              showSaveFilePicker?: (
                options?: SaveFilePickerOptions,
              ) => Promise<FileSystemFileHandle>;
            }
          ).showSaveFilePicker?.({
            suggestedName: finalFileName,
            types: [
              {
                description: "Replay export (JSON)",
                accept: { "application/json": [".json"] },
              },
            ],
          });
          if (!picker) {
            throw new Error("Unable to open save dialog");
          }
          const writable = await picker.createWritable();
          await writable.write(blob);
          await writable.close();
          toast.success("Replay export saved");
        } catch (error) {
          if (error instanceof DOMException && error.name === "AbortError") {
            setIsExporting(false);
            return;
          }
          const message =
            error instanceof Error
              ? error.message
              : "Failed to save replay export";
          toast.error(message);
          setIsExporting(false);
          return;
        }
      } else {
        const url = URL.createObjectURL(blob);
        try {
          const anchor = document.createElement("a");
          anchor.href = url;
          anchor.download = finalFileName;
          document.body.appendChild(anchor);
          anchor.click();
          document.body.removeChild(anchor);
          toast.success("Replay download started");
        } finally {
          URL.revokeObjectURL(url);
        }
      }

      // Create export record in the library for JSON exports
      const exportName = exportFileStem || `${workflowName} Export`;
      const createdExport = await createExport({
        executionId: execution.id,
        workflowId: execution.workflowId,
        name: exportName,
        format: 'json',
        settings: {
          chromeTheme: replayChromeTheme,
          backgroundTheme: replayBackgroundTheme,
          cursorTheme: replayCursorTheme,
          cursorScale: replayCursorScale,
        },
        fileSizeBytes: blob.size,
        durationMs: exportPreview?.totalDurationMs,
        frameCount: exportPreview?.capturedFrameCount,
      });

      if (createdExport) {
        setLastCreatedExport(createdExport);
        setShowExportSuccess(true);
      }

      setIsExportDialogOpen(false);
    } catch (error) {
      const message =
        error instanceof Error ? error.message : "Failed to export replay";
      toast.error(message);
    } finally {
      setIsExporting(false);
    }
  }, [
    preparedMovieSpec,
    exportFormat,
    exportPreview,
    execution.id,
    execution.workflowId,
    workflowName,
    exportFileStem,
    finalFileName,
    movieSpec,
    replayBackgroundTheme,
    replayChromeTheme,
    replayCursorClickAnimation,
    replayCursorInitialPosition,
    replayCursorScale,
    replayCursorTheme,
    replayFrames.length,
    supportsFileSystemAccess,
    useNativeFilePicker,
    createExport,
  ]);

  const getStatusIcon = () => {
    const statusTestId = `execution-status-${execution.status}`;
    // Add both general and specific status selectors for test flexibility
    const testIds = `${selectors.executions.viewer.status} ${statusTestId}`;
    switch (execution.status) {
      case "running":
        return (
          <Loader
            size={16}
            className="animate-spin text-blue-400"
            data-testid={testIds}
          />
        );
      case "completed":
        return (
          <CheckCircle
            size={16}
            className="text-green-400"
            data-testid={testIds}
          />
        );
      case "failed":
        return (
          <XCircle
            size={16}
            className="text-red-400"
            data-testid={testIds}
          />
        );
      case "cancelled":
        return (
          <AlertTriangle
            size={16}
            className="text-yellow-400"
            data-testid={testIds}
          />
        );
      default:
        return (
          <Clock
            size={16}
            className="text-gray-400"
            data-testid={testIds}
          />
        );
    }
  };

  const getLogColor = (level: LogEntry["level"]) => {
    switch (level) {
      case "error":
        return "text-red-400";
      case "warning":
        return "text-yellow-400";
      case "success":
        return "text-green-400";
      default:
        return "text-gray-300";
    }
  };

  return (
    <div
      className="h-full flex flex-col bg-flow-node min-h-0"
      data-testid={selectors.executions.viewer.root}
    >
      <div className="flex items-center justify-between p-3 border-b border-gray-800">
        <div className="flex items-center gap-3">
          {getStatusIcon()}
          <div>
            <div className="text-sm font-medium text-white">
              Execution #{execution.id.slice(0, 8)}
            </div>
            <div
              className="text-xs text-gray-500"
              data-testid={selectors.executions.viewer.status}
            >
              {statusMessage}
            </div>
            {heartbeatDescriptor && (
              <div
                className="mt-1 flex items-center gap-2 text-[11px]"
                data-testid={selectors.heartbeat.indicator}
              >
                {heartbeatDescriptor.tone === "stalled" ? (
                  <AlertTriangle
                    size={12}
                    className={heartbeatDescriptor.iconClass}
                    data-testid={
                      heartbeatDescriptor.tone === "stalled"
                        ? selectors.heartbeat.lagWarning
                        : undefined
                    }
                  />
                ) : (
                  <Activity
                    size={12}
                    className={heartbeatDescriptor.iconClass}
                  />
                )}
                <span
                  className={heartbeatDescriptor.textClass}
                  data-testid={selectors.heartbeat.status}
                >
                  {heartbeatDescriptor.label}
                </span>
                {inStepLabel && execution.lastHeartbeat && (
                  <span
                    className={`${heartbeatDescriptor.textClass} opacity-80`}
                  >
                    • {inStepLabel} in step
                  </span>
                )}
              </div>
            )}
          </div>
        </div>

        <div className="flex items-center gap-2">
          <button
            className="toolbar-button p-1.5 text-gray-500 opacity-50 cursor-not-allowed"
            title="Pause (coming soon)"
            disabled
            aria-disabled="true"
          >
            <Pause size={14} />
          </button>
          <button
            className="toolbar-button p-1.5"
            title={
              canRestart
                ? "Re-run workflow"
                : "Stop execution before re-running"
            }
            onClick={handleRestart}
            disabled={!canRestart || isRestarting || isStopping}
            data-testid={selectors.executions.viewer.rerunButton}
          >
            {isRestarting ? (
              <Loader size={14} className="animate-spin" />
            ) : (
              <RotateCw size={14} />
            )}
          </button>
          <button
            className="toolbar-button p-1.5 disabled:text-gray-500 disabled:opacity-50 disabled:cursor-not-allowed"
            title={
              replayFrames.length === 0
                ? "Replay not ready to export"
                : "Export replay"
            }
            onClick={handleOpenExportDialog}
            disabled={replayFrames.length === 0}
            data-testid={selectors.executions.actions.exportReplayButton}
          >
            <Download size={14} />
          </button>
          <button
            className="toolbar-button p-1.5 text-red-400 disabled:text-red-400/50 disabled:cursor-not-allowed"
            title={isRunning ? "Stop execution" : "Execution not running"}
            onClick={handleStop}
            disabled={!isRunning || isStopping}
            data-testid={selectors.executions.viewer.stopButton}
          >
            {isStopping ? (
              <Loader size={14} className="animate-spin" />
            ) : (
              <Square size={14} />
            )}
          </button>
          {onClose && (
            <>
              <button
                className="toolbar-button p-1.5 ml-2 border-l border-gray-700 pl-3 text-blue-400 hover:text-blue-300"
                title="Edit workflow"
                onClick={onClose}
                data-testid={selectors.executions.viewer.editWorkflowButton}
              >
                <Pencil size={14} />
              </button>
              <button
                className="toolbar-button p-1.5"
                title="Close"
                onClick={onClose}
              >
                <X size={14} />
              </button>
            </>
          )}
        </div>
      </div>

      <div className="h-2 bg-flow-bg">
        <div
          className="h-full bg-flow-accent transition-all duration-300"
          style={{ width: `${execution.progress}%` }}
        />
      </div>

      <div className="flex border-b border-gray-800">
        <button
          data-testid={selectors.executions.tabs.replay}
          className={`flex-1 px-3 py-2 text-sm font-medium transition-colors flex items-center justify-center gap-2 ${
            activeTab === "replay"
              ? "bg-flow-bg text-white border-b-2 border-flow-accent"
              : hasTimeline
                ? "text-gray-400 hover:text-white"
                : "text-gray-500 hover:text-white/80"
          }`}
          onClick={() => setActiveTab("replay")}
        >
          <PlayCircle size={14} />
          Replay ({replayFrames.length})
        </button>
        <button
          data-testid={selectors.executions.tabs.screenshots}
          className={`flex-1 px-3 py-2 text-sm font-medium transition-colors flex items-center justify-center gap-2 ${
            activeTab === "screenshots"
              ? "bg-flow-bg text-white border-b-2 border-flow-accent"
              : "text-gray-400 hover:text-white"
          }`}
          onClick={() => setActiveTab("screenshots")}
        >
          <Image size={14} />
          Screenshots ({screenshots.length})
        </button>
        <button
          data-testid={selectors.executions.tabs.logs}
          className={`flex-1 px-3 py-2 text-sm font-medium transition-colors flex items-center justify-center gap-2 ${
            activeTab === "logs"
              ? "bg-flow-bg text-white border-b-2 border-flow-accent"
              : "text-gray-400 hover:text-white"
          }`}
          onClick={() => setActiveTab("logs")}
        >
          <Terminal size={14} />
          Logs ({execution.logs.length})
        </button>
        {showExecutionSwitcher && (
          <button
            data-testid={selectors.executions.tabs.executions}
            className={`flex-1 px-3 py-2 text-sm font-medium transition-colors flex items-center justify-center gap-2 ${
              activeTab === "executions"
                ? "bg-flow-bg text-white border-b-2 border-flow-accent"
                : "text-gray-400 hover:text-white"
            }`}
            onClick={() => setActiveTab("executions")}
          >
            {isSwitchingExecution ? (
              <Loader size={14} className="animate-spin" />
            ) : (
              <ListTree size={14} />
            )}
            Executions
          </button>
        )}
      </div>

      <div className="flex-1 overflow-hidden flex flex-col min-h-0">
        {activeTab === "replay" ? (
          <div
            className="flex-1 overflow-auto p-3 space-y-3"
            data-testid={selectors.replay.player}
          >
            {!hasTimeline && (
              <div className="rounded-lg border border-dashed border-slate-700/60 bg-slate-900/60 px-4 py-3 text-sm text-slate-200/80">
                Replay frames stream in as each action runs. Leave this tab open
                to tailor the final cut in real time.
              </div>
            )}
            {(isFailed || isCancelled) && execution.progress < 100 && (
              <div className="rounded-lg border border-rose-500/30 bg-rose-500/10 p-3 flex items-start gap-3">
                <AlertTriangle
                  size={18}
                  className="text-rose-400 flex-shrink-0 mt-0.5"
                />
                <div className="flex-1 text-sm">
                  <div className="font-medium text-rose-200 mb-1">
                    Execution {isFailed ? "Failed" : "Cancelled"} - Replay
                    Incomplete
                  </div>
                  <div className="text-rose-100/80">
                    This replay shows only {replayFrames.length} of the
                    workflow's steps. Execution{" "}
                    {isFailed ? "failed" : "was cancelled"}
                    at {execution.currentStep || "an unknown step"}.
                  </div>
                  {isFailed && executionError && (
                    <div className="mt-2 text-xs font-mono text-rose-100/70 bg-rose-950/30 px-2 py-1 rounded">
                      {executionError}
                    </div>
                  )}
                </div>
              </div>
            )}
            <div className="rounded-2xl border border-white/5 bg-slate-950/40 px-4 py-3 shadow-inner">
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div>
                  <span className="text-[11px] uppercase tracking-[0.24em] text-slate-400">
                    Replay customization
                  </span>
                  <div className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-[11px] text-slate-500">
                    <span>Chrome • {selectedChromeOption.label}</span>
                    <span>Background • {selectedBackgroundOption.label}</span>
                    <span>Cursor • {selectedCursorOption.label}</span>
                    <span>
                      Click • {selectedCursorClickAnimationOption.label}
                    </span>
                  </div>
                </div>
                <button
                  type="button"
                  onClick={() => setIsCustomizationCollapsed((prev) => !prev)}
                  className="inline-flex h-8 w-8 items-center justify-center rounded-full border border-white/10 bg-white/5 text-slate-200 transition-colors transition-transform hover:bg-white/10 focus:outline-none focus:ring-2 focus:ring-flow-accent/60 focus:ring-offset-2 focus:ring-offset-slate-900"
                  aria-expanded={!isCustomizationCollapsed}
                  aria-label={
                    isCustomizationCollapsed
                      ? "Expand replay customization"
                      : "Collapse replay customization"
                  }
                >
                  <ChevronDown
                    size={16}
                    className={clsx("transition-transform duration-200", {
                      "-rotate-180": !isCustomizationCollapsed,
                    })}
                  />
                </button>
              </div>
              {!isCustomizationCollapsed && (
                <div className="mt-4 space-y-4">
                  <div>
                    <div className="mb-2 flex flex-wrap items-center justify-between gap-2">
                      <span className="text-[11px] uppercase tracking-[0.24em] text-slate-400">
                        Browser frame
                      </span>
                      <span className="text-[11px] text-slate-500">
                        Customize the replay window
                      </span>
                    </div>
                    <div className="flex flex-wrap items-center gap-2">
                      {REPLAY_CHROME_OPTIONS.map((option) => (
                        <button
                          key={option.id}
                          type="button"
                          onClick={() => setReplayChromeTheme(option.id)}
                          title={option.subtitle}
                          className={clsx(
                            "rounded-full px-3 py-1.5 text-xs font-medium transition-all focus:outline-none focus:ring-2 focus:ring-flow-accent/70 focus:ring-offset-2 focus:ring-offset-slate-900",
                            replayChromeTheme === option.id
                              ? "bg-flow-accent text-white shadow-[0_12px_35px_rgba(59,130,246,0.45)]"
                              : "bg-slate-900/60 text-slate-300 hover:bg-slate-900/80",
                          )}
                        >
                          {option.label}
                        </button>
                      ))}
                    </div>
                  </div>

                  <div className="border-t border-white/5 pt-3 space-y-3">
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <span className="text-[11px] uppercase tracking-[0.24em] text-slate-400">
                        Background
                      </span>
                      <span className="text-[11px] text-slate-500">
                        Set the stage behind the browser
                      </span>
                    </div>
                    <div ref={backgroundSelectorRef} className="relative">
                      <button
                        type="button"
                        className={clsx(
                          "flex w-full items-center justify-between rounded-xl border px-3 py-2 text-left text-sm transition-all focus:outline-none focus:ring-2 focus:ring-flow-accent/70 focus:ring-offset-2 focus:ring-offset-slate-900",
                          isBackgroundMenuOpen
                            ? "border-flow-accent/70 bg-slate-900/80 text-white"
                            : "border-white/10 bg-slate-900/60 text-slate-200 hover:border-flow-accent/40 hover:text-white",
                        )}
                        onClick={() => {
                          setIsBackgroundMenuOpen((open) => !open);
                          setIsCursorMenuOpen(false);
                          setIsCursorPositionMenuOpen(false);
                          setIsCursorClickAnimationMenuOpen(false);
                        }}
                        aria-haspopup="menu"
                        aria-expanded={isBackgroundMenuOpen}
                      >
                        <span className="flex items-center gap-3">
                          <span
                            aria-hidden
                            className="relative h-10 w-16 overflow-hidden rounded-lg border border-white/10 shadow-inner"
                            style={selectedBackgroundOption.previewStyle}
                          >
                            {selectedBackgroundOption.previewNode}
                          </span>
                          <span className="flex flex-col text-xs leading-tight text-slate-300">
                            <span className="text-sm font-medium text-white">
                              {selectedBackgroundOption.label}
                            </span>
                            <span className="text-[11px] text-slate-400">
                              {selectedBackgroundOption.subtitle}
                            </span>
                          </span>
                        </span>
                        <ChevronDown
                          size={14}
                          className={clsx(
                            "ml-3 flex-shrink-0 text-slate-400 transition-transform duration-150",
                            isBackgroundMenuOpen ? "rotate-180 text-white" : "",
                          )}
                        />
                      </button>

                      {isBackgroundMenuOpen && (
                        <div
                          role="menu"
                          className="absolute right-0 z-30 mt-2 w-full min-w-[260px] rounded-xl border border-white/10 bg-slate-950/95 p-2 shadow-2xl backdrop-blur-md sm:w-80"
                        >
                          {BACKGROUND_GROUP_ORDER.map((group) => {
                            const options = backgroundOptionsByGroup[group.id];
                            if (!options || options.length === 0) {
                              return null;
                            }
                            return (
                              <div key={group.id} className="py-1">
                                <div className="px-2 pb-1 text-[10px] uppercase tracking-[0.24em] text-slate-500">
                                  {group.label}
                                </div>
                                <div className="space-y-1">
                                  {options.map((option) => {
                                    const isActive =
                                      replayBackgroundTheme === option.id;
                                    return (
                                      <button
                                        key={option.id}
                                        type="button"
                                        role="menuitemradio"
                                        aria-checked={isActive}
                                        onClick={() =>
                                          handleBackgroundSelect(option)
                                        }
                                        className={clsx(
                                          "flex w-full items-center gap-3 rounded-lg border px-2.5 py-2 text-left transition-colors focus:outline-none focus:ring-2 focus:ring-flow-accent/60 focus:ring-offset-2 focus:ring-offset-slate-950",
                                          isActive
                                            ? "border-flow-accent/80 bg-flow-accent/20 text-white shadow-[0_12px_35px_rgba(59,130,246,0.32)]"
                                            : "border-white/5 bg-slate-900/60 text-slate-300 hover:border-flow-accent/40 hover:text-white",
                                        )}
                                      >
                                        <span
                                          className="relative h-10 w-16 flex-shrink-0 overflow-hidden rounded-lg border border-white/10 shadow-inner"
                                          style={option.previewStyle}
                                        >
                                          {option.previewNode}
                                        </span>
                                        <span className="flex flex-1 flex-col text-xs text-slate-300">
                                          <span className="flex items-center justify-between gap-2 text-sm font-medium">
                                            <span>{option.label}</span>
                                            {isActive && (
                                              <Check
                                                size={14}
                                                className="text-flow-accent"
                                              />
                                            )}
                                          </span>
                                          <span className="text-[11px] text-slate-400">
                                            {option.subtitle}
                                          </span>
                                        </span>
                                      </button>
                                    );
                                  })}
                                </div>
                              </div>
                            );
                          })}
                        </div>
                      )}
                    </div>
                  </div>

                  <div className="border-t border-white/5 pt-3 space-y-4">
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <span className="text-[11px] uppercase tracking-[0.24em] text-slate-400">
                        Cursor
                      </span>
                      <span className="text-[11px] text-slate-500">
                        Style the virtual pointer overlay
                      </span>
                    </div>
                    <div className="flex flex-col gap-3 lg:flex-row lg:flex-wrap lg:items-start lg:justify-between">
                      <div
                        ref={cursorSelectorRef}
                        className="relative flex-1 min-w-[220px]"
                      >
                        <button
                          type="button"
                          className={clsx(
                            "flex w-full items-center justify-between rounded-xl border px-3 py-2 text-left text-sm transition-all focus:outline-none focus:ring-2 focus:ring-flow-accent/70 focus:ring-offset-2 focus:ring-offset-slate-900",
                            isCursorMenuOpen
                              ? "border-flow-accent/70 bg-slate-900/80 text-white"
                              : "border-white/10 bg-slate-900/60 text-slate-200 hover:border-flow-accent/40 hover:text-white",
                          )}
                          onClick={() => {
                            setIsCursorMenuOpen((open) => !open);
                            setIsBackgroundMenuOpen(false);
                            setIsCursorPositionMenuOpen(false);
                            setIsCursorClickAnimationMenuOpen(false);
                          }}
                          aria-haspopup="menu"
                          aria-expanded={isCursorMenuOpen}
                        >
                          <span className="flex items-center gap-3">
                            <span className="relative flex h-10 w-12 items-center justify-center overflow-hidden rounded-lg border border-white/10 bg-slate-900/60">
                              {selectedCursorOption.preview}
                            </span>
                            <span className="flex flex-col text-xs leading-tight text-slate-300">
                              <span className="text-sm font-medium text-white">
                                {selectedCursorOption.label}
                              </span>
                              <span className="text-[11px] text-slate-400">
                                {selectedCursorOption.subtitle}
                              </span>
                            </span>
                          </span>
                          <ChevronDown
                            size={14}
                            className={clsx(
                              "ml-3 flex-shrink-0 text-slate-400 transition-transform duration-150",
                              isCursorMenuOpen ? "rotate-180 text-white" : "",
                            )}
                          />
                        </button>
                        {isCursorMenuOpen && (
                          <div
                            role="menu"
                            className="absolute right-0 z-30 mt-2 w-full min-w-[240px] rounded-xl border border-white/10 bg-slate-950/95 p-2 shadow-[0_20px_50px_rgba(15,23,42,0.55)] backdrop-blur"
                          >
                            {CURSOR_GROUP_ORDER.map((group) => {
                              const options = cursorOptionsByGroup[group.id];
                              if (!options || options.length === 0) {
                                return null;
                              }
                              return (
                                <div key={group.id} className="mb-2 last:mb-0">
                                  <div className="px-2 pb-1 text-[10px] uppercase tracking-[0.24em] text-slate-500">
                                    {group.label}
                                  </div>
                                  <div className="space-y-1">
                                    {options.map((option) => {
                                      const isActive =
                                        replayCursorTheme === option.id;
                                      return (
                                        <button
                                          key={option.id}
                                          type="button"
                                          role="menuitemradio"
                                          aria-checked={isActive}
                                          onClick={() =>
                                            handleCursorThemeSelect(option.id)
                                          }
                                          className={clsx(
                                            "flex w-full items-center gap-3 rounded-lg border px-2.5 py-2 text-left transition-colors focus:outline-none focus:ring-2 focus:ring-flow-accent/60 focus:ring-offset-2 focus:ring-offset-slate-950",
                                            isActive
                                              ? "border-flow-accent/80 bg-flow-accent/20 text-white shadow-[0_12px_35px_rgba(59,130,246,0.32)]"
                                              : "border-white/5 bg-slate-900/60 text-slate-300 hover:border-flow-accent/40 hover:text-white",
                                          )}
                                        >
                                          <span className="flex h-9 w-9 items-center justify-center rounded-lg bg-slate-900/70">
                                            {option.preview}
                                          </span>
                                          <span className="flex flex-1 flex-col text-xs text-slate-300">
                                            <span className="flex items-center justify-between gap-2 text-sm font-medium">
                                              <span>{option.label}</span>
                                              {isActive && (
                                                <Check
                                                  size={14}
                                                  className="text-flow-accent"
                                                />
                                              )}
                                            </span>
                                            <span className="text-[11px] text-slate-400">
                                              {option.subtitle}
                                            </span>
                                          </span>
                                        </button>
                                      );
                                    })}
                                  </div>
                                </div>
                              );
                            })}
                          </div>
                        )}
                      </div>
                      <div
                        ref={cursorPositionSelectorRef}
                        className="relative flex flex-1 flex-col gap-2 lg:max-w-xs"
                      >
                        <span className="text-[10px] uppercase tracking-[0.24em] text-slate-500">
                          Initial placement
                        </span>
                        <button
                          type="button"
                          className={clsx(
                            "flex w-full items-center justify-between rounded-xl border px-3 py-2 text-left text-sm transition-all focus:outline-none focus:ring-2 focus:ring-flow-accent/70 focus:ring-offset-2 focus:ring-offset-slate-900",
                            isCursorPositionMenuOpen
                              ? "border-flow-accent/70 bg-slate-900/80 text-white"
                              : "border-white/10 bg-slate-900/60 text-slate-200 hover:border-flow-accent/40 hover:text-white",
                          )}
                          onClick={() => {
                            setIsCursorPositionMenuOpen((open) => !open);
                            setIsBackgroundMenuOpen(false);
                            setIsCursorMenuOpen(false);
                            setIsCursorClickAnimationMenuOpen(false);
                          }}
                          aria-haspopup="menu"
                          aria-expanded={isCursorPositionMenuOpen}
                        >
                          <span className="flex flex-col text-xs leading-tight text-slate-300">
                            <span className="text-sm font-medium text-white">
                              {selectedCursorPositionOption.label}
                            </span>
                            <span className="text-[11px] text-slate-400">
                              {selectedCursorPositionOption.subtitle}
                            </span>
                          </span>
                          <ChevronDown
                            size={14}
                            className={clsx(
                              "ml-3 flex-shrink-0 text-slate-400 transition-transform duration-150",
                              isCursorPositionMenuOpen
                                ? "rotate-180 text-white"
                                : "",
                            )}
                          />
                        </button>
                        {isCursorPositionMenuOpen && (
                          <div
                            role="menu"
                            className="absolute right-0 z-30 mt-2 w-full rounded-xl border border-white/10 bg-slate-950/95 p-2 shadow-[0_18px_45px_rgba(15,23,42,0.55)] backdrop-blur"
                          >
                            <div className="space-y-1">
                              {REPLAY_CURSOR_POSITIONS.map((option) => {
                                const isActive =
                                  replayCursorInitialPosition === option.id;
                                return (
                                  <button
                                    key={option.id}
                                    type="button"
                                    role="menuitemradio"
                                    aria-checked={isActive}
                                    onClick={() =>
                                      handleCursorPositionSelect(option.id)
                                    }
                                    className={clsx(
                                      "w-full rounded-lg border px-2.5 py-2 text-left text-xs transition-colors focus:outline-none focus:ring-2 focus:ring-flow-accent/60 focus:ring-offset-2 focus:ring-offset-slate-950",
                                      isActive
                                        ? "border-flow-accent/80 bg-flow-accent/20 text-white shadow-[0_10px_28px_rgba(59,130,246,0.3)]"
                                        : "border-white/5 bg-slate-900/60 text-slate-300 hover:border-flow-accent/40 hover:text-white",
                                    )}
                                  >
                                    <span className="block text-[11px] uppercase tracking-[0.22em] text-slate-400">
                                      {option.label}
                                    </span>
                                    <span className="text-[11px] text-slate-400/90">
                                      {option.subtitle}
                                    </span>
                                  </button>
                                );
                              })}
                            </div>
                          </div>
                        )}
                      </div>
                      <div
                        ref={cursorClickAnimationSelectorRef}
                        className="relative flex flex-1 flex-col gap-2 lg:max-w-xs"
                      >
                        <span className="text-[10px] uppercase tracking-[0.24em] text-slate-500">
                          Click animation
                        </span>
                        <button
                          type="button"
                          className={clsx(
                            "flex w-full items-center justify-between rounded-xl border px-3 py-2 text-left text-sm transition-all focus:outline-none focus:ring-2 focus:ring-flow-accent/70 focus:ring-offset-2 focus:ring-offset-slate-900",
                            isCursorClickAnimationMenuOpen
                              ? "border-flow-accent/70 bg-slate-900/80 text-white"
                              : "border-white/10 bg-slate-900/60 text-slate-200 hover:border-flow-accent/40 hover:text-white",
                          )}
                          onClick={() => {
                            setIsCursorClickAnimationMenuOpen((open) => !open);
                            setIsCursorMenuOpen(false);
                            setIsCursorPositionMenuOpen(false);
                            setIsBackgroundMenuOpen(false);
                          }}
                          aria-haspopup="menu"
                          aria-expanded={isCursorClickAnimationMenuOpen}
                        >
                          <span className="flex items-center gap-3">
                            <span className="relative flex h-10 w-12 items-center justify-center overflow-hidden rounded-lg border border-white/10 bg-slate-900/60">
                              {selectedCursorClickAnimationOption.preview}
                            </span>
                            <span className="flex flex-col text-xs leading-tight text-slate-300">
                              <span className="text-sm font-medium text-white">
                                {selectedCursorClickAnimationOption.label}
                              </span>
                              <span className="text-[11px] text-slate-400">
                                {selectedCursorClickAnimationOption.subtitle}
                              </span>
                            </span>
                          </span>
                          <ChevronDown
                            size={14}
                            className={clsx(
                              "ml-3 flex-shrink-0 text-slate-400 transition-transform duration-150",
                              isCursorClickAnimationMenuOpen
                                ? "rotate-180 text-white"
                                : "",
                            )}
                          />
                        </button>
                        {isCursorClickAnimationMenuOpen && (
                          <div
                            role="menu"
                            className="absolute right-0 z-30 mt-2 w-full min-w-[220px] rounded-xl border border-white/10 bg-slate-950/95 p-2 shadow-[0_18px_45px_rgba(15,23,42,0.55)] backdrop-blur"
                          >
                            <div className="space-y-1">
                              {REPLAY_CURSOR_CLICK_ANIMATION_OPTIONS.map(
                                (option) => {
                                  const isActive =
                                    replayCursorClickAnimation === option.id;
                                  return (
                                    <button
                                      key={option.id}
                                      type="button"
                                      role="menuitemradio"
                                      aria-checked={isActive}
                                      onClick={() =>
                                        handleCursorClickAnimationSelect(
                                          option.id,
                                        )
                                      }
                                      className={clsx(
                                        "flex w-full items-center gap-3 rounded-lg border px-2.5 py-2 text-left text-xs transition-colors focus:outline-none focus:ring-2 focus:ring-flow-accent/60 focus:ring-offset-2 focus:ring-offset-slate-950",
                                        isActive
                                          ? "border-flow-accent/80 bg-flow-accent/20 text-white shadow-[0_12px_32px_rgba(59,130,246,0.28)]"
                                          : "border-white/5 bg-slate-900/60 text-slate-300 hover:border-flow-accent/40 hover:text-white",
                                      )}
                                    >
                                      <span className="flex h-9 w-9 items-center justify-center rounded-lg bg-slate-900/70">
                                        {option.preview}
                                      </span>
                                      <span className="flex flex-1 flex-col text-xs text-slate-300">
                                        <span className="flex items-center justify-between gap-2 text-sm font-medium">
                                          <span>{option.label}</span>
                                          {isActive && (
                                            <Check
                                              size={14}
                                              className="text-flow-accent"
                                            />
                                          )}
                                        </span>
                                        <span className="text-[11px] text-slate-400">
                                          {option.subtitle}
                                        </span>
                                      </span>
                                    </button>
                                  );
                                },
                              )}
                            </div>
                          </div>
                        )}
                      </div>
                    </div>
                    <div className="flex flex-col gap-2">
                      <span className="text-[10px] uppercase tracking-[0.24em] text-slate-500">
                        Cursor size
                      </span>
                      <div className="flex items-center gap-3">
                        <input
                          type="range"
                          min={CURSOR_SCALE_MIN}
                          max={CURSOR_SCALE_MAX}
                          step={0.05}
                          value={replayCursorScale}
                          onChange={(event) =>
                            handleCursorScaleChange(
                              Number.parseFloat(event.target.value),
                            )
                          }
                          className="flex-1 accent-flow-accent"
                        />
                        <span className="w-12 text-right text-xs text-slate-300">
                          {Math.round(replayCursorScale * 100)}%
                        </span>
                      </div>
                      <span className="text-[11px] text-slate-500">
                        Fine-tune pointer proportions for the recorded viewport.
                      </span>
                    </div>
                  </div>
                </div>
              )}
            </div>
            <div className="relative w-full overflow-hidden rounded-2xl border border-slate-800 bg-slate-900/60 shadow-[0_25px_70px_rgba(15,23,42,0.45)]">
              <iframe
                key={execution.id}
                ref={(node) => {
                  composerRef.current = node;
                  composerWindowRef.current = node?.contentWindow ?? null;
                }}
                src={composerUrl}
                title="Replay Composer"
                className="w-full border-0"
                style={{
                  aspectRatio: `${selectedDimensions.width} / ${selectedDimensions.height}`,
                  minHeight: "360px",
                }}
                allow="clipboard-read; clipboard-write"
              />
              {(isMovieSpecLoading || !isComposerReady) && !movieSpecError && (
                <div className="pointer-events-none absolute inset-0 flex items-center justify-center bg-slate-950/55">
                  <span className="text-xs uppercase tracking-[0.3em] text-slate-400">
                    {isMovieSpecLoading
                      ? "Loading replay spec…"
                      : "Initialising player…"}
                  </span>
                </div>
              )}
              {movieSpecError && (
                <div className="absolute inset-0 flex items-center justify-center bg-slate-950/65 p-6 text-center">
                  <div className="rounded-xl border border-slate-800 bg-slate-900/70 px-6 py-4 text-sm text-slate-200 shadow-[0_15px_45px_rgba(15,23,42,0.5)]">
                    <div className="mb-1 font-semibold text-slate-100">
                      Failed to load replay spec
                    </div>
                    <div className="text-xs text-slate-300/80">
                      {movieSpecError}
                    </div>
                    {(previewMetrics.capturedFrames > 0 ||
                      previewMetrics.totalDurationMs > 0) && (
                      <div className="mt-3 text-[11px] text-slate-400">
                        {previewMetrics.capturedFrames > 0 && (
                          <span>
                            {formatCapturedLabel(
                              previewMetrics.capturedFrames,
                              "frame",
                            )}
                          </span>
                        )}
                        {previewMetrics.capturedFrames > 0 &&
                          previewMetrics.totalDurationMs > 0 && (
                            <span> • </span>
                          )}
                        {previewMetrics.totalDurationMs > 0 && (
                          <span>
                            {formatSeconds(
                              previewMetrics.totalDurationMs / 1000,
                            )}{" "}
                            recorded
                          </span>
                        )}
                      </div>
                    )}
                    {activeSpecId && (
                      <div className="mt-1 text-[10px] uppercase tracking-[0.24em] text-slate-500">
                        Spec {activeSpecId.slice(0, 8)}
                      </div>
                    )}
                  </div>
                </div>
              )}
            </div>
          </div>
        ) : activeTab === "screenshots" ? (
          screenshots.length === 0 ? (
            <div className="flex flex-1 items-center justify-center p-6 text-center">
              <div>
                <Image size={32} className="mx-auto mb-3 text-gray-600" />
                <div className="text-sm text-gray-400 mb-1">
                  No screenshots captured
                </div>
                {isFailed && (
                  <div className="text-xs text-gray-500">
                    Execution failed before screenshot steps could run
                  </div>
                )}
                {isCancelled && (
                  <div className="text-xs text-gray-500">
                    Execution was cancelled before screenshot steps could run
                  </div>
                )}
                {execution.status === "completed" && (
                  <div className="text-xs text-gray-500">
                    This workflow does not include screenshot steps
                  </div>
                )}
              </div>
            </div>
          ) : (
            <>
              <div className="flex-1 p-3 overflow-auto">
                <div className="space-y-4" data-testid={selectors.executions.viewer.screenshots}>
                  {screenshots.map((screenshot) => (
                    <div
                      key={screenshot.id}
                      ref={(node) => {
                        if (node) {
                          screenshotRefs.current[screenshot.id] = node;
                        } else {
                          delete screenshotRefs.current[screenshot.id];
                        }
                      }}
                      onClick={() => setSelectedScreenshot(screenshot)}
                      className={clsx(
                        "cursor-pointer overflow-hidden rounded-xl border transition-all duration-200",
                        selectedScreenshot?.id === screenshot.id
                          ? "border-flow-accent/80 shadow-[0_22px_50px_rgba(59,130,246,0.35)]"
                          : "border-gray-800 hover:border-flow-accent/50 hover:shadow-[0_15px_40px_rgba(59,130,246,0.2)]",
                      )}
                      data-testid={selectors.timeline.frame}
                    >
                      <div className="bg-slate-900/80 px-3 py-2 flex items-center justify-between text-xs text-slate-300">
                        <span className="truncate font-medium">
                          {screenshot.stepName}
                        </span>
                        <span className="text-slate-400">
                          {format(screenshot.timestamp, "HH:mm:ss.SSS")}
                        </span>
                      </div>
                      <img
                        src={screenshot.url}
                        alt={screenshot.stepName}
                        loading="lazy"
                        className="block w-full"
                        data-testid={selectors.executions.viewer.screenshot}
                      />
                    </div>
                  ))}
                </div>
              </div>

              <div className="border-t border-gray-800 p-2 overflow-x-auto">
                <div className="flex gap-2">
                  {screenshots.map((screenshot) => (
                    <div
                      key={screenshot.id}
                      onClick={() => setSelectedScreenshot(screenshot)}
                      className={clsx(
                        "flex-shrink-0 w-20 h-20 rounded-lg overflow-hidden cursor-pointer border-2 transition-all duration-150",
                        selectedScreenshot?.id === screenshot.id
                          ? "border-flow-accent shadow-[0_12px_30px_rgba(59,130,246,0.35)]"
                          : "border-gray-700 hover:border-flow-accent/60",
                      )}
                    >
                      <img
                        src={screenshot.url}
                        alt={screenshot.stepName}
                        loading="lazy"
                        className="w-full h-full object-cover"
                      />
                    </div>
                  ))}
                </div>
              </div>
            </>
          )
        ) : activeTab === "executions" ? (
          <div className="flex-1 overflow-hidden p-3">
            {execution.workflowId ? (
              <div className="h-full overflow-hidden rounded-xl border border-gray-800 bg-flow-node/40">
                <ExecutionHistory
                  workflowId={execution.workflowId}
                  onSelectExecution={handleExecutionSwitch}
                />
              </div>
            ) : (
              <div className="flex h-full items-center justify-center text-sm text-gray-400">
                Workflow identifier unavailable for this execution.
              </div>
            )}
          </div>
        ) : (
          <div
            className="flex-1 overflow-auto p-3"
            data-testid={selectors.executions.viewer.logs}
          >
            <div className="terminal-output">
              {execution.logs.map((log) => (
                <div
                  key={log.id}
                  className="flex gap-2 mb-1"
                  data-testid={selectors.executions.logEntry}
                >
                  <span className="text-xs text-gray-600">
                    {format(log.timestamp, "HH:mm:ss")}
                  </span>
                  <span className={`flex-1 text-xs ${getLogColor(log.level)}`}>
                    {log.message}
                  </span>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>

      <ResponsiveDialog
        isOpen={isExportDialogOpen}
        onDismiss={handleCloseExportDialog}
        ariaLabelledBy={exportDialogTitleId}
        size="wide"
        overlayClassName="z-50"
        className="bg-flow-node border border-gray-800 shadow-2xl max-h-[90vh] flex flex-col"
      >
        <div className="flex items-center justify-between border-b border-gray-800 px-6 py-4">
          <div className="flex items-center gap-3">
            <span className="flex h-10 w-10 items-center justify-center rounded-lg bg-flow-accent/15 text-flow-accent">
              <Download size={18} />
            </span>
            <div>
              <h2
                id={exportDialogTitleId}
                className="text-lg font-semibold text-white"
              >
                Export replay
              </h2>
              <p
                id={exportDialogDescriptionId}
                className="text-sm text-gray-400"
              >
                Choose format, naming, and destination for execution #
                {execution.id.slice(0, 8)}.
              </p>
            </div>
          </div>
          <button
            type="button"
            onClick={handleCloseExportDialog}
            className="rounded-full p-2 text-gray-400 transition hover:bg-white/5 hover:text-white"
            aria-label="Close export dialog"
          >
            <X size={16} />
          </button>
        </div>

        <div
          className="flex-1 overflow-y-auto px-6 py-5 space-y-6"
          aria-describedby={exportDialogDescriptionId}
        >
          <section className="space-y-3">
            <div className="flex items-center justify-between">
              <div>
                <h3 className="text-sm font-semibold text-white">Format</h3>
                <p className="text-xs text-gray-400">
                  Pick the export target that fits your workflow.
                </p>
              </div>
              <span className="text-[11px] uppercase tracking-[0.2em] text-gray-500">
                {isBinaryExport ? "Direct download" : "Data bundle"}
              </span>
            </div>
            <div className="grid gap-3 md:grid-cols-3">
              {EXPORT_FORMAT_OPTIONS.map((option) => {
                const isSelected = option.id === exportFormat;
                const Icon = option.icon;
                return (
                  <button
                    key={option.id}
                    type="button"
                    onClick={() => setExportFormat(option.id)}
                    className={clsx(
                      "flex flex-col items-start gap-3 rounded-xl border px-4 py-3 text-left transition-all focus:outline-none focus:ring-2 focus:ring-flow-accent/60 focus:ring-offset-2 focus:ring-offset-slate-900",
                      isSelected
                        ? "border-flow-accent/70 bg-flow-accent/20 text-white shadow-[0_20px_45px_rgba(59,130,246,0.25)]"
                        : "border-white/10 bg-slate-900/60 text-slate-300 hover:border-flow-accent/40 hover:text-white",
                    )}
                  >
                    <span className="flex w-full items-center justify-between">
                      <span
                        className={clsx(
                          "flex h-10 w-10 items-center justify-center rounded-lg",
                          isSelected
                            ? "bg-flow-accent/80 text-white"
                            : "bg-slate-900/70 text-flow-accent",
                        )}
                      >
                        <Icon size={18} />
                      </span>
                      {option.badge && (
                        <span
                          className={clsx(
                            "rounded-full px-2 py-0.5 text-[10px] font-semibold uppercase tracking-[0.2em]",
                            isSelected
                              ? "bg-white/20 text-white"
                              : "bg-slate-800 text-slate-300",
                          )}
                        >
                          {option.badge}
                        </span>
                      )}
                    </span>
                    <div>
                      <div className="text-sm font-semibold">
                        {option.label}
                      </div>
                      <div className="text-xs text-slate-400">
                        {option.description}
                      </div>
                    </div>
                  </button>
                );
              })}
            </div>
          </section>

          <section className="space-y-3">
            <div className="flex items-center justify-between">
              <h3 className="text-sm font-semibold text-white">
                First frame preview
              </h3>
              {firstFrameLabel && (
                <span className="text-xs text-slate-400">
                  {firstFrameLabel}
                </span>
              )}
            </div>
            <div className="overflow-hidden rounded-xl border border-white/10 bg-slate-900/60">
              {preparedMovieSpec ? (
                <div
                  className="relative w-full"
                  style={{
                    aspectRatio: `${selectedDimensions.width} / ${selectedDimensions.height}`,
                  }}
                >
                  <iframe
                    key={`${execution.id}-preview`}
                    ref={(node) => {
                      previewComposerRef.current = node;
                      const nextWindow = node?.contentWindow ?? null;
                      if (composerPreviewWindowRef.current !== nextWindow) {
                        composerPreviewWindowRef.current = nextWindow;
                        if (!nextWindow) {
                          setIsPreviewComposerReady(false);
                          setPreviewComposerError(null);
                          composerPreviewOriginRef.current = null;
                        }
                      }
                    }}
                    src={composerPreviewUrl}
                    title="Replay preview"
                    className="absolute inset-0 h-full w-full border-0"
                    allow="clipboard-read; clipboard-write"
                  />
                  {!isPreviewComposerReady && firstFramePreviewUrl && (
                    <img
                      src={firstFramePreviewUrl}
                      alt="First frame snapshot"
                      className="absolute inset-0 h-full w-full object-cover"
                    />
                  )}
                  {!isPreviewComposerReady && (
                    <div className="absolute inset-0 flex items-center justify-center bg-slate-950/60 text-xs uppercase tracking-[0.3em] text-slate-300">
                      {previewComposerError ?? "Loading preview…"}
                    </div>
                  )}
                </div>
              ) : firstFramePreviewUrl ? (
                <img
                  src={firstFramePreviewUrl}
                  alt="First frame preview"
                  className="block w-full object-cover"
                  style={{
                    aspectRatio: `${selectedDimensions.width} / ${selectedDimensions.height}`,
                  }}
                />
              ) : (
                <div className="flex h-40 items-center justify-center text-sm text-slate-400">
                  Preview unavailable
                </div>
              )}
              <div className="flex items-center justify-between border-t border-white/5 px-3 py-2 text-[11px] uppercase tracking-[0.2em] text-slate-500">
                <span>
                  {selectedDimensions.width}×{selectedDimensions.height} px
                </span>
                <span>Canvas</span>
              </div>
            </div>
          </section>

          <section className="space-y-3">
            <div className="flex items-center justify-between">
              <h3 className="text-sm font-semibold text-white">Dimensions</h3>
              <span className="text-xs text-slate-400">
                {selectedDimensions.width}×{selectedDimensions.height} px
              </span>
            </div>
            <div className="grid gap-3 md:grid-cols-2">
              {dimensionPresetOptions.map((option) => {
                const isSelected = dimensionPreset === option.id;
                return (
                  <button
                    key={option.id}
                    type="button"
                    onClick={() => setDimensionPreset(option.id)}
                    className={clsx(
                      "flex flex-col items-start gap-1.5 rounded-xl border px-4 py-3 text-left transition-all focus:outline-none focus:ring-2 focus:ring-flow-accent/60 focus:ring-offset-2 focus:ring-offset-slate-900",
                      isSelected
                        ? "border-flow-accent/70 bg-flow-accent/20 text-white shadow-[0_18px_40px_rgba(59,130,246,0.25)]"
                        : "border-white/10 bg-slate-900/60 text-slate-300 hover:border-flow-accent/40 hover:text-white",
                    )}
                  >
                    <span className="text-sm font-semibold">
                      {option.label}
                    </span>
                    <span className="text-xs text-slate-400">
                      {option.id === "custom"
                        ? "Define width & height"
                        : `${option.width}×${option.height} px`}
                    </span>
                    {option.description && (
                      <span className="text-[11px] text-slate-500">
                        {option.description}
                      </span>
                    )}
                  </button>
                );
              })}
            </div>
            {dimensionPreset === "custom" && (
              <div className="grid gap-3 sm:grid-cols-2">
                <label className="flex flex-col gap-1">
                  <span className="text-xs text-slate-400">Width (px)</span>
                  <input
                    type="number"
                    min={320}
                    step={10}
                    value={customWidthInput}
                    onChange={(event) => {
                      setDimensionPreset("custom");
                      setCustomWidthInput(
                        event.target.value.replace(/[^0-9]/g, ""),
                      );
                    }}
                    className="rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-sm text-white placeholder:text-slate-500 focus:border-flow-accent focus:outline-none focus:ring-2 focus:ring-flow-accent/40"
                    placeholder={String(DEFAULT_EXPORT_WIDTH)}
                  />
                </label>
                <label className="flex flex-col gap-1">
                  <span className="text-xs text-slate-400">Height (px)</span>
                  <input
                    type="number"
                    min={320}
                    step={10}
                    value={customHeightInput}
                    onChange={(event) => {
                      setDimensionPreset("custom");
                      setCustomHeightInput(
                        event.target.value.replace(/[^0-9]/g, ""),
                      );
                    }}
                    className="rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-sm text-white placeholder:text-slate-500 focus:border-flow-accent focus:outline-none focus:ring-2 focus:ring-flow-accent/40"
                    placeholder={String(DEFAULT_EXPORT_HEIGHT)}
                  />
                </label>
              </div>
            )}
          </section>

          <section className="space-y-3">
            <div className="flex items-center justify-between">
              <h3 className="text-sm font-semibold text-white">File name</h3>
              <span className="text-xs text-gray-500">
                Extension follows format
              </span>
            </div>
            <div className="flex items-center gap-3">
              <input
                type="text"
                value={exportFileStem}
                onChange={(event) => setExportFileStem(event.target.value)}
                className="flex-1 rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-sm text-white placeholder:text-gray-500 focus:border-flow-accent focus:outline-none focus:ring-2 focus:ring-flow-accent/40"
                placeholder={defaultExportFileStem}
              />
              <span className="rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-xs text-gray-300">
                .{EXPORT_EXTENSIONS[exportFormat]}
              </span>
            </div>
            <p className="text-xs text-gray-500">
              Final file name:{" "}
              <code className="text-gray-300">{finalFileName}</code>
            </p>
          </section>

          {exportFormat === "json" ? (
            <section className="space-y-3">
              <div className="flex items-center justify-between">
                <h3 className="text-sm font-semibold text-white">
                  Destination
                </h3>
                {!supportsFileSystemAccess && (
                  <span className="text-xs text-amber-400">
                    Browser will download to defaults
                  </span>
                )}
              </div>
              <label
                className={clsx(
                  "flex items-center gap-3 rounded-lg border px-4 py-3 text-sm transition",
                  useNativeFilePicker && supportsFileSystemAccess
                    ? "border-flow-accent/60 bg-flow-accent/10 text-white"
                    : "border-white/10 bg-slate-900/60 text-slate-300",
                  !supportsFileSystemAccess && "opacity-60",
                )}
              >
                <input
                  type="checkbox"
                  className="h-4 w-4 rounded border-white/20 bg-slate-900 text-flow-accent focus:ring-flow-accent"
                  checked={useNativeFilePicker && supportsFileSystemAccess}
                  onChange={(event) =>
                    setUseNativeFilePicker(event.target.checked)
                  }
                  disabled={!supportsFileSystemAccess}
                />
                <div className="flex flex-col">
                  <span className="font-medium">
                    {supportsFileSystemAccess
                      ? "Choose save location on export"
                      : "Use browser default download folder"}
                  </span>
                  <span className="text-xs text-slate-400">
                    {supportsFileSystemAccess
                      ? "Open your OS save dialog to pick the destination each time."
                      : "Your browser will download using its configured destination."}
                  </span>
                </div>
              </label>
            </section>
          ) : (
            <section className="space-y-3">
              <div className="flex items-center justify-between">
                <h3 className="text-sm font-semibold text-white">Download</h3>
                <span className="text-xs text-slate-400">Server-rendered</span>
              </div>
              <div className="space-y-2 rounded-lg border border-white/10 bg-slate-900/60 p-3 text-xs text-slate-300">
                <p>
                  Your replay renders on the API using the same metadata that
                  powers the in-app player.
                </p>
                <p>
                  We’ll trigger a download automatically once rendering
                  finishes.
                </p>
              </div>
            </section>
          )}

          <section className="space-y-3">
            <div className="flex items-center justify-between">
              <h3 className="text-sm font-semibold text-white">
                Replay summary
              </h3>
              <span className="text-xs text-gray-500">{execution.status}</span>
            </div>
            <div className="grid gap-3 md:grid-cols-3">
              <div className="rounded-lg border border-white/10 bg-slate-900/60 px-3 py-2">
                <div className="text-xs text-gray-400">Frames</div>
                <div className="text-lg font-semibold text-white">
                  {estimatedFrameCount}
                </div>
              </div>
              <div className="rounded-lg border border-white/10 bg-slate-900/60 px-3 py-2">
                <div className="text-xs text-gray-400">Duration</div>
                <div className="text-lg font-semibold text-white">
                  {estimatedDurationSeconds != null
                    ? formatSeconds(estimatedDurationSeconds)
                    : "—"}
                </div>
              </div>
              <div className="rounded-lg border border-white/10 bg-slate-900/60 px-3 py-2">
                <div className="text-xs text-gray-400">Status</div>
                <div className="text-sm text-slate-300">
                  {exportStatusMessage}
                </div>
                {activeSpecId && (
                  <div className="mt-1 text-[10px] uppercase tracking-[0.22em] text-slate-500">
                    Spec {activeSpecId.slice(0, 8)}
                  </div>
                )}
                {previewMetrics.assetCount > 0 && (
                  <div className="mt-1 text-[11px] text-slate-500">
                    {formatCapturedLabel(previewMetrics.assetCount, "asset")}
                  </div>
                )}
              </div>
            </div>
          </section>
        </div>

        <div className="flex items-center justify-between border-t border-gray-800 bg-flow-bg/60 px-6 py-4">
          <div className="text-xs text-gray-400">
            {exportFormat === "json"
              ? "Exports the replay package with chosen theming so tooling can recreate animations."
              : "Rendering runs on the API and downloads the finished media when ready."}
          </div>
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={handleCloseExportDialog}
              className="rounded-lg border border-white/10 px-4 py-2 text-sm text-gray-300 transition hover:border-white/20 hover:text-white"
              disabled={isExporting}
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={handleConfirmExport}
              className={clsx(
                "flex items-center gap-2 rounded-lg bg-flow-accent px-4 py-2 text-sm font-semibold text-white transition disabled:cursor-not-allowed disabled:opacity-60",
                exportFormat !== "json" &&
                  "bg-gradient-to-r from-flow-accent to-sky-500",
              )}
              disabled={
                exportFormat === "json"
                  ? isExporting || isExportPreviewLoading
                  : isExporting || replayFrames.length === 0
              }
              data-testid={
                isExporting
                  ? selectors.executions.export.inProgress
                  : selectors.executions.actions.exportConfirmButton
              }
            >
              {exportFormat === "json" ? (
                isExporting || isExportPreviewLoading ? (
                  <>
                    <Loader size={16} className="animate-spin" />
                    {isExportPreviewLoading ? "Preparing…" : "Exporting…"}
                  </>
                ) : (
                  <>
                    <FolderOutput size={16} />
                    Export replay
                  </>
                )
              ) : isExporting ? (
                <>
                  <Loader size={16} className="animate-spin" />
                  Rendering…
                </>
              ) : (
                <>
                  <Download size={16} />
                  Export replay
                </>
              )}
            </button>
          </div>
        </div>
      </ResponsiveDialog>

      {/* Export Success Panel */}
      {showExportSuccess && lastCreatedExport && (
        <ExportSuccessPanel
          export_={lastCreatedExport}
          onClose={() => {
            setShowExportSuccess(false);
            setLastCreatedExport(null);
          }}
          onViewInLibrary={() => {
            setShowExportSuccess(false);
            setLastCreatedExport(null);
            // Navigate to exports tab - this will be handled by parent component
            window.dispatchEvent(new CustomEvent('navigate-to-exports'));
          }}
          onViewExecution={() => {
            setShowExportSuccess(false);
            setLastCreatedExport(null);
          }}
        />
      )}
    </div>
  );
}

interface EmptyExecutionViewerProps {
  workflowId: string;
  onClose?: () => void;
  showExecutionSwitcher?: boolean;
}

interface ExecutionViewerProps {
  workflowId: string;
  execution: Execution | null;
  onClose?: () => void;
  showExecutionSwitcher?: boolean;
}

const useCurrentWorkflowName = () =>
  useWorkflowStore((state) => state.currentWorkflow?.name ?? "Workflow");

const useWorkflowSave = () => useWorkflowStore((state) => state.saveWorkflow);

function EmptyExecutionViewer({
  workflowId,
  onClose,
  showExecutionSwitcher = false,
}: EmptyExecutionViewerProps) {
  const [activeTab, setActiveTab] = useState<ViewerTab>("replay");
  const [isStarting, setIsStarting] = useState(false);
  const startExecution = useExecutionStore((state) => state.startExecution);
  const loadExecution = useExecutionStore((state) => state.loadExecution);
  const workflowName = useCurrentWorkflowName();
  const saveWorkflow = useWorkflowSave();

  const handleStart = useCallback(async () => {
    if (!workflowId) {
      toast.error("Unable to determine workflow to execute");
      return;
    }

    setIsStarting(true);
    try {
      await startExecution(workflowId, () =>
        saveWorkflow({
          source: "execution-run",
          changeDescription: "Autosave before execution",
        }),
      );
      toast.success("Workflow execution started");
    } catch (error) {
      const message =
        error instanceof Error ? error.message : "Failed to start workflow";
      toast.error(message);
    } finally {
      setIsStarting(false);
    }
  }, [startExecution, workflowId, saveWorkflow]);

  const handleHistorySelect = useCallback(
    async (candidate: { id: string }) => {
      if (!candidate?.id) {
        return;
      }
      try {
        await loadExecution(candidate.id);
        setActiveTab("replay");
      } catch (error) {
        const message =
          error instanceof Error ? error.message : "Failed to load execution";
        toast.error(message);
      }
    },
    [loadExecution],
  );

  const renderStartButton = (options?: { variant?: "compact" | "large" }) => {
    const variant = options?.variant ?? "compact";
    const baseClasses =
      "inline-flex items-center gap-2 rounded-md font-medium transition focus:outline-none focus:ring-2 focus:ring-flow-accent/70 focus:ring-offset-2 focus:ring-offset-gray-900 disabled:cursor-not-allowed disabled:opacity-60";
    const palette =
      variant === "large"
        ? "bg-flow-accent px-5 py-2.5 text-base text-white shadow-lg shadow-flow-accent/30 hover:bg-flow-accent/90"
        : "bg-flow-accent px-3 py-1.5 text-sm text-white shadow-md shadow-flow-accent/30 hover:bg-flow-accent/85";

    return (
      <button
        type="button"
        onClick={handleStart}
        disabled={isStarting}
        className={`${baseClasses} ${palette}`}
      >
        {isStarting ? (
          <Loader size={16} className="animate-spin" />
        ) : (
          <PlayCircle size={16} />
        )}
        <span>{isStarting ? "Starting…" : "Start workflow"}</span>
      </button>
    );
  };

  const renderEmptyTab = (title: string, description: string) => (
    <div className="flex flex-1 flex-col items-center justify-center gap-6 px-6 text-center">
      <div className="max-w-md space-y-2">
        <h3 className="text-lg font-semibold text-white">{title}</h3>
        <p className="text-sm text-slate-400">{description}</p>
      </div>
      {renderStartButton({ variant: "large" })}
    </div>
  );

  return (
    <div data-testid={selectors.executions.viewer.root} className="h-full flex flex-col bg-flow-node min-h-0">
      <div className="flex items-center justify-between p-3 border-b border-gray-800">
        <div className="flex items-center gap-3">
          <PlayCircle size={20} className="text-flow-accent" />
          <div>
            <div className="text-sm font-medium text-white">{workflowName}</div>
            <div className="text-xs text-gray-500">
              Execution viewer ready — no runs started yet
            </div>
          </div>
        </div>

        <div className="flex items-center gap-2">
          {renderStartButton()}
          {onClose && (
            <button
              className="toolbar-button p-1.5 ml-2 border-l border-gray-700 pl-3"
              title="Close"
              onClick={onClose}
            >
              <X size={14} />
            </button>
          )}
        </div>
      </div>

      <div className="h-2 bg-flow-bg">
        <div
          className="h-full bg-flow-accent transition-all duration-300"
          style={{ width: "0%" }}
        />
      </div>

      <div className="flex border-b border-gray-800">
        <button
          data-testid={selectors.executions.tabs.replay}
          className={`flex-1 px-3 py-2 text-sm font-medium transition-colors flex items-center justify-center gap-2 ${
            activeTab === "replay"
              ? "bg-flow-bg text-white border-b-2 border-flow-accent"
              : "text-gray-400 hover:text-white"
          }`}
          onClick={() => setActiveTab("replay")}
        >
          <PlayCircle size={14} />
          Replay (0)
        </button>
        <button
          data-testid={selectors.executions.tabs.screenshots}
          className={`flex-1 px-3 py-2 text-sm font-medium transition-colors flex items-center justify-center gap-2 ${
            activeTab === "screenshots"
              ? "bg-flow-bg text-white border-b-2 border-flow-accent"
              : "text-gray-400 hover:text-white"
          }`}
          onClick={() => setActiveTab("screenshots")}
        >
          <Image size={14} />
          Screenshots (0)
        </button>
        <button
          data-testid={selectors.executions.tabs.logs}
          className={`flex-1 px-3 py-2 text-sm font-medium transition-colors flex items-center justify-center gap-2 ${
            activeTab === "logs"
              ? "bg-flow-bg text-white border-b-2 border-flow-accent"
              : "text-gray-400 hover:text-white"
          }`}
          onClick={() => setActiveTab("logs")}
        >
          <Terminal size={14} />
          Logs (0)
        </button>
        {showExecutionSwitcher && (
          <button
            data-testid={selectors.executions.tabs.executions}
            className={`flex-1 px-3 py-2 text-sm font-medium transition-colors flex items-center justify-center gap-2 ${
              activeTab === "executions"
                ? "bg-flow-bg text-white border-b-2 border-flow-accent"
                : "text-gray-400 hover:text-white"
            }`}
            onClick={() => setActiveTab("executions")}
          >
            <ListTree size={14} />
            Executions
          </button>
        )}
      </div>

      <div className="flex-1 overflow-hidden flex flex-col min-h-0">
        {activeTab === "replay" ? (
          renderEmptyTab(
            "No execution in progress",
            "Start the workflow to capture a replay of each automation step.",
          )
        ) : activeTab === "screenshots" ? (
          renderEmptyTab(
            "Screenshots will appear here",
            "As the workflow runs, screenshots from each step are collected for review.",
          )
        ) : activeTab === "logs" ? (
          renderEmptyTab(
            "Execution logs are empty",
            "Run the workflow to stream live log output, retries, and console messages.",
          )
        ) : showExecutionSwitcher && workflowId ? (
          <div className="flex-1 overflow-auto p-4 space-y-4">
            <div className="flex items-center justify-between gap-3">
              <div>
                <h3 className="text-sm font-medium text-white">
                  Execution history
                </h3>
                <p className="text-xs text-slate-400">
                  Select a past execution to inspect its replay, screenshots,
                  and logs.
                </p>
              </div>
              {renderStartButton()}
            </div>
            <ExecutionHistory
              workflowId={workflowId}
              onSelectExecution={handleHistorySelect}
            />
          </div>
        ) : (
          renderEmptyTab(
            "Execution history unavailable",
            "Enable execution history to review previous workflow runs.",
          )
        )}
      </div>
    </div>
  );
}

function ExecutionViewer({
  workflowId,
  execution,
  onClose,
  showExecutionSwitcher,
}: ExecutionViewerProps) {
  if (!workflowId) {
    return null;
  }

  if (execution && execution.id) {
    return (
      <ActiveExecutionViewer
        execution={execution}
        onClose={onClose}
        showExecutionSwitcher={showExecutionSwitcher}
      />
    );
  }

  return (
    <EmptyExecutionViewer
      workflowId={workflowId}
      onClose={onClose}
      showExecutionSwitcher={showExecutionSwitcher}
    />
  );
}

export default ExecutionViewer;
