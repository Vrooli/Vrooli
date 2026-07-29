import { ExternalLink, Info } from "lucide-react";

export function StepIntegrationsDeferred() {
  return <div data-testid="step-integrations-deferred">
    <h1 className="text-xl font-semibold sm:text-2xl">Integrations</h1>
    <div className="mt-6 rounded-xl border border-amber-400/30 bg-amber-400/10 p-5" role="status">
      <Info className="h-5 w-5 text-amber-300" aria-hidden="true" />
      <h2 className="mt-3 font-medium text-amber-100">Integration setup is deferred</h2>
      <p className="mt-2 text-sm text-slate-200">Connection bindings and OAuth flows are owned by the future integration-hub capability. This onboarding step does not create placeholder connections or fake bindings.</p>
      <a className="mt-3 inline-flex items-center gap-1 text-sm text-amber-200 underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-amber-300" href="/docs/configuration/integrations/connectors.md" target="_blank" rel="noreferrer">Read the integration contract <ExternalLink className="h-3.5 w-3.5" /></a>
    </div>
  </div>;
}
