import { memo, FC, useState, useEffect, useCallback, useMemo } from 'react';
import { logger } from '../../utils/logger';
import { Handle, Position, NodeProps, useReactFlow } from 'reactflow';
import { Globe, Eye, Loader, X, Monitor, FileText, Link2, AppWindow, RefreshCcw } from 'lucide-react';
import { getConfig } from '../../config';
import toast from 'react-hot-toast';

interface ConsoleLog {
  level: string;
  message: string;
  timestamp: string;
}

interface ScenarioOption {
  name: string;
  description: string;
  status: string;
}

type DestinationType = 'url' | 'scenario';

const destinationTabs: Array<{ type: DestinationType; label: string; icon: typeof Link2 }> = [
  { type: 'url', label: 'URL', icon: Link2 },
  { type: 'scenario', label: 'Scenario', icon: AppWindow },
];

const NavigateNode: FC<NodeProps> = ({ data = {}, selected, id }) => {
  const [showPreview, setShowPreview] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [previewImage, setPreviewImage] = useState<string | null>(null);
  const [consoleLogs, setConsoleLogs] = useState<ConsoleLog[]>([]);
  const [activeTab, setActiveTab] = useState<'screenshot' | 'console'>('screenshot');
  const [previewTargetUrl, setPreviewTargetUrl] = useState<string | null>(null);
  const [scenarioOptions, setScenarioOptions] = useState<ScenarioOption[]>([]);
  const [scenariosLoading, setScenariosLoading] = useState(false);
  const [scenariosError, setScenariosError] = useState<string | null>(null);

  const { getNodes, setNodes } = useReactFlow();

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

  const fetchScenarioOptions = useCallback(async () => {
    if (scenariosLoading) {
      return;
    }

    setScenariosLoading(true);
    setScenariosError(null);

    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/scenarios`);
      if (!response.ok) {
        const message = await response.text();
        throw new Error(message || `Failed to load scenarios (${response.status})`);
      }

      const payload = await response.json();
      const items = Array.isArray(payload?.scenarios) ? payload.scenarios : [];
      const mapped = items
        .map((item: any) => ({
          name: typeof item?.name === 'string' ? item.name : '',
          description: typeof item?.description === 'string' ? item.description : '',
          status: typeof item?.status === 'string' ? item.status : '',
        }))
        .filter(option => option.name);

      setScenarioOptions(mapped);
      if (mapped.length === 0) {
        setScenariosError('No scenarios found. Install or start a scenario, then refresh.');
      }
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Unable to load scenarios';
      setScenariosError(message);
      logger.error('Failed to load scenarios', { component: 'NavigateNode', action: 'fetchScenarioOptions' }, error);
      toast.error('Failed to load scenarios');
    } finally {
      setScenariosLoading(false);
    }
  }, [scenariosLoading]);

  useEffect(() => {
    if (destinationType === 'scenario' && scenarioOptions.length === 0 && !scenariosLoading) {
      void fetchScenarioOptions();
    }
  }, [destinationType, scenarioOptions.length, scenariosLoading, fetchScenarioOptions]);

  const resolveScenarioUrl = useCallback(async (): Promise<string | null> => {
    const trimmedScenario = scenarioName.trim();
    if (!trimmedScenario) {
      toast.error('Select which scenario to navigate to first');
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
      toast.error(message);
      return null;
    }
  }, [scenarioName, scenarioPath]);

  const handlePreview = useCallback(async () => {
    let targetUrl = '';

    if (destinationType === 'scenario') {
      const resolved = await resolveScenarioUrl();
      if (!resolved) {
        return;
      }
      targetUrl = resolved;
    } else {
      const trimmed = urlValue.trim();
      if (!trimmed) {
        toast.error('Please enter a valid URL');
        return;
      }
      targetUrl = trimmed;
    }

    setIsLoading(true);
    setShowPreview(true);

    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/preview-screenshot`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ url: targetUrl }),
      });

      if (!response.ok) {
        const errorText = await response.text();
        throw new Error(errorText || 'Failed to take screenshot');
      }

      const payload = await response.json();
      if (payload?.screenshot) {
        setPreviewImage(payload.screenshot);
        setConsoleLogs(Array.isArray(payload.consoleLogs) ? payload.consoleLogs : []);
        setPreviewTargetUrl(targetUrl);
      } else {
        throw new Error('No screenshot data received');
      }
    } catch (error) {
      logger.error('Failed to take screenshot', { component: 'NavigateNode', action: 'handlePreview', destinationType }, error);
      const errorMessage = error instanceof Error ? error.message : 'Failed to take preview screenshot';

      if (errorMessage.includes('not be reachable')) {
        toast.error('URL not accessible from browser automation service. Try using a public URL.');
      } else if (errorMessage.includes('timeout')) {
        toast.error('Screenshot request timed out. The page may be taking too long to load.');
      } else if (errorMessage.includes('connection')) {
        toast.error('Cannot connect to the target. Please verify accessibility.');
      } else {
        toast.error('Failed to take preview screenshot');
      }

      setShowPreview(false);
      setPreviewTargetUrl(null);
    } finally {
      setIsLoading(false);
    }
  }, [destinationType, resolveScenarioUrl, urlValue]);

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
    updateNodeData({ destinationType: nextType });
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
                onClick={() => void fetchScenarioOptions()}
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
              <button
                type="button"
                onClick={handlePreview}
                className="p-1.5 bg-flow-bg hover:bg-gray-700 rounded border border-gray-700 transition-colors"
                title="Preview scenario"
                aria-label="Preview scenario"
              >
                <Eye size={14} className="text-gray-400" />
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
            <button
              type="button"
              onClick={handlePreview}
              className="p-1.5 bg-flow-bg hover:bg-gray-700 rounded border border-gray-700 transition-colors"
              title="Preview URL"
              aria-label="Preview URL"
            >
              <Eye size={14} className="text-gray-400" />
            </button>
          </div>
        )}

        <Handle type="source" position={Position.Bottom} className="node-handle" />
      </div>

      {showPreview && (
        <div className="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50">
          <div className="bg-flow-node border border-gray-700 rounded-xl shadow-2xl w-full max-w-4xl max-h-[90vh] flex flex-col">
            <div className="flex items-center justify-between p-4 border-b border-gray-700">
              <div className="flex items-center gap-3">
                <Eye size={20} className="text-blue-400" />
                <div>
                  <h2 className="text-lg font-semibold text-white">Target Preview</h2>
                  <p className="text-xs text-gray-400 break-words">{previewDisplayTarget}</p>
                </div>
              </div>
              <button
                onClick={() => {
                  setShowPreview(false);
                  setPreviewImage(null);
                  setConsoleLogs([]);
                  setActiveTab('screenshot');
                  setPreviewTargetUrl(null);
                }}
                className="p-2 text-gray-400 hover:text-white hover:bg-gray-700 rounded-lg transition-colors"
                aria-label="Close preview"
              >
                <X size={18} />
              </button>
            </div>

            <div className="flex border-b border-gray-700">
              <button
                onClick={() => setActiveTab('screenshot')}
                className={`flex items-center gap-2 px-4 py-3 font-medium text-sm transition-colors ${
                  activeTab === 'screenshot'
                    ? 'border-b-2 border-blue-400 text-blue-400'
                    : 'text-gray-400 hover:text-white'
                }`}
                aria-pressed={activeTab === 'screenshot'}
              >
                <Monitor size={16} />
                Screenshot
              </button>
              <button
                onClick={() => setActiveTab('console')}
                className={`flex items-center gap-2 px-4 py-3 font-medium text-sm transition-colors ${
                  activeTab === 'console'
                    ? 'border-b-2 border-blue-400 text-blue-400'
                    : 'text-gray-400 hover:text-white'
                }`}
                aria-pressed={activeTab === 'console'}
              >
                <FileText size={16} />
                Console Logs
                {consoleLogs.length > 0 && (
                  <span className="px-1.5 py-0.5 bg-blue-600 text-white text-xs rounded-full">
                    {consoleLogs.length}
                  </span>
                )}
              </button>
            </div>

            <div className="flex-1 overflow-auto p-4">
              {isLoading ? (
                <div className="flex flex-col items-center justify-center h-64">
                  <Loader size={32} className="text-blue-400 animate-spin mb-4" />
                  <p className="text-gray-400 text-sm">Taking screenshot and capturing console logs...</p>
                  <p className="text-gray-500 text-xs mt-2">This may take a few seconds</p>
                </div>
              ) : activeTab === 'screenshot' ? (
                previewImage ? (
                  <div className="rounded-lg overflow-hidden border border-gray-700">
                    <img
                      src={previewImage}
                      alt="Preview"
                      className="w-full h-auto"
                      onError={() => {
                        toast.error('Failed to load preview image');
                        setPreviewImage(null);
                      }}
                    />
                  </div>
                ) : (
                  <div className="flex flex-col items-center justify-center h-64">
                    <p className="text-gray-400">Failed to load preview</p>
                  </div>
                )
              ) : (
                <div className="space-y-2">
                  {consoleLogs.length === 0 ? (
                    <div className="flex flex-col items-center justify-center h-64">
                      <FileText size={32} className="text-gray-500 mb-4" />
                      <p className="text-gray-400">No console logs captured</p>
                      <p className="text-gray-500 text-xs mt-2">Page may not have any console output</p>
                    </div>
                  ) : (
                    consoleLogs.map((log, index) => (
                      <div
                        key={`${log.timestamp}-${index}`}
                        className={`p-3 rounded-lg border text-sm font-mono ${
                          log.level === 'error'
                            ? 'bg-red-900/20 border-red-800 text-red-300'
                            : log.level === 'warn'
                            ? 'bg-yellow-900/20 border-yellow-800 text-yellow-300'
                            : log.level === 'info'
                            ? 'bg-blue-900/20 border-blue-800 text-blue-300'
                            : 'bg-gray-800 border-gray-700 text-gray-300'
                        }`}
                      >
                        <div className="flex items-center gap-2 mb-1">
                          <span className={`px-1.5 py-0.5 text-xs rounded ${
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
                          <span className="text-xs text-gray-500">
                            {new Date(log.timestamp).toLocaleTimeString()}
                          </span>
                        </div>
                        <div className="whitespace-pre-wrap break-words">
                          {log.message}
                        </div>
                      </div>
                    ))
                  )}
                </div>
              )}
            </div>

            {(previewImage || consoleLogs.length > 0) && !isLoading && (
              <div className="p-4 border-t border-gray-700">
                <div className="flex items-center justify-between">
                  <div className="text-xs text-gray-500">
                    <p>Preview captured at {new Date().toLocaleTimeString()}</p>
                    {consoleLogs.length > 0 && (
                      <p className="mt-1">Console logs: {consoleLogs.length} messages</p>
                    )}
                  </div>
                  <button
                    onClick={() => {
                      setShowPreview(false);
                      setPreviewImage(null);
                      setConsoleLogs([]);
                      setActiveTab('screenshot');
                      setPreviewTargetUrl(null);
                      toast.success('Preview closed');
                    }}
                    className="px-4 py-2 bg-flow-accent text-white rounded-lg text-sm hover:bg-blue-600 transition-colors"
                  >
                    Close Preview
                  </button>
                </div>
              </div>
            )}
          </div>
        </div>
      )}
    </>
  );
};

export default memo(NavigateNode);
