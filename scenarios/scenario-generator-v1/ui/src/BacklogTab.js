import React, { useState, useEffect, useCallback } from 'react';
import axios from 'axios';
import {
  Folder,
  FileText,
  Clock,
  Play,
  CheckCircle,
  AlertCircle,
  Loader,
  Plus,
  Edit,
  Trash2,
  Download,
  Upload,
  RefreshCw,
  ChevronRight,
  DollarSign,
  Tag,
  Calendar,
  User,
  ArrowRight,
  Archive,
  Filter,
  Search
} from 'lucide-react';

const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';

function BacklogTab() {
  const [backlogData, setBacklogData] = useState({
    pending: [],
    in_progress: [],
    completed: [],
    failed: []
  });
  const [loading, setLoading] = useState(true);
  const [selectedItem, setSelectedItem] = useState(null);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [showEditModal, setShowEditModal] = useState(false);
  const [selectedFolder, setSelectedFolder] = useState('pending');
  const [searchTerm, setSearchTerm] = useState('');
  const [error, setError] = useState(null);

  // Fetch backlog data with filtering
  const fetchBacklog = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      
      // Build query parameters
      const params = new URLSearchParams();
      if (searchTerm) params.append('search', searchTerm);
      
      const response = await axios.get(`${API_URL}/api/backlog?${params.toString()}`);
      setBacklogData(response.data || {
        pending: [],
        in_progress: [],
        completed: [],
        failed: []
      });
    } catch (err) {
      setError('Failed to load backlog. Make sure the API is running.');
      console.error('Error fetching backlog:', err);
    } finally {
      setLoading(false);
    }
  }, [searchTerm]);

  useEffect(() => {
    fetchBacklog();
    // Auto-refresh every 10 seconds
    const interval = setInterval(fetchBacklog, 10000);
    return () => clearInterval(interval);
  }, [fetchBacklog]);
  
  // Trigger search when searchTerm changes with debounce
  useEffect(() => {
    const delayedSearch = setTimeout(() => {
      fetchBacklog(); // This will now include the search term
    }, 300);
    
    return () => clearTimeout(delayedSearch);
  }, [searchTerm, fetchBacklog]);

  // Generate scenario from backlog item
  const handleGenerate = async (item) => {
    try {
      setError(null);
      const response = await axios.post(`${API_URL}/api/backlog/${item.id}/generate`);
      
      if (response.data) {
        alert(`Generation started for: ${item.name}`);
        fetchBacklog(); // Refresh to show updated status
      }
    } catch (err) {
      setError(`Failed to start generation: ${err.message}`);
      console.error('Error generating scenario:', err);
    }
  };

  // Move item between folders
  const handleMove = async (item, toState) => {
    try {
      setError(null);
      await axios.post(`${API_URL}/api/backlog/${item.id}/move`, {
        to_state: toState
      });
      fetchBacklog();
    } catch (err) {
      setError(`Failed to move item: ${err.message}`);
      console.error('Error moving item:', err);
    }
  };

  // Delete backlog item
  const handleDelete = async (item) => {
    if (!window.confirm(`Delete "${item.name}" from backlog?`)) {
      return;
    }
    
    try {
      setError(null);
      await axios.delete(`${API_URL}/api/backlog/${item.id}`);
      fetchBacklog();
    } catch (err) {
      setError(`Failed to delete item: ${err.message}`);
      console.error('Error deleting item:', err);
    }
  };

  const [metadata, setMetadata] = useState(null);
  
  // Fetch metadata from API
  useEffect(() => {
    const fetchMetadata = async () => {
      try {
        const response = await axios.get(`${API_URL}/api/backlog/metadata`);
        setMetadata(response.data);
      } catch (err) {
        console.error('Error fetching metadata:', err);
      }
    };
    
    fetchMetadata();
  }, []);
  
  // Get folder color and icon from API metadata
  const getFolderStyle = (folder) => {
    if (!metadata) return { color: 'text-gray-500', icon: Folder };
    
    const statusOption = metadata.status_options?.find(s => s.id === folder);
    if (!statusOption) return { color: 'text-gray-500', icon: Folder };
    
    const iconMap = {
      'clock': Clock,
      'loader': Loader, 
      'check-circle': CheckCircle,
      'alert-circle': AlertCircle
    };
    
    return {
      color: statusOption.color,
      icon: iconMap[statusOption.icon] || Folder
    };
  };

  // Get priority color from API metadata
  const getPriorityColor = (priority) => {
    if (!metadata) return 'text-gray-500';
    
    const priorityOption = metadata.priority_options?.find(p => p.id === priority?.toLowerCase());
    return priorityOption ? priorityOption.color : 'text-gray-500';
  };

  // Get current folder items (no client-side filtering needed)
  const getCurrentItems = () => {
    return backlogData[selectedFolder] || [];
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold text-white flex items-center">
          <Archive className="h-6 w-6 text-cyan-400 mr-2" />
          Scenario Backlog
        </h2>
        
        <div className="flex space-x-2">
          <button
            onClick={() => setShowCreateModal(true)}
            className="flex items-center px-4 py-2 bg-cyan-600 text-white rounded-lg hover:bg-cyan-700"
          >
            <Plus className="h-4 w-4 mr-2" />
            Add to Backlog
          </button>
          
          <button
            onClick={fetchBacklog}
            className="flex items-center px-4 py-2 bg-gray-700 text-white rounded-lg hover:bg-gray-600"
          >
            <RefreshCw className="h-4 w-4 mr-2" />
            Refresh
          </button>
        </div>
      </div>

      {/* Error Display */}
      {error && (
        <div className="bg-red-900/50 border border-red-500 text-red-200 px-4 py-3 rounded">
          <div className="flex items-center">
            <AlertCircle className="h-5 w-5 mr-2" />
            {error}
          </div>
        </div>
      )}

      {/* Search Bar */}
      <div className="relative">
        <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 h-5 w-5" />
        <input
          type="text"
          placeholder="Search backlog items..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          className="w-full pl-10 pr-4 py-3 bg-gray-800 border border-gray-600 rounded-lg text-white placeholder-gray-400 focus:outline-none focus:border-cyan-500"
        />
      </div>

      {/* Folder Tabs */}
      <div className="flex space-x-1 bg-gray-800/50 p-1 rounded-lg">
        {Object.keys(backlogData).map((folder) => {
          const { color, icon: Icon } = getFolderStyle(folder);
          const count = backlogData[folder]?.length || 0;
          const displayName = folder.replace('_', ' ').replace(/\b\w/g, l => l.toUpperCase());
          
          return (
            <button
              key={folder}
              onClick={() => setSelectedFolder(folder)}
              className={`flex-1 flex items-center justify-center px-4 py-3 rounded-md transition-all ${
                selectedFolder === folder
                  ? 'bg-cyan-600 text-white shadow-lg'
                  : 'text-gray-300 hover:text-white hover:bg-gray-700'
              }`}
            >
              <Icon className={`h-5 w-5 mr-2 ${selectedFolder === folder ? '' : color}`} />
              {displayName}
              <span className="ml-2 px-2 py-1 bg-black/20 rounded-full text-xs">
                {count}
              </span>
            </button>
          );
        })}
      </div>

      {/* Items List */}
      <div className="bg-gray-800/50 rounded-lg p-4">
        {loading ? (
          <div className="flex justify-center py-12">
            <Loader className="h-8 w-8 text-cyan-400 animate-spin" />
          </div>
        ) : getCurrentItems().length > 0 ? (
          <div className="space-y-4">
            {getCurrentItems().map((item) => (
              <BacklogItem
                key={item.id || item.filename}
                item={item}
                folder={selectedFolder}
                onView={() => setSelectedItem(item)}
                onGenerate={() => handleGenerate(item)}
                onMove={handleMove}
                onDelete={() => handleDelete(item)}
                onEdit={() => {
                  setSelectedItem(item);
                  setShowEditModal(true);
                }}
              />
            ))}
          </div>
        ) : (
          <div className="text-center py-12 text-gray-400">
            <FileText className="h-16 w-16 mx-auto mb-4 opacity-50" />
            <p className="text-lg">No items in {selectedFolder.replace('_', ' ')}</p>
            {selectedFolder === 'pending' && (
              <p className="mt-2">Add items to the backlog or check the backlog/pending folder</p>
            )}
          </div>
        )}
      </div>

      {/* Quick Stats */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <StatCard
          title="Total Pending"
          value={backlogData.pending?.length || 0}
          icon={Clock}
          color="text-yellow-500"
        />
        <StatCard
          title="In Progress"
          value={backlogData.in_progress?.length || 0}
          icon={Loader}
          color="text-blue-500"
        />
        <StatCard
          title="Completed"
          value={backlogData.completed?.length || 0}
          icon={CheckCircle}
          color="text-green-500"
        />
        <RevenueStatCard />
      </div>

      {/* Item Detail Modal */}
      {selectedItem && !showEditModal && (
        <BacklogItemModal
          item={selectedItem}
          onClose={() => setSelectedItem(null)}
          onGenerate={() => handleGenerate(selectedItem)}
          onEdit={() => {
            setShowEditModal(true);
          }}
        />
      )}

      {/* Create/Edit Modal */}
      {(showCreateModal || showEditModal) && (
        <BacklogFormModal
          item={showEditModal ? selectedItem : null}
          onClose={() => {
            setShowCreateModal(false);
            setShowEditModal(false);
            setSelectedItem(null);
          }}
          onSave={async (item) => {
            try {
              if (showEditModal) {
                await axios.put(`${API_URL}/api/backlog/${selectedItem.id}`, item);
              } else {
                await axios.post(`${API_URL}/api/backlog`, item);
              }
              fetchBacklog();
              setShowCreateModal(false);
              setShowEditModal(false);
              setSelectedItem(null);
            } catch (err) {
              setError(`Failed to save item: ${err.message}`);
            }
          }}
        />
      )}
    </div>
  );
}

