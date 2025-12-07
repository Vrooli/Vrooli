import React from 'react';
import {
  Plus,
  Play,
  Sparkles,
  MousePointer2,
  LayoutGrid,
  BarChart3,
  Download,
  ArrowRight,
  Globe,
  FileText,
  CheckCircle2,
  Zap,
} from 'lucide-react';
import { selectors } from '@constants/selectors';

interface WelcomeHeroProps {
  onCreateProject: () => void;
  onTryDemo?: () => void;
}

// Animated workflow preview showing nodes connecting
const WorkflowPreview: React.FC = () => {
  return (
    <div className="relative w-full max-w-2xl mx-auto">
      {/* Glow effect behind the preview */}
      <div className="absolute inset-0 bg-gradient-to-r from-blue-500/20 via-purple-500/20 to-blue-500/20 blur-3xl" />

      {/* Preview container */}
      <div className="relative bg-flow-node/80 backdrop-blur-sm border border-flow-border/50 rounded-xl p-6 shadow-2xl">
        {/* Mini header */}
        <div className="flex items-center gap-2 mb-4 pb-3 border-b border-flow-border/50">
          <div className="flex gap-1.5">
            <div className="w-3 h-3 rounded-full bg-red-500/60" />
            <div className="w-3 h-3 rounded-full bg-yellow-500/60" />
            <div className="w-3 h-3 rounded-full bg-green-500/60" />
          </div>
          <span className="text-xs text-flow-text-muted ml-2">workflow-builder.tsx</span>
        </div>

        {/* Animated workflow nodes */}
        <div className="flex items-center justify-center gap-4 py-4">
          {/* Start Node */}
          <div className="workflow-preview-node animate-fade-in-up" style={{ animationDelay: '0ms' }}>
            <div className="flex items-center gap-2 px-4 py-3 bg-green-500/20 border border-green-500/40 rounded-lg">
              <Globe size={18} className="text-green-400" />
              <span className="text-sm font-medium text-green-300">Navigate</span>
            </div>
          </div>

          {/* Connection line 1 */}
          <div className="workflow-connection animate-draw-line" style={{ animationDelay: '200ms' }}>
            <ArrowRight size={20} className="text-flow-accent" />
          </div>

          {/* Action Node */}
          <div className="workflow-preview-node animate-fade-in-up" style={{ animationDelay: '400ms' }}>
            <div className="flex items-center gap-2 px-4 py-3 bg-blue-500/20 border border-blue-500/40 rounded-lg">
              <MousePointer2 size={18} className="text-blue-400" />
              <span className="text-sm font-medium text-blue-300">Click</span>
            </div>
          </div>

          {/* Connection line 2 */}
          <div className="workflow-connection animate-draw-line" style={{ animationDelay: '600ms' }}>
            <ArrowRight size={20} className="text-flow-accent" />
          </div>

          {/* Extract Node */}
          <div className="workflow-preview-node animate-fade-in-up" style={{ animationDelay: '800ms' }}>
            <div className="flex items-center gap-2 px-4 py-3 bg-purple-500/20 border border-purple-500/40 rounded-lg">
              <FileText size={18} className="text-purple-400" />
              <span className="text-sm font-medium text-purple-300">Extract</span>
            </div>
          </div>

          {/* Connection line 3 */}
          <div className="workflow-connection animate-draw-line" style={{ animationDelay: '1000ms' }}>
            <ArrowRight size={20} className="text-flow-accent" />
          </div>

          {/* Success Node */}
          <div className="workflow-preview-node animate-fade-in-up" style={{ animationDelay: '1200ms' }}>
            <div className="flex items-center gap-2 px-4 py-3 bg-emerald-500/20 border border-emerald-500/40 rounded-lg">
              <CheckCircle2 size={18} className="text-emerald-400" />
              <span className="text-sm font-medium text-emerald-300">Done</span>
            </div>
          </div>
        </div>

        {/* Execution indicator */}
        <div className="flex items-center justify-center gap-2 mt-4 pt-3 border-t border-flow-border/50">
          <div className="w-2 h-2 rounded-full bg-green-400 animate-pulse" />
          <span className="text-xs text-flow-text-muted">Workflow executing...</span>
        </div>
      </div>
    </div>
  );
};

