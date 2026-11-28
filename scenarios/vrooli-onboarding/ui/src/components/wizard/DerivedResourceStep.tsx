import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Loader2 } from "lucide-react";
import { fetchDerivedResources } from "../../api/resources";
import type { OperatorState } from "../../api/operatorstate";
import { i18n } from "../../i18n";
import { CardShell } from "@vrooli/react-component-library/CardShell/1";
import { readCoreAuthority } from "../../api/coreAuthority";

export function DerivedResourceStep({ selected, operatorState, onToggle, onTierChange = () => undefined, target = "local" }: { selected: Set<string>; operatorState: OperatorState | null; onToggle: (name: string, enabled: boolean) => void; onTierChange?: (important: {kind: "scenario" | "resource"; name: string}[], members: {kind: "scenario" | "resource"; name: string}[], expectedRevision: string) => void; target?: string }) {
  const { data, isLoading, error } = useQuery({ queryKey: ["selection-resources", target, Array.from(selected).sort().join(",")], queryFn: () => fetchDerivedResources(target) });
  const authority = useQuery({ queryKey: ["core-authority", target], queryFn: () => readCoreAuthority(target) });
  const [important, setImportant] = useState<{kind: "scenario" | "resource"; name: string}[]>([]);
  const [coreMembers, setCoreMembers] = useState<{kind: "scenario" | "resource"; name: string}[]>([]);
  useEffect(() => {
    setImportant(authority.data?.authority.important ?? []);
    setCoreMembers(authority.data?.authority.members ?? []);
  }, [authority.data]);
  const changeTier = (nextImportant: {kind: "scenario" | "resource"; name: string}[], nextCore: {kind: "scenario" | "resource"; name: string}[]) => {
    setImportant(nextImportant); setCoreMembers(nextCore);
    if (authority.data?.revision) onTierChange(nextImportant, nextCore, authority.data.revision);
  };
  const required = data?.required ?? [];
  const optional = data?.optional ?? [];
  const standalone = data?.standalone ?? [];
  return <div data-testid="step-derived-resources">
    <h1 className="text-xl font-semibold sm:text-2xl">{i18n.t("onboarding.resources.heading")}</h1>
    <p className="mt-2 text-sm text-muted">{i18n.t("onboarding.resources.intro")}</p>
    {isLoading && <p className="mt-6 flex items-center gap-2 text-muted" role="status"><Loader2 className="h-4 w-4 animate-spin" />{i18n.t("onboarding.resources.loading")}</p>}
    {error && <p className="mt-6 text-danger" role="alert">{i18n.t("onboarding.resources.error")}</p>}
    {!isLoading && !error && <div className="mt-6 space-y-5">
      <ResourceGroup title={i18n.t("onboarding.resources.required")} items={required} locked operatorState={operatorState} onToggle={onToggle} important={important} coreMembers={coreMembers} onTierChange={changeTier} tierReady={Boolean(authority.data?.revision)} />
      <ResourceGroup title={i18n.t("onboarding.resources.optional")} items={optional} operatorState={operatorState} onToggle={onToggle} important={important} coreMembers={coreMembers} onTierChange={changeTier} tierReady={Boolean(authority.data?.revision)} />
      <ResourceGroup title={i18n.t("onboarding.resources.standalone")} items={standalone} operatorState={operatorState} onToggle={onToggle} important={important} coreMembers={coreMembers} onTierChange={changeTier} tierReady={Boolean(authority.data?.revision)} />
      {required.length + optional.length + standalone.length === 0 && <p data-testid="empty-note" role="status" className="text-muted">{i18n.t("onboarding.resources.none")}</p>}
    </div>}
  </div>;
}

function ResourceGroup({ title, items, locked = false, operatorState, onToggle, important, coreMembers, onTierChange, tierReady }: { title: string; items: { name: string; description?: string; category?: string; enabled?: boolean }[]; locked?: boolean; operatorState: OperatorState | null; onToggle: (name: string, enabled: boolean) => void; important: {kind: "scenario" | "resource"; name: string}[]; coreMembers: {kind: "scenario" | "resource"; name: string}[]; onTierChange: (important: {kind: "scenario" | "resource"; name: string}[], members: {kind: "scenario" | "resource"; name: string}[]) => void; tierReady: boolean }) {
  return <section aria-labelledby={`resource-group-${title.toLowerCase()}`} data-testid={`resources-${title.toLowerCase()}`} role="group">
    <h2 id={`resource-group-${title.toLowerCase()}`} className="text-sm font-semibold uppercase tracking-wide text-muted">{title}{locked && ` · ${i18n.t("onboarding.resources.alwaysIncluded")}`}</h2>
    {items.length === 0 ? <p className="mt-2 text-sm text-muted">{i18n.t("onboarding.resources.noResources", { group: title.toLowerCase() })}</p> : <ul className="mt-2 grid gap-3 sm:grid-cols-2">{items.map((resource) => { const checked = locked || operatorState?.resources?.[resource.name]?.enabled === true || resource.enabled === true; const importantSelected = important.some((m) => m.kind === "resource" && m.name === resource.name); const coreSelected = coreMembers.some((m) => m.kind === "resource" && m.name === resource.name); const updateTier = (tier: "important" | "core", enabled: boolean) => { const othersImportant = important.filter((m) => !(m.kind === "resource" && m.name === resource.name)); const othersCore = coreMembers.filter((m) => !(m.kind === "resource" && m.name === resource.name)); if (enabled && tier === "important") othersImportant.push({kind: "resource", name: resource.name}); if (enabled && tier === "core") othersCore.push({kind: "resource", name: resource.name}); onTierChange(othersImportant, othersCore); }; return <li key={resource.name}><CardShell
      className="scenario-choice-card"
      testId="resource-entry"
      interactive={!locked}
      selectLabel={`${i18n.t("onboarding.resources.enable")} ${resource.name}`}
      selection={{ selectionMode: true, selected: checked, disabled: locked, disabledReason: locked ? i18n.t("onboarding.resources.requiredByClosure") : undefined, onToggleSelect: () => onToggle(resource.name, !checked) }}
    >
      <div className="scenario-choice-card__body">
        <div className="scenario-choice-card__heading"><span className="scenario-choice-card__name">{resource.name}</span>{resource.category && <span className="scenario-choice-card__badge">{resource.category}</span>}</div>
        {resource.description && <p className="scenario-choice-card__description">{resource.description}</p>}
        {locked && <span className="scenario-choice-card__supporting" data-testid="required-reason" role="note">{i18n.t("onboarding.resources.requiredByClosure")}</span>}
      </div>
    </CardShell><div className="mt-2 flex flex-wrap gap-4 text-xs"><label className="flex items-center gap-2"><input type="checkbox" aria-label={i18n.t("onboarding.resources.importantResource", { name: resource.name })} checked={importantSelected} disabled={!tierReady || !checked || locked || coreSelected} onChange={(event) => updateTier("important", event.target.checked)} />{i18n.t("onboarding.resources.importantResourceLabel")}</label><label className="flex items-center gap-2"><input type="checkbox" aria-label={i18n.t("onboarding.resources.coreResource", { name: resource.name })} checked={coreSelected} disabled={!tierReady || !checked || locked || importantSelected} onChange={(event) => updateTier("core", event.target.checked)} />{i18n.t("onboarding.resources.coreResourceLabel")}</label></div></li>; })}</ul>}
  </section>;
}
