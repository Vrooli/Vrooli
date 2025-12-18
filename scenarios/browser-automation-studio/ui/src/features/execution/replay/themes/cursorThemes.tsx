/**
 * Cursor theme builders for the ReplayPlayer component
 *
 * Contains the buildCursorDecor function that generates visual decorations
 * for different cursor themes (white, black, aura, arrow*, hand*, disabled).
 */

import type { CSSProperties } from 'react';
import type { CursorDecor, ReplayCursorTheme } from '../types';
import { REPLAY_ARROW_CURSOR_PATH } from '../constants';

// =============================================================================
// Helper Functions
// =============================================================================

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

// =============================================================================
// Main Builder
// =============================================================================

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
