/**
 * ExportDialogContext - Provides export dialog state and actions to child components.
 *
 * This context eliminates prop drilling by making export dialog state available
 * to any component in the tree. The state is organized into logical groups:
 *
 * - formatState: Export format selection
 * - dimensionState: Output dimensions
 * - fileState: File naming and destination
 * - renderSourceState: Video render source selection
 * - previewState: Preview rendering state
 * - progressState: Export progress and loading
 * - metricsState: Frame count, duration, etc.
 * - actions: Dialog open/close/confirm
 *
 * @module export/context
 */

import {
  createContext,
  useContext,
  type ReactNode,
  type MutableRefObject,
} from "react";
import type { ReplayMovieSpec } from "@/types/export";
import type {
  ExportDimensionPreset,
  ExportDimensions,
  ExportFormat,
  ExportFormatOption,
  ExportRenderSource,
  ExportRenderSourceOption,
} from "../config";

// =============================================================================
// Types
// =============================================================================

/**
 * Format selection state.
 */
export interface ExportFormatState {
  format: ExportFormat;
  setFormat: (format: ExportFormat) => void;
  isBinaryExport: boolean;
  formatOptions: ExportFormatOption[];
}

/**
 * Dimension selection state.
 */
export interface ExportDimensionState {
  preset: ExportDimensionPreset;
  setPreset: (preset: ExportDimensionPreset) => void;
  presetOptions: Array<{
    id: ExportDimensionPreset;
    label: string;
    width?: number;
    height?: number;
    description?: string;
  }>;
  selectedDimensions: ExportDimensions;
  customWidthInput: string;
  customHeightInput: string;
  setCustomWidthInput: (value: string) => void;
  setCustomHeightInput: (value: string) => void;
}

/**
 * File naming and destination state.
 */
export interface ExportFileState {
  fileStem: string;
  setFileStem: (stem: string) => void;
  defaultFileStem: string;
  finalFileName: string;
  supportsFileSystemAccess: boolean;
  useNativeFilePicker: boolean;
  setUseNativeFilePicker: (use: boolean) => void;
}

/**
 * Render source selection state.
 */
export interface ExportRenderSourceState {
  source: ExportRenderSource;
  setSource: (source: ExportRenderSource) => void;
  sourceOptions: ExportRenderSourceOption[];
  recordedVideoAvailable: boolean;
  recordedVideoCount: number;
  recordedVideoLoading: boolean;
}

/**
 * Preview rendering state.
 */
export interface ExportPreviewState {
  movieSpec: ReplayMovieSpec | null;
  composerPreviewUrl: string;
  firstFramePreviewUrl: string | null;
  firstFrameLabel: string | null;
  composerRef: MutableRefObject<HTMLIFrameElement | null>;
  composerWindowRef: MutableRefObject<Window | null>;
  composerOriginRef: MutableRefObject<string | null>;
  isComposerReady: boolean;
  setIsComposerReady: (ready: boolean) => void;
  composerError: string | null;
  setComposerError: (error: string | null) => void;
}

/**
 * Export progress state.
 */
export interface ExportProgressState {
  isExporting: boolean;
  isPreviewLoading: boolean;
  statusMessage: string;
}

/**
 * Export metrics state.
 */
export interface ExportMetricsState {
  replayFramesLength: number;
  estimatedFrameCount?: number;
  estimatedDurationSeconds: number | null;
  activeSpecId: string | null;
  previewMetrics: {
    capturedFrames: number;
    assetCount: number;
    totalDurationMs: number;
  };
}

/**
 * Dialog actions.
 */
export interface ExportDialogActions {
  onClose: () => void;
  onConfirm: () => void;
}

/**
 * Complete export dialog context value.
 */
export interface ExportDialogContextValue {
  /** Dialog accessibility IDs */
  titleId: string;
  descriptionId: string;

  /** Grouped state */
  formatState: ExportFormatState;
  dimensionState: ExportDimensionState;
  fileState: ExportFileState;
  renderSourceState: ExportRenderSourceState;
  previewState: ExportPreviewState;
  progressState: ExportProgressState;
  metricsState: ExportMetricsState;

