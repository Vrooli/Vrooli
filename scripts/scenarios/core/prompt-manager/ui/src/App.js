import React, { useState, useEffect, useCallback } from 'react';
import axios from 'axios';
import TextareaAutosize from 'react-textarea-autosize';
import { 
  Search, 
  Plus, 
  Folder, 
  Star, 
  Heart,
  Copy, 
  Edit3, 
  Trash2, 
  Settings,
  Sun,
  Moon,
  FolderPlus,
  BookOpen,
  Clock,
  TrendingUp,
  Filter,
  X,
  Save,
  ChevronDown,
  ChevronRight,
  Tag,
  Zap,
  FileText,
  User,
  Database
} from 'lucide-react';

const API_URL = process.env.REACT_APP_API_URL || '';

function App() {
  // State management
  const [campaigns, setCampaigns] = useState([]);
  const [prompts, setPrompts] = useState([]);
  const [selectedCampaign, setSelectedCampaign] = useState(null);
  const [selectedPrompt, setSelectedPrompt] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  
  // UI state
  const [darkMode, setDarkMode] = useState(() => 
    localStorage.getItem('darkMode') === 'true' || false
  );
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');
  const [showNewCampaignForm, setShowNewCampaignForm] = useState(false);
  const [showNewPromptForm, setShowNewPromptForm] = useState(false);
  const [activeView, setActiveView] = useState('all'); // all, favorites, recent
  
  // Form state
  const [newCampaign, setNewCampaign] = useState({ name: '', description: '', color: '#3b82f6' });
  const [newPrompt, setNewPrompt] = useState({ 
    title: '', 
    content: '', 
    description: '', 
    campaign_id: '',
    tags: [] 
  });
  const [editingPrompt, setEditingPrompt] = useState(null);

  // Dark mode effect
  useEffect(() => {
    if (darkMode) {
      document.documentElement.classList.add('dark');
    } else {
      document.documentElement.classList.remove('dark');
    }
    localStorage.setItem('darkMode', darkMode);
  }, [darkMode]);

  // API functions
  const fetchCampaigns = useCallback(async () => {
    try {
      const response = await axios.get(`${API_URL}/api/campaigns`);
      setCampaigns(response.data || []);
    } catch (err) {
      setError('Failed to load campaigns');
      console.error('Error fetching campaigns:', err);
    }
  }, []);

  const fetchPrompts = useCallback(async (campaignId = null, view = 'all') => {
    try {
      setLoading(true);
      let url = `${API_URL}/api/prompts`;
      let params = {};
      
      if (campaignId) {
        url = `${API_URL}/api/campaigns/${campaignId}/prompts`;
      } else if (view === 'favorites') {
        params.favorites = 'true';
      } else if (view === 'recent') {
        params.limit = '20';
      }

      const response = await axios.get(url, { params });
      setPrompts(response.data || []);
    } catch (err) {
      setError('Failed to load prompts');
      console.error('Error fetching prompts:', err);
    } finally {
      setLoading(false);
    }
  }, []);

  const searchPrompts = useCallback(async (query) => {
    if (!query.trim()) {
      fetchPrompts(selectedCampaign?.id, activeView);
      return;
    }
    
    try {
      setLoading(true);
      const response = await axios.get(`${API_URL}/api/search/prompts?q=${encodeURIComponent(query)}`);
      setPrompts(response.data || []);
    } catch (err) {
      setError('Search failed');
      console.error('Error searching prompts:', err);
    } finally {
      setLoading(false);
    }
  }, [selectedCampaign, activeView, fetchPrompts]);

  const createCampaign = async () => {
    try {
      const response = await axios.post(`${API_URL}/api/campaigns`, newCampaign);
      setCampaigns(prev => [...prev, response.data]);
      setNewCampaign({ name: '', description: '', color: '#3b82f6' });
      setShowNewCampaignForm(false);
    } catch (err) {
      setError('Failed to create campaign');
      console.error('Error creating campaign:', err);
    }
  };

  const createPrompt = async () => {
    try {
      const promptData = {
        ...newPrompt,
        campaign_id: newPrompt.campaign_id || selectedCampaign?.id || campaigns[0]?.id
      };
      
      if (!promptData.campaign_id) {
        setError('Please select a campaign');
        return;
      }

      const response = await axios.post(`${API_URL}/api/prompts`, promptData);
      setPrompts(prev => [response.data, ...prev]);
      setNewPrompt({ title: '', content: '', description: '', campaign_id: '', tags: [] });
      setShowNewPromptForm(false);
      
      // Update campaign prompt count
      fetchCampaigns();
    } catch (err) {
      setError('Failed to create prompt');
      console.error('Error creating prompt:', err);
    }
  };

  const copyPromptToClipboard = async (prompt) => {
    try {
      await navigator.clipboard.writeText(prompt.content);
      
      // Record usage
      await axios.post(`${API_URL}/api/prompts/${prompt.id}/use`);
      
      // Update local state
      setPrompts(prev => 
        prev.map(p => 
          p.id === prompt.id 
            ? { ...p, usage_count: p.usage_count + 1, last_used: new Date().toISOString() }
            : p
        )
      );
      
      // Show success feedback (you could add a toast here)
      console.log('Copied to clipboard and usage recorded');
    } catch (err) {
      console.error('Failed to copy to clipboard:', err);
      setError('Failed to copy to clipboard');
    }
  };

  // Initial load
  useEffect(() => {
    fetchCampaigns();
    fetchPrompts();
  }, [fetchCampaigns, fetchPrompts]);

  // Update prompts when campaign or view changes
  useEffect(() => {
    if (!searchTerm) {
      fetchPrompts(selectedCampaign?.id, activeView);
    }
  }, [selectedCampaign, activeView, fetchPrompts, searchTerm]);

  // Search effect
  useEffect(() => {
    const timer = setTimeout(() => {
      if (searchTerm) {
        searchPrompts(searchTerm);
      }
    }, 300);
    
    return () => clearTimeout(timer);
  }, [searchTerm, searchPrompts]);

  const campaignColors = [
    '#3b82f6', '#ef4444', '#10b981', '#f59e0b', '#8b5cf6', 
    '#ec4899', '#06b6d4', '#84cc16', '#f97316', '#6b7280'
  ];

  return (
    <div className="flex h-screen bg-gray-50 dark:bg-gray-900">
      {/* Sidebar */}
      <div className={`${sidebarCollapsed ? 'w-16' : 'w-80'} bg-white dark:bg-gray-800 border-r border-gray-200 dark:border-gray-700 flex flex-col transition-all duration-300`}>
        {/* Header */}
        <div className="p-4 border-b border-gray-200 dark:border-gray-700">
          <div className="flex items-center justify-between">
            {!sidebarCollapsed && (
              <h1 className="text-xl font-bold text-gray-900 dark:text-white">
                📝 Prompts
              </h1>
            )}
            <div className="flex items-center gap-2">
              <button
                onClick={() => setDarkMode(!darkMode)}
                className="btn-ghost btn-sm"
                title="Toggle theme"
              >
                {darkMode ? <Sun size={16} /> : <Moon size={16} />}
              </button>
              <button
                onClick={() => setSidebarCollapsed(!sidebarCollapsed)}
                className="btn-ghost btn-sm"
              >
                {sidebarCollapsed ? <ChevronRight size={16} /> : <ChevronDown size={16} />}
              </button>
            </div>
          </div>
          
          {!sidebarCollapsed && (
            <div className="mt-4 flex gap-2">
              <button
                onClick={() => setShowNewCampaignForm(true)}
                className="btn-primary btn-sm flex items-center gap-1 flex-1"
              >
                <FolderPlus size={14} />
                Campaign
              </button>
              <button
                onClick={() => setShowNewPromptForm(true)}
                className="btn-secondary btn-sm flex items-center gap-1 flex-1"
              >
                <Plus size={14} />
                Prompt
              </button>
            </div>
          )}
        </div>

        {!sidebarCollapsed && (
          <>
            {/* View filters */}
            <div className="p-4 border-b border-gray-200 dark:border-gray-700">
              <div className="flex gap-1">
                {[
                  { key: 'all', label: 'All', icon: FileText },
                  { key: 'favorites', label: 'Favorites', icon: Heart },
                  { key: 'recent', label: 'Recent', icon: Clock }
                ].map(({ key, label, icon: Icon }) => (
                  <button
                    key={key}
                    onClick={() => {
                      setActiveView(key);
                      setSelectedCampaign(null);
                    }}
                    className={`sidebar-item flex-1 justify-center ${activeView === key && !selectedCampaign ? 'active' : ''}`}
                  >
                    <Icon size={14} />
                    <span className="ml-1 text-xs">{label}</span>
                  </button>
                ))}
              </div>
            </div>

            {/* Campaigns */}
            <div className="flex-1 overflow-y-auto">
              <div className="p-4">
                <h3 className="text-sm font-medium text-gray-500 dark:text-gray-400 mb-3">
                  CAMPAIGNS
                </h3>
                <div className="space-y-1">
                  {campaigns.map(campaign => (
                    <button
                      key={campaign.id}
                      onClick={() => {
                        setSelectedCampaign(campaign);
                        setActiveView('all');
                      }}
                      className={`sidebar-item w-full ${selectedCampaign?.id === campaign.id ? 'active' : ''}`}
                    >
                      <div 
                        className="w-3 h-3 rounded-full flex-shrink-0"
                        style={{ backgroundColor: campaign.color }}
                      />
                      <span className="ml-3 truncate">{campaign.name}</span>
                      <span className="ml-auto text-xs text-gray-500 dark:text-gray-400">
                        {campaign.prompt_count}
                      </span>
                    </button>
                  ))}
                </div>
              </div>
            </div>
          </>
        )}
      </div>

      {/* Main content */}
      <div className="flex-1 flex flex-col">
        {/* Top bar */}
        <div className="bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 p-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
                {selectedCampaign ? selectedCampaign.name : activeView === 'favorites' ? 'Favorite Prompts' : activeView === 'recent' ? 'Recent Prompts' : 'All Prompts'}
              </h2>
              <span className="badge badge-gray">
                {prompts.length} prompts
              </span>
            </div>
            
            <div className="flex items-center gap-3">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400" size={16} />
                <input
                  type="text"
                  placeholder="Search prompts..."
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  className="pl-10 pr-4 py-2 w-80 input"
                />
              </div>
            </div>
          </div>
        </div>

        {/* Content area */}
        <div className="flex-1 flex">
          {/* Prompts list */}
          <div className="w-1/3 border-r border-gray-200 dark:border-gray-700 overflow-y-auto">
            {loading ? (
              <div className="p-8 text-center">
                <div className="pulse-animation text-gray-500 dark:text-gray-400">
                  Loading prompts...
                </div>
              </div>
            ) : prompts.length === 0 ? (
              <div className="p-8 text-center text-gray-500 dark:text-gray-400">
                {searchTerm ? `No prompts found for "${searchTerm}"` : 'No prompts yet'}
                <div className="mt-4">
                  <button
                    onClick={() => setShowNewPromptForm(true)}
                    className="btn-primary btn-sm"
                  >
                    <Plus size={14} className="mr-1" />
                    Add your first prompt
                  </button>
                </div>
              </div>
            ) : (
              <div className="p-4 space-y-3">
                {prompts.map(prompt => (
                  <div
                    key={prompt.id}
                    onClick={() => setSelectedPrompt(prompt)}
                    className={`card cursor-pointer transition-all duration-200 hover:shadow-md ${
                      selectedPrompt?.id === prompt.id ? 'ring-2 ring-primary-500' : ''
                    }`}
                  >
                    <div className="p-4">
                      <div className="flex items-start justify-between">
                        <h3 className="font-medium text-gray-900 dark:text-white truncate flex-1">
                          {prompt.title}
                        </h3>
                        <div className="flex items-center gap-1 ml-2">
                          {prompt.is_favorite && (
                            <Heart size={14} className="text-red-500 fill-current" />
                          )}
                          <button
                            onClick={(e) => {
                              e.stopPropagation();
                              copyPromptToClipboard(prompt);
                            }}
                            className="opacity-0 group-hover:opacity-100 btn-ghost btn-xs"
                            title="Copy to clipboard"
                          >
                            <Copy size={12} />
                          </button>
                        </div>
                      </div>
                      
                      <p className="text-sm text-gray-600 dark:text-gray-300 mt-1 line-clamp-2">
                        {prompt.description || prompt.content?.slice(0, 100) + '...'}
                      </p>
                      
                      <div className="flex items-center justify-between mt-3 text-xs text-gray-500 dark:text-gray-400">
                        <div className="flex items-center gap-3">
                          <span>{prompt.campaign_name}</span>
                          <span>Used {prompt.usage_count} times</span>
                        </div>
                        {prompt.effectiveness_rating && (
                          <div className="flex items-center gap-1">
                            <Star size={12} className="text-yellow-500" />
                            <span>{prompt.effectiveness_rating}/5</span>
                          </div>
                        )}
                      </div>
                      
                      {prompt.tags && prompt.tags.length > 0 && (
                        <div className="flex flex-wrap gap-1 mt-2">
                          {prompt.tags.slice(0, 3).map(tag => (
                            <span key={tag} className="badge badge-gray text-xs">
                              {tag}
                            </span>
                          ))}
                          {prompt.tags.length > 3 && (
                            <span className="badge badge-gray text-xs">
                              +{prompt.tags.length - 3} more
                            </span>
                          )}
                        </div>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Prompt detail/editor */}
          <div className="flex-1 flex flex-col">
            {selectedPrompt ? (
              <div className="h-full flex flex-col">
                {/* Prompt header */}
                <div className="bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 p-6">
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
                        {selectedPrompt.title}
                      </h1>
                      {selectedPrompt.description && (
                        <p className="text-gray-600 dark:text-gray-300 mt-2">
                          {selectedPrompt.description}
                        </p>
                      )}
                      
                      {/* Metadata */}
                      <div className="flex flex-wrap items-center gap-4 mt-4 text-sm text-gray-500 dark:text-gray-400">
                        <div className="flex items-center gap-1">
                          <Folder size={14} />
                          {selectedPrompt.campaign_name}
                        </div>
                        <div className="flex items-center gap-1">
                          <TrendingUp size={14} />
                          Used {selectedPrompt.usage_count} times
                        </div>
                        {selectedPrompt.word_count && (
                          <div className="flex items-center gap-1">
                            <FileText size={14} />
                            {selectedPrompt.word_count} words
                          </div>
                        )}
                        {selectedPrompt.estimated_tokens && (
                          <div className="flex items-center gap-1">
                            <Database size={14} />
                            ~{selectedPrompt.estimated_tokens} tokens
                          </div>
                        )}
                        {selectedPrompt.effectiveness_rating && (
                          <div className="flex items-center gap-1">
                            <Star size={14} className="text-yellow-500" />
                            {selectedPrompt.effectiveness_rating}/5
                          </div>
                        )}
                      </div>
                    </div>
                    
                    <div className="flex items-center gap-2 ml-4">
                      <button
                        onClick={() => copyPromptToClipboard(selectedPrompt)}
                        className="btn-primary btn-sm"
                      >
                        <Copy size={14} className="mr-1" />
                        Copy & Use
                      </button>
                      <button
                        onClick={() => setEditingPrompt(selectedPrompt)}
                        className="btn-secondary btn-sm"
                      >
                        <Edit3 size={14} className="mr-1" />
                        Edit
                      </button>
                    </div>
                  </div>
                  
                  {/* Tags */}
                  {selectedPrompt.tags && selectedPrompt.tags.length > 0 && (
                    <div className="flex flex-wrap gap-2 mt-4">
                      {selectedPrompt.tags.map(tag => (
                        <span key={tag} className="badge badge-primary">
                          <Tag size={12} className="mr-1" />
                          {tag}
                        </span>
                      ))}
                    </div>
                  )}
                </div>
                
                {/* Prompt content */}
                <div className="flex-1 p-6 overflow-y-auto">
                  <div className="card h-full">
                    <div className="card-body h-full">
                      <pre className="whitespace-pre-wrap font-mono text-sm leading-relaxed h-full overflow-y-auto">
                        {selectedPrompt.content}
                      </pre>
                    </div>
                  </div>
                  
                  {/* Notes */}
                  {selectedPrompt.notes && (
                    <div className="mt-6">
                      <h3 className="font-medium text-gray-900 dark:text-white mb-2">Notes</h3>
                      <div className="card">
                        <div className="card-body">
                          <p className="text-gray-700 dark:text-gray-300">
                            {selectedPrompt.notes}
                          </p>
                        </div>
                      </div>
                    </div>
                  )}
                </div>
              </div>
            ) : (
              <div className="flex-1 flex items-center justify-center bg-gray-50 dark:bg-gray-900">
                <div className="text-center">
                  <FileText size={48} className="mx-auto text-gray-400 mb-4" />
                  <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">
                    Select a prompt to view
                  </h3>
                  <p className="text-gray-500 dark:text-gray-400">
                    Choose a prompt from the list to see its content and details
                  </p>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* New Campaign Modal */}
      {showNewCampaignForm && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-gray-800 rounded-lg p-6 w-full max-w-md">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold">Create New Campaign</h3>
              <button
                onClick={() => setShowNewCampaignForm(false)}
                className="btn-ghost btn-sm"
              >
                <X size={16} />
              </button>
            </div>
            
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium mb-2">Name</label>
                <input
                  type="text"
                  value={newCampaign.name}
                  onChange={(e) => setNewCampaign(prev => ({ ...prev, name: e.target.value }))}
                  className="input"
                  placeholder="e.g., Debugging, UX Design"
                />
              </div>
              
              <div>
                <label className="block text-sm font-medium mb-2">Description</label>
                <TextareaAutosize
                  value={newCampaign.description}
                  onChange={(e) => setNewCampaign(prev => ({ ...prev, description: e.target.value }))}
                  className="textarea"
                  placeholder="What kind of prompts will this campaign contain?"
                  minRows={3}
                />
              </div>
              
              <div>
                <label className="block text-sm font-medium mb-2">Color</label>
                <div className="flex gap-2 flex-wrap">
                  {campaignColors.map(color => (
                    <button
                      key={color}
                      onClick={() => setNewCampaign(prev => ({ ...prev, color }))}
                      className={`w-8 h-8 rounded-full ${newCampaign.color === color ? 'ring-2 ring-offset-2 ring-gray-400' : ''}`}
                      style={{ backgroundColor: color }}
                    />
                  ))}
                </div>
              </div>
            </div>
            
            <div className="flex justify-end gap-3 mt-6">
              <button
                onClick={() => setShowNewCampaignForm(false)}
                className="btn-secondary"
              >
                Cancel
              </button>
              <button
                onClick={createCampaign}
                disabled={!newCampaign.name}
                className="btn-primary"
              >
                Create Campaign
              </button>
            </div>
          </div>
        </div>
      )}

      {/* New Prompt Modal */}
      {showNewPromptForm && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-gray-800 rounded-lg p-6 w-full max-w-2xl max-h-[90vh] overflow-y-auto">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold">Add New Prompt</h3>
              <button
                onClick={() => setShowNewPromptForm(false)}
                className="btn-ghost btn-sm"
              >
                <X size={16} />
              </button>
            </div>
            
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium mb-2">Title</label>
                <input
                  type="text"
                  value={newPrompt.title}
                  onChange={(e) => setNewPrompt(prev => ({ ...prev, title: e.target.value }))}
                  className="input"
                  placeholder="e.g., Debug SQL Performance"
                />
              </div>
              
              <div>
                <label className="block text-sm font-medium mb-2">Campaign</label>
                <select
                  value={newPrompt.campaign_id}
                  onChange={(e) => setNewPrompt(prev => ({ ...prev, campaign_id: e.target.value }))}
                  className="input"
                >
                  <option value="">Select a campaign</option>
                  {campaigns.map(campaign => (
                    <option key={campaign.id} value={campaign.id}>
                      {campaign.name}
                    </option>
                  ))}
                </select>
              </div>
              
              <div>
                <label className="block text-sm font-medium mb-2">Content</label>
                <TextareaAutosize
                  value={newPrompt.content}
                  onChange={(e) => setNewPrompt(prev => ({ ...prev, content: e.target.value }))}
                  className="textarea font-mono"
                  placeholder="Enter your prompt content here..."
                  minRows={8}
                />
              </div>
              
              <div>
                <label className="block text-sm font-medium mb-2">Description (optional)</label>
                <TextareaAutosize
                  value={newPrompt.description}
                  onChange={(e) => setNewPrompt(prev => ({ ...prev, description: e.target.value }))}
                  className="textarea"
                  placeholder="Brief description of what this prompt does"
                  minRows={2}
                />
              </div>
            </div>
            
            <div className="flex justify-end gap-3 mt-6">
              <button
                onClick={() => setShowNewPromptForm(false)}
                className="btn-secondary"
              >
                Cancel
              </button>
              <button
                onClick={createPrompt}
                disabled={!newPrompt.title || !newPrompt.content}
                className="btn-primary"
              >
                Add Prompt
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Error toast */}
      {error && (
        <div className="fixed bottom-4 right-4 bg-red-500 text-white px-4 py-2 rounded-lg shadow-lg z-50">
          {error}
          <button
            onClick={() => setError(null)}
            className="ml-2 text-white hover:text-gray-200"
          >
            <X size={16} />
          </button>
        </div>
      )}
    </div>
  );
}

export default App;