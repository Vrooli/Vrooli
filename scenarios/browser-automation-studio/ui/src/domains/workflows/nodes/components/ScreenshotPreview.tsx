/**
 * ScreenshotPreview Component
 *
 * Displays a screenshot with element picker functionality.
 * Supports highlight overlays for selected/hovered elements.
 */

import { FC, memo, useCallback, useMemo, useRef, useState } from 'react';
import { ChevronDown, ChevronRight, Loader2, Sparkles } from 'lucide-react';
import type { BoundingBox } from '@/types/elements';

const DEFAULT_ASPECT_RATIO = '16 / 9';

export interface ScreenshotPreviewProps {
  /** Base64 data URL of the screenshot */
  screenshot: string | null;
  /** Whether the preview panel is expanded */
  isOpen: boolean;
  /** Toggle the preview panel open/closed */
  onToggle: () => void;
  /** Label describing the screenshot source */
  sourceLabel?: string | null;
  /** Timestamp when screenshot was captured */
  capturedAt?: string | null;
  /** Whether picker mode is active */
  pickerActive: boolean;
  /** Whether an element selection is in progress */
  isSelecting: boolean;
  /** Bounding box of the currently selected element */
  selectedBoundingBox?: BoundingBox | null;
  /** Bounding box of a hovered suggestion element */
  hoveredBoundingBox?: BoundingBox | null;
  /** Viewport settings for aspect ratio calculation */
  viewport?: { width: number; height: number } | null;
  /** Called when user clicks on the screenshot in picker mode */
  onPickerClick: (x: number, y: number) => void;
  /** Whether AI suggestions can be shown */
  canShowAiSuggestions: boolean;
  /** Toggle AI suggestions panel */
  onToggleAiPanel: () => void;
}

