import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import type { PointerEvent as ReactPointerEvent } from 'react';
import { logger } from '@/services/logger';

export const DEVICE_EMULATION_STORAGE_KEY = 'app-monitor:device-emulation-settings';
const DEVICE_MIN_WIDTH = 240;
const DEVICE_MIN_HEIGHT = 320;
const DEVICE_MAX_DIMENSION = 2400;

export const DEVICE_PRESETS = [
  { id: 'responsive', label: 'Responsive', width: 1280, height: 720 },
  { id: 'iphone-se', label: 'iPhone SE', width: 375, height: 667 },
  { id: 'iphone-12-pro', label: 'iPhone 12 Pro', width: 390, height: 844 },
  { id: 'ipad-mini', label: 'iPad Mini', width: 430, height: 932 },
  { id: 'ipad-pro', label: 'iPad Pro', width: 1024, height: 1366 },
  { id: 'galaxy-fold', label: 'Galaxy Fold', width: 344, height: 882 },
  { id: 'nest-hub', label: 'Nest Hub', width: 1024, height: 600 },
] as const satisfies ReadonlyArray<{ id: string; label: string; width: number; height: number }>;

export const DEVICE_ZOOM_LEVELS = [0.1, 0.2, 0.25, 0.33, 0.5, 0.67, 0.75, 1, 1.25, 1.5, 2] as const;

export type DevicePresetId = typeof DEVICE_PRESETS[number]['id'];
export type DeviceColorScheme = 'system' | 'light' | 'dark';
export type DeviceVisionMode = 'none' | 'blur' | 'grayscale' | 'protanopia' | 'deuteranopia' | 'tritanopia';
export type DeviceZoomLevel = typeof DEVICE_ZOOM_LEVELS[number];

interface DeviceEmulationState {
  presetId: DevicePresetId;
  customWidth: number;
  customHeight: number;
  zoom: DeviceZoomLevel;
  colorScheme: DeviceColorScheme;
  vision: DeviceVisionMode;
  isRotated: boolean;
}

const DEFAULT_DEVICE_EMULATION_STATE: Readonly<DeviceEmulationState> = {
  presetId: 'responsive',
  customWidth: 1280,
  customHeight: 720,
  zoom: 1,
  colorScheme: 'system',
  vision: 'none',
  isRotated: false,
};

const DEVICE_MIN_ZOOM = DEVICE_ZOOM_LEVELS[0];
const DEVICE_MAX_ZOOM = DEVICE_ZOOM_LEVELS[DEVICE_ZOOM_LEVELS.length - 1];

const mapDisplayToBaseDimensions = (dimensions: { width: number; height: number }, rotated: boolean) => {
  if (rotated) {
    return {
      width: dimensions.height,
      height: dimensions.width,
    };
  }
  return dimensions;
};

const pickZoomLevelForLimit = (limit: number) => {
  if (!Number.isFinite(limit) || limit <= 0) {
    return DEVICE_MIN_ZOOM;
  }

  const normalizedLimit = limit + 1e-6;
  for (let index = DEVICE_ZOOM_LEVELS.length - 1; index >= 0; index -= 1) {
    const candidate = DEVICE_ZOOM_LEVELS[index];
    if (candidate <= normalizedLimit) {
      return candidate;
    }
  }
  return DEVICE_MIN_ZOOM;
};

const clampDisplayToLimit = (value: number, limit: number, minimum: number) => {
  const roundedValue = Math.round(value);
  if (!Number.isFinite(limit) || limit <= 0) {
    return Math.max(minimum, roundedValue);
  }

  const roundedLimit = Math.max(1, Math.floor(limit));
  if (roundedLimit >= minimum) {
    return Math.min(Math.max(roundedValue, minimum), roundedLimit);
  }

  return Math.max(1, Math.min(roundedValue, roundedLimit));
};

const clampDimension = (value: number, min: number, max: number) => {
  if (!Number.isFinite(value)) {
    return min;
  }
  return Math.min(Math.max(Math.round(value), min), max);
};

