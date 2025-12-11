import React, { useEffect, useRef, useState } from 'react';
import { ArrowRight, CheckCircle2, FileText, Globe, LayoutGrid, MousePointer2 } from 'lucide-react';
import PreviewContainer from './PreviewContainer';

type TimeoutRef = ReturnType<typeof setTimeout> | null;

interface WorkflowNode {
  icon: React.ReactNode;
  label: string;
  colorClass: string;
}

const WORKFLOW_NODES: WorkflowNode[] = [
  { icon: <Globe size={16} />, label: 'Navigate', colorClass: 'green' },
  { icon: <MousePointer2 size={16} />, label: 'Click', colorClass: 'blue' },
  { icon: <FileText size={16} />, label: 'Extract', colorClass: 'purple' },
  { icon: <CheckCircle2 size={16} />, label: 'Done', colorClass: 'emerald' },
];

export const VisualBuilderPreview: React.FC<{ isActive: boolean }> = ({ isActive }) => {
  const [visibleNodes, setVisibleNodes] = useState<number>(0);
  const timeoutRef = useRef<TimeoutRef>(null);

  useEffect(() => {
    if (!isActive) {
      setVisibleNodes(0);
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
      return;
    }

    let nodeIndex = 0;
    const showNextNode = () => {
      if (nodeIndex <= WORKFLOW_NODES.length) {
        setVisibleNodes(nodeIndex);
        nodeIndex++;
        timeoutRef.current = setTimeout(showNextNode, 400);
      }
    };

    timeoutRef.current = setTimeout(showNextNode, 300);

    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
    };
  }, [isActive]);

  const getNodeColorClasses = (color: string) => {
    const colors: Record<string, { bg: string; border: string; text: string }> = {
      green: { bg: 'bg-green-500/20', border: 'border-green-500/40', text: 'text-green-300' },
      blue: { bg: 'bg-blue-500/20', border: 'border-blue-500/40', text: 'text-blue-300' },
      purple: { bg: 'bg-purple-500/20', border: 'border-purple-500/40', text: 'text-purple-300' },
      emerald: { bg: 'bg-emerald-500/20', border: 'border-emerald-500/40', text: 'text-emerald-300' },
    };
    return colors[color] || colors.blue;
  };

  return (
    <PreviewContainer
      headerText="workflow-builder.tsx"
      footerContent={
        visibleNodes >= WORKFLOW_NODES.length ? (
          <>
            <div className="w-2 h-2 rounded-full bg-green-400 animate-pulse" />
            <span className="text-xs text-flow-text-muted">Workflow executing...</span>
          </>
        ) : (
          <>
            <LayoutGrid size={14} className="text-blue-400" />
            <span className="text-xs text-flow-text-muted">Drag and drop to build...</span>
          </>
        )
      }
    >
      <div className="flex items-center justify-center gap-3 py-2">
        {WORKFLOW_NODES.map((node, index) => {
          const colors = getNodeColorClasses(node.colorClass);
          const isVisible = index < visibleNodes;
          const showConnection = index < visibleNodes - 1;

          return (
            <React.Fragment key={node.label}>
              <div
                className={`transition-all duration-300 ${
                  isVisible ? 'opacity-100 scale-100' : 'opacity-0 scale-75'
                }`}
              >
                <div className={`flex items-center gap-2 px-3 py-2 ${colors.bg} border ${colors.border} rounded-lg`}>
                  <span className={colors.text}>{node.icon}</span>
                  <span className={`text-sm font-medium ${colors.text}`}>{node.label}</span>
                </div>
              </div>
              {index < WORKFLOW_NODES.length - 1 && (
                <div className={`transition-all duration-300 ${
                  showConnection ? 'opacity-100' : 'opacity-0'
                }`}>
                  <ArrowRight size={18} className="text-flow-accent" />
                </div>
              )}
            </React.Fragment>
          );
        })}
      </div>
    </PreviewContainer>
  );
};

export default VisualBuilderPreview;