// Backlog Item Component
function BacklogItem({ item, folder, onView, onGenerate, onMove, onDelete, onEdit }) {
  const getPriorityBadge = (priority) => {
    const colors = {
      high: 'bg-red-900/50 text-red-300 border-red-700',
      medium: 'bg-yellow-900/50 text-yellow-300 border-yellow-700',
      low: 'bg-green-900/50 text-green-300 border-green-700'
    };
    return colors[priority?.toLowerCase()] || colors.medium;
  };

  return (
    <div className="bg-gray-700/50 border border-gray-600 rounded-lg p-4 hover:border-cyan-500 transition-all">
      <div className="flex items-start justify-between">
        <div className="flex-1">
          <div className="flex items-center space-x-2 mb-2">
            <h3 className="text-lg font-semibold text-white">
              {item.name}
            </h3>
            <span className={`px-2 py-1 rounded text-xs border ${getPriorityBadge(item.priority)}`}>
              {item.priority || 'medium'}
            </span>
          </div>
          
          <p className="text-gray-300 text-sm mb-3">
            {item.description}
          </p>
          
          <div className="flex items-center space-x-4 text-sm">
            <span className="text-gray-400 flex items-center">
              <Tag className="h-4 w-4 mr-1" />
              {item.category || 'general'}
            </span>
            <span className="text-cyan-400 flex items-center">
              <DollarSign className="h-4 w-4" />
              {(item.estimated_revenue || 0).toLocaleString()}
            </span>
            {item.metadata?.deadline && (
              <span className="text-gray-400 flex items-center">
                <Calendar className="h-4 w-4 mr-1" />
                {item.metadata.deadline}
              </span>
            )}
          </div>
          
          {item.tags && item.tags.length > 0 && (
            <div className="flex flex-wrap gap-1 mt-2">
              {item.tags.map((tag, idx) => (
                <span key={idx} className="px-2 py-1 bg-gray-800 text-gray-300 rounded text-xs">
                  {tag}
                </span>
              ))}
            </div>
          )}
        </div>
        
        <div className="flex flex-col space-y-2 ml-4">
          <button
            onClick={onView}
            className="p-2 text-gray-400 hover:text-white hover:bg-gray-600 rounded"
            title="View Details"
          >
            <FileText className="h-4 w-4" />
          </button>
          
          <button
            onClick={onEdit}
            className="p-2 text-gray-400 hover:text-white hover:bg-gray-600 rounded"
            title="Edit"
          >
            <Edit className="h-4 w-4" />
          </button>
          
          {folder === 'pending' && (
            <button
              onClick={onGenerate}
              className="p-2 text-cyan-400 hover:text-cyan-300 hover:bg-gray-600 rounded"
              title="Generate Now"
            >
              <Play className="h-4 w-4" />
            </button>
          )}
          
          {folder === 'failed' && (
            <button
              onClick={() => onMove(item, 'pending')}
              className="p-2 text-yellow-400 hover:text-yellow-300 hover:bg-gray-600 rounded"
              title="Retry"
            >
              <RefreshCw className="h-4 w-4" />
            </button>
          )}
          
          <button
            onClick={onDelete}
            className="p-2 text-red-400 hover:text-red-300 hover:bg-gray-600 rounded"
            title="Delete"
          >
            <Trash2 className="h-4 w-4" />
          </button>
        </div>
      </div>
    </div>
  );
}

