import { useQuery } from "@tanstack/react-query";
import { ArrowRight, TestTube } from "lucide-react";
import { fetchHealth } from "../lib/api";
import { useVariant } from "../contexts/VariantContext";
import { useMetrics } from "../hooks/useMetrics";
import { Button } from "../components/ui/button";

export default function PublicHome() {
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["health"],
    queryFn: fetchHealth
  });

  const { variant, loading: variantLoading, error: variantError } = useVariant();
  const { trackCTAClick } = useMetrics();

  return (
    <div className="min-h-screen bg-slate-950 text-slate-50 flex flex-col items-center justify-center p-6">
      <div className="w-full max-w-xl rounded-2xl border border-white/10 bg-white/5 p-8 shadow-2xl backdrop-blur">
        <p className="text-sm uppercase tracking-[0.2em] text-slate-400">Scenario Template</p>
        <h1 className="mt-3 text-3xl font-semibold">Landing Manager</h1>
        <p className="mt-2 text-slate-300">
          This starter UI is intentionally minimal. Replace it with your scenario-specific
          experience while keeping the styling conventions (Tailwind + shadcn) and API wiring in place.
        </p>

        {/* A/B Testing Variant Display */}
        <div className="mt-6 rounded-xl border border-white/10 bg-black/20 p-4">
          <p className="text-sm font-medium text-slate-400 flex items-center gap-2">
            <TestTube className="h-4 w-4" />
            A/B Test Variant
          </p>
          {variantLoading && <p className="mt-2 text-slate-200">Loading variant…</p>}
          {variantError && (
            <p className="mt-2 text-red-400">
              Failed to load variant: {variantError}
            </p>
          )}
          {variant && (
            <div className="mt-2 text-sm text-slate-200">
              <p><span className="text-slate-400">Name:</span> {variant.name}</p>
              <p><span className="text-slate-400">Slug:</span> {variant.slug}</p>
              <p><span className="text-slate-400">Weight:</span> {variant.weight}%</p>
              <p><span className="text-slate-400">Status:</span> {variant.status}</p>
            </div>
          )}
        </div>

        <div className="mt-6 rounded-xl border border-white/10 bg-black/20 p-4">
          <p className="text-sm font-medium text-slate-400">API Health</p>
          {isLoading && <p className="mt-2 text-slate-200">Checking API status…</p>}
          {error && (
            <p className="mt-2 text-red-400">
              Unable to reach the API. Make sure the scenario is running through `vrooli scenario run` so API_PORT/VITE_API_BASE_URL are available.
            </p>
          )}
          {data && (
            <div className="mt-2 text-sm text-slate-200">
              <p>Status: {data.status}</p>
              <p>Service: {data.service}</p>
              <p>Timestamp: {new Date(data.timestamp).toLocaleString()}</p>
            </div>
          )}
          <Button
            className="mt-4"
            onClick={() => {
              trackCTAClick('refresh-health-cta', { action: 'refresh_api_health' });
              refetch();
            }}
            data-testid="refresh-health-button"
          >
            Refresh
            <ArrowRight className="ml-2 h-4 w-4" />
          </Button>
        </div>
      </div>
    </div>
  );
}
