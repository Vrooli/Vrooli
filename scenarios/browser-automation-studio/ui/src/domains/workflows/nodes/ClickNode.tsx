import { memo, FC, useState, useEffect, useCallback, useMemo } from 'react';
import type { NodeProps } from 'reactflow';
import { MousePointer, Target } from 'lucide-react';
import toast from 'react-hot-toast';
import { useActionParams } from '@hooks/useActionParams';
import { useNodeData } from '@hooks/useNodeData';
import { useResiliencePanelProps } from '@hooks/useResiliencePanel';
import useUpstreamScreenshot from '@hooks/useUpstreamScreenshot';
import { useUrlInheritance } from '@hooks/useUrlInheritance';
import { useSyncedString, textInputHandler } from '@hooks/useSyncedField';
import type { ElementInfo, BoundingBox, ElementHierarchyEntry, ElementCoordinateResponse } from '@/types/elements';
import type { ClickParams } from '@utils/actionBuilder';
import { getConfig } from '@/config';
import { logger } from '@utils/logger';
import { useWorkflowStore, type ExecutionViewportSettings } from '@stores/workflowStore';
import { normalizeHierarchy, deriveSelector } from './utils/elementHierarchy';
import { DOMHierarchyNav, ScreenshotPreview, AISuggestionsPanel } from './components';
import ResiliencePanel from './ResiliencePanel';
import BaseNode from './BaseNode';
import { NodeUrlField } from './fields';

