import { Globe, MousePointer, Keyboard, Camera, Clock, Database, Code, Play } from 'lucide-react';

const nodeTypes = [
  { type: 'navigate', label: 'Navigate', icon: Globe, color: 'text-blue-400', description: 'Navigate to URL' },
  { type: 'click', label: 'Click', icon: MousePointer, color: 'text-green-400', description: 'Click an element' },
  { type: 'type', label: 'Type', icon: Keyboard, color: 'text-yellow-400', description: 'Type text in input' },
  { type: 'screenshot', label: 'Screenshot', icon: Camera, color: 'text-purple-400', description: 'Capture screenshot' },
  { type: 'wait', label: 'Wait', icon: Clock, color: 'text-gray-400', description: 'Wait for condition' },
  { type: 'extract', label: 'Extract', icon: Database, color: 'text-pink-400', description: 'Extract data' },
  { type: 'workflowCall', label: 'Call Workflow', icon: Play, color: 'text-violet-400', description: 'Execute another workflow' },
];

function NodePalette() {
  const onDragStart = (event: React.DragEvent, nodeType: string) => {
    event.dataTransfer.setData('nodeType', nodeType);
    event.dataTransfer.effectAllowed = 'move';
  };

  return (
    <div className="flex-1 overflow-y-auto p-3">
      <div className="mb-3">
        <h3 className="text-xs font-semibold text-gray-500 uppercase tracking-wider mb-2">
          Drag nodes to canvas
        </h3>
      </div>
      
      <div className="space-y-2">
        {nodeTypes.map((node) => {
          const Icon = node.icon;
          return (
            <div
              key={node.type}
              draggable
              onDragStart={(e) => onDragStart(e, node.type)}
              className="bg-flow-bg border border-gray-700 rounded-lg p-3 cursor-move hover:border-flow-accent transition-colors group"
            >
              <div className="flex items-start gap-3">
                <div className={`mt-0.5 ${node.color} group-hover:scale-110 transition-transform`}>
                  <Icon size={18} />
                </div>
                <div className="flex-1">
                  <div className="font-medium text-sm text-white mb-0.5">
                    {node.label}
                  </div>
                  <div className="text-xs text-gray-500">
                    {node.description}
                  </div>
                </div>
              </div>
            </div>
          );
        })}
      </div>
      
      <div className="mt-6 p-3 bg-flow-bg rounded-lg border border-gray-700">
        <div className="flex items-center gap-2 mb-2">
          <Code size={16} className="text-flow-accent" />
          <span className="text-xs font-semibold text-gray-400">PRO TIP</span>
        </div>
        <p className="text-xs text-gray-500">
          Connect nodes by dragging from output handles to input handles. 
          Right-click nodes for options.
        </p>
      </div>
    </div>
  );
}

export default NodePalette;