/**
 * useDeviceFilters — DevTools-style visual filters for the preview iframe.
 *
 * Sibling to `useDeviceEmulation`. Keeps three concerns:
 *   - `colorScheme`: system | light | dark, posted to the harness child
 *     via the iframe-bridge so the preview re-themes in place (DV-001).
 *   - `visionFilter`: SVG-filter id applied to the iframe wrapper for
 *     colorblind / grayscale simulation (DV-002).
 *   - `blurPx`: 0–10px Gaussian blur applied alongside the vision filter.
 *
 * State is persisted under react-component-library.filters.v1 so the
 * picked filter survives a reload — matches the emulator's persistence
 * pattern. Color-scheme posting is the consumer's concern; the hook
 * just exposes the value and a deterministic CSS filter chain.
 */
import { useCallback, useEffect, useMemo, useState } from "react";

export const DEVICE_FILTERS_STORAGE_KEY = "react-component-library.filters.v1";

export type ColorScheme = "system" | "light" | "dark";

export const VISION_FILTERS = [
  "none",
  "grayscale",
  "protanopia",
  "deuteranopia",
  "tritanopia",
] as const;
export type VisionFilter = (typeof VISION_FILTERS)[number];

export const BLUR_MIN = 0;
export const BLUR_MAX = 10;

interface FiltersState {
  colorScheme: ColorScheme;
  visionFilter: VisionFilter;
  blurPx: number;
}

const DEFAULT_STATE: Readonly<FiltersState> = {
  colorScheme: "system",
  visionFilter: "none",
  blurPx: 0,
};

const clampBlur = (value: number): number => {
  if (!Number.isFinite(value)) return 0;
  const rounded = Math.round(value);
  return Math.min(Math.max(rounded, BLUR_MIN), BLUR_MAX);
};

const sanitize = (raw: unknown): FiltersState => {
  if (!raw || typeof raw !== "object") return { ...DEFAULT_STATE };
  const r = raw as Partial<FiltersState>;
  const cs: ColorScheme =
    r.colorScheme === "light" || r.colorScheme === "dark" || r.colorScheme === "system"
      ? r.colorScheme
      : "system";
  const vf: VisionFilter = (VISION_FILTERS as readonly string[]).includes(r.visionFilter as string)
    ? (r.visionFilter as VisionFilter)
    : "none";
  return {
    colorScheme: cs,
    visionFilter: vf,
    blurPx: typeof r.blurPx === "number" ? clampBlur(r.blurPx) : 0,
  };
};

const readPersisted = (): FiltersState => {
  if (typeof window === "undefined") return { ...DEFAULT_STATE };
  try {
    const raw = window.localStorage.getItem(DEVICE_FILTERS_STORAGE_KEY);
    if (!raw) return { ...DEFAULT_STATE };
    return sanitize(JSON.parse(raw));
  } catch {
    return { ...DEFAULT_STATE };
  }
};

const writePersisted = (state: FiltersState): void => {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(DEVICE_FILTERS_STORAGE_KEY, JSON.stringify(state));
  } catch {
    // best-effort
  }
};

/**
 * filterCSS returns the `filter:` CSS value for the current filter +
 * blur combo. Exported so tests can assert it without spinning the
 * hook through React. Empty string means "no filter applied".
 */
export function filterCSS(visionFilter: VisionFilter, blurPx: number): string {
  const parts: string[] = [];
  if (visionFilter !== "none") {
    parts.push(`url(#rcl-vision-${visionFilter})`);
  }
  if (blurPx > 0) {
    parts.push(`blur(${blurPx}px)`);
  }
  return parts.join(" ");
}

export interface DeviceFiltersValue {
  colorScheme: ColorScheme;
  visionFilter: VisionFilter;
  visionFilters: typeof VISION_FILTERS;
  blurPx: number;
  blurMin: number;
  blurMax: number;
  /** `filter:` CSS value derived from visionFilter + blurPx. */
  filterCSS: string;
  setColorScheme: (cs: ColorScheme) => void;
  setVisionFilter: (vf: VisionFilter) => void;
  setBlurPx: (blur: number) => void;
  reset: () => void;
}

export function useDeviceFilters(): DeviceFiltersValue {
  const [state, setState] = useState<FiltersState>(() => readPersisted());

  useEffect(() => {
    writePersisted(state);
  }, [state]);

  const setColorScheme = useCallback((cs: ColorScheme) => {
    setState((prev) => ({ ...prev, colorScheme: cs }));
  }, []);

  const setVisionFilter = useCallback((vf: VisionFilter) => {
    setState((prev) => ({ ...prev, visionFilter: vf }));
  }, []);

  const setBlurPx = useCallback((blur: number) => {
    setState((prev) => ({ ...prev, blurPx: clampBlur(blur) }));
  }, []);

  const reset = useCallback(() => {
    setState({ ...DEFAULT_STATE });
  }, []);

  const css = useMemo(
    () => filterCSS(state.visionFilter, state.blurPx),
    [state.visionFilter, state.blurPx],
  );

  return {
    colorScheme: state.colorScheme,
    visionFilter: state.visionFilter,
    visionFilters: VISION_FILTERS,
    blurPx: state.blurPx,
    blurMin: BLUR_MIN,
    blurMax: BLUR_MAX,
    filterCSS: css,
    setColorScheme,
    setVisionFilter,
    setBlurPx,
    reset,
  };
}
