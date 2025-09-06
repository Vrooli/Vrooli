import React, { useState, useEffect } from 'react';
import { useQuery, useMutation, useQueryClient } from 'react-query';
import { DragDropContext, Droppable, Draggable } from 'react-beautiful-dnd';
import toast from 'react-hot-toast';
import { FiPlus, FiSearch, FiFilter, FiRefreshCw, FiSettings, FiLogIn } from 'react-icons/fi';
import { BiUpvote, BiDownvote } from 'react-icons/bi';
import { MdDragIndicator } from 'react-icons/md';
import api from './services/api';
import KanbanBoard from './components/KanbanBoard';
import FeatureRequestModal from './components/FeatureRequestModal';
import ScenarioSelector from './components/ScenarioSelector';
import CreateRequestModal from './components/CreateRequestModal';
import AuthModal from './components/AuthModal';

const STATUSES = [
  { id: 'proposed', label: 'Proposed', color: 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200' },
  { id: 'under_review', label: 'Under Review', color: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200' },
  { id: 'in_development', label: 'In Development', color: 'bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200' },
  { id: 'shipped', label: 'Shipped', color: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200' },
  { id: 'wont_fix', label: "Won't Fix", color: 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300' }
];

function App() {
  const [selectedScenario, setSelectedScenario] = useState(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [filterPriority, setFilterPriority] = useState('all');
  const [sortBy, setSortBy] = useState('votes');
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [selectedRequest, setSelectedRequest] = useState(null);
  const [isAuthModalOpen, setIsAuthModalOpen] = useState(false);
  const [user, setUser] = useState(null);
  const [darkMode, setDarkMode] = useState(false);

  const queryClient = useQueryClient();

  // Load dark mode preference
  useEffect(() => {
    const isDark = localStorage.getItem('darkMode') === 'true';
    setDarkMode(isDark);
    if (isDark) {
      document.documentElement.classList.add('dark');
    }
  }, []);

  // Toggle dark mode
  const toggleDarkMode = () => {
    const newDarkMode = !darkMode;
    setDarkMode(newDarkMode);
    localStorage.setItem('darkMode', newDarkMode.toString());
    if (newDarkMode) {
      document.documentElement.classList.add('dark');
    } else {
      document.documentElement.classList.remove('dark');
    }
  };

  // Load user from localStorage
  useEffect(() => {
    const storedUser = localStorage.getItem('user');
    if (storedUser) {
      setUser(JSON.parse(storedUser));
    }
  }, []);

  // Fetch scenarios
  const { data: scenarios = [], isLoading: loadingScenarios } = useQuery(
    'scenarios',
    api.getScenarios,
    {
      onSuccess: (data) => {
        if (data.length > 0 && !selectedScenario) {
          setSelectedScenario(data[0]);
        }
      }
    }
  );

  // Fetch feature requests
  const { data: featureRequests = [], isLoading: loadingRequests, refetch } = useQuery(
    ['featureRequests', selectedScenario?.id, sortBy],
    () => api.getFeatureRequests(selectedScenario.id, { sort: sortBy }),
    {
      enabled: !!selectedScenario,
    }
  );

  // Group requests by status
  const groupedRequests = React.useMemo(() => {
    const grouped = {};
    STATUSES.forEach(status => {
      grouped[status.id] = [];
    });

    let filtered = featureRequests;

    // Apply search filter
    if (searchTerm) {
      filtered = filtered.filter(req =>
        req.title.toLowerCase().includes(searchTerm.toLowerCase()) ||
        req.description.toLowerCase().includes(searchTerm.toLowerCase())
      );
    }

    // Apply priority filter
    if (filterPriority !== 'all') {
      filtered = filtered.filter(req => req.priority === filterPriority);
    }

    // Group by status
    filtered.forEach(request => {
      if (grouped[request.status]) {
        grouped[request.status].push(request);
      }
    });

    // Sort within each status
    Object.keys(grouped).forEach(status => {
      grouped[status].sort((a, b) => {
        if (sortBy === 'votes') return b.vote_count - a.vote_count;
        if (sortBy === 'date') return new Date(b.created_at) - new Date(a.created_at);
        if (sortBy === 'priority') {
          const priorityOrder = { critical: 4, high: 3, medium: 2, low: 1 };
          return priorityOrder[b.priority] - priorityOrder[a.priority];
        }
        return 0;
      });
    });

    return grouped;
  }, [featureRequests, searchTerm, filterPriority, sortBy]);

  // Vote mutation
  const voteMutation = useMutation(
    ({ requestId, value }) => api.voteOnRequest(requestId, value),
    {
      onSuccess: () => {
        queryClient.invalidateQueries(['featureRequests']);
        toast.success('Vote recorded!');
      },
      onError: () => {
        toast.error('Failed to record vote');
      }
    }
  );

  // Update request mutation
  const updateRequestMutation = useMutation(
    ({ requestId, updates }) => api.updateFeatureRequest(requestId, updates),
    {
      onSuccess: () => {
        queryClient.invalidateQueries(['featureRequests']);
        toast.success('Request updated!');
      },
      onError: () => {
        toast.error('Failed to update request');
      }
    }
  );

  // Handle drag end
  const onDragEnd = (result) => {
    if (!result.destination) return;

    const { source, destination, draggableId } = result;

    if (source.droppableId !== destination.droppableId) {
      // Status change
      updateRequestMutation.mutate({
        requestId: draggableId,
        updates: { status: destination.droppableId }
      });
    } else if (source.index !== destination.index) {
      // Position change within same status
      updateRequestMutation.mutate({
        requestId: draggableId,
        updates: { position: destination.index }
      });
    }
  };

  // Handle vote
  const handleVote = (requestId, value) => {
    if (!user && selectedScenario?.auth_config?.mode === 'authenticated') {
      toast.error('Please sign in to vote');
      setIsAuthModalOpen(true);
      return;
    }
    voteMutation.mutate({ requestId, value });
  };

  // Handle request creation
  const handleCreateRequest = (requestData) => {
    if (!user && selectedScenario?.auth_config?.mode === 'authenticated') {
      toast.error('Please sign in to create requests');
      setIsAuthModalOpen(true);
      return;
    }
    
    api.createFeatureRequest({
      ...requestData,
      scenario_id: selectedScenario.id
    }).then(() => {
      toast.success('Feature request created!');
      setIsCreateModalOpen(false);
      refetch();
    }).catch(() => {
      toast.error('Failed to create request');
    });
  };

  // Handle auth
  const handleAuth = (userData) => {
    setUser(userData);
    localStorage.setItem('user', JSON.stringify(userData));
    api.setAuthToken(userData.token);
    setIsAuthModalOpen(false);
    toast.success(`Welcome, ${userData.display_name}!`);
  };

  const handleLogout = () => {
    setUser(null);
    localStorage.removeItem('user');
    api.clearAuthToken();
    toast.success('Logged out successfully');
  };

  if (loadingScenarios) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="loading-dots text-primary-600">
          <span></span>
          <span></span>
          <span></span>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
      {/* Header */}
      <header className="bg-white dark:bg-gray-800 shadow-sm border-b border-gray-200 dark:border-gray-700">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-16">
            <div className="flex items-center space-x-4">
              <h1 className="text-xl font-bold text-gray-900 dark:text-white">
                Feature Voting
              </h1>
              <ScenarioSelector
                scenarios={scenarios}
                selectedScenario={selectedScenario}
                onSelect={setSelectedScenario}
              />
            </div>

            <div className="flex items-center space-x-4">
              {/* Search */}
              <div className="relative">
                <input
                  type="text"
                  placeholder="Search requests..."
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  className="pl-10 pr-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
                />
                <FiSearch className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400" />
              </div>

              {/* Filters */}
              <select
                value={filterPriority}
                onChange={(e) => setFilterPriority(e.target.value)}
                className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
              >
                <option value="all">All Priorities</option>
                <option value="critical">Critical</option>
                <option value="high">High</option>
                <option value="medium">Medium</option>
                <option value="low">Low</option>
              </select>

              <select
                value={sortBy}
                onChange={(e) => setSortBy(e.target.value)}
                className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
              >
                <option value="votes">Most Voted</option>
                <option value="date">Newest</option>
                <option value="priority">Priority</option>
              </select>

              {/* Actions */}
              <button
                onClick={refetch}
                className="p-2 text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white"
                title="Refresh"
              >
                <FiRefreshCw />
              </button>

              <button
                onClick={toggleDarkMode}
                className="p-2 text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white"
                title="Toggle dark mode"
              >
                <FiSettings />
              </button>

              {user ? (
                <div className="flex items-center space-x-2">
                  <span className="text-sm text-gray-600 dark:text-gray-400">
                    {user.display_name}
                  </span>
                  <button
                    onClick={handleLogout}
                    className="text-sm text-red-600 dark:text-red-400 hover:underline"
                  >
                    Logout
                  </button>
                </div>
              ) : (
                <button
                  onClick={() => setIsAuthModalOpen(true)}
                  className="flex items-center space-x-2 px-4 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700"
                >
                  <FiLogIn />
                  <span>Sign In</span>
                </button>
              )}

              <button
                onClick={() => setIsCreateModalOpen(true)}
                className="flex items-center space-x-2 px-4 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700"
              >
                <FiPlus />
                <span>New Request</span>
              </button>
            </div>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {selectedScenario ? (
          <div>
            {/* Scenario Info */}
            <div className="mb-6">
              <h2 className="text-2xl font-bold text-gray-900 dark:text-white mb-2">
                {selectedScenario.display_name}
              </h2>
              {selectedScenario.description && (
                <p className="text-gray-600 dark:text-gray-400">
                  {selectedScenario.description}
                </p>
              )}
              {selectedScenario.stats && (
                <div className="flex space-x-4 mt-2 text-sm text-gray-500 dark:text-gray-400">
                  <span>Total: {selectedScenario.stats.total_requests}</span>
                  <span>Open: {selectedScenario.stats.open_requests}</span>
                  <span>Shipped: {selectedScenario.stats.shipped_requests}</span>
                </div>
              )}
            </div>

            {/* Kanban Board */}
            {loadingRequests ? (
              <div className="flex items-center justify-center h-64">
                <div className="loading-dots text-primary-600">
                  <span></span>
                  <span></span>
                  <span></span>
                </div>
              </div>
            ) : (
              <KanbanBoard
                statuses={STATUSES}
                groupedRequests={groupedRequests}
                onDragEnd={onDragEnd}
                onVote={handleVote}
                onRequestClick={setSelectedRequest}
                user={user}
              />
            )}
          </div>
        ) : (
          <div className="text-center py-12">
            <p className="text-gray-500 dark:text-gray-400">
              No scenarios available. Please create a scenario first.
            </p>
          </div>
        )}
      </main>

      {/* Modals */}
      {isCreateModalOpen && (
        <CreateRequestModal
          isOpen={isCreateModalOpen}
          onClose={() => setIsCreateModalOpen(false)}
          onCreate={handleCreateRequest}
        />
      )}

      {selectedRequest && (
        <FeatureRequestModal
          request={selectedRequest}
          isOpen={!!selectedRequest}
          onClose={() => setSelectedRequest(null)}
          onVote={handleVote}
          onUpdate={(updates) => {
            updateRequestMutation.mutate({
              requestId: selectedRequest.id,
              updates
            });
            setSelectedRequest(null);
          }}
          user={user}
        />
      )}

      {isAuthModalOpen && (
        <AuthModal
          isOpen={isAuthModalOpen}
          onClose={() => setIsAuthModalOpen(false)}
          onAuth={handleAuth}
        />
      )}
    </div>
  );
}

export default App;