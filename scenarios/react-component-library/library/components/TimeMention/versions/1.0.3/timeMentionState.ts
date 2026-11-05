import {
  createContext,
  createElement,
  useContext,
  useMemo,
  useSyncExternalStore,
  type ReactNode,
} from "react";
import { TIME_MENTION_DISPLAYS, type TimeMentionDisplay } from "./timeMentionResolve";

/**
 * Shared state for every time mention on the page: the frame a host provides,
 * the reader's display preference, and one clock.
 *
 * The display preference belongs to the reader, not to a mention: switching it
 * from any mention's menu changes every mention on screen, and the choice is
 * remembered. A host that keeps its own settings can take it over with
 * `display` and `onDisplayChange`.
 */

export type TimeAnchorInput = number | string | Date | null | undefined;

export interface TimeMentionSettings {
  /** When the text was written. Without it, mentions that need it render as plain text. */
  anchor?: TimeAnchorInput;
  /** The reader's IANA zone; defaults to the browser's. */
  timeZone?: string;
  /** BCP 47 locale for formatting; defaults to the document language. */
  locale?: string;
  /** A fixed current time, for stories, tests and screenshots. */
  now?: number;
  display?: TimeMentionDisplay;
  onDisplayChange?: (display: TimeMentionDisplay) => void;
  /** False renders every mention as plain text. */
  enabled?: boolean;
}

const TimeMentionContext = createContext<TimeMentionSettings>({});

/** Provides settings to the mentions below it; unset fields inherit from an outer provider. */
export function TimeMentionProvider({
  children,
  ...settings
}: TimeMentionSettings & { children?: ReactNode }) {
  const outer = useContext(TimeMentionContext);
  const { anchor, timeZone, locale, now, display, onDisplayChange, enabled } = settings;
  const value = useMemo(() => {
    const merged: TimeMentionSettings = { ...outer };
    if (anchor !== undefined) merged.anchor = anchor;
    if (timeZone !== undefined) merged.timeZone = timeZone;
    if (locale !== undefined) merged.locale = locale;
    if (now !== undefined) merged.now = now;
    if (display !== undefined) merged.display = display;
    if (onDisplayChange !== undefined) merged.onDisplayChange = onDisplayChange;
    if (enabled !== undefined) merged.enabled = enabled;
    return merged;
  }, [outer, anchor, timeZone, locale, now, display, onDisplayChange, enabled]);
  return createElement(TimeMentionContext.Provider, { value }, children);
}

export function useTimeMentionSettings(): TimeMentionSettings {
  return useContext(TimeMentionContext);
}

export function normalizeAnchor(anchor: TimeAnchorInput): number | null {
  if (anchor === null || anchor === undefined || anchor === "") return null;
  const ms =
    anchor instanceof Date
      ? anchor.getTime()
      : typeof anchor === "number"
        ? anchor
        : Date.parse(anchor);
  return Number.isFinite(ms) ? ms : null;
}

// ── Display preference ──────────────────────────────────────────────────────

export const TIME_MENTION_DISPLAY_STORAGE_KEY = "rcl.time-mention.display";
const displayListeners = new Set<() => void>();
let storedDisplay: TimeMentionDisplay | null = null;

function isDisplay(value: unknown): value is TimeMentionDisplay {
  return TIME_MENTION_DISPLAYS.includes(value as TimeMentionDisplay);
}

function readStoredDisplay(): TimeMentionDisplay {
  if (storedDisplay) return storedDisplay;
  try {
    const value = globalThis.localStorage.getItem(TIME_MENTION_DISPLAY_STORAGE_KEY);
    storedDisplay = isDisplay(value) ? value : "annotate";
  } catch {
    storedDisplay = "annotate";
  }
  return storedDisplay;
}

function onStorage(event: StorageEvent) {
  if (event.key !== TIME_MENTION_DISPLAY_STORAGE_KEY) return;
  storedDisplay = isDisplay(event.newValue) ? event.newValue : "annotate";
  displayListeners.forEach((listener) => listener());
}

function subscribeDisplay(listener: () => void) {
  if (displayListeners.size === 0 && typeof window !== "undefined")
    window.addEventListener("storage", onStorage);
  displayListeners.add(listener);
  return () => {
    displayListeners.delete(listener);
    if (displayListeners.size === 0 && typeof window !== "undefined")
      window.removeEventListener("storage", onStorage);
  };
}

export function setStoredTimeMentionDisplay(display: TimeMentionDisplay) {
  storedDisplay = display;
  try {
    globalThis.localStorage.setItem(TIME_MENTION_DISPLAY_STORAGE_KEY, display);
  } catch {
    // Storage can be unavailable (private mode); the choice still applies to this page.
  }
  displayListeners.forEach((listener) => listener());
}

/** The reader's display preference and its setter, shared by every mention. */
export function useTimeMentionDisplay(): [
  TimeMentionDisplay,
  (display: TimeMentionDisplay) => void,
] {
  const settings = useTimeMentionSettings();
  const stored = useSyncExternalStore(
    subscribeDisplay,
    readStoredDisplay,
    () => "annotate" as const,
  );
  if (settings.display !== undefined)
    return [settings.display, settings.onDisplayChange ?? setStoredTimeMentionDisplay];
  return [stored, settings.onDisplayChange ?? setStoredTimeMentionDisplay];
}

// ── Clock ───────────────────────────────────────────────────────────────────

// One interval serves every mention on the page, and only while one is mounted.
// Thirty seconds keeps minute-granular countdowns honest without re-rendering
// long transcripts every second.
const CLOCK_INTERVAL_MS = 30_000;
const clockListeners = new Set<() => void>();
let clockTick = Date.now();
let clockTimer: ReturnType<typeof setInterval> | undefined;

function advanceClock() {
  clockTick = Date.now();
  clockListeners.forEach((listener) => listener());
}

function onVisibility() {
  if (typeof document !== "undefined" && document.visibilityState === "visible") advanceClock();
}

function subscribeClock(listener: () => void) {
  if (clockListeners.size === 0) {
    clockTick = Date.now();
    clockTimer = setInterval(advanceClock, CLOCK_INTERVAL_MS);
    if (typeof document !== "undefined")
      document.addEventListener("visibilitychange", onVisibility);
  }
  clockListeners.add(listener);
  return () => {
    clockListeners.delete(listener);
    if (clockListeners.size === 0) {
      clearInterval(clockTimer);
      clockTimer = undefined;
      if (typeof document !== "undefined")
        document.removeEventListener("visibilitychange", onVisibility);
    }
  };
}

/** The current time, from the provider when fixed, else the shared clock. */
export function useTimeMentionNow(): number {
  const settings = useTimeMentionSettings();
  const tick = useSyncExternalStore(
    subscribeClock,
    () => clockTick,
    () => clockTick,
  );
  return settings.now ?? tick;
}

export function defaultTimeZone(): string {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC";
  } catch {
    return "UTC";
  }
}
