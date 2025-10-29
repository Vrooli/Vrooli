import { memo, FC, useState, useEffect } from 'react';
import { logger } from '../../utils/logger';
import { Handle, Position, NodeProps, useReactFlow } from 'reactflow';
import { Globe, Eye, Loader, X, Monitor, FileText } from 'lucide-react';
import { getConfig } from '../../config';
import toast from 'react-hot-toast';

interface ConsoleLog {
  level: string;
  message: string;
  timestamp: string;
}

const NavigateNode: FC<NodeProps> = ({ data, selected, id }) => {
  const [showPreview, setShowPreview] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [previewImage, setPreviewImage] = useState<string | null>(null);
  const [consoleLogs, setConsoleLogs] = useState<ConsoleLog[]>([]);
  const [activeTab, setActiveTab] = useState<'screenshot' | 'console'>('screenshot');
  const [inputValue, setInputValue] = useState(data.url || '');
  const { getNodes, setNodes } = useReactFlow();

  // Sync input value when data.url changes (e.g., from external updates)
  useEffect(() => {
    setInputValue(data.url || '');
  }, [data.url]);

  const handlePreview = async () => {
    const url = data.url;
    if (!url) {
      toast.error('Please enter a URL first');
      return;
    }

    // Basic URL validation - let browserless handle protocol preprocessing
    if (!url.trim()) {
      toast.error('Please enter a valid URL');
      return;
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
        body: JSON.stringify({
          url: url
        }),
      });

      if (!response.ok) {
        const error = await response.text();
        throw new Error(error || 'Failed to take screenshot');
      }

      const data = await response.json();
      if (data.screenshot) {
        setPreviewImage(data.screenshot);
        setConsoleLogs(data.consoleLogs || []);
      } else {
        throw new Error('No screenshot data received');
      }
    } catch (error) {
      logger.error('Failed to take screenshot', { component: 'NavigateNode', action: 'handleTakeScreenshot' }, error);
      const errorMessage = error instanceof Error ? error.message : 'Failed to take preview screenshot';
      
      // Show more specific error messages
      if (errorMessage.includes('not be reachable')) {
        toast.error('URL not accessible from browser automation service. Try using a public URL.');
      } else if (errorMessage.includes('timeout')) {
        toast.error('Screenshot request timed out. The page may be taking too long to load.');
      } else if (errorMessage.includes('connection')) {
        toast.error('Cannot connect to the URL. Please check if it\'s accessible.');
      } else {
        toast.error('Failed to take preview screenshot');
      }
      setShowPreview(false);
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <>
      <div className={`workflow-node ${selected ? 'selected' : ''} w-80`}>
        <Handle type="target" position={Position.Top} className="node-handle" />
        
        <div className="flex items-center gap-2 mb-2">
          <Globe size={16} className="text-blue-400" />
          <span className="font-semibold text-sm">Navigate</span>
        </div>
        
        <div className="flex items-center gap-1">
          <input
            type="text"
            placeholder="Enter URL..."
            className="flex-1 px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
            value={inputValue}
            onChange={(e) => {
              const newUrl = e.target.value;
              setInputValue(newUrl);
              const nodes = getNodes();
              const updatedNodes = nodes.map(node => {
                if (node.id === id) {
                  return {
                    ...node,
                    data: {
                      ...node.data,
                      url: newUrl
                    }
                  };
                }
                return node;
              });
              setNodes(updatedNodes);
            }}
          />
          <button
            onClick={handlePreview}
            className="p-1.5 bg-flow-bg hover:bg-gray-700 rounded border border-gray-700 transition-colors"
            title="Preview URL"
          >
            <Eye size={14} className="text-gray-400" />
          </button>
        </div>
        
        <Handle type="source" position={Position.Bottom} className="node-handle" />
      </div>

      {/* Preview Modal */}
      {showPreview && (
        <div className="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50">
          <div className="bg-flow-node border border-gray-700 rounded-xl shadow-2xl w-full max-w-4xl max-h-[90vh] flex flex-col">
            <div className="flex items-center justify-between p-4 border-b border-gray-700">
              <div className="flex items-center gap-3">
                <Eye size={20} className="text-blue-400" />
                <div>
                  <h2 className="text-lg font-semibold text-white">URL Preview</h2>
                  <p className="text-xs text-gray-400">{data.url}</p>
                </div>
              </div>
              <button
                onClick={() => {
                  setShowPreview(false);
                  setPreviewImage(null);
                  setConsoleLogs([]);
                  setActiveTab('screenshot');
                }}
                className="p-2 text-gray-400 hover:text-white hover:bg-gray-700 rounded-lg transition-colors"
              >
                <X size={18} />
              </button>
            </div>
            
            {/* Tab Navigation */}
            <div className="flex border-b border-gray-700">
              <button
                onClick={() => setActiveTab('screenshot')}
                className={`flex items-center gap-2 px-4 py-3 font-medium text-sm transition-colors ${
                  activeTab === 'screenshot'
                    ? 'border-b-2 border-blue-400 text-blue-400'
                    : 'text-gray-400 hover:text-white'
                }`}
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
                // Console Logs Tab
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
                        key={index}
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