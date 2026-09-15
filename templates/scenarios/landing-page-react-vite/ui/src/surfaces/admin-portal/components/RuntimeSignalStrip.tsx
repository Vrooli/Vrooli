import { AlertTriangle, RefreshCw } from 'lucide-react';
import { Button } from '../../../shared/ui/button';
import { useLandingVariant, type VariantResolution } from '../../../app/providers/LandingVariantProvider';

const resolutionLabels: Record<VariantResolution, string> = {
  url_param: 'URL parameter',
  local_storage: 'Stored visitor assignment',
  api_select: 'Weighted API selection',
  fallback: 'Fallback payload',
  unknown: 'Unknown strategy',
};

/**
 * RuntimeSignalStrip exposes landing runtime status (active variant, fallback, refresh affordance)
 * to make admin workflows observable. Future agents can see landing health without leaving the portal.
 */
export function RuntimeSignalStrip() {
  const { variant, config, loading, error, resolution, statusNote, lastUpdated, refresh } = useLandingVariant();

  const variantLabel = variant ? `${variant.name ?? variant.slug} (${variant.slug})` : 'Variant not resolved yet';
  const resolutionLabel = resolutionLabels[resolution] ?? resolutionLabels.unknown;

  const fallbackActive = Boolean(config?.fallback);
  const configLabel = fallbackActive ? 'Fallback copy active' : 'Live API config';
  const configClass = fallbackActive ? 'bg-amber-500/20 text-amber-200 border-amber-500/30' : 'bg-emerald-500/20 text-emerald-200 border-emerald-500/30';
  const configDescription = fallbackActive
    ? 'Serving baked config until landing-config API responds.'
    : 'Connected to landing-config API.';

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

  return (
    <div className="mb-6 rounded-2xl border border-white/10 bg-white/5 px-6 py-4" data-testid="runtime-signal-strip">
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
