/**
 * useDeviceEmulation — viewport state machine for the preview iframe.
 *
 * Re-derived from requirement module 04-multi-viewport-emulator (req
 * VP-001..004). Distinct from app-monitor's same-named hook: rcl emulates
 * a *component* preview inside a fixed editor pane, not a full app
 * window, so we skip responsive-mode resize handles and container-
 * bound zoom clamping. Just: preset → display dimensions, zoom in
 * [0.1, 2.0] applied as CSS transform, rotate (swap w/h), reset,
 * localStorage persistence under react-component-library.emulator.v1.
 */
import { useCallback, useEffect, useMemo, useState } from "react";

export const DEVICE_EMULATION_STORAGE_KEY = "react-component-library.emulator.v1";

export const DEVICE_PRESETS = [
  { id: "iphone-14", label: "iPhone 14", width: 390, height: 844 },
  { id: "iphone-se", label: "iPhone SE", width: 375, height: 667 },
  { id: "ipad", label: "iPad", width: 768, height: 1024 },
  { id: "ipad-pro", label: "iPad Pro", width: 1024, height: 1366 },
  { id: "desktop-1280", label: "Desktop 1280", width: 1280, height: 800 },
  { id: "desktop-1440", label: "Desktop 1440", width: 1440, height: 900 },
  { id: "desktop-1920", label: "Desktop 1920", width: 1920, height: 1080 },
] as const;

export type DevicePresetId = (typeof DEVICE_PRESETS)[number]["id"];

export const ZOOM_MIN = 0.1;
export const ZOOM_MAX = 2.0;
const ZOOM_STEP = 0.1;

const DEFAULT_PRESET_ID: DevicePresetId = "desktop-1280";

interface EmulatorState {
  presetId: DevicePresetId;
  zoom: number;
  isRotated: boolean;
}

const DEFAULT_STATE: Readonly<EmulatorState> = {
  presetId: DEFAULT_PRESET_ID,
  zoom: 1,
  isRotated: false,
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
  zoomMin: number;
  zoomMax: number;
  isRotated: boolean;
  /** Display dimensions = preset dims, swapped if rotated. */
  displayWidth: number;
  displayHeight: number;
  /** On-screen dimensions after CSS scale. */
  scaledWidth: number;
  scaledHeight: number;
  setPreset: (id: DevicePresetId) => void;
  setZoom: (zoom: number) => void;
  zoomIn: () => void;
  zoomOut: () => void;
  resetZoom: () => void;
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

  const displayWidth = state.isRotated ? preset.height : preset.width;
  const displayHeight = state.isRotated ? preset.width : preset.height;

  const setPreset = useCallback((id: DevicePresetId) => {
    setState((prev) => sanitize({ ...prev, presetId: id }));
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
    zoomMin: ZOOM_MIN,
    zoomMax: ZOOM_MAX,
    isRotated: state.isRotated,
    displayWidth,
    displayHeight,
    scaledWidth: displayWidth * state.zoom,
    scaledHeight: displayHeight * state.zoom,
    setPreset,
    setZoom,
    zoomIn,
    zoomOut,
    resetZoom,
    rotate,
    reset,
  };
}
