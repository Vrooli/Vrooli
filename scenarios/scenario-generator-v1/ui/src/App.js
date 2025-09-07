import React, { useState, useEffect, useCallback } from 'react';
import axios from 'axios';
import BacklogTab from './BacklogTab';
import { 
  Search, 
  Play, 
  Star, 
  Zap, 
  Code, 
  Sparkles, 
  Trophy,
  Clock,
  Users,
  Eye,
  Rocket,
  Cpu,
  Terminal,
  Brain,
  FileText,
  Database,
  Settings,
  Plus,
  RefreshCw,
  CheckCircle,
  AlertCircle,
  Loader
} from 'lucide-react';

const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';

function App() {
  const [scenarios, setScenarios] = useState([]);
  const [templates, setTemplates] = useState([]);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedScenario, setSelectedScenario] = useState(null);
  const [generatingScenario, setGeneratingScenario] = useState(false);
  const [scenarioForm, setScenarioForm] = useState({
    name: '',
    description: '',
    prompt: '',
    complexity: 'intermediate',
    category: 'business-tool'
  });
  const [activeTab, setActiveTab] = useState('browse');
  const [featuredScenarios, setFeaturedScenarios] = useState([]);
  const [error, setError] = useState(null);

  const [metadata, setMetadata] = useState(null);
  
  // Default complexity levels with icons (will be enhanced by API data)
  const getComplexityLevels = () => {
    if (!metadata) {
      return [
        { id: 'simple', name: 'Simple', icon: Cpu, color: 'text-green-500', revenue: '$10K-20K' },
        { id: 'intermediate', name: 'Intermediate', icon: Brain, color: 'text-blue-500', revenue: '$15K-35K' },
        { id: 'advanced', name: 'Advanced', icon: Rocket, color: 'text-purple-500', revenue: '$25K-50K' }
      ];
    }
    
    // Merge API data with icons
    const iconMap = { simple: Cpu, intermediate: Brain, advanced: Rocket };
    return metadata.complexity_options?.map(option => ({
      ...option,
      icon: iconMap[option.id] || Cpu
    })) || [];
  };
  
  const getCategories = () => {
    return metadata?.category_options || [
      'business-tool', 'ai-automation', 'content-marketing', 'customer-service',
      'e-commerce', 'analytics', 'document-processing', 'financial', 'healthcare',
      'education', 'productivity', 'social-media', 'project-management'
    ];
  };

  // API functions
  const fetchScenarios = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await axios.get(`${API_URL}/api/scenarios`);
      setScenarios(response.data || []);
    } catch (err) {
      setError('Failed to load scenarios. Make sure the API is running.');
      console.error('Error fetching scenarios:', err);
    } finally {
      setLoading(false);
    }
  }, []);

  const fetchTemplates = useCallback(async () => {
    try {
      const response = await axios.get(`${API_URL}/api/templates`);
      setTemplates(response.data || []);
    } catch (err) {
      console.error('Error fetching templates:', err);
    }
  }, []);

  const fetchFeaturedScenarios = useCallback(async () => {
    try {
      const response = await axios.get(`${API_URL}/api/featured`);
      setFeaturedScenarios(response.data || []);
    } catch (err) {
      console.error('Error fetching featured scenarios:', err);
    }
  }, []);

  const searchScenarios = useCallback(async (query) => {
    if (!query.trim()) {
      fetchScenarios();
      return;
    }
    
    try {
      setLoading(true);
      const response = await axios.get(`${API_URL}/api/search/scenarios?q=${encodeURIComponent(query)}`);
      setScenarios(response.data || []);
    } catch (err) {
      setError('Search failed. Please try again.');
      console.error('Error searching scenarios:', err);
    } finally {
      setLoading(false);
    }
  }, [fetchScenarios]);

  const generateScenario = async () => {
    try {
      setGeneratingScenario(true);
      setError(null);
      
      // Validate using API endpoint
      const validationResponse = await axios.post(`${API_URL}/api/validate/scenario`, scenarioForm);
      const validation = validationResponse.data;
      
      if (!validation.valid) {
        setError(validation.errors.join(', '));
        return;
      }
      
      // Show warnings if any
      if (validation.warnings && validation.warnings.length > 0) {
        console.warn('Validation warnings:', validation.warnings);
      }
      
      const response = await axios.post(`${API_URL}/api/generate`, scenarioForm);
      
      if (response.data) {
        setScenarioForm({
          name: '',
          description: '',
          prompt: '',
          complexity: 'intermediate',
          category: 'business-tool'
        });
        
        // Switch to browse tab and refresh scenarios
        setActiveTab('browse');
        fetchScenarios();
        
        alert(`Scenario generation started! Generation ID: ${response.data.generation_id}`);
      }
    } catch (err) {
      if (err.response && err.response.data) {
        setError(err.response.data);
      } else {
        setError('Failed to generate scenario. Please try again.');
      }
      console.error('Error generating scenario:', err);
    } finally {
      setGeneratingScenario(false);
    }
  };

  const viewScenario = async (scenarioId) => {
    try {
      const response = await axios.get(`${API_URL}/api/scenarios/${scenarioId}`);
      setSelectedScenario(response.data);
    } catch (err) {
      setError('Failed to load scenario details');
      console.error('Error fetching scenario:', err);
    }
  };

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
    
    fetchScenarios();
    fetchTemplates();
    fetchFeaturedScenarios();
    fetchMetadata();
  }, [fetchScenarios, fetchTemplates, fetchFeaturedScenarios]);

  useEffect(() => {
    const delayedSearch = setTimeout(() => {
      if (searchTerm) {
        searchScenarios(searchTerm);
      } else {
        fetchScenarios();
      }
    }, 300);

    return () => clearTimeout(delayedSearch);
  }, [searchTerm, searchScenarios, fetchScenarios]);

  const getStatusIcon = (status) => {
    switch (status) {
      case 'completed':
        return <CheckCircle className="h-5 w-5 text-green-500" />;
      case 'generating':
        return <Loader className="h-5 w-5 text-blue-500 animate-spin" />;
      case 'failed':
        return <AlertCircle className="h-5 w-5 text-red-500" />;
      default:
        return <Clock className="h-5 w-5 text-yellow-500" />;
    }
  };

  const getComplexityColor = (complexity) => {
    const level = getComplexityLevels().find(l => l.id === complexity);
    return level ? level.color : 'text-gray-500';
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-gray-900 via-blue-900 to-purple-900">
      <div className="container mx-auto px-4 py-8">
        {/* Header */}
        <div className="text-center mb-8">
          <div className="flex items-center justify-center mb-4">
            <Brain className="h-12 w-12 text-cyan-400 mr-4" />
            <h1 className="text-4xl font-bold text-white">
              Scenario Generator
            </h1>
          </div>
          <p className="text-gray-300 text-lg">
            AI-Powered Vrooli Scenario Creation Platform
          </p>
        </div>

        {/* Error Display */}
        {error && (
          <div className="bg-red-900/50 border border-red-500 text-red-200 px-4 py-3 rounded mb-6">
            <div className="flex items-center">
              <AlertCircle className="h-5 w-5 mr-2" />
              {error}
            </div>
          </div>
        )}

        {/* Navigation Tabs */}
        <div className="flex space-x-1 mb-8 bg-gray-800/50 p-1 rounded-lg">
          {[
            { id: 'backlog', name: 'Backlog', icon: Clock },
            { id: 'browse', name: 'Browse Scenarios', icon: Eye },
            { id: 'generate', name: 'Generate New', icon: Plus },
            { id: 'templates', name: 'Templates', icon: FileText }
          ].map((tab) => {
            const Icon = tab.icon;
            return (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`flex-1 flex items-center justify-center px-4 py-3 rounded-md transition-all ${
                  activeTab === tab.id
                    ? 'bg-cyan-600 text-white shadow-lg'
                    : 'text-gray-300 hover:text-white hover:bg-gray-700'
                }`}
              >
                <Icon className="h-5 w-5 mr-2" />
                {tab.name}
              </button>
            );
          })}
        </div>

        {/* Backlog Tab */}
        {activeTab === 'backlog' && (
          <BacklogTab />
        )}

        {/* Browse Tab */}
        {activeTab === 'browse' && (
          <div>
            {/* Search Bar */}
            <div className="relative mb-6">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 h-5 w-5" />
              <input
                type="text"
                placeholder="Search scenarios..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="w-full pl-10 pr-4 py-3 bg-gray-800 border border-gray-600 rounded-lg text-white placeholder-gray-400 focus:outline-none focus:border-cyan-500"
              />
            </div>

            {/* Featured Scenarios */}
            {featuredScenarios.length > 0 && (
              <div className="mb-8">
                <h2 className="text-2xl font-bold text-white mb-4 flex items-center">
                  <Star className="h-6 w-6 text-yellow-500 mr-2" />
                  Featured Scenarios
                </h2>
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                  {featuredScenarios.slice(0, 6).map((scenario) => (
                    <ScenarioCard key={scenario.id} scenario={scenario} onView={viewScenario} />
                  ))}
                </div>
              </div>
            )}

            {/* All Scenarios */}
            <div className="mb-8">
              <div className="flex items-center justify-between mb-4">
                <h2 className="text-2xl font-bold text-white">
                  All Scenarios ({scenarios.length})
                </h2>
                <button
                  onClick={fetchScenarios}
                  className="flex items-center px-4 py-2 bg-gray-700 text-white rounded-lg hover:bg-gray-600"
                >
                  <RefreshCw className="h-4 w-4 mr-2" />
                  Refresh
                </button>
              </div>

              {loading ? (
                <div className="flex justify-center py-12">
                  <Loader className="h-8 w-8 text-cyan-400 animate-spin" />
                </div>
              ) : scenarios.length > 0 ? (
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                  {scenarios.map((scenario) => (
                    <ScenarioCard key={scenario.id} scenario={scenario} onView={viewScenario} />
                  ))}
                </div>
              ) : (
                <div className="text-center py-12 text-gray-400">
                  <Brain className="h-16 w-16 mx-auto mb-4 opacity-50" />
                  <p className="text-lg">No scenarios found</p>
                  <p>Try adjusting your search or generate a new scenario</p>
                </div>
              )}
            </div>
          </div>
        )}

        {/* Generate Tab */}
        {activeTab === 'generate' && (
          <div className="max-w-2xl mx-auto">
            <h2 className="text-2xl font-bold text-white mb-6 flex items-center">
              <Sparkles className="h-6 w-6 text-cyan-400 mr-2" />
              Generate New Scenario
            </h2>

            <div className="bg-gray-800/50 p-6 rounded-lg space-y-6">
              <div>
                <label className="block text-white font-medium mb-2">
                  Scenario Name *
                </label>
                <input
                  type="text"
                  value={scenarioForm.name}
                  onChange={(e) => setScenarioForm({...scenarioForm, name: e.target.value})}
                  placeholder="e.g., Customer Support Dashboard"
                  className="w-full px-4 py-3 bg-gray-700 border border-gray-600 rounded-lg text-white placeholder-gray-400 focus:outline-none focus:border-cyan-500"
                />
              </div>

              <div>
                <label className="block text-white font-medium mb-2">
                  Description *
                </label>
                <input
                  type="text"
                  value={scenarioForm.description}
                  onChange={(e) => setScenarioForm({...scenarioForm, description: e.target.value})}
                  placeholder="Brief description of what this scenario does"
                  className="w-full px-4 py-3 bg-gray-700 border border-gray-600 rounded-lg text-white placeholder-gray-400 focus:outline-none focus:border-cyan-500"
                />
              </div>

              <div>
                <label className="block text-white font-medium mb-2">
                  Detailed Prompt *
                </label>
                <textarea
                  value={scenarioForm.prompt}
                  onChange={(e) => setScenarioForm({...scenarioForm, prompt: e.target.value})}
                  placeholder="Describe in detail what you want this scenario to do. Include specific features, target users, and business requirements."
                  rows={6}
                  className="w-full px-4 py-3 bg-gray-700 border border-gray-600 rounded-lg text-white placeholder-gray-400 focus:outline-none focus:border-cyan-500"
                />
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div>
                  <label className="block text-white font-medium mb-2">
                    Complexity Level
                  </label>
                  <select
                    value={scenarioForm.complexity}
                    onChange={(e) => setScenarioForm({...scenarioForm, complexity: e.target.value})}
                    className="w-full px-4 py-3 bg-gray-700 border border-gray-600 rounded-lg text-white focus:outline-none focus:border-cyan-500"
                  >
                    {getComplexityLevels().map((level) => (
                      <option key={level.id} value={level.id}>
                        {level.name} ({level.revenue})
                      </option>
                    ))}
                  </select>
                </div>

                <div>
                  <label className="block text-white font-medium mb-2">
                    Category
                  </label>
                  <select
                    value={scenarioForm.category}
                    onChange={(e) => setScenarioForm({...scenarioForm, category: e.target.value})}
                    className="w-full px-4 py-3 bg-gray-700 border border-gray-600 rounded-lg text-white focus:outline-none focus:border-cyan-500"
                  >
                    {getCategories().map((category) => (
                      <option key={category} value={category}>
                        {category.replace('-', ' ').replace(/\b\w/g, l => l.toUpperCase())}
                      </option>
                    ))}
                  </select>
                </div>
              </div>

              <button
                onClick={generateScenario}
                disabled={generatingScenario}
                className="w-full flex items-center justify-center px-6 py-3 bg-gradient-to-r from-cyan-600 to-blue-600 text-white font-bold rounded-lg hover:from-cyan-700 hover:to-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {generatingScenario ? (
                  <>
                    <Loader className="h-5 w-5 mr-2 animate-spin" />
                    Generating Scenario...
                  </>
                ) : (
                  <>
                    <Rocket className="h-5 w-5 mr-2" />
                    Generate Scenario
                  </>
                )}
              </button>
            </div>
          </div>
        )}

        {/* Templates Tab */}
        {activeTab === 'templates' && (
          <div>
            <h2 className="text-2xl font-bold text-white mb-6 flex items-center">
              <FileText className="h-6 w-6 text-cyan-400 mr-2" />
              Scenario Templates
            </h2>

            {templates.length > 0 ? (
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {templates.map((template) => (
                  <TemplateCard key={template.id} template={template} />
                ))}
              </div>
            ) : (
              <div className="text-center py-12 text-gray-400">
                <FileText className="h-16 w-16 mx-auto mb-4 opacity-50" />
                <p className="text-lg">No templates available</p>
                <p>Templates will appear here once loaded</p>
              </div>
            )}
          </div>
        )}

        {/* Scenario Modal */}
        {selectedScenario && (
          <ScenarioModal scenario={selectedScenario} onClose={() => setSelectedScenario(null)} />
        )}
      </div>
    </div>
  );
}

