import { ChevronRight, ChevronDown, Folder, FolderOpen, FileCode, Plus, Search } from 'lucide-react';
import { useState } from 'react';
import NodePalette from './NodePalette';

interface SidebarProps {
  selectedFolder: string;
  onFolderSelect: (folder: string) => void;
}

interface FolderItem {
  path: string;
  name: string;
  children?: FolderItem[];
  workflows?: Array<{ id: string; name: string }>;
}

const mockFolders: FolderItem[] = [
  {
    path: '/ui-validation',
    name: 'UI Validation',
    children: [
      {
        path: '/ui-validation/checkout',
        name: 'Checkout Flow',
        workflows: [
          { id: '1', name: 'checkout-test' },
          { id: '2', name: 'payment-validation' },
        ],
      },
      {
        path: '/ui-validation/login',
        name: 'Login Tests',
        workflows: [
          { id: '3', name: 'login-flow' },
        ],
      },
    ],
  },
  {
    path: '/data-collection',
    name: 'Data Collection',
    workflows: [
      { id: '4', name: 'competitor-pricing' },
      { id: '5', name: 'news-scraper' },
    ],
  },
  {
    path: '/automation',
    name: 'Automation',
    workflows: [
      { id: '6', name: 'invoice-download' },
    ],
  },
];

function FolderTree({ 
  item, 
  level = 0, 
  selectedFolder, 
  onFolderSelect 
}: { 
  item: FolderItem; 
  level?: number; 
  selectedFolder: string; 
  onFolderSelect: (folder: string) => void;
}) {
  const [expanded, setExpanded] = useState(true);
  const hasChildren = item.children && item.children.length > 0;
  const isSelected = selectedFolder === item.path;

  return (
    <div>
      <div
        className={`folder-item ${isSelected ? 'selected' : ''}`}
        style={{ paddingLeft: `${level * 12 + 8}px` }}
        onClick={() => {
          onFolderSelect(item.path);
          if (hasChildren) setExpanded(!expanded);
        }}
      >
        {hasChildren && (
          expanded ? <ChevronDown size={14} /> : <ChevronRight size={14} />
        )}
        {expanded && hasChildren ? (
          <FolderOpen size={14} className="text-yellow-500" />
        ) : (
          <Folder size={14} className="text-yellow-600" />
        )}
        <span className="text-sm">{item.name}</span>
        {item.workflows && (
          <span className="ml-auto text-xs text-gray-500">
            {item.workflows.length}
          </span>
        )}
      </div>
      
      {expanded && hasChildren && item.children?.map((child) => (
        <FolderTree
          key={child.path}
          item={child}
          level={level + 1}
          selectedFolder={selectedFolder}
          onFolderSelect={onFolderSelect}
        />
      ))}
      
      {expanded && item.workflows?.map((workflow) => (
        <div
          key={workflow.id}
          className="folder-item text-gray-400 hover:text-white"
          style={{ paddingLeft: `${(level + 1) * 12 + 8}px` }}
        >
          <FileCode size={14} />
          <span className="text-xs">{workflow.name}</span>
        </div>
      ))}
    </div>
  );
}

function Sidebar({ selectedFolder, onFolderSelect }: SidebarProps) {
  const [searchQuery, setSearchQuery] = useState('');
  const [activeTab, setActiveTab] = useState<'folders' | 'nodes'>('nodes');

  return (
    <div className="w-64 bg-flow-node border-r border-gray-800 flex flex-col">
      <div className="flex border-b border-gray-800">
        <button
          className={`flex-1 px-4 py-2 text-sm font-medium transition-colors ${
            activeTab === 'nodes' 
              ? 'bg-flow-bg text-white border-b-2 border-flow-accent' 
              : 'text-gray-400 hover:text-white'
          }`}
          onClick={() => setActiveTab('nodes')}
        >
          Node Palette
        </button>
        <button
          className={`flex-1 px-4 py-2 text-sm font-medium transition-colors ${
            activeTab === 'folders' 
              ? 'bg-flow-bg text-white border-b-2 border-flow-accent' 
              : 'text-gray-400 hover:text-white'
          }`}
          onClick={() => setActiveTab('folders')}
        >
          Workflows
        </button>
      </div>
      
      {activeTab === 'folders' ? (
        <>
          <div className="p-3 border-b border-gray-800">
            <div className="relative">
              <Search size={16} className="absolute left-2 top-1/2 -translate-y-1/2 text-gray-500" />
              <input
                type="text"
                placeholder="Search workflows..."
                className="w-full pl-8 pr-3 py-1.5 bg-flow-bg rounded text-sm border border-gray-700 focus:border-flow-accent focus:outline-none"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
              />
            </div>
          </div>
          
          <div className="flex-1 overflow-y-auto p-3">
            <div className="folder-tree">
              {mockFolders.map((folder) => (
                <FolderTree
                  key={folder.path}
                  item={folder}
                  selectedFolder={selectedFolder}
                  onFolderSelect={onFolderSelect}
                />
              ))}
            </div>
            
            <button className="w-full mt-4 p-2 border border-dashed border-gray-700 rounded-lg text-gray-400 hover:border-flow-accent hover:text-white transition-colors flex items-center justify-center gap-2">
              <Plus size={16} />
              <span className="text-sm">New Folder</span>
            </button>
          </div>
        </>
      ) : (
        <NodePalette />
      )}
    </div>
  );
}

export default Sidebar;