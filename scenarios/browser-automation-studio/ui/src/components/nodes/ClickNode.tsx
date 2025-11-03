import { memo, FC, useState, useEffect, useCallback, useMemo, useRef } from 'react';
import { Handle, Position, NodeProps, useReactFlow } from 'reactflow';
import { MousePointer, Globe, Target, ChevronDown, ChevronRight, Loader2, Sparkles, Wand2, Brain, ArrowUp, ArrowDown } from 'lucide-react';
import toast from 'react-hot-toast';
import { useUpstreamUrl } from '../../hooks/useUpstreamUrl';
import useUpstreamScreenshot from '../../hooks/useUpstreamScreenshot';
import type { ElementInfo, BoundingBox, ElementHierarchyEntry, ElementCoordinateResponse } from '../../types/elements';
import { getConfig } from '../../config';
import { logger } from '../../utils/logger';
import { useWorkflowStore, type ExecutionViewportSettings } from '../../stores/workflowStore';

const DEFAULT_ASPECT_RATIO = '16 / 9';

const normalizeHierarchy = (value: unknown): ElementHierarchyEntry[] => {
  if (!Array.isArray(value)) {
    return [];
  }

  return value
    .map((entry) => {
      if (!entry || typeof entry !== 'object') {
        return null;
      }

      const candidate = entry as Partial<ElementHierarchyEntry> & { element?: ElementInfo };
      if (!candidate.element) {
        return null;
      }

      const selector = typeof candidate.selector === 'string' && candidate.selector.trim().length > 0
        ? candidate.selector.trim()
        : (Array.isArray(candidate.element.selectors) && candidate.element.selectors.length > 0
            ? candidate.element.selectors[0].selector
            : '');

      const depth = Number.isFinite(candidate.depth as number)
        ? Number(candidate.depth)
        : 0;

      const path = Array.isArray(candidate.path)
        ? candidate.path.filter((segment): segment is string => typeof segment === 'string')
        : [];

      const pathSummary = typeof candidate.pathSummary === 'string' && candidate.pathSummary.trim().length > 0
        ? candidate.pathSummary
        : path.join(' > ');

      return {
        element: candidate.element,
        selector,
        depth,
        path,
        pathSummary,
      };
    })
    .filter((entry): entry is ElementHierarchyEntry => Boolean(entry && entry.element));
};

const deriveSelector = (entry: ElementHierarchyEntry | null | undefined): string => {
  if (!entry) {
    return '';
  }
  if (typeof entry.selector === 'string' && entry.selector.trim().length > 0) {
    return entry.selector.trim();
  }
  const selectors = Array.isArray(entry.element?.selectors) ? entry.element.selectors : [];
  return selectors.length > 0 ? selectors[0].selector : '';
};

const summarizeCandidate = (entry: ElementHierarchyEntry | null | undefined): string => {
  if (!entry || !entry.element) {
    return '';
  }
  const tagName = typeof entry.element.tagName === 'string' ? entry.element.tagName.toLowerCase() : '';
  const id = entry.element.attributes?.id ? `#${entry.element.attributes.id}` : '';
  const text = typeof entry.element.text === 'string' ? entry.element.text.trim() : '';
  const textSnippet = text.length > 0 ? ` • ${text.length > 40 ? `${text.slice(0, 40)}…` : text}` : '';
  const base = `${tagName}${id}`.trim();
  return (base + textSnippet).trim() || deriveSelector(entry) || 'element';
};

const stringifyPath = (entry: ElementHierarchyEntry | null | undefined): string => {
  if (!entry) {
    return '';
  }
  if (typeof entry.pathSummary === 'string' && entry.pathSummary.trim().length > 0) {
    return entry.pathSummary;
  }
  return Array.isArray(entry.path) ? entry.path.filter(Boolean).join(' > ') : '';
};