// Scenario Card Component
function ScenarioCard({ scenario, onView }) {
  return (
    <div className="bg-gray-800/70 border border-gray-600 rounded-lg p-4 hover:border-cyan-500 transition-all">
      <div className="flex items-start justify-between mb-3">
        <h3 className="text-lg font-semibold text-white truncate">
          {scenario.name}
        </h3>
        <div className="flex items-center ml-2">
          {getStatusIcon(scenario.status)}
        </div>
      </div>
      
      <p className="text-gray-300 text-sm mb-3 line-clamp-2">
        {scenario.description}
      </p>
      
      <div className="flex items-center justify-between text-sm">
        <div className="flex items-center space-x-2">
          <span className={`font-medium ${getComplexityColor(scenario.complexity)}`}>
            {scenario.complexity}
          </span>
          <span className="text-gray-400">
            ${scenario.estimated_revenue?.toLocaleString()}
          </span>
        </div>
        
        <button
          onClick={() => onView(scenario.id)}
          className="flex items-center text-cyan-400 hover:text-cyan-300"
        >
          <Eye className="h-4 w-4 mr-1" />
          View
        </button>
      </div>
    </div>
  );
}

// Template Card Component
function TemplateCard({ template }) {
  return (
    <div className="bg-gray-800/70 border border-gray-600 rounded-lg p-4 hover:border-cyan-500 transition-all">
      <h3 className="text-lg font-semibold text-white mb-2">
        {template.name}
      </h3>
      
      <p className="text-gray-300 text-sm mb-3">
        {template.description}
      </p>
      
      <div className="flex items-center justify-between text-sm">
        <span className="text-gray-400">
          {template.category.replace('-', ' ')}
        </span>
        
        <span className="text-cyan-400">
          ${template.estimated_revenue_min?.toLocaleString()} - ${template.estimated_revenue_max?.toLocaleString()}
        </span>
      </div>
    </div>
  );
}

