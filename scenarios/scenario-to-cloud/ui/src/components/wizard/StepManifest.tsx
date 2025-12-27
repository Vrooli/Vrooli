import { useState, useEffect, useCallback, useMemo } from "react";
import { Code, FormInput, AlertCircle, Check, AlertTriangle, Search } from "lucide-react";
import { Button } from "../ui/button";
import { Textarea, Input } from "../ui/input";
import { Alert } from "../ui/alert";
import { HelpTooltip } from "../ui/tooltip";
import { selectors } from "../../consts/selectors";
import type { useDeployment } from "../../hooks/useDeployment";
import {
  listScenarios,
  checkReachability,
  type ScenarioInfo,
  type ReachabilityResult,
  type PortConfig,
} from "../../lib/api";

type Mode = "form" | "json";

interface StepManifestProps {
  deployment: ReturnType<typeof useDeployment>;
}

// Parse port range to get a reasonable default (first port in range)
function getPortFromConfig(config: PortConfig): number | null {
  if (config.port) return config.port;
  if (config.range) {
    const match = config.range.match(/^(\d+)/);
    if (match) return parseInt(match[1], 10);
  }
  return null;
}

// Convert service.json port key to a human-readable label
function portKeyToLabel(key: string): string {
  // Handle common cases
  const labels: Record<string, string> = {
    ui: "UI Port",
    api: "API Port",
    ws: "WebSocket Port",
    websocket: "WebSocket Port",
  };
  if (labels[key]) return labels[key];

  // Convert snake_case or kebab-case to Title Case
  return key
    .replace(/[-_]/g, " ")
    .replace(/\b\w/g, (c) => c.toUpperCase())
    + " Port";
}