const ClickNode: FC<NodeProps> = ({ data, selected, id }) => {
  const upstreamUrl = useUpstreamUrl(id);
  const upstreamScreenshot = useUpstreamScreenshot(id);
  const screenshot = upstreamScreenshot?.dataUrl ?? null;
  const executionViewport = useWorkflowStore((state) => state.currentWorkflow?.executionViewport as ExecutionViewportSettings | undefined);
  const rawHierarchyData = (data as Record<string, unknown>)?.elementHierarchy;
  const rawHierarchyIndex = (data as Record<string, unknown>)?.elementHierarchyIndex;

  const [hierarchyCandidates, setHierarchyCandidates] = useState<ElementHierarchyEntry[]>(() =>
    normalizeHierarchy(rawHierarchyData)
  );
  const [hierarchyIndex, setHierarchyIndex] = useState<number>(() => {
    const normalized = normalizeHierarchy(rawHierarchyData);
    if (normalized.length === 0) {
      return -1;
    }
    const storedIndex = typeof rawHierarchyIndex === 'number'
      ? Number(rawHierarchyIndex)
      : 0;
    return normalized[storedIndex] ? storedIndex : 0;
  });

  const [isPreviewOpen, setIsPreviewOpen] = useState<boolean>(() => Boolean(screenshot));
  const [pickerActive, setPickerActive] = useState(false);
  const [isSelecting, setIsSelecting] = useState(false);
  const [hoverPosition, setHoverPosition] = useState<{ x: number; y: number } | null>(null);
  const [clickPosition, setClickPosition] = useState<{ x: number; y: number } | null>(null);
  const [selector, setSelector] = useState<string>(data.selector || '');
  const [imageSize, setImageSize] = useState<{ width: number; height: number } | null>(null);
  const [showAiPanel, setShowAiPanel] = useState(false);
  const [aiIntent, setAiIntent] = useState('');
  const [aiSuggestions, setAiSuggestions] = useState<ElementInfo[]>([]);
  const [aiLoading, setAiLoading] = useState(false);
  const [hoveredSuggestion, setHoveredSuggestion] = useState<ElementInfo | null>(null);

  const imageRef = useRef<HTMLImageElement | null>(null);
  const { getNodes, setNodes } = useReactFlow();

  useEffect(() => {
    setSelector(data.selector || '');
  }, [data.selector]);

  useEffect(() => {
    const normalized = normalizeHierarchy(rawHierarchyData);
    setHierarchyCandidates(normalized);

    if (normalized.length === 0) {
      setHierarchyIndex(-1);
      return;
    }

    const storedIndex = typeof rawHierarchyIndex === 'number'
      ? Number(rawHierarchyIndex)
      : 0;
    setHierarchyIndex(normalized[storedIndex] ? storedIndex : 0);
  }, [rawHierarchyData, rawHierarchyIndex]);

  useEffect(() => {
    if (screenshot) {
      setIsPreviewOpen(true);
    } else {
      setPickerActive(false);
      setHoverPosition(null);
      setClickPosition(null);
      setImageSize(null);
      setAiSuggestions([]);
      setShowAiPanel(false);
    }
  }, [screenshot]);

  useEffect(() => {
    if (!showAiPanel) {
      setHoveredSuggestion(null);
    }
  }, [showAiPanel]);

  const updateNodeData = useCallback((updates: Record<string, unknown>) => {
    const nodes = getNodes();
    const updatedNodes = nodes.map((node) => {
      if (node.id !== id) {
        return node;
      }
      return {
        ...node,
        data: {
          ...node.data,
          ...updates,
        },
      };
    });
    setNodes(updatedNodes);
  }, [getNodes, id, setNodes]);

  const handleElementSelection = useCallback((nextSelector: string, elementInfo: ElementInfo, options?: { hierarchy?: ElementHierarchyEntry[]; hierarchyIndex?: number }) => {
    setSelector(nextSelector);

    const payload: Record<string, unknown> = {
      selector: nextSelector,
      elementInfo,
    };

    if (options && 'hierarchy' in options) {
      const normalized = normalizeHierarchy(options.hierarchy);
      if (normalized.length > 0) {
        const targetIndex = typeof options.hierarchyIndex === 'number' && normalized[options.hierarchyIndex]
          ? options.hierarchyIndex
          : 0;
        setHierarchyCandidates(normalized);
        setHierarchyIndex(targetIndex);
        payload.elementHierarchy = normalized;
        payload.elementHierarchyIndex = targetIndex;
      } else {
        setHierarchyCandidates([]);
        setHierarchyIndex(-1);
        payload.elementHierarchy = undefined;
        payload.elementHierarchyIndex = undefined;
      }
    }

    updateNodeData(payload);
  }, [updateNodeData]);

  const elementInfo = (data.elementInfo ?? null) as { boundingBox?: BoundingBox; bounding_box?: BoundingBox } | null;

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

  const highlightStyle = useMemo(() => {
    const source = hoveredSuggestion?.boundingBox ?? hoveredSuggestion?.bounding_box ?? boundingBox;
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
  }, [boundingBox, hoveredSuggestion, imageSize]);

  const previewAspectRatio = useMemo(() => {
    if (executionViewport && Number.isFinite(executionViewport.width) && Number.isFinite(executionViewport.height) && executionViewport.width > 0 && executionViewport.height > 0) {
      return `${executionViewport.width} / ${executionViewport.height}`;
    }
    if (imageSize && imageSize.width > 0 && imageSize.height > 0) {
      return `${imageSize.width} / ${imageSize.height}`;
    }
    return DEFAULT_ASPECT_RATIO;
  }, [executionViewport, imageSize]);

  const previewBoxStyle = useMemo(() => ({
    aspectRatio: previewAspectRatio,
    width: '100%',
    maxHeight: '240px',
  }), [previewAspectRatio]);

  const hasHierarchySelection = hierarchyCandidates.length > 0 && hierarchyIndex >= 0 && hierarchyIndex < hierarchyCandidates.length;
  const selectedHierarchy = hasHierarchySelection ? hierarchyCandidates[hierarchyIndex] : null;
  const canSelectParent = hasHierarchySelection && hierarchyIndex < hierarchyCandidates.length - 1;
  const canSelectChild = hasHierarchySelection && hierarchyIndex > 0;

  const handleHierarchyShift = useCallback((delta: number) => {
    if (!hasHierarchySelection) {
      return;
    }

    const nextIndex = Math.max(0, Math.min(hierarchyIndex + delta, hierarchyCandidates.length - 1));
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

    handleElementSelection(nextSelector, entry.element, { hierarchy: hierarchyCandidates, hierarchyIndex: nextIndex });
  }, [handleElementSelection, hasHierarchySelection, hierarchyCandidates, hierarchyIndex]);

  const togglePicker = useCallback(() => {
    if (!screenshot) {
      toast.error('Connect a screenshot-producing node before picking elements');
      return;
    }
    if (!upstreamUrl) {
      toast.error('Connect to a Navigate node to pick elements from the page');
      return;
    }
    setPickerActive((prev) => !prev);
    setHoverPosition(null);
    setClickPosition(null);
    setIsPreviewOpen(true);
  }, [screenshot, upstreamUrl]);

  const handleScreenshotClick = useCallback(async (event: React.MouseEvent<HTMLImageElement>) => {
    if (!pickerActive || !imageSize || !upstreamUrl) {
      return;
    }

    const img = event.currentTarget;
    const { offsetX, offsetY } = event.nativeEvent;

    const x = Math.round(offsetX * (img.naturalWidth / img.offsetWidth));
    const y = Math.round(offsetY * (img.naturalHeight / img.offsetHeight));

    setClickPosition({ x, y });
    setIsSelecting(true);

    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/element-at-coordinate`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ url: upstreamUrl, x, y }),
      });

      if (!response.ok) {
        const message = await response.text();
        throw new Error(message || 'Failed to locate element');
      }

      const payload = await response.json() as ElementCoordinateResponse;
      const candidates = normalizeHierarchy(payload?.candidates ?? []);

      if (candidates.length === 0) {
        toast.error('No selector found at that position');
        return;
      }

      const preferredIndex = Number.isInteger(payload?.selectedIndex) && payload.selectedIndex >= 0 && payload.selectedIndex < candidates.length
        ? payload.selectedIndex
        : 0;
      const chosen = candidates[preferredIndex] ?? candidates[0];
      const bestSelector = deriveSelector(chosen);

      if (!bestSelector) {
        toast.error('No selector found at that position');
        return;
      }

      handleElementSelection(bestSelector, chosen.element, { hierarchy: candidates, hierarchyIndex: preferredIndex });
      toast.success('Element selector updated');
      setPickerActive(false);
    } catch (error) {
      logger.error('Failed to pick element from screenshot', { component: 'ClickNode', action: 'handleScreenshotClick', nodeId: id, url: upstreamUrl }, error);
      toast.error('Failed to pick element from screenshot');
    } finally {
      setIsSelecting(false);
    }
  }, [handleElementSelection, id, imageSize, pickerActive, upstreamUrl]);

  const handleScreenshotMouseMove = useCallback((event: React.MouseEvent<HTMLImageElement>) => {
    if (!pickerActive || !imageSize) {
      return;
    }
    const img = event.currentTarget;
    const { offsetX, offsetY } = event.nativeEvent;
    const x = Math.round(offsetX * (img.naturalWidth / img.offsetWidth));
    const y = Math.round(offsetY * (img.naturalHeight / img.offsetHeight));
    setHoverPosition({ x, y });
  }, [imageSize, pickerActive]);

  const handleScreenshotMouseLeave = useCallback(() => {
    setHoverPosition(null);
  }, []);

  const handleImageLoad = useCallback((event: React.SyntheticEvent<HTMLImageElement>) => {
    const img = event.currentTarget;
    if (img?.naturalWidth && img?.naturalHeight) {
      setImageSize({ width: img.naturalWidth, height: img.naturalHeight });
    }
  }, []);

  const runAiSuggestions = useCallback(async () => {
    if (!upstreamUrl) {
      toast.error('Connect to a Navigate node before requesting AI suggestions');
      return;
    }

    const trimmedIntent = aiIntent.trim();
    if (!trimmedIntent) {
      toast.error('Describe the element you want to click first');
      return;
    }

    setAiLoading(true);
    setHoveredSuggestion(null);

    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/ai-analyze-elements`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ url: upstreamUrl, intent: trimmedIntent }),
      });

      if (!response.ok) {
        const message = await response.text();
        throw new Error(message || 'Failed to analyze page');
      }

      const suggestions: ElementInfo[] = await response.json();
      const normalized = Array.isArray(suggestions) ? suggestions : [];
      setAiSuggestions(normalized);

      if (normalized.length === 0) {
        toast('AI could not find matching elements. Try refining the description.', { icon: '🤔' });
      } else {
        toast.success(`Found ${normalized.length} suggestion${normalized.length > 1 ? 's' : ''}`);
      }
    } catch (error) {
      logger.error('Failed to fetch AI suggestions', { component: 'ClickNode', action: 'runAiSuggestions', nodeId: id, url: upstreamUrl }, error);
      const message = error instanceof Error ? error.message : 'AI suggestions failed';
      toast.error(message);
    } finally {
      setAiLoading(false);
    }
  }, [aiIntent, id, upstreamUrl]);

  const canUsePicker = Boolean(screenshot && upstreamUrl);

  const screenshotSourceLabel = useMemo(() => {
    if (!upstreamScreenshot) {
      return null;
    }
    if (upstreamScreenshot.nodeType === 'navigate') {
      return 'Preview from Navigate node';
    }
    return `Preview from ${upstreamScreenshot.nodeType} node`;
  }, [upstreamScreenshot]);

  return (
    <>
      <div className={`workflow-node ${selected ? 'selected' : ''} w-80`}>
        <Handle type="target" position={Position.Top} className="node-handle" />

        <div className="flex items-center gap-2 mb-2">
          <MousePointer size={16} className="text-green-400" />
          <span className="font-semibold text-sm">Click</span>
        </div>

        {upstreamUrl && (
          <div className="flex items-center gap-1 mb-2 p-1 bg-flow-bg/50 rounded text-xs border border-gray-700">
            <Globe size={12} className="text-blue-400 flex-shrink-0" />
            <span className="text-gray-400 truncate" title={upstreamUrl}>
              {upstreamUrl}
            </span>
          </div>
        )}

        <div className="flex items-center gap-1 mb-2">
          <input
            type="text"
            placeholder="CSS Selector..."
            className="flex-1 px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
            value={selector}
            onChange={(e) => setSelector(e.target.value)}
            onBlur={() => updateNodeData({ selector })}
          />
          <div
            className="inline-block"
            title={!canUsePicker ? 'Connect to a Navigate or Screenshot node with preview to pick elements' : ''}
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
              title={canUsePicker ? (pickerActive ? 'Cancel picking mode' : 'Pick element from screenshot preview') : undefined}
              disabled={!canUsePicker}
            >
              <Target size={14} />
            </button>
          </div>
        </div>

        {hasHierarchySelection && selectedHierarchy && (
          <div className="mb-3 rounded-md border border-gray-800 bg-flow-bg/70 px-2 py-2">
            <div className="flex items-center justify-between gap-2">
              <span className="text-[11px] font-semibold uppercase tracking-wide text-gray-400">DOM context</span>
              <div className="flex items-center gap-1">
                <button
                  type="button"
                  onClick={() => handleHierarchyShift(1)}
                  disabled={!canSelectParent}
                  className={`inline-flex items-center gap-1 rounded border px-2 py-1 text-[11px] transition-colors ${
                    canSelectParent
                      ? 'border-gray-700 text-gray-300 hover:border-flow-accent hover:text-white'
                      : 'border-gray-800 text-gray-600 cursor-not-allowed'
                  }`}
                  title="Select parent element"
                >
                  <ArrowUp size={12} />
                  Parent
                </button>
                <button
                  type="button"
                  onClick={() => handleHierarchyShift(-1)}
                  disabled={!canSelectChild}
                  className={`inline-flex items-center gap-1 rounded border px-2 py-1 text-[11px] transition-colors ${
                    canSelectChild
                      ? 'border-gray-700 text-gray-300 hover:border-flow-accent hover:text-white'
                      : 'border-gray-800 text-gray-600 cursor-not-allowed'
                  }`}
                  title="Select child element"
                >
                  <ArrowDown size={12} />
                  Child
                </button>
              </div>
            </div>
            <div className="mt-1 text-[10px] text-gray-500">
              Depth {hierarchyIndex + 1} / {hierarchyCandidates.length}
            </div>
            <div className="mt-1 text-[12px] text-gray-200 truncate" title={summarizeCandidate(selectedHierarchy)}>
              {summarizeCandidate(selectedHierarchy)}
            </div>
            <div className="mt-1 text-[10px] font-mono text-gray-500 break-words" title={stringifyPath(selectedHierarchy)}>
              {stringifyPath(selectedHierarchy)}
            </div>
          </div>
        )}

        <div className="border border-gray-800 rounded-lg bg-flow-bg/60 mb-3 overflow-hidden">
          <button
            type="button"
            onClick={() => setIsPreviewOpen((open) => !open)}
            className="w-full flex items-center justify-between px-3 py-2 text-xs text-gray-300 border-b border-gray-800 bg-flow-bg/60 hover:bg-flow-bg/80 transition-colors"
          >
            <span className="flex items-center gap-2">
              {isPreviewOpen ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
              Preview
            </span>
            {screenshotSourceLabel && <span className="text-[10px] text-gray-500">{screenshotSourceLabel}</span>}
          </button>

          {isPreviewOpen && (
            screenshot ? (
              <div className="relative w-full overflow-hidden bg-black/40" style={previewBoxStyle}>
                <img
                  ref={imageRef}
                  src={screenshot}
                  alt="Upstream preview"
                  className={`h-full w-full object-contain ${pickerActive ? 'cursor-crosshair' : 'cursor-default'}`}
                  onLoad={handleImageLoad}
                  onClick={handleScreenshotClick}
                  onMouseMove={handleScreenshotMouseMove}
                  onMouseLeave={handleScreenshotMouseLeave}
                />

                {pickerActive && (
                  <div className="absolute inset-0 pointer-events-none">
                    <div className="absolute inset-0 bg-black/25" />
                    <div className="absolute top-2 left-2 bg-black/70 text-[11px] text-white px-2 py-1 rounded">
                      Click anywhere on the screenshot to select an element
                    </div>
                  </div>
                )}

                {hoverPosition && pickerActive && (
                  <div className="absolute top-2 right-2 bg-gray-900/80 text-[11px] text-gray-100 px-2 py-1 rounded font-mono pointer-events-none">
                    ({hoverPosition.x}, {hoverPosition.y})
                  </div>
                )}

                {clickPosition && pickerActive && imageSize && (
                  <div
                    className="absolute w-3 h-3 bg-blue-500 border-2 border-white rounded-full pointer-events-none transform -translate-x-1/2 -translate-y-1/2"
                    style={{
                      left: `${(clickPosition.x / imageSize.width) * 100}%`,
                      top: `${(clickPosition.y / imageSize.height) * 100}%`,
                    }}
                  />
                )}

                {highlightStyle && !pickerActive && (
                  <div
                    className={`absolute rounded-sm pointer-events-none ${
                      hoveredSuggestion ? 'border-2 border-purple-400/80 bg-purple-400/10' : 'border-2 border-emerald-400/90 bg-emerald-400/10'
                    }`}
                    style={highlightStyle}
                  >
                    <div
                      className={`absolute -top-4 left-0 text-black text-[10px] font-medium px-1 rounded ${
                        hoveredSuggestion ? 'bg-purple-500' : 'bg-emerald-500'
                      }`}
                    >
                      {hoveredSuggestion ? 'Suggested element' : 'Selected element'}
                    </div>
                  </div>
                )}

                {isSelecting && (
                  <div className="absolute inset-0 flex items-center justify-center bg-black/40">
                    <Loader2 size={18} className="text-blue-400 animate-spin" />
                  </div>
                )}
              </div>
            ) : (
              <div className="px-3 py-4 text-xs text-gray-500">
                Connect this node to a Navigate node with a captured preview or to a Screenshot node to view the screenshot here.
              </div>
            )
          )}

          {isPreviewOpen && screenshot && (
            <div className="flex items-center justify-between gap-2 px-3 py-2 border-t border-gray-800 bg-flow-bg/70">
              <button
                type="button"
                className={`inline-flex items-center gap-1 text-[11px] transition-colors ${
                  upstreamUrl
                    ? 'text-gray-300 hover:text-white'
                    : 'text-gray-500 cursor-not-allowed'
                }`}
                disabled={!upstreamUrl}
                onClick={() => {
                  if (!upstreamUrl) {
                    toast.error('Connect to a Navigate node to use AI suggestions');
                    return;
                  }
                  setShowAiPanel((prev) => !prev);
                }}
              >
                <Sparkles size={12} className="text-yellow-400" />
                AI suggestions
              </button>
              {upstreamScreenshot?.capturedAt && (
                <span className="text-[10px] text-gray-500">
                  Captured {new Date(upstreamScreenshot.capturedAt).toLocaleTimeString()}
                </span>
              )}
            </div>
          )}
        </div>

        {showAiPanel && (
          <div className="border border-gray-800 rounded-lg bg-flow-bg/60 p-3 space-y-3">
            <div className="flex items-center gap-2 text-xs text-gray-300">
              <Wand2 size={14} className="text-purple-400" />
              <span>Describe what you want to click and let AI propose selectors.</span>
            </div>
            <div className="space-y-2">
              <label className="flex items-center justify-between text-[11px] text-gray-400">
                <span>Description</span>
                <span className="text-gray-600">example: "primary login button"</span>
              </label>
              <textarea
                rows={2}
                className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
                placeholder="What element should be clicked?"
                value={aiIntent}
                onChange={(event) => setAiIntent(event.target.value)}
              />
              <div className="flex items-center justify-end">
                <button
                  type="button"
                  onClick={runAiSuggestions}
                  disabled={aiLoading}
                  className="inline-flex items-center gap-1 px-3 py-1.5 text-xs rounded-md bg-purple-600 text-white hover:bg-purple-500 transition-colors disabled:opacity-60"
                >
                  {aiLoading ? (
                    <>
                      <Loader2 size={12} className="animate-spin" />
                      Analyzing…
                    </>
                  ) : (
                    <>
                      <Brain size={12} />
                      Suggest selectors
                    </>
                  )}
                </button>
              </div>
            </div>

            {aiSuggestions.length > 0 && (
              <div className="space-y-2">
                <p className="text-[11px] text-gray-400">AI suggestions</p>
                <div className="space-y-2 max-h-48 overflow-y-auto pr-1">
                  {aiSuggestions.map((suggestion, index) => {
                    const primarySelector = suggestion.selectors?.[0]?.selector ?? '';
                    const tagName = suggestion.tagName ? suggestion.tagName.toLowerCase() : 'element';
                    const confidence = suggestion.confidence ?? 0;

                    return (
                      <button
                        key={`${primarySelector}-${index}`}
                        type="button"
                        className="w-full text-left border border-gray-700 rounded-md bg-flow-bg/80 hover:border-flow-accent hover:bg-flow-bg/60 transition-colors px-3 py-2 text-xs"
                        onMouseEnter={() => setHoveredSuggestion(suggestion)}
                        onMouseLeave={() => setHoveredSuggestion(null)}
                        onClick={() => {
                          if (!primarySelector) {
                            toast.error('Suggestion does not include a usable selector');
                            return;
                          }
                          handleElementSelection(primarySelector, suggestion);
                          setShowAiPanel(false);
                          setHoveredSuggestion(null);
                        }}
                      >
                        <div className="flex items-center justify-between gap-2">
                          <div className="font-semibold text-gray-100 truncate">{primarySelector || '(no selector)'}</div>
                          <div className="text-[10px] text-gray-400 whitespace-nowrap">{confidence ? `${Math.round(confidence * 100)}% confidence` : 'confidence n/a'}</div>
                        </div>
                        <div className="mt-1 text-[11px] text-gray-400 flex flex-wrap gap-2">
                          <span className="px-1.5 py-0.5 rounded bg-gray-800 text-gray-200">{tagName}</span>
                          {suggestion.text && (
                            <span className="truncate">text: “{suggestion.text.slice(0, 40)}{suggestion.text.length > 40 ? '…' : ''}”</span>
                          )}
                        </div>
                      </button>
                    );
                  })}
                </div>
              </div>
            )}
          </div>
        )}

        <Handle type="source" position={Position.Bottom} className="node-handle" />
      </div>
    </>
  );
};

export default memo(ClickNode);
