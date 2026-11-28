import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { fetchHostRequirements, type HostRequirement } from "../../api/host";
import { hostMutationCapability, type OnboardingTargetKind } from "../../api/hostCapabilities";
import { useOnboardingTargetCapability } from "../../api/useOnboardingTargetCapability";
import { i18n } from "../../i18n";
import { CardShell } from "@vrooli/react-component-library/CardShell/1";
import { Checkbox } from "@vrooli/react-component-library/Checkbox/1";
import { Input } from "@vrooli/react-component-library/Input/1";
import { Select } from "@vrooli/react-component-library/Select/1";

function Requirements({ title, items, onToggle, onConfig, editable, targetKind }: { title: string; items: HostRequirement[]; onToggle: (name: string, value: boolean) => void; onConfig?: (name: string, config: Record<string, unknown>) => void; editable: boolean; targetKind: OnboardingTargetKind }) {
  const groupTestId = title.toLowerCase() === "tools" ? "host-tools" : "host-safeguards";
  return <section className="mt-6" data-testid={groupTestId} role="group"><h2 className="text-lg font-medium">{title}</h2>{title === "Safeguards" && <p data-testid="change-summary" role="note" className="mt-2 text-xs text-muted">{i18n.t("onboarding.host.changeSummary")}</p>}<div className="mt-3 space-y-3">{items.length === 0 ? <p className="text-sm text-muted">{i18n.t("onboarding.host.none", { group: title.toLowerCase() })}</p> : items.map((item) => {
    const selected = item.status === "required" || item.status === "opted_in";
    const disabledReason = item.required ? i18n.t("onboarding.host.required") : !editable ? i18n.t("onboarding.host.remoteReadOnly") : undefined;
    return <CardShell key={item.name} className="scenario-choice-card" testId="requirement-entry" interactive={false} selectLabel={item.name} selection={{ selectionMode: true, selected, disabled: item.required || !editable, disabledReason, onToggleSelect: () => { if (editable) onToggle(item.name, !selected); } }}>
      <div className="scenario-choice-card__body">
        <div className="scenario-choice-card__heading"><span className="scenario-choice-card__name">{item.name}</span><span className="scenario-choice-card__resources">{item.required && <span className="scenario-choice-card__badge">{i18n.t("onboarding.host.required")}</span>}{item.risk && <span className="scenario-choice-card__badge" data-testid="risk-indicator" role="note">{i18n.t("onboarding.host.risk", { risk: item.risk })}</span>}</span></div>
        <p className="scenario-choice-card__description">{item.reason || item.description}</p>
        <span className="scenario-choice-card__supporting" data-testid="privilege-indicator" role="note">{i18n.t("onboarding.host.privilege", { value: item.privilege || i18n.t("onboarding.host.notDeclared") })} · {i18n.t("onboarding.host.bundling", { value: item.bundling || i18n.t("onboarding.host.notDeclared") })}{item.platforms?.length ? i18n.t("onboarding.host.platforms", { value: item.platforms.join(", ") }) : ""}</span>
        {targetKind === "local" && onConfig && item.config_schema?.properties && <ConfigFields item={item} onChange={(config) => onConfig(item.name, config)} disabled={!editable} />}
      </div>
    </CardShell>;
  })}</div></section>;
}
function ConfigFields({ item, onChange, disabled }: { item: HostRequirement; onChange: (config: Record<string, unknown>) => void; disabled: boolean }) {
  const schema = item.config_schema as { properties?: Record<string, { type?: string; title?: string; description?: string; enum?: unknown[]; default?: unknown }> };
  const [values, setValues] = useState<Record<string, unknown>>(item.config ?? {});
  return <fieldset data-testid={`config-${item.name}`} disabled={disabled} className="mt-3 space-y-2 border-t border-border-muted pt-3">
    <legend className="text-sm font-medium">{i18n.t("onboarding.host.configuration")}</legend>
    {Object.entries(schema.properties ?? {}).map(([key, property]) => {
      const value = values[key] ?? property.default ?? "";
      const label = property.title || key;
      const update = (next: unknown) => {
        const config = { ...values, [key]: next };
        setValues(config);
        onChange(config);
      };
      if (property.enum?.length) return <div key={key} className="block text-sm">
        <span className="block">{label}</span>
        <Select aria-label={label} className="mt-1" value={primitiveInputValue(value)} onValueChange={update as (value: string) => void} options={property.enum.map((option) => {
          const text = primitiveInputValue(option);
          return { value: text, label: text };
        })} />
      </div>;
      if (property.type === "boolean") return <Checkbox key={key} aria-label={label} checked={Boolean(value)} onCheckedChange={update} label={label} />;
      if (!["string", "number", "integer"].includes(property.type ?? "string")) return <p key={key} className="text-xs text-muted" role="note">{i18n.t("onboarding.host.unsupportedConfigType", { type: property.type ?? "unknown" })} {property.description || label}</p>;
      return <label key={key} className="block text-sm">{label}
        <Input aria-label={label} type={property.type === "number" || property.type === "integer" ? "number" : "text"} className="mt-1" value={primitiveInputValue(value)} onChange={(event) => update(property.type === "number" || property.type === "integer" ? Number(event.target.value) : event.target.value)} />
        {property.description && <span className="mt-1 block text-xs text-muted">{property.description}</span>}
      </label>;
    })}
  </fieldset>;
}

