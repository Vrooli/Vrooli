import { useState, type MouseEvent as ReactMouseEvent } from 'react';
import { ChevronLeft, ChevronRight } from 'lucide-react';
import { NodePalette } from '@features/workflows';

interface SidebarProps {
  selectedFolder: string;
  onFolderSelect: (folder: string) => void;
  projectId?: string;
}

function Sidebar({
  selectedFolder: _selectedFolder,
  onFolderSelect: _onFolderSelect,
  projectId: _projectId,
}: SidebarProps) {
  const COLLAPSED_WIDTH = 48;
  const MIN_WIDTH = 220;
  const MAX_WIDTH = 420;
  const DEFAULT_WIDTH = 256;

  const clampWidth = (value: number) => Math.min(MAX_WIDTH, Math.max(MIN_WIDTH, value));
  const getInitialCollapsed = () => {
    if (typeof window === 'undefined') {
      return false;
    }
    return window.matchMedia('(max-width: 768px)').matches;
  };

  const [isCollapsed, setIsCollapsed] = useState(getInitialCollapsed);
  const [sidebarWidth, setSidebarWidth] = useState(DEFAULT_WIDTH);
  const [savedWidth, setSavedWidth] = useState(DEFAULT_WIDTH);
  const [isResizing, setIsResizing] = useState(false);

  const handleToggleCollapse = () => {
    setIsCollapsed((prev) => {
      if (prev) {
        setSidebarWidth(() => clampWidth(savedWidth));
        return false;
      }

      setSavedWidth(sidebarWidth);
      return true;
    });
  };

  const handleResizeMouseDown = (event: ReactMouseEvent<HTMLDivElement>) => {
    if (isCollapsed) {
      return;
    }

    event.preventDefault();
    event.stopPropagation();

    const startX = event.clientX;
    const startWidth = sidebarWidth;

    setIsResizing(true);
    document.body.style.userSelect = 'none';
    document.body.style.cursor = 'col-resize';

    const onMouseMove = (moveEvent: MouseEvent) => {
      const deltaX = moveEvent.clientX - startX;
      const nextWidth = clampWidth(startWidth + deltaX);
      setSidebarWidth(nextWidth);
      setSavedWidth(nextWidth);
    };

    const onMouseUp = () => {
      setIsResizing(false);
      document.body.style.userSelect = '';
      document.body.style.cursor = '';
      window.removeEventListener('mousemove', onMouseMove);
      window.removeEventListener('mouseup', onMouseUp);
    };

    window.addEventListener('mousemove', onMouseMove);
    window.addEventListener('mouseup', onMouseUp);
  };

  return (
    <div
      className={`bg-flow-node border-r border-gray-800 flex flex-col transition-[width] duration-150 ${
        isResizing ? 'select-none' : ''
      }`}
      style={{ width: isCollapsed ? COLLAPSED_WIDTH : sidebarWidth }}
    >
      <div className="flex items-center justify-between border-b border-gray-800 px-3 py-2">
        {!isCollapsed && (
          <span className="text-sm font-semibold text-white uppercase tracking-wide">
            Node Palette
          </span>
        )}
        <button
          type="button"
          onClick={handleToggleCollapse}
          className="p-2 text-gray-400 hover:text-white hover:bg-gray-700 rounded-md transition-colors"
          title={isCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
          aria-label={isCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
        >
          {isCollapsed ? <ChevronRight size={18} /> : <ChevronLeft size={18} />}
        </button>
      </div>

      {!isCollapsed && (
        <div className="flex flex-1 min-h-0">
          <div className="flex-1 overflow-y-auto">
            <NodePalette />
          </div>
          <div
            role="separator"
            aria-orientation="vertical"
            className={`w-1 cursor-col-resize transition-colors ${
              isResizing ? 'bg-flow-accent/50' : 'bg-transparent hover:bg-flow-accent/40'
            }`}
            onMouseDown={handleResizeMouseDown}
            aria-label="Resize node palette sidebar"
          />
        </div>
      )}
    </div>
  );
}

export default Sidebar;
