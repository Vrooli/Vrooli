import React, { useState, useEffect, useCallback, useRef } from 'react';
import {
  Sparkles,
  Video,
  LayoutGrid,
  BarChart3,
  Globe,
  MousePointer2,
  FileText,
  CheckCircle2,
  ArrowRight,
  Type,
  Clock,
  Loader2,
  Terminal,
  CheckCircle,
  Circle,
  Pause,
} from 'lucide-react';

// Type alias for timeout refs
type TimeoutRef = ReturnType<typeof setTimeout> | null;

// ============================================
// TYPES & CONSTANTS
// ============================================

export interface FeatureConfig {
  id: string;
  title: string;
  label: string;
  icon: React.ReactNode;
  gradient: string;
  accentColor: string;
}

export const FEATURE_CONFIGS: FeatureConfig[] = [
  {
    id: 'ai-powered',
    title: 'AI-Powered',
    label: 'AI generates your workflow',
    icon: <Sparkles size={16} />,
    gradient: 'from-purple-500/20 to-pink-500/20',
    accentColor: 'purple',
  },
  {
    id: 'record-mode',
    title: 'Record Mode',
    label: 'Record your browser actions',
    icon: <Video size={16} />,
    gradient: 'from-red-500/20 to-orange-500/20',
    accentColor: 'red',
  },
  {
    id: 'visual-builder',
    title: 'Visual Builder',
    label: 'Build with drag-and-drop',
    icon: <LayoutGrid size={16} />,
    gradient: 'from-blue-500/20 to-cyan-500/20',
    accentColor: 'blue',
  },
  {
    id: 'test-monitor',
    title: 'Test & Monitor',
    label: 'Watch executions live',
    icon: <BarChart3 size={16} />,
    gradient: 'from-green-500/20 to-emerald-500/20',
    accentColor: 'green',
  },
];

const CYCLE_DURATION = 6000; // 6 seconds per preview
const ANIMATION_DURATION = 500; // Transition duration in ms

// ============================================
// PREVIEW CONTAINER (Shared wrapper)
// ============================================

interface PreviewContainerProps {
  children: React.ReactNode;
  headerText: string;
  footerContent?: React.ReactNode;
}

const PreviewContainer: React.FC<PreviewContainerProps> = ({
  children,
  headerText,
  footerContent,
}) => (
  <div className="relative bg-flow-node/80 backdrop-blur-sm border border-flow-border/50 rounded-xl p-6 shadow-2xl">
    {/* Mini header */}
    <div className="flex items-center gap-2 mb-4 pb-3 border-b border-flow-border/50">
      <div className="flex gap-1.5">
        <div className="w-3 h-3 rounded-full bg-red-500/60" />
        <div className="w-3 h-3 rounded-full bg-yellow-500/60" />
        <div className="w-3 h-3 rounded-full bg-green-500/60" />
      </div>
      <span className="text-xs text-flow-text-muted ml-2">{headerText}</span>
    </div>

    {/* Content */}
    <div className="min-h-[140px] flex flex-col justify-center">
      {children}
    </div>

    {/* Footer */}
    {footerContent && (
      <div className="flex items-center justify-center gap-2 mt-4 pt-3 border-t border-flow-border/50">
        {footerContent}
      </div>
    )}
  </div>
);

// ============================================
// AI-POWERED PREVIEW
// ============================================

const AI_PROMPT = "Log into Amazon and add the first search result for 'wireless mouse' to cart";

