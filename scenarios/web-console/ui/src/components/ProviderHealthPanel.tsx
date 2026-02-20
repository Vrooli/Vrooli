// DOC: docs/concepts/ARCHITECTURE.md#file-map
// [REQ:P1-003b] Provider Health Dashboard
import { useState, useEffect, useCallback } from "react";
import { Activity, Power, PowerOff, RefreshCw, AlertTriangle } from "lucide-react";
import { cn } from "../lib/classnames";
import { Button } from "./ui/button";
import {
  type ProviderHealth,
  type ProviderConfig,
  getAIConfig,
  updateAIConfig,
  toErrorInfo,
} from "../lib/api";

interface ProviderHealthPanelProps {
  open: boolean;
}

export default function ProviderHealthPanel({ open }: ProviderHealthPanelProps) {
  const [providers, setProviders] = useState<ProviderConfig[]>([]);
  const [health, setHealth] = useState<ProviderHealth[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await getAIConfig();
      setProviders(data.providers);
      setHealth(data.health);
    } catch (err) {
      setError(toErrorInfo(err).message);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (!open) return;
    refresh();
  }, [open, refresh]);

  const toggleProvider = useCallback(
    async (name: string, currentEnabled: boolean) => {
      setError(null);
      try {
        const data = await updateAIConfig({ name, enabled: !currentEnabled });
        setProviders(data.providers);
        setHealth(data.health);
      } catch (err) {
        setError(toErrorInfo(err).message);
      }
    },
    [],
  );

  const healthFor = (name: string): ProviderHealth | undefined =>
    health.find((h) => h.name === name);

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between px-1">
        <div className="text-xs font-medium uppercase tracking-wider text-wc-text-faint">
          AI Providers
        </div>
        <Button
          data-testid="provider-refresh"
          variant="ghost"
          size="icon"
          className="h-5 w-5"
          onClick={refresh}
          disabled={loading}
        >
          <RefreshCw className={cn("h-3 w-3", loading && "animate-spin")} />
        </Button>
      </div>

      {error && (
        <div
          data-testid="provider-error"
          className="flex items-center gap-1.5 rounded border border-amber-500/30 bg-amber-500/10 px-2 py-1 text-xs text-amber-300"
        >
          <AlertTriangle className="h-3 w-3 shrink-0" />
          <span className="truncate">{error}</span>
        </div>
      )}

      {providers.map((provider) => {
        const h = healthFor(provider.name);
        const total = (h?.success_count ?? 0) + (h?.error_count ?? 0);
        return (
          <div
            key={provider.name}
            data-testid={`provider-card-${provider.name}`}
            className="rounded-md border border-wc-default bg-wc-surface-input px-3 py-2"
          >
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Activity
                  className={cn("h-3.5 w-3.5", h?.available ? "text-green-400" : "text-wc-text-faint")}
                />
                <span className="text-sm font-medium text-wc-text-primary">
                  {provider.name}
                </span>
              </div>
              <Button
                data-testid={`provider-toggle-${provider.name}`}
                variant="ghost"
                size="icon"
                className="h-6 w-6"
                onClick={() => toggleProvider(provider.name, provider.enabled)}
                title={provider.enabled ? "Disable" : "Enable"}
              >
                {provider.enabled ? (
                  <Power className="h-3.5 w-3.5 text-green-400" />
                ) : (
                  <PowerOff className="h-3.5 w-3.5 text-wc-text-faint" />
                )}
              </Button>
            </div>
            {h && total > 0 && (
              <div className="mt-1 flex gap-3 text-xs text-wc-text-faint">
                <span>{h.success_count} ok</span>
                <span>{h.error_count} err</span>
                {h.last_latency && <span>{h.last_latency}</span>}
              </div>
            )}
            <div className="mt-0.5 text-xs text-wc-text-faint">
              priority {provider.priority} &middot; {provider.timeout_sec}s timeout
            </div>
          </div>
        );
      })}

      {providers.length === 0 && !loading && (
        <div className="text-center text-xs text-wc-text-faint">
          No providers configured
        </div>
      )}
    </div>
  );
}
