/**
 * Background theme builders for the ReplayPlayer component
 *
 * Contains the buildBackgroundDecor function that generates visual decorations
 * for different background themes (sunset, ocean, nebula, grid, geo*, etc.)
 */

import type { BackgroundDecor, ReplayBackgroundTheme } from '../types';

// Background images - using ES imports for reliable Vite asset handling
import geometricPrismUrl from '../../../../assets/replay-backgrounds/geometric-prism.jpg';
import geometricOrbitUrl from '../../../../assets/replay-backgrounds/geometric-orbit.jpg';
import geometricMosaicUrl from '../../../../assets/replay-backgrounds/geometric-mosaic.jpg';

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
