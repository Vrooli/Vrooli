import React, { useState, useEffect, useCallback, useRef } from 'react';
import { Pause, Sparkles } from 'lucide-react';
import { PREVIEW_RENDERERS } from './FeaturePreviews';
import { FEATURE_CONFIGS, type FeatureConfig } from './featureConfigs';

// ============================================
// TYPES & CONSTANTS
// ============================================

const CYCLE_DURATION = 6000; // 6 seconds per preview
const ANIMATION_DURATION = 500; // Transition duration in ms

// Default feature for type safety (used when array index returns undefined)
const DEFAULT_FEATURE: FeatureConfig = FEATURE_CONFIGS[0] ?? {
  id: 'ai-powered',
  title: 'AI-Powered',
  label: 'AI generates your workflow',
  icon: <Sparkles size={16} />,
  gradient: 'from-purple-500/20 to-pink-500/20',
  accentColor: 'purple',
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
  activeIndex?: number;
  onActiveIndexChange?: (index: number) => void;
}

export const FeatureShowcase: React.FC<FeatureShowcaseProps> = ({
  activeIndex: controlledActiveIndex,
  onActiveIndexChange,
}) => {
  const [internalActiveIndex, setInternalActiveIndex] = useState(controlledActiveIndex ?? 0);
  const [isPaused, setIsPaused] = useState(false);
  const [isTransitioning, setIsTransitioning] = useState(false);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const activeIndex = controlledActiveIndex ?? internalActiveIndex;

  useEffect(() => {
    if (controlledActiveIndex === undefined) return;
    setInternalActiveIndex(controlledActiveIndex);
  }, [controlledActiveIndex]);

  useEffect(() => {
    if (controlledActiveIndex !== undefined) return;
    onActiveIndexChange?.(activeIndex);
  }, [activeIndex, controlledActiveIndex, onActiveIndexChange]);

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
        const nextIndex = (activeIndex + 1) % FEATURE_CONFIGS.length;
        if (controlledActiveIndex === undefined) {
          setInternalActiveIndex(nextIndex);
        } else {
          onActiveIndexChange?.(nextIndex);
        }
        setIsTransitioning(false);
      }, ANIMATION_DURATION / 2);
    }, CYCLE_DURATION);

    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
    };
  }, [activeIndex, controlledActiveIndex, isPaused, onActiveIndexChange]);

  const handleManualSelect = useCallback((index: number) => {
    setIsTransitioning(true);
    setTimeout(() => {
      if (controlledActiveIndex === undefined) {
        setInternalActiveIndex(index);
      } else {
        onActiveIndexChange?.(index);
      }
      setIsTransitioning(false);
    }, ANIMATION_DURATION / 2);
  }, [controlledActiveIndex, onActiveIndexChange]);

  const handleMouseEnter = useCallback(() => {
    setIsPaused(true);
  }, []);

  const handleMouseLeave = useCallback(() => {
    setIsPaused(false);
  }, []);

  const activeFeature = FEATURE_CONFIGS[activeIndex] ?? DEFAULT_FEATURE;
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
  return classes[color] ?? 'text-blue-400 bg-blue-500/10 border border-blue-500/20';
}

function getAccentDotClass(color: string): string {
  const classes: Record<string, string> = {
    purple: 'bg-purple-400',
    red: 'bg-red-400',
    blue: 'bg-blue-400',
    green: 'bg-green-400',
    amber: 'bg-amber-400',
  };
  return classes[color] ?? 'bg-blue-400';
}

function getGlowClass(color: string): string {
  const classes: Record<string, string> = {
    purple: 'bg-gradient-to-r from-purple-500/20 via-pink-500/20 to-purple-500/20',
    red: 'bg-gradient-to-r from-red-500/20 via-orange-500/20 to-red-500/20',
    blue: 'bg-gradient-to-r from-blue-500/20 via-cyan-500/20 to-blue-500/20',
    green: 'bg-gradient-to-r from-green-500/20 via-emerald-500/20 to-green-500/20',
    amber: 'bg-gradient-to-r from-amber-500/20 via-orange-500/20 to-amber-500/20',
  };
  return classes[color] ?? 'bg-gradient-to-r from-blue-500/20 via-cyan-500/20 to-blue-500/20';
}

export default FeatureShowcase;
