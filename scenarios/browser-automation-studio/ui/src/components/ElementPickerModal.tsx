import { useState, useEffect, useCallback } from 'react';
import { X, Target, Settings, Loader, Eye, Monitor, AlertCircle, Brain } from 'lucide-react';
import { getConfig } from '../config';
import toast from 'react-hot-toast';
import BrowserInspectorTab from './BrowserInspectorTab';

interface SelectorOption {
  selector: string;
  type: string;
  robustness: number;
  fallback: boolean;
}

interface ElementInfo {
  text: string;
  tagName: string;
  type: string;
  selectors: SelectorOption[];
  boundingBox: {
    x: number;
    y: number;
    width: number;
    height: number;
  };
  confidence: number;
  category: string;
  attributes: Record<string, string>;
}



interface ElementPickerModalProps {
  isOpen: boolean;
  onClose: () => void;
  url: string;
  onSelectElement: (selector: string, elementInfo: ElementInfo) => void;
  selectedSelector?: string;
}

type TabType = 'visual' | 'suggestions' | 'browser' | 'custom';

const ElementPickerModal: React.FC<ElementPickerModalProps> = ({
  isOpen,
  onClose,
  url,
  onSelectElement,
  selectedSelector
}) => {
  const [activeTab, setActiveTab] = useState<TabType>('visual');
  const [isLoading, setIsLoading] = useState(false);
  const [screenshot, setScreenshot] = useState<string | null>(null);
  const [customSelector, setCustomSelector] = useState(selectedSelector || '');
  const [selectedElement, setSelectedElement] = useState<ElementInfo | null>(null);
  const [clickPosition, setClickPosition] = useState<{x: number; y: number} | null>(null);
  const [screenshotDimensions, setScreenshotDimensions] = useState<{width: number; height: number} | null>(null);
  const [hoverPosition, setHoverPosition] = useState<{x: number; y: number} | null>(null);
  const [aiSuggestions, setAiSuggestions] = useState<ElementInfo[]>([]);
  const [aiAnalyzing, setAiAnalyzing] = useState(false);
  const [userIntent, setUserIntent] = useState('');

  // Take screenshot when modal opens
  useEffect(() => {
    if (isOpen && url && !screenshot) {
      takeScreenshot();
    }
  }, [isOpen, url]);

  const takeScreenshot = async () => {
    if (!url) return;

    setIsLoading(true);
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/preview-screenshot`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ url }),
      });

      if (!response.ok) {
        throw new Error(`Failed to take screenshot: ${response.status}`);
      }

      const data = await response.json();
      if (data.screenshot) {
        setScreenshot(data.screenshot);
      }
    } catch (error) {
      console.error('Failed to take screenshot:', error);
      toast.error('Failed to take screenshot');
    } finally {
      setIsLoading(false);
    }
  };

  const getElementAtCoordinate = async (x: number, y: number) => {
    if (!url) return;

    setIsLoading(true);
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/element-at-coordinate`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ url, x, y }),
      });

      if (!response.ok) {
        throw new Error(`Failed to get element: ${response.status}`);
      }

      const element: ElementInfo = await response.json();
      setSelectedElement(element);
      // Use the best selector
      const bestSelector = element.selectors?.[0]?.selector || '';
      setCustomSelector(bestSelector);
      setActiveTab('custom');
    } catch (error) {
      console.error('Failed to get element at coordinate:', error);
      toast.error('Failed to get element at coordinate');
    } finally {
      setIsLoading(false);
    }
  };

  const handleScreenshotClick = useCallback((e: React.MouseEvent<HTMLImageElement>) => {
    const img = e.target as HTMLImageElement;
    
    // Calculate coordinates relative to the image's natural size
    const x = Math.round(e.nativeEvent.offsetX * (img.naturalWidth / img.offsetWidth));
    const y = Math.round(e.nativeEvent.offsetY * (img.naturalHeight / img.offsetHeight));
    
    console.log('Click debug:', {
      offsetX: e.nativeEvent.offsetX,
      offsetY: e.nativeEvent.offsetY,
      naturalWidth: img.naturalWidth,
      naturalHeight: img.naturalHeight,
      offsetWidth: img.offsetWidth,
      offsetHeight: img.offsetHeight,
      scaleX: img.naturalWidth / img.offsetWidth,
      scaleY: img.naturalHeight / img.offsetHeight,
      calculatedX: x,
      calculatedY: y
    });
    
    setClickPosition({ x, y });
    getElementAtCoordinate(x, y);
  }, [url]);

  const handleScreenshotMouseMove = useCallback((e: React.MouseEvent<HTMLImageElement>) => {
    const img = e.target as HTMLImageElement;
    const x = Math.round(e.nativeEvent.offsetX * (img.naturalWidth / img.offsetWidth));
    const y = Math.round(e.nativeEvent.offsetY * (img.naturalHeight / img.offsetHeight));
    setHoverPosition({ x, y });
  }, []);

  const handleScreenshotMouseLeave = useCallback(() => {
    setHoverPosition(null);
  }, []);

  const analyzeWithAI = async () => {
    if (!url || !userIntent.trim()) {
      toast.error('Please provide a description of what element you want to select');
      return;
    }

    setAiAnalyzing(true);
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/ai-analyze-elements`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ 
          url, 
          intent: userIntent.trim() 
        }),
      });

      if (!response.ok) {
        throw new Error(`Failed to analyze with AI: ${response.status}`);
      }

      const suggestions: ElementInfo[] = await response.json();
      setAiSuggestions(suggestions);
      
      if (suggestions.length > 0) {
        // Auto-select the top suggestion
        setSelectedElement(suggestions[0]);
        setCustomSelector(suggestions[0].selectors?.[0]?.selector || '');
      }
    } catch (error) {
      console.error('Failed to analyze with AI:', error);
      toast.error('AI analysis failed. Make sure Ollama is running.');
    } finally {
      setAiAnalyzing(false);
    }
  };

  const handleSelectorConfirm = useCallback(() => {
    if (customSelector && selectedElement) {
      onSelectElement(customSelector, selectedElement);
      onClose();
    } else if (customSelector) {
      // Create minimal element info if no element selected
      const elementInfo: ElementInfo = {
        text: '',
        tagName: '',
        type: 'element',
        selectors: [{ selector: customSelector, type: 'custom', robustness: 0.5, fallback: false }],
        boundingBox: { x: 0, y: 0, width: 0, height: 0 },
        confidence: 0.5,
        category: 'general',
        attributes: {}
      };
      onSelectElement(customSelector, elementInfo);
      onClose();
    }
  }, [customSelector, selectedElement, onSelectElement, onClose]);


  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50 p-4">
      <div className="bg-flow-node border border-gray-700 rounded-xl shadow-2xl w-[95vw] max-w-none max-h-[90vh] flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-gray-700">
          <div className="flex items-center gap-3">
            <Target size={20} className="text-blue-400" />
            <div>
              <h2 className="text-lg font-semibold text-white">Element Picker</h2>
              <p className="text-xs text-gray-400">{url}</p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-2 text-gray-400 hover:text-white hover:bg-gray-700 rounded-lg transition-colors"
          >
            <X size={18} />
          </button>
        </div>

        {/* Tab Navigation */}
        <div className="flex border-b border-gray-700 overflow-x-auto scrollbar-hide min-h-[3rem]">
          <button
            onClick={() => setActiveTab('visual')}
            className={`flex items-center gap-2 px-4 py-3 font-medium text-sm transition-colors flex-shrink-0 ${
              activeTab === 'visual'
                ? 'border-b-2 border-blue-400 text-blue-400'
                : 'text-gray-400 hover:text-white'
            }`}
          >
            <Eye size={16} />
            Visual Selection
          </button>
          <button
            onClick={() => setActiveTab('suggestions')}
            className={`flex items-center gap-2 px-4 py-3 font-medium text-sm transition-colors flex-shrink-0 ${
              activeTab === 'suggestions'
                ? 'border-b-2 border-blue-400 text-blue-400'
                : 'text-gray-400 hover:text-white'
            }`}
          >
            <Brain size={16} />
            AI Suggestions
          </button>
          <button
            onClick={() => setActiveTab('browser')}
            className={`flex items-center gap-2 px-4 py-3 font-medium text-sm transition-colors flex-shrink-0 ${
              activeTab === 'browser'
                ? 'border-b-2 border-blue-400 text-blue-400'
                : 'text-gray-400 hover:text-white'
            }`}
          >
            <Monitor size={16} />
            Browser Inspector
          </button>
          <button
            onClick={() => setActiveTab('custom')}
            className={`flex items-center gap-2 px-4 py-3 font-medium text-sm transition-colors flex-shrink-0 ${
              activeTab === 'custom'
                ? 'border-b-2 border-blue-400 text-blue-400'
                : 'text-gray-400 hover:text-white'
            }`}
          >
            <Settings size={16} />
            Custom Selector
            {selectedElement && (
              <span className="px-1.5 py-0.5 bg-green-600 text-white text-xs rounded-full">
                ✓
              </span>
            )}
          </button>
        </div>

        {/* Content Area */}
        <div className="flex-1 overflow-hidden">
          {isLoading ? (
            <div className="flex-1 flex flex-col items-center justify-center h-64">
              <Loader size={32} className="text-blue-400 animate-spin mb-4" />
              <p className="text-gray-400 text-sm">Analyzing page elements...</p>
              <p className="text-gray-500 text-xs mt-2">This may take a moment</p>
            </div>
          ) : (
            <div className="overflow-auto">
              {activeTab === 'visual' && (
                <div className="p-4">
                  {screenshot ? (
                    <div className="space-y-4">
                      <div className="p-3 bg-flow-bg border border-gray-700 rounded-lg">
                        <p className="text-sm text-gray-300">
                          Click anywhere on the screenshot to select an element at that position.
                        </p>
                        {clickPosition && (
                          <p className="text-xs text-gray-500 mt-1">
                            Last click: ({clickPosition.x}, {clickPosition.y})
                          </p>
                        )}
                      </div>
                      
                      <div className="relative w-full flex justify-center">
                        <img
                          src={screenshot}
                          alt="Page Screenshot"
                          className={`w-full h-auto border-2 rounded-lg cursor-crosshair transition-colors ${
                            hoverPosition ? 'border-purple-400' : 'border-gray-700'
                          }`}
                          onClick={handleScreenshotClick}
                          onMouseMove={handleScreenshotMouseMove}
                          onMouseLeave={handleScreenshotMouseLeave}
                          onLoad={(e) => {
                            const img = e.target as HTMLImageElement;
                            if (img && img.offsetWidth > 0 && img.offsetHeight > 0) {
                              setScreenshotDimensions({
                                width: img.naturalWidth,
                                height: img.naturalHeight
                              });
                            }
                          }}
                        />
                        
                        {/* Hover coordinate display */}
                        {hoverPosition && (
                          <div className="absolute top-2 left-2 bg-purple-600 text-white px-2 py-1 rounded text-xs font-mono pointer-events-none">
                            ({hoverPosition.x}, {hoverPosition.y})
                          </div>
                        )}
                        
                        {/* Show click position indicator */}
                        {clickPosition && screenshotDimensions && (
                          <div
                            className="absolute w-3 h-3 bg-red-500 rounded-full border-2 border-white transform -translate-x-1/2 -translate-y-1/2 pointer-events-none"
                            style={{
                              left: `${(clickPosition.x / screenshotDimensions.width) * 100}%`,
                              top: `${(clickPosition.y / screenshotDimensions.height) * 100}%`
                            }}
                          />
                        )}
                      </div>

                      {/* Selected Element Info below screenshot */}
                      {selectedElement && (
                        <div className="p-3 bg-flow-bg border border-gray-700 rounded-lg">
                          <h3 className="text-sm font-medium text-white mb-2">Selected Element</h3>
                          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                            <div className="space-y-2 text-sm">
                              {selectedElement.text && (
                                <p><span className="text-gray-400">Text:</span> "{selectedElement.text}"</p>
                              )}
                              <p><span className="text-gray-400">Tag:</span> {selectedElement.tagName.toLowerCase()}</p>
                              <p><span className="text-gray-400">Type:</span> {selectedElement.type}</p>
                              {clickPosition && (
                                <p><span className="text-gray-400">Position:</span> ({clickPosition.x}, {clickPosition.y})</p>
                              )}
                            </div>
                            
                            <div>
                              <h4 className="text-xs font-medium text-gray-300 mb-2">Available Selectors</h4>
                              <div className="space-y-1 max-h-32 overflow-y-auto">
                                {selectedElement.selectors.map((selector, index) => (
                                  <div
                                    key={index}
                                    className="flex items-center justify-between p-2 bg-gray-800 rounded cursor-pointer hover:bg-gray-700"
                                    onClick={() => {
                                      setCustomSelector(selector.selector);
                                      setActiveTab('custom');
                                    }}
                                  >
                                    <code className="text-xs text-gray-300 truncate">{selector.selector}</code>
                                    <div className="flex items-center gap-2 flex-shrink-0">
                                      <span className={`px-1.5 py-0.5 text-xs rounded ${
                                        selector.robustness > 0.8 ? 'bg-green-900/20 text-green-300' :
                                        selector.robustness > 0.6 ? 'bg-yellow-900/20 text-yellow-300' :
                                        'bg-red-900/20 text-red-300'
                                      }`}>
                                        {Math.round(selector.robustness * 100)}%
                                      </span>
                                      {selector.fallback && (
                                        <div title="Fallback selector">
                                          <AlertCircle size={12} className="text-orange-400" />
                                        </div>
                                      )}
                                    </div>
                                  </div>
                                ))}
                              </div>
                            </div>
                          </div>
                        </div>
                      )}
                    </div>
                  ) : (
                    <div className="flex flex-col items-center justify-center h-64">
                      <Monitor size={32} className="text-gray-500 mb-4" />
                      <p className="text-gray-400">No screenshot available</p>
                    </div>
                  )}
                </div>
              )}

              {activeTab === 'suggestions' && (
                <div className="p-4">
                  <div className="space-y-4">
                    <div>
                      <label className="block text-sm font-medium text-gray-300 mb-2">
                        What element are you looking for?
                      </label>
                      <div className="flex gap-2">
                        <input
                          type="text"
                          placeholder="e.g., 'search button', 'login form', 'submit button'..."
                          value={userIntent}
                          onChange={(e) => setUserIntent(e.target.value)}
                          className="flex-1 px-3 py-2 bg-flow-bg border border-gray-700 rounded-lg text-sm focus:border-blue-400 focus:outline-none"
                          onKeyPress={(e) => e.key === 'Enter' && !aiAnalyzing && analyzeWithAI()}
                        />
                        <button
                          onClick={analyzeWithAI}
                          disabled={aiAnalyzing || !userIntent.trim()}
                          className="px-4 py-2 bg-purple-600 text-white rounded-lg hover:bg-purple-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors flex items-center gap-2"
                        >
                          {aiAnalyzing ? (
                            <>
                              <Loader size={16} className="animate-spin" />
                              Analyzing...
                            </>
                          ) : (
                            <>
                              <Brain size={16} />
                              Analyze
                            </>
                          )}
                        </button>
                      </div>
                      <p className="text-xs text-gray-500 mt-1">
                        Describe what you want to interact with and AI will suggest the best elements
                      </p>
                    </div>

                    {aiSuggestions.length > 0 && (
                      <div className="space-y-2">
                        <h3 className="text-sm font-medium text-white">AI Suggestions</h3>
                        {aiSuggestions.map((suggestion, index) => (
                          <div
                            key={index}
                            className="p-3 bg-flow-bg border border-gray-700 rounded-lg cursor-pointer hover:border-purple-400 transition-colors"
                            onClick={() => {
                              setSelectedElement(suggestion);
                              setCustomSelector(suggestion.selectors?.[0]?.selector || '');
                              setActiveTab('custom');
                            }}
                          >
                            <div className="flex items-center justify-between mb-2">
                              <span className="text-sm font-medium text-purple-300">
                                Suggestion #{index + 1}
                              </span>
                              <span className={`px-2 py-1 text-xs rounded ${
                                suggestion.confidence > 0.8 ? 'bg-green-900/20 text-green-300' :
                                suggestion.confidence > 0.6 ? 'bg-yellow-900/20 text-yellow-300' :
                                'bg-red-900/20 text-red-300'
                              }`}>
                                {Math.round(suggestion.confidence * 100)}% confidence
                              </span>
                            </div>
                            <p className="text-sm text-gray-300 mb-2">
                              {suggestion.text || `${suggestion.tagName.toLowerCase()} element`}
                            </p>
                            <code className="text-xs text-gray-400 bg-gray-800 px-2 py-1 rounded">
                              {suggestion.selectors?.[0]?.selector}
                            </code>
                          </div>
                        ))}
                      </div>
                    )}

                    {aiSuggestions.length === 0 && userIntent && !aiAnalyzing && (
                      <div className="text-center py-8 text-gray-500">
                        <Brain size={32} className="mx-auto mb-2 opacity-50" />
                        <p className="text-sm">No suggestions found. Try a different description.</p>
                      </div>
                    )}
                  </div>
                </div>
              )}

              {activeTab === 'browser' && (
                <BrowserInspectorTab
                  url={url}
                  onSelectElement={(selector, elementInfo) => {
                    setSelectedElement(elementInfo);
                    setCustomSelector(selector);
                    setActiveTab('custom');
                  }}
                />
              )}

              {activeTab === 'custom' && (
                <div className="p-4">
                  <div className="space-y-4">
                    <div>
                      <label className="block text-sm font-medium text-gray-300 mb-2">
                        CSS Selector
                      </label>
                      <input
                        type="text"
                        placeholder="Enter CSS selector..."
                        value={customSelector}
                        onChange={(e) => setCustomSelector(e.target.value)}
                        className="w-full px-3 py-2 bg-flow-bg border border-gray-700 rounded-lg text-sm focus:border-blue-400 focus:outline-none font-mono"
                      />
                    </div>
                    
                    {selectedElement && (
                      <div className="p-3 bg-flow-bg border border-gray-700 rounded-lg">
                        <h3 className="text-sm font-medium text-white mb-2">Selected Element</h3>
                        <div className="space-y-2">
                          {selectedElement.text && (
                            <p className="text-sm">Text: "{selectedElement.text}"</p>
                          )}
                          <p className="text-sm">Type: {selectedElement.type}</p>
                          <p className="text-sm">Tag: {selectedElement.tagName.toLowerCase()}</p>
                          
                          <div className="mt-3">
                            <h4 className="text-xs font-medium text-gray-300 mb-2">Available Selectors</h4>
                            <div className="space-y-1">
                              {selectedElement.selectors.map((selector, index) => (
                                <div
                                  key={index}
                                  className="flex items-center justify-between p-2 bg-gray-800 rounded cursor-pointer hover:bg-gray-700"
                                  onClick={() => setCustomSelector(selector.selector)}
                                >
                                  <code className="text-xs text-gray-300">{selector.selector}</code>
                                  <div className="flex items-center gap-2">
                                    <span className={`px-1.5 py-0.5 text-xs rounded ${
                                      selector.robustness > 0.8 ? 'bg-green-900/20 text-green-300' :
                                      selector.robustness > 0.6 ? 'bg-yellow-900/20 text-yellow-300' :
                                      'bg-red-900/20 text-red-300'
                                    }`}>
                                      {Math.round(selector.robustness * 100)}%
                                    </span>
                                    {selector.fallback && (
                                      <div title="Fallback selector">
                                        <AlertCircle size={12} className="text-orange-400" />
                                      </div>
                                    )}
                                  </div>
                                </div>
                              ))}
                            </div>
                          </div>
                        </div>
                      </div>
                    )}
                  </div>
                </div>
              )}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="border-t border-gray-700 p-4">
          <div className="flex items-center justify-between">
            <div className="text-xs text-gray-500">
              <p>
                Page: {url}
                {selectedElement && ' • Element selected'}
                {clickPosition && ` • Click: (${clickPosition.x}, ${clickPosition.y})`}
              </p>
            </div>
            <div className="flex gap-2">
              <button
                onClick={onClose}
                className="px-4 py-2 text-gray-400 hover:text-white transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleSelectorConfirm}
                disabled={!customSelector}
                className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                Use Selector
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default ElementPickerModal;