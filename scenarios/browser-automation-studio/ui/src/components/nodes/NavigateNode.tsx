import { memo, FC, useState, useEffect, useCallback, useMemo, useRef } from 'react';
import { logger } from '../../utils/logger';
import { Handle, Position, NodeProps, useReactFlow } from 'reactflow';
import { Globe, Loader, Monitor, FileText, Link2, AppWindow, RefreshCcw, ChevronDown, ChevronRight } from 'lucide-react';
import { getConfig } from '../../config';
import toast from 'react-hot-toast';
import { useScenarioStore } from '../../stores/scenarioStore';
import { useWorkflowStore } from '../../stores/workflowStore';
import type { ExecutionViewportSettings } from '../../stores/workflowStore';

interface ConsoleLog {
  level: string;
  message: string;
  timestamp: string;
}

type DestinationType = 'url' | 'scenario';

const destinationTabs: Array<{ type: DestinationType; label: string; icon: typeof Link2 }> = [
  { type: 'url', label: 'URL', icon: Link2 },
  { type: 'scenario', label: 'Scenario', icon: AppWindow },
];

const MIN_VIEWPORT_DIMENSION = 200;
const MAX_VIEWPORT_DIMENSION = 10000;
const DEFAULT_PREVIEW_VIEWPORT = { width: 1920, height: 1080 } as const;

const clampViewportDimension = (value: number): number => {
  if (!Number.isFinite(value)) {
    return MIN_VIEWPORT_DIMENSION;
  }
  return Math.min(Math.max(Math.round(value), MIN_VIEWPORT_DIMENSION), MAX_VIEWPORT_DIMENSION);
};

