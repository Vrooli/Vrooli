/**
 * FolderBrowser Component
 *
 * Generalized folder browser for navigating directories and selecting import targets.
 * Supports different scan modes: projects, workflows, assets, files.
 * Extracted and enhanced from FolderBrowserPanel.
 */

import { useState, useEffect, useCallback } from 'react';
import {
  FolderOpen,
  Package,
  FileCode,
  ChevronUp,
  ChevronRight,
  Check,
  Loader2,
  AlertCircle,
  RefreshCw,
} from 'lucide-react';
import { useFolderScanner } from '../hooks/useFolderScanner';
import type { FolderEntry, ScanMode } from '../types';

export interface FolderBrowserProps {
  /** Scan mode determines what we're looking for */
  mode: ScanMode;
  /** Project ID for project-scoped scans */
  projectId?: string;
  /** Callback when an import target is selected */
  onSelect: (entry: FolderEntry) => void;
  /** Callback when navigation occurs */
  onNavigate?: (path: string) => void;
  /** Initial path to start browsing */
  initialPath?: string;
  /** Whether to show already-registered items */
  showRegistered?: boolean;
  /** Scan depth */
  scanDepth?: 1 | 2;
  /** Custom class name */
  className?: string;
  /** Test ID for testing */
  testId?: string;
}

export function FolderBrowser({
  mode,
  projectId,
  onSelect,
  onNavigate,
  initialPath,
  showRegistered: initialShowRegistered = false,
  scanDepth = 1,
  className = '',
  testId,
}: FolderBrowserProps) {
  const [showRegistered, setShowRegistered] = useState(initialShowRegistered);

  const {
    isScanning,
    scanResult,
    error,
    scanFolder,
    navigateUp,
    navigateTo,
  } = useFolderScanner({
    mode,
    projectId,
    depth: scanDepth,
    initialPath,
  });

  // Initial scan on mount
  useEffect(() => {
    scanFolder(initialPath);
  }, [scanFolder, initialPath]);

  // Handle navigation with callback
  const handleNavigateTo = useCallback(
    async (path: string) => {
      await navigateTo(path);
      onNavigate?.(path);
    },
    [navigateTo, onNavigate]
  );

  const handleNavigateUp = useCallback(async () => {
    if (scanResult?.parent) {
      await navigateUp();
      onNavigate?.(scanResult.parent);
    }
  }, [navigateUp, scanResult, onNavigate]);

  // Filter entries based on showRegistered toggle
  const filteredEntries = scanResult?.entries.filter((entry) => {
    if (!entry.isRegistered) return true;
    return showRegistered;
  }) ?? [];

  // Count hidden registered items
  const hiddenCount = scanResult?.entries.filter((e) => e.isRegistered).length ?? 0;

  // Get display path (truncate if too long)
  const getDisplayPath = (path: string) => {
    const maxLength = 40;
    if (path.length <= maxLength) return path;
    return '...' + path.slice(-maxLength + 3);
  };

  // Get icon for entry based on mode
  const getEntryIcon = (entry: FolderEntry) => {
    if (!entry.isTarget) {
      return <FolderOpen size={16} className="text-blue-400" />;
    }
    switch (mode) {
      case 'projects':
        return (
          <Package
            size={16}
            className={entry.isRegistered ? 'text-gray-500' : 'text-green-400'}
          />
        );
      case 'workflows':
        return (
          <FileCode
            size={16}
            className={entry.isRegistered ? 'text-gray-500' : 'text-purple-400'}
          />
        );
      default:
        return (
          <FolderOpen
            size={16}
            className={entry.isRegistered ? 'text-gray-500' : 'text-blue-400'}
          />
        );
    }
  };

  // Get label for registered items based on mode
  const getRegisteredLabel = () => {
    switch (mode) {
      case 'projects':
        return 'Registered';
      case 'workflows':
        return 'Indexed';
      default:
        return 'Known';
    }
  };

  if (error) {
    return (
      <div
        className={`p-4 bg-red-500/10 border border-red-500/30 rounded-xl ${className}`}
        data-testid={testId}
      >
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2 text-red-400">
            <AlertCircle size={16} />
            <span className="text-sm">{error}</span>
          </div>
          <button
            type="button"
            onClick={() => scanFolder(scanResult?.path)}
            className="p-1 text-red-400 hover:text-red-300 transition-colors"
            aria-label="Retry"
          >
            <RefreshCw size={16} />
          </button>
        </div>
      </div>
    );
  }

  if (isScanning && !scanResult) {
    return (
      <div
        className={`p-6 bg-gray-800/30 border border-gray-700/50 rounded-xl flex items-center justify-center ${className}`}
        data-testid={testId}
      >
        <Loader2 size={20} className="animate-spin text-gray-400 mr-2" />
        <span className="text-gray-400 text-sm">Scanning...</span>
      </div>
    );
  }

  if (!scanResult) {
    return null;
  }

  return (
    <div
      className={`bg-gray-800/30 border border-gray-700/50 rounded-xl overflow-hidden ${className}`}
      data-testid={testId}
    >
      {/* Header with navigation */}
      <div className="flex items-center justify-between px-3 py-2 border-b border-gray-700/50 bg-gray-800/50">
        <button
          type="button"
          onClick={handleNavigateUp}
          disabled={!scanResult.parent || isScanning}
          className="flex items-center gap-1.5 px-2 py-1 text-xs text-gray-400 hover:text-white hover:bg-gray-700/50 rounded-lg transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          aria-label="Navigate up"
        >
          <ChevronUp size={14} />
          Up
        </button>
        <div
          className="flex items-center gap-1.5 text-xs text-gray-500 font-mono truncate max-w-[60%]"
          title={scanResult.path}
        >
          <FolderOpen size={12} className="flex-shrink-0" />
          {getDisplayPath(scanResult.path)}
        </div>
        {isScanning && <Loader2 size={14} className="animate-spin text-gray-500" />}
      </div>

      {/* Show registered toggle */}
      {hiddenCount > 0 && (
        <div className="px-3 py-2 border-b border-gray-700/30">
          <label className="flex items-center gap-2 text-xs text-gray-400 cursor-pointer hover:text-gray-300">
            <input
              type="checkbox"
              checked={showRegistered}
              onChange={(e) => setShowRegistered(e.target.checked)}
              className="w-3.5 h-3.5 rounded border-gray-600 bg-gray-700 text-flow-accent focus:ring-flow-accent focus:ring-offset-0"
            />
            <span>
              Show already {getRegisteredLabel().toLowerCase()} ({hiddenCount} hidden)
            </span>
          </label>
        </div>
      )}

      {/* Entry list */}
      <div className="max-h-64 overflow-y-auto">
        {filteredEntries.length === 0 ? (
          <div className="px-4 py-6 text-center text-gray-500 text-sm">
            {scanResult.entries.length === 0
              ? 'No items found'
              : `No importable items in this folder`}
          </div>
        ) : (
          filteredEntries.map((entry) => (
            <FolderEntryRow
              key={entry.path}
              entry={entry}
              icon={getEntryIcon(entry)}
              registeredLabel={getRegisteredLabel()}
              isScanning={isScanning}
              onNavigate={() => handleNavigateTo(entry.path)}
              onSelect={() => onSelect(entry)}
            />
          ))
        )}
      </div>
    </div>
  );
}