  /** Actions */
  actions: ExportDialogActions;

  /** Utilities */
  formatSeconds: (value: number) => string;
}

// =============================================================================
// Context
// =============================================================================

const ExportDialogContext = createContext<ExportDialogContextValue | null>(null);

// =============================================================================
// Hook
// =============================================================================

/**
 * Hook to access export dialog context.
 * Must be used within an ExportDialogProvider.
 *
 * @throws Error if used outside of provider
 */
export function useExportDialogContext(): ExportDialogContextValue {
  const context = useContext(ExportDialogContext);
  if (!context) {
    throw new Error(
      "useExportDialogContext must be used within an ExportDialogProvider",
    );
  }
  return context;
}

/**
 * Hook to access format state from context.
 */
export function useExportFormatState(): ExportFormatState {
  return useExportDialogContext().formatState;
}

/**
 * Hook to access dimension state from context.
 */
export function useExportDimensionState(): ExportDimensionState {
  return useExportDialogContext().dimensionState;
}

/**
 * Hook to access file state from context.
 */
export function useExportFileState(): ExportFileState {
  return useExportDialogContext().fileState;
}

/**
 * Hook to access render source state from context.
 */
export function useExportRenderSourceState(): ExportRenderSourceState {
  return useExportDialogContext().renderSourceState;
}

/**
 * Hook to access preview state from context.
 */
export function useExportPreviewState(): ExportPreviewState {
  return useExportDialogContext().previewState;
}

/**
 * Hook to access progress state from context.
 */
export function useExportProgressState(): ExportProgressState {
  return useExportDialogContext().progressState;
}

/**
 * Hook to access metrics state from context.
 */
export function useExportMetricsState(): ExportMetricsState {
  return useExportDialogContext().metricsState;
}

/**
 * Hook to access dialog actions from context.
 */
export function useExportDialogActions(): ExportDialogActions {
  return useExportDialogContext().actions;
}

// =============================================================================
// Provider
// =============================================================================

interface ExportDialogProviderProps {
  children: ReactNode;
  value: ExportDialogContextValue;
}

/**
 * Provider component for export dialog context.
 */
export function ExportDialogProvider({
  children,
  value,
}: ExportDialogProviderProps): JSX.Element {
  return (
    <ExportDialogContext.Provider value={value}>
      {children}
    </ExportDialogContext.Provider>
  );
}

// =============================================================================
// Builder
// =============================================================================

/**
 * Options for building export dialog context value.
 * This maps from the current useExecutionExport return format to the context value format.
 */
export interface BuildExportDialogContextOptions {
  // Dialog IDs
  dialogTitleId: string;
  dialogDescriptionId: string;

  // Format state
  exportFormat: ExportFormat;
  setExportFormat: (format: ExportFormat) => void;
  isBinaryExport: boolean;
  exportFormatOptions: ExportFormatOption[];

  // Dimension state
  dimensionPreset: ExportDimensionPreset;
  setDimensionPreset: (preset: ExportDimensionPreset) => void;
  dimensionPresetOptions: Array<{
    id: ExportDimensionPreset;
    label: string;
    width?: number;
    height?: number;
    description?: string;
  }>;
  selectedDimensions: ExportDimensions;
  customWidthInput: string;
  customHeightInput: string;
  setCustomWidthInput: (value: string) => void;
  setCustomHeightInput: (value: string) => void;

  // File state
  exportFileStem: string;
  setExportFileStem: (stem: string) => void;
  defaultExportFileStem: string;
  finalFileName: string;
  supportsFileSystemAccess: boolean;
  useNativeFilePicker: boolean;
  setUseNativeFilePicker: (use: boolean) => void;

  // Render source state
  renderSource: ExportRenderSource;
  setRenderSource: (source: ExportRenderSource) => void;
  exportRenderSourceOptions: ExportRenderSourceOption[];
  recordedVideoAvailable: boolean;
  recordedVideoCount: number;
  recordedVideoLoading: boolean;

