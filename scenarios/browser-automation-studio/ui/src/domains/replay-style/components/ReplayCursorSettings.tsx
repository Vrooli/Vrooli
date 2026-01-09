import clsx from 'clsx';
import { ChevronDown } from 'lucide-react';
import { useCallback, useEffect, useMemo, useRef, useState, type MutableRefObject } from 'react';
import { RangeSlider } from '@shared/ui';
import {
  CURSOR_GROUP_ORDER,
  REPLAY_CURSOR_CLICK_ANIMATION_OPTIONS,
  REPLAY_CURSOR_OPTIONS,
  REPLAY_CURSOR_POSITIONS,
  getReplayCursorClickAnimationOption,
  getReplayCursorOption,
  getReplayCursorPositionOption,
  type ClickAnimationOption,
  type CursorOption,
  type CursorPositionOption,
} from '../catalog';
import { MAX_CURSOR_SCALE, MIN_CURSOR_SCALE } from '../constants';
import {
  clampReplayCursorScale,
  type ReplayCursorClickAnimation,
  type ReplayCursorInitialPosition,
  type ReplayCursorTheme,
} from '../model';

type ReplayCursorSettingsVariant = 'settings' | 'compact';

interface ReplayCursorSettingsProps {
  cursorTheme: ReplayCursorTheme;
  cursorInitialPosition: ReplayCursorInitialPosition;
  cursorClickAnimation: ReplayCursorClickAnimation;
  cursorScale: number;
  onCursorThemeChange: (value: ReplayCursorTheme) => void;
  onCursorInitialPositionChange: (value: ReplayCursorInitialPosition) => void;
  onCursorClickAnimationChange: (value: ReplayCursorClickAnimation) => void;
  onCursorScaleChange: (value: number) => void;
  variant?: ReplayCursorSettingsVariant;
}

const setupDismissListeners = (
  isOpen: boolean,
  ref: MutableRefObject<HTMLElement | null>,
  onClose: () => void,
) => {
  if (!isOpen || typeof document === 'undefined') {
    return undefined;
  }
  const handlePointerDown = (event: MouseEvent) => {
    const target = event.target as Node | null;
    if (!target) return;
    if (ref.current && !ref.current.contains(target)) {
      onClose();
    }
  };
  const handleKeyDown = (event: KeyboardEvent) => {
    if (event.key === 'Escape') {
      onClose();
    }
  };
  document.addEventListener('mousedown', handlePointerDown);
  document.addEventListener('keydown', handleKeyDown);
  return () => {
    document.removeEventListener('mousedown', handlePointerDown);
    document.removeEventListener('keydown', handleKeyDown);
  };
};

const getCursorOptionsByGroup = (): Record<CursorOption['group'], CursorOption[]> => {
  const base: Record<CursorOption['group'], CursorOption[]> = {
    hidden: [],
    halo: [],
    arrow: [],
    hand: [],
  };
  for (const option of REPLAY_CURSOR_OPTIONS) {
    base[option.group].push(option);
  }
  return base;
};

const buildSettingsCardClass = (isActive: boolean) =>
  clsx(
    'flex flex-col items-center p-3 rounded-lg border transition-all text-center',
    isActive
      ? 'border-flow-accent bg-flow-accent/10 ring-1 ring-flow-accent/50'
      : 'border-gray-700 hover:border-gray-600 bg-gray-800/30',
  );

