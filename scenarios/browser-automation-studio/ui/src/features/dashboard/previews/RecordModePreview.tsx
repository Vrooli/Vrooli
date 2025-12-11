import React, { useEffect, useRef, useState } from 'react';
import { ArrowRight, Circle, Clock, Globe, MousePointer2, Type } from 'lucide-react';
import PreviewContainer from './PreviewContainer';

type TimeoutRef = ReturnType<typeof setTimeout> | null;

interface RecordedAction {
  type: 'navigate' | 'click' | 'type' | 'scroll';
  target: string;
  timestamp: string;
}

const RECORDED_ACTIONS: RecordedAction[] = [
  { type: 'navigate', target: 'amazon.com', timestamp: '0:00' },
  { type: 'click', target: '#search-box', timestamp: '0:02' },
  { type: 'type', target: '"wireless mouse"', timestamp: '0:03' },
  { type: 'click', target: 'Search button', timestamp: '0:05' },
  { type: 'click', target: 'First result', timestamp: '0:08' },
];

const ACTION_ICONS: Record<RecordedAction['type'], React.ReactNode> = {
  navigate: <Globe size={12} className="text-green-400" />,
  click: <MousePointer2 size={12} className="text-blue-400" />,
  type: <Type size={12} className="text-yellow-400" />,
  scroll: <ArrowRight size={12} className="text-gray-400" />,
};

export const RecordModePreview: React.FC<{ isActive: boolean }> = ({ isActive }) => {
  const [visibleActions, setVisibleActions] = useState<number>(0);
  const [isRecording, setIsRecording] = useState(false);
  const timeoutRef = useRef<TimeoutRef>(null);

  useEffect(() => {
    if (!isActive) {
      setVisibleActions(0);
      setIsRecording(false);
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
      return;
    }

    setIsRecording(true);

    let actionIndex = 0;
    const showNextAction = () => {
      if (actionIndex < RECORDED_ACTIONS.length) {
        setVisibleActions(actionIndex + 1);
        actionIndex++;
        timeoutRef.current = setTimeout(showNextAction, 800);
      }
    };

    timeoutRef.current = setTimeout(showNextAction, 600);

    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
    };
  }, [isActive]);

  return (
    <PreviewContainer
      headerText="record-session"
      footerContent={
        <>
          {isRecording && (
            <div className="w-2 h-2 rounded-full bg-red-500 animate-pulse" />
          )}
          <span className="text-xs text-flow-text-muted">
            {isRecording ? 'Recording in progress...' : 'Ready to record'}
          </span>
        </>
      }
    >
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <div className={`flex items-center gap-1.5 px-2 py-1 rounded-md ${
            isRecording ? 'bg-red-500/20 border border-red-500/40' : 'bg-flow-surface'
          }`}>
            {isRecording ? (
              <>
                <div className="w-2 h-2 rounded-full bg-red-500 animate-pulse" />
                <span className="text-xs font-medium text-red-400">REC</span>
              </>
            ) : (
              <>
                <Circle size={10} className="text-gray-500" />
                <span className="text-xs text-gray-500">IDLE</span>
              </>
            )}
          </div>
        </div>
        <div className="flex items-center gap-1 text-xs text-flow-text-muted">
          <Clock size={12} />
          <span>{RECORDED_ACTIONS[visibleActions - 1]?.timestamp || '0:00'}</span>
        </div>
      </div>

      <div className="space-y-1.5 max-h-[100px] overflow-hidden">
        {RECORDED_ACTIONS.slice(0, visibleActions).map((action, index) => (
          <div
            key={index}
            className="flex items-center gap-2 px-2 py-1.5 bg-flow-surface/50 rounded-md animate-slide-in-right text-xs"
            style={{ animationDelay: `${index * 50}ms` }}
          >
            <span className="text-flow-text-muted w-8">{action.timestamp}</span>
            <div className="flex items-center gap-1.5">
              {ACTION_ICONS[action.type]}
              <span className="text-flow-text-secondary capitalize">{action.type}</span>
            </div>
            <span className="text-flow-text-muted truncate">{action.target}</span>
          </div>
        ))}
      </div>
    </PreviewContainer>
  );
};

export default RecordModePreview;
