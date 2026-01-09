/**
 * Hook for managing export dialog form state.
 *
 * This hook isolates the dialog's form state management from:
 * - Movie spec transformations (pure functions)
 * - API calls (export client)
 * - Preview rendering (composer iframe)
 *
 * By separating form state, the component becomes easier to test
 * and the state management logic is clearly visible.
 */

import { useCallback, useEffect, useMemo, useState } from "react";
import {
  DEFAULT_DIMENSIONS,
  extractCanvasDimensions,
  parseDimensionInput,
  resolveDimensionPreset,
  type DimensionPresetId,
  type Dimensions,
} from "../transformations";
import {
  EXPORT_EXTENSIONS,
  type ExportFormat,
  type ExportRenderSource,
} from "../config";
import type { ReplayMovieSpec } from "@/types/export";

// Re-export types for backwards compatibility
export type { ExportFormat, ExportRenderSource } from "../config";

export interface UseExportDialogStateOptions {
  /**
   * The execution ID being exported.
   */
  executionId: string;

  /**
   * The movie spec to extract initial dimensions from.
   */
  movieSpec: ReplayMovieSpec | null;

  /**
   * Callback when render source changes (for parent coordination).
   */
  onRenderSourceChange?: (source: ExportRenderSource) => void;
}

export interface ExportDialogFormState {
  // Format selection
  format: ExportFormat;
  setFormat: (format: ExportFormat) => void;
  isBinaryExport: boolean;

  // Render source
  renderSource: ExportRenderSource;
  setRenderSource: (source: ExportRenderSource) => void;

  // Dimensions
  dimensionPreset: DimensionPresetId;
  setDimensionPreset: (preset: DimensionPresetId) => void;
  customWidthInput: string;
  customHeightInput: string;
  setCustomWidthInput: (value: string) => void;
  setCustomHeightInput: (value: string) => void;
  selectedDimensions: Dimensions;
  specDimensions: Dimensions;

  // File naming
  fileStem: string;
  setFileStem: (stem: string) => void;
  defaultFileStem: string;

  // Native file picker
  useNativeFilePicker: boolean;
  setUseNativeFilePicker: (use: boolean) => void;
  supportsFileSystemAccess: boolean;

  // Dialog state
  isOpen: boolean;
  open: () => void;
  close: () => void;

  // Reset form to defaults
  reset: () => void;
}

// =============================================================================
// Constants
// =============================================================================

// Use EXPORT_EXTENSIONS from config module for file extensions

// =============================================================================
// Hook Implementation
// =============================================================================

export function useExportDialogState({
  executionId,
  movieSpec,
  onRenderSourceChange,
}: UseExportDialogStateOptions): ExportDialogFormState {
  // Dialog open/close
  const [isOpen, setIsOpen] = useState(false);

  // Format state
  const [format, setFormatInternal] = useState<ExportFormat>("mp4");
  const [renderSource, setRenderSourceInternal] = useState<ExportRenderSource>("auto");

  // Dimension state
  const [dimensionPreset, setDimensionPreset] = useState<DimensionPresetId>("spec");
  const [customWidthInput, setCustomWidthInput] = useState(() =>
    String(DEFAULT_DIMENSIONS.width),
  );
  const [customHeightInput, setCustomHeightInput] = useState(() =>
    String(DEFAULT_DIMENSIONS.height),
  );

  // File naming state
  const defaultFileStem = useMemo(
    () => `browser-automation-replay-${executionId.slice(0, 8)}`,
    [executionId],
  );
  const [fileStem, setFileStem] = useState(defaultFileStem);

  // Native file picker state
  const [useNativeFilePicker, setUseNativeFilePicker] = useState(false);
  const supportsFileSystemAccess =
    typeof window !== "undefined" &&
    typeof (window as typeof window & { showSaveFilePicker?: unknown })
      .showSaveFilePicker === "function";

  // Derived state
  const isBinaryExport = format !== "json";

  const specDimensions = useMemo(
    () => extractCanvasDimensions(movieSpec?.presentation),
    [movieSpec],
  );

  const customDimensions = useMemo(
    () => ({
      width: parseDimensionInput(customWidthInput, DEFAULT_DIMENSIONS.width),
      height: parseDimensionInput(customHeightInput, DEFAULT_DIMENSIONS.height),
    }),
    [customWidthInput, customHeightInput],
  );

  const selectedDimensions = useMemo(
    () => resolveDimensionPreset(dimensionPreset, specDimensions, customDimensions),
    [dimensionPreset, specDimensions, customDimensions],
  );

  // Reset custom dimensions when spec changes
  useEffect(() => {
    setCustomWidthInput(String(specDimensions.width));
    setCustomHeightInput(String(specDimensions.height));
    setDimensionPreset("spec");
  }, [executionId, specDimensions.width, specDimensions.height]);

  // Reset native file picker when format changes to non-JSON
  useEffect(() => {
    if (format !== "json" && useNativeFilePicker) {
      setUseNativeFilePicker(false);
    }
  }, [format, useNativeFilePicker]);

  // Wrapped setters with side effects
  const setFormat = useCallback((newFormat: ExportFormat) => {
    setFormatInternal(newFormat);
  }, []);

  const setRenderSource = useCallback(
    (source: ExportRenderSource) => {
      setRenderSourceInternal(source);
      onRenderSourceChange?.(source);
    },
    [onRenderSourceChange],
  );

  const open = useCallback(() => {
    setIsOpen(true);
  }, []);

  const close = useCallback(() => {
    setIsOpen(false);
  }, []);

  const reset = useCallback(() => {
    setFormatInternal("mp4");
    setRenderSourceInternal("auto");
    setDimensionPreset("spec");
    setFileStem(defaultFileStem);
    setUseNativeFilePicker(false);
  }, [defaultFileStem]);

  // Reset form when execution changes
  useEffect(() => {
    reset();
  }, [executionId, reset]);

  return {
    // Format
    format,
    setFormat,
    isBinaryExport,

    // Render source
    renderSource,
    setRenderSource,

    // Dimensions
    dimensionPreset,
    setDimensionPreset,
    customWidthInput,
    customHeightInput,
    setCustomWidthInput,
    setCustomHeightInput,
    selectedDimensions,
    specDimensions,

    // File naming
    fileStem,
    setFileStem,
    defaultFileStem,

    // Native file picker
    useNativeFilePicker,
    setUseNativeFilePicker,
    supportsFileSystemAccess,

    // Dialog state
    isOpen,
    open,
    close,

    // Reset
    reset,
  };
}

/**
 * Builds the final file name from stem and format.
 */
export function buildFileName(stem: string, format: ExportFormat): string {
  const extension = EXPORT_EXTENSIONS[format];
  return `${stem}.${extension}`;
}

// Note: sanitizeFileStem is now exported from ../config module