export function ReplayCursorSettings({
  cursorTheme,
  cursorInitialPosition,
  cursorClickAnimation,
  cursorScale,
  onCursorThemeChange,
  onCursorInitialPositionChange,
  onCursorClickAnimationChange,
  onCursorScaleChange,
  variant = 'settings',
}: ReplayCursorSettingsProps) {
  const isCompact = variant === 'compact';

  const cursorOptionsByGroup = useMemo(() => getCursorOptionsByGroup(), []);
  const selectedCursorOption = useMemo(() => getReplayCursorOption(cursorTheme), [cursorTheme]);
  const selectedCursorPositionOption = useMemo(
    () => getReplayCursorPositionOption(cursorInitialPosition),
    [cursorInitialPosition],
  );
  const selectedCursorClickAnimationOption = useMemo<ClickAnimationOption>(
    () => getReplayCursorClickAnimationOption(cursorClickAnimation),
    [cursorClickAnimation],
  );

  const [isCursorMenuOpen, setIsCursorMenuOpen] = useState(false);
  const [isCursorPositionMenuOpen, setIsCursorPositionMenuOpen] = useState(false);
  const [isCursorClickAnimationMenuOpen, setIsCursorClickAnimationMenuOpen] = useState(false);
  const cursorSelectorRef = useRef<HTMLDivElement | null>(null);
  const cursorPositionSelectorRef = useRef<HTMLDivElement | null>(null);
  const cursorClickAnimationSelectorRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => setupDismissListeners(isCursorMenuOpen, cursorSelectorRef, () => setIsCursorMenuOpen(false)), [isCursorMenuOpen]);
  useEffect(
    () =>
      setupDismissListeners(
        isCursorPositionMenuOpen,
        cursorPositionSelectorRef,
        () => setIsCursorPositionMenuOpen(false),
      ),
    [isCursorPositionMenuOpen],
  );
  useEffect(
    () =>
      setupDismissListeners(
        isCursorClickAnimationMenuOpen,
        cursorClickAnimationSelectorRef,
        () => setIsCursorClickAnimationMenuOpen(false),
      ),
    [isCursorClickAnimationMenuOpen],
  );

  const handleCursorThemeSelect = useCallback(
    (value: ReplayCursorTheme) => {
      onCursorThemeChange(value);
      setIsCursorMenuOpen(false);
    },
    [onCursorThemeChange],
  );
  const handleCursorPositionSelect = useCallback(
    (value: ReplayCursorInitialPosition) => {
      onCursorInitialPositionChange(value);
      setIsCursorPositionMenuOpen(false);
    },
    [onCursorInitialPositionChange],
  );
  const handleCursorClickAnimationSelect = useCallback(
    (value: ReplayCursorClickAnimation) => {
      onCursorClickAnimationChange(value);
      setIsCursorClickAnimationMenuOpen(false);
    },
    [onCursorClickAnimationChange],
  );
  const handleCursorScaleChange = useCallback(
    (value: number) => {
      if (!Number.isFinite(value)) {
        return;
      }
      onCursorScaleChange(clampReplayCursorScale(value));
    },
    [onCursorScaleChange],
  );

  if (isCompact) {
    return (
      <div className="space-y-4">
        <div className="flex flex-col gap-3 lg:flex-row lg:flex-wrap lg:items-start lg:justify-between">
          <div ref={cursorSelectorRef} className="relative flex-1 min-w-[220px]">
            <button
              type="button"
              className={clsx(
                'flex w-full items-center justify-between rounded-xl border px-3 py-2 text-left text-sm transition-all focus:outline-none focus:ring-2 focus:ring-flow-accent/70 focus:ring-offset-2 focus:ring-offset-slate-900',
                isCursorMenuOpen
                  ? 'border-flow-accent/70 bg-slate-900/80 text-white'
                  : 'border-white/10 bg-slate-900/60 text-slate-200 hover:border-flow-accent/40 hover:text-white',
              )}
              onClick={() => {
                setIsCursorMenuOpen(!isCursorMenuOpen);
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
                  'ml-3 flex-shrink-0 text-slate-400 transition-transform duration-150',
                  isCursorMenuOpen ? 'rotate-180 text-white' : '',
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
                          const isActive = cursorTheme === option.id;
                          return (
                            <button
                              key={option.id}
                              type="button"
                              className={clsx(
                                'flex w-full items-center gap-3 rounded-lg border px-2 py-2 text-left text-xs transition-all',
                                isActive
                                  ? 'border-flow-accent/70 bg-flow-accent/10 text-white'
                                  : 'border-transparent text-slate-300 hover:border-white/10 hover:bg-white/5',
                              )}
                              onClick={() => handleCursorThemeSelect(option.id)}
                            >
                              <span className="flex h-8 w-10 items-center justify-center rounded-lg border border-white/10 bg-slate-900/60">
                                {option.preview}
                              </span>
                              <span className="flex flex-col">
                                <span className="text-sm font-medium text-white">{option.label}</span>
                                <span className="text-[11px] text-slate-400">{option.subtitle}</span>
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

          <div ref={cursorPositionSelectorRef} className="relative flex-1 min-w-[200px]">
            <button
              type="button"
              className={clsx(
                'flex w-full items-center justify-between rounded-xl border px-3 py-2 text-left text-sm transition-all focus:outline-none focus:ring-2 focus:ring-flow-accent/70 focus:ring-offset-2 focus:ring-offset-slate-900',
                isCursorPositionMenuOpen
                  ? 'border-flow-accent/70 bg-slate-900/80 text-white'
                  : 'border-white/10 bg-slate-900/60 text-slate-200 hover:border-flow-accent/40 hover:text-white',
              )}
              onClick={() => {
                setIsCursorPositionMenuOpen(!isCursorPositionMenuOpen);
                setIsCursorMenuOpen(false);
                setIsCursorClickAnimationMenuOpen(false);
              }}
              aria-haspopup="menu"
              aria-expanded={isCursorPositionMenuOpen}
            >
              <span className="flex flex-col gap-1">
                <span className="text-[11px] uppercase tracking-[0.24em] text-slate-400">Start position</span>
                <span className="text-sm font-medium text-white">{selectedCursorPositionOption.label}</span>
                <span className="text-[11px] text-slate-400">{selectedCursorPositionOption.subtitle}</span>
              </span>
              <ChevronDown
                size={14}
                className={clsx(
                  'ml-3 flex-shrink-0 text-slate-400 transition-transform duration-150',
                  isCursorPositionMenuOpen ? 'rotate-180 text-white' : '',
                )}
              />
            </button>
            {isCursorPositionMenuOpen && (
              <div
                role="menu"
                className="absolute right-0 z-30 mt-2 w-full rounded-xl border border-white/10 bg-slate-950/95 p-2 shadow-[0_20px_50px_rgba(15,23,42,0.55)] backdrop-blur"
              >
                {REPLAY_CURSOR_POSITIONS.map((option) => {
                  const isActive = cursorInitialPosition === option.id;
                  return (
                    <button
                      key={option.id}
                      type="button"
                      className={clsx(
                        'flex w-full flex-col gap-1 rounded-lg border px-2 py-2 text-left text-xs transition-all',
                        isActive
                          ? 'border-flow-accent/70 bg-flow-accent/10 text-white'
                          : 'border-transparent text-slate-300 hover:border-white/10 hover:bg-white/5',
                      )}
                      onClick={() => handleCursorPositionSelect(option.id)}
                    >
                      <span className="text-sm font-medium text-white">{option.label}</span>
                      <span className="text-[11px] text-slate-400">{option.subtitle}</span>
                    </button>
                  );
                })}
              </div>
            )}
          </div>

          <div ref={cursorClickAnimationSelectorRef} className="relative flex flex-1 flex-col gap-2 lg:max-w-xs">
            <button
              type="button"
              className={clsx(
                'flex w-full items-center justify-between rounded-xl border px-3 py-2 text-left text-sm transition-all focus:outline-none focus:ring-2 focus:ring-flow-accent/70 focus:ring-offset-2 focus:ring-offset-slate-900',
                isCursorClickAnimationMenuOpen
                  ? 'border-flow-accent/70 bg-slate-900/80 text-white'
                  : 'border-white/10 bg-slate-900/60 text-slate-200 hover:border-flow-accent/40 hover:text-white',
              )}
              onClick={() => {
                setIsCursorClickAnimationMenuOpen(!isCursorClickAnimationMenuOpen);
                setIsCursorMenuOpen(false);
                setIsCursorPositionMenuOpen(false);
              }}
              aria-haspopup="menu"
              aria-expanded={isCursorClickAnimationMenuOpen}
            >
              <span className="flex flex-col gap-1">
                <span className="text-[11px] uppercase tracking-[0.24em] text-slate-400">Click animation</span>
                <span className="text-sm font-medium text-white">{selectedCursorClickAnimationOption.label}</span>
                <span className="text-[11px] text-slate-400">{selectedCursorClickAnimationOption.subtitle}</span>
              </span>
              <ChevronDown
                size={14}
                className={clsx(
                  'ml-3 flex-shrink-0 text-slate-400 transition-transform duration-150',
                  isCursorClickAnimationMenuOpen ? 'rotate-180 text-white' : '',
                )}
              />
            </button>
            {isCursorClickAnimationMenuOpen && (
              <div
                role="menu"
                className="absolute right-0 z-30 mt-2 w-full rounded-xl border border-white/10 bg-slate-950/95 p-2 shadow-[0_20px_50px_rgba(15,23,42,0.55)] backdrop-blur"
              >
                {REPLAY_CURSOR_CLICK_ANIMATION_OPTIONS.map((option) => {
                  const isActive = cursorClickAnimation === option.id;
                  return (
                    <button
                      key={option.id}
                      type="button"
                      className={clsx(
                        'flex w-full items-center gap-3 rounded-lg border px-2 py-2 text-left text-xs transition-all',
                        isActive
                          ? 'border-flow-accent/70 bg-flow-accent/10 text-white'
                          : 'border-transparent text-slate-300 hover:border-white/10 hover:bg-white/5',
                      )}
                      onClick={() => handleCursorClickAnimationSelect(option.id)}
                    >
                      <span className="flex h-8 w-10 items-center justify-center rounded-lg border border-white/10 bg-slate-900/60">
                        {option.preview}
                      </span>
                      <span className="flex flex-col">
                        <span className="text-sm font-medium text-white">{option.label}</span>
                        <span className="text-[11px] text-slate-400">{option.subtitle}</span>
                      </span>
                    </button>
                  );
                })}
              </div>
            )}
          </div>
        </div>

        <div className="rounded-xl border border-white/10 bg-slate-900/60 px-3 py-2">
          <div className="flex items-center justify-between text-[11px] uppercase tracking-[0.24em] text-slate-400">
            <span>Cursor scale</span>
            <span>{Math.round(cursorScale * 100)}%</span>
          </div>
        <RangeSlider
          min={MIN_CURSOR_SCALE}
          max={MAX_CURSOR_SCALE}
          step={0.05}
          value={cursorScale}
          onChange={handleCursorScaleChange}
          ariaLabel="Cursor scale"
          className="mt-2"
        />
          <div className="mt-1 text-[11px] text-slate-500">
            Scale how prominent the cursor appears.
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div>
        <div className="text-xs font-medium text-gray-500 uppercase tracking-wide mb-2">
          Cursor style
        </div>
        <div className="space-y-4">
          {CURSOR_GROUP_ORDER.map((group) => {
            const options = cursorOptionsByGroup[group.id];
            if (!options || options.length === 0) {
              return null;
            }
            return (
              <div key={group.id}>
                <div className="text-xs font-medium text-gray-500 uppercase tracking-wide mb-2">
                  {group.label}
                </div>
                <div className="grid grid-cols-2 sm:grid-cols-3 gap-2">
                  {options.map((option) => {
                    const isActive = cursorTheme === option.id;
                    return (
                      <button
                        key={option.id}
                        type="button"
                        onClick={() => onCursorThemeChange(option.id)}
                        className={buildSettingsCardClass(isActive)}
                      >
                        <div className="mb-2">{option.preview}</div>
                        <span className="text-sm font-medium text-surface">{option.label}</span>
                        <span className="text-xs text-gray-500 mt-0.5">{option.subtitle}</span>
                      </button>
                    );
                  })}
                </div>
              </div>
            );
          })}
        </div>
      </div>

      <div>
        <div className="flex items-center justify-between mb-2">
          <span className="text-sm text-gray-300">Cursor size</span>
          <span className="text-sm text-gray-400">{Math.round(cursorScale * 100)}%</span>
        </div>
        <RangeSlider
          min={MIN_CURSOR_SCALE}
          max={MAX_CURSOR_SCALE}
          step={0.1}
          value={cursorScale}
          onChange={handleCursorScaleChange}
          ariaLabel="Cursor size"
        />
      </div>

      <div>
        <div className="text-xs font-medium text-gray-500 uppercase tracking-wide mb-2">
          Click animation
        </div>
        <div className="grid grid-cols-2 sm:grid-cols-3 gap-2">
          {REPLAY_CURSOR_CLICK_ANIMATION_OPTIONS.map((option: ClickAnimationOption) => {
            const isActive = cursorClickAnimation === option.id;
            return (
              <button
                key={option.id}
                type="button"
                onClick={() => onCursorClickAnimationChange(option.id)}
                className={buildSettingsCardClass(isActive)}
              >
                <div className="mb-2">{option.preview}</div>
                <span className="text-sm font-medium text-surface">{option.label}</span>
                <span className="text-xs text-gray-500 mt-0.5">{option.subtitle}</span>
              </button>
            );
          })}
        </div>
      </div>

      <div>
        <div className="text-xs font-medium text-gray-500 uppercase tracking-wide mb-2">
          Initial cursor position
        </div>
        <div className="grid grid-cols-2 sm:grid-cols-3 gap-2">
          {REPLAY_CURSOR_POSITIONS.map((option: CursorPositionOption) => {
            const isActive = cursorInitialPosition === option.id;
            return (
              <button
                key={option.id}
                type="button"
                onClick={() => onCursorInitialPositionChange(option.id)}
                className={buildSettingsCardClass(isActive)}
              >
                <span className="text-sm font-medium text-surface">{option.label}</span>
                <span className="text-xs text-gray-500 mt-0.5">{option.subtitle}</span>
              </button>
            );
          })}
        </div>
      </div>
    </div>
  );
}
