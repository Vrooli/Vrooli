/**
 * useDeviceEmulation — viewport state machine for the preview iframe.
 *
 * Re-derived from requirement module 04-multi-viewport-emulator (req
 * VP-001..004). Distinct from app-monitor's same-named hook: rcl emulates
 * a *component* preview inside the editor workbench. Its toolbar mirrors
 * app-monitor's DevTools-style control shape, including responsive width
 * and height inputs, but keeps all state local to this scenario.
 */
import { useCallback, useEffect, useMemo, useState } from "react";
import { VIEWPORT_AXIS_PRESETS } from "./viewportAxis";

export const DEVICE_EMULATION_STORAGE_KEY = "react-component-library.emulator.v1";

const axisPreset = (id: string) => {
  const preset = VIEWPORT_AXIS_PRESETS.find((candidate) => candidate.id === id);
  if (!preset) throw new Error(`Unknown experience-manager viewport axis: ${id}`);
  return preset;
};

export const DEVICE_PRESETS = [
  { ...axisPreset("mobile"), label: "Mobile" },
  { ...axisPreset("tablet"), label: "Tablet" },
  { ...axisPreset("desktop"), label: "Desktop" },
  { ...axisPreset("wide"), label: "Wide" },
  {
    id: "responsive",
    label: "Responsive",
    width: axisPreset("desktop").width,
    height: axisPreset("desktop").height,
  },
] as const;

export type DevicePresetId = (typeof DEVICE_PRESETS)[number]["id"];

export const ZOOM_MIN = 0.1;
export const ZOOM_MAX = 2.0;
const ZOOM_STEP = 0.1;
export const DEVICE_ZOOM_LEVELS = [
  0.1, 0.2, 0.25, 0.33, 0.5, 0.67, 0.75, 0.9, 1, 1.25, 1.5, 2,
] as const;

const DEFAULT_PRESET_ID: DevicePresetId = "desktop";
const DEFAULT_CUSTOM_WIDTH = 1280;
const DEFAULT_CUSTOM_HEIGHT = 720;
const DIMENSION_MIN = 1;
const DIMENSION_MAX = 2400;

interface EmulatorState {
  presetId: DevicePresetId;
  customWidth: number;
  customHeight: number;
  zoom: number;
  isRotated: boolean;
}

const DEFAULT_STATE: Readonly<EmulatorState> = {
  presetId: DEFAULT_PRESET_ID,
  customWidth: DEFAULT_CUSTOM_WIDTH,
  customHeight: DEFAULT_CUSTOM_HEIGHT,
  zoom: 1,
  isRotated: false,
};

const clampDimension = (value: number): number => {
  if (!Number.isFinite(value)) return DIMENSION_MIN;
  return Math.min(Math.max(Math.round(value), DIMENSION_MIN), DIMENSION_MAX);
};

const clampZoom = (value: number): number => {
  if (!Number.isFinite(value)) return 1;
  const rounded = Math.round(value * 100) / 100;
  return Math.min(Math.max(rounded, ZOOM_MIN), ZOOM_MAX);
};

const sanitize = (raw: unknown): EmulatorState => {
  if (!raw || typeof raw !== "object") return { ...DEFAULT_STATE };
  const r = raw as Partial<EmulatorState>;
  const preset = DEVICE_PRESETS.find((p) => p.id === r.presetId);
  return {
    presetId: preset?.id ?? DEFAULT_PRESET_ID,
    customWidth:
      typeof r.customWidth === "number" ? clampDimension(r.customWidth) : DEFAULT_CUSTOM_WIDTH,
    customHeight:
      typeof r.customHeight === "number" ? clampDimension(r.customHeight) : DEFAULT_CUSTOM_HEIGHT,
    zoom: typeof r.zoom === "number" ? clampZoom(r.zoom) : 1,
    isRotated: Boolean(r.isRotated),
  };
};

const readPersisted = (): EmulatorState => {
  if (typeof window === "undefined") return { ...DEFAULT_STATE };
  try {
    const raw = window.localStorage.getItem(DEVICE_EMULATION_STORAGE_KEY);
    if (!raw) return { ...DEFAULT_STATE };
    return sanitize(JSON.parse(raw));
  } catch {
    return { ...DEFAULT_STATE };
  }
};

const writePersisted = (state: EmulatorState): void => {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(DEVICE_EMULATION_STORAGE_KEY, JSON.stringify(state));
  } catch {
    // best-effort; storage quota / private mode failures must not crash
  }
};

