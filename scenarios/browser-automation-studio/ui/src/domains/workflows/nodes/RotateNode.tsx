import { FC, memo, useMemo } from 'react';
import type { NodeProps } from 'reactflow';
import { Smartphone } from 'lucide-react';
import { useActionParams } from '@hooks/useActionParams';
import {
  useSyncedNumber,
  useSyncedSelect,
  numberInputHandler,
  selectInputHandler,
} from '@hooks/useSyncedField';
import BaseNode from './BaseNode';

// RotateParams interface for V2 native action params
interface RotateParams {
  orientation?: string;
  angle?: number;
  waitForMs?: number;
}

const ORIENTATION_PORTRAIT = 'portrait';
const ORIENTATION_LANDSCAPE = 'landscape';
const PORTRAIT_ANGLES: readonly number[] = [0, 180];
const LANDSCAPE_ANGLES: readonly number[] = [90, 270];

const allowedAnglesForOrientation = (orientation: string): readonly number[] =>
  orientation === ORIENTATION_LANDSCAPE ? LANDSCAPE_ANGLES : PORTRAIT_ANGLES;

const RotateNode: FC<NodeProps> = ({ selected, id }) => {
  const { params, updateParams } = useActionParams<RotateParams>(id);

  // Select field for orientation
  const orientation = useSyncedSelect(params?.orientation ?? ORIENTATION_PORTRAIT, {
    onCommit: (v) => {
      const nextAllowedAngles = allowedAnglesForOrientation(v);
      const currentAngle = angle.value;
      const nextAngle = nextAllowedAngles.includes(currentAngle) ? currentAngle : nextAllowedAngles[0];
      updateParams({ orientation: v, angle: nextAngle });
      angle.setValue(nextAngle);
    },
  });

  // Number field for angle - handled specially since options depend on orientation
  const angle = useSyncedNumber(params?.angle ?? allowedAnglesForOrientation(params?.orientation ?? ORIENTATION_PORTRAIT)[0], {
    onCommit: (v) => updateParams({ angle: v }),
  });

  // Number field for wait
  const waitForMs = useSyncedNumber(params?.waitForMs ?? 0, {
    min: 0,
    onCommit: (v) => updateParams({ waitForMs: v || undefined }),
  });

  const allowedAngles = useMemo(() => allowedAnglesForOrientation(orientation.value), [orientation.value]);

  // Custom handler for angle since it's a select with numeric values
  const handleAngleChange = (value: string) => {
    const parsed = Number(value);
    if (Number.isFinite(parsed) && allowedAngles.includes(parsed)) {
      angle.setValue(parsed);
      updateParams({ angle: parsed });
    }
  };

  return (
    <BaseNode selected={selected} icon={Smartphone} iconClassName="text-sky-300" title="Rotate">
      <p className="text-[11px] text-gray-500 mb-3">
        Swap between portrait & landscape during mobile flows.
      </p>

      <div className="space-y-3 text-xs">
        <div>
          <label className="text-gray-400 block mb-1">Orientation</label>
          <select
            value={orientation.value}
            onChange={selectInputHandler(orientation.setValue, orientation.commit)}
            className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
          >
            <option value={ORIENTATION_PORTRAIT}>Portrait (default)</option>
            <option value={ORIENTATION_LANDSCAPE}>Landscape (swap width/height)</option>
          </select>
        </div>

        <div>
          <label className="text-gray-400 block mb-1">Angle</label>
          <select
            value={String(angle.value)}
            onChange={(event) => handleAngleChange(event.target.value)}
            className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
          >
            {allowedAngles.map((option) => (
              <option key={option} value={option.toString()}>
                {option === 0 && '0° (portrait primary)'}
                {option === 180 && '180° (portrait secondary)'}
                {option === 90 && '90° (landscape primary)'}
                {option === 270 && '270° (landscape secondary)'}
                {[0, 90, 180, 270].includes(option) ? null : `${option}°`}
              </option>
            ))}
          </select>
        </div>

        <div>
          <label className="text-gray-400 block mb-1">Wait after rotate (ms)</label>
          <input
            type="number"
            min={0}
            step={50}
            value={waitForMs.value}
            onChange={numberInputHandler(waitForMs.setValue)}
            onBlur={waitForMs.commit}
            className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            placeholder="250"
          />
          <p className="text-[10px] text-gray-500 mt-1">
            Give responsive layouts time to settle after orientation changes.
          </p>
        </div>
      </div>
    </BaseNode>
  );
};

export default memo(RotateNode);
