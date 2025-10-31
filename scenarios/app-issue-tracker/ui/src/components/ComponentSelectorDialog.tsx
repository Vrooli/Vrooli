import { useCallback, useEffect, useMemo, useState } from 'react';
import { Check, Loader2, RefreshCw, Search, X } from 'lucide-react';
import { Modal } from './Modal';
import type { Target } from '../types/issue';
import type { Component } from '../types/component';
import { useComponentStore } from '../stores/componentStore';
import { resolveApiBase } from '@vrooli/api-base';
import '../styles/component-selector.css';

declare const __API_PORT__: string | undefined;

const FALLBACK_API_PORT =
  typeof __API_PORT__ === 'string' && __API_PORT__.trim().length > 0 ? __API_PORT__ : '15000';

const API_BASE_URL = resolveApiBase({
  explicitUrl: import.meta.env.VITE_API_BASE_URL as string | undefined,
  defaultPort: FALLBACK_API_PORT,
  appendSuffix: true,
});

interface ComponentSelectorDialogProps {
  selectedTargets: Target[];
  onSelectTargets: (targets: Target[]) => void;
  onClose: () => void;
}

export function ComponentSelectorDialog({
  selectedTargets,
  onSelectTargets,
  onClose,
}: ComponentSelectorDialogProps) {
  const { scenarios, resources, loading, error, fetchComponents } = useComponentStore();
  const [searchQuery, setSearchQuery] = useState('');
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [localSelectedTargets, setLocalSelectedTargets] = useState<Target[]>(selectedTargets);

  // Sync with external changes
  useEffect(() => {
    setLocalSelectedTargets(selectedTargets);
  }, [selectedTargets]);

  const isTargetSelected = useCallback(
    (type: 'scenario' | 'resource', id: string): boolean => {
      return localSelectedTargets.some((t) => t.type === type && t.id === id);
    },
    [localSelectedTargets],
  );

  const toggleTarget = useCallback(
    (component: Component) => {
      const target: Target = {
        type: component.type,
        id: component.id,
        name: component.display_name,
      };

      setLocalSelectedTargets((prev) => {
        const isSelected = prev.some((t) => t.type === target.type && t.id === target.id);

        if (isSelected) {
          return prev.filter((t) => !(t.type === target.type && t.id === target.id));
        } else {
          return [...prev, target];
        }
      });
    },
    [],
  );

  const filteredScenarios = useMemo(() => {
    if (!searchQuery.trim()) return scenarios;

    const query = searchQuery.toLowerCase();
    return scenarios.filter(
      (s) =>
        s.id.toLowerCase().includes(query) ||
        s.display_name.toLowerCase().includes(query) ||
        (s.description && s.description.toLowerCase().includes(query)),
    );
  }, [scenarios, searchQuery]);

  const filteredResources = useMemo(() => {
    if (!searchQuery.trim()) return resources;

    const query = searchQuery.toLowerCase();
    return resources.filter(
      (r) =>
        r.id.toLowerCase().includes(query) ||
        r.display_name.toLowerCase().includes(query) ||
        (r.description && r.description.toLowerCase().includes(query)),
    );
  }, [resources, searchQuery]);

  const handleConfirm = useCallback(() => {
    onSelectTargets(localSelectedTargets);
    onClose();
  }, [localSelectedTargets, onClose, onSelectTargets]);

  const handleCancel = useCallback(() => {
    setLocalSelectedTargets(selectedTargets);
    onClose();
  }, [onClose, selectedTargets]);

  const handleRefresh = useCallback(async () => {
    setIsRefreshing(true);
    try {
      await fetchComponents(API_BASE_URL);
    } finally {
      setIsRefreshing(false);
    }
  }, [fetchComponents]);

  const hasResults = filteredScenarios.length > 0 || filteredResources.length > 0;

  return (
    <Modal onClose={handleCancel}>
      <div className="component-selector-dialog">
        <div className="component-selector-header">
          <h2 className="component-selector-title">Select Targets</h2>
          <div className="component-selector-header-actions">
            <button
              type="button"
              className="component-selector-refresh"
              onClick={handleRefresh}
              disabled={isRefreshing || loading}
              aria-label="Refresh components"
              title="Refresh components"
            >
              <RefreshCw size={18} className={isRefreshing ? 'spinning' : ''} />
            </button>
            <button
              type="button"
              className="component-selector-close"
              onClick={handleCancel}
              aria-label="Close"
            >
              <X size={20} />
            </button>
          </div>
        </div>

        <div className="component-selector-search">
          <Search size={18} className="component-selector-search-icon" />
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search scenarios and resources..."
            className="component-selector-search-input"
            autoFocus
          />
          {searchQuery && (
            <button
              type="button"
              className="component-selector-search-clear"
              onClick={() => setSearchQuery('')}
              aria-label="Clear search"
            >
              <X size={16} />
            </button>
          )}
        </div>

        {loading && (
          <div className="component-selector-loading">
            <Loader2 size={24} className="component-selector-spinner" />
            <p>Loading components...</p>
          </div>
        )}

        {error && (
          <div className="component-selector-error">
            <p className="component-selector-error-message">{error}</p>
            <p className="component-selector-error-hint">
              Make sure the API server is running and try again.
            </p>
          </div>
        )}

        {!loading && !error && (
          <div className="component-selector-content">
            {!hasResults && (
              <div className="component-selector-empty">
                <p>No components found matching "{searchQuery}"</p>
              </div>
            )}

            {filteredScenarios.length > 0 && (
              <div className="component-selector-group">
                <h3 className="component-selector-group-title">
                  Scenarios ({filteredScenarios.length})
                </h3>
                <div className="component-selector-list">
                  {filteredScenarios.map((scenario) => {
                    const selected = isTargetSelected('scenario', scenario.id);
                    return (
                      <button
                        key={`scenario-${scenario.id}`}
                        type="button"
                        className={`component-selector-item ${selected ? 'is-selected' : ''}`.trim()}
                        onClick={() => toggleTarget(scenario)}
                      >
                        <div className="component-selector-item-checkbox">
                          {selected && <Check size={16} />}
                        </div>
                        <div className="component-selector-item-content">
                          <div className="component-selector-item-name">{scenario.display_name}</div>
                          <div className="component-selector-item-id">scenario:{scenario.id}</div>
                        </div>
                      </button>
                    );
                  })}
                </div>
              </div>
            )}

            {filteredResources.length > 0 && (
              <div className="component-selector-group">
                <h3 className="component-selector-group-title">
                  Resources ({filteredResources.length})
                </h3>
                <div className="component-selector-list">
                  {filteredResources.map((resource) => {
                    const selected = isTargetSelected('resource', resource.id);
                    return (
                      <button
                        key={`resource-${resource.id}`}
                        type="button"
                        className={`component-selector-item ${selected ? 'is-selected' : ''}`.trim()}
                        onClick={() => toggleTarget(resource)}
                      >
                        <div className="component-selector-item-checkbox">
                          {selected && <Check size={16} />}
                        </div>
                        <div className="component-selector-item-content">
                          <div className="component-selector-item-name">{resource.display_name}</div>
                          <div className="component-selector-item-id">resource:{resource.id}</div>
                        </div>
                      </button>
                    );
                  })}
                </div>
              </div>
            )}
          </div>
        )}

        <div className="component-selector-footer">
          <div className="component-selector-selected-count">
            {localSelectedTargets.length} target{localSelectedTargets.length !== 1 ? 's' : ''} selected
          </div>
          <div className="component-selector-actions">
            <button
              type="button"
              className="button button--secondary"
              onClick={handleCancel}
            >
              Cancel
            </button>
            <button
              type="button"
              className="button button--primary"
              onClick={handleConfirm}
              disabled={localSelectedTargets.length === 0}
            >
              Confirm Selection
            </button>
          </div>
        </div>
      </div>
    </Modal>
  );
}
