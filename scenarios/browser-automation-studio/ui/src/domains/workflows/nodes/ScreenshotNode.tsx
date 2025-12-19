import { memo, FC, useState } from 'react';
import type { NodeProps } from 'reactflow';
import {
  Camera,
  Globe,
  SlidersHorizontal,
  Focus,
  Wand2,
  Droplet,
  ZoomIn,
  Palette,
  Timer,
} from 'lucide-react';
import { useActionParams } from '@hooks/useActionParams';
import { useNodeData } from '@hooks/useNodeData';
import {
  useSyncedString,
  useSyncedBoolean,
  useSyncedField,
  textInputHandler,
} from '@hooks/useSyncedField';
import type { ScreenshotParams } from '@utils/actionBuilder';
import BaseNode from './BaseNode';
import {
  NodeTextField,
  NodeTextArea,
  NodeCheckbox,
  NodeUrlField,
  FieldRow,
} from './fields';

// Helper to parse selector lists from comma/newline separated string
const parseSelectorList = (raw: string): string[] =>
  raw
    .split(/[\n,]+/)
    .map((entry) => entry.trim())
    .filter((entry) => entry.length > 0);

// Helper to format selector lists to comma-separated string
const formatSelectorList = (selectors: string[] | undefined): string =>
  Array.isArray(selectors) ? selectors.join(', ') : '';

// Helper to parse optional number from string
const parseOptionalNumber = (
  value: string,
  options: { float?: boolean; min?: number; max?: number } = {},
): number | undefined => {
  if (value === '') return undefined;
  const parsed = options.float ? Number.parseFloat(value) : Number.parseInt(value, 10);
  if (Number.isNaN(parsed)) return undefined;
  let result = parsed;
  if (options.min !== undefined) result = Math.max(options.min, result);
  if (options.max !== undefined) result = Math.min(options.max, result);
  return result;
};

