import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { X, RefreshCw, Check, AlertTriangle, Settings2 } from "lucide-react";
import { Button } from "../components/ui/button";
import { HealthBadge } from "../components/HealthBadge";
import {
  fetchPresets,
  fetchCollectorHealth,
  fetchCircuitBreakers,
  applyPreset,
  resetCircuitBreaker,
  type PresetInfo,
} from "../lib/api";

interface ConfigurationProps {
  onClose: () => void;
}

// [REQ:SCS-UI-002] Configuration UI with toggles and presets
export function Configuration({ onClose }: ConfigurationProps) {
  const queryClient = useQueryClient();

  const { data: presetsData, isLoading: presetsLoading } = useQuery({
    queryKey: ["presets"],
    queryFn: fetchPresets,
  });

  const { data: healthData } = useQuery({
    queryKey: ["collector-health"],
    queryFn: fetchCollectorHealth,
    refetchInterval: 10000,
  });

  const { data: breakersData } = useQuery({
    queryKey: ["circuit-breakers"],
    queryFn: fetchCircuitBreakers,
    refetchInterval: 10000,
  });

  const applyPresetMutation = useMutation({
    mutationFn: (name: string) => applyPreset(name),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["scores"] });
      queryClient.invalidateQueries({ queryKey: ["config"] });
    },
  });

  const resetBreakersMutation = useMutation({
    mutationFn: () => resetCircuitBreaker(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["circuit-breakers"] });
      queryClient.invalidateQueries({ queryKey: ["collector-health"] });
    },
  });

  const trippedBreakers = breakersData?.tripped ?? [];

  return (
    <div className="fixed inset-0 bg-black/80 flex items-center justify-center z-50 p-6" data-testid="configuration-panel">
      <div className="w-full max-w-2xl rounded-2xl border border-white/10 bg-slate-900 shadow-2xl overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-white/10">
          <h2 className="text-xl font-semibold flex items-center gap-2">
            <Settings2 className="h-5 w-5 text-slate-400" />
            Scoring Configuration
          </h2>
          <button
            onClick={onClose}
            className="p-2 rounded-lg hover:bg-white/10 transition-colors"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="p-6 max-h-[70vh] overflow-y-auto space-y-6">
          {/* Presets Section */}
          <section>
            <h3 className="text-sm font-medium text-slate-400 uppercase tracking-wider mb-3">
              Presets
            </h3>
            <div className="grid grid-cols-2 gap-3" data-testid="presets-list">
              {presetsLoading && (
                <div className="col-span-2 py-4 text-center text-slate-400">
                  <RefreshCw className="h-5 w-5 animate-spin mx-auto" />
                </div>
              )}
              {presetsData?.presets?.map((preset: PresetInfo) => (
                <button
                  key={preset.name}
                  onClick={() => applyPresetMutation.mutate(preset.name)}
                  disabled={applyPresetMutation.isPending}
                  className="flex flex-col items-start p-4 rounded-xl border border-white/10 bg-white/5 hover:bg-white/10 transition-colors text-left disabled:opacity-50"
                  data-testid={`preset-${preset.name}`}
                >
                  <span className="font-medium text-slate-200 capitalize">
                    {preset.name.replace(/-/g, " ")}
                  </span>
                  <span className="text-xs text-slate-400 mt-1">
                    {preset.description}
                  </span>
                </button>
              ))}
            </div>
            {applyPresetMutation.isSuccess && (
              <p className="mt-3 text-sm text-emerald-400 flex items-center gap-1">
                <Check className="h-4 w-4" />
                Preset applied successfully
              </p>
            )}
          </section>

          {/* Collector Health Section */}
          <section>
            <h3 className="text-sm font-medium text-slate-400 uppercase tracking-wider mb-3">
              Collector Health
            </h3>
            <div className="rounded-xl border border-white/10 bg-white/5 divide-y divide-white/10">
              {healthData &&
                Object.entries(healthData.collectors).map(([name, collector]) => (
                  <div
                    key={name}
                    className="flex items-center justify-between p-4"
                  >
                    <span className="text-slate-200 capitalize">{name}</span>
                    <HealthBadge status={collector.status} />
                  </div>
                ))}
              {!healthData && (
                <div className="p-4 text-center text-slate-400">
                  Loading health data...
                </div>
              )}
            </div>
          </section>

          {/* Circuit Breaker Section */}
          <section>
            <div className="flex items-center justify-between mb-3">
              <h3 className="text-sm font-medium text-slate-400 uppercase tracking-wider">
                Circuit Breakers
              </h3>
              {trippedBreakers.length > 0 && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => resetBreakersMutation.mutate()}
                  disabled={resetBreakersMutation.isPending}
                >
                  <RefreshCw
                    className={`h-4 w-4 mr-2 ${
                      resetBreakersMutation.isPending ? "animate-spin" : ""
                    }`}
                  />
                  Reset All
                </Button>
              )}
            </div>

            {trippedBreakers.length > 0 && (
              <div
                className="p-4 rounded-xl border border-amber-500/30 bg-amber-500/10 mb-3"
                data-testid="tripped-breakers-alert"
              >
                <div className="flex items-start gap-3">
                  <AlertTriangle className="h-5 w-5 text-amber-400 mt-0.5" />
                  <div>
                    <p className="font-medium text-amber-200">
                      {trippedBreakers.length} breaker
                      {trippedBreakers.length > 1 ? "s" : ""} tripped
                    </p>
                    <p className="text-sm text-amber-300/70 mt-1">
                      These collectors are temporarily disabled due to repeated
                      failures.
                    </p>
                    <div className="flex flex-wrap gap-2 mt-2">
                      {trippedBreakers.map((name: string) => (
                        <span
                          key={name}
                          className="px-2 py-1 rounded bg-amber-500/20 text-amber-200 text-xs"
                        >
                          {name}
                        </span>
                      ))}
                    </div>
                  </div>
                </div>
              </div>
            )}

            <div className="rounded-xl border border-white/10 bg-white/5 p-4">
              <div className="grid grid-cols-4 gap-4 text-center">
                <div>
                  <p className="text-2xl font-bold text-slate-200">
                    {breakersData?.stats?.total ?? 0}
                  </p>
                  <p className="text-xs text-slate-400">Total</p>
                </div>
                <div>
                  <p className="text-2xl font-bold text-emerald-400">
                    {breakersData?.stats?.closed ?? 0}
                  </p>
                  <p className="text-xs text-slate-400">Closed</p>
                </div>
                <div>
                  <p className="text-2xl font-bold text-red-400">
                    {breakersData?.stats?.open ?? 0}
                  </p>
                  <p className="text-xs text-slate-400">Open</p>
                </div>
                <div>
                  <p className="text-2xl font-bold text-amber-400">
                    {breakersData?.stats?.half_open ?? 0}
                  </p>
                  <p className="text-xs text-slate-400">Half-Open</p>
                </div>
              </div>
            </div>
          </section>
        </div>

        {/* Footer */}
        <div className="p-6 border-t border-white/10 bg-white/5">
          <Button onClick={onClose} className="w-full">
            Close
          </Button>
        </div>
      </div>
    </div>
  );
}
