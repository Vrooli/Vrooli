import { useDesktopSession } from "./useDesktopSession";
import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { fromJsonString, toJsonString } from "@bufbuild/protobuf";
import { SurfaceRefSchema } from "@vrooli/proto-types/common/v1/surface_pb";
import { listSurfaces, SurfaceKind, SurfaceCapabilityState, type SurfaceRef, type SurfaceDescriptor } from "../../api/surfaces";
import { Button } from "../../components/ui/button";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { portalLearningSensor } from "../../lib/learningSensors";

import { DesktopSessionPanel } from "./DesktopSessionPanel";
import { EmbeddedScenarioFrame } from "../companion/EmbeddedScenarioFrame";

const storageKey = "portal.selected-surface.v1";
// Labels are presentation; exact owner IDs and host relation identify selection.
function surfaceKey(ref: SurfaceRef | undefined): string {
  return ref ? JSON.stringify([ref.target?.ownerScenario, ref.target?.resourceId, ref.target?.hostNodeId, ref.ownerScenario, ref.surfaceId]) : "";
}
function restoreSelection(): SurfaceRef | undefined {
  try {
    const raw = sessionStorage.getItem(storageKey);
    return raw ? fromJsonString(SurfaceRefSchema, raw) : undefined;
  } catch { return undefined; }
}

function DevicePanel({ surface }: { surface: SurfaceDescriptor }) {
  const { t } = useTranslation();
  const buttons = surface.capabilities.filter(fact => fact.capability.startsWith("device.button."));
  const properties = surface.capabilities.filter(fact => fact.capability.startsWith("device.property."));
  const label = (capability: string) => capability.replace(/^device\.(button|property)\./, "").replace(/[._:-]+/g, " ");
  const renderFacts = (facts: typeof buttons) => facts.length ? <ul className="list-disc pl-5">{facts.map(fact => <li key={fact.capability}><span className="capitalize">{label(fact.capability)}</span>: {fact.state === SurfaceCapabilityState.READY ? t(strings.surfaces.ready) : t(strings.surfaces.deviceCapabilityUnavailable)}</li>)}</ul> : <p className="text-app-muted-foreground">{t(strings.surfaces.deviceCapabilityUnavailable)}</p>;
  return <div role="region" aria-labelledby="device-panel-heading" className="rounded border border-app-border p-3">
    <h5 id="device-panel-heading" className="font-medium">{t(strings.surfaces.devicePanelTitle)}</h5>
    <p className="mt-1 text-app-muted-foreground">{t(strings.surfaces.devicePanelDescription)}</p>
    <h6 className="mt-2 font-medium">{t(strings.surfaces.deviceButtons)}</h6>
    {renderFacts(buttons)}
    <h6 className="mt-2 font-medium">{t(strings.surfaces.deviceProperties)}</h6>
    {renderFacts(properties)}
  </div>;
}

function ScenarioPanel({ surface }: { surface: SurfaceDescriptor }) {
  const { t } = useTranslation();
  const scenario = surface.ref ? surface.ref.ownerScenario.trim() : "";
  // A provider must opt into the embedded proxy contract. The surface
  // identity is still the owner reference; the URL is only a same-origin
  // presentation route and carries no authority or credentials.
  const embeddable = surface.protocolVersions.includes("vrooli.scenario.embed.v1") && /^[a-z][a-z0-9]*(-[a-z0-9]+)*$/.test(scenario);
  if (!embeddable) return <p className="text-app-muted-foreground">{t(strings.surfaces.inspectOnly)}</p>;
  return (
    <div role="region" aria-label={surface.displayLabel} className="space-y-2">
      <EmbeddedScenarioFrame src={`/embedded/${scenario}/`} title={surface.displayLabel} className="min-h-96 w-full rounded border border-app-border" />
      <p className="text-xs text-app-muted-foreground">{surface.ref?.ownerScenario}/{surface.ref?.surfaceId}</p>
    </div>
  );
}

