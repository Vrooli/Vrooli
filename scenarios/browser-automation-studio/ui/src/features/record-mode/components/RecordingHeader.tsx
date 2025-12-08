interface RecordingHeaderProps {
  isRecording: boolean;
  actionCount: number;
  isSidebarOpen: boolean;
  onToggleTimeline: () => void;
  onClose?: () => void;
}

export function RecordingHeader({
  isRecording,
  actionCount,
  isSidebarOpen,
  onToggleTimeline,
  onClose,
}: RecordingHeaderProps) {
  return (
    <div className="flex items-center justify-between px-4 py-3 border-b border-gray-200 dark:border-gray-700">
      <div className="flex items-center gap-3">
        <h1 className="text-lg font-semibold text-gray-900 dark:text-white">Record Mode</h1>
        {isRecording && (
          <span className="flex items-center gap-1.5 px-2 py-0.5 text-xs font-medium text-red-600 bg-red-100 dark:bg-red-900/30 dark:text-red-400 rounded-full">
            <span className="w-1.5 h-1.5 bg-red-500 rounded-full animate-pulse" />
            Recording
          </span>
        )}
      </div>
      <div className="flex items-center gap-2">
        <button
          onClick={onToggleTimeline}
          className="relative p-2 text-gray-600 dark:text-gray-300 hover:text-gray-900 dark:hover:text-white"
          title={isSidebarOpen ? 'Hide timeline' : 'Show timeline'}
        >
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h10M4 18h7" />
          </svg>
          <span className="absolute -top-1 -right-1 inline-flex items-center justify-center px-1.5 py-0.5 text-[10px] font-semibold text-white bg-blue-500 rounded-full">
            {actionCount}
          </span>
        </button>
        {onClose && (
          <button
            onClick={onClose}
            className="p-2 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
            title="Close Record Mode"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        )}
      </div>
    </div>
  );
}