const sanitizeDeviceEmulationState = (
  state: Partial<DeviceEmulationState>,
  options?: { minWidth?: number; minHeight?: number },
): DeviceEmulationState => {
  const base = { ...DEFAULT_DEVICE_EMULATION_STATE };
  const preset = DEVICE_PRESETS.find(candidate => candidate.id === state.presetId) ?? DEVICE_PRESETS[0];
  const zoomCandidate = typeof state.zoom === 'number' ? state.zoom : base.zoom;
  const widthCandidate = typeof state.customWidth === 'number' ? state.customWidth : base.customWidth;
  const heightCandidate = typeof state.customHeight === 'number' ? state.customHeight : base.customHeight;
  const colorCandidate = typeof state.colorScheme === 'string' ? state.colorScheme : base.colorScheme;
  const visionCandidate = typeof state.vision === 'string' ? state.vision : base.vision;

  const zoom = DEVICE_ZOOM_LEVELS.includes(zoomCandidate as DeviceZoomLevel)
    ? (zoomCandidate as DeviceZoomLevel)
    : base.zoom;
  const enforcedMinWidth = options?.minWidth !== undefined
    ? Math.max(1, Math.round(options.minWidth))
    : DEVICE_MIN_WIDTH;
  const enforcedMinHeight = options?.minHeight !== undefined
    ? Math.max(1, Math.round(options.minHeight))
    : DEVICE_MIN_HEIGHT;
  const customWidth = clampDimension(widthCandidate, enforcedMinWidth, DEVICE_MAX_DIMENSION);
  const customHeight = clampDimension(heightCandidate, enforcedMinHeight, DEVICE_MAX_DIMENSION);
  const colorScheme: DeviceColorScheme = ['system', 'light', 'dark'].includes(colorCandidate)
    ? colorCandidate as DeviceColorScheme
    : base.colorScheme;
  const vision: DeviceVisionMode = ['none', 'blur', 'grayscale', 'protanopia', 'deuteranopia', 'tritanopia'].includes(visionCandidate)
    ? visionCandidate as DeviceVisionMode
    : base.vision;

  return {
    presetId: preset.id as DevicePresetId,
    customWidth,
    customHeight,
    zoom,
    colorScheme,
    vision,
    isRotated: Boolean(state.isRotated),
  };
};

const readDeviceEmulationSettings = (): { active: boolean; state: DeviceEmulationState } => {
  if (typeof window === 'undefined') {
    return { active: false, state: { ...DEFAULT_DEVICE_EMULATION_STATE } };
  }

  try {
    const raw = window.localStorage.getItem(DEVICE_EMULATION_STORAGE_KEY);
    if (!raw) {
      return { active: false, state: { ...DEFAULT_DEVICE_EMULATION_STATE } };
    }

    const parsed = JSON.parse(raw) as unknown;
    if (!parsed || typeof parsed !== 'object') {
      return { active: false, state: { ...DEFAULT_DEVICE_EMULATION_STATE } };
    }

    const record = parsed as { active?: unknown; state?: unknown };
    const rawState = typeof record.state === 'object' && record.state !== null
      ? sanitizeDeviceEmulationState(record.state as Partial<DeviceEmulationState>)
      : { ...DEFAULT_DEVICE_EMULATION_STATE };

    let active = false;
    if (typeof record.active === 'boolean') {
      active = record.active;
    }

    return { active, state: rawState };
  } catch (error) {
    logger.warn('Failed to read device emulation settings', error);
    return { active: false, state: { ...DEFAULT_DEVICE_EMULATION_STATE } };
  }
};

const writeDeviceEmulationSettings = (active: boolean, state: DeviceEmulationState) => {
  if (typeof window === 'undefined') {
    return;
  }

  try {
    window.localStorage.setItem(
      DEVICE_EMULATION_STORAGE_KEY,
      JSON.stringify({ active, state: sanitizeDeviceEmulationState(state) }),
    );
  } catch (error) {
    logger.warn('Failed to persist device emulation settings', error);
  }
};

interface UseDeviceEmulationOptions {
  container: HTMLDivElement | null;
}

