import { useState, useEffect } from 'react';
import { ArrowLeft, Plus, FileCode, Play, Clock, PlayCircle, Loader, WifiOff, Edit2, Check, X, FolderOpen, LayoutGrid, FolderTree, ChevronRight, ChevronDown } from 'lucide-react';
import { Project, useProjectStore } from '../stores/projectStore';
import { getConfig } from '../config';
import toast from 'react-hot-toast';

interface Workflow {
  id: string;
  name: string;
  description?: string;
  folder_path: string;
  created_at: string;
  updated_at: string;
  project_id: string;
  stats?: {
    execution_count: number;
    last_execution?: string;
    success_rate?: number;
  };
}

interface ProjectDetailProps {
  project: Project;
  onBack: () => void;
  onWorkflowSelect: (workflow: Workflow) => void;
  onCreateWorkflow: () => void;
}

interface FolderItem {
  path: string;
  name: string;
  children?: FolderItem[];
  workflows?: Workflow[];
  expanded?: boolean;
}

function ProjectDetail({ project, onBack, onWorkflowSelect, onCreateWorkflow }: ProjectDetailProps) {
  const [workflows, setWorkflows] = useState<Workflow[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [executionInProgress, setExecutionInProgress] = useState<Record<string, boolean>>({});
  const [isEditingPath, setIsEditingPath] = useState(false);
  const [editedPath, setEditedPath] = useState(project.folder_path);
  const [isSavingPath, setIsSavingPath] = useState(false);
  const [viewMode, setViewMode] = useState<'card' | 'tree'>('card');
  const [folderStructure, setFolderStructure] = useState<FolderItem[]>([]);
  const [expandedFolders, setExpandedFolders] = useState<Set<string>>(new Set());
  const { updateProject } = useProjectStore();

  const filteredWorkflows = workflows.filter(workflow =>
    workflow.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
    workflow.description?.toLowerCase().includes(searchTerm.toLowerCase())
  );

  // Build folder structure from workflows
  const buildFolderStructure = (workflowList: Workflow[]): FolderItem[] => {
    const folderMap = new Map<string, FolderItem>();
    
    workflowList.forEach(workflow => {
      const pathParts = workflow.folder_path.split('/').filter(Boolean);
      let currentPath = '';
      let parent: FolderItem | null = null;
      
      pathParts.forEach((part) => {
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
            if (!parent.children.find(c => c.path === currentPath)) {
              parent.children.push(folder);
            }
          }
        }
        
        parent = folderMap.get(currentPath) || null;
      });
      
      // Add workflow to the appropriate folder (only if we have a parent folder)
      if (parent) {
        const parentFolder = parent as FolderItem;
        if (!parentFolder.workflows) {
          parentFolder.workflows = [];
        }
        parentFolder.workflows.push(workflow);
      } else if (pathParts.length === 0) {
        // Workflow is at root level, create a default folder
        const defaultFolder: FolderItem = {
          path: '/workflows',
          name: 'Workflows',
          workflows: [workflow]
        };
        if (!folderMap.has('/workflows')) {
          folderMap.set('/workflows', defaultFolder);
        } else {
          const folder = folderMap.get('/workflows');
          if (folder && folder.workflows) {
            folder.workflows.push(workflow);
          }
        }
      }
    });
    
    // Get root folders
    const rootFolders: FolderItem[] = [];
    folderMap.forEach((folder, path) => {
      if (path.split('/').filter(Boolean).length === 1) {
        rootFolders.push(folder);
      }
    });
    
    // If no folder structure exists, create a default one
    if (rootFolders.length === 0 && workflowList.length > 0) {
      rootFolders.push({
        path: '/workflows',
        name: 'All Workflows',
        workflows: workflowList
      });
    }
    
    return rootFolders;
  };

  const fetchWorkflows = async () => {
    setIsLoading(true);
    setError(null);
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/projects/${project.id}/workflows`);
      if (!response.ok) {
        throw new Error(`Failed to fetch workflows: ${response.status}`);
      }
      const data = await response.json();
      const workflowList = data.workflows || [];
      setWorkflows(workflowList);
      setFolderStructure(buildFolderStructure(workflowList));
    } catch (error) {
      console.error('Failed to fetch workflows:', error);
      setError(error instanceof Error ? error.message : 'Failed to fetch workflows');
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchWorkflows();
  }, [project.id]);

  useEffect(() => {
    setEditedPath(project.folder_path);
  }, [project.folder_path]);

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    });
  };

  const handleExecuteWorkflow = async (e: React.MouseEvent, workflowId: string) => {
    e.stopPropagation(); // Prevent workflow selection
    setExecutionInProgress(prev => ({ ...prev, [workflowId]: true }));
    
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/workflows/${workflowId}/execute`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
      });

      if (!response.ok) {
        throw new Error(`Failed to execute workflow: ${response.status}`);
      }

      const result = await response.json();
      console.log('Workflow execution started:', result);
      // You could show a success message here
    } catch (error) {
      console.error('Failed to execute workflow:', error);
      alert('Failed to execute workflow. Please try again.');
    } finally {
      setExecutionInProgress(prev => ({ ...prev, [workflowId]: false }));
    }
  };

  const handleSavePath = async () => {
    if (editedPath === project.folder_path) {
      setIsEditingPath(false);
      return;
    }

    setIsSavingPath(true);
    try {
      await updateProject(project.id, { folder_path: editedPath });
      toast.success('Project path updated successfully');
      setIsEditingPath(false);
    } catch (error) {
      toast.error('Failed to update project path');
      setEditedPath(project.folder_path); // Reset to original value
    } finally {
      setIsSavingPath(false);
    }
  };

  const handleCancelEdit = () => {
    setEditedPath(project.folder_path);
    setIsEditingPath(false);
  };

  const toggleFolder = (path: string) => {
    const newExpanded = new Set(expandedFolders);
    if (newExpanded.has(path)) {
      newExpanded.delete(path);
    } else {
      newExpanded.add(path);
    }
    setExpandedFolders(newExpanded);
  };

  // Tree View Component
  const FolderTreeItem = ({ item, level = 0 }: { item: FolderItem; level?: number }) => {
    const isExpanded = expandedFolders.has(item.path);
    const hasChildren = (item.children && item.children.length > 0) || (item.workflows && item.workflows.length > 0);
    
    return (
      <div>
        <div
          className="flex items-center gap-2 px-2 py-1.5 hover:bg-flow-node rounded cursor-pointer transition-colors"
          style={{ paddingLeft: `${level * 20 + 8}px` }}
          onClick={() => toggleFolder(item.path)}
        >
          {hasChildren && (
            isExpanded ? <ChevronDown size={14} className="text-gray-400" /> : <ChevronRight size={14} className="text-gray-400" />
          )}
          {!hasChildren && <div className="w-3.5" />}
          <FolderOpen size={14} className="text-yellow-500" />
          <span className="text-sm text-gray-300">{item.name}</span>
          {item.workflows && item.workflows.length > 0 && (
            <span className="ml-auto text-xs text-gray-500 mr-2">
              {item.workflows.length} workflow{item.workflows.length !== 1 ? 's' : ''}
            </span>
          )}
        </div>
        
        {isExpanded && (
          <>
            {item.children?.map((child) => (
              <FolderTreeItem key={child.path} item={child} level={level + 1} />
            ))}
            {item.workflows?.map((workflow) => (
              <div
                key={workflow.id}
                onClick={() => onWorkflowSelect(workflow)}
                className="flex items-center gap-2 px-2 py-1.5 hover:bg-flow-node rounded cursor-pointer transition-colors group"
                style={{ paddingLeft: `${(level + 1) * 20 + 28}px` }}
              >
                <FileCode size={14} className="text-green-400" />
                <span className="text-sm text-gray-400 group-hover:text-white">{workflow.name}</span>
                <div className="ml-auto flex items-center gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
                  <button
                    onClick={(e) => handleExecuteWorkflow(e, workflow.id)}
                    disabled={executionInProgress[workflow.id]}
                    className="p-1.5 text-gray-400 hover:text-white hover:bg-gray-700 rounded transition-colors"
                    title="Execute Workflow"
                  >
                    {executionInProgress[workflow.id] ? (
                      <Loader size={14} className="animate-spin" />
                    ) : (
                      <PlayCircle size={14} />
                    )}
                  </button>
                </div>
              </div>
            ))}
          </>
        )}
      </div>
    );
  };

  // Status bar for API connection issues
  const StatusBar = () => {
    if (!error) return null;
    
    return (
      <div className="bg-red-900/20 border-l-4 border-red-500 p-4 mx-6 mb-4 rounded-r-lg">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <WifiOff size={20} className="text-red-400" />
            <div>
              <div className="text-red-400 font-medium">API Connection Failed</div>
              <div className="text-red-300/80 text-sm">{error}</div>
            </div>
          </div>
          <button
            onClick={() => {
              setError(null);
              fetchWorkflows();
            }}
            className="px-3 py-1 bg-red-500 text-white text-sm rounded-lg hover:bg-red-600 transition-colors"
          >
            Retry
          </button>
        </div>
      </div>
    );
  };

  return (
    <div className="flex-1 flex flex-col h-full overflow-hidden">
      {/* Header */}
      <div className="p-6 border-b border-gray-800">
        <div className="flex items-center justify-between mb-6">
          <div className="flex items-center gap-4">
            <button
              onClick={onBack}
              className="p-2 text-gray-400 hover:text-white hover:bg-gray-700 rounded-lg transition-colors"
            >
              <ArrowLeft size={20} />
            </button>
            <div>
              <h1 className="text-2xl font-bold text-white mb-2">{project.name}</h1>
              <p className="text-gray-400">
                {project.description || 'Project workflows and automation'}
              </p>
              {/* Folder Path Display/Edit */}
              <div className="mt-3 flex items-center gap-2">
                <FolderOpen size={14} className="text-gray-500" />
                {isEditingPath ? (
                  <div className="flex items-center gap-2 flex-1">
                    <input
                      type="text"
                      value={editedPath}
                      onChange={(e) => setEditedPath(e.target.value)}
                      className="flex-1 px-2 py-1 bg-flow-node border border-gray-700 rounded text-sm text-white focus:outline-none focus:border-flow-accent"
                      placeholder="Enter save path..."
                      disabled={isSavingPath}
                    />
                    <button
                      onClick={handleSavePath}
                      disabled={isSavingPath}
                      className="p-1.5 text-green-400 hover:text-green-300 disabled:opacity-50"
                      title="Save"
                    >
                      {isSavingPath ? (
                        <Loader size={14} className="animate-spin" />
                      ) : (
                        <Check size={14} />
                      )}
                    </button>
                    <button
                      onClick={handleCancelEdit}
                      disabled={isSavingPath}
                      className="p-1.5 text-red-400 hover:text-red-300 disabled:opacity-50"
                      title="Cancel"
                    >
                      <X size={14} />
                    </button>
                  </div>
                ) : (
                  <div className="flex items-center gap-2">
                    <span className="text-sm text-gray-500">
                      Save path: <span className="text-gray-400 font-mono">{project.folder_path}</span>
                    </span>
                    <button
                      onClick={() => setIsEditingPath(true)}
                      className="p-1 text-gray-400 hover:text-white transition-colors"
                      title="Edit save path"
                    >
                      <Edit2 size={14} />
                    </button>
                  </div>
                )}
              </div>
            </div>
          </div>
          <div className="flex items-center gap-3">
            <button
              onClick={onCreateWorkflow}
              className="flex items-center gap-2 px-4 py-2 bg-flow-accent text-white rounded-lg hover:bg-blue-600 transition-colors"
            >
              <Plus size={16} />
              New Workflow
            </button>
          </div>
        </div>

        {/* Project Stats */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
          <div className="bg-flow-node border border-gray-700 rounded-lg p-4">
            <div className="text-2xl font-bold text-white">
              {project.stats?.workflow_count || workflows.length}
            </div>
            <div className="text-sm text-gray-400">Workflows</div>
          </div>
          <div className="bg-flow-node border border-gray-700 rounded-lg p-4">
            <div className="text-2xl font-bold text-white">
              {project.stats?.execution_count || 0}
            </div>
            <div className="text-sm text-gray-400">Total Executions</div>
          </div>
          <div className="bg-flow-node border border-gray-700 rounded-lg p-4">
            <div className="text-2xl font-bold text-white">
              {project.stats?.last_execution ? formatDate(project.stats.last_execution) : 'Never'}
            </div>
            <div className="text-sm text-gray-400">Last Execution</div>
          </div>
          <div className="bg-flow-node border border-gray-700 rounded-lg p-4">
            <div className="text-2xl font-bold text-white">
              {formatDate(project.updated_at)}
            </div>
            <div className="text-sm text-gray-400">Last Updated</div>
          </div>
        </div>

        {/* Search and View Mode Toggle */}
        <div className="flex items-center gap-3">
          <div className="relative flex-1">
            <input
              type="text"
              placeholder="Search workflows..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full px-4 py-2 bg-flow-node border border-gray-700 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-flow-accent"
            />
          </div>
          <div className="flex items-center bg-flow-node border border-gray-700 rounded-lg p-1">
            <button
              onClick={() => setViewMode('card')}
              className={`px-3 py-1.5 rounded flex items-center gap-2 transition-colors ${
                viewMode === 'card' 
                  ? 'bg-flow-accent text-white' 
                  : 'text-gray-400 hover:text-white'
              }`}
              title="Card View"
            >
              <LayoutGrid size={16} />
              <span className="text-sm">Cards</span>
            </button>
            <button
              onClick={() => setViewMode('tree')}
              className={`px-3 py-1.5 rounded flex items-center gap-2 transition-colors ${
                viewMode === 'tree' 
                  ? 'bg-flow-accent text-white' 
                  : 'text-gray-400 hover:text-white'
              }`}
              title="Tree View"
            >
              <FolderTree size={16} />
              <span className="text-sm">Tree</span>
            </button>
          </div>
        </div>
      </div>

      {/* Status Bar for API errors */}
      <StatusBar />

      {/* Content */}
      <div className="flex-1 overflow-auto p-6">
        {isLoading ? (
          <div className="flex items-center justify-center h-64">
            <div className="text-gray-400">Loading workflows...</div>
          </div>
        ) : filteredWorkflows.length === 0 && searchTerm === '' ? (
          <div className="flex items-center justify-center h-64">
            <div className="text-center">
              <div className="text-gray-600 mb-4">
                {error ? <WifiOff size={48} /> : <FileCode size={48} />}
              </div>
              <h3 className="text-lg font-semibold text-white mb-2">
                {error ? 'Unable to Load Workflows' : 'No Workflows Yet'}
              </h3>
              <p className="text-gray-400 mb-6">
                {error 
                  ? 'There was an issue connecting to the API. You can still use the interface when the connection is restored.'
                  : 'Create your first workflow to start automating tasks in this project'
                }
              </p>
              {!error && (
                <button
                  onClick={onCreateWorkflow}
                  className="flex items-center gap-2 px-4 py-2 bg-flow-accent text-white rounded-lg hover:bg-blue-600 transition-colors"
                >
                  <Plus size={16} />
                  Create Workflow
                </button>
              )}
            </div>
          </div>
        ) : filteredWorkflows.length === 0 ? (
          <div className="flex items-center justify-center h-64">
            <div className="text-center">
              <div className="text-gray-600 mb-4">
                <FileCode size={48} />
              </div>
              <h3 className="text-lg font-semibold text-white mb-2">No Workflows Found</h3>
              <p className="text-gray-400">No workflows match your search criteria</p>
            </div>
          </div>
        ) : viewMode === 'tree' ? (
          <div className="bg-flow-node border border-gray-700 rounded-lg p-4">
            {folderStructure.length === 0 ? (
              <div className="text-center text-gray-400 py-8">
                <FileCode size={48} className="mx-auto mb-4 opacity-50" />
                <p>No folder structure available</p>
              </div>
            ) : (
              <div className="space-y-1">
                {folderStructure.map((folder) => (
                  <FolderTreeItem key={folder.path} item={folder} />
                ))}
              </div>
            )}
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
            {filteredWorkflows.map((workflow) => (
              <div
                key={workflow.id}
                onClick={() => onWorkflowSelect(workflow)}
                className="bg-flow-node border border-gray-700 rounded-lg p-6 cursor-pointer hover:border-flow-accent hover:shadow-lg hover:shadow-blue-500/20 transition-all"
              >
                {/* Workflow Header */}
                <div className="flex items-start justify-between mb-4">
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 bg-green-500/20 rounded-lg flex items-center justify-center">
                      <FileCode size={16} className="text-green-400" />
                    </div>
                    <div>
                      <h3 className="font-semibold text-white truncate max-w-32" title={workflow.name}>
                        {workflow.name}
                      </h3>
                    </div>
                  </div>
                  
                  {/* Action Buttons */}
                  <div className="flex items-center gap-1">
                    {/* Execute Button */}
                    <button
                      onClick={(e) => handleExecuteWorkflow(e, workflow.id)}
                      disabled={executionInProgress[workflow.id]}
                      className="p-2 text-gray-400 hover:text-white hover:bg-gray-700 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                      title={executionInProgress[workflow.id] ? 'Executing workflow...' : 'Execute Workflow'}
                    >
                      {executionInProgress[workflow.id] ? (
                        <Loader size={14} className="animate-spin" />
                      ) : (
                        <PlayCircle size={14} />
                      )}
                    </button>
                  </div>
                </div>

                {/* Workflow Description */}
                {workflow.description && (
                  <p className="text-gray-400 text-sm mb-4 line-clamp-2">
                    {workflow.description}
                  </p>
                )}

                {/* Workflow Stats */}
                <div className="grid grid-cols-2 gap-4 mb-4">
                  <div className="text-center">
                    <div className="text-lg font-semibold text-white">
                      {workflow.stats?.execution_count || 0}
                    </div>
                    <div className="text-xs text-gray-500">Executions</div>
                  </div>
                  <div className="text-center">
                    <div className="text-lg font-semibold text-white">
                      {workflow.stats?.success_rate ? `${workflow.stats.success_rate}%` : 'N/A'}
                    </div>
                    <div className="text-xs text-gray-500">Success Rate</div>
                  </div>
                </div>

                {/* Last Activity */}
                <div className="flex items-center justify-between text-xs text-gray-500 pt-4 border-t border-gray-700">
                  <div className="flex items-center gap-1">
                    <Clock size={12} />
                    <span>Updated {formatDate(workflow.updated_at)}</span>
                  </div>
                  {workflow.stats?.last_execution && (
                    <div className="flex items-center gap-1">
                      <Play size={12} />
                      <span>Last run {formatDate(workflow.stats.last_execution)}</span>
                    </div>
                  )}
                </div>

                {/* Folder Path */}
                <div className="mt-2 text-xs text-gray-600 truncate" title={workflow.folder_path}>
                  {workflow.folder_path}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

export default ProjectDetail;