export function StepManifest({ deployment }: StepManifestProps) {
  const { manifestJson, setManifestJson, parsedManifest } = deployment;
  const [mode, setMode] = useState<Mode>("form");

  // Scenarios list state
  const [scenarios, setScenarios] = useState<ScenarioInfo[]>([]);
  const [scenariosLoading, setScenariosLoading] = useState(true);
  const [scenariosError, setScenariosError] = useState<string | null>(null);
  const [scenarioSearch, setScenarioSearch] = useState("");
  const [showScenarioDropdown, setShowScenarioDropdown] = useState(false);

  // Reachability check state
  const [hostReachability, setHostReachability] = useState<ReachabilityResult | null>(null);
  const [domainReachability, setDomainReachability] = useState<ReachabilityResult | null>(null);
  const [isCheckingHost, setIsCheckingHost] = useState(false);
  const [isCheckingDomain, setIsCheckingDomain] = useState(false);

  // Fetch scenarios on mount
  useEffect(() => {
    let cancelled = false;
    async function fetchScenarios() {
      try {
        setScenariosLoading(true);
        const res = await listScenarios();
        if (!cancelled) {
          setScenarios(res.scenarios);
          setScenariosError(null);
        }
      } catch (e) {
        if (!cancelled) {
          setScenariosError(e instanceof Error ? e.message : "Failed to load scenarios");
        }
      } finally {
        if (!cancelled) {
          setScenariosLoading(false);
        }
      }
    }
    fetchScenarios();
    return () => { cancelled = true; };
  }, []);

  // Form state derived from parsed manifest
  const formValues = parsedManifest.ok
    ? {
        host: parsedManifest.value.target?.vps?.host ?? "",
        scenarioId: parsedManifest.value.scenario?.id ?? "",
        domain: parsedManifest.value.edge?.domain ?? "",
        ports: parsedManifest.value.ports ?? {},
        includePackages: parsedManifest.value.bundle?.include_packages ?? true,
        includeAutoheal: parsedManifest.value.bundle?.include_autoheal ?? true,
        caddyEnabled: parsedManifest.value.edge?.caddy?.enabled ?? true,
      }
    : null;

  // Scenario validation
  const selectedScenario = useMemo(() => {
    if (!formValues?.scenarioId) return null;
    return scenarios.find((s) => s.id === formValues.scenarioId) ?? null;
  }, [scenarios, formValues?.scenarioId]);

  const scenarioIdError = useMemo(() => {
    if (!formValues?.scenarioId) return null;
    if (scenariosLoading) return null;
    if (scenarios.length === 0) return null; // Can't validate without list
    if (!selectedScenario) {
      return `Scenario "${formValues.scenarioId}" not found. Choose from available scenarios.`;
    }
    return null;
  }, [formValues?.scenarioId, scenarios, selectedScenario, scenariosLoading]);

  // Filter scenarios for dropdown
  const filteredScenarios = useMemo(() => {
    if (!scenarioSearch) return scenarios;
    const search = scenarioSearch.toLowerCase();
    return scenarios.filter(
      (s) =>
        s.id.toLowerCase().includes(search) ||
        s.displayName?.toLowerCase().includes(search) ||
        s.description?.toLowerCase().includes(search)
    );
  }, [scenarios, scenarioSearch]);

  // Check host reachability with debounce
  const checkHostReachabilityDebounced = useCallback(async (host: string) => {
    if (!host || host === "203.0.113.10") {
      setHostReachability(null);
      return;
    }
    setIsCheckingHost(true);
    try {
      const res = await checkReachability(host, undefined);
      const result = res.results.find((r) => r.type === "host");
      setHostReachability(result ?? null);
    } catch {
      setHostReachability(null);
    } finally {
      setIsCheckingHost(false);
    }
  }, []);

  // Check domain reachability with debounce
  const checkDomainReachabilityDebounced = useCallback(async (domain: string) => {
    if (!domain || domain === "example.com") {
      setDomainReachability(null);
      return;
    }
    setIsCheckingDomain(true);
    try {
      const res = await checkReachability(undefined, domain);
      const result = res.results.find((r) => r.type === "domain");
      setDomainReachability(result ?? null);
    } catch {
      setDomainReachability(null);
    } finally {
      setIsCheckingDomain(false);
    }
  }, []);

  // Debounced reachability checks
  useEffect(() => {
    if (!formValues?.host) return;
    const timer = setTimeout(() => {
      checkHostReachabilityDebounced(formValues.host);
    }, 1000);
    return () => clearTimeout(timer);
  }, [formValues?.host, checkHostReachabilityDebounced]);

  useEffect(() => {
    if (!formValues?.domain) return;
    const timer = setTimeout(() => {
      checkDomainReachabilityDebounced(formValues.domain);
    }, 1000);
    return () => clearTimeout(timer);
  }, [formValues?.domain, checkDomainReachabilityDebounced]);

  const updateFormField = (field: string, value: string | number | boolean) => {
    if (!parsedManifest.ok) return;

    const manifest = { ...parsedManifest.value };

    switch (field) {
      case "host":
        manifest.target = { ...manifest.target, vps: { ...manifest.target.vps, host: value as string } };
        break;
      case "scenarioId": {
        const scenarioId = value as string;
        manifest.scenario = { ...manifest.scenario, id: scenarioId };
        manifest.dependencies = { ...manifest.dependencies, scenarios: [scenarioId] };

        // Auto-populate ALL ports from the selected scenario's service.json
        const scenario = scenarios.find((s) => s.id === scenarioId);
        if (scenario?.ports) {
          const ports: Record<string, number> = {};
          for (const [key, config] of Object.entries(scenario.ports)) {
            const port = getPortFromConfig(config);
            if (port) {
              // Normalize websocket -> ws for consistency
              const normalizedKey = key === "websocket" ? "ws" : key;
              ports[normalizedKey] = port;
            }
          }
          manifest.ports = ports;
        } else {
          // Clear ports if scenario has none defined
          manifest.ports = {};
        }
        break;
      }
      case "domain":
        manifest.edge = { ...manifest.edge, domain: value as string };
        break;
      case "includePackages":
        manifest.bundle = { ...manifest.bundle, include_packages: value as boolean };
        break;
      case "includeAutoheal":
        manifest.bundle = { ...manifest.bundle, include_autoheal: value as boolean };
        break;
      case "caddyEnabled":
        manifest.edge = { ...manifest.edge, caddy: { ...manifest.edge.caddy, enabled: value as boolean } };
        break;
      default:
        // Handle dynamic port fields (e.g., "port:ui", "port:playwright_driver")
        if (field.startsWith("port:")) {
          const portKey = field.slice(5);
          manifest.ports = { ...manifest.ports, [portKey]: value as number };
        }
        break;
    }

    setManifestJson(JSON.stringify(manifest, null, 2));
  };

  const selectScenario = (scenarioId: string) => {
    updateFormField("scenarioId", scenarioId);
    setScenarioSearch("");
    setShowScenarioDropdown(false);
  };

  // Get warning/status message for host
  const hostWarning = useMemo(() => {
    if (!hostReachability) return undefined;
    if (hostReachability.reachable) return undefined;
    return hostReachability.hint ?? hostReachability.message;
  }, [hostReachability]);

  // Get warning/status message for domain
  const domainWarning = useMemo(() => {
    if (!domainReachability) return undefined;
    if (domainReachability.reachable) return undefined;
    return domainReachability.hint ?? domainReachability.message;
  }, [domainReachability]);

  // Get success hint for host/domain
  const hostHint = useMemo(() => {
    if (isCheckingHost) return "Checking reachability...";
    if (hostReachability?.reachable) return `✓ ${hostReachability.message}`;
    return "The IP address or hostname of your VPS";
  }, [hostReachability, isCheckingHost]);

  const domainHint = useMemo(() => {
    if (isCheckingDomain) return "Checking DNS...";
    if (domainReachability?.reachable) return `✓ ${domainReachability.message}`;
    return "Domain name for HTTPS (requires DNS configured)";
  }, [domainReachability, isCheckingDomain]);

  return (
    <div className="space-y-6">
      {/* Mode Toggle */}
      <div className="flex items-center gap-2">
        <Button
          variant={mode === "form" ? "default" : "outline"}
          size="sm"
          onClick={() => setMode("form")}
        >
          <FormInput className="h-4 w-4 mr-1.5" />
          Form
        </Button>
        <Button
          variant={mode === "json" ? "default" : "outline"}
          size="sm"
          onClick={() => setMode("json")}
        >
          <Code className="h-4 w-4 mr-1.5" />
          JSON
        </Button>
      </div>

      {/* JSON Parse Error */}
      {!parsedManifest.ok && (
        <Alert variant="error" title="Invalid JSON">
          {parsedManifest.error}
        </Alert>
      )}

      {/* Form Mode */}
      {mode === "form" && formValues && (
        <div className="grid gap-6 md:grid-cols-2">
          {/* Scenario Section - Moved to top for logical flow */}
          <div className="space-y-4 md:col-span-2">
            <h3 className="text-sm font-medium text-slate-300 flex items-center gap-2">
              Scenario
              <HelpTooltip content="The scenario to deploy. Must exist in your Vrooli installation." />
            </h3>
            <div className="relative">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-500" />
                <input
                  type="text"
                  placeholder="Search scenarios..."
                  value={showScenarioDropdown ? scenarioSearch : formValues.scenarioId}
                  onChange={(e) => {
                    setScenarioSearch(e.target.value);
                    if (!showScenarioDropdown) {
                      setShowScenarioDropdown(true);
                    }
                  }}
                  onFocus={() => {
                    setShowScenarioDropdown(true);
                    setScenarioSearch("");
                  }}
                  onBlur={() => {
                    // Delay to allow click on dropdown item
                    setTimeout(() => setShowScenarioDropdown(false), 200);
                  }}
                  className={`w-full rounded-lg border bg-black/30 pl-9 pr-3 py-2 text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-slate-500 ${
                    scenarioIdError ? "border-red-500/50" : "border-white/10"
                  }`}
                />
                {scenariosLoading && (
                  <div className="absolute right-3 top-1/2 -translate-y-1/2">
                    <div className="h-4 w-4 animate-spin rounded-full border-2 border-slate-500 border-t-transparent" />
                  </div>
                )}
              </div>

              {/* Scenario Dropdown */}
              {showScenarioDropdown && !scenariosLoading && (
                <div className="absolute z-10 mt-1 w-full max-h-60 overflow-auto rounded-lg border border-white/10 bg-slate-900 shadow-lg">
                  {scenariosError ? (
                    <div className="px-3 py-2 text-sm text-red-400">{scenariosError}</div>
                  ) : filteredScenarios.length === 0 ? (
                    <div className="px-3 py-2 text-sm text-slate-500">No scenarios found</div>
                  ) : (
                    filteredScenarios.map((scenario) => (
                      <button
                        key={scenario.id}
                        type="button"
                        onClick={() => selectScenario(scenario.id)}
                        className={`w-full text-left px-3 py-2 hover:bg-slate-800 flex items-center gap-2 ${
                          scenario.id === formValues.scenarioId ? "bg-slate-800" : ""
                        }`}
                      >
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2">
                            <span className="text-sm text-slate-100 font-medium truncate">
                              {scenario.id}
                            </span>
                            {scenario.id === formValues.scenarioId && (
                              <Check className="h-4 w-4 text-green-400 flex-shrink-0" />
                            )}
                          </div>
                          {scenario.displayName && scenario.displayName !== scenario.id && (
                            <div className="text-xs text-slate-400 truncate">{scenario.displayName}</div>
                          )}
                        </div>
                      </button>
                    ))
                  )}
                </div>
              )}

              {/* Selected scenario info */}
              {selectedScenario && !showScenarioDropdown && (
                <p className="mt-1.5 text-xs text-green-400 flex items-center gap-1">
                  <Check className="h-3 w-3" />
                  {selectedScenario.displayName ?? selectedScenario.id}
                  {selectedScenario.ports && Object.keys(selectedScenario.ports).length > 0 && (
                    <span className="text-slate-500 ml-1">
                      · {Object.keys(selectedScenario.ports).length} port(s) defined
                    </span>
                  )}
                </p>
              )}
              {scenarioIdError && (
                <p className="mt-1.5 text-xs text-red-400 flex items-center gap-1">
                  <AlertCircle className="h-3 w-3" />
                  {scenarioIdError}
                </p>
              )}
            </div>
          </div>

          {/* Target Section */}
          <div className="space-y-4 md:col-span-2">
            <h3 className="text-sm font-medium text-slate-300 flex items-center gap-2">
              Target Server
              <HelpTooltip content="The VPS where your scenario will be deployed. Must have SSH access." />
            </h3>
            <div className="grid gap-4 sm:grid-cols-2">
              <Input
                label="Host IP or Hostname"
                placeholder="203.0.113.10"
                value={formValues.host}
                onChange={(e) => updateFormField("host", e.target.value)}
                hint={hostHint}
                warning={hostWarning}
                isLoading={isCheckingHost}
              />
              <Input
                label="Domain"
                placeholder="example.com"
                value={formValues.domain}
                onChange={(e) => updateFormField("domain", e.target.value)}
                hint={domainHint}
                warning={domainWarning}
                isLoading={isCheckingDomain}
              />
            </div>
          </div>

          {/* Ports Section */}
          <div className="space-y-4 md:col-span-2">
            <h3 className="text-sm font-medium text-slate-300 flex items-center gap-2">
              Ports
              <HelpTooltip content="Ports for the deployed services, loaded from the scenario's service.json." />
              {selectedScenario?.ports && Object.keys(selectedScenario.ports).length > 0 && (
                <span className="text-xs text-slate-500 font-normal">
                  (from {selectedScenario.id})
                </span>
              )}
            </h3>
            {selectedScenario?.ports && Object.keys(selectedScenario.ports).length > 0 ? (
              <div className="grid gap-4 sm:grid-cols-2 md:grid-cols-3">
                {Object.entries(selectedScenario.ports).map(([key, config]) => {
                  // Normalize websocket -> ws to match manifest
                  const portKey = key === "websocket" ? "ws" : key;
                  const currentValue = formValues.ports[portKey] ?? getPortFromConfig(config) ?? 0;
                  return (
                    <Input
                      key={key}
                      label={portKeyToLabel(key)}
                      type="number"
                      value={currentValue}
                      onChange={(e) => updateFormField(`port:${portKey}`, parseInt(e.target.value, 10) || 0)}
                      hint={config.description}
                    />
                  );
                })}
              </div>
            ) : (
              <div className="text-sm text-slate-500 italic">
                {formValues.scenarioId
                  ? "No ports defined in this scenario's service.json"
                  : "Select a scenario to configure ports"}
              </div>
            )}
          </div>

          {/* Options Section */}
          <div className="space-y-4 md:col-span-2">
            <h3 className="text-sm font-medium text-slate-300">Bundle Options</h3>
            <div className="flex flex-wrap gap-4">
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={formValues.includePackages}
                  onChange={(e) => updateFormField("includePackages", e.target.checked)}
                  className="rounded border-slate-600 bg-slate-800 text-blue-500 focus:ring-blue-500"
                />
                <span className="text-sm text-slate-300">Include packages</span>
                <HelpTooltip content="Include the packages/ directory in the bundle" />
              </label>
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={formValues.includeAutoheal}
                  onChange={(e) => updateFormField("includeAutoheal", e.target.checked)}
                  className="rounded border-slate-600 bg-slate-800 text-blue-500 focus:ring-blue-500"
                />
                <span className="text-sm text-slate-300">Include autoheal</span>
                <HelpTooltip content="Include vrooli-autoheal for automatic recovery" />
              </label>
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={formValues.caddyEnabled}
                  onChange={(e) => updateFormField("caddyEnabled", e.target.checked)}
                  className="rounded border-slate-600 bg-slate-800 text-blue-500 focus:ring-blue-500"
                />
                <span className="text-sm text-slate-300">Enable Caddy</span>
                <HelpTooltip content="Use Caddy for automatic HTTPS with Let's Encrypt" />
              </label>
            </div>
          </div>
        </div>
      )}

      {/* Form mode with parse error */}
      {mode === "form" && !formValues && (
        <Alert variant="warning" title="Cannot edit in form mode">
          <div className="flex items-center gap-2">
            <AlertCircle className="h-4 w-4" />
            Fix the JSON syntax errors to use form editing, or switch to JSON mode.
          </div>
        </Alert>
      )}

      {/* JSON Mode */}
      {mode === "json" && (
        <Textarea
          data-testid={selectors.manifest.input}
          label="Cloud Manifest JSON"
          hint="The complete deployment manifest in JSON format"
          value={manifestJson}
          onChange={(e) => setManifestJson(e.target.value)}
          className="font-mono text-xs h-80"
          spellCheck={false}
          error={!parsedManifest.ok ? parsedManifest.error : undefined}
        />
      )}
    </div>
  );
}
