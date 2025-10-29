import { useState, useCallback } from 'react';
import { logger } from '../utils/logger';
import { Monitor, Search, ChevronRight, ChevronDown, Code, Loader } from 'lucide-react';
import { getConfig } from '../config';
import toast from 'react-hot-toast';

interface DOMNode {
  tagName: string;
  id: string | null;
  className: string | null;
  text: string | null;
  type: string | null;
  href: string | null;
  ariaLabel: string | null;
  placeholder: string | null;
  value: string | null;
  selector: string;
  children?: DOMNode[];
}

interface ElementInfo {
  text: string;
  tagName: string;
  type: string;
  selectors: Array<{
    selector: string;
    type: string;
    robustness: number;
    fallback: boolean;
  }>;
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

interface BrowserInspectorTabProps {
  url: string;
  onSelectElement: (selector: string, elementInfo: ElementInfo) => void;
}

const BrowserInspectorTab: React.FC<BrowserInspectorTabProps> = ({ url, onSelectElement }) => {
  const [domTree, setDomTree] = useState<DOMNode | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(new Set());
  const [selectedNode, setSelectedNode] = useState<DOMNode | null>(null);
  const [searchQuery, setSearchQuery] = useState('');

  const fetchDOMTree = useCallback(async () => {
    if (!url) return;

    setIsLoading(true);
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/dom-tree`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ url }),
      });

      if (!response.ok) {
        throw new Error(`Failed to fetch DOM tree: ${response.status}`);
      }

      const data = await response.json();
      setDomTree(data);
      
      // Auto-expand first few levels
      const nodesToExpand = new Set<string>();
      const expandLevel = (node: DOMNode, level: number, maxLevel: number) => {
        if (level < maxLevel) {
          nodesToExpand.add(node.selector);
          if (node.children) {
            node.children.forEach(child => expandLevel(child, level + 1, maxLevel));
          }
        }
      };
      if (data) {
        expandLevel(data, 0, 2);
      }
      setExpandedNodes(nodesToExpand);
    } catch (error) {
      logger.error('Failed to fetch DOM tree', { component: 'BrowserInspectorTab', action: 'fetchDOMTree' }, error);
      toast.error('Failed to fetch DOM tree');
    } finally {
      setIsLoading(false);
    }
  }, [url]);

  const toggleNode = (selector: string) => {
    setExpandedNodes(prev => {
      const newSet = new Set(prev);
      if (newSet.has(selector)) {
        newSet.delete(selector);
      } else {
        newSet.add(selector);
      }
      return newSet;
    });
  };

  const selectNode = (node: DOMNode) => {
    setSelectedNode(node);
    
    // Create element info for the callback
    const elementInfo: ElementInfo = {
      text: node.text || '',
      tagName: node.tagName,
      type: node.type || 'element',
      selectors: [{
        selector: node.selector,
        type: 'css',
        robustness: node.id ? 1.0 : 0.7,
        fallback: false,
      }],
      boundingBox: { x: 0, y: 0, width: 0, height: 0 },
      confidence: 0.9,
      category: 'general',
      attributes: {
        id: node.id || '',
        class: node.className || '',
        type: node.type || '',
        href: node.href || '',
        'aria-label': node.ariaLabel || '',
        placeholder: node.placeholder || '',
      },
    };
    
    onSelectElement(node.selector, elementInfo);
  };

  const renderNode = (node: DOMNode, depth: number = 0): JSX.Element => {
    // Ensure node has required properties
    if (!node || !node.tagName) {
      return <></>;
    }
    
    const hasChildren = node.children && node.children.length > 0;
    const isExpanded = expandedNodes.has(node.selector);
    const isSelected = selectedNode?.selector === node.selector;
    const isInteractive = ['BUTTON', 'INPUT', 'SELECT', 'TEXTAREA', 'A'].includes(node.tagName);
    
    // Check if node matches search
    const matchesSearch = !searchQuery || 
      node.tagName.toLowerCase().includes(searchQuery.toLowerCase()) ||
      (node.text && node.text.toLowerCase().includes(searchQuery.toLowerCase())) ||
      (node.id && node.id.toLowerCase().includes(searchQuery.toLowerCase())) ||
      (node.selector && node.selector.toLowerCase().includes(searchQuery.toLowerCase()));

    if (!matchesSearch && !hasChildren) {
      return <></>;
    }

    return (
      <div key={node.selector} className="select-none">
        <div 
          className={`flex items-center gap-1 px-2 py-1 hover:bg-gray-700 rounded cursor-pointer ${
            isSelected ? 'bg-blue-900/30 border-l-2 border-blue-400' : ''
          } ${isInteractive ? 'text-blue-300' : 'text-gray-300'}`}
          style={{ paddingLeft: `${depth * 20 + 8}px` }}
          onClick={() => selectNode(node)}
        >
          {hasChildren && (
            <button
              onClick={(e) => {
                e.stopPropagation();
                toggleNode(node.selector);
              }}
              className="p-0.5 hover:bg-gray-600 rounded"
            >
              {isExpanded ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
            </button>
          )}
          {!hasChildren && <span className="w-5" />}
          
          <span className="font-mono text-xs">
            &lt;{node.tagName ? node.tagName.toLowerCase() : 'unknown'}
            {node.id && <span className="text-green-400"> id="{node.id}"</span>}
            {node.className && <span className="text-yellow-400"> class="{node.className.substring(0, 30)}..."</span>}
            &gt;
          </span>
          
          {node.text && (
            <span className="text-xs text-gray-500 ml-2 truncate max-w-[200px]">
              {node.text.substring(0, 50)}
            </span>
          )}
        </div>
        
        {hasChildren && isExpanded && (
          <div>
            {node.children!.map(child => renderNode(child, depth + 1))}
          </div>
        )}
      </div>
    );
  };

  return (
    <div className="p-4">
      <div className="space-y-4">
        <div className="flex gap-2">
          <div className="flex-1 relative">
            <Search size={16} className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400" />
            <input
              type="text"
              placeholder="Search elements by tag, id, class, or text..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-10 pr-3 py-2 bg-flow-bg border border-gray-700 rounded-lg text-sm focus:border-blue-400 focus:outline-none"
            />
          </div>
          <button
            onClick={fetchDOMTree}
            disabled={isLoading || !url}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors flex items-center gap-2"
          >
            {isLoading ? (
              <>
                <Loader size={16} className="animate-spin" />
                Loading DOM...
              </>
            ) : (
              <>
                <Monitor size={16} />
                Fetch DOM Tree
              </>
            )}
          </button>
        </div>

        {domTree ? (
          <div className="bg-flow-bg border border-gray-700 rounded-lg">
            <div className="p-3 border-b border-gray-700">
              <h3 className="text-sm font-medium text-white flex items-center gap-2">
                <Code size={16} />
                DOM Tree Inspector
              </h3>
              <p className="text-xs text-gray-400 mt-1">
                Click on any element to select it. Interactive elements are highlighted in blue.
              </p>
            </div>
            
            <div className="p-2 max-h-96 overflow-y-auto font-mono text-xs">
              {renderNode(domTree)}
            </div>
            
            {selectedNode && (
              <div className="p-3 border-t border-gray-700">
                <h4 className="text-sm font-medium text-white mb-2">Selected Element</h4>
                <div className="space-y-1 text-xs">
                  <p><span className="text-gray-400">Tag:</span> {selectedNode.tagName.toLowerCase()}</p>
                  <p><span className="text-gray-400">Selector:</span> <code className="text-blue-300">{selectedNode.selector}</code></p>
                  {selectedNode.text && (
                    <p><span className="text-gray-400">Text:</span> "{selectedNode.text.substring(0, 100)}"</p>
                  )}
                  {selectedNode.id && (
                    <p><span className="text-gray-400">ID:</span> {selectedNode.id}</p>
                  )}
                  {selectedNode.className && (
                    <p><span className="text-gray-400">Classes:</span> {selectedNode.className}</p>
                  )}
                </div>
              </div>
            )}
          </div>
        ) : (
          <div className="text-center py-8 text-gray-500">
            <Monitor size={32} className="mx-auto mb-2 opacity-50" />
            <p className="text-sm">Click "Fetch DOM Tree" to inspect the page structure</p>
            <p className="text-xs mt-2">This will extract all interactive elements from the page</p>
          </div>
        )}
      </div>
    </div>
  );
};

export default BrowserInspectorTab;