import { ZoomIn, ZoomOut, Maximize2, Grid, Lock, Unlock, Copy, Trash2, Undo, Redo } from 'lucide-react';
import { useReactFlow } from 'reactflow';
import { useState } from 'react';

function WorkflowToolbar() {
  const { zoomIn, zoomOut, fitView } = useReactFlow();
  const [locked, setLocked] = useState(false);
  const [showGrid, setShowGrid] = useState(true);

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
        onClick={() => setShowGrid(!showGrid)}
        className={`toolbar-button ${showGrid ? 'active' : ''}`}
        title="Toggle Grid"
      >
        <Grid size={18} />
      </button>
      
      <button
        onClick={() => setLocked(!locked)}
        className="toolbar-button"
        title={locked ? 'Unlock' : 'Lock'}
      >
        {locked ? <Lock size={18} /> : <Unlock size={18} />}
      </button>
      
      <div className="w-px h-6 bg-gray-700" />
      
      <button
        className="toolbar-button"
        title="Undo"
      >
        <Undo size={18} />
      </button>
      
      <button
        className="toolbar-button"
        title="Redo"
      >
        <Redo size={18} />
      </button>
      
      <div className="w-px h-6 bg-gray-700" />
      
      <button
        className="toolbar-button"
        title="Duplicate"
      >
        <Copy size={18} />
      </button>
      
      <button
        className="toolbar-button text-red-400 hover:text-red-300"
        title="Delete"
      >
        <Trash2 size={18} />
      </button>
    </div>
  );
}

export default WorkflowToolbar;