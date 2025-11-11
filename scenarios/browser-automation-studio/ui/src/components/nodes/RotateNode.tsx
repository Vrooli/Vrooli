import { FC, memo, useCallback, useEffect, useMemo, useState } from 'react';
import { Handle, NodeProps, Position, useReactFlow } from 'reactflow';
import { Smartphone } from 'lucide-react';

const ORIENTATION_PORTRAIT = 'portrait';
const ORIENTATION_LANDSCAPE = 'landscape';
const PORTRAIT_ANGLES: readonly number[] = [0, 180];
const LANDSCAPE_ANGLES: readonly number[] = [90, 270];

const normalizeOrientation = (value: unknown): string => {
  const text = typeof value === 'string' ? value.toLowerCase() : '';
  if (text === ORIENTATION_LANDSCAPE) {
    return ORIENTATION_LANDSCAPE;
  }
  return ORIENTATION_PORTRAIT;
};

const allowedAnglesForOrientation = (orientation: string): readonly number[] => (
  orientation === ORIENTATION_LANDSCAPE ? LANDSCAPE_ANGLES : PORTRAIT_ANGLES
);

const deriveAngle = (raw: unknown, orientation: string): number => {
  const numeric = Number(raw);
  const allowed = allowedAnglesForOrientation(orientation);
  if (Number.isFinite(numeric) && allowed.includes(Number(numeric))) {
    return Number(numeric);
  }
  return allowed[0];
};

const clampWait = (value: unknown): number => {
  const numeric = Number(value);
  if (!Number.isFinite(numeric)) {
    return 0;
  }
  return Math.max(0, Math.round(numeric));
};

const RotateNode: FC<NodeProps> = ({ data, selected, id }) => {
  const nodeData = (data ?? {}) as Record<string, unknown>;
  const { getNodes, setNodes } = useReactFlow();

  const dataOrientation = normalizeOrientation(nodeData.orientation);
  const [orientation, setOrientation] = useState<string>(dataOrientation);
  const [angle, setAngle] = useState<number>(() => deriveAngle(nodeData.angle, dataOrientation));
  const [waitForMs, setWaitForMs] = useState<number>(() => clampWait(nodeData.waitForMs));

  useEffect(() => {
    setOrientation(dataOrientation);
  }, [dataOrientation]);

  useEffect(() => {
    setAngle(deriveAngle(nodeData.angle, dataOrientation));
  }, [nodeData.angle, dataOrientation]);

  useEffect(() => {
    setWaitForMs(clampWait(nodeData.waitForMs));
  }, [nodeData.waitForMs]);

  const updateNodeData = useCallback((updates: Record<string, unknown>) => {
    const nodes = getNodes();
    setNodes(nodes.map((node) => {
      if (node.id !== id) {
        return node;
      }
      const nextData = { ...(node.data ?? {}) } as Record<string, unknown>;
      Object.entries(updates).forEach(([key, value]) => {
        if (value === undefined) {
          delete nextData[key];
        } else {
          nextData[key] = value;
        }
      });
      return { ...node, data: nextData };
    }));
  }, [getNodes, setNodes, id]);

  const allowedAngles = useMemo(() => allowedAnglesForOrientation(orientation), [orientation]);

  const handleOrientationChange = useCallback((value: string) => {
    const normalized = value === ORIENTATION_LANDSCAPE ? ORIENTATION_LANDSCAPE : ORIENTATION_PORTRAIT;
    const nextAllowedAngles = allowedAnglesForOrientation(normalized);
    const nextAngle = nextAllowedAngles.includes(angle) ? angle : nextAllowedAngles[0];
    setOrientation(normalized);
    setAngle(nextAngle);
    updateNodeData({ orientation: normalized, angle: nextAngle });
  }, [angle, updateNodeData]);

  const handleAngleChange = useCallback((value: string) => {
    const parsed = Number(value);
    if (!Number.isFinite(parsed)) {
      return;
    }
    const normalized = allowedAngles.includes(parsed) ? parsed : allowedAngles[0];
    setAngle(normalized);
    updateNodeData({ angle: normalized });
  }, [allowedAngles, updateNodeData]);

  const handleWaitBlur = useCallback(() => {
    const normalized = clampWait(waitForMs);
    setWaitForMs(normalized);
    updateNodeData({ waitForMs: normalized || undefined });
  }, [waitForMs, updateNodeData]);

  return (
    <div className={`workflow-node ${selected ? 'selected' : ''}`}>
      <Handle type="target" position={Position.Top} className="node-handle" />

      <div className="flex items-start gap-2 mb-3">
        <Smartphone size={16} className="text-sky-300" />
        <div>
          <div className="font-semibold text-sm">Rotate</div>
          <p className="text-[11px] text-gray-500">Swap between portrait & landscape during mobile flows.</p>
        </div>
      </div>

      <div className="space-y-3 text-xs">
        <div>
          <label className="text-gray-400 block mb-1">Orientation</label>
          <select
            value={orientation}
            onChange={(event) => handleOrientationChange(event.target.value)}
            className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
          >
            <option value={ORIENTATION_PORTRAIT}>Portrait (default)</option>
            <option value={ORIENTATION_LANDSCAPE}>Landscape (swap width/height)</option>
          </select>
        </div>

        <div>
          <label className="text-gray-400 block mb-1">Angle</label>
          <select
            value={String(angle)}
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
            value={waitForMs}
            onChange={(event) => setWaitForMs(Number(event.target.value))}
            onBlur={handleWaitBlur}
            className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            placeholder="250"
          />
          <p className="text-[10px] text-gray-500 mt-1">Give responsive layouts time to settle after orientation changes.</p>
        </div>
      </div>

      <Handle type="source" position={Position.Bottom} className="node-handle" />
    </div>
  );
};

export default memo(RotateNode);