interface DeviceEmulationToolbarBindings {
  presets: typeof DEVICE_PRESETS;
  selectedPresetId: DevicePresetId;
  displayWidth: number;
  displayHeight: number;
  zoomLevels: typeof DEVICE_ZOOM_LEVELS;
  zoom: DeviceZoomLevel;
  colorScheme: DeviceColorScheme;
  vision: DeviceVisionMode;
  isResponsive: boolean;
  maxResponsiveWidth: number | null;
  maxResponsiveHeight: number | null;
  onPresetChange: (presetId: DevicePresetId) => void;
  onDimensionChange: (dimension: 'width' | 'height', value: number) => void;
  onZoomChange: (zoom: DeviceZoomLevel) => void;
  onColorSchemeChange: (scheme: DeviceColorScheme) => void;
  onVisionChange: (vision: DeviceVisionMode) => void;
  onRotate: () => void;
  onReset: () => void;
}

interface DeviceEmulationViewportBindings {
  displayWidth: number;
  displayHeight: number;
  zoomedWidth: number;
  zoomedHeight: number;
  zoom: DeviceZoomLevel;
  colorScheme: DeviceColorScheme;
  vision: DeviceVisionMode;
  isResponsive: boolean;
  onResizePointerDown: (event: ReactPointerEvent<HTMLDivElement>) => void;
}

interface DeviceEmulationHookValue {
  isActive: boolean;
  toggleActive: () => void;
  toolbar: DeviceEmulationToolbarBindings;
  viewport: DeviceEmulationViewportBindings;
}

