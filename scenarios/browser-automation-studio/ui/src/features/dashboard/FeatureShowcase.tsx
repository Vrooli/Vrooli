import React, { useState, useEffect, useCallback, useRef } from 'react';
import {
  Sparkles,
  Video,
  LayoutGrid,
  BarChart3,
  ArrowDownToLine,
  Pause,
} from 'lucide-react';
import { PREVIEW_RENDERERS } from './FeaturePreviews';

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
  {
    id: 'exports',
    title: 'Exports',
    label: 'Style & share replays',
    icon: <ArrowDownToLine size={16} />,
    gradient: 'from-amber-500/20 to-orange-500/20',
    accentColor: 'amber',
  },
];

const CYCLE_DURATION = 6000; // 6 seconds per preview
const ANIMATION_DURATION = 500; // Transition duration in ms

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
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    onActiveIndexChange?.(activeIndex);
  }, [activeIndex, onActiveIndexChange]);

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
  const previews = PREVIEW_RENDERERS.map((render, index) =>
    render(activeIndex === index && !isTransitioning)
  );

  return (
    <div
      className="w-full"
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
    >
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

      <div className="relative w-full max-w-2xl mx-auto">
        <div
          className={`absolute inset-0 blur-3xl transition-colors duration-500 ${
            getGlowClass(activeFeature.accentColor)
          }`}
        />

        <div
          className={`relative transition-all duration-300 ${
            isTransitioning ? 'opacity-0 scale-98' : 'opacity-100 scale-100'
          }`}
        >
          {previews[activeIndex]}
        </div>
      </div>

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
    amber: 'text-amber-400 bg-amber-500/10 border border-amber-500/20',
  };
  return classes[color] || classes.blue;
}

function getAccentDotClass(color: string): string {
  const classes: Record<string, string> = {
    purple: 'bg-purple-400',
    red: 'bg-red-400',
    blue: 'bg-blue-400',
    green: 'bg-green-400',
    amber: 'bg-amber-400',
  };
  return classes[color] || classes.blue;
}

function getGlowClass(color: string): string {
  const classes: Record<string, string> = {
    purple: 'bg-gradient-to-r from-purple-500/20 via-pink-500/20 to-purple-500/20',
    red: 'bg-gradient-to-r from-red-500/20 via-orange-500/20 to-red-500/20',
    blue: 'bg-gradient-to-r from-blue-500/20 via-cyan-500/20 to-blue-500/20',
    green: 'bg-gradient-to-r from-green-500/20 via-emerald-500/20 to-green-500/20',
    amber: 'bg-gradient-to-r from-amber-500/20 via-orange-500/20 to-amber-500/20',
  };
  return classes[color] || classes.blue;
}

export default FeatureShowcase;