const NavigateNode: FC<NodeProps> = ({ data = {}, selected, id }) => {
  const [isPreviewOpen, setIsPreviewOpen] = useState(true);
  const [isFetchingPreview, setIsFetchingPreview] = useState(false);
  const [previewImage, setPreviewImage] = useState<string | null>(null);
  const [consoleLogs, setConsoleLogs] = useState<ConsoleLog[]>([]);
  const [activeTab, setActiveTab] = useState<'screenshot' | 'console'>('screenshot');
  const [previewTargetUrl, setPreviewTargetUrl] = useState<string | null>(null);
  const [previewError, setPreviewError] = useState<string | null>(null);
  const [lastPreviewSignature, setLastPreviewSignature] = useState<string | null>(null);
  const [lastPreviewAt, setLastPreviewAt] = useState<number | null>(null);
  const requestIdRef = useRef(0);

  const { getNodes, setNodes } = useReactFlow();
  const { scenarios: scenarioOptions, isLoading: scenariosLoading, error: scenariosError, fetchScenarios } = useScenarioStore();
  const executionViewport = useWorkflowStore((state) => state.currentWorkflow?.executionViewport as ExecutionViewportSettings | undefined);

  const previewViewport = useMemo(() => {
    if (!executionViewport || !Number.isFinite(executionViewport.width) || !Number.isFinite(executionViewport.height)) {
      return { ...DEFAULT_PREVIEW_VIEWPORT };
    }
    return {
      width: clampViewportDimension(executionViewport.width),
      height: clampViewportDimension(executionViewport.height),
    };
  }, [executionViewport]);

  const scenarioName = useMemo(() => {
    const direct = typeof data.scenario === 'string' ? data.scenario.trim() : '';
    if (direct) {
      return direct;
    }
    return typeof data.scenarioName === 'string' ? data.scenarioName.trim() : '';
  }, [data.scenario, data.scenarioName]);

  const scenarioPath = useMemo(() => (typeof data.scenarioPath === 'string' ? data.scenarioPath : ''), [data.scenarioPath]);

  const destinationType = useMemo<DestinationType>(() => {
    const rawType = typeof data.destinationType === 'string' ? data.destinationType.trim().toLowerCase() : '';
    if (rawType === 'scenario' || rawType === 'url') {
      return rawType as DestinationType;
    }
    return scenarioName ? 'scenario' : 'url';
  }, [data.destinationType, scenarioName]);

  const urlValue = typeof data.url === 'string' ? data.url : '';

  const updateNodeData = useCallback((updates: Record<string, unknown>) => {
    const nodes = getNodes();
    const updatedNodes = nodes.map(node => {
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

  // Pre-fetch scenarios when destinationType is scenario
  useEffect(() => {
    if (destinationType === 'scenario' && scenarioOptions.length === 0 && !scenariosLoading) {
      void fetchScenarios();
    }
  }, [destinationType, scenarioOptions.length, scenariosLoading, fetchScenarios]);

  const resolveScenarioUrl = useCallback(async (options?: { silent?: boolean }): Promise<string | null> => {
    const trimmedScenario = scenarioName.trim();
    if (!trimmedScenario) {
      if (!options?.silent) {
        toast.error('Select which scenario to navigate to first');
      }
      return null;
    }

    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/scenarios/${encodeURIComponent(trimmedScenario)}/port`);
      if (!response.ok) {
        const message = await response.text();
        throw new Error(message || 'Unable to resolve scenario port');
      }

      const info = await response.json();
      const baseUrl: string | undefined = typeof info?.url === 'string' && info.url.trim() !== ''
        ? info.url
        : info?.port
          ? `http://localhost:${info.port}`
          : undefined;

      if (!baseUrl) {
        throw new Error('Scenario port is unavailable');
      }

      const trimmedPath = scenarioPath.trim();
      if (!trimmedPath) {
        return baseUrl;
      }

      try {
        const urlWithPath = new URL(trimmedPath, baseUrl.endsWith('/') ? baseUrl : `${baseUrl}/`);
        return urlWithPath.toString();
      } catch {
        const normalizedBase = baseUrl.replace(/\/+$/, '');
        const normalizedPath = trimmedPath.startsWith('/') ? trimmedPath : `/${trimmedPath}`;
        return `${normalizedBase}${normalizedPath}`;
      }
    } catch (error) {
      logger.error('Failed to resolve scenario URL', { component: 'NavigateNode', action: 'resolveScenarioUrl', scenario: scenarioName }, error);
      const message = error instanceof Error ? error.message : 'Failed to resolve scenario URL';
      if (!options?.silent) {
        toast.error(message);
      }
      return null;
    }
  }, [scenarioName, scenarioPath]);

  const previewSignature = useMemo(() => {
    if (destinationType === 'scenario') {
      const trimmedScenario = scenarioName.trim();
      if (!trimmedScenario) {
        return null;
      }
      const trimmedPath = scenarioPath.trim();
      return `scenario::${trimmedScenario}::${trimmedPath}`;
    }

    const trimmedUrl = urlValue.trim();
    if (trimmedUrl.length < 5) {
      return null;
    }

    return `url::${trimmedUrl}`;
  }, [destinationType, scenarioName, scenarioPath, urlValue]);

  const fetchPreview = useCallback(async (signature: string, options?: { force?: boolean }) => {
    if (!signature) {
      return;
    }

    if (!options?.force && signature === lastPreviewSignature) {
      return;
    }

    const requestId = requestIdRef.current + 1;
    requestIdRef.current = requestId;

    setIsFetchingPreview(true);
    setPreviewError(null);

    try {
      let targetUrl: string | null = null;

      if (signature.startsWith('scenario::')) {
        targetUrl = await resolveScenarioUrl({ silent: !options?.force });
        if (!targetUrl) {
          if (requestId === requestIdRef.current) {
            setPreviewTargetUrl(null);
            setPreviewImage(null);
            setConsoleLogs([]);
            setPreviewError('Select a scenario to preview');
            setActiveTab('screenshot');
          }
          return;
        }
      } else {
        const trimmed = urlValue.trim();
        if (!trimmed) {
          if (requestId === requestIdRef.current) {
            setPreviewTargetUrl(null);
            setPreviewImage(null);
            setConsoleLogs([]);
            setPreviewError('Enter a URL to preview');
            setActiveTab('screenshot');
          }
          return;
        }
        targetUrl = trimmed;
      }

      if (requestId !== requestIdRef.current) {
        return;
      }

      setPreviewTargetUrl(targetUrl);

      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/preview-screenshot`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ url: targetUrl, viewport: previewViewport }),
      });

      if (!response.ok) {
        const errorText = await response.text();
        throw new Error(errorText || 'Failed to take screenshot');
      }

      const payload = await response.json();
      if (requestId !== requestIdRef.current) {
        return;
      }

      if (payload?.screenshot) {
        const logs = Array.isArray(payload.consoleLogs) ? payload.consoleLogs : [];
        setPreviewImage(payload.screenshot);
        setConsoleLogs(logs);
        setActiveTab((prev) => (prev === 'console' && logs.length === 0 ? 'screenshot' : prev));
        setPreviewError(null);
        setLastPreviewSignature(signature);
        setLastPreviewAt(Date.now());
      } else {
        throw new Error('No screenshot data received');
      }
    } catch (error) {
      logger.error('Failed to take screenshot', { component: 'NavigateNode', action: 'fetchPreview', destinationType }, error);
      const errorMessage = error instanceof Error ? error.message : 'Failed to take preview screenshot';

      if (requestId === requestIdRef.current) {
        setPreviewError(errorMessage);
        setPreviewImage(null);
        setConsoleLogs([]);
        setActiveTab('screenshot');
      }

      if (options?.force) {
        toast.error(errorMessage);
      }
    } finally {
      if (requestId === requestIdRef.current) {
        setIsFetchingPreview(false);
      }
    }
  }, [destinationType, lastPreviewSignature, previewViewport, resolveScenarioUrl, urlValue]);

  useEffect(() => {
    if (!previewSignature) {
      setLastPreviewSignature((prev) => (prev !== null ? null : prev));
      return;
    }

    const timeoutId = window.setTimeout(() => {
      void fetchPreview(previewSignature);
    }, 700);

    return () => {
      window.clearTimeout(timeoutId);
    };
  }, [fetchPreview, previewSignature]);

  useEffect(() => {
    if (previewImage && !isPreviewOpen) {
      setIsPreviewOpen(true);
    }
  }, [previewImage, isPreviewOpen]);

  useEffect(() => {
    const currentScreenshot = typeof data?.previewScreenshot === 'string' ? data.previewScreenshot : null;
    const desiredScreenshot = typeof previewImage === 'string' && previewImage.trim().length > 0 ? previewImage : null;

    const storedSource = typeof data?.previewScreenshotSourceUrl === 'string' ? data.previewScreenshotSourceUrl : null;
    const desiredSource = typeof previewTargetUrl === 'string' && previewTargetUrl.trim().length > 0 ? previewTargetUrl : null;

    if (desiredScreenshot && desiredScreenshot !== currentScreenshot) {
      updateNodeData({
        previewScreenshot: desiredScreenshot,
        previewScreenshotCapturedAt: (lastPreviewAt ? new Date(lastPreviewAt) : new Date()).toISOString(),
        previewScreenshotSourceUrl: desiredSource ?? storedSource ?? undefined,
      });
    } else if (!desiredScreenshot && currentScreenshot) {
      updateNodeData({
        previewScreenshot: undefined,
        previewScreenshotCapturedAt: undefined,
        previewScreenshotSourceUrl: undefined,
      });
    } else if (desiredScreenshot && (desiredSource ?? '') !== (storedSource ?? '')) {
      updateNodeData({
        previewScreenshotSourceUrl: desiredSource ?? undefined,
      });
    }
  }, [data?.previewScreenshot, data?.previewScreenshotSourceUrl, lastPreviewAt, previewImage, previewTargetUrl, updateNodeData]);

  const handleUrlChange = useCallback((value: string) => {
    updateNodeData({ url: value, destinationType: 'url' });
  }, [updateNodeData]);

  const handleScenarioChange = useCallback((value: string) => {
    updateNodeData({ scenario: value, scenarioName: value, destinationType: 'scenario' });
  }, [updateNodeData]);

  const handleScenarioPathChange = useCallback((value: string) => {
    updateNodeData({ scenarioPath: value, destinationType: 'scenario' });
  }, [updateNodeData]);

  const handleDestinationChange = useCallback((nextType: DestinationType) => {
    if (nextType === 'scenario') {
      // Clear URL when switching to scenario mode
      updateNodeData({ destinationType: 'scenario', url: '' });
    } else {
      // Clear scenario fields when switching to URL mode
      updateNodeData({ destinationType: 'url', scenario: '', scenarioName: '', scenarioPath: '' });
    }
  }, [updateNodeData]);

  const scenarioSuggestionId = `navigate-scenario-options-${id}`;

  const previewDisplayTarget = useMemo(() => {
    if (previewTargetUrl) {
      return previewTargetUrl;
    }
    if (destinationType === 'scenario') {
      if (!scenarioName) {
        return 'Select a scenario to preview';
      }
      return `${scenarioName}${scenarioPath ? ` → ${scenarioPath}` : ''}`;
    }
    return urlValue || 'No URL provided';
  }, [destinationType, previewTargetUrl, scenarioName, scenarioPath, urlValue]);

  return (
    <>
      <div className={`workflow-node ${selected ? 'selected' : ''} w-80`}>
        <Handle type="target" position={Position.Top} className="node-handle" />

        <div className="flex items-center gap-2 mb-2">
          <Globe size={16} className="text-blue-400" />
          <span className="font-semibold text-sm">Navigate</span>
        </div>

        <div className="flex items-center gap-2 mb-3">
          {destinationTabs.map((option) => {
            const Icon = option.icon;
            const isActive = destinationType === option.type;
            return (
              <button
                key={option.type}
                type="button"
                onClick={() => handleDestinationChange(option.type)}
                className={`flex items-center gap-2 px-3 py-1.5 rounded-md border text-xs font-semibold tracking-wide transition-colors ${
                  isActive
                    ? 'border-flow-accent bg-flow-accent/20 text-white'
                    : 'border-gray-700 text-gray-400 hover:text-white hover:border-flow-accent'
                }`}
                aria-pressed={isActive}
              >
                <Icon size={14} className={isActive ? 'text-flow-accent' : 'text-gray-400'} />
                <span>{option.label}</span>
              </button>
            );
          })}
        </div>

        {destinationType === 'scenario' ? (
          <div className="space-y-2">
            <div className="flex items-center gap-1">
              <input
                type="text"
                list={scenarioSuggestionId}
                placeholder="Scenario name..."
                className="flex-1 px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
                value={scenarioName}
                onChange={(e) => handleScenarioChange(e.target.value)}
              />
              <datalist id={scenarioSuggestionId}>
                {scenarioOptions.map(option => (
                  <option
                    key={option.name}
                    value={option.name}
                    label={option.description}
                  />
                ))}
              </datalist>
              <button
                type="button"
                onClick={() => void fetchScenarios()}
                className="p-1.5 bg-flow-bg hover:bg-gray-700 rounded border border-gray-700 transition-colors"
                title="Refresh scenarios"
                aria-label="Refresh scenarios"
                disabled={scenariosLoading}
              >
                {scenariosLoading ? (
                  <Loader size={14} className="text-gray-400 animate-spin" />
                ) : (
                  <RefreshCcw size={14} className="text-gray-400" />
                )}
              </button>
            </div>
            {scenariosError && (
              <p className="text-[11px] text-red-400">{scenariosError}</p>
            )}
            <input
              type="text"
              placeholder="Optional path (e.g., /dashboard)"
              className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
              value={scenarioPath}
              onChange={(e) => handleScenarioPathChange(e.target.value)}
            />
            <p className="text-[11px] text-gray-500">
              Resolves the scenario&apos;s current UI port at execution time. Provide a path to deep link inside the app.
            </p>
          </div>
        ) : (
          <div className="flex items-center gap-1">
            <input
              type="text"
              placeholder="Enter URL..."
              className="flex-1 px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
              value={urlValue}
              onChange={(e) => handleUrlChange(e.target.value)}
            />
          </div>
        )}

        <div className="mt-3 border-t border-gray-800 pt-3">
          <div className="flex items-center justify-between">
            <button
              type="button"
              onClick={() => setIsPreviewOpen((open) => !open)}
              className="flex items-center gap-2 text-xs font-semibold uppercase tracking-wide text-gray-300 hover:text-white transition-colors"
            >
              {isPreviewOpen ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
              <span>Preview</span>
            </button>
            <div className="flex items-center gap-2">
              {lastPreviewAt && (
                <span className="text-[10px] text-gray-500">
                  Updated {new Date(lastPreviewAt).toLocaleTimeString()}
                </span>
              )}
              <button
                type="button"
                className="p-1.5 rounded border border-gray-700 bg-flow-bg text-gray-400 hover:text-white hover:border-flow-accent transition-colors disabled:opacity-40 disabled:hover:text-gray-400 disabled:hover:border-gray-700"
                onClick={(event) => {
                  event.stopPropagation();
                  if (previewSignature) {
                    void fetchPreview(previewSignature, { force: true });
                  }
                }}
                disabled={isFetchingPreview || !previewSignature}
                title="Refresh preview"
                aria-label="Refresh preview"
              >
                {isFetchingPreview ? (
                  <Loader size={14} className="animate-spin" />
                ) : (
                  <RefreshCcw size={14} />
                )}
              </button>
            </div>
          </div>

          {isPreviewOpen && (
            <div className="mt-3 space-y-3">
              <div className="text-[11px] text-gray-400 break-words">
                {previewDisplayTarget}
              </div>

              <div className="space-y-3">
                <div className="flex border-b border-gray-800">
                  <button
                    type="button"
                    onClick={() => setActiveTab('screenshot')}
                    className={`flex items-center gap-2 px-3 py-2 text-xs font-semibold transition-colors ${
                      activeTab === 'screenshot'
                        ? 'text-blue-400 border-b-2 border-blue-400'
                        : 'text-gray-400 hover:text-white'
                    }`}
                    aria-pressed={activeTab === 'screenshot'}
                  >
                    <Monitor size={14} />
                    Screenshot
                  </button>
                  <button
                    type="button"
                    onClick={() => setActiveTab('console')}
                    className={`flex items-center gap-2 px-3 py-2 text-xs font-semibold transition-colors ${
                      activeTab === 'console'
                        ? 'text-blue-400 border-b-2 border-blue-400'
                        : 'text-gray-400 hover:text-white'
                    }`}
                    aria-pressed={activeTab === 'console'}
                  >
                    <FileText size={14} />
                    Console
                    {consoleLogs.length > 0 && (
                      <span className="px-1 py-0.5 rounded-full bg-blue-600 text-white text-[10px]">
                        {consoleLogs.length}
                      </span>
                    )}
                  </button>
                </div>

                <div className="relative">
                  <div className="h-48 rounded-lg border border-gray-700 bg-gray-900/60 overflow-hidden">
                    {previewError && !isFetchingPreview ? (
                      <div className="flex h-full items-center justify-center px-3 text-center text-xs text-red-200">
                        {previewError}
                      </div>
                    ) : activeTab === 'screenshot' ? (
                      previewImage ? (
                        <img
                          src={previewImage}
                          alt="Preview"
                          className="h-full w-full object-cover"
                          onError={() => {
                            setPreviewError('Failed to load preview image');
                            setPreviewImage(null);
                          }}
                        />
                      ) : (
                        <div className="flex h-full items-center justify-center px-3 text-center text-[11px] text-gray-400">
                          {previewSignature ? 'Waiting for screenshot…' : 'Provide a valid URL or scenario to generate a live preview.'}
                        </div>
                      )
                    ) : consoleLogs.length > 0 ? (
                      <div className="h-full overflow-y-auto p-3 pr-2">
                        {consoleLogs.map((log, index) => (
                          <div
                            key={`${log.timestamp}-${index}`}
                            className={`mb-2 rounded border p-2 text-[11px] font-mono last:mb-0 ${
                              log.level === 'error'
                                ? 'bg-red-900/20 border-red-800 text-red-200'
                                : log.level === 'warn'
                                ? 'bg-yellow-900/20 border-yellow-800 text-yellow-200'
                                : log.level === 'info'
                                ? 'bg-blue-900/20 border-blue-800 text-blue-200'
                                : 'bg-gray-800 border-gray-700 text-gray-200'
                            }`}
                          >
                            <div className="flex items-center gap-2 mb-1">
                              <span className={`px-1 py-0.5 text-[10px] rounded ${
                                log.level === 'error'
                                  ? 'bg-red-800 text-red-200'
                                  : log.level === 'warn'
                                  ? 'bg-yellow-800 text-yellow-200'
                                  : log.level === 'info'
                                  ? 'bg-blue-800 text-blue-200'
                                  : 'bg-gray-700 text-gray-200'
                              }`}>
                                {log.level.toUpperCase()}
                              </span>
                              <span className="text-[10px] text-gray-500">
                                {new Date(log.timestamp).toLocaleTimeString()}
                              </span>
                            </div>
                            <div className="whitespace-pre-wrap break-words">
                              {log.message}
                            </div>
                          </div>
                        ))}
                      </div>
                    ) : isFetchingPreview ? (
                      <div className="h-full overflow-hidden p-3">
                        <div className="space-y-2">
                          {Array.from({ length: 4 }).map((_, idx) => (
                            <div
                              // eslint-disable-next-line react/no-array-index-key
                              key={idx}
                              className="h-6 w-full animate-pulse rounded bg-gray-700/60"
                            />
                          ))}
                        </div>
                      </div>
                    ) : (
                      <div className="flex h-full items-center justify-center px-3 text-center text-[11px] text-gray-500">
                        No console messages captured yet.
                      </div>
                    )}
                  </div>

                  {isFetchingPreview && (
                    <div className="absolute inset-0 flex flex-col items-center justify-center gap-2 bg-black/40 backdrop-blur-sm">
                      <Loader size={20} className="animate-spin text-blue-400" />
                      <span className="text-[11px] text-gray-300">Refreshing preview…</span>
                    </div>
                  )}
                </div>
              </div>
            </div>
          )}
        </div>

        <Handle type="source" position={Position.Bottom} className="node-handle" />
      </div>
      
    </>
  );
};

export default memo(NavigateNode);
