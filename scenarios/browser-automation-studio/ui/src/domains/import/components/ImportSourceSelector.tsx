/**
 * ImportSourceSelector Component
 *
 * Shared layout component for the dual-input pattern used across import modals.
 * Provides responsive layout: side-by-side on desktop, stacked on mobile.
 */

import { useRef, useCallback } from 'react';
import { Loader2, CheckCircle2, AlertCircle } from 'lucide-react';

import { DropZone } from './DropZone';
import { FolderBrowser } from './FolderBrowser';
import type { DropZoneVariant, FolderEntry, ScanMode, SelectedFile } from '../types';

export interface ImportSourceSelectorProps {
  // DropZone props
  dropZoneVariant: DropZoneVariant;
  dropZoneAccept?: readonly string[] | string[];
  dropZoneLabel?: string;
  dropZoneDescription?: string;
  onFilesSelected?: (files: SelectedFile[]) => void;
  onDropZoneFolderSelected?: (path: string) => void;

  // FolderBrowser props
  folderBrowserMode: ScanMode;
  projectId?: string;
  initialPath?: string;
  onFolderSelect: (entry: FolderEntry) => void;
  onNavigate?: (path: string) => void;
  showRegistered?: boolean;

  // Manual path input
  showPathInput?: boolean;
  pathValue?: string;
  pathPlaceholder?: string;
  pathLabel?: string;
  onPathChange?: (path: string) => void;
  onPathSubmit?: () => void;
  pathValidationStatus?: 'valid' | 'invalid' | 'checking' | null;
  pathError?: string | null;

  // State
  disabled?: boolean;
  isLoading?: boolean;

  // Custom class names
  className?: string;
  dropZoneClassName?: string;
  browserClassName?: string;
}

export function ImportSourceSelector({
  // DropZone props
  dropZoneVariant,
  dropZoneAccept,
  dropZoneLabel,
  dropZoneDescription,
  onFilesSelected,
  onDropZoneFolderSelected,

  // FolderBrowser props
  folderBrowserMode,
  projectId,
  initialPath,
  onFolderSelect,
  onNavigate,
  showRegistered = false,

  // Manual path input
  showPathInput = true,
  pathValue = '',
  pathPlaceholder = '/path/to/folder',
  pathLabel = 'Path',
  onPathChange,
  onPathSubmit,
  pathValidationStatus,
  pathError,

  // State
  disabled = false,
  isLoading = false,

  // Custom class names
  className = '',
  dropZoneClassName = '',
  browserClassName = '',
}: ImportSourceSelectorProps) {
  const pathInputRef = useRef<HTMLInputElement>(null);

  const handlePathKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'Enter' && onPathSubmit) {
        e.preventDefault();
        onPathSubmit();
      }
    },
    [onPathSubmit]
  );

  return (
    <div className={`space-y-4 ${className}`}>
      {/* Responsive layout: side-by-side on md+, stacked on mobile */}
      <div className="flex flex-col md:flex-row md:items-stretch">
        {/* Left side: DropZone */}
        <div className={`flex flex-col flex-1 ${dropZoneClassName}`}>
          <DropZone
            variant={dropZoneVariant}
            accept={dropZoneAccept}
            onFilesSelected={onFilesSelected}
            onFolderSelected={onDropZoneFolderSelected}
            label={dropZoneLabel}
            description={dropZoneDescription}
            showPreview={false}
            disabled={disabled || isLoading}
            className="h-full min-h-[200px]"
          />
        </div>

        {/* Mobile divider - only shown on small screens */}
        <div className="flex items-center gap-3 py-4 md:hidden">
          <div className="flex-1 h-px bg-gray-700/50" />
          <span className="text-xs text-gray-500">or browse below</span>
          <div className="flex-1 h-px bg-gray-700/50" />
        </div>

        {/* Desktop vertical divider - only shown on md+ */}
        <div className="hidden md:flex md:flex-col md:items-center md:px-4">
          <div className="flex-1 w-px bg-gray-700/50" />
          <span className="text-xs text-gray-500 py-2 whitespace-nowrap">or</span>
          <div className="flex-1 w-px bg-gray-700/50" />
        </div>

        {/* Right side: Path input + FolderBrowser */}
        <div className={`flex flex-col flex-1 space-y-3 ${browserClassName}`}>
          {/* Path Input */}
          {showPathInput && (
            <div>
              <label className="block text-sm font-medium text-gray-300 mb-2">
                {pathLabel}
              </label>
              <div className="relative">
                <input
                  ref={pathInputRef}
                  type="text"
                  value={pathValue}
                  onChange={(e) => onPathChange?.(e.target.value)}
                  onKeyDown={handlePathKeyDown}
                  className={`w-full px-4 py-2.5 pr-10 bg-gray-800/50 border-2 rounded-xl text-white placeholder-gray-500 focus:outline-none font-mono text-sm transition-colors ${
                    pathError
                      ? 'border-red-500/50 focus:border-red-500'
                      : pathValidationStatus === 'valid'
                      ? 'border-green-500/50 focus:border-green-500'
                      : 'border-gray-700/50 focus:border-flow-accent'
                  }`}
                  placeholder={pathPlaceholder}
                  disabled={disabled || isLoading}
                />
                {/* Validation status indicator */}
                <div className="absolute right-3 top-1/2 -translate-y-1/2">
                  {pathValidationStatus === 'checking' && (
                    <Loader2 size={16} className="animate-spin text-gray-500" />
                  )}
                  {pathValidationStatus === 'valid' && (
                    <CheckCircle2 size={16} className="text-green-400" />
                  )}
                  {pathValidationStatus === 'invalid' && (
                    <AlertCircle size={16} className="text-red-400" />
                  )}
                </div>
              </div>
              {pathError && (
                <p className="mt-1.5 text-red-400 text-xs flex items-center gap-1">
                  <AlertCircle size={12} />
                  {pathError}
                </p>
              )}
            </div>
          )}

          {/* Folder Browser */}
          <FolderBrowser
            mode={folderBrowserMode}
            projectId={projectId}
            onSelect={onFolderSelect}
            onNavigate={onNavigate}
            initialPath={initialPath}
            showRegistered={showRegistered}
            className="flex-1 min-h-[150px]"
          />
        </div>
      </div>

      {/* Loading indicator */}
      {isLoading && (
        <div className="flex items-center justify-center gap-2 text-gray-400 py-2">
          <Loader2 size={16} className="animate-spin" />
          <span className="text-sm">Processing...</span>
        </div>
      )}
    </div>
  );
}

export default ImportSourceSelector;