// Scenario Modal Component
function ScenarioModal({ scenario, onClose }) {
  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
      <div className="bg-gray-800 rounded-lg max-w-4xl w-full max-h-[90vh] overflow-y-auto">
        <div className="p-6">
          <div className="flex items-start justify-between mb-4">
            <h2 className="text-2xl font-bold text-white">
              {scenario.name}
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
              <p className="text-gray-300">{scenario.description}</p>
            </div>
            
            <div>
              <h3 className="text-lg font-semibold text-white mb-2">Status</h3>
              <div className="flex items-center">
                {getStatusIcon(scenario.status)}
                <span className="ml-2 text-gray-300">{scenario.status}</span>
              </div>
            </div>
            
            <div>
              <h3 className="text-lg font-semibold text-white mb-2">Details</h3>
              <div className="grid grid-cols-2 gap-4 text-sm">
                <div>
                  <span className="text-gray-400">Complexity:</span>
                  <span className="ml-2 text-white">{scenario.complexity}</span>
                </div>
                <div>
                  <span className="text-gray-400">Category:</span>
                  <span className="ml-2 text-white">{scenario.category}</span>
                </div>
                <div>
                  <span className="text-gray-400">Revenue Estimate:</span>
                  <span className="ml-2 text-white">${scenario.estimated_revenue?.toLocaleString()}</span>
                </div>
                <div>
                  <span className="text-gray-400">Created:</span>
                  <span className="ml-2 text-white">{new Date(scenario.created_at).toLocaleDateString()}</span>
                </div>
              </div>
            </div>
            
            {scenario.prompt && (
              <div>
                <h3 className="text-lg font-semibold text-white mb-2">Original Prompt</h3>
                <p className="text-gray-300 bg-gray-900/50 p-3 rounded">
                  {scenario.prompt}
                </p>
              </div>
            )}
            
            {scenario.claude_response && (
              <div>
                <h3 className="text-lg font-semibold text-white mb-2">Claude Output</h3>
                <pre className="text-gray-300 bg-gray-900/50 p-3 rounded text-sm overflow-x-auto max-h-60">
                  {scenario.claude_response.substring(0, 1000)}
                  {scenario.claude_response.length > 1000 && '...'}
                </pre>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

export default App;