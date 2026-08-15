import { ExternalLink, Info } from "lucide-react";

export function StepIntegrationsDeferred() {
  return <div data-testid="step-integrations-deferred">
    <h1 className="text-xl font-semibold sm:text-2xl">Integrations</h1>
    <div data-testid="step-integrations-deferred" className="mt-6 rounded-xl border border-warning/30 bg-warning-surface p-5" role="status">
      <Info className="h-5 w-5 text-warning" aria-hidden="true" />
      <h2 className="mt-3 font-medium text-warning">Integration setup is deferred</h2>
      <p className="mt-2 text-sm text-foreground">Connection bindings and OAuth flows are owned by the future integration-hub capability. This onboarding step does not create placeholder connections or fake bindings.</p>
      <a data-testid="integrations-contract-link" className="mt-3 inline-flex min-h-11 items-center gap-1 rounded px-2 text-sm text-warning underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-focus" href="/docs/configuration/integrations/connectors.md" target="_blank" rel="noreferrer">Read the integration contract <ExternalLink className="h-3.5 w-3.5" /></a>
    </div>
  </div>;
}
