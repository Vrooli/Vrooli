import React, { useState, useCallback } from 'react';
import {
  Plus,
  GraduationCap,
  Sparkles,
  Video,
  LayoutGrid,
  BarChart3,
  ArrowRight,
  Circle,
} from 'lucide-react';
import { selectors } from '@constants/selectors';
import { FeatureShowcase } from './FeatureShowcase';

// ============================================
// TYPES
// ============================================

interface WelcomeHeroProps {
  onCreateFirstWorkflow: () => void;
  onOpenTutorial?: () => void;
  onStartRecording?: () => void;
}

interface FeatureCardProps {
  icon: React.ReactNode;
  title: string;
  description: string;
  delay: number;
  gradient: string;
  accentColor: string;
  isActive: boolean;
  onClick?: () => void;
}

// ============================================
// FEATURE CARD COMPONENT
// ============================================

const FeatureCard: React.FC<FeatureCardProps> = ({
  icon,
  title,
  description,
  delay,
  gradient,
  accentColor,
  isActive,
  onClick,
}) => (
  <div
    className={`feature-card feature-card-active accent-${accentColor} animate-fade-in-up group h-full ${
      isActive ? 'is-active' : ''
    }`}
    style={{ animationDelay: `${delay}ms` }}
    onClick={onClick}
    role={onClick ? 'button' : undefined}
    tabIndex={onClick ? 0 : undefined}
    onKeyDown={onClick ? (e) => {
      if (e.key === 'Enter' || e.key === ' ') {
        e.preventDefault();
        onClick();
      }
    } : undefined}
  >
    <div
      className={`relative h-full p-5 rounded-xl border border-flow-border/50 bg-flow-surface/50 backdrop-blur-sm transition-all duration-300 ${
        isActive
          ? 'border-transparent'
          : 'hover:border-flow-border'
      } ${onClick ? 'cursor-pointer' : ''}`}
    >
      {/* Icon container with gradient background */}
      <div
        className={`inline-flex items-center justify-center w-12 h-12 rounded-xl bg-gradient-to-br ${gradient} mb-4 transition-transform duration-300 ${
          isActive ? 'scale-110' : 'group-hover:scale-110'
        }`}
      >
        {icon}
      </div>
      <h3 className="text-base font-semibold text-white mb-2">{title}</h3>
      <p className="text-sm text-flow-text-muted leading-relaxed">{description}</p>

      {/* Active indicator */}
      {isActive && (
        <div className="absolute top-3 right-3">
          <span className="flex h-2 w-2">
            <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-flow-accent opacity-75" />
            <span className="relative inline-flex rounded-full h-2 w-2 bg-flow-accent" />
          </span>
        </div>
      )}
    </div>
  </div>
);

// ============================================
// FEATURE CONFIGURATION
// ============================================

const FEATURES = [
  {
    icon: <Sparkles size={24} className="text-white" />,
    title: 'AI-Powered',
    description: 'Describe tasks in plain English and let AI build the workflow for you automatically.',
    gradient: 'from-purple-500/20 to-pink-500/20',
    accentColor: 'purple',
  },
  {
    icon: <Video size={24} className="text-white" />,
    title: 'Record Mode',
    description: 'Browse naturally while your actions are tracked. Select a range and instantly create a workflow.',
    gradient: 'from-red-500/20 to-orange-500/20',
    accentColor: 'red',
  },
  {
    icon: <LayoutGrid size={24} className="text-white" />,
    title: 'Visual Builder',
    description: 'Drag-and-drop nodes to create complex automations without writing any code.',
    gradient: 'from-blue-500/20 to-cyan-500/20',
    accentColor: 'blue',
  },
  {
    icon: <BarChart3 size={24} className="text-white" />,
    title: 'Test & Monitor',
    description: 'Run automations in real-time and watch live results with detailed logging.',
    gradient: 'from-green-500/20 to-emerald-500/20',
    accentColor: 'green',
  },
];

// ============================================
// WELCOME HERO COMPONENT
// ============================================

