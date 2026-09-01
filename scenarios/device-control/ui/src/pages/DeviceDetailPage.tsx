/* eslint-disable @typescript-eslint/no-misused-promises, no-restricted-syntax, @typescript-eslint/no-base-to-string */
import { useCallback, useEffect, useMemo, useState } from "react";
import { useParams, useSearchParams } from "react-router-dom";
import { actuateDevice, acquireSession, describeDevice, listSessions, readDeviceState, releaseSession, watchDeviceEvents, type Capability, type Device, type DeviceState, type PropertyDescriptor, type Session } from "../api/deviceControl";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { selectors } from "../consts/selectors";

type PanelProps = { device: Device; state: DeviceState; refresh: () => Promise<void> };

function capabilities(device: Device): Capability[] {
  const fromTransports = (device.transports ?? []).flatMap((transport) => Object.values(transport.capabilities ?? {}));
  return [...device.capabilities, ...fromTransports].filter((item, index, all) => all.findIndex((candidate) => candidate.name === item.name && candidate.status === item.status) === index);
}
function available(device: Device, name: string) { return capabilities(device).some((item) => item.name === name && item.status === "available"); }

async function press(device: Device, key: string, text?: string) {
  const lease = await acquireSession(device.id, "browser-operator");
  try { await actuateDevice(device.id, { actor: "browser-operator", lease_token: lease.session.lease_token ?? "", key: text ? undefined : key, text }); } finally { await releaseSession(lease.session.id).catch(() => undefined); }
}

function RemotePanel({ device, refresh }: PanelProps) {
  const keys = ["DPAD_UP", "DPAD_LEFT", "DPAD_CENTER", "DPAD_RIGHT", "DPAD_DOWN", "BACK", "HOME"];
  return <Card data-testid={selectors.pages.deviceRemotePanel}><CardHeader><CardTitle>Directional remote</CardTitle></CardHeader><CardContent className="flex flex-col gap-3"><div className="grid grid-cols-3 gap-2">{keys.map((key) => <Button key={key} onClick={() => press(device, key).then(refresh)}>{key}</Button>)}</div><input aria-label="Remote text" className="rounded-md border p-2" placeholder="Type text" onKeyDown={(event) => { if (event.key === "Enter") void press(device, "", event.currentTarget.value).then(refresh); }} /></CardContent></Card>;
}
async function mediaActuate(device: Device, media: string) { const lease = await acquireSession(device.id, "browser-operator"); try { await actuateDevice(device.id, { actor: "browser-operator", lease_token: lease.session.lease_token ?? "", media }); } finally { await releaseSession(lease.session.id).catch(() => undefined); } }
async function propertyActuate(device: Device, name: string, value: unknown) { const lease = await acquireSession(device.id, "browser-operator"); try { await actuateDevice(device.id, { actor: "browser-operator", lease_token: lease.session.lease_token ?? "", property: name, value }); } finally { await releaseSession(lease.session.id).catch(() => undefined); } }
function MediaPanel({ device, state, refresh }: PanelProps) { const current = state.properties?.volume?.value; const muted = state.properties?.muted?.value; return <Card data-testid={selectors.pages.deviceMediaPanel}><CardHeader><CardTitle>Media transport</CardTitle></CardHeader><CardContent className="flex flex-wrap gap-2"><div data-testid={selectors.pages.deviceNowPlaying} className="basis-full">Now playing: {String(state.properties?.application?.value ?? "unknown")} · {String(state.properties?.player_state?.value ?? "unknown")}</div>{["play", "pause", "stop", "previous", "next"].map((action) => <Button key={action} onClick={() => mediaActuate(device, action).then(refresh)}>{action}</Button>)}<label className="flex items-center gap-2">Volume<input type="range" min="0" max="1" step="0.01" aria-label="Volume" value={typeof current === "number" ? current : 0} onChange={(event) => void propertyActuate(device, "volume", Number(event.currentTarget.value)).then(refresh)} /></label><label className="flex items-center gap-2">Mute<input type="checkbox" aria-label="Mute" checked={muted === true} onChange={(event) => void propertyActuate(device, "muted", event.currentTarget.checked).then(refresh)} /></label></CardContent></Card>; }
function PropertyPanel({ device, state, refresh }: PanelProps) { const values = state.properties ?? {}; const descriptors: PropertyDescriptor[] = device.properties ?? Object.keys(values).map((name) => ({ name, value_type: typeof values[name]?.value, writable: true })); return <Card data-testid={selectors.pages.devicePropertyPanel}><CardHeader><CardTitle>Properties</CardTitle></CardHeader><CardContent className="flex flex-col gap-2">{descriptors.map((descriptor) => { const property = values[descriptor.name]; const value = property?.value; const change = (next: unknown) => void propertyActuate(device, descriptor.name, next).then(refresh); if (descriptor.enumeration?.length) return <label key={descriptor.name}>{descriptor.name}<select aria-label={descriptor.name} defaultValue={String(value ?? "")} onChange={(event) => change(event.currentTarget.value)}>{descriptor.enumeration.map((option) => <option key={option}>{option}</option>)}</select></label>; if (descriptor.value_type === "boolean") return <label key={descriptor.name}>{descriptor.name}<input aria-label={descriptor.name} type="checkbox" checked={value === true} disabled={!descriptor.writable} onChange={(event) => change(event.currentTarget.checked)} /></label>; if (descriptor.value_type === "number" && descriptor.minimum !== undefined && descriptor.maximum !== undefined) return <label key={descriptor.name}>{descriptor.name}<input aria-label={descriptor.name} type="range" min={descriptor.minimum} max={descriptor.maximum} value={typeof value === "number" ? value : descriptor.minimum} disabled={!descriptor.writable} onChange={(event) => change(Number(event.currentTarget.value))} /></label>; return <label key={descriptor.name}>{descriptor.name}<input aria-label={descriptor.name} type="text" defaultValue={String(value ?? "")} disabled={!descriptor.writable} onBlur={(event) => change(event.currentTarget.value)} /></label>; })}</CardContent></Card>; }
function LiveViewPanel({ state }: PanelProps) { return <Card data-testid={selectors.pages.deviceLiveView}><CardHeader><CardTitle>Live view</CardTitle></CardHeader><CardContent><p>Polled observation · refresh every 2 seconds</p><pre className="mt-2 overflow-auto rounded-md border p-2">{JSON.stringify(state, null, 2)}</pre></CardContent></Card>; }
function SensorPanel({ state }: PanelProps) { return <Card data-testid={selectors.pages.deviceSensorPanel}><CardHeader><CardTitle>Sensors</CardTitle></CardHeader><CardContent>{Object.entries(state.properties ?? {}).map(([name, value]) => <p key={name}>{name}: {String(value.value)}</p>)}</CardContent></Card>; }
function LogPanel({ state }: PanelProps) { return <Card data-testid={selectors.pages.deviceLogPanel}><CardHeader><CardTitle>Device logs</CardTitle></CardHeader><CardContent><pre>{JSON.stringify(state.unavailable ?? {}, null, 2)}</pre></CardContent></Card>; }