// Stat Card Component
function StatCard({ title, value, icon: Icon, color }) {
  return (
    <div className="bg-gray-800/50 border border-gray-600 rounded-lg p-4">
      <div className="flex items-center justify-between">
        <div>
          <p className="text-gray-400 text-sm">{title}</p>
          <p className="text-2xl font-bold text-white">{value}</p>
        </div>
        <Icon className={`h-8 w-8 ${color}`} />
      </div>
    </div>
  );
}

// Backlog Item Modal
function BacklogItemModal({ item, onClose, onGenerate, onEdit }) {
  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
      <div className="bg-gray-800 rounded-lg max-w-4xl w-full max-h-[90vh] overflow-y-auto">
        <div className="p-6">
          <div className="flex items-start justify-between mb-4">
            <h2 className="text-2xl font-bold text-white">
              {item.name}
            </h2>
            <button
              onClick={onClose}
              className="text-gray-400 hover:text-white"
            >
              ✕
            </button>
          </div>
          
          <div className="space-y-4">
            <div>
              <h3 className="text-lg font-semibold text-white mb-2">Description</h3>
              <p className="text-gray-300">{item.description}</p>
            </div>
            
            <div>
              <h3 className="text-lg font-semibold text-white mb-2">Prompt</h3>
              <pre className="text-gray-300 bg-gray-900/50 p-3 rounded whitespace-pre-wrap">
                {item.prompt}
              </pre>
            </div>
            
            <div className="grid grid-cols-2 gap-4">
              <div>
                <h3 className="text-lg font-semibold text-white mb-2">Details</h3>
                <div className="space-y-2 text-sm">
                  <div>
                    <span className="text-gray-400">Priority:</span>
                    <span className="ml-2 text-white">{item.priority}</span>
                  </div>
                  <div>
                    <span className="text-gray-400">Complexity:</span>
                    <span className="ml-2 text-white">{item.complexity}</span>
                  </div>
                  <div>
                    <span className="text-gray-400">Category:</span>
                    <span className="ml-2 text-white">{item.category}</span>
                  </div>
                  <div>
                    <span className="text-gray-400">Est. Revenue:</span>
                    <span className="ml-2 text-white">${item.estimated_revenue?.toLocaleString()}</span>
                  </div>
                </div>
              </div>
              
              {item.metadata && (
                <div>
                  <h3 className="text-lg font-semibold text-white mb-2">Metadata</h3>
                  <div className="space-y-2 text-sm">
                    {item.metadata.requested_by && (
                      <div>
                        <span className="text-gray-400">Requested By:</span>
                        <span className="ml-2 text-white">{item.metadata.requested_by}</span>
                      </div>
                    )}
                    {item.metadata.requested_date && (
                      <div>
                        <span className="text-gray-400">Requested:</span>
                        <span className="ml-2 text-white">{item.metadata.requested_date}</span>
                      </div>
                    )}
                    {item.metadata.deadline && (
                      <div>
                        <span className="text-gray-400">Deadline:</span>
                        <span className="ml-2 text-white">{item.metadata.deadline}</span>
                      </div>
                    )}
                  </div>
                </div>
              )}
            </div>
            
            {item.validation_criteria && item.validation_criteria.length > 0 && (
              <div>
                <h3 className="text-lg font-semibold text-white mb-2">Validation Criteria</h3>
                <ul className="list-disc list-inside text-gray-300 space-y-1">
                  {item.validation_criteria.map((criterion, idx) => (
                    <li key={idx}>{criterion}</li>
                  ))}
                </ul>
              </div>
            )}
            
            {item.notes && (
              <div>
                <h3 className="text-lg font-semibold text-white mb-2">Notes</h3>
                <p className="text-gray-300 bg-gray-900/50 p-3 rounded">
                  {item.notes}
                </p>
              </div>
            )}
            
            <div className="flex space-x-2 pt-4">
              <button
                onClick={onGenerate}
                className="flex items-center px-4 py-2 bg-cyan-600 text-white rounded-lg hover:bg-cyan-700"
              >
                <Play className="h-4 w-4 mr-2" />
                Generate Scenario
              </button>
              
              <button
                onClick={onEdit}
                className="flex items-center px-4 py-2 bg-gray-700 text-white rounded-lg hover:bg-gray-600"
              >
                <Edit className="h-4 w-4 mr-2" />
                Edit
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

// Backlog Form Modal
function BacklogFormModal({ item, onClose, onSave }) {
  const [formData, setFormData] = useState(item || {
    name: '',
    description: '',
    prompt: '',
    complexity: 'intermediate',
    category: 'business-tool',
    priority: 'medium',
    estimated_revenue: 25000,
    tags: [],
    metadata: {
      requested_by: '',
      requested_date: new Date().toISOString().split('T')[0],
      deadline: ''
    },
    resources_required: ['postgres', 'redis', 'claude-code'],
    validation_criteria: [],
    notes: ''
  });

  const handleSubmit = (e) => {
    e.preventDefault();
    onSave(formData);
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
      <div className="bg-gray-800 rounded-lg max-w-2xl w-full max-h-[90vh] overflow-y-auto">
        <div className="p-6">
          <div className="flex items-start justify-between mb-4">
            <h2 className="text-2xl font-bold text-white">
              {item ? 'Edit Backlog Item' : 'Add to Backlog'}
            </h2>
            <button
              onClick={onClose}
              className="text-gray-400 hover:text-white"
            >
              ✕
            </button>
          </div>
          
          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label className="block text-white font-medium mb-2">
                Name *
              </label>
              <input
                type="text"
                required
                value={formData.name}
                onChange={(e) => setFormData({...formData, name: e.target.value})}
                className="w-full px-4 py-3 bg-gray-700 border border-gray-600 rounded-lg text-white focus:outline-none focus:border-cyan-500"
              />
            </div>
            
            <div>
              <label className="block text-white font-medium mb-2">
                Description *
              </label>
              <input
                type="text"
                required
                value={formData.description}
                onChange={(e) => setFormData({...formData, description: e.target.value})}
                className="w-full px-4 py-3 bg-gray-700 border border-gray-600 rounded-lg text-white focus:outline-none focus:border-cyan-500"
              />
            </div>
            
            <div>
              <label className="block text-white font-medium mb-2">
                Prompt *
              </label>
              <textarea
                required
                rows={6}
                value={formData.prompt}
                onChange={(e) => setFormData({...formData, prompt: e.target.value})}
                className="w-full px-4 py-3 bg-gray-700 border border-gray-600 rounded-lg text-white focus:outline-none focus:border-cyan-500"
              />
            </div>
            
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-white font-medium mb-2">
                  Priority
                </label>
                <select
                  value={formData.priority}
                  onChange={(e) => setFormData({...formData, priority: e.target.value})}
                  className="w-full px-4 py-3 bg-gray-700 border border-gray-600 rounded-lg text-white focus:outline-none focus:border-cyan-500"
                >
                  <option value="high">High</option>
                  <option value="medium">Medium</option>
                  <option value="low">Low</option>
                </select>
              </div>
              
              <div>
                <label className="block text-white font-medium mb-2">
                  Complexity
                </label>
                <select
                  value={formData.complexity}
                  onChange={(e) => setFormData({...formData, complexity: e.target.value})}
                  className="w-full px-4 py-3 bg-gray-700 border border-gray-600 rounded-lg text-white focus:outline-none focus:border-cyan-500"
                >
                  <option value="simple">Simple</option>
                  <option value="intermediate">Intermediate</option>
                  <option value="advanced">Advanced</option>
                </select>
              </div>
            </div>
            
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-white font-medium mb-2">
                  Category
                </label>
                <input
                  type="text"
                  value={formData.category}
                  onChange={(e) => setFormData({...formData, category: e.target.value})}
                  className="w-full px-4 py-3 bg-gray-700 border border-gray-600 rounded-lg text-white focus:outline-none focus:border-cyan-500"
                />
              </div>
              
              <div>
                <label className="block text-white font-medium mb-2">
                  Est. Revenue ($)
                </label>
                <input
                  type="number"
                  value={formData.estimated_revenue}
                  onChange={(e) => setFormData({...formData, estimated_revenue: parseInt(e.target.value)})}
                  className="w-full px-4 py-3 bg-gray-700 border border-gray-600 rounded-lg text-white focus:outline-none focus:border-cyan-500"
                />
              </div>
            </div>
            
            <div>
              <label className="block text-white font-medium mb-2">
                Tags (comma-separated)
              </label>
              <input
                type="text"
                value={formData.tags?.join(', ') || ''}
                onChange={(e) => setFormData({
                  ...formData, 
                  tags: e.target.value.split(',').map(t => t.trim()).filter(t => t)
                })}
                className="w-full px-4 py-3 bg-gray-700 border border-gray-600 rounded-lg text-white focus:outline-none focus:border-cyan-500"
              />
            </div>
            
            <div>
              <label className="block text-white font-medium mb-2">
                Notes
              </label>
              <textarea
                rows={3}
                value={formData.notes}
                onChange={(e) => setFormData({...formData, notes: e.target.value})}
                className="w-full px-4 py-3 bg-gray-700 border border-gray-600 rounded-lg text-white focus:outline-none focus:border-cyan-500"
              />
            </div>
            
            <div className="flex space-x-2 pt-4">
              <button
                type="submit"
                className="flex items-center px-6 py-3 bg-cyan-600 text-white rounded-lg hover:bg-cyan-700"
              >
                {item ? 'Update' : 'Add to Backlog'}
              </button>
              
              <button
                type="button"
                onClick={onClose}
                className="px-6 py-3 bg-gray-700 text-white rounded-lg hover:bg-gray-600"
              >
                Cancel
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
}

// Revenue component that fetches from API
function RevenueStatCard() {
  const [revenueData, setRevenueData] = React.useState(null);
  
  React.useEffect(() => {
    const fetchStats = async () => {
      try {
        const response = await axios.get(`${API_URL}/api/backlog/stats`);
        setRevenueData(response.data);
      } catch (err) {
        console.error('Error fetching backlog stats:', err);
      }
    };
    
    fetchStats();
    const interval = setInterval(fetchStats, 30000); // Refresh every 30 seconds
    return () => clearInterval(interval);
  }, []);
  
  return (
    <StatCard
      title="Total Revenue"
      value={revenueData ? `$${revenueData.total_revenue_k}k` : 'Loading...'}
      icon={DollarSign}
      color="text-cyan-500"
    />
  );
}

export default BacklogTab;