import { useMemo, useState } from 'react';
import { UnifiedTimeline } from '../timeline/TimelineEventCard';
import type { RecordedAction, SelectorValidation, RecordingSessionProfile } from '../types/types';
import type { TimelineItem, TimelineMode } from '../types/timeline-unified';

interface RecordActionsPanelProps {
  /** Legacy: RecordedActions for backward compatibility */
  actions: RecordedAction[];
  /** Unified timeline items (preferred) */
  timelineItems?: TimelineItem[];
  /** Current mode: recording or execution */
  mode?: TimelineMode;
  isRecording: boolean;
  isLoading: boolean;
  isReplaying: boolean;
  /** Whether timeline is receiving live updates */
  isLive?: boolean;
  hasUnstableSelectors: boolean;
  timelineWidth: number;
  isResizingSidebar: boolean;
  onResizeStart: (event: React.MouseEvent<HTMLDivElement>) => void;
  sessionProfiles: RecordingSessionProfile[];
  sessionProfilesLoading: boolean;
  selectedSessionProfileId: string | null;
  onSelectSessionProfile: (profileId: string | null) => void;
  onCreateSessionProfile: () => void;
  onClearRequested: () => void;
  onCreateWorkflow: () => void;
  onDeleteAction?: (index: number) => void;
  onValidateSelector?: (selector: string) => Promise<SelectorValidation>;
  onEditSelector?: (index: number, newSelector: string) => void;
  onEditPayload?: (index: number, payload: Record<string, unknown>) => void;
  /** Selection mode state */
  isSelectionMode: boolean;
  selectedIndices: Set<number>;
  onToggleSelectionMode: () => void;
  onActionClick: (index: number, shiftKey: boolean, ctrlKey: boolean) => void;
  onSelectAll: () => void;
  onSelectNone: () => void;
}