export function SurfaceCatalog() {
  const { t } = useTranslation();
  const desktop = useDesktopSession();
  const [selectedRef, setSelectedRef] = useState<SurfaceRef | undefined>(restoreSelection);
  const [desktopActive, setDesktopActive] = useState(false);
  const [now, setNow] = useState(Date.now);
  const query = useQuery({ queryKey: ["surface-catalog"], queryFn: ({ signal }) => listSurfaces(signal), retry: false, staleTime: 15_000 });
  useEffect(() => {
    if (!query.data) return;
    portalLearningSensor.record({
      kind: "discovery",
      temperature: "warm",
      topology: "local",
      targetClass: "device-panel",
    });
  }, [query.data]);
  useEffect(() => {
    const timer = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(timer);
  }, []);
  useEffect(() => {
    try {
      if (selectedRef) sessionStorage.setItem(storageKey, toJsonString(SurfaceRefSchema, selectedRef));
      else sessionStorage.removeItem(storageKey);
    } catch { /* Storage denial must not prevent browsing or chatting. */ }
  }, [selectedRef]);
  const surfaces = query.data?.surfaces ?? [];
  const effectiveRef = desktop.active?.ref.surface ?? selectedRef;
  const selectedKey = surfaceKey(effectiveRef);
  const selected = surfaces.find(surface => surfaceKey(surface.ref) === selectedKey);
  const kindLabel = (kind: SurfaceKind) => t(kind === SurfaceKind.TERMINAL ? strings.surfaces.terminal : kind === SurfaceKind.DESKTOP ? strings.surfaces.desktop : kind === SurfaceKind.BROWSER ? strings.surfaces.browser : kind === SurfaceKind.SCENARIO ? strings.surfaces.scenario : strings.surfaces.device);
  return <section aria-labelledby="surfaces-heading" className="rounded-xl border border-app-border bg-app-card p-4">
    <div className="flex items-center justify-between gap-3">
      <h3 id="surfaces-heading" className="font-semibold">{t(strings.surfaces.title)}</h3>
      <Button type="button" variant="outline" size="sm" onClick={() => void query.refetch()} disabled={query.isFetching}>{t(strings.surfaces.refresh)}</Button>
    </div>
    <p className="mt-2 text-sm text-app-muted-foreground">{t(strings.surfaces.description)}</p>
    {query.isPending && <p role="status">{t(strings.surfaces.loading)}</p>}
    {query.isError && <p role="alert" className="mt-2 text-sm">{t(strings.surfaces.refreshFailed)}</p>}
    {query.data?.sources.filter(source => source.state !== "ready").map(source => <p role="status" key={source.ownerScenario} className="mt-2 text-sm">{t(strings.surfaces.sourceUnavailable, { owner: source.ownerScenario })}</p>)}
    {!query.isPending && !surfaces.length && <p className="mt-3 text-sm">{t(strings.surfaces.empty)}</p>}
    <label htmlFor="surface-selection" className="mt-3 block text-sm font-medium">{t(strings.surfaces.select)}</label>
    <select disabled={desktopActive || Boolean(desktop.active) || desktop.opening || desktop.recoveryRequired} id="surface-selection" className="mt-1 w-full rounded-control border border-app-border bg-app-background p-2 text-app-foreground" value={selectedKey} onChange={event => setSelectedRef(surfaces.find(surface => surfaceKey(surface.ref) === event.target.value)?.ref)}>
      <option value="">{t(strings.surfaces.none)}</option>
      {effectiveRef && !selected && <option value={selectedKey}>{t(strings.surfaces.unavailableSelection)}</option>}
      {surfaces.map(surface => <option key={surfaceKey(surface.ref)} value={surfaceKey(surface.ref)}>{surface.displayLabel} — {kindLabel(surface.kind)} · {surface.ref?.target?.ownerScenario}/{surface.ref?.target?.resourceId}</option>)}
    </select>
    {effectiveRef && !selected && <p role="status" className="mt-2 text-sm">{t(strings.surfaces.selectionMissing)}</p>}
    {selected && <div className="mt-3 space-y-2 text-sm" aria-live="polite">
      <p className="font-medium">{selected.displayLabel} — {kindLabel(selected.kind)}</p>
      <p>{t(strings.surfaces.owner, { owner: selected.ref?.ownerScenario })}</p>
      <ul>{selected.capabilities.map(fact => {
        const expired = !fact.expiresAt || Number(fact.expiresAt.seconds) * 1000 <= now;
        const label = expired ? strings.surfaces.stale : fact.state === SurfaceCapabilityState.READY ? strings.surfaces.ready : fact.state === SurfaceCapabilityState.DENIED ? strings.surfaces.denied : fact.state === SurfaceCapabilityState.MISSING || fact.state === SurfaceCapabilityState.UNSUPPORTED ? strings.surfaces.unavailable : strings.surfaces.checkRequired;
        return <li key={fact.capability}>{fact.capability}: {t(label)}</li>;
      })}</ul>
      {selected.kind === SurfaceKind.DESKTOP && selected.ref && !(desktop.restored && desktop.recoveryRequired) ? <DesktopSessionPanel key={selectedKey} surface={selected.ref} onActive={setDesktopActive} /> : selected.kind === SurfaceKind.DEVICE_PANEL ? <DevicePanel surface={selected} /> : selected.kind === SurfaceKind.SCENARIO ? <ScenarioPanel surface={selected} /> : <p className="text-app-muted-foreground">{t(strings.surfaces.inspectOnly)}</p>}
    </div>}
  </section>;
}