  // Preview state
  preparedMovieSpec: ReplayMovieSpec | null;
  composerPreviewUrl: string;
  firstFramePreviewUrl: string | null;
  firstFrameLabel: string | null;
  previewComposerRef: MutableRefObject<HTMLIFrameElement | null>;
  composerPreviewWindowRef: MutableRefObject<Window | null>;
  composerPreviewOriginRef: MutableRefObject<string | null>;
  isPreviewComposerReady: boolean;
  setIsPreviewComposerReady: (ready: boolean) => void;
  previewComposerError: string | null;
  setPreviewComposerError: (error: string | null) => void;

  // Progress state
  isExporting: boolean;
  isExportPreviewLoading: boolean;
  exportStatusMessage: string;

  // Metrics state
  replayFramesLength: number;
  estimatedFrameCount?: number;
  estimatedDurationSeconds: number | null;
  activeSpecId: string | null;
  previewMetrics: {
    capturedFrames: number;
    assetCount: number;
    totalDurationMs: number;
  };

  // Actions
  onClose: () => void;
  onConfirm: () => void;

  // Utilities
  formatSeconds: (value: number) => string;
}

/**
 * Builds an export dialog context value from the flat props structure.
 * This is the bridge between the current useExecutionExport hook and the new context.
 */
export function buildExportDialogContextValue(
  options: BuildExportDialogContextOptions,
): ExportDialogContextValue {
  return {
    titleId: options.dialogTitleId,
    descriptionId: options.dialogDescriptionId,

    formatState: {
      format: options.exportFormat,
      setFormat: options.setExportFormat,
      isBinaryExport: options.isBinaryExport,
      formatOptions: options.exportFormatOptions,
    },

    dimensionState: {
      preset: options.dimensionPreset,
      setPreset: options.setDimensionPreset,
      presetOptions: options.dimensionPresetOptions,
      selectedDimensions: options.selectedDimensions,
      customWidthInput: options.customWidthInput,
      customHeightInput: options.customHeightInput,
      setCustomWidthInput: options.setCustomWidthInput,
      setCustomHeightInput: options.setCustomHeightInput,
    },

    fileState: {
      fileStem: options.exportFileStem,
      setFileStem: options.setExportFileStem,
      defaultFileStem: options.defaultExportFileStem,
      finalFileName: options.finalFileName,
      supportsFileSystemAccess: options.supportsFileSystemAccess,
      useNativeFilePicker: options.useNativeFilePicker,
      setUseNativeFilePicker: options.setUseNativeFilePicker,
    },

    renderSourceState: {
      source: options.renderSource,
      setSource: options.setRenderSource,
      sourceOptions: options.exportRenderSourceOptions,
      recordedVideoAvailable: options.recordedVideoAvailable,
      recordedVideoCount: options.recordedVideoCount,
      recordedVideoLoading: options.recordedVideoLoading,
    },

    previewState: {
      movieSpec: options.preparedMovieSpec,
      composerPreviewUrl: options.composerPreviewUrl,
      firstFramePreviewUrl: options.firstFramePreviewUrl,
      firstFrameLabel: options.firstFrameLabel,
      composerRef: options.previewComposerRef,
      composerWindowRef: options.composerPreviewWindowRef,
      composerOriginRef: options.composerPreviewOriginRef,
      isComposerReady: options.isPreviewComposerReady,
      setIsComposerReady: options.setIsPreviewComposerReady,
      composerError: options.previewComposerError,
      setComposerError: options.setPreviewComposerError,
    },

    progressState: {
      isExporting: options.isExporting,
      isPreviewLoading: options.isExportPreviewLoading,
      statusMessage: options.exportStatusMessage,
    },

    metricsState: {
      replayFramesLength: options.replayFramesLength,
      estimatedFrameCount: options.estimatedFrameCount,
      estimatedDurationSeconds: options.estimatedDurationSeconds,
      activeSpecId: options.activeSpecId,
      previewMetrics: options.previewMetrics,
    },

    actions: {
      onClose: options.onClose,
      onConfirm: options.onConfirm,
    },

    formatSeconds: options.formatSeconds,
  };
}
