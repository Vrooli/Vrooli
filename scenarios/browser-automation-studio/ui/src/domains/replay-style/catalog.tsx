import type { CSSProperties, ReactNode } from 'react';
import type {
  ReplayBackgroundTheme,
  ReplayChromeTheme,
  ReplayCursorTheme,
  ReplayCursorInitialPosition,
  ReplayCursorClickAnimation,
} from './model';
import { REPLAY_ARROW_CURSOR_PATH } from '@/domains/exports/replay/constants';

import geometricPrismUrl from '@/assets/replay-backgrounds/geometric-prism.jpg';
import geometricOrbitUrl from '@/assets/replay-backgrounds/geometric-orbit.jpg';
import geometricMosaicUrl from '@/assets/replay-backgrounds/geometric-mosaic.jpg';

export type BackgroundDecor = {
  containerClass: string;
  containerStyle?: CSSProperties;
  contentClass: string;
  baseLayer?: ReactNode;
  overlay?: ReactNode;
};

export type CursorDecor = {
  wrapperClass?: string;
  wrapperStyle?: CSSProperties;
  renderBase: ReactNode | null;
  trailColor: string;
  trailWidth: number;
  offset?: { x: number; y: number };
  transformOrigin?: string;
};

export type ChromeDecor = {
  frameClass: string;
  contentClass?: string;
  header: ReactNode | null;
};

export interface ChromeThemeOption {
  id: ReplayChromeTheme;
  label: string;
  subtitle: string;
}

export type BackgroundKind = 'abstract' | 'solid' | 'minimal' | 'geometric';

export interface BackgroundOption {
  id: ReplayBackgroundTheme;
  label: string;
  subtitle: string;
  previewStyle: CSSProperties;
  previewNode?: ReactNode;
  kind: BackgroundKind;
}

export type CursorGroup = 'hidden' | 'halo' | 'arrow' | 'hand';

export interface CursorOption {
  id: ReplayCursorTheme;
  label: string;
  subtitle: string;
  group: CursorGroup;
  preview: ReactNode;
}

export interface ClickAnimationOption {
  id: ReplayCursorClickAnimation;
  label: string;
  subtitle: string;
  preview: ReactNode;
}

export interface CursorPositionOption {
  id: ReplayCursorInitialPosition;
  label: string;
  subtitle: string;
}

export const REPLAY_CHROME_OPTIONS: ChromeThemeOption[] = [
  { id: 'aurora', label: 'Aurora', subtitle: 'macOS-inspired chrome' },
  { id: 'chromium', label: 'Chromium', subtitle: 'Modern minimal controls' },
  { id: 'midnight', label: 'Midnight', subtitle: 'Gradient showcase frame' },
  { id: 'minimal', label: 'Minimal', subtitle: 'Hide browser chrome' },
];

