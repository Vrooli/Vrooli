/**
 * Chrome theme builders for the ReplayPlayer component
 *
 * Contains the buildChromeDecor function that generates browser chrome decorations
 * for different themes (aurora, chromium, midnight, minimal).
 */

import type { ChromeDecor, ReplayChromeTheme } from '../types';

export const buildChromeDecor = (theme: ReplayChromeTheme, title: string): ChromeDecor => {
  switch (theme) {
    case 'chromium':
      return {
        frameClass: 'border border-[#c4c8ce] bg-[#dee1e6] shadow-[0_24px_70px_rgba(15,23,42,0.4)] rounded-lg',
        contentClass: 'bg-white',
        header: (
          <div className="bg-[#dee1e6]">
            {/* Tab bar */}
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
            {/* Toolbar */}
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
