import { useQuery } from "@tanstack/react-query";
import { Loader2 } from "lucide-react";
import { fetchDerivedResources } from "../../api/resources";
import type { OperatorState } from "../../api/operatorstate";
import { i18n } from "../../i18n";
import { Checkbox } from "@vrooli/react-component-library/Checkbox/1";

export function DerivedResourceStep({ selected, operatorState, onToggle }: { selected: Set<string>; operatorState: OperatorState | null; onToggle: (name: string, enabled: boolean) => void }) {
  const { data, isLoading, error } = useQuery({ queryKey: ["selection-resources", Array.from(selected).sort().join(",")], queryFn: () => fetchDerivedResources() });
  const required = data?.required ?? [];
  const optional = data?.optional ?? [];
  const standalone = data?.standalone ?? [];
  return <div data-testid="step-derived-resources">
    <h1 className="text-xl font-semibold sm:text-2xl">{i18n.t("onboarding.resources.heading")}</h1>
    <p className="mt-2 text-sm text-muted">{i18n.t("onboarding.resources.intro")}</p>
    {isLoading && <p className="mt-6 flex items-center gap-2 text-muted" role="status"><Loader2 className="h-4 w-4 animate-spin" />{i18n.t("onboarding.resources.loading")}</p>}
    {error && <p className="mt-6 text-danger" role="alert">{i18n.t("onboarding.resources.error")}</p>}
    {!isLoading && !error && <div className="mt-6 space-y-5">
      <ResourceGroup title={i18n.t("onboarding.resources.required")} items={required} locked operatorState={operatorState} onToggle={onToggle} />
      <ResourceGroup title={i18n.t("onboarding.resources.optional")} items={optional} operatorState={operatorState} onToggle={onToggle} />
      <ResourceGroup title={i18n.t("onboarding.resources.standalone")} items={standalone} operatorState={operatorState} onToggle={onToggle} />
      {required.length + optional.length + standalone.length === 0 && <p data-testid="empty-note" role="status" className="text-muted">{i18n.t("onboarding.resources.none")}</p>}
    </div>}
  </div>;
}

function ResourceGroup({ title, items, locked = false, operatorState, onToggle }: { title: string; items: { name: string; description?: string; category?: string; enabled?: boolean }[]; locked?: boolean; operatorState: OperatorState | null; onToggle: (name: string, enabled: boolean) => void }) {
  return <section aria-labelledby={`resource-group-${title.toLowerCase()}`} data-testid={`resources-${title.toLowerCase()}`} role="group">
    <h2 id={`resource-group-${title.toLowerCase()}`} className="text-sm font-semibold uppercase tracking-wide text-muted">{title}{locked && ` · ${i18n.t("onboarding.resources.alwaysIncluded")}`}</h2>
    {items.length === 0 ? <p className="mt-2 text-sm text-muted">{i18n.t("onboarding.resources.noResources", { group: title.toLowerCase() })}</p> : <ul className="mt-2 grid gap-2 sm:grid-cols-2">{items.map((resource) => { const checked = locked || operatorState?.resources?.[resource.name]?.enabled === true || resource.enabled === true; return <li key={resource.name} className="rounded-lg border border-muted bg-surface-muted px-3 py-2 text-sm"><Checkbox data-testid="resource-entry" checked={checked} disabled={locked} onCheckedChange={(enabled) => onToggle(resource.name, enabled)} label={<><span className="font-medium text-foreground">{resource.name}</span>{resource.category && <span className="ml-2 text-xs text-muted">{resource.category}</span>}{resource.description && <span className="mt-1 block text-xs text-muted">{resource.description}</span>}{locked && <span className="mt-1 block text-xs text-primary-soft" data-testid="required-reason" role="note">{i18n.t("onboarding.resources.requiredByClosure")}</span>}</>} /></li>; })}</ul>}
  </section>;
}
