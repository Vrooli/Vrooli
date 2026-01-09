import React, { useEffect, useRef, useState } from 'react';
import { CheckCircle, Circle, Loader2, Terminal } from 'lucide-react';
import PreviewContainer from './PreviewContainer';

type TimeoutRef = ReturnType<typeof setTimeout> | null;

interface ExecutionStep {
  name: string;
  status: 'pending' | 'running' | 'passed' | 'failed';
  duration?: string;
}

const EXECUTION_STEPS: Omit<ExecutionStep, 'status'>[] = [
  { name: 'Navigate to page', duration: '234ms' },
  { name: 'Click search box', duration: '89ms' },
  { name: 'Enter search term', duration: '156ms' },
  { name: 'Submit search', duration: '342ms' },
  { name: 'Verify results', duration: '178ms' },
];

export const TestMonitorPreview: React.FC<{ isActive: boolean }> = ({ isActive }) => {
  const [stepStatuses, setStepStatuses] = useState<ExecutionStep['status'][]>(
    EXECUTION_STEPS.map(() => 'pending')
  );
  const [progress, setProgress] = useState(0);
  const timeoutRef = useRef<TimeoutRef>(null);

  useEffect(() => {
    if (!isActive) {
      setStepStatuses(EXECUTION_STEPS.map(() => 'pending'));
      setProgress(0);
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
      return;
    }

    let stepIndex = 0;
    const executeNextStep = () => {
      if (stepIndex < EXECUTION_STEPS.length) {
        setStepStatuses(prev => {
          const newStatuses = [...prev];
          newStatuses[stepIndex] = 'running';
          return newStatuses;
        });

        timeoutRef.current = setTimeout(() => {
          setStepStatuses(prev => {
            const newStatuses = [...prev];
            newStatuses[stepIndex] = 'passed';
            return newStatuses;
          });
          setProgress(((stepIndex + 1) / EXECUTION_STEPS.length) * 100);
          stepIndex++;
          timeoutRef.current = setTimeout(executeNextStep, 300);
        }, 600);
      }
    };

    timeoutRef.current = setTimeout(executeNextStep, 400);

    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
    };
  }, [isActive]);

  const getStatusIcon = (status: ExecutionStep['status']) => {
    switch (status) {
      case 'passed':
        return <CheckCircle size={12} className="text-green-400" />;
      case 'running':
        return <Loader2 size={12} className="text-blue-400 animate-spin" />;
      case 'failed':
        return <Circle size={12} className="text-red-400" />;
      default:
        return <Circle size={12} className="text-gray-500" />;
    }
  };

  const allPassed = stepStatuses.every(s => s === 'passed');

  return (
    <PreviewContainer
      headerText="execution-monitor"
      footerContent={
        allPassed ? (
          <>
            <CheckCircle size={14} className="text-green-400" />
            <span className="text-xs text-green-400">All steps passed!</span>
            <span className="text-xs text-flow-text-muted ml-2">Total: 999ms</span>
          </>
        ) : (
          <>
            <Terminal size={14} className="text-flow-accent" />
            <span className="text-xs text-flow-text-muted">Running workflow...</span>
          </>
        )
      }
    >
      <div className="mb-3">
        <div className="flex items-center justify-between text-xs mb-1">
          <span className="text-flow-text-muted">Progress</span>
          <span className="text-flow-text-secondary">{Math.round(progress)}%</span>
        </div>
        <div className="h-1.5 bg-flow-surface rounded-full overflow-hidden">
          <div
            className="h-full bg-gradient-to-r from-green-500 to-emerald-400 rounded-full transition-all duration-500 ease-out"
            style={{ width: `${progress}%` }}
          />
        </div>
      </div>

      <div className="space-y-1 max-h-[85px] overflow-hidden">
        {EXECUTION_STEPS.map((step, index) => (
          <div
            key={step.name}
            className={`flex items-center justify-between px-2 py-1 rounded text-xs transition-colors ${
              stepStatuses[index] === 'running' ? 'bg-blue-500/10' : ''
            }`}
          >
            <div className="flex items-center gap-2">
              {getStatusIcon(stepStatuses[index])}
              <span className={`${
                stepStatuses[index] === 'passed' ? 'text-flow-text-secondary' :
                stepStatuses[index] === 'running' ? 'text-blue-300' :
                'text-flow-text-muted'
              }`}>
                {step.name}
              </span>
            </div>
            {stepStatuses[index] === 'passed' && (
              <span className="text-flow-text-muted">{step.duration}</span>
            )}
          </div>
        ))}
      </div>
    </PreviewContainer>
  );
};

export default TestMonitorPreview;
