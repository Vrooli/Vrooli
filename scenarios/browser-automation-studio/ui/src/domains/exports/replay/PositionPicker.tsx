/**
 * PositionPicker - Visual grid for selecting watermark position
 */

import type { WatermarkPosition } from '@stores/settingsStore';

interface PositionPickerProps {
  value: WatermarkPosition;
  onChange: (position: WatermarkPosition) => void;
  disabled?: boolean;
}

const POSITIONS: Array<{ id: WatermarkPosition; row: number; col: number }> = [
  { id: 'top-left', row: 0, col: 0 },
  { id: 'top-right', row: 0, col: 2 },
  { id: 'center', row: 1, col: 1 },
  { id: 'bottom-left', row: 2, col: 0 },
  { id: 'bottom-right', row: 2, col: 2 },
];

export function PositionPicker({ value, onChange, disabled = false }: PositionPickerProps) {
  return (
    <div className="inline-grid grid-cols-3 gap-1.5">
      {[0, 1, 2].map((row) =>
        [0, 1, 2].map((col) => {
          const position = POSITIONS.find((p) => p.row === row && p.col === col);
          const isActive = position?.id === value;
          const isClickable = !!position;

          return (
            <button
              key={`${row}-${col}`}
              type="button"
              onClick={() => position && onChange(position.id)}
              disabled={disabled || !isClickable}
              className={`
                w-7 h-7 rounded-md transition-all flex items-center justify-center
                ${
                  isClickable
                    ? isActive
                      ? 'bg-flow-accent ring-2 ring-flow-accent/50'
                      : 'bg-gray-700 hover:bg-gray-600'
                    : 'bg-gray-800/30'
                }
                ${disabled ? 'opacity-50 cursor-not-allowed' : isClickable ? 'cursor-pointer' : 'cursor-default'}
              `}
              aria-label={position?.id || 'empty'}
            >
              {isActive && (
                <div className="w-2.5 h-2.5 bg-white rounded-full" />
              )}
              {isClickable && !isActive && (
                <div className="w-1.5 h-1.5 bg-gray-500 rounded-full" />
              )}
            </button>
          );
        }),
      )}
    </div>
  );
}

export default PositionPicker;