export const useDeviceEmulation = ({ container }: UseDeviceEmulationOptions): DeviceEmulationHookValue => {
  const initialSettings = useMemo(() => readDeviceEmulationSettings(), []);
  const [state, setState] = useState<DeviceEmulationState>(initialSettings.state);
  const [isActive, setIsActive] = useState<boolean>(initialSettings.active);
  const [viewportBounds, setViewportBounds] = useState<Readonly<{ width: number; height: number }>>({
    width: Number.POSITIVE_INFINITY,
    height: Number.POSITIVE_INFINITY,
  });
  const deviceResizeStateRef = useRef<{
    pointerId: number;
    startX: number;
    startY: number;
    startWidth: number;
    startHeight: number;
    target: HTMLElement | null;
    mode: 'both' | 'width' | 'height';
  } | null>(null);

  const selectedPreset = useMemo(() => {
    return DEVICE_PRESETS.find(candidate => candidate.id === state.presetId) ?? DEVICE_PRESETS[0];
  }, [state.presetId]);

  const baseDimensions = useMemo(() => {
    const preset = selectedPreset;
    const baseWidth = preset.id === 'responsive' ? state.customWidth : preset.width;
    const baseHeight = preset.id === 'responsive' ? state.customHeight : preset.height;
    return { preset, baseWidth, baseHeight };
  }, [selectedPreset, state.customHeight, state.customWidth]);

  const displayDimensions = useMemo(() => {
    const { baseWidth, baseHeight } = baseDimensions;
    if (state.isRotated) {
      return { width: baseHeight, height: baseWidth };
    }
    return { width: baseWidth, height: baseHeight };
  }, [baseDimensions, state.isRotated]);

  const zoomedDimensions = useMemo(() => ({
    width: displayDimensions.width * state.zoom,
    height: displayDimensions.height * state.zoom,
  }), [displayDimensions.height, displayDimensions.width, state.zoom]);

  const currentZoomDisplayLimits = useMemo(() => {
    if (!isActive) {
      return null;
    }

    if (!Number.isFinite(viewportBounds.width) || !Number.isFinite(viewportBounds.height)) {
      return null;
    }

    const widthLimit = viewportBounds.width / state.zoom;
    const heightLimit = viewportBounds.height / state.zoom;

    if (!Number.isFinite(widthLimit) || !Number.isFinite(heightLimit)) {
      return null;
    }

    return { width: widthLimit, height: heightLimit } as const;
  }, [isActive, state.zoom, viewportBounds.height, viewportBounds.width]);

  const maxZoomForDimensions = useMemo(() => {
    if (!isActive) {
      return DEVICE_MAX_ZOOM;
    }

    if (!Number.isFinite(viewportBounds.width) || !Number.isFinite(viewportBounds.height)) {
      return DEVICE_MAX_ZOOM;
    }

    const width = displayDimensions.width;
    const height = displayDimensions.height;
    if (width <= 0 || height <= 0) {
      return DEVICE_MAX_ZOOM;
    }

    const widthLimit = viewportBounds.width / width;
    const heightLimit = viewportBounds.height / height;
    const limit = Math.min(widthLimit, heightLimit);

    return Number.isFinite(limit) && limit > 0 ? limit : DEVICE_MAX_ZOOM;
  }, [displayDimensions.height, displayDimensions.width, isActive, viewportBounds.height, viewportBounds.width]);

  useEffect(() => {
    if (!isActive) {
      deviceResizeStateRef.current = null;
    }
  }, [isActive]);

  useEffect(() => {
    if (selectedPreset.id !== 'responsive') {
      deviceResizeStateRef.current = null;
    }
  }, [selectedPreset.id]);

  useEffect(() => {
    if (!isActive || !Number.isFinite(viewportBounds.width) || !Number.isFinite(viewportBounds.height)) {
      return;
    }

    setState(prev => {
      let next = sanitizeDeviceEmulationState(prev);

      if (next.presetId === 'responsive') {
        const maxDisplayWidth = viewportBounds.width / next.zoom;
        const maxDisplayHeight = viewportBounds.height / next.zoom;

        const currentDisplay = {
          width: next.isRotated ? next.customHeight : next.customWidth,
          height: next.isRotated ? next.customWidth : next.customHeight,
        };

        const limitedDisplay = {
          width: clampDisplayToLimit(currentDisplay.width, maxDisplayWidth, DEVICE_MIN_WIDTH),
          height: clampDisplayToLimit(currentDisplay.height, maxDisplayHeight, DEVICE_MIN_HEIGHT),
        };

        if (limitedDisplay.width !== currentDisplay.width || limitedDisplay.height !== currentDisplay.height) {
          const base = mapDisplayToBaseDimensions(limitedDisplay, next.isRotated);
          next = sanitizeDeviceEmulationState({
            ...next,
            customWidth: base.width,
            customHeight: base.height,
          }, {
            minWidth: Math.max(1, Math.min(DEVICE_MIN_WIDTH, base.width)),
            minHeight: Math.max(1, Math.min(DEVICE_MIN_HEIGHT, base.height)),
          });
        }
      } else {
        const preset = DEVICE_PRESETS.find(candidate => candidate.id === next.presetId) ?? DEVICE_PRESETS[0];
        const presetDisplay = {
          width: next.isRotated ? preset.height : preset.width,
          height: next.isRotated ? preset.width : preset.height,
        };

        const widthLimit = presetDisplay.width > 0 ? viewportBounds.width / presetDisplay.width : Number.POSITIVE_INFINITY;
        const heightLimit = presetDisplay.height > 0 ? viewportBounds.height / presetDisplay.height : Number.POSITIVE_INFINITY;
        const allowedZoom = pickZoomLevelForLimit(Math.min(widthLimit, heightLimit));
        if (next.zoom > allowedZoom) {
          next = sanitizeDeviceEmulationState({ ...next, zoom: allowedZoom });
        }
      }

      return next;
    });
  }, [isActive, viewportBounds.height, viewportBounds.width]);

  useEffect(() => {
    if (!isActive || selectedPreset.id === 'responsive') {
      return;
    }

    const allowedZoom = pickZoomLevelForLimit(maxZoomForDimensions);
    if (allowedZoom < state.zoom - 1e-6) {
      setState(prev => sanitizeDeviceEmulationState({ ...prev, zoom: allowedZoom }));
    }
  }, [isActive, maxZoomForDimensions, selectedPreset.id, state.zoom]);

  useEffect(() => {
    if (!isActive || selectedPreset.id !== 'responsive' || !currentZoomDisplayLimits) {
      return;
    }

    const currentDisplay = {
      width: state.isRotated ? state.customHeight : state.customWidth,
      height: state.isRotated ? state.customWidth : state.customHeight,
    };

    const limitedWidth = clampDisplayToLimit(
      currentDisplay.width,
      currentZoomDisplayLimits.width,
      DEVICE_MIN_WIDTH,
    );
    const limitedHeight = clampDisplayToLimit(
      currentDisplay.height,
      currentZoomDisplayLimits.height,
      DEVICE_MIN_HEIGHT,
    );

    if (limitedWidth === currentDisplay.width && limitedHeight === currentDisplay.height) {
      return;
    }

    const base = mapDisplayToBaseDimensions({ width: limitedWidth, height: limitedHeight }, state.isRotated);
    setState(prev => sanitizeDeviceEmulationState({
      ...prev,
      customWidth: base.width,
      customHeight: base.height,
    }, {
      minWidth: Math.max(1, Math.min(DEVICE_MIN_WIDTH, base.width)),
      minHeight: Math.max(1, Math.min(DEVICE_MIN_HEIGHT, base.height)),
    }));
  }, [currentZoomDisplayLimits, isActive, selectedPreset.id, state.customHeight, state.customWidth, state.isRotated, state.zoom]);

  useEffect(() => {
    writeDeviceEmulationSettings(isActive, state);
  }, [isActive, state]);

  useLayoutEffect(() => {
    if (!isActive || typeof window === 'undefined') {
      setViewportBounds({
        width: Number.POSITIVE_INFINITY,
        height: Number.POSITIVE_INFINITY,
      });
      return;
    }

    const containerNode = container;
    if (!containerNode) {
      setViewportBounds({
        width: Number.POSITIVE_INFINITY,
        height: Number.POSITIVE_INFINITY,
      });
      return;
    }

    const computeBounds = () => {
      const style = window.getComputedStyle(containerNode);
      const paddingX = Number.parseFloat(style.paddingLeft || '0') + Number.parseFloat(style.paddingRight || '0');
      const paddingY = Number.parseFloat(style.paddingTop || '0') + Number.parseFloat(style.paddingBottom || '0');
      setViewportBounds({
        width: Math.max(0, containerNode.clientWidth - paddingX),
        height: Math.max(0, containerNode.clientHeight - paddingY),
      });
    };

    computeBounds();

    const resizeHandler = () => {
      computeBounds();
    };

    let observer: ResizeObserver | null = null;

    if (typeof ResizeObserver === 'function') {
      observer = new ResizeObserver(() => computeBounds());
      observer.observe(containerNode);
    }

    window.addEventListener('resize', resizeHandler);

    return () => {
      window.removeEventListener('resize', resizeHandler);
      if (observer) {
        observer.disconnect();
      }
    };
  }, [container, isActive]);

  useEffect(() => {
    const handlePointerMove = (event: PointerEvent) => {
      const resizeState = deviceResizeStateRef.current;
      if (!resizeState || resizeState.pointerId !== event.pointerId) {
        return;
      }

      const adjustWidth = resizeState.mode === 'both' || resizeState.mode === 'width';
      const adjustHeight = resizeState.mode === 'both' || resizeState.mode === 'height';
      const deltaX = adjustWidth ? (event.clientX - resizeState.startX) / state.zoom : 0;
      const deltaY = adjustHeight ? (event.clientY - resizeState.startY) / state.zoom : 0;
      const widthLimit = Number.isFinite(viewportBounds.width)
        ? viewportBounds.width / state.zoom
        : Number.POSITIVE_INFINITY;
      const heightLimit = Number.isFinite(viewportBounds.height)
        ? viewportBounds.height / state.zoom
        : Number.POSITIVE_INFINITY;

      const nextDisplayWidth = adjustWidth
        ? clampDisplayToLimit(resizeState.startWidth + deltaX, widthLimit, DEVICE_MIN_WIDTH)
        : resizeState.startWidth;
      const nextDisplayHeight = adjustHeight
        ? clampDisplayToLimit(resizeState.startHeight + deltaY, heightLimit, DEVICE_MIN_HEIGHT)
        : resizeState.startHeight;

      setState(prev => {
        if (prev.presetId !== 'responsive') {
          return prev;
        }

        const baseDims = mapDisplayToBaseDimensions(
          { width: nextDisplayWidth, height: nextDisplayHeight },
          prev.isRotated,
        );

        if (baseDims.width === prev.customWidth && baseDims.height === prev.customHeight) {
          return prev;
        }

        return sanitizeDeviceEmulationState({
          ...prev,
          customWidth: baseDims.width,
          customHeight: baseDims.height,
        }, {
          minWidth: Math.max(1, Math.min(DEVICE_MIN_WIDTH, baseDims.width)),
          minHeight: Math.max(1, Math.min(DEVICE_MIN_HEIGHT, baseDims.height)),
        });
      });
    };

    const handlePointerRelease = (event: PointerEvent) => {
      const resizeState = deviceResizeStateRef.current;
      if (!resizeState || resizeState.pointerId !== event.pointerId) {
        return;
      }

      const target = resizeState.target;
      if (target && typeof target.releasePointerCapture === 'function') {
        try {
          target.releasePointerCapture(event.pointerId);
        } catch (error) {
          logger.warn('Failed to release pointer capture for device resize handle', error);
        }
      }

      deviceResizeStateRef.current = null;
    };

    window.addEventListener('pointermove', handlePointerMove);
    window.addEventListener('pointerup', handlePointerRelease);
    window.addEventListener('pointercancel', handlePointerRelease);

    return () => {
      window.removeEventListener('pointermove', handlePointerMove);
      window.removeEventListener('pointerup', handlePointerRelease);
      window.removeEventListener('pointercancel', handlePointerRelease);
    };
  }, [state.zoom, viewportBounds.height, viewportBounds.width]);

  const toggleActive = useCallback(() => {
    setIsActive(current => !current);
  }, []);

  const handlePresetChange = useCallback((nextPresetId: DevicePresetId) => {
    setState(prev => sanitizeDeviceEmulationState({ ...prev, presetId: nextPresetId }));
  }, []);

  const handleDimensionChange = useCallback((dimension: 'width' | 'height', value: number) => {
    if (Number.isNaN(value)) {
      return;
    }

    setState(prev => {
      if (prev.presetId !== 'responsive') {
        return prev;
      }

      const limits = currentZoomDisplayLimits;
      const currentDisplay = {
        width: prev.isRotated ? prev.customHeight : prev.customWidth,
        height: prev.isRotated ? prev.customWidth : prev.customHeight,
      };

      const nextDisplay = { ...currentDisplay };
      if (dimension === 'width') {
        nextDisplay.width = clampDisplayToLimit(
          value,
          limits ? limits.width : Number.POSITIVE_INFINITY,
          DEVICE_MIN_WIDTH,
        );
      } else {
        nextDisplay.height = clampDisplayToLimit(
          value,
          limits ? limits.height : Number.POSITIVE_INFINITY,
          DEVICE_MIN_HEIGHT,
        );
      }

      if (nextDisplay.width === currentDisplay.width && nextDisplay.height === currentDisplay.height) {
        return prev;
      }

      const base = mapDisplayToBaseDimensions(nextDisplay, prev.isRotated);
      return sanitizeDeviceEmulationState({
        ...prev,
        customWidth: base.width,
        customHeight: base.height,
      }, {
        minWidth: Math.max(1, Math.min(DEVICE_MIN_WIDTH, base.width)),
        minHeight: Math.max(1, Math.min(DEVICE_MIN_HEIGHT, base.height)),
      });
    });
  }, [currentZoomDisplayLimits]);

  const handleZoomChange = useCallback((zoom: DeviceZoomLevel) => {
    if (!DEVICE_ZOOM_LEVELS.includes(zoom)) {
      return;
    }

    setState(prev => {
      let next = sanitizeDeviceEmulationState({ ...prev, zoom });

      if (!isActive || !Number.isFinite(viewportBounds.width) || !Number.isFinite(viewportBounds.height)) {
        return next;
      }

      if (next.presetId === 'responsive') {
        const maxDisplayWidth = viewportBounds.width / next.zoom;
        const maxDisplayHeight = viewportBounds.height / next.zoom;

        const currentDisplay = {
          width: next.isRotated ? next.customHeight : next.customWidth,
          height: next.isRotated ? next.customWidth : next.customHeight,
        };

        const limitedDisplay = {
          width: clampDisplayToLimit(currentDisplay.width, maxDisplayWidth, DEVICE_MIN_WIDTH),
          height: clampDisplayToLimit(currentDisplay.height, maxDisplayHeight, DEVICE_MIN_HEIGHT),
        };

        if (limitedDisplay.width !== currentDisplay.width || limitedDisplay.height !== currentDisplay.height) {
          const base = mapDisplayToBaseDimensions(limitedDisplay, next.isRotated);
          next = sanitizeDeviceEmulationState({
            ...next,
            customWidth: base.width,
            customHeight: base.height,
          }, {
            minWidth: Math.max(1, Math.min(DEVICE_MIN_WIDTH, base.width)),
            minHeight: Math.max(1, Math.min(DEVICE_MIN_HEIGHT, base.height)),
          });
        }
      } else {
        const preset = DEVICE_PRESETS.find(candidate => candidate.id === next.presetId) ?? DEVICE_PRESETS[0];
        const presetDisplay = {
          width: next.isRotated ? preset.height : preset.width,
          height: next.isRotated ? preset.width : preset.height,
        };

        const widthLimit = presetDisplay.width > 0 ? viewportBounds.width / presetDisplay.width : Number.POSITIVE_INFINITY;
        const heightLimit = presetDisplay.height > 0 ? viewportBounds.height / presetDisplay.height : Number.POSITIVE_INFINITY;
        const allowedZoom = pickZoomLevelForLimit(Math.min(widthLimit, heightLimit));
        if (next.zoom > allowedZoom) {
          next = sanitizeDeviceEmulationState({ ...next, zoom: allowedZoom });
        }
      }

      return next;
    });
  }, [isActive, viewportBounds.height, viewportBounds.width]);

  const handleColorSchemeChange = useCallback((scheme: DeviceColorScheme) => {
    setState(prev => sanitizeDeviceEmulationState({ ...prev, colorScheme: scheme }));
  }, []);

  const handleVisionChange = useCallback((vision: DeviceVisionMode) => {
    setState(prev => sanitizeDeviceEmulationState({ ...prev, vision }));
  }, []);

  const handleRotate = useCallback(() => {
    setState(prev => sanitizeDeviceEmulationState({ ...prev, isRotated: !prev.isRotated }));
  }, []);

  const handleReset = useCallback(() => {
    setState({ ...DEFAULT_DEVICE_EMULATION_STATE });
    setIsActive(true);
  }, []);

  const handleResizePointerDown = useCallback((event: ReactPointerEvent<HTMLDivElement>) => {
    if (selectedPreset.id !== 'responsive') {
      return;
    }

    event.preventDefault();
    event.stopPropagation();

    const pointerId = event.pointerId;
    const resizeModeAttr = event.currentTarget.dataset.resizeMode;
    const resizeMode = resizeModeAttr === 'width'
      ? 'width'
      : resizeModeAttr === 'height'
        ? 'height'
        : 'both';
    deviceResizeStateRef.current = {
      pointerId,
      startX: event.clientX,
      startY: event.clientY,
      startWidth: displayDimensions.width,
      startHeight: displayDimensions.height,
      target: event.currentTarget,
      mode: resizeMode,
    };

    if (typeof event.currentTarget.setPointerCapture === 'function') {
      event.currentTarget.setPointerCapture(pointerId);
    }
  }, [displayDimensions.height, displayDimensions.width, selectedPreset.id]);

  const toolbarBindings: DeviceEmulationToolbarBindings = {
    presets: DEVICE_PRESETS,
    selectedPresetId: selectedPreset.id as DevicePresetId,
    displayWidth: displayDimensions.width,
    displayHeight: displayDimensions.height,
    zoomLevels: DEVICE_ZOOM_LEVELS,
    zoom: state.zoom,
    colorScheme: state.colorScheme,
    vision: state.vision,
    isResponsive: selectedPreset.id === 'responsive',
    maxResponsiveWidth: currentZoomDisplayLimits?.width ?? null,
    maxResponsiveHeight: currentZoomDisplayLimits?.height ?? null,
    onPresetChange: handlePresetChange,
    onDimensionChange: handleDimensionChange,
    onZoomChange: handleZoomChange,
    onColorSchemeChange: handleColorSchemeChange,
    onVisionChange: handleVisionChange,
    onRotate: handleRotate,
    onReset: handleReset,
  };

  const viewportBindings: DeviceEmulationViewportBindings = {
    displayWidth: displayDimensions.width,
    displayHeight: displayDimensions.height,
    zoomedWidth: zoomedDimensions.width,
    zoomedHeight: zoomedDimensions.height,
    zoom: state.zoom,
    colorScheme: state.colorScheme,
    vision: state.vision,
    isResponsive: selectedPreset.id === 'responsive',
    onResizePointerDown: handleResizePointerDown,
  };

  return {
    isActive,
    toggleActive,
    toolbar: toolbarBindings,
    viewport: viewportBindings,
  };
};

export type { DeviceEmulationToolbarBindings, DeviceEmulationViewportBindings };
