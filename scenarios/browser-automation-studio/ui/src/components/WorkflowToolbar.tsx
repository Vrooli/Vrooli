import { ZoomIn, ZoomOut, Maximize2, Grid, Lock, Unlock, Copy, Trash2, Undo, Redo } from 'lucide-react';
import { useReactFlow } from 'reactflow';

interface WorkflowToolbarProps {
  showGrid: boolean;
  onToggleGrid: () => void;
  locked: boolean;
  onToggleLock: () => void;
  onUndo: () => void;
  onRedo: () => void;
  canUndo: boolean;
  canRedo: boolean;
  onDuplicate: () => void;
  onDelete: () => void;
}

function WorkflowToolbar({ showGrid, onToggleGrid, locked, onToggleLock, onUndo, onRedo, canUndo, canRedo, onDuplicate, onDelete }: WorkflowToolbarProps) {
  const { zoomIn, zoomOut, fitView } = useReactFlow();

  return (
    <div className="absolute top-4 left-4 z-10 workflow-toolbar">
      <button
        onClick={() => zoomIn()}
        className="toolbar-button"
        title="Zoom In"
      >
        <ZoomIn size={18} />
      </button>
      
      <button
        onClick={() => zoomOut()}
        className="toolbar-button"
        title="Zoom Out"
      >
        <ZoomOut size={18} />
      </button>
      
      <button
        onClick={() => fitView()}
        className="toolbar-button"
        title="Fit View"
      >
        <Maximize2 size={18} />
      </button>
      
      <div className="w-px h-6 bg-gray-700" />
      
      <button
        onClick={onToggleGrid}
        className={`toolbar-button ${showGrid ? 'active' : ''}`}
        title="Toggle Grid"
      >
        <Grid size={18} />
      </button>
      
      <button
        onClick={onToggleLock}
        className="toolbar-button"
        title={locked ? 'Unlock' : 'Lock'}
      >
        {locked ? <Lock size={18} /> : <Unlock size={18} />}
      </button>
      
      <div className="w-px h-6 bg-gray-700" />
      
      <button
        onClick={onUndo}
        className={`toolbar-button ${!canUndo ? 'opacity-50 cursor-not-allowed' : ''}`}
        title="Undo"
        disabled={!canUndo}
      >
        <Undo size={18} />
      </button>
      
      <button
        onClick={onRedo}
        className={`toolbar-button ${!canRedo ? 'opacity-50 cursor-not-allowed' : ''}`}
        title="Redo"
        disabled={!canRedo}
      >
        <Redo size={18} />
      </button>
      
      <div className="w-px h-6 bg-gray-700" />
      
      <button
        onClick={onDuplicate}
        className="toolbar-button"
        title="Duplicate Selected"
      >
        <Copy size={18} />
      </button>
      
      <button
        onClick={onDelete}
        className="toolbar-button text-red-400 hover:text-red-300"
        title="Delete Selected"
      >
        <Trash2 size={18} />
      </button>
    </div>
  );
}

export default WorkflowToolbar;