const AIPreview: React.FC<{ isActive: boolean }> = ({ isActive }) => {
  const [displayedText, setDisplayedText] = useState('');
  const [isGenerating, setIsGenerating] = useState(false);
  const [generatedNodes, setGeneratedNodes] = useState<string[]>([]);
  const timeoutRef = useRef<TimeoutRef>(null);

  // Reset and start animation when becoming active
  useEffect(() => {
    if (!isActive) {
      // Reset state when not active
      setDisplayedText('');
      setIsGenerating(false);
      setGeneratedNodes([]);
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
      return;
    }

    // Typing animation
    let charIndex = 0;
    const typeNextChar = () => {
      if (charIndex < AI_PROMPT.length) {
        setDisplayedText(AI_PROMPT.slice(0, charIndex + 1));
        charIndex++;
        timeoutRef.current = setTimeout(typeNextChar, 35);
      } else {
        // Start generating after typing completes
        timeoutRef.current = setTimeout(() => {
          setIsGenerating(true);
          // Show nodes one by one
          const nodes = ['Navigate', 'Search', 'Click', 'Add to Cart'];
          nodes.forEach((node, i) => {
            timeoutRef.current = setTimeout(() => {
              setGeneratedNodes(prev => [...prev, node]);
              if (i === nodes.length - 1) {
                setTimeout(() => setIsGenerating(false), 300);
              }
            }, 400 * (i + 1));
          });
        }, 500);
      }
    };

    timeoutRef.current = setTimeout(typeNextChar, 300);

    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
    };
  }, [isActive]);

  return (
    <PreviewContainer
      headerText="ai-workflow-generator"
      footerContent={
        isGenerating ? (
          <>
            <Loader2 size={14} className="text-purple-400 animate-spin" />
            <span className="text-xs text-flow-text-muted">Generating workflow...</span>
          </>
        ) : generatedNodes.length > 0 ? (
          <>
            <CheckCircle size={14} className="text-green-400" />
            <span className="text-xs text-flow-text-muted">Workflow generated!</span>
          </>
        ) : (
          <>
            <Sparkles size={14} className="text-purple-400" />
            <span className="text-xs text-flow-text-muted">Describe your automation...</span>
          </>
        )
      }
    >
      {/* Prompt input area */}
      <div className="mb-4">
        <div className="flex items-start gap-3 p-3 bg-purple-500/10 border border-purple-500/30 rounded-lg">
          <Sparkles size={18} className="text-purple-400 mt-0.5 flex-shrink-0" />
          <div className="flex-1 min-h-[24px]">
            <span className="text-sm text-purple-200">{displayedText}</span>
            {displayedText.length < AI_PROMPT.length && (
              <span className="inline-block w-0.5 h-4 bg-purple-400 ml-0.5 animate-pulse" />
            )}
          </div>
        </div>
      </div>

      {/* Generated nodes */}
      {generatedNodes.length > 0 && (
        <div className="flex items-center justify-center gap-2 flex-wrap">
          {generatedNodes.map((node, index) => (
            <React.Fragment key={node}>
              <div
                className="flex items-center gap-1.5 px-3 py-1.5 bg-purple-500/20 border border-purple-500/40 rounded-md animate-scale-in"
                style={{ animationDelay: `${index * 100}ms` }}
              >
                <span className="text-xs font-medium text-purple-300">{node}</span>
              </div>
              {index < generatedNodes.length - 1 && (
                <ArrowRight size={14} className="text-purple-400/60 animate-fade-in" />
              )}
            </React.Fragment>
          ))}
        </div>
      )}
    </PreviewContainer>
  );
};

// ============================================
// RECORD MODE PREVIEW
// ============================================

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

