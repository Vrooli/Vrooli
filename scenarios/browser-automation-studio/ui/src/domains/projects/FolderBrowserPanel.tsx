/**
 * FolderBrowserPanel
 *
 * A folder browser component for navigating directories and selecting projects to import.
 * Shows projects (both importable and already-registered) and regular folders for navigation.
 */

import { useState } from 'react';
import {
  FolderOpen,
  Package,
  ChevronUp,
  Check,
  Loader2,
  AlertCircle,
  ChevronRight,
} from 'lucide-react';
import { selectors } from '@constants/selectors';
import type { FolderEntry, ScanResult } from './hooks/useFolderBrowser';

interface FolderBrowserPanelProps {
  scanResult: ScanResult | null;
  isScanning: boolean;
  error: string | null;
  onNavigateUp: () => void;
  onNavigateTo: (path: string) => void;
  onSelectProject: (entry: FolderEntry) => void;
}

export function FolderBrowserPanel({
  scanResult,
  isScanning,
  error,
  onNavigateUp,
  onNavigateTo,
  onSelectProject,
}: FolderBrowserPanelProps) {
  const [showRegistered, setShowRegistered] = useState(false);

  // Filter entries based on showRegistered toggle
  const filteredEntries = scanResult?.entries.filter((entry) => {
    // Always show non-registered items
    if (!entry.is_registered) return true;
    // Only show registered items if toggle is on
    return showRegistered;
  }) ?? [];

  // Count hidden registered projects
  const hiddenCount = (scanResult?.entries.filter((e) => e.is_registered) ?? []).length;

  // Get display path (truncate if too long)
  const getDisplayPath = (path: string) => {
    const maxLength = 40;
    if (path.length <= maxLength) return path;
    return '...' + path.slice(-maxLength + 3);
  };

  if (error) {
    return (
      <div
        className="mt-4 p-4 bg-red-500/10 border border-red-500/30 rounded-xl"
        data-testid={selectors.dialogs.projectImport.folderBrowser.panel}
      >
        <div className="flex items-center gap-2 text-red-400">
          <AlertCircle size={16} />
          <span className="text-sm">{error}</span>
        </div>
      </div>
    );
  }

  if (isScanning && !scanResult) {
    return (
      <div
        className="mt-4 p-6 bg-gray-800/30 border border-gray-700/50 rounded-xl flex items-center justify-center"
        data-testid={selectors.dialogs.projectImport.folderBrowser.panel}
      >
        <Loader2 size={20} className="animate-spin text-gray-400 mr-2" />
        <span className="text-gray-400 text-sm">Scanning folder...</span>
      </div>
    );
  }

  if (!scanResult) {
    return null;
  }

  return (
    <div
      className="mt-4 bg-gray-800/30 border border-gray-700/50 rounded-xl overflow-hidden"
      data-testid={selectors.dialogs.projectImport.folderBrowser.panel}
    >
      {/* Header with navigation */}
      <div className="flex items-center justify-between px-3 py-2 border-b border-gray-700/50 bg-gray-800/50">
        <button
          onClick={onNavigateUp}
          disabled={!scanResult.parent || isScanning}
          className="flex items-center gap-1.5 px-2 py-1 text-xs text-gray-400 hover:text-white hover:bg-gray-700/50 rounded-lg transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          data-testid={selectors.dialogs.projectImport.folderBrowser.upButton}
        >
          <ChevronUp size={14} />
          Up
        </button>
        <div
          className="flex items-center gap-1.5 text-xs text-gray-500 font-mono truncate max-w-[60%]"
          title={scanResult.path}
          data-testid={selectors.dialogs.projectImport.folderBrowser.currentPath}
        >
          <FolderOpen size={12} className="flex-shrink-0" />
          {getDisplayPath(scanResult.path)}
        </div>
        {isScanning && (
          <Loader2 size={14} className="animate-spin text-gray-500" />
        )}
      </div>

      {/* Show registered toggle */}
      {hiddenCount > 0 && (
        <div className="px-3 py-2 border-b border-gray-700/30">
          <label
            className="flex items-center gap-2 text-xs text-gray-400 cursor-pointer hover:text-gray-300"
            data-testid={selectors.dialogs.projectImport.folderBrowser.showRegisteredToggle}
          >
            <input
              type="checkbox"
              checked={showRegistered}
              onChange={(e) => setShowRegistered(e.target.checked)}
              className="w-3.5 h-3.5 rounded border-gray-600 bg-gray-700 text-flow-accent focus:ring-flow-accent focus:ring-offset-0"
            />
            <span>
              Show already registered ({hiddenCount} hidden)
            </span>
          </label>
        </div>
      )}

      {/* Entry list */}
      <div
        className="max-h-64 overflow-y-auto"
        data-testid={selectors.dialogs.projectImport.folderBrowser.entryList}
      >
        {filteredEntries.length === 0 ? (
          <div className="px-4 py-6 text-center text-gray-500 text-sm">
            {scanResult.entries.length === 0
              ? 'No folders found'
              : 'No importable projects in this folder'}
          </div>
        ) : (
          filteredEntries.map((entry) => (
            <FolderEntryRow
              key={entry.path}
              entry={entry}
              isScanning={isScanning}
              onNavigate={() => onNavigateTo(entry.path)}
              onSelect={() => onSelectProject(entry)}
            />
          ))
        )}
      </div>
    </div>
  );
}

