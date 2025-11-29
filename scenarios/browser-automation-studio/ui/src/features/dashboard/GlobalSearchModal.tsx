import React, { useState, useEffect, useRef, useCallback } from 'react';
import { getConfig } from '../../config';

interface SearchResult {
  id: string;
  name: string;
  type: 'workflow' | 'project' | 'execution';
  projectId?: string;
  projectName?: string;
  workflowId?: string;
  status?: string;
  updatedAt?: Date;
}

interface GlobalSearchModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSelectWorkflow: (projectId: string, workflowId: string) => void;
  onSelectProject: (projectId: string) => void;
  onSelectExecution: (executionId: string, workflowId: string) => void;
  onRunWorkflow: (workflowId: string) => void;
}

export const GlobalSearchModal: React.FC<GlobalSearchModalProps> = ({
  isOpen,
  onClose,
  onSelectWorkflow,
  onSelectProject,
  onSelectExecution,
  onRunWorkflow,
}) => {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<SearchResult[]>([]);
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [isLoading, setIsLoading] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const resultsRef = useRef<HTMLDivElement>(null);

  // Focus input when modal opens
  useEffect(() => {
    if (isOpen) {
      setQuery('');
      setResults([]);
      setSelectedIndex(0);
      setTimeout(() => {
        inputRef.current?.focus();
      }, 50);
    }
  }, [isOpen]);

  // Search function
  const search = useCallback(async (searchQuery: string) => {
    if (!searchQuery.trim()) {
      setResults([]);
      return;
    }

    setIsLoading(true);
    try {
      const config = await getConfig();
      const searchLower = searchQuery.toLowerCase();

      // Fetch all data in parallel
      const [projectsRes, workflowsRes, executionsRes] = await Promise.all([
        fetch(`${config.API_URL}/projects`),
        fetch(`${config.API_URL}/workflows?limit=50`),
        fetch(`${config.API_URL}/executions?limit=20`),
      ]);

      const [projectsData, workflowsData, executionsData] = await Promise.all([
        projectsRes.json(),
        workflowsRes.json(),
        executionsRes.json(),
      ]);

      const projectsMap = new Map<string, string>();
      const searchResults: SearchResult[] = [];

      // Process projects
      if (Array.isArray(projectsData.projects)) {
        projectsData.projects.forEach((p: { id: string; name: string; description?: string }) => {
          projectsMap.set(p.id, p.name);
          if (p.name.toLowerCase().includes(searchLower) || (p.description ?? '').toLowerCase().includes(searchLower)) {
            searchResults.push({
              id: p.id,
              name: p.name,
              type: 'project',
            });
          }
        });
      }

      // Process workflows
      if (Array.isArray(workflowsData.workflows)) {
        workflowsData.workflows.forEach((w: { id: string; name: string; project_id?: string; projectId?: string; updated_at?: string; updatedAt?: string }) => {
          const projectId = w.project_id ?? w.projectId ?? '';
          if (w.name.toLowerCase().includes(searchLower)) {
            searchResults.push({
              id: w.id,
              name: w.name,
              type: 'workflow',
              projectId,
              projectName: projectsMap.get(projectId) ?? 'Unknown',
              updatedAt: new Date(w.updated_at ?? w.updatedAt ?? new Date().toISOString()),
            });
          }
        });
      }

      // Build workflow names map for executions
      const workflowNames = new Map<string, { name: string; projectId?: string; projectName?: string }>();
      if (Array.isArray(workflowsData.workflows)) {
        workflowsData.workflows.forEach((w: { id: string; name: string; project_id?: string; projectId?: string }) => {
          const projectId = w.project_id ?? w.projectId ?? '';
          workflowNames.set(w.id, {
            name: w.name,
            projectId,
            projectName: projectsMap.get(projectId),
          });
        });
      }

      // Process executions (only if query matches workflow name)
      if (Array.isArray(executionsData.executions)) {
        executionsData.executions.forEach((e: { id: string; workflow_id?: string; workflowId?: string; status: string }) => {
          const workflowId = e.workflow_id ?? e.workflowId ?? '';
          const workflowInfo = workflowNames.get(workflowId);
          if (workflowInfo && workflowInfo.name.toLowerCase().includes(searchLower)) {
            searchResults.push({
              id: e.id,
              name: workflowInfo.name,
              type: 'execution',
              workflowId,
              projectId: workflowInfo.projectId,
              projectName: workflowInfo.projectName,
              status: e.status,
            });
          }
        });
      }

      // Sort: workflows first, then projects, then executions
      searchResults.sort((a, b) => {
        const typeOrder = { workflow: 0, project: 1, execution: 2 };
        return typeOrder[a.type] - typeOrder[b.type];
      });

      setResults(searchResults.slice(0, 10));
      setSelectedIndex(0);
    } catch (error) {
      console.error('Search failed:', error);
      setResults([]);
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Debounced search
  useEffect(() => {
    const timer = setTimeout(() => {
      search(query);
    }, 200);
    return () => clearTimeout(timer);
  }, [query, search]);

  // Handle keyboard navigation
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (!isOpen) return;

      switch (e.key) {
        case 'Escape':
          onClose();
          break;
        case 'ArrowDown':
          e.preventDefault();
          setSelectedIndex((prev) => Math.min(prev + 1, results.length - 1));
          break;
        case 'ArrowUp':
          e.preventDefault();
          setSelectedIndex((prev) => Math.max(prev - 1, 0));
          break;
        case 'Enter':
          e.preventDefault();
          if (results[selectedIndex]) {
            handleSelect(results[selectedIndex]);
          }
          break;
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, results, selectedIndex, onClose]);

  // Scroll selected item into view
  useEffect(() => {
    if (resultsRef.current) {
      const selectedElement = resultsRef.current.children[selectedIndex] as HTMLElement;
      if (selectedElement) {
        selectedElement.scrollIntoView({ block: 'nearest' });
      }
    }
  }, [selectedIndex]);

  const handleSelect = (result: SearchResult) => {
    onClose();
    switch (result.type) {
      case 'workflow':
        if (result.projectId) {
          onSelectWorkflow(result.projectId, result.id);
        }
        break;
      case 'project':
        onSelectProject(result.id);
        break;
      case 'execution':
        if (result.workflowId) {
          onSelectExecution(result.id, result.workflowId);
        }
        break;
    }
  };

  const handleRun = (e: React.MouseEvent, result: SearchResult) => {
    e.stopPropagation();
    if (result.type === 'workflow') {
      onClose();
      onRunWorkflow(result.id);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center pt-[15vh]">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/60 backdrop-blur-sm"
        onClick={onClose}
      />

      {/* Modal */}
      <div className="relative w-full max-w-xl bg-gray-900 border border-gray-700 rounded-xl shadow-2xl overflow-hidden">
        {/* Search input */}
        <div className="flex items-center gap-3 px-4 py-3 border-b border-gray-700">
          <svg className="w-5 h-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
          <input
            ref={inputRef}
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search workflows, projects, executions..."
            className="flex-1 bg-transparent text-white placeholder-gray-500 outline-none text-base"
          />
          {isLoading && (
            <div className="w-4 h-4 border-2 border-gray-600 border-t-gray-300 rounded-full animate-spin" />
          )}
          <kbd className="px-1.5 py-0.5 text-xs text-gray-500 bg-gray-800 border border-gray-700 rounded">
            esc
          </kbd>
        </div>

        {/* Results */}
        <div ref={resultsRef} className="max-h-80 overflow-y-auto">
          {query && results.length === 0 && !isLoading && (
            <div className="px-4 py-8 text-center text-gray-500">
              No results found for "{query}"
            </div>
          )}

          {results.map((result, index) => (
            <div
              key={`${result.type}-${result.id}`}
              onClick={() => handleSelect(result)}
              className={`flex items-center gap-3 px-4 py-3 cursor-pointer transition-colors ${
                index === selectedIndex ? 'bg-gray-800' : 'hover:bg-gray-800/50'
              }`}
            >
              {/* Icon */}
              <div className={`flex items-center justify-center w-8 h-8 rounded-lg ${
                result.type === 'workflow' ? 'bg-blue-500/20 text-blue-400' :
                result.type === 'project' ? 'bg-purple-500/20 text-purple-400' :
                'bg-green-500/20 text-green-400'
              }`}>
                {result.type === 'workflow' && (
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
                  </svg>
                )}
                {result.type === 'project' && (
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
                  </svg>
                )}
                {result.type === 'execution' && (
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                )}
              </div>

              {/* Content */}
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <span className="text-white font-medium truncate">{result.name}</span>
                  <span className="text-xs text-gray-500 capitalize">{result.type}</span>
                  {result.type === 'execution' && result.status && (
                    <span className={`text-xs px-1.5 py-0.5 rounded ${
                      result.status === 'completed' ? 'bg-green-500/20 text-green-400' :
                      result.status === 'failed' ? 'bg-red-500/20 text-red-400' :
                      result.status === 'running' ? 'bg-blue-500/20 text-blue-400' :
                      'bg-gray-500/20 text-gray-400'
                    }`}>
                      {result.status}
                    </span>
                  )}
                </div>
                {result.projectName && (
                  <div className="text-xs text-gray-500 truncate">
                    in {result.projectName}
                  </div>
                )}
              </div>

              {/* Actions */}
              {result.type === 'workflow' && (
                <button
                  onClick={(e) => handleRun(e, result)}
                  className="px-2 py-1 text-xs text-green-400 hover:text-green-300 hover:bg-green-500/10 rounded transition-colors"
                  title="Run workflow"
                >
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
                  </svg>
                </button>
              )}

              <svg className="w-4 h-4 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
              </svg>
            </div>
          ))}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between px-4 py-2 border-t border-gray-700 bg-gray-800/50 text-xs text-gray-500">
          <div className="flex items-center gap-3">
            <span className="flex items-center gap-1">
              <kbd className="px-1 py-0.5 bg-gray-700 rounded">↑</kbd>
              <kbd className="px-1 py-0.5 bg-gray-700 rounded">↓</kbd>
              to navigate
            </span>
            <span className="flex items-center gap-1">
              <kbd className="px-1 py-0.5 bg-gray-700 rounded">↵</kbd>
              to select
            </span>
          </div>
          <span>
            <kbd className="px-1.5 py-0.5 bg-gray-700 rounded">⌘K</kbd> to search
          </span>
        </div>
      </div>
    </div>
  );
};