const ClickNode: FC<NodeProps> = ({ selected, id }) => {
  // URL inheritance for element picker
  const { effectiveUrl, upstreamUrl } = useUrlInheritance(id);

  const upstreamScreenshot = useUpstreamScreenshot(id);
  const screenshot = upstreamScreenshot?.dataUrl ?? null;
  const executionViewport = useWorkflowStore(
    (state) => state.currentWorkflow?.executionViewport as ExecutionViewportSettings | undefined,
  );

  const { getValue, updateData } = useNodeData(id);
  const rawHierarchyData = getValue<ElementHierarchyEntry[]>('elementHierarchy');
  const rawHierarchyIndex = getValue<number>('elementHierarchyIndex');

  // V2 Native: Use action params as source of truth
  const { params, updateParams } = useActionParams<ClickParams>(id);

  // Hierarchy state
  const [hierarchyCandidates, setHierarchyCandidates] = useState<ElementHierarchyEntry[]>(() =>
    normalizeHierarchy(rawHierarchyData),
  );
  const [hierarchyIndex, setHierarchyIndex] = useState<number>(() => {
    const normalized = normalizeHierarchy(rawHierarchyData);
    if (normalized.length === 0) {
      return -1;
    }
    const storedIndex = typeof rawHierarchyIndex === 'number' ? Number(rawHierarchyIndex) : 0;
    return normalized[storedIndex] ? storedIndex : 0;
  });

  // UI state
  const [isPreviewOpen, setIsPreviewOpen] = useState<boolean>(() => Boolean(screenshot));
  const [pickerActive, setPickerActive] = useState(false);
  const [isSelecting, setIsSelecting] = useState(false);
  const [showAiPanel, setShowAiPanel] = useState(false);
  const [aiSuggestions, setAiSuggestions] = useState<ElementInfo[]>([]);
  const [hoveredSuggestion, setHoveredSuggestion] = useState<ElementInfo | null>(null);

  // V2 Native: Selector field using useSyncedString
  const selector = useSyncedString(params?.selector ?? '', {
    onCommit: (v) => updateParams({ selector: v || undefined }),
  });

  // Sync hierarchy from node data
  useEffect(() => {
    const normalized = normalizeHierarchy(rawHierarchyData);
    setHierarchyCandidates(normalized);

    if (normalized.length === 0) {
      setHierarchyIndex(-1);
      return;
    }

    const storedIndex = typeof rawHierarchyIndex === 'number' ? Number(rawHierarchyIndex) : 0;
    setHierarchyIndex(normalized[storedIndex] ? storedIndex : 0);
  }, [rawHierarchyData, rawHierarchyIndex]);

  // Handle screenshot availability
  useEffect(() => {
    if (screenshot) {
      setIsPreviewOpen(true);
    } else {
      setPickerActive(false);
      setAiSuggestions([]);
      setShowAiPanel(false);
    }
  }, [screenshot]);

  // Clear hovered suggestion when AI panel closes
  useEffect(() => {
    if (!showAiPanel) {
      setHoveredSuggestion(null);
    }
  }, [showAiPanel]);

  // Handle element selection (from picker or AI suggestions)
  const handleElementSelection = useCallback(
    (
      nextSelector: string,
      elementInfo: ElementInfo,
      options?: { hierarchy?: ElementHierarchyEntry[]; hierarchyIndex?: number },
    ) => {
      selector.setValue(nextSelector);
      updateParams({ selector: nextSelector });

      const dataPayload: Record<string, unknown> = { elementInfo };

      if (options && 'hierarchy' in options) {
        const normalized = normalizeHierarchy(options.hierarchy);
        if (normalized.length > 0) {
          const targetIndex =
            typeof options.hierarchyIndex === 'number' && normalized[options.hierarchyIndex]
              ? options.hierarchyIndex
              : 0;
          setHierarchyCandidates(normalized);
          setHierarchyIndex(targetIndex);
          dataPayload.elementHierarchy = normalized;
          dataPayload.elementHierarchyIndex = targetIndex;
        } else {
          setHierarchyCandidates([]);
          setHierarchyIndex(-1);
          dataPayload.elementHierarchy = undefined;
          dataPayload.elementHierarchyIndex = undefined;
        }
      }

      updateData(dataPayload);
    },
    [selector, updateData, updateParams],
  );

  // Get bounding box for highlight overlay
  const elementInfo = getValue<{ boundingBox?: BoundingBox; bounding_box?: BoundingBox }>(
    'elementInfo',
  );
  const boundingBox = useMemo(() => {
    if (!elementInfo) {
      return null;
    }
    const box = elementInfo.boundingBox ?? elementInfo.bounding_box;
    if (!box) {
      return null;
    }
    const { width, height } = box;
    if (typeof width !== 'number' || typeof height !== 'number' || width <= 0 || height <= 0) {
      return null;
    }
    return box;
  }, [elementInfo]);

  // Handle hierarchy navigation (parent/child)
  const handleHierarchyShift = useCallback(
    (delta: number) => {
      const hasSelection =
        hierarchyCandidates.length > 0 &&
        hierarchyIndex >= 0 &&
        hierarchyIndex < hierarchyCandidates.length;
      if (!hasSelection) {
        return;
      }

      const nextIndex = Math.max(
        0,
        Math.min(hierarchyIndex + delta, hierarchyCandidates.length - 1),
      );
      if (nextIndex === hierarchyIndex) {
        return;
      }

      const entry = hierarchyCandidates[nextIndex];
      if (!entry) {
        return;
      }

      const nextSelector = deriveSelector(entry);
      if (!nextSelector) {
        toast.error('No selector available for that element');
        return;
      }

      handleElementSelection(nextSelector, entry.element, {
        hierarchy: hierarchyCandidates,
        hierarchyIndex: nextIndex,
      });
    },
    [handleElementSelection, hierarchyCandidates, hierarchyIndex],
  );

  // Toggle picker mode
  const togglePicker = useCallback(() => {
    if (!screenshot) {
      toast.error('Connect a screenshot-producing node before picking elements');
      return;
    }
    if (!effectiveUrl) {
      toast.error('Set a page URL before picking elements');
      return;
    }
    setPickerActive((prev) => !prev);
    setIsPreviewOpen(true);
  }, [effectiveUrl, screenshot]);

  // Handle picker click on screenshot
  const handlePickerClick = useCallback(
    async (x: number, y: number) => {
      if (!effectiveUrl) {
        return;
      }

      setIsSelecting(true);

      try {
        const config = await getConfig();
        const response = await fetch(`${config.API_URL}/element-at-coordinate`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ url: effectiveUrl, x, y }),
        });

        if (!response.ok) {
          const message = await response.text();
          throw new Error(message || 'Failed to locate element');
        }

        const payload = (await response.json()) as ElementCoordinateResponse;
        const candidates = normalizeHierarchy(payload?.candidates ?? []);

        if (candidates.length === 0) {
          toast.error('No selector found at that position');
          return;
        }

        const preferredIndex =
          Number.isInteger(payload?.selectedIndex) &&
          payload.selectedIndex >= 0 &&
          payload.selectedIndex < candidates.length
            ? payload.selectedIndex
            : 0;
        const chosen = candidates[preferredIndex] ?? candidates[0];
        const bestSelector = deriveSelector(chosen);

        if (!bestSelector) {
          toast.error('No selector found at that position');
          return;
        }

        handleElementSelection(bestSelector, chosen.element, {
          hierarchy: candidates,
          hierarchyIndex: preferredIndex,
        });
        toast.success('Element selector updated');
        setPickerActive(false);
      } catch (error) {
        logger.error(
          'Failed to pick element from screenshot',
          {
            component: 'ClickNode',
            action: 'handlePickerClick',
            nodeId: id,
            url: effectiveUrl ?? upstreamUrl ?? null,
          },
          error,
        );
        toast.error('Failed to pick element from screenshot');
      } finally {
        setIsSelecting(false);
      }
    },
    [effectiveUrl, handleElementSelection, id, upstreamUrl],
  );

  // Handle AI suggestion selection
  const handleAiSelectSuggestion = useCallback(
    (selectorValue: string, elementInfo: ElementInfo) => {
      handleElementSelection(selectorValue, elementInfo);
      setHoveredSuggestion(null);
    },
    [handleElementSelection],
  );

  const canUsePicker = Boolean(screenshot && effectiveUrl);

  const screenshotSourceLabel = useMemo(() => {
    if (!upstreamScreenshot) {
      return null;
    }
    if (upstreamScreenshot.nodeType === 'navigate') {
      return 'Preview from Navigate node';
    }
    return `Preview from ${upstreamScreenshot.nodeType} node`;
  }, [upstreamScreenshot]);

  const resilience = useResiliencePanelProps(id);

  // Get hovered bounding box for highlight
  const hoveredBoundingBox = useMemo(() => {
    if (!hoveredSuggestion) {
      return null;
    }
    return hoveredSuggestion.boundingBox ?? hoveredSuggestion.bounding_box ?? null;
  }, [hoveredSuggestion]);

  return (
    <BaseNode
      selected={selected}
      icon={MousePointer}
      iconClassName="text-green-400"
      title="Click"
      className="w-80"
    >
      <NodeUrlField nodeId={id} errorMessage="Provide a URL to target this click." />

      {/* Selector input with picker button */}
      <div className="flex items-center gap-1 mb-2">
        <input
          type="text"
          placeholder="CSS Selector..."
          className="flex-1 px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
          value={selector.value}
          onChange={textInputHandler(selector.setValue)}
          onBlur={selector.commit}
        />
        <div
          className="inline-block"
          title={!canUsePicker ? 'Provide a screenshot and page URL to pick elements' : ''}
        >
          <button
            onClick={togglePicker}
            className={`p-1.5 rounded border border-gray-700 transition-colors ${
              canUsePicker
                ? pickerActive
                  ? 'bg-green-600/80 text-white'
                  : 'bg-flow-bg hover:bg-gray-700 text-gray-400'
                : 'bg-flow-bg opacity-50 cursor-not-allowed text-gray-600'
            }`}
            title={
              canUsePicker
                ? pickerActive
                  ? 'Cancel picking mode'
                  : 'Pick element from screenshot preview'
                : undefined
            }
            disabled={!canUsePicker}
          >
            <Target size={14} />
          </button>
        </div>
      </div>

      {/* DOM hierarchy navigation */}
      <DOMHierarchyNav
        candidates={hierarchyCandidates}
        selectedIndex={hierarchyIndex}
        onShift={handleHierarchyShift}
      />

      {/* Screenshot preview with picker */}
      <ScreenshotPreview
        screenshot={screenshot}
        isOpen={isPreviewOpen}
        onToggle={() => setIsPreviewOpen((open) => !open)}
        sourceLabel={screenshotSourceLabel}
        capturedAt={upstreamScreenshot?.capturedAt}
        pickerActive={pickerActive}
        isSelecting={isSelecting}
        selectedBoundingBox={boundingBox}
        hoveredBoundingBox={hoveredBoundingBox}
        viewport={executionViewport}
        onPickerClick={handlePickerClick}
        canShowAiSuggestions={Boolean(effectiveUrl)}
        onToggleAiPanel={() => {
          if (!effectiveUrl) {
            toast.error('Set a page URL to use AI suggestions');
            return;
          }
          setShowAiPanel((prev) => !prev);
        }}
      />

      {/* AI suggestions panel */}
      {showAiPanel && (
        <AISuggestionsPanel
          nodeId={id}
          effectiveUrl={effectiveUrl}
          upstreamUrl={upstreamUrl}
          suggestions={aiSuggestions}
          onSuggestionsChange={setAiSuggestions}
          onSelectSuggestion={handleAiSelectSuggestion}
          onHoverSuggestion={setHoveredSuggestion}
          onClose={() => setShowAiPanel(false)}
        />
      )}

      <ResiliencePanel {...resilience} />
    </BaseNode>
  );
};

export default memo(ClickNode);
