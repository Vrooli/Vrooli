import { useState, useEffect } from 'react';
import { Pause, RotateCw, X, Terminal, Image, Clock, CheckCircle, XCircle, Loader } from 'lucide-react';
import { format } from 'date-fns';

interface Screenshot {
  id: string;
  timestamp: Date;
  url: string;
  stepName: string;
}

interface LogEntry {
  id: string;
  timestamp: Date;
  level: 'info' | 'warning' | 'error' | 'success';
  message: string;
}

interface ExecutionProps {
  execution: {
    id: string;
    workflowId: string;
    status: 'pending' | 'running' | 'completed' | 'failed';
    startedAt: Date;
    completedAt?: Date;
    screenshots: Screenshot[];
    logs: LogEntry[];
    currentStep?: string;
    progress: number;
  };
}

function ExecutionViewer({ execution }: ExecutionProps) {
  const [activeTab, setActiveTab] = useState<'screenshots' | 'logs'>('screenshots');
  const [selectedScreenshot, setSelectedScreenshot] = useState<Screenshot | null>(null);
  const [autoScroll, setAutoScroll] = useState(true);

  useEffect(() => {
    if (execution.screenshots.length > 0 && !selectedScreenshot) {
      setSelectedScreenshot(execution.screenshots[execution.screenshots.length - 1]);
    }
  }, [execution.screenshots, selectedScreenshot]);

  const getStatusIcon = () => {
    switch (execution.status) {
      case 'running':
        return <Loader size={16} className="animate-spin text-blue-400" />;
      case 'completed':
        return <CheckCircle size={16} className="text-green-400" />;
      case 'failed':
        return <XCircle size={16} className="text-red-400" />;
      default:
        return <Clock size={16} className="text-gray-400" />;
    }
  };

  const getLogColor = (level: LogEntry['level']) => {
    switch (level) {
      case 'error': return 'text-red-400';
      case 'warning': return 'text-yellow-400';
      case 'success': return 'text-green-400';
      default: return 'text-gray-300';
    }
  };

  return (
    <div className="h-full flex flex-col bg-flow-node">
      <div className="flex items-center justify-between p-3 border-b border-gray-800">
        <div className="flex items-center gap-3">
          {getStatusIcon()}
          <div>
            <div className="text-sm font-medium text-white">
              Execution #{execution.id.slice(0, 8)}
            </div>
            <div className="text-xs text-gray-500">
              {execution.currentStep || 'Initializing...'}
            </div>
          </div>
        </div>
        
        <div className="flex items-center gap-2">
          <button className="toolbar-button p-1.5" title="Pause">
            <Pause size={14} />
          </button>
          <button className="toolbar-button p-1.5" title="Restart">
            <RotateCw size={14} />
          </button>
          <button className="toolbar-button p-1.5 text-red-400" title="Stop">
            <X size={14} />
          </button>
        </div>
      </div>
      
      <div className="h-2 bg-flow-bg">
        <div 
          className="h-full bg-flow-accent transition-all duration-300"
          style={{ width: `${execution.progress}%` }}
        />
      </div>
      
      <div className="flex border-b border-gray-800">
        <button
          className={`flex-1 px-3 py-2 text-sm font-medium transition-colors flex items-center justify-center gap-2 ${
            activeTab === 'screenshots' 
              ? 'bg-flow-bg text-white border-b-2 border-flow-accent' 
              : 'text-gray-400 hover:text-white'
          }`}
          onClick={() => setActiveTab('screenshots')}
        >
          <Image size={14} />
          Screenshots ({execution.screenshots.length})
        </button>
        <button
          className={`flex-1 px-3 py-2 text-sm font-medium transition-colors flex items-center justify-center gap-2 ${
            activeTab === 'logs' 
              ? 'bg-flow-bg text-white border-b-2 border-flow-accent' 
              : 'text-gray-400 hover:text-white'
          }`}
          onClick={() => setActiveTab('logs')}
        >
          <Terminal size={14} />
          Logs ({execution.logs.length})
        </button>
      </div>
      
      <div className="flex-1 overflow-hidden flex flex-col">
        {activeTab === 'screenshots' ? (
          <>
            {selectedScreenshot && (
              <div className="flex-1 p-3 overflow-auto">
                <div className="screenshot-viewer">
                  <div className="bg-gray-800 px-3 py-2 flex items-center justify-between">
                    <span className="text-xs text-gray-400">
                      {selectedScreenshot.stepName}
                    </span>
                    <span className="text-xs text-gray-500">
                      {format(selectedScreenshot.timestamp, 'HH:mm:ss.SSS')}
                    </span>
                  </div>
                  <img 
                    src={selectedScreenshot.url} 
                    alt={selectedScreenshot.stepName}
                    className="w-full"
                  />
                </div>
              </div>
            )}
            
            <div className="border-t border-gray-800 p-2 overflow-x-auto">
              <div className="flex gap-2">
                {execution.screenshots.map((screenshot) => (
                  <div
                    key={screenshot.id}
                    onClick={() => setSelectedScreenshot(screenshot)}
                    className={`flex-shrink-0 w-20 h-20 rounded overflow-hidden cursor-pointer border-2 transition-all ${
                      selectedScreenshot?.id === screenshot.id 
                        ? 'border-flow-accent' 
                        : 'border-gray-700 hover:border-gray-600'
                    }`}
                  >
                    <img 
                      src={screenshot.url} 
                      alt={screenshot.stepName}
                      className="w-full h-full object-cover"
                    />
                  </div>
                ))}
              </div>
            </div>
          </>
        ) : (
          <div className="flex-1 overflow-auto p-3">
            <div className="terminal-output">
              {execution.logs.map((log) => (
                <div key={log.id} className="flex gap-2 mb-1">
                  <span className="text-xs text-gray-600">
                    {format(log.timestamp, 'HH:mm:ss')}
                  </span>
                  <span className={`flex-1 text-xs ${getLogColor(log.level)}`}>
                    {log.message}
                  </span>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

export default ExecutionViewer;