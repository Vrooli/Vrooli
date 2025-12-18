/**
 * Replay theme configuration options
 *
 * Contains all the preset options for customizing replay appearance including:
 * - Chrome themes (browser frame styles)
 * - Background themes (canvas backdrop)
 * - Cursor themes (pointer visualization)
 * - Click animations
 */
import type { CSSProperties, ReactNode } from "react";
import type {
  ReplayChromeTheme,
  ReplayBackgroundTheme,
  ReplayCursorTheme,
  ReplayCursorInitialPosition,
  ReplayCursorClickAnimation,
} from "./ReplayPlayer";

// Background images - using ES imports for reliable Vite asset handling
import geometricPrismUrl from "@/assets/replay-backgrounds/geometric-prism.jpg";
import geometricOrbitUrl from "@/assets/replay-backgrounds/geometric-orbit.jpg";
import geometricMosaicUrl from "@/assets/replay-backgrounds/geometric-mosaic.jpg";

// =============================================================================
// Chrome Theme Options
// =============================================================================

export const REPLAY_CHROME_OPTIONS: Array<{
  id: ReplayChromeTheme;
  label: string;
  subtitle: string;
}> = [
  { id: "aurora", label: "Aurora", subtitle: "macOS-inspired chrome" },
  { id: "chromium", label: "Chromium", subtitle: "Modern minimal controls" },
  { id: "midnight", label: "Midnight", subtitle: "Gradient showcase frame" },
  { id: "minimal", label: "Minimal", subtitle: "Hide browser chrome" },
];

// =============================================================================
// Background Theme Options
// =============================================================================

export type BackgroundKind = "abstract" | "solid" | "minimal" | "geometric";

export interface BackgroundOption {
  id: ReplayBackgroundTheme;
  label: string;
  subtitle: string;
  previewStyle: CSSProperties;
  previewNode?: ReactNode;
  kind: BackgroundKind;
}

export const REPLAY_BACKGROUND_OPTIONS: BackgroundOption[] = [
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

export const BACKGROUND_GROUP_ORDER: Array<{
  id: BackgroundKind;
  label: string;
}> = [
  { id: "abstract", label: "Abstract" },
  { id: "solid", label: "Solid" },
  { id: "minimal", label: "Minimal" },
  { id: "geometric", label: "Geometric" },
];

// =============================================================================
// Cursor Theme Options
// =============================================================================

export type CursorGroup = "hidden" | "halo" | "arrow" | "hand";

export interface CursorOption {
  id: ReplayCursorTheme;
  label: string;
  subtitle: string;
  group: CursorGroup;
  preview: ReactNode;
}

const ARROW_CURSOR_PATH =
  "M6 3L6 22L10.4 18.1L13.1 26.4L15.9 25.2L13.1 17.5L22 17.5L6 3Z";

const HAND_POINTER_PATHS = [
  "M22 14a8 8 0 0 1-8 8",
  "M18 11v-1a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v0",
  "M14 10V9a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v1",
  "M10 9.5V4a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v10",
  "M18 11a2 2 0 1 1 4 0v3a8 8 0 0 1-8 8h-2c-2.8 0-4.5-.86-5.99-2.34l-3.6-3.6a2 2 0 0 1 2.83-2.82L7 15",
];

export const REPLAY_CURSOR_OPTIONS: CursorOption[] = [
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

export const CURSOR_GROUP_ORDER: Array<{ id: CursorGroup; label: string }> = [
  { id: "hidden", label: "Hidden" },
  { id: "halo", label: "Halo Cursors" },
  { id: "arrow", label: "Arrowhead Cursors" },
  { id: "hand", label: "Pointing Hands" },
];

// =============================================================================
// Click Animation Options
// =============================================================================

export interface ClickAnimationOption {
  id: ReplayCursorClickAnimation;
  label: string;
  subtitle: string;
  preview: ReactNode;
}

export const REPLAY_CURSOR_CLICK_ANIMATION_OPTIONS: ClickAnimationOption[] = [
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

// =============================================================================
// Cursor Position Options
// =============================================================================

export interface CursorPositionOption {
  id: ReplayCursorInitialPosition;
  label: string;
  subtitle: string;
}

export const REPLAY_CURSOR_POSITIONS: CursorPositionOption[] = [
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

// =============================================================================
// Cursor Scale Constraints
// =============================================================================

export const CURSOR_SCALE_MIN = 0.6;
export const CURSOR_SCALE_MAX = 1.8;

// =============================================================================
// Type Guards
// =============================================================================

export const isReplayChromeTheme = (
  value: unknown,
): value is ReplayChromeTheme =>
  REPLAY_CHROME_OPTIONS.some((o) => o.id === value);

export const isReplayBackgroundTheme = (
  value: unknown,
): value is ReplayBackgroundTheme =>
  REPLAY_BACKGROUND_OPTIONS.some((o) => o.id === value);

export const isReplayCursorTheme = (
  value: unknown,
): value is ReplayCursorTheme =>
  REPLAY_CURSOR_OPTIONS.some((o) => o.id === value);

export const isReplayCursorInitialPosition = (
  value: unknown,
): value is ReplayCursorInitialPosition =>
  REPLAY_CURSOR_POSITIONS.some((o) => o.id === value);

export const isReplayCursorClickAnimation = (
  value: unknown,
): value is ReplayCursorClickAnimation =>
  REPLAY_CURSOR_CLICK_ANIMATION_OPTIONS.some((o) => o.id === value);
