import { useState } from "react";
import { Code, FormInput, AlertCircle } from "lucide-react";
import { Button } from "../ui/button";
import { Textarea, Input } from "../ui/input";
import { Alert } from "../ui/alert";
import { HelpTooltip } from "../ui/tooltip";
import { selectors } from "../../consts/selectors";
import type { useDeployment } from "../../hooks/useDeployment";

type Mode = "form" | "json";

interface StepManifestProps {
  deployment: ReturnType<typeof useDeployment>;
}

export function StepManifest({ deployment }: StepManifestProps) {
  const { manifestJson, setManifestJson, parsedManifest } = deployment;
  const [mode, setMode] = useState<Mode>("form");

  // Form state derived from parsed manifest
  const formValues = parsedManifest.ok
    ? {
        host: parsedManifest.value.target?.vps?.host ?? "",
        scenarioId: parsedManifest.value.scenario?.id ?? "",
        domain: parsedManifest.value.edge?.domain ?? "",
        uiPort: parsedManifest.value.ports?.ui ?? 3000,
        apiPort: parsedManifest.value.ports?.api ?? 3001,
        wsPort: parsedManifest.value.ports?.ws ?? 3002,
        includePackages: parsedManifest.value.bundle?.include_packages ?? true,
        includeAutoheal: parsedManifest.value.bundle?.include_autoheal ?? true,
        caddyEnabled: parsedManifest.value.edge?.caddy?.enabled ?? true,
      }
    : null;

  const updateFormField = (field: string, value: string | number | boolean) => {
    if (!parsedManifest.ok) return;

    const manifest = { ...parsedManifest.value };

    switch (field) {
      case "host":
        manifest.target = { ...manifest.target, vps: { ...manifest.target.vps, host: value as string } };
        break;
      case "scenarioId":
        manifest.scenario = { ...manifest.scenario, id: value as string };
        manifest.dependencies = { ...manifest.dependencies, scenarios: [value as string] };
        break;
      case "domain":
        manifest.edge = { ...manifest.edge, domain: value as string };
        break;
      case "uiPort":
        manifest.ports = { ...manifest.ports, ui: value as number };
        break;
      case "apiPort":
        manifest.ports = { ...manifest.ports, api: value as number };
        break;
      case "wsPort":
        manifest.ports = { ...manifest.ports, ws: value as number };
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
    }

    setManifestJson(JSON.stringify(manifest, null, 2));
  };

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
                hint="The IP address or hostname of your VPS"
              />
              <Input
                label="Domain"
                placeholder="example.com"
                value={formValues.domain}
                onChange={(e) => updateFormField("domain", e.target.value)}
                hint="Domain name for HTTPS (requires DNS configured)"
              />
            </div>
          </div>

          {/* Scenario Section */}
          <div className="space-y-4 md:col-span-2">
            <h3 className="text-sm font-medium text-slate-300 flex items-center gap-2">
              Scenario
              <HelpTooltip content="The scenario to deploy. Must exist in your Vrooli installation." />
            </h3>
            <Input
              label="Scenario ID"
              placeholder="landing-page-business-suite"
              value={formValues.scenarioId}
              onChange={(e) => updateFormField("scenarioId", e.target.value)}
              hint="The unique identifier of the scenario to deploy"
            />
          </div>

          {/* Ports Section */}
          <div className="space-y-4 md:col-span-2">
            <h3 className="text-sm font-medium text-slate-300 flex items-center gap-2">
              Ports
              <HelpTooltip content="Fixed ports for the deployed services. Caddy will proxy these on 80/443." />
            </h3>
            <div className="grid gap-4 sm:grid-cols-3">
              <Input
                label="UI Port"
                type="number"
                value={formValues.uiPort}
                onChange={(e) => updateFormField("uiPort", parseInt(e.target.value, 10))}
              />
              <Input
                label="API Port"
                type="number"
                value={formValues.apiPort}
                onChange={(e) => updateFormField("apiPort", parseInt(e.target.value, 10))}
              />
              <Input
                label="WebSocket Port"
                type="number"
                value={formValues.wsPort}
                onChange={(e) => updateFormField("wsPort", parseInt(e.target.value, 10))}
              />
            </div>
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
