import { FC, memo, useMemo } from 'react';
import type { NodeProps } from 'reactflow';
import { Smartphone } from 'lucide-react';
import { useActionParams } from '@hooks/useActionParams';
import { useSyncedNumber, useSyncedSelect } from '@hooks/useSyncedField';
import BaseNode from './BaseNode';
import { NodeNumberField, NodeSelectField } from './fields';

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

const ORIENTATION_OPTIONS = [
  { value: ORIENTATION_PORTRAIT, label: 'Portrait (default)' },
  { value: ORIENTATION_LANDSCAPE, label: 'Landscape (swap width/height)' },
];

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
        <NodeSelectField field={orientation} label="Orientation" options={ORIENTATION_OPTIONS} />

        {/* Angle field has special handling - options depend on orientation */}
        <div>
          <label className="text-gray-400 block mb-1 text-xs">Angle</label>
          <select
            value={String(angle.value)}
            onChange={(event) => handleAngleChange(event.target.value)}
            className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
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

        <NodeNumberField
          field={waitForMs}
          label="Wait after rotate (ms)"
          min={0}
          step={50}
          placeholder="250"
          description="Give responsive layouts time to settle after orientation changes."
        />
      </div>
    </BaseNode>
  );
};

export default memo(RotateNode);