export interface DeviceEmulationValue {
  presets: typeof DEVICE_PRESETS;
  presetId: DevicePresetId;
  zoom: number;
  zoomLevels: typeof DEVICE_ZOOM_LEVELS;
  zoomMin: number;
  zoomMax: number;
  isRotated: boolean;
  isResponsive: boolean;
  /** Display dimensions = preset dims, swapped if rotated. */
  displayWidth: number;
  displayHeight: number;
  /** On-screen dimensions after CSS scale. */
  scaledWidth: number;
  scaledHeight: number;
  setPreset: (id: DevicePresetId) => void;
  setDimension: (dimension: "width" | "height", value: number) => void;
  setZoom: (zoom: number) => void;
  zoomIn: () => void;
  zoomOut: () => void;
  resetZoom: () => void;
  fitToPane: () => void;
  rotate: () => void;
  reset: () => void;
}

export function useDeviceEmulation(): DeviceEmulationValue {
  const [state, setState] = useState<EmulatorState>(() => readPersisted());

  useEffect(() => {
    writePersisted(state);
  }, [state]);

  const preset = useMemo(
    () => DEVICE_PRESETS.find((p) => p.id === state.presetId) ?? DEVICE_PRESETS[0],
    [state.presetId],
  );

  const baseWidth = state.presetId === "responsive" ? state.customWidth : preset.width;
  const baseHeight = state.presetId === "responsive" ? state.customHeight : preset.height;
  const displayWidth = state.isRotated ? baseHeight : baseWidth;
  const displayHeight = state.isRotated ? baseWidth : baseHeight;
  const isResponsive = state.presetId === "responsive";

  const setPreset = useCallback((id: DevicePresetId) => {
    setState((prev) => sanitize({ ...prev, presetId: id }));
  }, []);

  const setDimension = useCallback((dimension: "width" | "height", value: number) => {
    setState((prev) => {
      const next = prev.isRotated
        ? dimension === "width"
          ? { ...prev, customHeight: value }
          : { ...prev, customWidth: value }
        : dimension === "width"
          ? { ...prev, customWidth: value }
          : { ...prev, customHeight: value };
      return sanitize({ ...next, presetId: "responsive" });
    });
  }, []);

  const setZoom = useCallback((zoom: number) => {
    setState((prev) => ({ ...prev, zoom: clampZoom(zoom) }));
  }, []);

  const zoomIn = useCallback(() => {
    setState((prev) => ({ ...prev, zoom: clampZoom(prev.zoom + ZOOM_STEP) }));
  }, []);

  const zoomOut = useCallback(() => {
    setState((prev) => ({ ...prev, zoom: clampZoom(prev.zoom - ZOOM_STEP) }));
  }, []);

  const resetZoom = useCallback(() => {
    setState((prev) => ({ ...prev, zoom: 1 }));
  }, []);

  const fitToPane = useCallback(() => {
    const frame = document.querySelector<HTMLElement>("[data-emulator-viewport-frame]");
    const availableWidth = frame?.clientWidth ?? 0;
    const availableHeight = frame?.clientHeight ?? 0;
    if (availableWidth <= 0 || availableHeight <= 0) {
      setState((prev) => ({ ...prev, zoom: 1 }));
      return;
    }
    const horizontalPadding = 24;
    const nextZoom = Math.min(
      1,
      (availableWidth - horizontalPadding) / displayWidth,
      (availableHeight - horizontalPadding) / displayHeight,
    );
    setState((prev) => ({ ...prev, zoom: clampZoom(nextZoom) }));
  }, [displayHeight, displayWidth]);

  const rotate = useCallback(() => {
    setState((prev) => ({ ...prev, isRotated: !prev.isRotated }));
  }, []);

  const reset = useCallback(() => {
    setState({ ...DEFAULT_STATE });
  }, []);

  return {
    presets: DEVICE_PRESETS,
    presetId: state.presetId,
    zoom: state.zoom,
    zoomLevels: DEVICE_ZOOM_LEVELS,
    zoomMin: ZOOM_MIN,
    zoomMax: ZOOM_MAX,
    isRotated: state.isRotated,
    isResponsive,
    displayWidth,
    displayHeight,
    scaledWidth: displayWidth * state.zoom,
    scaledHeight: displayHeight * state.zoom,
    setPreset,
    setDimension,
    setZoom,
    zoomIn,
    zoomOut,
    resetZoom,
    fitToPane,
    rotate,
    reset,
  };
}