export const WelcomeHero: React.FC<WelcomeHeroProps> = ({
  onCreateFirstWorkflow,
  onOpenTutorial,
  onStartRecording,
}) => {
  const [activeFeatureIndex, setActiveFeatureIndex] = useState(0);

  const handleActiveIndexChange = useCallback((index: number) => {
    setActiveFeatureIndex(index);
  }, []);

  return (
    <div className="relative flex-1 flex flex-col items-center justify-center px-4 py-12 overflow-hidden">
      {/* Animated background */}
      <div className="hero-background" />

      {/* Radial gradient overlay */}
      <div className="absolute inset-0 bg-gradient-radial from-blue-500/5 via-transparent to-transparent" />

      {/* Content container */}
      <div className="relative z-10 w-full max-w-5xl mx-auto">
        {/* Hero section */}
        <div className="text-center mb-12 animate-fade-in-up">
          {/* Animated logo */}
          <div className="relative inline-flex items-center justify-center mb-8">
            {/* Glow rings */}
            <div className="absolute inset-0 w-24 h-24 rounded-2xl bg-flow-accent/20 blur-xl animate-pulse-slow" />
            <div className="absolute inset-0 w-24 h-24 rounded-2xl bg-purple-500/10 blur-2xl animate-pulse-slower" />

            {/* Icon container */}
            <div className="relative w-20 h-20 flex items-center justify-center bg-gradient-to-br from-flow-accent/20 to-purple-500/20 rounded-2xl border border-flow-accent/30 shadow-lg shadow-flow-accent/20 overflow-hidden">
              <img
                src="/manifest-icon-192.maskable.png"
                alt="Vrooli Ascension logo"
                className="w-full h-full object-cover drop-shadow-[0_0_8px_rgba(59,130,246,0.5)]"
              />
            </div>
          </div>

          {/* Heading with gradient */}
          <h1 className="text-4xl sm:text-5xl font-bold mb-4">
            <span className="hero-gradient-text">
              Welcome to Vrooli Ascension
            </span>
          </h1>

          {/* Subtitle */}
          <p className="text-lg sm:text-xl text-flow-text-secondary max-w-2xl mx-auto mb-8 leading-relaxed">
            Build powerful browser automations in minutes — no code required.
            <br className="hidden sm:block" />
            <span className="text-flow-text-muted">Let AI generate workflows from plain English descriptions.</span>
          </p>

          {/* CTA Buttons */}
          <div className="flex flex-col sm:flex-row items-center justify-center gap-4 mb-12">
            <button
              data-testid={selectors.dashboard.createFirstWorkflowButton}
              onClick={onCreateFirstWorkflow}
              className="hero-button-primary group"
            >
              <Plus size={20} className="transition-transform group-hover:rotate-90 duration-300" />
              <span>Create Your First Workflow</span>
              <div className="hero-button-glow" />
            </button>

            {onStartRecording && (
              <button
                onClick={onStartRecording}
                className="inline-flex items-center gap-2 px-5 py-3 rounded-xl border border-red-500/40 bg-red-500/15 text-red-100 hover:bg-red-500/25 transition-colors"
              >
                <Circle size={18} className="text-red-300 fill-red-300" />
                <span>Start Recording</span>
              </button>
            )}

            {onOpenTutorial && (
              <button
                data-testid={selectors.dashboard.startTutorialButton}
                onClick={onOpenTutorial}
                className="hero-button-secondary group"
              >
                <GraduationCap size={20} className="transition-transform group-hover:scale-110 duration-300" />
                <span>Start Tutorial</span>
                <ArrowRight size={16} className="ml-1 opacity-0 -translate-x-2 group-hover:opacity-100 group-hover:translate-x-0 transition-all duration-300" />
              </button>
            )}
          </div>
        </div>

        {/* Feature cards */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-12">
          {FEATURES.map((feature, index) => (
            <FeatureCard
              key={feature.title}
              icon={feature.icon}
              title={feature.title}
              description={feature.description}
              delay={100 + index * 100}
              gradient={feature.gradient}
              accentColor={feature.accentColor}
              isActive={index === activeFeatureIndex}
            />
          ))}
        </div>

        {/* Feature showcase with cycling previews */}
        <div className="animate-fade-in-up" style={{ animationDelay: '600ms' }}>
          <FeatureShowcase onActiveIndexChange={handleActiveIndexChange} />
        </div>

        {/* Stats section */}
        <div className="mt-16 pt-8 border-t border-flow-border/30">
          <div className="grid grid-cols-3 gap-8 max-w-2xl mx-auto">
            <div className="text-center animate-fade-in-up" style={{ animationDelay: '800ms' }}>
              <div className="text-3xl font-bold hero-gradient-text mb-1">50+</div>
              <div className="text-sm text-flow-text-muted">Workflow Templates</div>
            </div>
            <div className="text-center animate-fade-in-up" style={{ animationDelay: '900ms' }}>
              <div className="text-3xl font-bold hero-gradient-text mb-1">10x</div>
              <div className="text-sm text-flow-text-muted">Faster Than Manual</div>
            </div>
            <div className="text-center animate-fade-in-up" style={{ animationDelay: '1000ms' }}>
              <div className="text-3xl font-bold hero-gradient-text mb-1">Zero</div>
              <div className="text-sm text-flow-text-muted">Code Required</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default WelcomeHero;
