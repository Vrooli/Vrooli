import type { ProbeResponse, ProxyHintsResponse } from "../lib/api";
import { Button } from "./ui/button";
import { Checkbox } from "./ui/checkbox";
import { Input } from "./ui/input";
import { Label } from "./ui/label";

interface ConnectionTester {
  isPending: boolean;
  mutate: () => void;
}

interface ExternalServerSectionProps {
  proxyUrl: string;
  onProxyUrlChange: (value: string) => void;
  scenarioName: string;
  proxyHints?: ProxyHintsResponse | null;
  connectionTester: ConnectionTester;
  connectionResult: ProbeResponse | null;
  connectionError: string | null;
  autoManageTier1: boolean;
  onAutoManageTier1Change: (value: boolean) => void;
  vrooliBinaryPath: string;
  onVrooliBinaryPathChange: (value: string) => void;
}

export function ExternalServerSection({
  proxyUrl,
  onProxyUrlChange,
  scenarioName,
  proxyHints,
  connectionTester,
  connectionResult,
  connectionError,
  autoManageTier1,
  onAutoManageTier1Change,
  vrooliBinaryPath,
  onVrooliBinaryPathChange
}: ExternalServerSectionProps) {
  return (
    <div className="rounded-lg border border-slate-700 bg-slate-900/40 p-4 space-y-3">
      <div>
        <Label htmlFor="proxyUrl">Proxy URL</Label>
        <p className="text-xs text-slate-400 mb-2">
          Paste the exact URL you open in your browser (for example <code>https://app-monitor.yourdomain.com/apps/{scenarioName || "scenario"}/proxy/</code>). This keeps all traffic inside the secure tunnel.
        </p>
      </div>
      <Input
        id="proxyUrl"
        value={proxyUrl}
        onChange={(e) => onProxyUrlChange(e.target.value)}
        placeholder="https://app-monitor.example.dev/apps/picker-wheel/proxy/"
      />
      <p className="text-xs text-slate-400 space-x-1">
        <span>Desktop apps simply load this URL. Use the Cloudflare/app-monitor address if you want remote access.</span>
      </p>

      {proxyHints?.hints && proxyHints.hints.length > 0 && (
        <div className="rounded border border-slate-800 bg-black/20 p-3 space-y-2">
          <p className="text-xs uppercase tracking-wide text-slate-400">Detected URLs</p>
          <div className="space-y-2">
            {proxyHints.hints.map((hint) => (
              <button
                key={hint.url}
                type="button"
                onClick={() => onProxyUrlChange(hint.url)}
                className="w-full rounded border border-slate-700 bg-slate-950/30 px-3 py-2 text-left text-sm hover:border-blue-500"
              >
                <div className="font-medium text-slate-200">{hint.url}</div>
                <div className="text-xs text-slate-400">
                  {hint.message} · Source: {hint.source}
                </div>
              </button>
            ))}
          </div>
        </div>
      )}

      <div className="flex items-center gap-2">
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => connectionTester.mutate()}
          disabled={connectionTester.isPending || !proxyUrl}
        >
          {connectionTester.isPending ? "Testing..." : "Test connection"}
        </Button>
        {connectionResult && connectionResult.server.status === "ok" && connectionResult.api.status === "ok" && (
          <span className="text-xs text-green-300">Both URLs responded ✔</span>
        )}
        {connectionError && <span className="text-xs text-red-300">{connectionError}</span>}
      </div>
      {(connectionResult?.server || connectionResult?.api) && (
        <div className="rounded border border-slate-800 bg-black/20 p-3 text-xs text-slate-300 space-y-1">
          <p className="font-semibold text-slate-200">Connectivity snapshot</p>
          <p>
            UI URL: {connectionResult?.server.status === "ok" ? "reachable" : connectionResult?.server.message || "no response"}
          </p>
          <p>
            API URL: {connectionResult?.api.status === "ok" ? "reachable" : connectionResult?.api.message || "no response"}
          </p>
        </div>
      )}

      <Checkbox
        checked={autoManageTier1}
        onChange={(e) => onAutoManageTier1Change(e.target.checked)}
        label="Automatically run the scenario locally with the vrooli CLI (advanced)"
      />
      <p className="text-xs text-slate-400">
        If enabled, the desktop app will look for the `vrooli` binary, run `vrooli setup`, and start/stop the scenario on the user's machine. Enable only when the end user expects to host the full stack locally.
      </p>

      <Label htmlFor="vrooliBinary" className={autoManageTier1 ? undefined : "text-slate-500"}>
        vrooli CLI path
      </Label>
      <Input
        id="vrooliBinary"
        value={vrooliBinaryPath}
        onChange={(e) => onVrooliBinaryPathChange(e.target.value)}
        disabled={!autoManageTier1}
        placeholder="vrooli"
      />
    </div>
  );
}