const RecordModePreview: React.FC<{ isActive: boolean }> = ({ isActive }) => {
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

    // Start recording indicator
    setIsRecording(true);

    // Show actions one by one
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
      {/* Recording indicator */}
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

      {/* Action timeline */}
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

// ============================================
// VISUAL BUILDER PREVIEW
// ============================================

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

const VisualBuilderPreview: React.FC<{ isActive: boolean }> = ({ isActive }) => {
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

    // Show nodes one by one
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
      {/* Workflow nodes */}
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

// ============================================
// TEST & MONITOR PREVIEW
// ============================================

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

const TestMonitorPreview: React.FC<{ isActive: boolean }> = ({ isActive }) => {
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

    // Execute steps one by one
    let stepIndex = 0;
    const executeNextStep = () => {
      if (stepIndex < EXECUTION_STEPS.length) {
        // Set current step to running
        setStepStatuses(prev => {
          const newStatuses = [...prev];
          newStatuses[stepIndex] = 'running';
          return newStatuses;
        });

        // After a delay, mark as passed and move to next
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
      {/* Progress bar */}
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

      {/* Step list */}
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

// ============================================
// NAVIGATION DOTS
// ============================================

interface NavigationDotsProps {
  total: number;
  active: number;
  onSelect: (index: number) => void;
  isPaused: boolean;
}

const NavigationDots: React.FC<NavigationDotsProps> = ({
  total,
  active,
  onSelect,
  isPaused,
}) => (
  <div className="flex items-center justify-center gap-2 mt-4">
    {Array.from({ length: total }).map((_, index) => (
      <button
        key={index}
        onClick={() => onSelect(index)}
        className={`relative h-2 rounded-full transition-all duration-300 ${
          index === active
            ? 'w-8 bg-flow-accent'
            : 'w-2 bg-flow-border hover:bg-flow-text-muted'
        }`}
        aria-label={`Go to preview ${index + 1}`}
      >
        {index === active && !isPaused && (
          <span
            className="absolute inset-0 bg-white/30 rounded-full origin-left animate-progress-bar"
            style={{ animationDuration: `${CYCLE_DURATION}ms` }}
          />
        )}
      </button>
    ))}
    {isPaused && (
      <div className="flex items-center gap-1 ml-2 text-xs text-flow-text-muted">
        <Pause size={12} />
        <span>Paused</span>
      </div>
    )}
  </div>
);

// ============================================
// FEATURE SHOWCASE (Main Component)
// ============================================

interface FeatureShowcaseProps {
  onActiveIndexChange?: (index: number) => void;
}

export const FeatureShowcase: React.FC<FeatureShowcaseProps> = ({
  onActiveIndexChange,
}) => {
  const [activeIndex, setActiveIndex] = useState(0);
  const [isPaused, setIsPaused] = useState(false);
  const [isTransitioning, setIsTransitioning] = useState(false);
  const intervalRef = useRef<TimeoutRef>(null);

  // Notify parent of active index changes
  useEffect(() => {
    onActiveIndexChange?.(activeIndex);
  }, [activeIndex, onActiveIndexChange]);

  // Auto-cycle logic
  useEffect(() => {
    if (isPaused) {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
        intervalRef.current = null;
      }
      return;
    }

    intervalRef.current = setInterval(() => {
      setIsTransitioning(true);
      setTimeout(() => {
        setActiveIndex(prev => (prev + 1) % FEATURE_CONFIGS.length);
        setIsTransitioning(false);
      }, ANIMATION_DURATION / 2);
    }, CYCLE_DURATION);

    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
    };
  }, [isPaused]);

  const handleManualSelect = useCallback((index: number) => {
    setIsTransitioning(true);
    setTimeout(() => {
      setActiveIndex(index);
      setIsTransitioning(false);
    }, ANIMATION_DURATION / 2);
  }, []);

  const handleMouseEnter = useCallback(() => {
    setIsPaused(true);
  }, []);

  const handleMouseLeave = useCallback(() => {
    setIsPaused(false);
  }, []);

  const activeFeature = FEATURE_CONFIGS[activeIndex];

  // Preview components mapping
  const previews = [
    <AIPreview key="ai" isActive={activeIndex === 0 && !isTransitioning} />,
    <RecordModePreview key="record" isActive={activeIndex === 1 && !isTransitioning} />,
    <VisualBuilderPreview key="visual" isActive={activeIndex === 2 && !isTransitioning} />,
    <TestMonitorPreview key="test" isActive={activeIndex === 3 && !isTransitioning} />,
  ];

  return (
    <div
      className="w-full"
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
    >
      {/* Label badge */}
      <div className="text-center mb-6">
        <span
          className={`inline-flex items-center gap-2 px-3 py-1.5 text-xs font-medium rounded-full transition-all duration-300 ${
            getAccentClasses(activeFeature.accentColor)
          }`}
        >
          <span className={`w-1.5 h-1.5 rounded-full animate-pulse ${
            getAccentDotClass(activeFeature.accentColor)
          }`} />
          {activeFeature.label}
        </span>
      </div>

      {/* Preview container with transition */}
      <div className="relative w-full max-w-2xl mx-auto">
        {/* Glow effect that changes color based on active feature */}
        <div
          className={`absolute inset-0 blur-3xl transition-colors duration-500 ${
            getGlowClass(activeFeature.accentColor)
          }`}
        />

        {/* Preview with crossfade */}
        <div
          className={`relative transition-all duration-300 ${
            isTransitioning ? 'opacity-0 scale-98' : 'opacity-100 scale-100'
          }`}
        >
          {previews[activeIndex]}
        </div>
      </div>

      {/* Navigation dots */}
      <NavigationDots
        total={FEATURE_CONFIGS.length}
        active={activeIndex}
        onSelect={handleManualSelect}
        isPaused={isPaused}
      />
    </div>
  );
};

// ============================================
// UTILITY FUNCTIONS
// ============================================

function getAccentClasses(color: string): string {
  const classes: Record<string, string> = {
    purple: 'text-purple-400 bg-purple-500/10 border border-purple-500/20',
    red: 'text-red-400 bg-red-500/10 border border-red-500/20',
    blue: 'text-blue-400 bg-blue-500/10 border border-blue-500/20',
    green: 'text-green-400 bg-green-500/10 border border-green-500/20',
  };
  return classes[color] || classes.blue;
}

function getAccentDotClass(color: string): string {
  const classes: Record<string, string> = {
    purple: 'bg-purple-400',
    red: 'bg-red-400',
    blue: 'bg-blue-400',
    green: 'bg-green-400',
  };
  return classes[color] || classes.blue;
}

function getGlowClass(color: string): string {
  const classes: Record<string, string> = {
    purple: 'bg-gradient-to-r from-purple-500/20 via-pink-500/20 to-purple-500/20',
    red: 'bg-gradient-to-r from-red-500/20 via-orange-500/20 to-red-500/20',
    blue: 'bg-gradient-to-r from-blue-500/20 via-cyan-500/20 to-blue-500/20',
    green: 'bg-gradient-to-r from-green-500/20 via-emerald-500/20 to-green-500/20',
  };
  return classes[color] || classes.blue;
}

export default FeatureShowcase;