/** Individual folder entry row */
interface FolderEntryRowProps {
  entry: FolderEntry;
  icon: React.ReactNode;
  registeredLabel: string;
  isScanning: boolean;
  onNavigate: () => void;
  onSelect: () => void;
}

function FolderEntryRow({
  entry,
  icon,
  registeredLabel,
  isScanning,
  onNavigate,
  onSelect,
}: FolderEntryRowProps) {
  const isTarget = entry.isTarget;
  const isRegistered = entry.isRegistered;

  // Styling based on type and status
  const rowStyles = isTarget
    ? isRegistered
      ? 'bg-gray-800/30 border-gray-600/30 text-gray-400'
      : 'bg-green-500/5 border-green-500/20 hover:border-green-500/40'
    : 'hover:bg-gray-700/30';

  return (
    <div
      className={`flex items-center justify-between px-3 py-2 border-b border-gray-700/20 last:border-b-0 transition-colors ${rowStyles}`}
      data-path={entry.path}
      data-is-target={entry.isTarget}
      data-is-registered={entry.isRegistered}
    >
      <div className="flex items-center gap-2 min-w-0 flex-1">
        {/* Icon */}
        {icon}

        {/* Name and status */}
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <span
              className={`text-sm truncate ${
                isTarget && !isRegistered ? 'text-white font-medium' : 'text-gray-300'
              }`}
            >
              {entry.suggestedName || entry.name}
            </span>
            {isRegistered && (
              <span className="flex items-center gap-1 px-1.5 py-0.5 bg-gray-700/50 rounded text-[10px] text-gray-400">
                <Check size={10} />
                {registeredLabel}
              </span>
            )}
          </div>
          {entry.suggestedName && entry.suggestedName !== entry.name && (
            <span className="text-[10px] text-gray-500 font-mono truncate block">
              {entry.name}/
            </span>
          )}
        </div>
      </div>

      {/* Action button */}
      <div className="flex-shrink-0 ml-2">
        {isTarget ? (
          <button
            type="button"
            onClick={onSelect}
            disabled={isScanning}
            className={`px-3 py-1 text-xs font-medium rounded-lg transition-colors ${
              isRegistered
                ? 'text-gray-400 hover:text-gray-300 hover:bg-gray-700/50'
                : 'text-green-300 bg-green-500/20 hover:bg-green-500/30 border border-green-500/30'
            } disabled:opacity-50`}
          >
            Select
          </button>
        ) : (
          <button
            type="button"
            onClick={onNavigate}
            disabled={isScanning}
            className="flex items-center gap-1 px-2 py-1 text-xs text-gray-400 hover:text-white hover:bg-gray-700/50 rounded-lg transition-colors disabled:opacity-50"
          >
            Open
            <ChevronRight size={12} />
          </button>
        )}
      </div>
    </div>
  );
}

export default FolderBrowser;