const ScreenshotPreview: FC<ScreenshotPreviewProps> = ({
  screenshot,
  isOpen,
  onToggle,
  sourceLabel,
  capturedAt,
  pickerActive,
  isSelecting,
  selectedBoundingBox,
  hoveredBoundingBox,
  viewport,
  onPickerClick,
  canShowAiSuggestions,
  onToggleAiPanel,
}) => {
  const imageRef = useRef<HTMLImageElement | null>(null);
  const [imageSize, setImageSize] = useState<{ width: number; height: number } | null>(null);
  const [hoverPosition, setHoverPosition] = useState<{ x: number; y: number } | null>(null);
  const [clickPosition, setClickPosition] = useState<{ x: number; y: number } | null>(null);

  const getImageCoordinates = useCallback((event: React.MouseEvent<HTMLImageElement>) => {
    const img = event.currentTarget;
    const rect = img.getBoundingClientRect();
    if (!img.naturalWidth || !img.naturalHeight || rect.width <= 0 || rect.height <= 0) {
      return null;
    }

    const relativeX = event.clientX - rect.left;
    const relativeY = event.clientY - rect.top;
    const clampedX = Math.max(0, Math.min(relativeX, rect.width));
    const clampedY = Math.max(0, Math.min(relativeY, rect.height));
    const scaleX = img.naturalWidth / rect.width;
    const scaleY = img.naturalHeight / rect.height;

    return {
      x: Math.round(clampedX * scaleX),
      y: Math.round(clampedY * scaleY),
    };
  }, []);

  const handleImageLoad = useCallback((event: React.SyntheticEvent<HTMLImageElement>) => {
    const img = event.currentTarget;
    if (img?.naturalWidth && img?.naturalHeight) {
      setImageSize({ width: img.naturalWidth, height: img.naturalHeight });
    }
  }, []);

  const handleClick = useCallback(
    (event: React.MouseEvent<HTMLImageElement>) => {
      if (!pickerActive || !imageSize) {
        return;
      }

      const coordinates = getImageCoordinates(event);
      if (!coordinates) {
        return;
      }

      setClickPosition(coordinates);
      onPickerClick(coordinates.x, coordinates.y);
    },
    [getImageCoordinates, imageSize, onPickerClick, pickerActive],
  );

  const handleMouseMove = useCallback(
    (event: React.MouseEvent<HTMLImageElement>) => {
      if (!pickerActive || !imageSize) {
        return;
      }
      const coordinates = getImageCoordinates(event);
      setHoverPosition(coordinates);
    },
    [getImageCoordinates, imageSize, pickerActive],
  );

  const handleMouseLeave = useCallback(() => {
    setHoverPosition(null);
  }, []);

  // Calculate highlight style for bounding box overlay
  const highlightStyle = useMemo(() => {
    const source = hoveredBoundingBox ?? selectedBoundingBox;
    if (!source || !imageSize) {
      return null;
    }
    const { width: imgWidth, height: imgHeight } = imageSize;
    if (imgWidth <= 0 || imgHeight <= 0) {
      return null;
    }
    return {
      left: `${(source.x / imgWidth) * 100}%`,
      top: `${(source.y / imgHeight) * 100}%`,
      width: `${(source.width / imgWidth) * 100}%`,
      height: `${(source.height / imgHeight) * 100}%`,
    } as const;
  }, [hoveredBoundingBox, selectedBoundingBox, imageSize]);

  // Calculate aspect ratio for preview container
  const previewAspectRatio = useMemo(() => {
    if (
      viewport &&
      Number.isFinite(viewport.width) &&
      Number.isFinite(viewport.height) &&
      viewport.width > 0 &&
      viewport.height > 0
    ) {
      return `${viewport.width} / ${viewport.height}`;
    }
    if (imageSize && imageSize.width > 0 && imageSize.height > 0) {
      return `${imageSize.width} / ${imageSize.height}`;
    }
    return DEFAULT_ASPECT_RATIO;
  }, [viewport, imageSize]);

  const previewBoxStyle = useMemo(
    () => ({
      aspectRatio: previewAspectRatio,
      width: '100%',
      maxHeight: '240px',
    }),
    [previewAspectRatio],
  );

  const isHovered = Boolean(hoveredBoundingBox);

  return (
    <div className="border border-gray-800 rounded-lg bg-flow-bg/60 mb-3 overflow-hidden">
      {/* Header */}
      <button
        type="button"
        onClick={onToggle}
        className="w-full flex items-center justify-between px-3 py-2 text-xs text-gray-300 border-b border-gray-800 bg-flow-bg/60 hover:bg-flow-bg/80 transition-colors"
      >
        <span className="flex items-center gap-2">
          {isOpen ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
          Preview
        </span>
        {sourceLabel && <span className="text-[10px] text-gray-500">{sourceLabel}</span>}
      </button>

      {/* Screenshot area */}
      {isOpen &&
        (screenshot ? (
          <div className="relative w-full overflow-hidden bg-black/40" style={previewBoxStyle}>
            <img
              ref={imageRef}
              src={screenshot}
              alt="Upstream preview"
              className={`h-full w-full object-contain ${pickerActive ? 'cursor-crosshair' : 'cursor-default'}`}
              onLoad={handleImageLoad}
              onClick={handleClick}
              onMouseMove={handleMouseMove}
              onMouseLeave={handleMouseLeave}
            />

            {/* Picker mode overlay */}
            {pickerActive && (
              <div className="absolute inset-0 pointer-events-none">
                <div className="absolute inset-0 bg-black/25" />
                <div className="absolute top-2 left-2 bg-black/70 text-[11px] text-white px-2 py-1 rounded">
                  Click anywhere on the screenshot to select an element
                </div>
              </div>
            )}

            {/* Hover coordinates display */}
            {hoverPosition && pickerActive && (
              <div className="absolute top-2 right-2 bg-gray-900/80 text-[11px] text-gray-100 px-2 py-1 rounded font-mono pointer-events-none">
                ({hoverPosition.x}, {hoverPosition.y})
              </div>
            )}

            {/* Click position indicator */}
            {clickPosition && pickerActive && imageSize && (
              <div
                className="absolute w-3 h-3 bg-blue-500 border-2 border-white rounded-full pointer-events-none transform -translate-x-1/2 -translate-y-1/2"
                style={{
                  left: `${(clickPosition.x / imageSize.width) * 100}%`,
                  top: `${(clickPosition.y / imageSize.height) * 100}%`,
                }}
              />
            )}

            {/* Element highlight overlay */}
            {highlightStyle && !pickerActive && (
              <div
                className={`absolute rounded-sm pointer-events-none ${
                  isHovered
                    ? 'border-2 border-purple-400/80 bg-purple-400/10'
                    : 'border-2 border-emerald-400/90 bg-emerald-400/10'
                }`}
                style={highlightStyle}
              >
                <div
                  className={`absolute -top-4 left-0 text-black text-[10px] font-medium px-1 rounded ${
                    isHovered ? 'bg-purple-500' : 'bg-emerald-500'
                  }`}
                >
                  {isHovered ? 'Suggested element' : 'Selected element'}
                </div>
              </div>
            )}

            {/* Loading overlay during selection */}
            {isSelecting && (
              <div className="absolute inset-0 flex items-center justify-center bg-black/40">
                <Loader2 size={18} className="text-blue-400 animate-spin" />
              </div>
            )}
          </div>
        ) : (
          <div className="px-3 py-4 text-xs text-gray-500">
            Connect this node to a Navigate node with a captured preview or to a Screenshot node to
            view the screenshot here.
          </div>
        ))}

      {/* Footer */}
      {isOpen && screenshot && (
        <div className="flex items-center justify-between gap-2 px-3 py-2 border-t border-gray-800 bg-flow-bg/70">
          <button
            type="button"
            className={`inline-flex items-center gap-1 text-[11px] transition-colors ${
              canShowAiSuggestions
                ? 'text-gray-300 hover:text-white'
                : 'text-gray-500 cursor-not-allowed'
            }`}
            disabled={!canShowAiSuggestions}
            onClick={onToggleAiPanel}
          >
            <Sparkles size={12} className="text-yellow-400" />
            AI suggestions
          </button>
          {capturedAt && (
            <span className="text-[10px] text-gray-500">
              Captured {new Date(capturedAt).toLocaleTimeString()}
            </span>
          )}
        </div>
      )}
    </div>
  );
};

export default memo(ScreenshotPreview);