function Panels({ device, state, refresh }: PanelProps) {
  const showLive = available(device, "screenshot") && available(device, "input");
  const showRemote = available(device, "input") && !showLive;
  return <div className="grid gap-4 lg:grid-cols-2">{showLive && <LiveViewPanel device={device} state={state} refresh={refresh} />}{showRemote && <RemotePanel device={device} state={state} refresh={refresh} />}{available(device, "media") && <MediaPanel device={device} state={state} refresh={refresh} />}{available(device, "property") && <PropertyPanel device={device} state={state} refresh={refresh} />}{available(device, "sensor") && <SensorPanel device={device} state={state} refresh={refresh} />}{available(device, "device-logs") && <LogPanel device={device} state={state} refresh={refresh} />}</div>;
}

export function DeviceDetailPage() {
  const { deviceId = "" } = useParams();
  const [params] = useSearchParams();
  const [device, setDevice] = useState<Device>();
  const [state, setState] = useState<DeviceState>({});
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [session, setSession] = useState<Session>();
  const fixture = params.get("fixture");
  const load = useCallback(async () => {
    if (fixture === "loading") { setLoading(true); return; }
    if (fixture === "request-error") { setLoading(false); setError("Device request failed"); return; }
    setLoading(true);
    try {
      const result = await describeDevice(deviceId);
      const nextDevice = { ...result.device };
      if (fixture === "empty") nextDevice.capabilities = [];
      if (fixture === "unreachable") { nextDevice.status = "unreachable"; nextDevice.health = "unreachable"; nextDevice.health_reason = "Host node is offline."; }
      setDevice(nextDevice);
      setState(await readDeviceState(deviceId));
      const sessions = await listSessions().catch(() => ({ sessions: [] as Session[] }));
      setSession(sessions.sessions.find((item) => item.device_id === deviceId && item.state === "held"));
      setError("");
    } catch (cause) { setError(cause instanceof Error ? cause.message : "Device request failed"); } finally { setLoading(false); }
  }, [deviceId, fixture]);
  useEffect(() => { void load(); }, [load]);
  useEffect(() => watchDeviceEvents(deviceId, (event) => setState((current) => ({ ...current, properties: { ...current.properties, [event.attribute]: { value: event.new_value, status: "available", transport: "event" } } }))), [deviceId]);
  useEffect(() => { if (!device || !available(device, "screenshot") || fixture === "loading") return undefined; const timer = window.setInterval(() => void load(), 2000); return () => window.clearInterval(timer); }, [device, fixture, load]);
  const rows = useMemo(() => device ? capabilities(device) : [], [device]);
  if (loading) return <section data-testid={selectors.pages.deviceDetail} className="p-6" role="status">Loading device…</section>;
  if (error || !device) return <section data-testid={selectors.pages.deviceDetail} className="p-6"><p role="alert">{error || "Device not found"}</p></section>;
  const unreachable = device.health === "unreachable" || device.status === "unreachable";
  return <section data-testid={selectors.pages.deviceDetail} className="flex flex-col gap-4" aria-labelledby="device-detail-heading">
    <div><h2 id="device-detail-heading" className="text-2xl font-semibold">{device.name || device.id}</h2><p className="text-app-muted-foreground">{device.model || device.serial || device.id}</p></div>
    {unreachable && <p data-testid={selectors.pages.deviceUnreachableReason} role="alert">{device.health_reason || "The device is unreachable."}</p>}
    <Card data-testid={selectors.pages.deviceCapabilityMatrix}><CardHeader><CardTitle>Capabilities</CardTitle></CardHeader><CardContent><table className="w-full text-left"><tbody>{rows.map((capability) => <tr key={`${capability.name}-${capability.status}`}><td>{capability.name}</td><td data-testid={selectors.pages.capabilityDisposition}>{capability.status}</td><td data-testid={selectors.pages.capabilityReason}>{capability.status === "available" ? "Ready" : capability.prerequisite || capability.reason || capability.next_action || "Unavailable"}{capability.status === "unsupported" && <span data-testid={selectors.pages.unsupportedReason} className="sr-only">{capability.reason || "Unsupported"}</span>}</td></tr>)}</tbody></table></CardContent></Card>
    <Card data-testid={selectors.pages.deviceStrategySelection}><CardHeader><CardTitle>{device.strategy_id || "Selected strategy"}</CardTitle></CardHeader><CardContent data-testid={selectors.pages.deviceStrategyRationale}>Capabilities determine the control panels; transport health and prerequisites are shown below.</CardContent></Card>
    <Card data-testid={selectors.pages.deviceIdentityClaims}><CardHeader><CardTitle>Identity claims</CardTitle></CardHeader><CardContent>{device.identity_reason && <p data-testid={selectors.pages.deviceIdentityReason}>{device.identity_reason}</p>}{device.claims?.length ? <ul>{device.claims.map((claim) => <li key={`${claim.kind}-${claim.value}-${claim.evidence}`}>{claim.kind}={claim.value} · {claim.evidence} · {claim.strategy_id}</li>)}</ul> : <p>No hardware identity claim has been observed.</p>}</CardContent></Card>
    <div data-testid={selectors.pages.deviceTransport} className="flex flex-col gap-2"><p className="rounded-md border p-3 text-sm">Host node: {device.host_node_id || "host node unavailable"}</p>{(device.transports ?? [{ strategy_id: device.strategy_id, name: device.transport || "default", endpoint: device.endpoint, health: device.health || device.status, health_reason: device.health_reason, capabilities: Object.fromEntries(rows.map((item) => [item.name, item])) }]).map((transport) => <div key={`${transport.strategy_id}-${transport.name}`} className="rounded-md border p-3"><p>{transport.strategy_id} · {transport.name}</p><p className="text-sm">{transport.endpoint || "endpoint unavailable"} · {transport.health || "unknown"}</p></div>)}</div>
    {fixture === "stale" || fixture === "probing" ? <p role="status">Showing the last known snapshot while a probe is in flight.</p> : null}
    <Card data-testid={selectors.pages.deviceSessionHistory}><CardHeader><CardTitle>Sessions</CardTitle></CardHeader><CardContent>{session ? `Active lease held by ${session.actor} until ${session.expires_at}` : "No active session"}</CardContent></Card>
    <p data-testid="lease-bar" role="status">{session ? `Live session: ${session.actor}` : "No live session"}</p>
    {!unreachable && <Button data-testid={selectors.pages.deviceProbeNow} onClick={() => void load()}>Probe now</Button>}
    {!unreachable && <Panels device={device} state={state} refresh={load} />}
  </section>;
}
