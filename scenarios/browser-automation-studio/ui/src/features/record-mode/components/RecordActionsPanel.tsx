import { useMemo, useState } from 'react';
import { ActionTimeline } from '../ActionTimeline';
import type { RecordedAction, SelectorValidation, RecordingSessionProfile } from '../types';

interface RecordActionsPanelProps {
  actions: RecordedAction[];
  isRecording: boolean;
  isStartingRecording: boolean;
  isLoading: boolean;
  isReplaying: boolean;
  hasUnstableSelectors: boolean;
  timelineWidth: number;
  isResizingSidebar: boolean;
  onResizeStart: (event: React.MouseEvent<HTMLDivElement>) => void;
  sessionProfiles: RecordingSessionProfile[];
  sessionProfilesLoading: boolean;
  selectedSessionProfileId: string | null;
  onSelectSessionProfile: (profileId: string | null) => void;
  onCreateSessionProfile: () => void;
  onStartRecording: () => void;
  onStopRecording: () => void;
  onClearRequested: () => void;
  onTestRecording: () => void;
  onGenerateWorkflow: () => void;
  onDeleteAction?: (index: number) => void;
  onValidateSelector?: (selector: string) => Promise<SelectorValidation>;
  onEditSelector?: (index: number, newSelector: string) => void;
  onEditPayload?: (index: number, payload: Record<string, unknown>) => void;
}

export function RecordActionsPanel({
  actions,
  isRecording,
  isStartingRecording,
  isLoading,
  isReplaying,
  hasUnstableSelectors,
  timelineWidth,
  isResizingSidebar,
  onResizeStart,
  sessionProfiles,
  sessionProfilesLoading,
  selectedSessionProfileId,
  onSelectSessionProfile,
  onCreateSessionProfile,
  onStartRecording,
  onStopRecording,
  onClearRequested,
  onTestRecording,
  onGenerateWorkflow,
  onDeleteAction,
  onValidateSelector,
  onEditSelector,
  onEditPayload,
}: RecordActionsPanelProps) {
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

  return (
    <div
      className="relative h-full flex flex-col border-r border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900"
      style={{ width: `${timelineWidth}px`, minWidth: '240px', maxWidth: '640px' }}
    >
      <div className="flex items-center justify-between px-3 py-2 border-b border-gray-200 dark:border-gray-700">
        <div className="flex items-center gap-2">
          {!isRecording ? (
            <div className="relative flex items-center gap-1" onMouseLeave={() => setSessionMenuOpen(false)}>
              <button
                onClick={() => {
                  setSessionMenuOpen(false);
                  onStartRecording();
                }}
                disabled={isStartingRecording}
                className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-white bg-red-500 rounded-md hover:bg-red-600 disabled:opacity-50"
              >
                {isStartingRecording ? (
                  <svg className="w-3.5 h-3.5 animate-spin" fill="none" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                  </svg>
                ) : (
                  <svg className="w-3.5 h-3.5" fill="currentColor" viewBox="0 0 24 24">
                    <circle cx="12" cy="12" r="8" />
                  </svg>
                )}
                Record
              </button>
              <button
                type="button"
                className="h-8 w-8 flex items-center justify-center rounded-md border border-gray-200 dark:border-gray-700 text-gray-600 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-800"
                aria-label="Choose recording session"
                onClick={() => setSessionMenuOpen((open) => !open)}
              >
                <svg className="w-4 h-4" viewBox="0 0 20 20" fill="currentColor">
                  <path
                    fillRule="evenodd"
                    d="M5.23 7.21a.75.75 0 011.06.02L10 10.94l3.71-3.71a.75.75 0 111.06 1.06l-4.24 4.25a.75.75 0 01-1.06 0L5.21 8.29a.75.75 0 01.02-1.08z"
                    clipRule="evenodd"
                  />
                </svg>
              </button>
              {sessionMenuOpen && (
                <div className="absolute top-full left-0 z-20 mt-1 w-72 rounded-md border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 shadow-lg">
                  <div className="px-3 py-2 border-b border-gray-100 dark:border-gray-700">
                    <p className="text-xs font-semibold text-gray-700 dark:text-gray-200">Recording sessions</p>
                    <p className="text-[11px] text-gray-500 dark:text-gray-400">
                      Reuse a session to stay signed in or start with a fresh profile.
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
                              ? 'bg-red-50 dark:bg-red-900/20 text-red-700 dark:text-red-200'
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
                      Use new session
                    </button>
                  </div>
                </div>
              )}
            </div>
          ) : (
            <button
              onClick={onStopRecording}
              disabled={isLoading}
              className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-white bg-gray-700 rounded-md hover:bg-gray-800 disabled:opacity-50"
            >
              {isLoading ? (
                <svg className="w-3.5 h-3.5 animate-spin" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                </svg>
              ) : (
                <svg className="w-3.5 h-3.5" fill="currentColor" viewBox="0 0 24 24">
                  <rect x="6" y="6" width="12" height="12" rx="1" />
                </svg>
              )}
              Stop
            </button>
          )}

          {actions.length > 0 && !isRecording && (
            <button
              onClick={onClearRequested}
              className="text-xs text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 px-2 py-1"
            >
              Clear
            </button>
          )}
        </div>

        <div className="flex items-center gap-2 text-xs text-gray-500">
          <span className="hidden sm:inline-flex items-center gap-1 px-2 py-1 rounded-full bg-gray-100 dark:bg-gray-800 border border-gray-200 dark:border-gray-700">
            <span className="text-gray-600 dark:text-gray-300">Session:</span>
            <span className="font-medium text-gray-800 dark:text-gray-100">
              {selectedSession?.name ?? 'Default'}
            </span>
          </span>
          {actions.length > 0 && hasUnstableSelectors && !isRecording && (
            <span className="px-2 py-0.5 bg-yellow-50 dark:bg-yellow-900/20 text-yellow-700 dark:text-yellow-300 rounded-full">
              Review selectors
            </span>
          )}
          <span>
            {actions.length} step{actions.length !== 1 ? 's' : ''}
          </span>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto">
        <ActionTimeline
          actions={actions}
          isRecording={isRecording}
          onDeleteAction={onDeleteAction}
          onValidateSelector={onValidateSelector}
          onEditSelector={onEditSelector}
          onEditPayload={onEditPayload}
        />
      </div>

      {actions.length > 0 && !isRecording && (
        <div className="px-3 py-2 border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50">
          <div className="flex gap-2">
            <button
              onClick={onTestRecording}
              disabled={isReplaying || isLoading}
              className="flex-1 flex items-center justify-center gap-1.5 px-3 py-1.5 text-xs font-medium text-gray-700 dark:text-gray-200 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-md hover:bg-gray-50 dark:hover:bg-gray-600 disabled:opacity-50"
            >
              {isReplaying ? 'Testing…' : 'Test'}
            </button>
            <button
              onClick={onGenerateWorkflow}
              disabled={isReplaying || isLoading}
              className="flex-1 flex items-center justify-center gap-1.5 px-3 py-1.5 text-xs font-medium text-white bg-blue-500 rounded-md hover:bg-blue-600 disabled:opacity-50"
            >
              Generate
            </button>
          </div>
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