// Feature card component
interface FeatureCardProps {
  icon: React.ReactNode;
  title: string;
  description: string;
  delay: number;
  gradient: string;
}

const FeatureCard: React.FC<FeatureCardProps> = ({ icon, title, description, delay, gradient }) => (
  <div
    className={`feature-card animate-fade-in-up group`}
    style={{ animationDelay: `${delay}ms` }}
  >
    <div className={`relative p-5 rounded-xl border border-flow-border/50 bg-flow-surface/50 backdrop-blur-sm hover:border-flow-border transition-all duration-300 hover:shadow-lg hover:shadow-${gradient}/10`}>
      {/* Icon container with gradient background */}
      <div className={`inline-flex items-center justify-center w-12 h-12 rounded-xl bg-gradient-to-br ${gradient} mb-4 group-hover:scale-110 transition-transform duration-300`}>
        {icon}
      </div>
      <h3 className="text-base font-semibold text-white mb-2">{title}</h3>
      <p className="text-sm text-flow-text-muted leading-relaxed">{description}</p>
    </div>
  </div>
);

export const WelcomeHero: React.FC<WelcomeHeroProps> = ({ onCreateProject, onTryDemo }) => {
  const features = [
    {
      icon: <Sparkles size={24} className="text-white" />,
      title: 'AI-Powered',
      description: 'Describe tasks in plain English and let AI build the workflow for you automatically.',
      gradient: 'from-purple-500/20 to-pink-500/20',
    },
    {
      icon: <LayoutGrid size={24} className="text-white" />,
      title: 'Visual Builder',
      description: 'Drag-and-drop nodes to create complex automations without writing any code.',
      gradient: 'from-blue-500/20 to-cyan-500/20',
    },
    {
      icon: <BarChart3 size={24} className="text-white" />,
      title: 'Test & Monitor',
      description: 'Run automations in real-time and watch live results with detailed logging.',
      gradient: 'from-green-500/20 to-emerald-500/20',
    },
    {
      icon: <Download size={24} className="text-white" />,
      title: 'Export Anywhere',
      description: 'Export workflows to code, JSON, or schedule automated runs on your terms.',
      gradient: 'from-orange-500/20 to-amber-500/20',
    },
  ];

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
            <div className="relative p-5 bg-gradient-to-br from-flow-accent/20 to-purple-500/20 rounded-2xl border border-flow-accent/30 shadow-lg shadow-flow-accent/20">
              <Zap size={48} className="text-flow-accent drop-shadow-[0_0_8px_rgba(59,130,246,0.5)]" />
            </div>
          </div>

          {/* Heading with gradient */}
          <h1 className="text-4xl sm:text-5xl font-bold mb-4">
            <span className="hero-gradient-text">
              Welcome to Browser Automation Studio
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
              data-testid={selectors.dashboard.newProjectButton}
              onClick={onCreateProject}
              className="hero-button-primary group"
            >
              <Plus size={20} className="transition-transform group-hover:rotate-90 duration-300" />
              <span>Create Your First Project</span>
              <div className="hero-button-glow" />
            </button>

            {onTryDemo && (
              <button
                data-testid={selectors.dashboard.tryDemoButton}
                onClick={onTryDemo}
                className="hero-button-secondary group"
              >
                <Play size={20} className="transition-transform group-hover:scale-110 duration-300" />
                <span>Try Demo Workflow</span>
                <ArrowRight size={16} className="ml-1 opacity-0 -translate-x-2 group-hover:opacity-100 group-hover:translate-x-0 transition-all duration-300" />
              </button>
            )}
          </div>
        </div>

        {/* Feature cards */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-12">
          {features.map((feature, index) => (
            <FeatureCard
              key={feature.title}
              icon={feature.icon}
              title={feature.title}
              description={feature.description}
              delay={100 + index * 100}
              gradient={feature.gradient}
            />
          ))}
        </div>

        {/* Workflow preview */}
        <div className="animate-fade-in-up" style={{ animationDelay: '600ms' }}>
          <div className="text-center mb-6">
            <span className="inline-flex items-center gap-2 px-3 py-1.5 text-xs font-medium text-flow-accent bg-flow-accent/10 border border-flow-accent/20 rounded-full">
              <span className="w-1.5 h-1.5 rounded-full bg-flow-accent animate-pulse" />
              See it in action
            </span>
          </div>
          <WorkflowPreview />
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