function primitiveInputValue(value: unknown): string {
  if (typeof value === "string" || typeof value === "number" || typeof value === "boolean") return String(value);
  return "";
}
export function HostRequirementStep({ onTool, onSafeguard, onHostConfig, target = "local" }: { onTool: (name: string, value: boolean) => void; onSafeguard: (name: string, value: boolean) => void; onHostConfig?: (kind: "host_tools" | "host_safeguards", name: string, config: Record<string, unknown>) => void; target?: string }) {
  const { capability: targetCapability, resolving, resolutionError, refreshTarget } = useOnboardingTargetCapability(target);
  const { data, isLoading, isError, error } = useQuery({ queryKey: ["host-requirements", target], queryFn: () => fetchHostRequirements(target), enabled: !resolving && !resolutionError && targetCapability.kind !== "unsupported" });
  const ownerReady = !isLoading && !isError && Boolean(data);
  const toolsEditable = targetCapability.kind === "local" && ownerReady && hostMutationCapability(target, "tool").supported;
  const safeguardsEditable = targetCapability.kind === "local" && ownerReady && hostMutationCapability(target, "safeguard").supported;
  const targetKind = targetCapability.kind;
  const targetResolved = !resolving && !resolutionError && targetKind !== "unsupported";
  return <div data-testid="step-host-requirements"><h1 className="text-xl font-semibold sm:text-2xl">{i18n.t("onboarding.host.heading")}</h1><p className="mt-2 text-sm text-muted">{i18n.t("onboarding.host.intro")}</p>{targetKind === "remote-node" && <p className="mt-3 rounded-md border border-border-muted p-3 text-sm" role="note" data-testid="remote-host-read-only">{i18n.t("onboarding.host.remoteReadOnly")}</p>}{resolving && <p role="status" className="mt-6">{i18n.t("onboarding.host.checkingTarget")}</p>}{resolutionError && <><p role="alert" className="mt-6 text-danger">{i18n.t("onboarding.host.targetVerifyError")} {resolutionError instanceof Error ? resolutionError.message : "target list unavailable"}</p><button type="button" onClick={() => void refreshTarget()}>{i18n.t("onboarding.host.checkTargetAgain")}</button></>}{targetKind === "unsupported" && !resolving && !resolutionError && <p className="mt-3 rounded-md border border-border-muted p-3 text-sm" role="alert" data-testid="unsupported-host-target">{targetCapability.reason}</p>}{isLoading && <p role="status" className="mt-6">{i18n.t("onboarding.host.loading")}</p>}{isError && <p role="alert" className="mt-6 text-danger">{error instanceof Error ? error.message : i18n.t("onboarding.host.error")}</p>}{targetResolved && <><Requirements title={i18n.t("onboarding.host.tools")} items={data?.tools ?? []} onToggle={onTool} onConfig={onHostConfig ? (name, config) => onHostConfig("host_tools", name, config) : undefined} editable={toolsEditable} targetKind={targetKind} /><Requirements title={i18n.t("onboarding.host.safeguards")} items={data?.safeguards ?? []} onToggle={onSafeguard} onConfig={onHostConfig ? (name, config) => onHostConfig("host_safeguards", name, config) : undefined} editable={safeguardsEditable} targetKind={targetKind} /></>}</div>;
}
