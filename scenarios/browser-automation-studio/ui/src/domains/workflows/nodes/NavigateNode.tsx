import { memo, FC, useState, useEffect, useCallback, useMemo, useRef } from 'react';
import { createPortal } from 'react-dom';
import { logger } from '@utils/logger';
import type { NodeProps } from 'reactflow';
import { Globe, Loader, Monitor, FileText, Link2, AppWindow, RefreshCcw, ChevronDown, ChevronRight, X } from 'lucide-react';
import { getConfig } from '@/config';
import toast from 'react-hot-toast';
import { useScenarioStore } from '@stores/scenarioStore';
import { useWorkflowStore } from '@stores/workflowStore';
import type { ExecutionViewportSettings } from '@stores/workflowStore';
import { ResponsiveDialog } from '@shared/layout';
import { useActionParams } from '@hooks/useActionParams';
import { useNodeData } from '@hooks/useNodeData';
import type { NavigateParams } from '@utils/actionBuilder';
import BaseNode from './BaseNode';

interface ConsoleLog {
  level: string;
  message: string;
  timestamp: string;
}

type DestinationType = 'url' | 'scenario';

const destinationTabs: Array<{ type: DestinationType; label: string; icon: typeof Link2 }> = [
  { type: 'url', label: 'URL', icon: Link2 },
  { type: 'scenario', label: 'App', icon: AppWindow },
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

const NavigateNode: FC<NodeProps> = ({ selected, id }) => {
  const nodeRef = useRef<HTMLDivElement>(null);
  const [isPreviewOpen, setIsPreviewOpen] = useState(true);
  const [isFetchingPreview, setIsFetchingPreview] = useState(false);
  const [previewImage, setPreviewImage] = useState<string | null>(null);
  const [consoleLogs, setConsoleLogs] = useState<ConsoleLog[]>([]);
  const [activeTab, setActiveTab] = useState<'screenshot' | 'console'>('screenshot');
  const [previewTargetUrl, setPreviewTargetUrl] = useState<string | null>(null);
  const [previewError, setPreviewError] = useState<string | null>(null);
  const [lastPreviewSignature, setLastPreviewSignature] = useState<string | null>(null);
  const [lastPreviewAt, setLastPreviewAt] = useState<number | null>(null);
  const [isScenarioDialogOpen, setScenarioDialogOpen] = useState(false);
  const [scenarioSearchTerm, setScenarioSearchTerm] = useState('');
  const requestIdRef = useRef(0);
  const lastPreviewViewportRef = useRef<{ width: number; height: number } | null>(null);

  const { getValue, updateData } = useNodeData(id);
  const { scenarios: scenarioOptions, isLoading: scenariosLoading, error: scenariosError, fetchScenarios } = useScenarioStore();
  const executionViewport = useWorkflowStore((state) => state.currentWorkflow?.executionViewport as ExecutionViewportSettings | undefined);

  // V2 Native: Use action params as source of truth
  const { params, updateParams } = useActionParams<NavigateParams>(id);

  const previewViewport = useMemo(() => {
    if (!executionViewport || !Number.isFinite(executionViewport.width) || !Number.isFinite(executionViewport.height)) {
      return { ...DEFAULT_PREVIEW_VIEWPORT };
    }
    return {
      width: clampViewportDimension(executionViewport.width),
      height: clampViewportDimension(executionViewport.height),
    };
  }, [executionViewport]);

  const previewAspectRatio = useMemo(() => {
    if (previewViewport.width > 0 && previewViewport.height > 0) {
      return `${previewViewport.width} / ${previewViewport.height}`;
    }
    return '16 / 9';
  }, [previewViewport.height, previewViewport.width]);

  const previewContainerStyle = useMemo(() => ({
    aspectRatio: previewAspectRatio,
    width: '100%',
    maxHeight: '240px',
  }), [previewAspectRatio]);

  // V2 Native: Read from action params instead of data
  const scenarioName = useMemo(() => {
    const direct = typeof params?.scenario === 'string' ? params.scenario.trim() : '';
    if (direct) {
      return direct;
    }
    return '';
  }, [params?.scenario]);

  const scenarioPath = useMemo(() => (typeof params?.scenarioPath === 'string' ? params.scenarioPath : ''), [params?.scenarioPath]);

  const filteredScenarios = useMemo(() => {
    const term = scenarioSearchTerm.trim().toLowerCase();
    if (!term) {
      return scenarioOptions;
    }
    return scenarioOptions.filter((scenario) => {
      const name = scenario.name?.toLowerCase?.() ?? '';
      const description = scenario.description?.toLowerCase?.() ?? '';
      return name.includes(term) || description.includes(term);
    });
  }, [scenarioOptions, scenarioSearchTerm]);

  // V2 Native: Read destinationType from action params
  const destinationType = useMemo<DestinationType>(() => {
    const rawType = typeof params?.destinationType === 'string' ? params.destinationType.trim().toLowerCase() : '';
    if (rawType === 'scenario' || rawType === 'url') {
      return rawType as DestinationType;
    }
    return scenarioName ? 'scenario' : 'url';
  }, [params?.destinationType, scenarioName]);

  // V2 Native: Read URL from action params
  const urlValue = typeof params?.url === 'string' ? params.url : '';

  // Pre-fetch scenarios when destinationType is scenario
  useEffect(() => {
    if (destinationType === 'scenario' && scenarioOptions.length === 0 && !scenariosLoading) {
      void fetchScenarios();
    }
  }, [destinationType, scenarioOptions.length, scenariosLoading, fetchScenarios]);

  const openScenarioDialog = useCallback(() => {
    if (!isScenarioDialogOpen) {
      if (scenarioOptions.length === 0 && !scenariosLoading) {
        void fetchScenarios();
      }
      setScenarioSearchTerm('');
    }
    setScenarioDialogOpen(true);
  }, [fetchScenarios, isScenarioDialogOpen, scenarioOptions.length, scenariosLoading]);

  const closeScenarioDialog = useCallback(() => {
    setScenarioDialogOpen(false);
  }, []);

  const resolveScenarioUrl = useCallback(async (options?: { silent?: boolean }): Promise<string | null> => {
    const trimmedScenario = scenarioName.trim();
    if (!trimmedScenario) {
      if (!options?.silent) {
        toast.error('Select which app to navigate to first');
      }
      return null;
    }

    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/scenarios/${encodeURIComponent(trimmedScenario)}/port`);
      if (!response.ok) {
        const message = await response.text();
        throw new Error(message || 'Unable to resolve app port');
      }

      const info = await response.json();
      const baseUrl: string | undefined = typeof info?.url === 'string' && info.url.trim() !== ''
        ? info.url
        : info?.port
          ? `http://localhost:${info.port}`
          : undefined;

      if (!baseUrl) {
        throw new Error('App port is unavailable');
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
      logger.error('Failed to resolve app URL', { component: 'NavigateNode', action: 'resolveScenarioUrl', scenario: scenarioName }, error);
      const message = error instanceof Error ? error.message : 'Failed to resolve app URL';
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

    const activeViewport = previewViewport;

    if (!options?.force && signature === lastPreviewSignature) {
      const previousViewport = lastPreviewViewportRef.current;
      if (previousViewport && previousViewport.width === activeViewport.width && previousViewport.height === activeViewport.height) {
        return;
      }
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
            setPreviewError('Select an app to preview');
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
        body: JSON.stringify({ url: targetUrl, viewport: activeViewport }),
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
        lastPreviewViewportRef.current = {
          width: activeViewport.width,
          height: activeViewport.height,
        };
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
        lastPreviewViewportRef.current = {
          width: activeViewport.width,
          height: activeViewport.height,
        };
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

  // Persist preview screenshot to node data
  const storedScreenshot = getValue<string>('previewScreenshot');
  const storedSourceUrl = getValue<string>('previewScreenshotSourceUrl');

  useEffect(() => {
    const desiredScreenshot = typeof previewImage === 'string' && previewImage.trim().length > 0 ? previewImage : null;
    const desiredSource = typeof previewTargetUrl === 'string' && previewTargetUrl.trim().length > 0 ? previewTargetUrl : null;

    if (desiredScreenshot && desiredScreenshot !== storedScreenshot) {
      updateData({
        previewScreenshot: desiredScreenshot,
        previewScreenshotCapturedAt: (lastPreviewAt ? new Date(lastPreviewAt) : new Date()).toISOString(),
        previewScreenshotSourceUrl: desiredSource ?? storedSourceUrl ?? undefined,
      });
    } else if (!desiredScreenshot && storedScreenshot) {
      updateData({
        previewScreenshot: undefined,
        previewScreenshotCapturedAt: undefined,
        previewScreenshotSourceUrl: undefined,
      });
    } else if (desiredScreenshot && (desiredSource ?? '') !== (storedSourceUrl ?? '')) {
      updateData({
        previewScreenshotSourceUrl: desiredSource ?? undefined,
      });
    }
  }, [lastPreviewAt, previewImage, previewTargetUrl, storedScreenshot, storedSourceUrl, updateData]);

  // V2 Native: Update URL in action params
  const handleUrlChange = useCallback((value: string) => {
    updateParams({ url: value, destinationType: 'url' });
  }, [updateParams]);

  // V2 Native: Update scenario in action params
  const handleScenarioChange = useCallback((value: string) => {
    updateParams({ scenario: value, destinationType: 'scenario' });
  }, [updateParams]);

  const handleScenarioSelect = useCallback((value: string) => {
    handleScenarioChange(value);
    setScenarioDialogOpen(false);
    setScenarioSearchTerm('');
  }, [handleScenarioChange]);

  // V2 Native: Update scenarioPath in action params
  const handleScenarioPathChange = useCallback((value: string) => {
    updateParams({ scenarioPath: value, destinationType: 'scenario' });
  }, [updateParams]);

  // V2 Native: Update destinationType and clear other fields in action params
  const handleDestinationChange = useCallback((nextType: DestinationType) => {
    if (nextType === 'scenario') {
      // Clear URL when switching to scenario mode
      updateParams({ destinationType: 'scenario', url: undefined });
      openScenarioDialog();
    } else {
      // Clear scenario fields when switching to URL mode
      updateParams({ destinationType: 'url', scenario: undefined, scenarioPath: undefined });
    }
  }, [openScenarioDialog, updateParams]);

  const previewDisplayTarget = useMemo(() => {
    if (previewTargetUrl) {
      return previewTargetUrl;
    }
    if (destinationType === 'scenario') {
      if (!scenarioName) {
        return 'Select an app to preview';
      }
      return `${scenarioName}${scenarioPath ? ` → ${scenarioPath}` : ''}`;
    }
    return urlValue || 'No URL provided';
  }, [destinationType, previewTargetUrl, scenarioName, scenarioPath, urlValue]);

  const scenarioDialog = (
    <ResponsiveDialog
      isOpen={isScenarioDialogOpen}
      onDismiss={closeScenarioDialog}
      ariaLabel="Select app"
      className="w-full max-w-lg"
    >
      <div className="flex h-full max-h-[80vh] flex-col gap-3">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h2 className="text-base font-semibold text-surface">Select App</h2>
            <p className="text-xs text-gray-400">Browse available apps to navigate during this workflow.</p>
          </div>
          <button
            type="button"
            onClick={closeScenarioDialog}
            className="rounded border border-gray-800 p-1 text-subtle hover:text-surface hover:border-flow-accent transition-colors"
            aria-label="Close app selector"
          >
            <X size={16} />
          </button>
        </div>

        <div className="flex items-center gap-2">
          <input
            type="search"
            autoFocus
            placeholder="Search apps..."
            className="flex-1 rounded border border-gray-800 bg-flow-bg px-3 py-2 text-sm text-surface focus:border-flow-accent focus:outline-none"
            value={scenarioSearchTerm}
            onChange={(event) => setScenarioSearchTerm(event.target.value)}
          />
          <button
            type="button"
            onClick={() => void fetchScenarios()}
            className="rounded border border-gray-800 bg-flow-bg p-2 text-subtle transition-colors hover:border-flow-accent hover:text-surface disabled:opacity-40"
            aria-label="Refresh apps"
            disabled={scenariosLoading}
          >
            {scenariosLoading ? <Loader size={16} className="animate-spin" /> : <RefreshCcw size={16} />}
          </button>
        </div>

        <div className="flex-1 overflow-y-auto rounded border border-gray-800 bg-flow-bg/90">
          {scenariosLoading && scenarioOptions.length === 0 ? (
            <div className="p-4 text-sm text-gray-400">Loading apps…</div>
          ) : filteredScenarios.length === 0 ? (
            <div className="p-4 text-sm text-gray-400">
              {scenarioSearchTerm ? 'No apps match your search.' : 'No apps available. Try refreshing.'}
            </div>
          ) : (
            filteredScenarios.map((option) => {
              const isActive = option.name === scenarioName;
              return (
                <button
                  key={option.name}
                  type="button"
                  onClick={() => handleScenarioSelect(option.name)}
                  className={`block w-full text-left px-3 py-2 transition-colors ${
                    isActive ? 'bg-flow-accent/20 text-surface' : 'text-gray-200 hover:bg-flow-accent/10'
                  }`}
                >
                  <div className="text-sm font-semibold">{option.name}</div>
                  <div className="mt-1 text-xs text-gray-400">
                    {option.description || 'No description provided.'}
                  </div>
                </button>
              );
            })
          )}
        </div>

        {scenariosError && (
          <p className="text-xs text-red-400">{scenariosError}</p>
        )}
      </div>
    </ResponsiveDialog>
  );

  const scenarioDialogContent = typeof document === 'undefined'
    ? scenarioDialog
    : createPortal(scenarioDialog, document.body);

  // Add data-type attribute to React Flow wrapper div for test automation
  useEffect(() => {
    if (nodeRef.current) {
      const reactFlowNode = nodeRef.current.closest('.react-flow__node');
      if (reactFlowNode) {
        reactFlowNode.setAttribute('data-type', 'navigate');
      }
    }
  }, []);

  return (
    <>
      <div ref={nodeRef}>
        <BaseNode selected={selected} icon={Globe} iconClassName="text-blue-400" title="Navigate" className="w-80">
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
                      ? 'border-flow-accent bg-flow-accent/20 text-surface'
                      : 'border-gray-700 text-subtle hover:text-surface hover:border-flow-accent'
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
                  readOnly
                  placeholder="Select an app..."
                  className="flex-1 px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none cursor-pointer"
                  value={scenarioName}
                  onClick={openScenarioDialog}
                  onKeyDown={(event) => {
                    if (event.key === 'Enter' || event.key === ' ') {
                      event.preventDefault();
                      openScenarioDialog();
                    }
                  }}
                  role="combobox"
                  aria-expanded={isScenarioDialogOpen}
                  aria-haspopup="dialog"
                  aria-label="Selected app"
                />
                <button
                  type="button"
                  onClick={openScenarioDialog}
                  className="p-1.5 bg-flow-bg hover:bg-gray-700 rounded border border-gray-700 transition-colors"
                  title="Choose app"
                  aria-label="Choose app"
                >
                  <ChevronDown size={14} className="text-gray-300" />
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
                Resolves the app&apos;s current UI port at execution time. Provide a path to deep link inside the app.
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
                className="flex items-center gap-2 text-xs font-semibold uppercase tracking-wide text-subtle hover:text-surface transition-colors"
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
                  className="p-1.5 rounded border border-gray-700 bg-flow-bg text-subtle hover:text-surface hover:border-flow-accent transition-colors disabled:opacity-40 disabled:hover:text-subtle disabled:hover:border-gray-700"
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
                          : 'text-subtle hover:text-surface'
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
                          : 'text-subtle hover:text-surface'
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
                    <div
                      className="rounded-lg border border-gray-700 bg-gray-900/60 overflow-hidden flex items-center justify-center"
                      style={previewContainerStyle}
                    >
                      {previewError && !isFetchingPreview ? (
                        <div className="flex h-full items-center justify-center px-3 text-center text-xs text-red-200">
                          {previewError}
                        </div>
                      ) : activeTab === 'screenshot' ? (
                        previewImage ? (
                          <img
                            src={previewImage}
                            alt="Preview"
                            className="h-full w-full object-contain"
                            onError={() => {
                              setPreviewError('Failed to load preview image');
                              setPreviewImage(null);
                            }}
                          />
                        ) : (
                          <div className="flex h-full items-center justify-center px-3 text-center text-[11px] text-gray-400">
                            {previewSignature ? 'Waiting for screenshot…' : 'Provide a valid URL or app to generate a live preview.'}
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
                                key={`preview-skeleton-${idx}`}
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
        </BaseNode>
      </div>

      {scenarioDialogContent}
    </>
  );
};

export default memo(NavigateNode);
