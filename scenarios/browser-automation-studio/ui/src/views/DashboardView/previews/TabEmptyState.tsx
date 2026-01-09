import React from 'react';

interface Feature {
  title: string;
  description: string;
  icon: React.ReactNode;
}

interface CTA {
  label: string;
  onClick: () => void;
}

interface ProgressStep {
  label: string;
  active?: boolean;
  completed?: boolean;
}

interface TabEmptyStateProps {
  icon: React.ReactNode;
  title: string;
  subtitle: string;
  preview: React.ReactNode;
  primaryCta: CTA;
  secondaryCta?: CTA;
  features?: Feature[];
  progressPath?: ProgressStep[];
  variant?: 'standard' | 'polished';
}

export const TabEmptyState: React.FC<TabEmptyStateProps> = ({
  icon,
  title,
  subtitle,
  preview,
  primaryCta,
  secondaryCta,
  features = [],
  progressPath = [],
  variant = 'standard',
}) => {
  if (variant === 'polished') {
    return (
      <div className="relative overflow-hidden rounded-2xl border border-flow-border/40 bg-transparent">
        <div className="grid gap-6 p-6 sm:p-8 lg:grid-cols-[1.15fr_0.85fr] lg:items-start">
          {/* Content */}
          <div className="space-y-6 max-w-xl">
            <div className="flex items-start gap-3">
              <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-gradient-to-br from-flow-accent/20 to-purple-500/15 text-surface border border-flow-border/60 shadow-md shadow-blue-500/10">
                {icon}
              </div>
              <div className="space-y-1">
                <div className="text-lg font-semibold text-surface leading-tight hero-gradient-text">
                  {title}
                </div>
                <p className="text-sm text-flow-text-muted leading-snug max-w-md">
                  {subtitle}
                </p>
              </div>
            </div>

            {/* CTAs */}
            <div className="flex flex-col sm:flex-row sm:items-center gap-3">
              <button
                onClick={primaryCta.onClick}
                className="hero-button-primary btn-press w-full sm:w-auto justify-center shadow-lg shadow-blue-500/20"
              >
                {primaryCta.label}
                <div className="hero-button-glow" />
              </button>
              {secondaryCta && (
                <button
                  onClick={secondaryCta.onClick}
                  className="hero-button-secondary w-full sm:w-auto justify-center"
                >
                  {secondaryCta.label}
                </button>
              )}
            </div>

            {/* Progress path */}
            {progressPath.length > 0 && (
              <div className="flex flex-wrap items-center gap-2 text-xs text-flow-text-muted">
                <span className="text-[10px] uppercase tracking-wide text-flow-text-secondary mr-1">
                  Next steps
                </span>
                {progressPath.map((step, idx) => {
                  const circleClasses = step.active
                    ? 'bg-flow-accent text-white border-flow-accent/70'
                    : step.completed
                      ? 'bg-emerald-500/20 text-emerald-200 border-emerald-400/60'
                      : 'bg-flow-node/60 text-flow-text-muted border-flow-border/60';

                  return (
                    <div key={step.label} className="flex items-center gap-2">
                      <div
                        className={`flex h-5 w-5 items-center justify-center rounded-full border text-[10px] font-semibold ${circleClasses}`}
                      >
                        {step.completed ? '✓' : idx + 1}
                      </div>
                      <span
                        className={`font-medium ${
                          step.active
                            ? 'text-surface'
                            : step.completed
                              ? 'text-emerald-100'
                              : ''
                        }`}
                      >
                        {step.label}
                      </span>
                    </div>
                  );
                })}
              </div>
            )}

            {/* Features */}
            {features.length > 0 && (
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-1">
                {features.map((feature, index) => (
                  <div
                    key={feature.title}
                    className="rounded-lg border border-flow-border/40 bg-transparent p-3 hover:border-flow-border/70 transition-colors duration-200"
                    style={{ animationDelay: `${index * 60}ms` }}
                  >
                    <div className="flex items-start gap-3">
                      <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-flow-accent/8 text-flow-accent border border-flow-border/50">
                        {feature.icon}
                      </div>
                      <div className="space-y-1">
                        <div className="text-sm font-semibold text-surface">
                          {feature.title}
                        </div>
                        <p className="text-xs text-flow-text-muted leading-snug">
                          {feature.description}
                        </p>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Preview */}
          <div className="relative">
            <div className="relative rounded-2xl border border-flow-border/50 bg-flow-node/60 p-4 sm:p-6 shadow-[0_8px_24px_rgba(0,0,0,0.22)] overflow-hidden opacity-70">
              <div className="absolute right-3 top-3 text-[10px] uppercase tracking-wide text-flow-text-muted bg-flow-surface/60 px-2 py-0.5 rounded-md border border-flow-border/40">
                Example
              </div>
              {preview}
            </div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="relative overflow-hidden rounded-2xl border border-flow-border/40 bg-transparent">
      <div className="grid gap-8 p-6 sm:p-8 lg:grid-cols-2 lg:items-center">
        {/* Content */}
        <div className="space-y-5">
          <div className="inline-flex items-center gap-3 rounded-xl bg-flow-node/80 px-3 py-2 border border-flow-border/60 shadow-inner shadow-black/10">
            <div className="flex h-11 w-11 items-center justify-center rounded-lg bg-gradient-to-br from-flow-accent/25 to-purple-500/20 text-surface shadow-md shadow-blue-500/20">
              {icon}
            </div>
            <div>
              <div className="text-sm font-semibold text-surface hero-gradient-text leading-tight">
                {title}
              </div>
              <div className="text-xs text-flow-text-muted leading-tight">{subtitle}</div>
            </div>
          </div>

          {/* CTAs */}
          <div className="flex flex-col sm:flex-row sm:items-center gap-3">
            <button
              onClick={primaryCta.onClick}
              className="hero-button-primary btn-press w-full sm:w-auto justify-center shadow-lg shadow-blue-500/20"
            >
              {primaryCta.label}
              <div className="hero-button-glow" />
            </button>
            {secondaryCta && (
              <button
                onClick={secondaryCta.onClick}
                className="hero-button-secondary w-full sm:w-auto justify-center"
              >
                {secondaryCta.label}
              </button>
            )}
          </div>

          {/* Progress path */}
          {progressPath.length > 0 && (
            <div className="flex flex-wrap items-center gap-3 text-xs text-flow-text-muted">
              {progressPath.map((step, idx) => {
                const circleClasses = step.active
                  ? 'bg-flow-accent text-white border-flow-accent/70'
                  : step.completed
                    ? 'bg-emerald-500/20 text-emerald-200 border-emerald-400/60'
                    : 'bg-flow-node text-flow-text-muted border-flow-border';

                const lineClasses = step.completed
                  ? 'bg-emerald-400/70'
                  : step.active
                    ? 'bg-flow-accent/60'
                    : 'bg-flow-border';

                return (
                  <React.Fragment key={step.label}>
                    <div className="flex items-center gap-2">
                      <div
                        className={`flex h-7 w-7 items-center justify-center rounded-full border text-[11px] font-semibold ${circleClasses}`}
                      >
                        {step.completed ? '✓' : idx + 1}
                      </div>
                      <span className={`font-medium ${step.active ? 'text-surface' : step.completed ? 'text-emerald-100' : ''}`}>
                        {step.label}
                      </span>
                    </div>
                    {idx < progressPath.length - 1 && (
                      <div className={`h-px w-8 rounded-full ${lineClasses}`} />
                    )}
                  </React.Fragment>
                );
              })}
            </div>
          )}

          {/* Features */}
          {features.length > 0 && (
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              {features.map((feature, index) => (
                <div
                  key={feature.title}
                  className="feature-card rounded-xl border border-flow-border/60 bg-flow-node/70 p-4 hover:border-flow-accent/40 transition-all duration-200"
                  style={{ animationDelay: `${index * 60}ms` }}
                >
                  <div className="flex items-start gap-3">
                    <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-flow-accent/10 text-flow-accent border border-flow-border/60">
                      {feature.icon}
                    </div>
                    <div className="space-y-1">
                      <div className="font-semibold text-surface">{feature.title}</div>
                      <p className="text-sm text-flow-text-muted leading-relaxed">
                        {feature.description}
                      </p>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Preview */}
        <div className="relative">
          <div className="relative rounded-2xl border border-flow-border/60 bg-flow-node/70 p-4 sm:p-6 shadow-[0_10px_30px_rgba(0,0,0,0.25)] overflow-hidden">
            {preview}
          </div>
        </div>
      </div>
    </div>
  );
};

export default TabEmptyState;
