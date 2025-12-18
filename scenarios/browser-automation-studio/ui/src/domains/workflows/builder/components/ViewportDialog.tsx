import { useState, useEffect } from 'react';
import type { ExecutionViewportSettings, ViewportPreset } from '@stores/workflowStore';
import { ResponsiveDialog } from '@shared/layout';
import { selectors } from '@constants/selectors';

const DEFAULT_DESKTOP_VIEWPORT: ExecutionViewportSettings = {
  width: 1920,
  height: 1080,
  preset: 'desktop',
};

const DEFAULT_MOBILE_VIEWPORT: ExecutionViewportSettings = {
  width: 390,
  height: 844,
  preset: 'mobile',
};

const MIN_VIEWPORT_DIMENSION = 200;
const MAX_VIEWPORT_DIMENSION = 10000;

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
  if (width === DEFAULT_DESKTOP_VIEWPORT.width && height === DEFAULT_DESKTOP_VIEWPORT.height) {
    return 'desktop';
  }
  if (width === DEFAULT_MOBILE_VIEWPORT.width && height === DEFAULT_MOBILE_VIEWPORT.height) {
    return 'mobile';
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
  const [widthValue, setWidthValue] = useState('');
  const [heightValue, setHeightValue] = useState('');
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!isOpen) {
      return;
    }

    const normalized = normalizeViewportSetting(initialValue);
    setWidthValue(String(normalized.width));
    setHeightValue(String(normalized.height));
    setError(null);
  }, [initialValue, isOpen]);

  const numericWidth = Number.parseInt(widthValue, 10);
  const numericHeight = Number.parseInt(heightValue, 10);
  const activePreset = determineViewportPreset(numericWidth, numericHeight);

  const handlePresetSelect = (presetViewport: ExecutionViewportSettings) => {
    const normalized = normalizeViewportSetting(presetViewport);
    setWidthValue(String(normalized.width));
    setHeightValue(String(normalized.height));
    setError(null);
  };

  const handleWidthChange = (value: string) => {
    setWidthValue(value.replace(/[^0-9]/g, ''));
  };

  const handleHeightChange = (value: string) => {
    setHeightValue(value.replace(/[^0-9]/g, ''));
  };

  const handleSave = () => {
    const parsedWidth = Number.parseInt(widthValue, 10);
    const parsedHeight = Number.parseInt(heightValue, 10);

    if (!Number.isFinite(parsedWidth) || parsedWidth < MIN_VIEWPORT_DIMENSION) {
      setError(
        `Width must be between ${MIN_VIEWPORT_DIMENSION} and ${MAX_VIEWPORT_DIMENSION} pixels.`
      );
      return;
    }

    if (!Number.isFinite(parsedHeight) || parsedHeight < MIN_VIEWPORT_DIMENSION) {
      setError(
        `Height must be between ${MIN_VIEWPORT_DIMENSION} and ${MAX_VIEWPORT_DIMENSION} pixels.`
      );
      return;
    }

    if (parsedWidth > MAX_VIEWPORT_DIMENSION || parsedHeight > MAX_VIEWPORT_DIMENSION) {
      setError(`Dimensions cannot exceed ${MAX_VIEWPORT_DIMENSION} pixels.`);
      return;
    }

    const width = clampViewportDimension(parsedWidth);
    const height = clampViewportDimension(parsedHeight);
    const preset = determineViewportPreset(width, height);

    onSave({ width, height, preset });
  };

  if (!isOpen) {
    return null;
  }

  const presetButtons: Array<{
    id: ViewportPreset;
    label: string;
    viewport: ExecutionViewportSettings;
  }> = [
    { id: 'desktop', label: 'Desktop', viewport: DEFAULT_DESKTOP_VIEWPORT },
    { id: 'mobile', label: 'Mobile', viewport: DEFAULT_MOBILE_VIEWPORT },
  ];

  return (
    <ResponsiveDialog
      isOpen={isOpen}
      onDismiss={onDismiss}
      ariaLabel="Configure execution dimensions"
      className="bg-flow-node border border-gray-800 rounded-lg shadow-2xl w-[360px] max-w-[90vw]"
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

      <div className="px-6 py-5 space-y-5">
        <div>
          <span className="text-xs font-semibold uppercase tracking-wide text-gray-400">
            Presets
          </span>
          <div className="mt-2 grid grid-cols-2 gap-2">
            {presetButtons.map(({ id, label, viewport }) => {
              const isActive = activePreset === id;
              return (
                <button
                  type="button"
                  key={id}
                  onClick={() => handlePresetSelect(viewport)}
                  className={`flex flex-col rounded-md border px-3 py-2 text-left text-xs transition-colors ${
                    isActive
                      ? 'border-flow-accent bg-flow-accent/20 text-surface'
                      : 'border-gray-700 text-subtle hover:border-flow-accent hover:text-surface'
                  }`}
                  data-testid={selectors.viewport.dialog.presetButton({ preset: id })}
                >
                  <span className="font-semibold text-sm">{label}</span>
                  <span className="mt-0.5 text-[11px] text-gray-400">
                    {viewport.width} × {viewport.height}
                  </span>
                </button>
              );
            })}
          </div>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <label className="block text-xs font-semibold uppercase tracking-wide text-gray-400">
            Width (px)
            <input
              type="number"
              min={MIN_VIEWPORT_DIMENSION}
              max={MAX_VIEWPORT_DIMENSION}
              value={widthValue}
              onChange={(event) => handleWidthChange(event.target.value)}
              className="mt-1 w-full rounded-md border border-gray-700 bg-flow-bg px-3 py-2 text-sm text-surface focus:border-flow-accent focus:outline-none"
              data-testid={selectors.viewport.dialog.widthInput}
            />
          </label>
          <label className="block text-xs font-semibold uppercase tracking-wide text-gray-400">
            Height (px)
            <input
              type="number"
              min={MIN_VIEWPORT_DIMENSION}
              max={MAX_VIEWPORT_DIMENSION}
              value={heightValue}
              onChange={(event) => handleHeightChange(event.target.value)}
              className="mt-1 w-full rounded-md border border-gray-700 bg-flow-bg px-3 py-2 text-sm text-surface focus:border-flow-accent focus:outline-none"
              data-testid={selectors.viewport.dialog.heightInput}
            />
          </label>
        </div>

        <p className="text-xs text-gray-500">
          Recommended desktop preset works well for most workflows. Use custom
          dimensions for responsive testing or narrow layouts.
        </p>

        {error && (
          <div
            className="rounded-md border border-red-500/60 bg-red-500/10 px-3 py-2 text-xs text-red-300"
            data-testid={selectors.viewport.dialog.error}
          >
            {error}
          </div>
        )}
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
