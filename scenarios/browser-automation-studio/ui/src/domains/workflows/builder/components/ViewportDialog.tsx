import { useState, useEffect, useCallback } from 'react';
import type { ExecutionViewportSettings, ViewportPreset } from '@stores/workflowStore';
import { ResponsiveDialog } from '@shared/layout';
import { ViewportPicker, VIEWPORT_PRESETS } from '@shared/ui';
import { selectors } from '@constants/selectors';

const DEFAULT_DESKTOP_VIEWPORT: ExecutionViewportSettings = {
  width: 1920,
  height: 1080,
  preset: 'desktop',
};

const MIN_VIEWPORT_DIMENSION = 320;
const MAX_VIEWPORT_DIMENSION = 3840;

const clampViewportDimension = (value: number): number => {
  if (!Number.isFinite(value)) {
    return MIN_VIEWPORT_DIMENSION;
  }
  return Math.min(
    Math.max(Math.round(value), MIN_VIEWPORT_DIMENSION),
    MAX_VIEWPORT_DIMENSION
  );
};

const determineViewportPreset = (width: number, height: number): ViewportPreset => {
  if (!Number.isFinite(width) || !Number.isFinite(height)) {
    return 'custom';
  }
  // Check if matches any of the standard presets
  const matchingPreset = VIEWPORT_PRESETS.find(
    (p) => p.width === width && p.height === height
  );
  if (matchingPreset) {
    // Map to the ViewportPreset type based on dimensions
    if (width === 1920 && height === 1080) return 'desktop';
    if (width <= 500) return 'mobile';
  }
  return 'custom';
};

export const normalizeViewportSetting = (
  viewport?: ExecutionViewportSettings | null
): ExecutionViewportSettings => {
  if (
    !viewport ||
    !Number.isFinite(viewport.width) ||
    !Number.isFinite(viewport.height)
  ) {
    return { ...DEFAULT_DESKTOP_VIEWPORT };
  }
  const width = clampViewportDimension(viewport.width);
  const height = clampViewportDimension(viewport.height);
  const preset = viewport.preset ?? determineViewportPreset(width, height);
  return { width, height, preset };
};

interface ViewportDialogProps {
  isOpen: boolean;
  onDismiss: () => void;
  onSave: (viewport: ExecutionViewportSettings) => void;
  initialValue?: ExecutionViewportSettings;
}

export function ViewportDialog({
  isOpen,
  onDismiss,
  onSave,
  initialValue,
}: ViewportDialogProps) {
  const [dimensions, setDimensions] = useState({ width: 1920, height: 1080 });
  const [useCustom, setUseCustom] = useState(false);

  useEffect(() => {
    if (!isOpen) {
      return;
    }

    const normalized = normalizeViewportSetting(initialValue);
    setDimensions({ width: normalized.width, height: normalized.height });
    // Check if current dimensions match any preset
    const matchesPreset = VIEWPORT_PRESETS.some(
      (p) => p.width === normalized.width && p.height === normalized.height
    );
    setUseCustom(!matchesPreset);
  }, [initialValue, isOpen]);

  const handleDimensionsChange = useCallback(
    (newDimensions: { width: number; height: number }) => {
      setDimensions(newDimensions);
    },
    []
  );

  const handleSave = useCallback(() => {
    const width = clampViewportDimension(dimensions.width);
    const height = clampViewportDimension(dimensions.height);
    const preset = determineViewportPreset(width, height);

    onSave({ width, height, preset });
  }, [dimensions, onSave]);

  if (!isOpen) {
    return null;
  }

  return (
    <ResponsiveDialog
      isOpen={isOpen}
      onDismiss={onDismiss}
      ariaLabel="Configure execution dimensions"
      className="bg-flow-node border border-gray-800 rounded-lg shadow-2xl w-[420px] max-w-[90vw]"
      data-testid={selectors.viewport.dialog.root}
    >
      <div className="px-6 py-4 border-b border-gray-800">
        <h2
          className="text-lg font-semibold text-surface"
          data-testid={selectors.viewport.dialog.title}
        >
          Execution dimensions
        </h2>
        <p className="mt-1 text-sm text-gray-400">
          Apply these dimensions to workflow runs and preview screenshots.
        </p>
      </div>

      <div className="px-6 py-5">
        <ViewportPicker
          value={dimensions}
          onChange={handleDimensionsChange}
          useCustom={useCustom}
          onUseCustomChange={setUseCustom}
          showCustomToggle={true}
          variant="dark"
        />
      </div>

      <div className="flex items-center justify-end gap-3 border-t border-gray-800 px-6 py-4">
        <button
          type="button"
          className="rounded-md border border-gray-700 bg-flow-bg px-4 py-2 text-sm font-semibold text-surface hover:border-gray-500 hover:text-surface"
          onClick={onDismiss}
          data-testid={selectors.viewport.dialog.cancelButton}
        >
          Cancel
        </button>
        <button
          type="button"
          className="rounded-md bg-flow-accent px-4 py-2 text-sm font-semibold text-white hover:bg-blue-500 disabled:opacity-50"
          onClick={handleSave}
          data-testid={selectors.viewport.dialog.saveButton}
        >
          Save
        </button>
      </div>
    </ResponsiveDialog>
  );
}