export const REPLAY_BACKGROUND_OPTIONS: BackgroundOption[] = [
  {
    id: 'aurora',
    label: 'Aurora Glow',
    subtitle: 'Iridescent gradient wash',
    previewStyle: {
      backgroundImage:
        'linear-gradient(135deg, rgba(56,189,248,0.7), rgba(129,140,248,0.7))',
    },
    kind: 'abstract',
  },
  {
    id: 'sunset',
    label: 'Sunset Bloom',
    subtitle: 'Fuchsia → amber ambience',
    previewStyle: {
      backgroundImage:
        'linear-gradient(135deg, rgba(244,114,182,0.9), rgba(251,191,36,0.88))',
      backgroundColor: '#43112d',
    },
    kind: 'abstract',
  },
  {
    id: 'ocean',
    label: 'Ocean Depths',
    subtitle: 'Cerulean blue gradient',
    previewStyle: {
      backgroundImage:
        'linear-gradient(135deg, rgba(14,165,233,0.78), rgba(30,64,175,0.82))',
    },
    kind: 'abstract',
  },
  {
    id: 'nebula',
    label: 'Nebula Drift',
    subtitle: 'Cosmic violet haze',
    previewStyle: {
      backgroundImage:
        'linear-gradient(135deg, rgba(147,51,234,0.78), rgba(99,102,241,0.78))',
    },
    kind: 'abstract',
  },
  {
    id: 'grid',
    label: 'Tech Grid',
    subtitle: 'Futuristic lattice backdrop',
    previewStyle: {
      backgroundColor: '#0f172a',
      backgroundImage:
        'linear-gradient(rgba(96,165,250,0.34) 1px, transparent 1px), linear-gradient(90deg, rgba(96,165,250,0.31) 1px, transparent 1px)',
      backgroundSize: '14px 14px',
    },
    kind: 'abstract',
  },
  {
    id: 'charcoal',
    label: 'Charcoal',
    subtitle: 'Deep neutral tone',
    previewStyle: {
      backgroundColor: '#0f172a',
    },
    kind: 'solid',
  },
  {
    id: 'steel',
    label: 'Steel Slate',
    subtitle: 'Cool slate finish',
    previewStyle: {
      backgroundColor: '#1f2937',
    },
    kind: 'solid',
  },
  {
    id: 'emerald',
    label: 'Evergreen',
    subtitle: 'Saturated green solid',
    previewStyle: {
      backgroundColor: '#064e3b',
    },
    kind: 'solid',
  },
  {
    id: 'none',
    label: 'No Background',
    subtitle: 'Edge-to-edge browser',
    previewStyle: {
      backgroundColor: 'transparent',
      backgroundImage:
        'linear-gradient(45deg, rgba(148,163,184,0.35) 25%, transparent 25%, transparent 50%, rgba(148,163,184,0.35) 50%, rgba(148,163,184,0.35) 75%, transparent 75%, transparent)',
      backgroundSize: '10px 10px',
    },
    kind: 'minimal',
  },
  {
    id: 'geoPrism',
    label: 'Prismatic Peaks',
    subtitle: 'Layered neon triangles',
    previewStyle: {
      backgroundColor: '#0f172a',
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
    kind: 'geometric',
  },
  {
    id: 'geoOrbit',
    label: 'Orbital Glow',
    subtitle: 'Concentric energy orbits',
    previewStyle: {
      backgroundColor: '#0b1120',
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
    kind: 'geometric',
  },
  {
    id: 'geoMosaic',
    label: 'Isometric Mosaic',
    subtitle: 'Staggered tile lattice',
    previewStyle: {
      backgroundColor: '#0b1526',
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
    kind: 'geometric',
  },
];

export const BACKGROUND_GROUP_ORDER: Array<{
  id: BackgroundKind;
  label: string;
}> = [
  { id: 'abstract', label: 'Abstract' },
  { id: 'solid', label: 'Solid' },
  { id: 'minimal', label: 'Minimal' },
  { id: 'geometric', label: 'Geometric' },
];

const HAND_POINTER_PATHS = [
  'M22 14a8 8 0 0 1-8 8',
  'M18 11v-1a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v0',
  'M14 10V9a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v1',
  'M10 9.5V4a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v10',
  'M18 11a2 2 0 1 1 4 0v3a8 8 0 0 1-8 8h-2c-2.8 0-4.5-.86-5.99-2.34l-3.6-3.6a2 2 0 0 1 2.83-2.82L7 15',
];

export const REPLAY_CURSOR_OPTIONS: CursorOption[] = [
  {
    id: 'disabled',
    group: 'hidden',
    label: 'Hidden',
    subtitle: 'No virtual cursor overlay',
    preview: (
      <span className="inline-flex h-7 w-7 items-center justify-center rounded-full border border-white/15 text-[10px] uppercase tracking-[0.18em] text-slate-400">
        Off
      </span>
    ),
  },
  {
    id: 'white',
    group: 'halo',
    label: 'Soft White',
    subtitle: 'Clean highlight for dark scenes',
    preview: (
      <span className="relative inline-flex h-7 w-7 items-center justify-center rounded-full border-2 border-white/85 bg-white/90 shadow-[0_8px_20px_rgba(148,163,184,0.4)]">
        <span className="h-1.5 w-1.5 rounded-full bg-slate-500/60" />
      </span>
    ),
  },
  {
    id: 'black',
    group: 'halo',
    label: 'Carbon Dark',
    subtitle: 'High contrast for bright scenes',
    preview: (
      <span className="relative inline-flex h-7 w-7 items-center justify-center rounded-full border-2 border-black bg-slate-900 shadow-[0_8px_20px_rgba(15,23,42,0.55)]">
        <span className="h-2 w-2 rounded-full bg-white/80" />
      </span>
    ),
  },
  {
    id: 'aura',
    group: 'halo',
    label: 'Aura Glow',
    subtitle: 'Brand accent trail',
    preview: (
      <span className="relative inline-flex h-7 w-7 items-center justify-center rounded-full border-2 border-cyan-200/80 bg-gradient-to-br from-sky-400 via-emerald-300 to-violet-400 shadow-[0_10px_22px_rgba(56,189,248,0.45)]">
        <span className="absolute -inset-0.5 rounded-full border border-cyan-300/50 opacity-70" />
      </span>
    ),
  },
  {
    id: 'arrowLight',
    group: 'arrow',
    label: 'Classic Light',
    subtitle: 'OS-style white arrow',
    preview: (
      <span className="relative inline-flex h-7 w-7 items-center justify-center text-white">
        <svg viewBox="0 0 32 32" className="h-6 w-6">
          <path
            d={REPLAY_ARROW_CURSOR_PATH}
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
    id: 'arrowDark',
    group: 'arrow',
    label: 'Noir Precision',
    subtitle: 'Deep slate pointer with halo',
    preview: (
      <span className="relative inline-flex h-7 w-7 items-center justify-center text-white">
        <svg viewBox="0 0 32 32" className="h-6 w-6">
          <path
            d={REPLAY_ARROW_CURSOR_PATH}
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
    id: 'arrowNeon',
    group: 'arrow',
    label: 'Neon Signal',
    subtitle: 'Gradient arrow with glow',
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
            d={REPLAY_ARROW_CURSOR_PATH}
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
    id: 'handNeutral',
    group: 'hand',
    label: 'Pointer Neutral',
    subtitle: 'Classic hand cursor outline',
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
    id: 'handAura',
    group: 'hand',
    label: 'Pointer Aura',
    subtitle: 'Gradient hand with halo',
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
  { id: 'hidden', label: 'Hidden' },
  { id: 'halo', label: 'Halo Cursors' },
  { id: 'arrow', label: 'Arrowhead Cursors' },
  { id: 'hand', label: 'Pointing Hands' },
];

export const REPLAY_CURSOR_CLICK_ANIMATION_OPTIONS: ClickAnimationOption[] = [
  {
    id: 'none',
    label: 'None',
    subtitle: 'No click highlight',
    preview: (
      <span className="inline-flex h-7 w-7 items-center justify-center rounded-full border border-white/15 text-[10px] uppercase tracking-[0.18em] text-slate-400">
        Off
      </span>
    ),
  },
  {
    id: 'pulse',
    label: 'Pulse',
    subtitle: 'Radial glow on click',
    preview: (
      <span className="relative inline-flex h-7 w-7 items-center justify-center">
        <span className="absolute h-6 w-6 rounded-full border border-sky-300/60 bg-sky-400/20" />
        <span className="absolute h-10 w-10 rounded-full border border-sky-400/30" />
        <span className="relative h-2 w-2 rounded-full bg-white/80" />
      </span>
    ),
  },
  {
    id: 'ripple',
    label: 'Ripple',
    subtitle: 'Expanding ring accent',
    preview: (
      <span className="relative inline-flex h-7 w-7 items-center justify-center">
        <span className="absolute h-5 w-5 rounded-full border border-violet-300/70" />
        <span className="absolute h-9 w-9 rounded-full border border-violet-400/40" />
        <span className="relative h-1.5 w-1.5 rounded-full bg-violet-200" />
      </span>
    ),
  },
];

export const REPLAY_CURSOR_POSITIONS: CursorPositionOption[] = [
  { id: 'center', label: 'Center', subtitle: 'Start replay from the middle' },
  {
    id: 'top-left',
    label: 'Top Left',
    subtitle: 'Anchor to the navigation corner',
  },
  { id: 'top-right', label: 'Top Right', subtitle: 'Anchor to utility corner' },
  {
    id: 'bottom-left',
    label: 'Bottom Left',
    subtitle: 'Anchor to lower control area',
  },
  {
    id: 'bottom-right',
    label: 'Bottom Right',
    subtitle: 'Anchor to lower action edge',
  },
  {
    id: 'random',
    label: 'Randomized',
    subtitle: 'Fresh placement each replay',
  },
];

interface HaloConfig {
  wrapperClass?: string;
  wrapperStyle?: CSSProperties;
  baseClass: string;
  baseStyle?: CSSProperties;
  dotClass?: string;
  dotStyle?: CSSProperties;
  showDot?: boolean;
  trailColor: string;
  trailWidth: number;
}

const buildHalo = (config: HaloConfig): CursorDecor => ({
  wrapperClass: config.wrapperClass,
  wrapperStyle: config.wrapperStyle,
  renderBase: (
    <span className={config.baseClass} style={config.baseStyle}>
      {config.showDot === false || !config.dotClass ? null : (
        <span className={config.dotClass} style={config.dotStyle} />
      )}
    </span>
  ),
  trailColor: config.trailColor,
  trailWidth: config.trailWidth,
});

export const buildCursorDecor = (theme: ReplayCursorTheme): CursorDecor => {
  switch (theme) {
    case 'black':
      return buildHalo({
        baseClass:
          'relative z-10 inline-flex h-5 w-5 items-center justify-center rounded-full border-2 border-black bg-slate-900 shadow-[0_10px_28px_rgba(15,23,42,0.55)]',
        dotClass: 'h-2 w-2 rounded-full bg-white/80',
        trailColor: 'rgba(30,41,59,0.7)',
        trailWidth: 2,
      });
    case 'aura':
      return buildHalo({
        wrapperClass: 'drop-shadow-[0_18px_45px_rgba(14,165,233,0.45)]',
        baseClass:
          'relative z-10 inline-flex h-6 w-6 items-center justify-center rounded-full border-2 border-cyan-200/80 bg-gradient-to-br from-sky-400 via-emerald-300 to-violet-400 shadow-[0_14px_38px_rgba(56,189,248,0.45)]',
        showDot: false,
        trailColor: 'rgba(14,165,233,0.8)',
        trailWidth: 2.3,
      });
    case 'disabled':
      return {
        renderBase: null,
        trailColor: 'rgba(0,0,0,0)',
        trailWidth: 0,
      };
    case 'arrowLight':
      return {
        wrapperClass: 'drop-shadow-[0_18px_32px_rgba(15,23,42,0.55)]',
        renderBase: (
          <span className="relative inline-flex h-8 w-8 items-center justify-center text-white">
            <svg viewBox="0 0 32 32" className="h-8 w-8">
              <path
                d={REPLAY_ARROW_CURSOR_PATH}
                fill="rgba(255,255,255,0.96)"
                stroke="rgba(15,23,42,0.85)"
                strokeWidth={1.4}
                strokeLinejoin="round"
              />
            </svg>
          </span>
        ),
        trailColor: 'rgba(148,163,184,0.55)',
        trailWidth: 1.1,
        offset: { x: 8, y: 10 },
        transformOrigin: '18% 12%',
      };
    case 'arrowDark':
      return {
        wrapperClass: 'drop-shadow-[0_20px_40px_rgba(15,23,42,0.6)]',
        renderBase: (
          <span className="relative inline-flex h-8 w-8 items-center justify-center text-white">
            <svg viewBox="0 0 32 32" className="h-8 w-8">
              <path
                d={REPLAY_ARROW_CURSOR_PATH}
                fill="rgba(30,41,59,0.95)"
                stroke="rgba(226,232,240,0.92)"
                strokeWidth={1.3}
                strokeLinejoin="round"
              />
            </svg>
          </span>
        ),
        trailColor: 'rgba(51,65,85,0.65)',
        trailWidth: 1.2,
        offset: { x: 8, y: 10 },
        transformOrigin: '18% 12%',
      };
    case 'arrowNeon':
      return {
        wrapperClass: 'drop-shadow-[0_22px_52px_rgba(59,130,246,0.55)]',
        renderBase: (
          <span className="relative inline-flex h-8 w-8 items-center justify-center text-white">
            <svg viewBox="0 0 32 32" className="h-8 w-8">
              <defs>
                <linearGradient id="replay-cursor-neon" x1="0%" y1="0%" x2="100%" y2="100%">
                  <stop offset="0%" stopColor="#38bdf8" />
                  <stop offset="45%" stopColor="#34d399" />
                  <stop offset="100%" stopColor="#a855f7" />
                </linearGradient>
              </defs>
              <path
                d={REPLAY_ARROW_CURSOR_PATH}
                fill="url(#replay-cursor-neon)"
                stroke="rgba(191,219,254,0.9)"
                strokeWidth={1.2}
                strokeLinejoin="round"
              />
            </svg>
            <span className="pointer-events-none absolute -inset-1 rounded-full bg-cyan-300/25 blur-lg" />
          </span>
        ),
        trailColor: 'rgba(56,189,248,0.7)',
        trailWidth: 1.4,
        offset: { x: 8, y: 10 },
        transformOrigin: '18% 12%',
      };
    case 'handAura':
      return {
        wrapperClass: 'drop-shadow-[0_20px_48px_rgba(56,189,248,0.4)]',
        renderBase: (
          <span className="relative inline-flex h-8 w-8 items-center justify-center">
            <svg viewBox="0 0 24 24" className="h-8 w-8" fill="none">
              <defs>
                <linearGradient id="replay-hand-gradient" x1="10%" y1="10%" x2="80%" y2="80%">
                  <stop offset="0%" stopColor="#38bdf8" />
                  <stop offset="55%" stopColor="#34d399" />
                  <stop offset="100%" stopColor="#a855f7" />
                </linearGradient>
              </defs>
              <g stroke="url(#replay-hand-gradient)" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round">
                <path d="M22 14a8 8 0 0 1-8 8" />
                <path d="M18 11v-1a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v0" />
                <path d="M14 10V9a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v1" />
                <path d="M10 9.5V4a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v10" />
                <path d="M18 11a2 2 0 1 1 4 0v3a8 8 0 0 1-8 8h-2c-2.8 0-4.5-.86-5.99-2.34l-3.6-3.6a2 2 0 0 1 2.83-2.82L7 15" />
              </g>
              <circle
                cx={18.5}
                cy={12.5}
                r={3.8}
                fill="url(#replay-hand-gradient)"
                opacity={0.12}
              />
            </svg>
          </span>
        ),
        trailColor: 'rgba(56,189,248,0.72)',
        trailWidth: 1.3,
        offset: { x: 6, y: 12 },
        transformOrigin: '46% 18%',
      };
    case 'handNeutral':
      return {
        wrapperClass: 'drop-shadow-[0_16px_36px_rgba(15,23,42,0.55)]',
        renderBase: (
          <span className="relative inline-flex h-8 w-8 items-center justify-center">
            <svg viewBox="0 0 24 24" className="h-8 w-8" fill="none">
              <g stroke="rgba(241,245,249,0.92)" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round">
                <path d="M22 14a8 8 0 0 1-8 8" />
                <path d="M18 11v-1a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v0" />
                <path d="M14 10V9a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v1" />
                <path d="M10 9.5V4a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v10" />
                <path d="M18 11a2 2 0 1 1 4 0v3a8 8 0 0 1-8 8h-2c-2.8 0-4.5-.86-5.99-2.34l-3.6-3.6a2 2 0 0 1 2.83-2.82L7 15" />
              </g>
            </svg>
          </span>
        ),
        trailColor: 'rgba(148,163,184,0.75)',
        trailWidth: 1.2,
        offset: { x: 6, y: 12 },
        transformOrigin: '46% 18%',
      };
    case 'white':
    default:
      return buildHalo({
        baseClass:
          'relative z-10 inline-flex h-5 w-5 items-center justify-center rounded-full border-2 border-white/90 bg-white/85 shadow-[0_12px_32px_rgba(148,163,184,0.45)]',
        dotClass: 'h-1.5 w-1.5 rounded-full bg-slate-500/60',
        trailColor: 'rgba(59,130,246,0.65)',
        trailWidth: 1.8,
      });
  }
};

export const buildBackgroundDecor = (theme: ReplayBackgroundTheme): BackgroundDecor => {
  switch (theme) {
    case 'sunset':
      return {
        containerClass:
          'border border-rose-200/35 shadow-[0_26px_70px_rgba(236,72,153,0.42)] bg-slate-950',
        containerStyle: {
          backgroundImage:
            'linear-gradient(135deg, rgba(244,114,182,0.92) 0%, rgba(251,191,36,0.88) 100%)',
          backgroundColor: '#43112d',
        },
        contentClass: 'p-6 sm:p-7 backdrop-blur-[1.2px]',
        baseLayer: (
          <div className="pointer-events-none absolute inset-0 overflow-hidden">
            <div
              className="absolute inset-[-20%]"
              style={{
                background:
                  'radial-gradient(95% 80% at 0% 5%, rgba(255,245,235,0.42), transparent 65%), radial-gradient(105% 85% at 100% 100%, rgba(254,215,170,0.32), transparent 70%)',
              }}
            />
            <div
              className="absolute inset-0 opacity-28 mix-blend-soft-light"
              style={{
                backgroundImage:
                  'repeating-linear-gradient(145deg, rgba(254,226,226,0.45) 0, rgba(254,226,226,0.45) 1px, transparent 1px, transparent 16px)',
              }}
            />
          </div>
        ),
        overlay: (
          <div
            className="pointer-events-none absolute inset-0 opacity-45 mix-blend-screen"
            style={{
              background:
                'radial-gradient(120% 130% at 15% 10%, rgba(236,72,153,0.3), transparent 65%), radial-gradient(125% 120% at 85% 90%, rgba(251,191,36,0.28), transparent 60%)',
            }}
          />
        ),
      };
    case 'ocean':
      return {
        containerClass:
          'border border-sky-300/25 bg-gradient-to-br from-sky-900 via-slate-950 to-slate-900 shadow-[0_24px_65px_rgba(14,165,233,0.38)]',
        contentClass: 'p-6 sm:p-7 backdrop-blur-[1px]',
        overlay: (
          <div
            className="pointer-events-none absolute inset-0 opacity-45"
            style={{
              background:
                'radial-gradient(130% 150% at 10% 10%, rgba(125,211,252,0.25), transparent 60%), radial-gradient(140% 160% at 90% 90%, rgba(14,116,144,0.28), transparent 65%)',
            }}
          />
        ),
      };
    case 'nebula':
      return {
        containerClass:
          'border border-purple-300/30 bg-gradient-to-br from-violet-800 via-indigo-950 to-slate-950 shadow-[0_26px_70px_rgba(124,58,237,0.36)]',
        contentClass: 'p-6 sm:p-7 backdrop-blur-[1.5px]',
        overlay: (
          <div
            className="pointer-events-none absolute inset-0 opacity-55 mix-blend-screen"
            style={{
              background:
                'radial-gradient(140% 140% at 0% 0%, rgba(196,181,253,0.32), transparent 65%), radial-gradient(120% 130% at 100% 90%, rgba(167,139,250,0.28), transparent 60%)',
            }}
          />
        ),
      };
    case 'grid':
      return {
        containerClass:
          'border border-cyan-300/25 bg-slate-950 shadow-[0_24px_60px_rgba(8,47,73,0.5)]',
        contentClass: 'p-6 sm:p-7',
        baseLayer: (
          <div className="pointer-events-none absolute inset-0">
            <div
              className="absolute inset-0 opacity-75"
              style={{
                background:
                  'radial-gradient(120% 140% at 12% 12%, rgba(56,189,248,0.22), transparent 65%), radial-gradient(120% 130% at 88% 88%, rgba(14,116,144,0.28), transparent 70%)',
              }}
            />
          </div>
        ),
        overlay: (
          <div
            className="pointer-events-none absolute inset-0 opacity-82 mix-blend-screen"
            style={{
              backgroundImage:
                'linear-gradient(rgba(148,163,184,0.42) 1px, transparent 1px), linear-gradient(90deg, rgba(148,163,184,0.42) 1px, transparent 1px)',
              backgroundSize: '28px 28px',
              backgroundPosition: 'center',
            }}
          />
        ),
      };
    case 'geoPrism':
      return {
        containerClass:
          'border border-cyan-200/35 bg-slate-950 shadow-[0_32px_90px_rgba(56,189,248,0.36)]',
        containerStyle: {
          backgroundColor: '#0f172a',
        },
        contentClass: 'p-6 sm:p-7 bg-slate-950/35',
        baseLayer: (
          <div className="pointer-events-none absolute inset-0 overflow-hidden">
            <img src={geometricPrismUrl} alt="" loading="lazy" className="h-full w-full object-cover" />
            <div className="absolute inset-0 bg-gradient-to-br from-cyan-400/28 via-transparent to-indigo-500/24 mix-blend-screen" />
            <div className="absolute inset-0 bg-slate-950/42" />
          </div>
        ),
        overlay: (
          <div
            className="pointer-events-none absolute inset-0"
            style={{
              background:
                'radial-gradient(140% 120% at 12% 12%, rgba(56,189,248,0.25), transparent 65%), radial-gradient(140% 130% at 88% 88%, rgba(129,140,248,0.24), transparent 60%)',
            }}
          />
        ),
      };
    case 'geoOrbit':
      return {
        containerClass:
          'border border-sky-200/35 bg-slate-950 shadow-[0_30px_88px_rgba(14,165,233,0.38)]',
        containerStyle: {
          backgroundColor: '#0b1120',
        },
        contentClass: 'p-6 sm:p-7 bg-slate-950/40',
        baseLayer: (
          <div className="pointer-events-none absolute inset-0 overflow-hidden">
            <img src={geometricOrbitUrl} alt="" loading="lazy" className="h-full w-full object-cover" />
            <div className="absolute inset-0 bg-gradient-to-br from-sky-300/22 via-transparent to-amber-300/22 mix-blend-screen" />
            <div className="absolute inset-0 bg-slate-950/45" />
          </div>
        ),
        overlay: (
          <div
            className="pointer-events-none absolute inset-0"
            style={{
              background:
                'radial-gradient(130% 120% at 18% 15%, rgba(94,234,212,0.18), transparent 65%), radial-gradient(120% 120% at 82% 85%, rgba(250,204,21,0.16), transparent 60%)',
            }}
          />
        ),
      };
    case 'geoMosaic':
      return {
        containerClass:
          'border border-amber-200/30 bg-slate-950 shadow-[0_28px_80px_rgba(245,158,11,0.32)]',
        containerStyle: {
          backgroundColor: '#0b1526',
        },
        contentClass: 'p-6 sm:p-7 bg-slate-950/38',
        baseLayer: (
          <div className="pointer-events-none absolute inset-0 overflow-hidden">
            <img src={geometricMosaicUrl} alt="" loading="lazy" className="h-full w-full object-cover" />
            <div className="absolute inset-0 bg-gradient-to-tr from-sky-400/20 via-transparent to-indigo-400/22 mix-blend-screen" />
            <div className="absolute inset-0 bg-slate-950/45" />
          </div>
        ),
        overlay: (
          <div
            className="pointer-events-none absolute inset-0"
            style={{
              background:
                'linear-gradient(160deg, rgba(14,165,233,0.18) 0%, rgba(99,102,241,0.24) 45%, transparent 78%)',
            }}
          />
        ),
      };
    case 'charcoal':
      return {
        containerClass: 'border border-slate-700/60 bg-slate-950 shadow-[0_18px_55px_rgba(15,23,42,0.55)]',
        contentClass: 'p-6 sm:p-7',
      };
    case 'steel':
      return {
        containerClass: 'border border-slate-600/60 bg-slate-800 shadow-[0_18px_52px_rgba(30,41,59,0.52)]',
        contentClass: 'p-6 sm:p-7',
      };
    case 'emerald':
      return {
        containerClass:
          'border border-emerald-300/30 bg-gradient-to-br from-emerald-900 via-emerald-950 to-slate-950 shadow-[0_22px_60px_rgba(16,185,129,0.35)]',
        contentClass: 'p-6 sm:p-7 backdrop-blur-[1px]',
        overlay: (
          <div
            className="pointer-events-none absolute inset-0 opacity-45"
            style={{
              background:
                'radial-gradient(140% 140% at 10% 15%, rgba(167,243,208,0.28), transparent 65%), radial-gradient(120% 130% at 90% 90%, rgba(45,197,253,0.22), transparent 60%)',
            }}
          />
        ),
      };
    case 'none':
      return {
        containerClass: 'border border-transparent bg-transparent shadow-none',
        contentClass: 'p-0',
      };
    case 'aurora':
    default:
      return {
        containerClass:
          'border border-white/10 bg-gradient-to-br from-slate-900 via-slate-950 to-black shadow-[0_20px_60px_rgba(15,23,42,0.4)]',
        contentClass: 'p-6 sm:p-7',
        overlay: (
          <div
            className="pointer-events-none absolute inset-0 opacity-55"
            style={{
              background:
                'radial-gradient(130% 140% at 10% 10%, rgba(56,189,248,0.25), transparent 60%), radial-gradient(120% 140% at 90% 90%, rgba(129,140,248,0.23), transparent 60%)',
            }}
          />
        ),
      };
  }
};

export const buildChromeDecor = (theme: ReplayChromeTheme, title: string): ChromeDecor => {
  switch (theme) {
    case 'chromium':
      return {
        frameClass: 'border border-[#c4c8ce] bg-[#dee1e6] shadow-[0_24px_70px_rgba(15,23,42,0.4)] rounded-lg',
        contentClass: 'bg-white',
        header: (
          <div className="bg-[#dee1e6]">
            <div className="flex items-end gap-1 px-3 pt-2">
              <div className="flex h-8 w-36 items-center rounded-t-lg border border-b-0 border-[#c4c8ce] bg-white px-3 text-[11px] font-medium text-slate-700 shadow-sm">
                <span className="truncate">New Tab</span>
              </div>
              <div className="ml-auto flex items-center gap-2 pb-2 pr-1">
                <span className="h-2.5 w-2.5 rounded-full bg-slate-400/60 hover:bg-slate-400" />
                <span className="h-2.5 w-2.5 rounded-full bg-slate-400/60 hover:bg-slate-400" />
                <span className="h-2.5 w-2.5 rounded-full bg-slate-400/60 hover:bg-slate-400" />
              </div>
            </div>
            <div className="flex items-center gap-2 border-b border-[#c4c8ce] bg-white px-3 py-2">
              <div className="flex items-center gap-1">
                <button type="button" className="inline-flex h-7 w-7 items-center justify-center rounded-full text-slate-500 hover:bg-slate-100" aria-label="Back">
                  <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" /></svg>
                </button>
                <button type="button" className="inline-flex h-7 w-7 items-center justify-center rounded-full text-slate-400 hover:bg-slate-100" aria-label="Forward">
                  <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" /></svg>
                </button>
                <button type="button" className="inline-flex h-7 w-7 items-center justify-center rounded-full text-slate-500 hover:bg-slate-100" aria-label="Reload">
                  <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" /></svg>
                </button>
              </div>
              <div className="flex-1 flex items-center rounded-full border border-[#c4c8ce] bg-[#f1f3f4] px-4 py-1.5">
                <svg className="h-3.5 w-3.5 text-slate-400 mr-2 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 11c0 3.517-1.009 6.799-2.753 9.571m-3.44-2.04l.054-.09A13.916 13.916 0 008 11a4 4 0 118 0c0 1.017-.07 2.019-.203 3m-2.118 6.844A21.88 21.88 0 0015.171 17m3.839 1.132c.645-2.266.99-4.659.99-7.132A8 8 0 008 4.07M3 15.364c.64-1.319 1-2.8 1-4.364 0-1.457.39-2.823 1.07-4" /></svg>
                <span className="text-[12px] text-slate-600 truncate">{title}</span>
              </div>
              <div className="flex items-center gap-1">
                <button type="button" className="inline-flex h-7 w-7 items-center justify-center rounded-full text-slate-500 hover:bg-slate-100" aria-label="Bookmark">
                  <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" /></svg>
                </button>
                <button type="button" className="inline-flex h-7 w-7 items-center justify-center rounded-full text-slate-500 hover:bg-slate-100" aria-label="More">
                  <svg className="h-4 w-4" fill="currentColor" viewBox="0 0 24 24"><circle cx="12" cy="5" r="1.5" /><circle cx="12" cy="12" r="1.5" /><circle cx="12" cy="19" r="1.5" /></svg>
                </button>
                <div className="ml-1 inline-flex h-7 w-7 items-center justify-center rounded-full bg-blue-500 text-[11px] font-semibold text-white">
                  U
                </div>
              </div>
            </div>
          </div>
        ),
      };
    case 'midnight':
      return {
        frameClass: 'border border-indigo-500/15 bg-gradient-to-br from-indigo-950/85 via-slate-950/80 to-black/75 shadow-[0_32px_90px_rgba(79,70,229,0.35)]',
        contentClass: 'bg-slate-950/25',
        header: (
          <div className="flex items-center justify-between border-b border-indigo-400/15 bg-gradient-to-r from-indigo-950/50 via-slate-950/35 to-indigo-900/35 px-5 py-3 text-[11px] uppercase tracking-[0.22em] text-indigo-200/80">
            <div className="flex items-center gap-3 text-indigo-300/70">
              <span className="h-2 w-8 rounded-full bg-indigo-400/70" />
              <span className="h-2 w-4 rounded-full bg-indigo-400/40" />
            </div>
            <div className="max-w-sm truncate text-indigo-100/90">{title}</div>
            <div className="flex items-center gap-2 text-indigo-300/60">
              <span className="h-2 w-2 rounded-full bg-indigo-400/70" />
              <span className="h-2 w-2 rounded-full bg-indigo-400/70" />
            </div>
          </div>
        ),
      };
    case 'minimal':
      return {
        frameClass: 'border border-transparent bg-transparent shadow-none',
        contentClass: 'bg-transparent',
        header: null,
      };
    case 'aurora':
    default:
      return {
        frameClass: 'border border-white/10 bg-slate-950/60 backdrop-blur-sm shadow-[0_20px_60px_rgba(15,23,42,0.4)]',
        contentClass: 'bg-slate-950/35',
        header: (
          <div className="flex items-center gap-3 border-b border-white/5 bg-slate-950/40 px-5 py-3 text-xs text-slate-300">
            <div className="flex items-center gap-2">
              <span className="inline-flex h-2.5 w-2.5 rounded-full bg-rose-500" />
              <span className="inline-flex h-2.5 w-2.5 rounded-full bg-amber-400" />
              <span className="inline-flex h-2.5 w-2.5 rounded-full bg-emerald-400" />
            </div>
            <div className="truncate text-slate-200">{title}</div>
          </div>
        ),
      };
  }
};
