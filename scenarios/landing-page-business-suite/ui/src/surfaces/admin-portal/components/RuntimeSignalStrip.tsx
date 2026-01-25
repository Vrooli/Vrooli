import { useState } from 'react';
import { AlertTriangle, RefreshCw, ChevronDown, ChevronUp, Clock } from 'lucide-react';
import { Button } from '../../../shared/ui/button';
import { ToggleSwitch } from '../../../shared/ui/ToggleSwitch';
import { useLandingVariant } from '../../../app/providers/LandingVariantProvider';
import { useComingSoonToggle } from '../hooks/useComingSoonToggle';
import { RESOLUTION_LABELS, getResolutionLabel } from '../config/variant.constants';

interface RuntimeSignalStripProps {
  /** Display mode: 'full' shows all details, 'compact' shows single-line expandable badge */
  mode?: 'full' | 'compact';
}

/**
 * RuntimeSignalStrip exposes landing runtime status (active variant, fallback, refresh affordance)
 * to make admin workflows observable. Future agents can see landing health without leaving the portal.
 *
 * Two modes:
 * - `full`: Original full-width card (~120px height)
 * - `compact`: Single-line expandable badge (~40px collapsed, expands on click)
 */
export function RuntimeSignalStrip({ mode = 'full' }: RuntimeSignalStripProps) {
  const [expanded, setExpanded] = useState(false);
  const { variant, config, loading, error, resolution, statusNote, lastUpdated, refresh } = useLandingVariant();
  const { comingSoonEnabled, toggling, handleToggle } = useComingSoonToggle();

  const variantLabel = variant ? `${variant.name ?? variant.slug} (${variant.slug})` : 'Variant not resolved yet';
  const resolutionLabel = getResolutionLabel(resolution);

  const fallbackActive = Boolean(config?.fallback);
  const configLabel = fallbackActive ? 'Fallback copy active' : 'Live API config';
  const configClass = fallbackActive ? 'bg-amber-500/20 text-amber-200 border-amber-500/30' : 'bg-emerald-500/20 text-emerald-200 border-emerald-500/30';
  const configDescription = fallbackActive
    ? 'Serving baked config until landing-config API responds.'
    : 'Connected to landing-config API.';

  // Error state - same for both modes
  if (error) {
    return (
      <div className="mb-6 rounded-2xl border border-red-500/30 bg-red-500/10 px-6 py-4" data-testid="runtime-signal-error">
        <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
          <div className="flex items-start gap-3">
            <AlertTriangle className="h-5 w-5 text-red-300" />
            <div>
              <p className="text-red-100 font-semibold">Landing config unavailable</p>
              <p className="text-red-100/80 text-sm">{error}</p>
            </div>
          </div>
          <Button variant="outline" size="sm" onClick={() => void refresh()} disabled={loading} className="gap-2">
            <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
            Retry sync
          </Button>
        </div>
      </div>
    );
  }

  // Compact mode - single-line expandable badge
  if (mode === 'compact') {
    return (
      <div className="mb-6" data-testid="runtime-signal-compact">
        <button
          type="button"
          onClick={() => setExpanded(!expanded)}
          className="flex items-center gap-2 rounded-full border border-white/10 bg-white/5 px-4 py-2 text-xs font-medium hover:bg-white/10 transition-colors"
          data-testid="runtime-signal-toggle"
        >
          {comingSoonEnabled ? (
            <span className="flex items-center gap-1 rounded-full border px-2 py-0.5 bg-purple-500/20 text-purple-200 border-purple-500/30">
              <Clock className="h-3 w-3" />
              Coming soon
            </span>
          ) : (
            <>
              <span className={`rounded-full border px-2 py-0.5 ${configClass}`}>{configLabel}</span>
              <span className="text-slate-200">{variant?.name ?? variant?.slug ?? 'No variant'}</span>
            </>
          )}
          {expanded ? (
            <ChevronUp className="h-3 w-3 text-slate-400" />
          ) : (
            <ChevronDown className="h-3 w-3 text-slate-400" />
          )}
        </button>

        {expanded && (
          <div className="mt-2 rounded-xl border border-white/10 bg-white/5 p-4" data-testid="runtime-signal-expanded">
            {/* Coming soon toggle row */}
            <div className="flex items-center justify-between pb-3 mb-3 border-b border-white/10">
              <div className="flex items-center gap-2">
                <Clock className={`h-4 w-4 ${comingSoonEnabled ? 'text-purple-400' : 'text-slate-400'}`} />
                <span className="text-sm font-medium text-slate-200">Coming soon mode</span>
                {comingSoonEnabled && (
                  <span className="text-xs text-purple-300">(variant hidden from visitors)</span>
                )}
              </div>
              <ToggleSwitch
                checked={comingSoonEnabled}
                onToggle={() => void handleToggle()}
                loading={toggling}
                disabled={toggling}
                aria-label="Toggle coming soon mode"
                checkedClassName="bg-purple-500"
              />
            </div>

            <div className="flex flex-wrap gap-2 text-xs font-medium">
              <span className={`rounded-full border px-3 py-1 ${configClass}`}>{configLabel}</span>
              <span className="rounded-full border border-white/10 px-3 py-1 text-slate-200">
                {variantLabel}
              </span>
              <span className="rounded-full border border-white/10 px-3 py-1 text-slate-300">
                Source: {resolutionLabel}
              </span>
            </div>
            <div className="mt-2 flex items-center justify-between text-xs text-slate-400">
              <span>
                {statusNote ?? configDescription}
                {lastUpdated && (
                  <span className="ml-2 text-slate-300/80">
                    Last sync {new Date(lastUpdated).toLocaleTimeString()}
                  </span>
                )}
              </span>
              <Button
                variant="ghost"
                size="sm"
                onClick={(e) => {
                  e.stopPropagation();
                  void refresh();
                }}
                disabled={loading}
                className="gap-1 h-6 px-2"
                data-testid="runtime-refresh"
              >
                <RefreshCw className={`h-3 w-3 ${loading ? 'animate-spin' : ''}`} />
                {loading ? 'Syncing' : 'Refresh'}
              </Button>
            </div>
          </div>
        )}
      </div>
    );
  }

  // Full mode (default)
  return (
    <div className="mb-6 rounded-2xl border border-white/10 bg-white/5 px-6 py-4" data-testid="runtime-signal-strip">
      {/* Coming soon toggle row */}
      <div className="flex items-center justify-between pb-4 mb-4 border-b border-white/10">
        <div className="flex items-center gap-3">
          <Clock className={`h-5 w-5 ${comingSoonEnabled ? 'text-purple-400' : 'text-slate-400'}`} />
          <div>
            <span className="text-sm font-medium text-slate-200">Coming soon mode</span>
            <p className="text-xs text-slate-400">
              {comingSoonEnabled ? 'Visitors see the coming soon page instead of the landing variant' : 'Landing variant is visible to visitors'}
            </p>
          </div>
        </div>
        <ToggleSwitch
          checked={comingSoonEnabled}
          onToggle={() => void handleToggle()}
          loading={toggling}
          disabled={toggling}
          aria-label="Toggle coming soon mode"
          checkedClassName="bg-purple-500"
        />
      </div>

      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <p className="text-xs uppercase tracking-[0.3em] text-slate-400">Landing runtime</p>
          <div className="mt-3 flex flex-wrap gap-2 text-xs font-medium">
            <span className={`rounded-full border px-3 py-1 ${configClass}`}>{configLabel}</span>
            <span className="rounded-full border border-white/10 px-3 py-1 text-slate-200">
              {variantLabel}
            </span>
            <span className="rounded-full border border-white/10 px-3 py-1 text-slate-300">
              Source: {resolutionLabel}
            </span>
          </div>
          <div className="mt-2 text-xs text-slate-400">
            {statusNote ?? configDescription}
            {lastUpdated && (
              <span className="ml-2 text-slate-300/80">
                Last sync {new Date(lastUpdated).toLocaleTimeString()}
              </span>
            )}
          </div>
        </div>
        <Button
          variant="outline"
          onClick={() => void refresh()}
          disabled={loading}
          className="gap-2 self-start md:self-auto"
          data-testid="runtime-refresh"
        >
          <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
          {loading ? 'Syncing...' : 'Refresh config'}
        </Button>
      </div>
    </div>
  );
}