interface FolderEntryRowProps {
  entry: FolderEntry;
  isScanning: boolean;
  onNavigate: () => void;
  onSelect: () => void;
}

function FolderEntryRow({ entry, isScanning, onNavigate, onSelect }: FolderEntryRowProps) {
  const isProject = entry.is_project;
  const isRegistered = entry.is_registered;

  // Project styling
  const projectStyles = isProject
    ? isRegistered
      ? 'bg-gray-800/30 border-gray-600/30 text-gray-400' // Registered - muted
      : 'bg-green-500/5 border-green-500/20 hover:border-green-500/40' // Importable - highlighted
    : 'hover:bg-gray-700/30'; // Regular folder

  return (
    <div
      className={`flex items-center justify-between px-3 py-2 border-b border-gray-700/20 last:border-b-0 transition-colors ${projectStyles}`}
      data-testid={selectors.dialogs.projectImport.folderBrowser.entry}
      data-path={entry.path}
      data-is-project={entry.is_project}
      data-is-registered={entry.is_registered}
    >
      <div className="flex items-center gap-2 min-w-0 flex-1">
        {/* Icon */}
        {isProject ? (
          <Package
            size={16}
            className={isRegistered ? 'text-gray-500' : 'text-green-400'}
          />
        ) : (
          <FolderOpen size={16} className="text-blue-400" />
        )}

        {/* Name and status */}
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <span
              className={`text-sm truncate ${
                isProject && !isRegistered ? 'text-white font-medium' : 'text-gray-300'
              }`}
            >
              {entry.suggested_name || entry.name}
            </span>
            {isRegistered && (
              <span className="flex items-center gap-1 px-1.5 py-0.5 bg-gray-700/50 rounded text-[10px] text-gray-400">
                <Check size={10} />
                Registered
              </span>
            )}
          </div>
          {entry.suggested_name && entry.suggested_name !== entry.name && (
            <span className="text-[10px] text-gray-500 font-mono truncate block">
              {entry.name}/
            </span>
          )}
        </div>
      </div>

      {/* Action button */}
      <div className="flex-shrink-0 ml-2">
        {isProject ? (
          <button
            onClick={onSelect}
            disabled={isScanning}
            className={`px-3 py-1 text-xs font-medium rounded-lg transition-colors ${
              isRegistered
                ? 'text-gray-400 hover:text-gray-300 hover:bg-gray-700/50'
                : 'text-green-300 bg-green-500/20 hover:bg-green-500/30 border border-green-500/30'
            } disabled:opacity-50`}
            data-testid={selectors.dialogs.projectImport.folderBrowser.selectButton}
          >
            Select
          </button>
        ) : (
          <button
            onClick={onNavigate}
            disabled={isScanning}
            className="flex items-center gap-1 px-2 py-1 text-xs text-gray-400 hover:text-white hover:bg-gray-700/50 rounded-lg transition-colors disabled:opacity-50"
            data-testid={selectors.dialogs.projectImport.folderBrowser.openButton}
          >
            Open
            <ChevronRight size={12} />
          </button>
        )}
      </div>
    </div>
  );
}

export default FolderBrowserPanel;
