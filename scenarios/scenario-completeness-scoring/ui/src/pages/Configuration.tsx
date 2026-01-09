import { useEffect, useMemo, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { X, RefreshCw, Check, AlertTriangle, Settings2, Info, RotateCcw, Save } from "lucide-react";
import { Button } from "../components/ui/button";
import { HealthBadge } from "../components/HealthBadge";
import {
  fetchCollectorHealth,
  fetchCircuitBreakers,
  resetCircuitBreaker,
  fetchConfig,
  updateConfig,
  resetConfig,
  fetchConfigSchema,
  type ScoringConfig,
  type ConfigSchema,
} from "../lib/api";

interface ConfigurationProps {
  onClose: () => void;
  /** If provided, shows context about which scenario triggered the config panel */
  scenarioContext?: string;
}

function clone<T>(value: T): T {
  // eslint-disable-next-line @typescript-eslint/no-unsafe-return
  return typeof structuredClone === "function" ? structuredClone(value) : (JSON.parse(JSON.stringify(value)) as T);
}

function getAtPath(obj: unknown, path: string): unknown {
  const parts = path.split(".");
  let cur: any = obj;
  for (const part of parts) {
    if (cur == null) return undefined;
    cur = cur[part];
  }
  return cur;
}

function setAtPath<T extends object>(obj: T, path: string, value: unknown): T {
  const next = clone(obj);
  const parts = path.split(".");
  let cur: any = next;
  for (let i = 0; i < parts.length - 1; i++) {
    const key = parts[i];
    if (cur[key] == null) cur[key] = {};
    cur = cur[key];
  }
  cur[parts[parts.length - 1]] = value;
  return next;
}

function validateConfigDraft(cfg: ScoringConfig | null): { ok: boolean; errors: string[]; weightSum: number } {
  if (!cfg) return { ok: false, errors: ["Loading configuration..."], weightSum: 0 };

  const errors: string[] = [];
  const weightSum = (cfg.weights?.quality ?? 0) + (cfg.weights?.coverage ?? 0) + (cfg.weights?.quantity ?? 0) + (cfg.weights?.ui ?? 0);
  if (weightSum !== 100) errors.push(`Weights must sum to 100 (currently ${weightSum}).`);

  const hasDimension =
    cfg.components.quality.enabled || cfg.components.coverage.enabled || cfg.components.quantity.enabled || cfg.components.ui.enabled;
  if (!hasDimension) errors.push("Enable at least one scoring dimension.");

  if (cfg.components.quality.enabled) {
    const any = cfg.components.quality.requirement_pass_rate || cfg.components.quality.target_pass_rate || cfg.components.quality.test_pass_rate;
    if (!any) errors.push("Quality is enabled but no quality sub-metrics are enabled.");
  }
  if (cfg.components.coverage.enabled) {
    const any = cfg.components.coverage.test_coverage_ratio || cfg.components.coverage.requirement_depth;
    if (!any) errors.push("Coverage is enabled but no coverage sub-metrics are enabled.");
  }
  if (cfg.components.quantity.enabled) {
    const any = cfg.components.quantity.requirements || cfg.components.quantity.targets || cfg.components.quantity.tests;
    if (!any) errors.push("Quantity is enabled but no quantity sub-metrics are enabled.");
  }
  if (cfg.components.ui.enabled) {
    const any =
      cfg.components.ui.template_detection ||
      cfg.components.ui.component_complexity ||
      cfg.components.ui.api_integration ||
      cfg.components.ui.routing ||
      cfg.components.ui.code_volume;
    if (!any) errors.push("UI is enabled but no UI sub-metrics are enabled.");
  }
  if (cfg.penalties.enabled) {
    const any =
      cfg.penalties.insufficient_test_coverage ||
      cfg.penalties.invalid_test_location ||
      cfg.penalties.monolithic_test_files ||
      cfg.penalties.single_layer_validation ||
      cfg.penalties.target_mapping_ratio ||
      cfg.penalties.superficial_test_implementation ||
      cfg.penalties.manual_validations;
    if (!any) errors.push("Penalties are enabled but no penalty types are enabled.");
  }

  return { ok: errors.length === 0, errors, weightSum };
}

function Section({
  schema,
  cfg,
  onChange,
}: {
  schema: ConfigSchema["sections"][number];
  cfg: ScoringConfig;
  onChange: (path: string, value: unknown) => void;
}) {
  const enabledPath = schema.enabled_path;
  const weightPath = schema.weight_path;

  const enabledValue = enabledPath ? Boolean(getAtPath(cfg, enabledPath)) : undefined;
  const weightValue = weightPath ? Number(getAtPath(cfg, weightPath) ?? 0) : undefined;

  return (
    <section className="rounded-2xl border border-white/10 bg-white/5 overflow-hidden" data-testid={`config-section-${schema.key}`}>
      <div className="p-4 border-b border-white/10 flex items-start justify-between gap-4">
        <div>
          <h3 className="font-medium text-slate-200">{schema.title}</h3>
          <p className="text-xs text-slate-400 mt-1">{schema.description}</p>
        </div>
        <div className="flex items-center gap-3 flex-shrink-0">
          {typeof weightValue === "number" && (
            <div className="flex items-center gap-2">
              <span className="text-xs text-slate-400">Weight</span>
              <input
                type="number"
                min={0}
                max={100}
                value={Number.isFinite(weightValue) ? weightValue : 0}
                onChange={(e) => onChange(weightPath!, Number(e.target.value))}
                className="w-20 px-2 py-1 rounded-md bg-slate-900 border border-white/10 text-slate-200 text-sm"
              />
            </div>
          )}
          {typeof enabledValue === "boolean" && (
            <label className="flex items-center gap-2 text-sm text-slate-300">
              <input
                type="checkbox"
                checked={enabledValue}
                onChange={(e) => onChange(enabledPath!, e.target.checked)}
                className="accent-emerald-500"
              />
              Enabled
            </label>
          )}
        </div>
      </div>

      <div className="p-4 grid grid-cols-1 md:grid-cols-2 gap-3">
        {schema.fields.map((field) => {
          const value = getAtPath(cfg, field.path);
          const isBoolean = field.type === "boolean";
          const isInteger = field.type === "integer";

          return (
            <label
              key={field.path}
              className="flex items-start gap-3 p-3 rounded-xl border border-white/10 bg-slate-900/40"
              data-testid={`config-field-${field.path}`}
            >
              {isBoolean && (
                <input
                  type="checkbox"
                  checked={Boolean(value)}
                  onChange={(e) => onChange(field.path, e.target.checked)}
                  className="mt-1 accent-emerald-500"
                />
              )}
              {isInteger && (
                <input
                  type="number"
                  value={Number(value ?? 0)}
                  min={field.min ?? 0}
                  max={field.max ?? 100}
                  onChange={(e) => onChange(field.path, Number(e.target.value))}
                  className="w-24 px-2 py-1 mt-0.5 rounded-md bg-slate-900 border border-white/10 text-slate-200 text-sm"
                />
              )}
              <div>
                <div className="text-sm text-slate-200">{field.title}</div>
                <div className="text-xs text-slate-400 mt-0.5">{field.description}</div>
              </div>
            </label>
          );
        })}
      </div>
    </section>
  );
}

// [REQ:SCS-UI-002] Configuration UI with toggles and explained sections
export function Configuration({ onClose, scenarioContext }: ConfigurationProps) {
  const queryClient = useQueryClient();
  const [draft, setDraft] = useState<ScoringConfig | null>(null);
  const [dirty, setDirty] = useState(false);

  const { data: schemaResp } = useQuery({
    queryKey: ["config-schema"],
    queryFn: fetchConfigSchema,
  });
  const schema: ConfigSchema | undefined = schemaResp?.schema;

  const { data: configResp, isLoading: configLoading } = useQuery({
    queryKey: ["config"],
    queryFn: fetchConfig,
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

  useEffect(() => {
    if (!dirty && configResp?.config) setDraft(configResp.config);
  }, [configResp?.config, dirty]);

  const validation = useMemo(() => validateConfigDraft(draft), [draft]);

  const saveMutation = useMutation({
    mutationFn: (cfg: ScoringConfig) => updateConfig(cfg),
    onSuccess: () => {
      setDirty(false);
      queryClient.invalidateQueries({ queryKey: ["scores"] });
      queryClient.invalidateQueries({ queryKey: ["config"] });
    },
  });

  const resetMutation = useMutation({
    mutationFn: () => resetConfig(),
    onSuccess: () => {
      setDirty(false);
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

  const onChange = (path: string, value: unknown) => {
    if (!draft) return;
    setDirty(true);
    setDraft(setAtPath(draft, path, value));
  };

  const trippedBreakers = breakersData?.tripped ?? [];

  return (
    <div className="fixed inset-0 bg-black/80 flex items-center justify-center z-50 p-6" data-testid="configuration-panel">
      <div className="w-full max-w-3xl rounded-2xl border border-white/10 bg-slate-900 shadow-2xl overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-white/10">
          <h2 className="text-xl font-semibold flex items-center gap-2">
            <Settings2 className="h-5 w-5 text-slate-400" />
            Scoring Configuration
          </h2>
          <button onClick={onClose} className="p-2 rounded-lg hover:bg-white/10 transition-colors">
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="p-6 max-h-[70vh] overflow-y-auto space-y-6">
          {/* Scenario Context Indicator */}
          {scenarioContext && (
            <div
              className="flex items-center gap-2 px-3 py-2 rounded-lg bg-blue-500/10 border border-blue-500/20 text-sm"
              data-testid="scenario-context-indicator"
            >
              <Info className="h-4 w-4 text-blue-400 flex-shrink-0" />
              <span className="text-blue-300/80">
                Viewing from: <span className="font-medium text-blue-200">{scenarioContext}</span>
              </span>
              <span className="text-blue-300/60 text-xs">— Changes apply globally</span>
            </div>
          )}

          {/* Config Editor */}
          <section>
            <div className="flex items-center justify-between gap-4 mb-3">
              <div>
                <h3 className="text-sm font-medium text-slate-400 uppercase tracking-wider">Configuration</h3>
                <p className="text-xs text-slate-500 mt-1">
                  Toggle exactly what you care about, and set weights to match what “complete” means for you.
                </p>
              </div>
              <div className="flex items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => resetMutation.mutate()}
                  disabled={resetMutation.isPending}
                  data-testid="config-reset"
                >
                  <RotateCcw className="h-4 w-4 mr-2" />
                  Reset
                </Button>
                <Button
                  size="sm"
                  onClick={() => draft && saveMutation.mutate(draft)}
                  disabled={!draft || !validation.ok || saveMutation.isPending}
                  data-testid="config-save"
                >
                  <Save className="h-4 w-4 mr-2" />
                  Save
                </Button>
              </div>
            </div>

            {(configLoading || !draft) && (
              <div className="py-8 text-center text-slate-400">
                <RefreshCw className="h-5 w-5 animate-spin mx-auto" />
              </div>
            )}

            {!!draft && !!schema && (
              <div className="space-y-4" data-testid="config-editor">
                {!validation.ok && (
                  <div className="p-4 rounded-xl border border-amber-500/30 bg-amber-500/10" data-testid="config-validation-errors">
                    <div className="flex items-start gap-3">
                      <AlertTriangle className="h-5 w-5 text-amber-400 mt-0.5" />
                      <div>
                        <p className="font-medium text-amber-200">Fix configuration issues</p>
                        <ul className="mt-2 text-sm text-amber-300/80 list-disc pl-5 space-y-1">
                          {validation.errors.map((e) => (
                            <li key={e}>{e}</li>
                          ))}
                        </ul>
                      </div>
                    </div>
                  </div>
                )}

                <div className="text-xs text-slate-500">
                  Weights sum: <span className="text-slate-300 font-medium">{validation.weightSum}</span>
                  {dirty && <span className="ml-2 text-slate-400">(unsaved)</span>}
                </div>

                {schema.sections.map((section) => (
                  <Section key={section.key} schema={section} cfg={draft} onChange={onChange} />
                ))}
              </div>
            )}

            {saveMutation.isSuccess && (
              <p className="mt-3 text-sm text-emerald-400 flex items-center gap-1">
                <Check className="h-4 w-4" />
                Configuration saved
              </p>
            )}
          </section>

          {/* Collector Health Section */}
          <section>
            <h3 className="text-sm font-medium text-slate-400 uppercase tracking-wider mb-3">Collector Health</h3>
            <div className="rounded-xl border border-white/10 bg-white/5 divide-y divide-white/10">
              {healthData &&
                Object.entries(healthData.collectors).map(([name, collector]) => (
                  <div key={name} className="flex items-center justify-between p-4">
                    <span className="text-slate-200 capitalize">{name}</span>
                    <HealthBadge status={collector.status} />
                  </div>
                ))}
              {!healthData && <div className="p-4 text-center text-slate-400">Loading health data...</div>}
            </div>
          </section>

          {/* Circuit Breaker Section */}
          <section>
            <div className="flex items-center justify-between mb-3">
              <h3 className="text-sm font-medium text-slate-400 uppercase tracking-wider">Circuit Breakers</h3>
              {trippedBreakers.length > 0 && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => resetBreakersMutation.mutate()}
                  disabled={resetBreakersMutation.isPending}
                >
                  <RefreshCw className={`h-4 w-4 mr-2 ${resetBreakersMutation.isPending ? "animate-spin" : ""}`} />
                  Reset All
                </Button>
              )}
            </div>

            {trippedBreakers.length > 0 && (
              <div className="p-4 rounded-xl border border-amber-500/30 bg-amber-500/10 mb-3" data-testid="tripped-breakers-alert">
                <div className="flex items-start gap-3">
                  <AlertTriangle className="h-5 w-5 text-amber-400 mt-0.5" />
                  <div>
                    <p className="font-medium text-amber-200">
                      {trippedBreakers.length} breaker{trippedBreakers.length > 1 ? "s" : ""} tripped
                    </p>
                    <p className="text-sm text-amber-300/70 mt-1">These collectors are temporarily disabled due to repeated failures.</p>
                    <div className="flex flex-wrap gap-2 mt-2">
                      {trippedBreakers.map((name: string) => (
                        <span key={name} className="px-2 py-1 rounded bg-amber-500/20 text-amber-200 text-xs">
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
                  <p className="text-2xl font-bold text-slate-200">{breakersData?.stats?.total ?? 0}</p>
                  <p className="text-xs text-slate-400">Total</p>
                </div>
                <div>
                  <p className="text-2xl font-bold text-emerald-400">{breakersData?.stats?.closed ?? 0}</p>
                  <p className="text-xs text-slate-400">Closed</p>
                </div>
                <div>
                  <p className="text-2xl font-bold text-red-400">{breakersData?.stats?.open ?? 0}</p>
                  <p className="text-xs text-slate-400">Open</p>
                </div>
                <div>
                  <p className="text-2xl font-bold text-amber-400">{breakersData?.stats?.half_open ?? 0}</p>
                  <p className="text-xs text-slate-400">Half-open</p>
                </div>
              </div>
            </div>
          </section>
        </div>
      </div>
    </div>
  );
}