export function RecordActionsPanel({
  actions,
  timelineItems,
  mode = 'recording',
  isRecording,
  isLoading,
  isReplaying,
  isLive,
  hasUnstableSelectors,
  timelineWidth,
  isResizingSidebar,
  onResizeStart,
  sessionProfiles,
  sessionProfilesLoading,
  selectedSessionProfileId,
  onSelectSessionProfile,
  onCreateSessionProfile,
  onClearRequested,
  onCreateWorkflow,
  onDeleteAction,
  onValidateSelector: _onValidateSelector,
  onEditSelector,
  onEditPayload: _onEditPayload,
  isSelectionMode,
  selectedIndices,
  onToggleSelectionMode,
  onActionClick,
  onSelectAll,
  onSelectNone,
}: RecordActionsPanelProps) {
  // Use unified timeline items if provided, otherwise fall back to actions count
  const itemCount = timelineItems?.length ?? actions.length;
  const [sessionMenuOpen, setSessionMenuOpen] = useState(false);

  const selectedSession = useMemo(
    () => sessionProfiles.find((profile) => profile.id === selectedSessionProfileId),
    [sessionProfiles, selectedSessionProfileId]
  );

  const formatLastUsed = (value?: string) => {
    if (!value) return 'never';
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
      return 'unknown';
    }
    return date.toLocaleString();
  };

  const selectionCount = selectedIndices.size;

  return (
    <div
      className="relative h-full flex flex-col border-r border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900"
      style={{ width: `${timelineWidth}px`, minWidth: '240px', maxWidth: '640px' }}
    >
      <div className="flex items-center justify-between px-3 py-2 border-b border-gray-200 dark:border-gray-700">
        <div className="flex items-center gap-2">
          {/* Recording indicator (always on when page is open) */}
          {isRecording && (
            <div className="flex items-center gap-1.5 px-2 py-1 text-xs text-red-600 dark:text-red-400">
              <span className="relative flex h-2 w-2">
                <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-red-400 opacity-75" />
                <span className="relative inline-flex rounded-full h-2 w-2 bg-red-500" />
              </span>
              Recording
            </div>
          )}

          {/* Select button - toggles selection mode */}
          <button
            onClick={onToggleSelectionMode}
            disabled={itemCount === 0}
            className={`flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-md transition-colors ${
              isSelectionMode
                ? 'text-blue-700 dark:text-blue-300 bg-blue-100 dark:bg-blue-900/50 border border-blue-300 dark:border-blue-700'
                : 'text-gray-700 dark:text-gray-200 bg-gray-100 dark:bg-gray-800 border border-gray-200 dark:border-gray-700 hover:bg-gray-200 dark:hover:bg-gray-700'
            } disabled:opacity-50`}
          >
            <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
            </svg>
            {isSelectionMode ? 'Exit Select' : 'Select'}
          </button>

          {/* Selection actions when in selection mode */}
          {isSelectionMode && (
            <div className="flex items-center gap-1">
              <button
                onClick={onSelectAll}
                className="px-2 py-1 text-xs text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200"
              >
                All
              </button>
              <button
                onClick={onSelectNone}
                className="px-2 py-1 text-xs text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200"
              >
                None
              </button>
            </div>
          )}

          {/* Session selector - combines dropdown with indicator */}
          {!isSelectionMode && (
            <div className="relative" onMouseLeave={() => setSessionMenuOpen(false)}>
              <button
                type="button"
                className="flex items-center gap-1.5 px-2.5 py-1.5 text-xs rounded-md border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-gray-700 dark:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700"
                onClick={() => setSessionMenuOpen((open) => !open)}
                title="Select recording session"
              >
              <svg className="w-3.5 h-3.5 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
              </svg>
              <span className="font-medium max-w-[100px] truncate">{selectedSession?.name ?? 'Default'}</span>
              {selectedSession?.hasStorageState && (
                <span className="w-1.5 h-1.5 rounded-full bg-green-500" title="Auth saved" />
              )}
              <svg className={`w-3.5 h-3.5 text-gray-400 transition-transform ${sessionMenuOpen ? 'rotate-180' : ''}`} viewBox="0 0 20 20" fill="currentColor">
                <path fillRule="evenodd" d="M5.23 7.21a.75.75 0 011.06.02L10 10.94l3.71-3.71a.75.75 0 111.06 1.06l-4.24 4.25a.75.75 0 01-1.06 0L5.21 8.29a.75.75 0 01.02-1.08z" clipRule="evenodd" />
              </svg>
            </button>
            {sessionMenuOpen && (
              <div className="absolute top-full left-0 z-20 mt-1 w-72 rounded-md border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 shadow-lg">
                <div className="px-3 py-2 border-b border-gray-100 dark:border-gray-700">
                  <p className="text-xs font-semibold text-gray-700 dark:text-gray-200">Recording sessions</p>
                  <p className="text-[11px] text-gray-500 dark:text-gray-400">
                    Reuse a session to stay signed in or start fresh.
                  </p>
                </div>
                <div className="max-h-64 overflow-y-auto">
                  {sessionProfilesLoading ? (
                    <div className="px-3 py-2 text-xs text-gray-500">Loading sessions…</div>
                  ) : sessionProfiles.length === 0 ? (
                    <button
                      className="w-full text-left px-3 py-2 text-sm text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-700"
                      onClick={() => {
                        onCreateSessionProfile();
                        setSessionMenuOpen(false);
                      }}
                    >
                      Use new session
                      <div className="text-xs text-gray-500">No saved sessions yet</div>
                    </button>
                  ) : (
                    sessionProfiles.map((profile) => (
                      <button
                        key={profile.id}
                        className={`w-full text-left px-3 py-2 text-sm hover:bg-gray-50 dark:hover:bg-gray-700 ${
                          profile.id === selectedSessionProfileId
                            ? 'bg-blue-50 dark:bg-blue-900/20 text-blue-700 dark:text-blue-200'
                            : 'text-gray-800 dark:text-gray-100'
                        }`}
                        onClick={() => {
                          onSelectSessionProfile(profile.id);
                          setSessionMenuOpen(false);
                        }}
                      >
                        <div className="flex items-center justify-between gap-2">
                          <span className="font-medium">{profile.name}</span>
                          {profile.hasStorageState && (
                            <span className="text-[10px] px-2 py-0.5 rounded-full bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-200">
                              Auth saved
                            </span>
                          )}
                        </div>
                        <div className="text-[11px] text-gray-500 dark:text-gray-400">
                          Last used {formatLastUsed(profile.lastUsedAt)}
                        </div>
                      </button>
                    ))
                  )}
                </div>
                <div className="border-t border-gray-100 dark:border-gray-700">
                  <button
                    className="w-full text-left px-3 py-2 text-sm text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-700"
                    onClick={() => {
                      onCreateSessionProfile();
                      setSessionMenuOpen(false);
                    }}
                  >
                    + New session
                  </button>
                </div>
              </div>
            )}
          </div>
          )}

          {mode === 'recording' && itemCount > 0 && !isSelectionMode && (
            <button
              onClick={onClearRequested}
              className="text-xs text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 px-2 py-1"
            >
              Clear
            </button>
          )}
        </div>

        <div className="flex items-center gap-2 text-xs text-gray-500">
          {isSelectionMode && selectionCount > 0 && (
            <span className="px-2 py-0.5 bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300 rounded-full font-medium">
              {selectionCount} selected
            </span>
          )}
          {!isSelectionMode && itemCount > 0 && hasUnstableSelectors && (
            <span className="px-2 py-0.5 bg-yellow-50 dark:bg-yellow-900/20 text-yellow-700 dark:text-yellow-300 rounded-full">
              Review selectors
            </span>
          )}
          {/* Mode indicator for execution */}
          {mode === 'execution' && (
            <span className="px-2 py-0.5 bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300 rounded-full">
              Executing
            </span>
          )}
          <span>
            {itemCount} step{itemCount !== 1 ? 's' : ''}
          </span>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto">
        {timelineItems ? (
          <UnifiedTimeline
            items={timelineItems}
            isLoading={isLoading}
            isLive={isLive ?? isRecording}
            mode={mode}
            isSelectionMode={isSelectionMode}
            selectedIndices={selectedIndices}
            onItemClick={onActionClick}
            onDeleteItem={mode === 'recording' ? onDeleteAction : undefined}
            onEditSelector={mode === 'recording' && onEditSelector ? (index) => onEditSelector(index, '') : undefined}
          />
        ) : (
          <UnifiedTimeline
            items={[]}
            isLoading={isLoading}
            isLive={isRecording}
            mode={mode}
            isSelectionMode={isSelectionMode}
            selectedIndices={selectedIndices}
            onItemClick={onActionClick}
            onDeleteItem={onDeleteAction}
          />
        )}
      </div>

      {/* Footer: Create Workflow button when actions exist (recording mode only) */}
      {mode === 'recording' && itemCount > 0 && (
        <div className="px-3 py-2 border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50">
          <button
            onClick={onCreateWorkflow}
            disabled={isReplaying || isLoading || (isSelectionMode && selectionCount === 0)}
            className="w-full flex items-center justify-center gap-2 px-4 py-2 text-sm font-medium text-white bg-blue-500 rounded-lg hover:bg-blue-600 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {isSelectionMode && selectionCount > 0 ? (
              <>
                Create Workflow ({selectionCount} steps)
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 7l5 5m0 0l-5 5m5-5H6" />
                </svg>
              </>
            ) : isSelectionMode ? (
              'Select steps to create workflow'
            ) : (
              <>
                Create Workflow
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 7l5 5m0 0l-5 5m5-5H6" />
                </svg>
              </>
            )}
          </button>
          {!isSelectionMode && (
            <p className="text-xs text-gray-500 dark:text-gray-400 text-center mt-2">
              Use <span className="font-medium">Select</span> to choose specific steps
            </p>
          )}
        </div>
      )}

      <div
        className={`absolute top-0 right-[-6px] h-full w-3 cursor-col-resize ${
          isResizingSidebar ? 'bg-blue-100 dark:bg-blue-900/40' : ''
        }`}
        onMouseDown={onResizeStart}
      />
    </div>
  );
}
