import { ChevronRight, ChevronDown, ChevronLeft, Folder, FolderOpen, FileCode, Plus, Search } from 'lucide-react';
import { logger } from '../utils/logger';
import { useState, useEffect } from 'react';
import NodePalette from './NodePalette';
import { getConfig } from '../config';

interface SidebarProps {
  selectedFolder: string;
  onFolderSelect: (folder: string) => void;
  projectId?: string;
}

interface Workflow {
  id: string;
  name: string;
  folder_path: string;
  description?: string;
}

interface FolderItem {
  path: string;
  name: string;
  children?: FolderItem[];
  workflows?: Array<{ id: string; name: string }>;
}

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

function Sidebar({ selectedFolder, onFolderSelect, projectId }: SidebarProps) {
  const [searchQuery, setSearchQuery] = useState('');
  const [activeTab, setActiveTab] = useState<'folders' | 'nodes'>('nodes');
  const [folderStructure, setFolderStructure] = useState<FolderItem[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [isCollapsed, setIsCollapsed] = useState(false);

  // Fetch real workflows and organize into folder structure
  useEffect(() => {
    if (!projectId) return;
    
    const fetchWorkflows = async () => {
      setIsLoading(true);
      try {
        const config = await getConfig();
        const response = await fetch(`${config.API_URL}/projects/${projectId}/workflows`);
        if (!response.ok) {
          logger.error('Failed to fetch workflows', { component: 'Sidebar', action: 'fetchWorkflows' });
          return;
        }
        const data = await response.json();
        const workflows: Workflow[] = data.workflows || [];
        
        // Build folder structure from workflow paths
        const folderMap = new Map<string, FolderItem>();
        
        workflows.forEach(workflow => {
          const pathParts = workflow.folder_path.split('/').filter(Boolean);
          let currentPath = '';
          let parent: FolderItem | null = null;
          
          pathParts.forEach((part, index) => {
            currentPath += '/' + part;
            
            if (!folderMap.has(currentPath)) {
              const folder: FolderItem = {
                path: currentPath,
                name: part,
                children: [],
                workflows: []
              };
              folderMap.set(currentPath, folder);
              
              if (parent) {
                if (!parent.children) parent.children = [];
                parent.children.push(folder);
              }
            }
            
            parent = folderMap.get(currentPath) || null;
            
            // Add workflow to the deepest folder
            if (index === pathParts.length - 1 && parent) {
              if (!parent.workflows) parent.workflows = [];
              parent.workflows.push({
                id: workflow.id,
                name: workflow.name
              });
            }
          });
        });
        
        // Get root folders
        const rootFolders: FolderItem[] = [];
        folderMap.forEach((folder, path) => {
          if (path.split('/').filter(Boolean).length === 1) {
            rootFolders.push(folder);
          }
        });
        
        // If no folder structure exists, create a default one with all workflows
        if (rootFolders.length === 0 && workflows.length > 0) {
          rootFolders.push({
            path: '/workflows',
            name: 'All Workflows',
            workflows: workflows.map(w => ({ id: w.id, name: w.name }))
          });
        }
        
        setFolderStructure(rootFolders);
      } catch (error) {
        logger.error('Failed to fetch workflows', { component: 'Sidebar', action: 'fetchWorkflows' }, error);
      } finally {
        setIsLoading(false);
      }
    };
    
    fetchWorkflows();
  }, [projectId]);

  return (
    <div className={`bg-flow-node border-r border-gray-800 flex flex-col transition-all duration-200 ${isCollapsed ? 'w-12' : 'w-64'}`}>
      {!isCollapsed && (
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
      )}

      {/* Toggle Button */}
      <button
        onClick={() => setIsCollapsed(!isCollapsed)}
        className={`flex items-center justify-center p-2 text-gray-400 hover:text-white hover:bg-gray-700 transition-colors border-b border-gray-800 ${isCollapsed ? '' : ''}`}
        title={isCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
      >
        {isCollapsed ? <ChevronRight size={20} /> : <ChevronLeft size={20} />}
      </button>

      {!isCollapsed && (
        <>
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
                {isLoading ? (
                  <div className="text-center text-gray-400 text-sm py-4">
                    Loading workflows...
                  </div>
                ) : folderStructure.length === 0 ? (
                  <div className="text-center text-gray-500 text-sm py-4">
                    No workflows found
                  </div>
                ) : (
                  <div className="folder-tree">
                    {folderStructure.map((folder) => (
                      <FolderTree
                        key={folder.path}
                        item={folder}
                        selectedFolder={selectedFolder}
                        onFolderSelect={onFolderSelect}
                      />
                    ))}
                  </div>
                )}

                <button className="w-full mt-4 p-2 border border-dashed border-gray-700 rounded-lg text-gray-400 hover:border-flow-accent hover:text-white transition-colors flex items-center justify-center gap-2">
                  <Plus size={16} />
                  <span className="text-sm">New Folder</span>
                </button>
              </div>
            </>
          ) : (
            <NodePalette />
          )}
        </>
      )}
    </div>
  );
}

export default Sidebar;