const ScreenshotNode: FC<NodeProps> = ({ selected, id }) => {
  // Node data hook for UI-specific fields
  const { getValue, updateData } = useNodeData(id);

  // V2 Native: Use action params as source of truth
  const { params, updateParams } = useActionParams<ScreenshotParams>(id);

  // Action params fields
  const fullPage = useSyncedBoolean(params?.fullPage ?? false, {
    onCommit: (v) => updateParams({ fullPage: v }),
  });

  // String fields (node.data)
  const name = useSyncedString(getValue<string>('name') ?? '', {
    onCommit: (v) => updateData({ name: v || undefined }),
  });
  const focusSelector = useSyncedString(getValue<string>('focusSelector') ?? '', {
    onCommit: (v) => updateData({ focusSelector: v || undefined }),
  });
  const highlightColor = useSyncedString(getValue<string>('highlightColor') ?? '', {
    onCommit: (v) => updateData({ highlightColor: v || undefined }),
  });
  const background = useSyncedString(getValue<string>('background') ?? '', {
    onCommit: (v) => updateData({ background: v || undefined }),
  });

  // Selector list fields (stored as arrays, edited as strings)
  const highlightInput = useSyncedField(formatSelectorList(getValue<string[]>('highlightSelectors')), {
    onCommit: (v) => {
      const selectors = parseSelectorList(v);
      updateData({ highlightSelectors: selectors.length > 0 ? selectors : undefined });
    },
  });
  const maskInput = useSyncedField(formatSelectorList(getValue<string[]>('maskSelectors')), {
    onCommit: (v) => {
      const selectors = parseSelectorList(v);
      updateData({ maskSelectors: selectors.length > 0 ? selectors : undefined });
    },
  });

  // Optional number fields (stored as numbers, edited as strings for optional handling)
  const highlightPadding = useSyncedField(getValue<number>('highlightPadding')?.toString() ?? '', {
    onCommit: (v) => updateData({ highlightPadding: parseOptionalNumber(v, { min: 0 }) }),
  });
  const maskOpacity = useSyncedField(getValue<number>('maskOpacity')?.toString() ?? '', {
    onCommit: (v) => updateData({ maskOpacity: parseOptionalNumber(v, { float: true, min: 0, max: 1 }) }),
  });
  const zoomFactor = useSyncedField(getValue<number>('zoomFactor')?.toString() ?? '', {
    onCommit: (v) => updateData({ zoomFactor: parseOptionalNumber(v, { float: true, min: 0 }) }),
  });
  const viewportWidth = useSyncedField(getValue<number>('viewportWidth')?.toString() ?? '', {
    onCommit: (v) => updateData({ viewportWidth: parseOptionalNumber(v, { min: 320 }) }),
  });
  const viewportHeight = useSyncedField(getValue<number>('viewportHeight')?.toString() ?? '', {
    onCommit: (v) => updateData({ viewportHeight: parseOptionalNumber(v, { min: 320 }) }),
  });
  const waitForMs = useSyncedField(getValue<number>('waitForMs')?.toString() ?? '', {
    onCommit: (v) => updateData({ waitForMs: parseOptionalNumber(v, { min: 0 }) }),
  });

  // UI state (not persisted)
  const [showAdvanced, setShowAdvanced] = useState<boolean>(false);

  return (
    <BaseNode
      selected={selected}
      icon={Camera}
      iconClassName="text-purple-400"
      title="Screenshot"
      className="w-80"
    >
      <NodeUrlField nodeId={id} errorMessage="Provide a URL to capture screenshots from." />

      <input
        type="text"
        placeholder="Screenshot name..."
        className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none mb-2"
        value={name.value}
        onChange={textInputHandler(name.setValue)}
        onBlur={name.commit}
      />

      <NodeCheckbox field={fullPage} label="Full page" />

      <button
        type="button"
        className="mt-3 mb-2 flex items-center gap-2 text-xs text-gray-400 hover:text-white transition-colors"
        onClick={() => setShowAdvanced((prev) => !prev)}
      >
        <SlidersHorizontal size={14} />
        {showAdvanced ? 'Hide advanced capture options' : 'Show advanced capture options'}
      </button>

      {showAdvanced && (
        <div className="space-y-3 border-t border-gray-800 pt-3 text-xs">
          <NodeTextField
            field={focusSelector}
            label="Focus selector"
            placeholder="CSS selector to focus before capture"
            icon={Focus}
            iconClassName="text-blue-300"
          />

          <NodeTextArea
            field={highlightInput}
            label="Highlight selectors"
            placeholder="e.g. h1, .cta-button"
            description="comma or newline separated"
            rows={2}
            icon={Wand2}
            iconClassName="text-purple-300"
          />

          <FieldRow>
            <NodeTextField
              field={highlightColor}
              label="Highlight color"
              placeholder="#00E5FF"
              icon={Palette}
              iconClassName="text-sky-300"
            />
            <div>
              <label className="text-xs text-gray-400 flex items-center gap-2 mb-1">
                <Wand2 size={12} className="text-purple-300" /> Padding (px)
              </label>
              <input
                type="number"
                min={0}
                className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
                value={highlightPadding.value}
                onChange={(e) => highlightPadding.setValue(e.target.value)}
                onBlur={highlightPadding.commit}
              />
            </div>
          </FieldRow>

          <NodeTextArea
            field={maskInput}
            label="Mask selectors"
            placeholder="e.g. .ads, .cookie-banner"
            description="hide sensitive areas"
            rows={2}
            icon={Droplet}
            iconClassName="text-emerald-300"
          />

          <FieldRow>
            <div>
              <label className="text-xs text-gray-400 flex items-center gap-2 mb-1">
                <Droplet size={12} className="text-emerald-300" /> Mask opacity
              </label>
              <input
                type="number"
                min={0}
                max={1}
                step={0.05}
                className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
                value={maskOpacity.value}
                onChange={(e) => maskOpacity.setValue(e.target.value)}
                onBlur={maskOpacity.commit}
              />
            </div>
            <div>
              <label className="text-xs text-gray-400 flex items-center gap-2 mb-1">
                <ZoomIn size={12} className="text-pink-300" /> Zoom factor
              </label>
              <input
                type="number"
                min={0}
                step={0.05}
                className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
                value={zoomFactor.value}
                onChange={(e) => zoomFactor.setValue(e.target.value)}
                onBlur={zoomFactor.commit}
              />
            </div>
          </FieldRow>

          <FieldRow>
            <div>
              <label className="text-xs text-gray-400 flex items-center gap-2 mb-1">
                <Globe size={12} className="text-blue-300" /> Viewport width
              </label>
              <input
                type="number"
                min={320}
                className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
                value={viewportWidth.value}
                onChange={(e) => viewportWidth.setValue(e.target.value)}
                onBlur={viewportWidth.commit}
              />
            </div>
            <div>
              <label className="text-xs text-gray-400 flex items-center gap-2 mb-1">
                <Globe size={12} className="text-blue-300" /> Viewport height
              </label>
              <input
                type="number"
                min={320}
                className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
                value={viewportHeight.value}
                onChange={(e) => viewportHeight.setValue(e.target.value)}
                onBlur={viewportHeight.commit}
              />
            </div>
          </FieldRow>

          <FieldRow>
            <NodeTextField
              field={background}
              label="Page background"
              placeholder="#0f172a or linear-gradient(...)"
              icon={Palette}
              iconClassName="text-sky-300"
            />
            <div>
              <label className="text-xs text-gray-400 flex items-center gap-2 mb-1">
                <Timer size={12} className="text-amber-300" /> Wait before capture (ms)
              </label>
              <input
                type="number"
                min={0}
                className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
                value={waitForMs.value}
                onChange={(e) => waitForMs.setValue(e.target.value)}
                onBlur={waitForMs.commit}
              />
            </div>
          </FieldRow>
        </div>
      )}
    </BaseNode>
  );
};

export default memo(ScreenshotNode);
