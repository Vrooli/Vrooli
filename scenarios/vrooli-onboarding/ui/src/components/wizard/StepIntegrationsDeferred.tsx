import { useQuery } from "@tanstack/react-query";
import { ExternalLink, Info } from "lucide-react";
import { fetchV2Readiness } from "../../lib/api";

export function StepIntegrationsDeferred() {
  const { data } = useQuery({ queryKey: ["v2-readiness"], queryFn: fetchV2Readiness });
  const integrations = data?.integrations.filter((item) => item.category === "integration") ?? [];
  return <div data-testid="step-integrations-deferred">
    <h1 className="text-xl font-semibold sm:text-2xl">Integrations</h1>
    <div data-testid="step-integrations-deferred" className="mt-6 rounded-xl border border-warning/30 bg-warning-surface p-5" role="status">
      <Info className="h-5 w-5 text-warning" aria-hidden="true" />
      <h2 className="mt-3 font-medium text-warning">Integration setup is deferred</h2>
      <p className="mt-2 text-sm text-foreground">Connection bindings and authorization flows are owned by the integration capability. This onboarding step reports declared requirements but does not create placeholder connections or fake bindings.</p>
      {integrations.length > 0 && <ul data-testid="declared-integrations" className="mt-4 space-y-2">{integrations.map((integration) => <li key={integration.name} className="rounded-lg border border-warning/20 p-3 text-sm"><span className="font-medium">{integration.name}</span>{integration.required && <span className="ml-2 text-muted">required</span>}{integration.detail && <p className="mt-1 text-xs text-muted">{integration.detail}</p>}</li>)}</ul>}
      {data && integrations.length === 0 && <p className="mt-3 text-sm text-muted">No selected capability declares an external integration requirement.</p>}
      <a data-testid="integrations-contract-link" className="mt-3 inline-flex min-h-11 items-center gap-1 rounded px-2 text-sm text-warning underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-focus" href="/docs/configuration/integrations/connectors.md" target="_blank" rel="noreferrer">Read the integration contract <ExternalLink className="h-3.5 w-3.5" /></a>
    </div>
  </div>;
}
