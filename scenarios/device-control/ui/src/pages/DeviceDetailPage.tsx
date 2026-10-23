/* eslint-disable @typescript-eslint/no-misused-promises, no-restricted-syntax, @typescript-eslint/no-base-to-string */
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useParams, useSearchParams } from "react-router-dom";
import { actuateDevice, acquireSession, describeDevice, listSessions, readDeviceState, releaseSession, reconnectDevice, runFlow, evidenceUrl, watchDeviceEvents, type Capability, type Device, type DeviceState, type PropertyDescriptor, type Session } from "../api/deviceControl";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { selectors } from "../consts/selectors";

type Action = { key?: string; text?: string; media?: string; property?: string; value?: unknown };
type CapturedFrame = { id: string; created_at: string };
type PanelProps = { device: Device; state: DeviceState; refresh: () => Promise<void>; act: (action: Action, capability: string) => Promise<void>; capture: () => Promise<CapturedFrame> };

function capabilities(device: Device): Capability[] {
  const fromTransports = (device.transports ?? []).flatMap((transport) => Object.values(transport.capabilities ?? {}));
  return [...device.capabilities, ...fromTransports].filter((item, index, all) => all.findIndex((candidate) => candidate.name === item.name && candidate.status === item.status) === index);
}
function transportFor(device: Device, name: string): string | undefined {
  if (device.transports?.length) return device.transports.find((transport) => transport.health === "available" && transport.capabilities?.[name]?.status === "available")?.name;
  return device.status === "available" && (!device.health || device.health === "available") && device.capabilities.some((item) => item.name === name && item.status === "available") ? device.transport || "default" : undefined;
}
function available(device: Device, name: string) { return transportFor(device, name) !== undefined; }

function RemotePanel({ act }: PanelProps) {
  const keys = ["DPAD_UP", "DPAD_LEFT", "DPAD_CENTER", "DPAD_RIGHT", "DPAD_DOWN", "BACK", "HOME"];
  return <Card data-testid={selectors.pages.deviceRemotePanel}><CardHeader><CardTitle>Directional remote</CardTitle></CardHeader><CardContent className="flex flex-col gap-3"><div className="grid grid-cols-3 gap-2">{keys.map((key) => <Button key={key} onClick={() => act({ key }, "input")}>{key}</Button>)}</div><input aria-label="Remote text" className="rounded-md border p-2" placeholder="Type text" onKeyDown={(event) => { if (event.key === "Enter") void act({ text: event.currentTarget.value }, "input"); }} /></CardContent></Card>;
}
function propertyLabel(name: string, domain?: string) { return domain ? `${name} (${domain})` : name; }
function MediaPanel({ state, act }: PanelProps) { const currentProperty = state.properties?.volume; const current = currentProperty?.value; const muted = state.properties?.muted?.value; const receiverOnly = currentProperty?.state_domain === "receiver_volume"; const conflict = state.conflicts?.volume; return <Card data-testid={selectors.pages.deviceMediaPanel}><CardHeader><CardTitle>Media transport</CardTitle></CardHeader><CardContent className="flex flex-wrap gap-2"><div data-testid={selectors.pages.deviceNowPlaying} className="basis-full">Now playing: {String(state.properties?.application?.value ?? "unknown")} · {String(state.properties?.player_state?.value ?? "unknown")}</div>{receiverOnly && <p className="basis-full text-sm">Receiver volume is reported; physical output volume is unavailable.</p>}{conflict && <p className="basis-full text-sm">Volume transports disagree: {conflict}</p>}{["play", "pause", "stop", "previous", "next"].map((action) => <Button key={action} onClick={() => act({ media: action }, "media")}>{action}</Button>)}<label className="flex items-center gap-2">{receiverOnly ? "Receiver volume" : "Volume"}<input type="range" min="0" max="1" step="0.01" aria-label={receiverOnly ? "Receiver volume" : "Volume"} value={typeof current === "number" ? current : 0} onChange={(event) => void act({ property: "volume", value: Number(event.currentTarget.value) }, "media")} /></label><label className="flex items-center gap-2">{receiverOnly ? "Receiver mute" : "Mute"}<input type="checkbox" aria-label={receiverOnly ? "Receiver mute" : "Mute"} checked={muted === true} onChange={(event) => void act({ property: "muted", value: event.currentTarget.checked }, "media")} /></label></CardContent></Card>; }
function PropertyPanel({ device, state, act }: PanelProps) { const values = state.properties ?? {}; const descriptors: PropertyDescriptor[] = device.properties ?? Object.keys(values).map((name) => ({ name, value_type: typeof values[name]?.value, writable: false })); return <Card data-testid={selectors.pages.devicePropertyPanel}><CardHeader><CardTitle>Properties</CardTitle></CardHeader><CardContent className="flex flex-col gap-2">{descriptors.map((descriptor) => { const property = values[descriptor.name]; const value = property?.value; const label = propertyLabel(descriptor.name, descriptor.state_domain || property?.state_domain); const change = (next: unknown) => void act({ property: descriptor.name, value: next }, "property"); if (descriptor.enumeration?.length) return <label key={descriptor.name}>{label}<select aria-label={descriptor.name} disabled={!descriptor.writable} defaultValue={String(value ?? "")} onChange={(event) => change(event.currentTarget.value)}>{descriptor.enumeration.map((option) => <option key={option}>{option}</option>)}</select></label>; if (descriptor.value_type === "boolean") return <label key={descriptor.name}>{label}<input aria-label={descriptor.name} type="checkbox" checked={value === true} disabled={!descriptor.writable} onChange={(event) => change(event.currentTarget.checked)} /></label>; if (descriptor.value_type === "number" && descriptor.minimum !== undefined && descriptor.maximum !== undefined) return <label key={descriptor.name}>{label}<input aria-label={descriptor.name} type="range" min={descriptor.minimum} max={descriptor.maximum} value={typeof value === "number" ? value : descriptor.minimum} disabled={!descriptor.writable} onChange={(event) => change(Number(event.currentTarget.value))} /></label>; return <label key={descriptor.name}>{label}<input aria-label={descriptor.name} type="text" defaultValue={String(value ?? "")} disabled={!descriptor.writable} onBlur={(event) => change(event.currentTarget.value)} /></label>; })}</CardContent></Card>; }
function LiveViewPanel({ capture }: PanelProps) {
  const [frame, setFrame] = useState<CapturedFrame>();
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState("");
  const observe = async () => {
    setBusy(true); setFailure("");
    try { setFrame(await capture()); } catch (cause) { setFailure(cause instanceof Error ? cause.message : "Capture failed"); }
    finally { setBusy(false); }
  };
  return <Card data-testid={selectors.pages.deviceLiveView}><CardHeader><CardTitle>Device screen</CardTitle></CardHeader><CardContent className="space-y-3"><p>Capture a redacted frame to verify what is visible.</p><Button disabled={busy} onClick={observe}>{busy ? "Capturing…" : "Capture screen"}</Button>{failure && <p role="alert">{failure}</p>}{frame && <figure><img className="w-full rounded-md" src={evidenceUrl(frame.id)} alt="Captured device screen" /><figcaption>Captured {new Date(frame.created_at).toLocaleTimeString()} · still image</figcaption></figure>}</CardContent></Card>;
}
function SensorPanel({ state }: PanelProps) { return <Card data-testid={selectors.pages.deviceSensorPanel}><CardHeader><CardTitle>Sensors</CardTitle></CardHeader><CardContent>{Object.entries(state.properties ?? {}).map(([name, value]) => <p key={name}>{name}: {String(value.value)}</p>)}</CardContent></Card>; }
function LogPanel({ state }: PanelProps) { return <Card data-testid={selectors.pages.deviceLogPanel}><CardHeader><CardTitle>Capability diagnostics</CardTitle></CardHeader><CardContent><pre>{JSON.stringify(state.unavailable ?? {}, null, 2)}</pre></CardContent></Card>; }

function Panels(props: PanelProps) {
  const { device } = props;
  return <div className="grid gap-4 lg:grid-cols-2">{available(device, "screenshot") && <LiveViewPanel {...props} />}{available(device, "input") && <RemotePanel {...props} />}{available(device, "media") && <MediaPanel {...props} />}{available(device, "property") && <PropertyPanel {...props} />}{available(device, "sensor") && <SensorPanel {...props} />}{available(device, "device-logs") && <LogPanel {...props} />}</div>;
}

export function DeviceDetailPage() {
  const { deviceId = "" } = useParams();
  return <DeviceDetail key={deviceId} deviceId={deviceId} />;
}

function DeviceDetail({ deviceId }: { deviceId: string }) {
  const [params] = useSearchParams();
  const [device, setDevice] = useState<Device>();
  const [state, setState] = useState<DeviceState>({});
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [session, setSession] = useState<Session>();
  const fixture = params.get("fixture");
  const alive = useRef(true);
  const isActive = () => alive.current;
  const ownedLease = useRef<Session>();
  const acquiring = useRef<Promise<Session>>();
  const refreshInFlight = useRef(false);
  const actionQueue = useRef<Promise<void>>(Promise.resolve());
  const [actionError, setActionError] = useState("");
  const [pendingActions, setPendingActions] = useState(0);
  const [reconnecting, setReconnecting] = useState(false);
  const getLease = async () => {
    if (ownedLease.current && Date.parse(ownedLease.current.expires_at) > Date.now()) return ownedLease.current;
    if (!acquiring.current) acquiring.current = acquireSession(deviceId, "browser-operator").then(({ session: acquired }) => { ownedLease.current = acquired; setSession(acquired); return acquired; }).finally(() => { acquiring.current = undefined; });
    return acquiring.current;
  };
  const endControl = () => {
    const operation = actionQueue.current.then(async () => {
      const lease = ownedLease.current;
      if (!lease) return;
      await releaseSession(lease.id); ownedLease.current = undefined; setSession(undefined); setActionError("");
    }).catch((cause: unknown) => setActionError(cause instanceof Error ? cause.message : "Release failed"));
    actionQueue.current = operation;
    return operation;
  };
  useEffect(() => {
    alive.current = true;
    return () => {
      alive.current = false;
      const release = async () => { await actionQueue.current; const lease = ownedLease.current ?? await acquiring.current; if (lease) await releaseSession(lease.id); };
      void release().catch(() => undefined);
    };
  }, [deviceId]);
  const load = useCallback(async () => {
    if (fixture === "loading") { setLoading(true); return; }
    if (fixture === "request-error") { setLoading(false); setError("Device request failed"); return; }
    if (refreshInFlight.current) return;
    refreshInFlight.current = true;
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
    } catch (cause) { setError(cause instanceof Error ? cause.message : "Device request failed"); } finally { refreshInFlight.current = false; setLoading(false); }
  }, [deviceId, fixture]);
  useEffect(() => { void load(); }, [load]);
  useEffect(() => watchDeviceEvents(deviceId, (event) => setState((current) => ({ ...current, properties: { ...current.properties, [event.attribute]: { value: event.new_value, status: "available", transport: "event" } } }))), [deviceId]);
  const hasDevice = Boolean(device);
  useEffect(() => {
    if (!hasDevice || fixture === "loading") return undefined;
    let active = true;
    let timer: ReturnType<typeof setTimeout>;
    const poll = async () => {
      try { const next = await readDeviceState(deviceId); if (active) setState(next); }
      catch (cause) { if (active) setActionError(cause instanceof Error ? cause.message : "State refresh failed"); }
      finally { if (active) timer = setTimeout(() => void poll(), 2000); }
    };
    timer = setTimeout(() => void poll(), 2000);
    return () => { active = false; clearTimeout(timer); };
  }, [hasDevice, deviceId, fixture]);
  const act = (action: Action, capability: string) => {
    setPendingActions((count) => count + 1);
    const operation = actionQueue.current.then(async () => {
      if (!device || !isActive()) return;
      const transport = transportFor(device, capability);
      if (!transport) throw new Error("No healthy transport supports this control");
      const lease = await getLease();
      if (!isActive()) return;
      await actuateDevice(deviceId, { ...action, actor: "browser-operator", lease_token: lease.lease_token ?? "", transport: transport === "default" ? undefined : transport });
      setState(await readDeviceState(deviceId));
      setActionError("");
    }).catch((cause: unknown) => setActionError(cause instanceof Error ? cause.message : "Control failed"))
      .finally(() => setPendingActions((count) => count - 1));
    actionQueue.current = operation;
    return operation;
  };
  const capture = () => {
    const operation = actionQueue.current.then(async () => {
      if (!isActive()) throw new Error("Device view closed");
      const lease = await getLease();
      const transport = device && transportFor(device, "screenshot");
      if (!transport) throw new Error("No healthy transport supports capture");
      if (!isActive()) throw new Error("Device view closed");
      const result = await runFlow(deviceId, "browser-operator", lease.lease_token ?? "", { id: "operator-frame", transport: transport === "default" ? undefined : transport, max_duration_ms: 15000, require_unlocked: true, steps: [{ id: "screen", kind: "observe" }] });
      if (result.disposition !== "passed" || !result.evidence?.[0]) throw new Error(result.chapters?.find((chapter) => chapter.disposition !== "passed")?.message ?? "No verified frame returned");
      return result.evidence[0];
    });
    actionQueue.current = operation.then(() => undefined, () => undefined);
    return operation;
  };
  const rows = useMemo(() => device ? capabilities(device) : [], [device]);
  if (loading) return <section data-testid={selectors.pages.deviceDetail} className="p-6" role="status">Loading device…</section>;
  if (error || !device) return <section data-testid={selectors.pages.deviceDetail} className="p-6"><p role="alert">{error || "Device not found"}</p></section>;
  const unreachable = device.health === "unreachable" || device.status === "unreachable";
  return <section data-testid={selectors.pages.deviceDetail} className="flex flex-col gap-4" aria-labelledby="device-detail-heading">
    <div><h2 id="device-detail-heading" className="text-2xl font-semibold">{device.name || device.id}</h2><p className="text-app-muted-foreground">{device.model || device.serial || device.id}</p></div>
    {unreachable && <p data-testid={selectors.pages.deviceUnreachableReason} role="alert">{device.health_reason || "The device is unreachable."}</p>}
    <Card data-testid={selectors.pages.deviceSessionHistory}><CardHeader><CardTitle>Sessions</CardTitle></CardHeader><CardContent>{session ? `Active lease held by ${session.actor} until ${session.expires_at}` : "No active session"}</CardContent></Card>
    <p data-testid="lease-bar" role="status">{session ? `Live session: ${session.actor}` : "No live session"}</p>
    {(device.transport === "wireless" || device.transports?.some((transport) => transport.name === "wireless")) && <Button disabled={reconnecting} onClick={async () => { setReconnecting(true); try { const result = await reconnectDevice(deviceId); setDevice(result.device); setState(await readDeviceState(deviceId)); setActionError(""); } catch (cause) { setActionError(cause instanceof Error ? cause.message : "Reconnect failed"); } finally { setReconnecting(false); } }}>{reconnecting ? "Reconnecting…" : "Refresh connection"}</Button>}
    {!unreachable && <Button data-testid={selectors.pages.deviceProbeNow} onClick={() => void load()}>Refresh state</Button>}
    {actionError && <p role="alert">{actionError}</p>}
    {pendingActions > 0 && <p role="status">Sending control…</p>}
    {session?.id === ownedLease.current?.id && session && <Button onClick={endControl}>End control session</Button>}
    {!unreachable && <Panels device={device} state={state} refresh={load} act={act} capture={capture} />}
    <Card data-testid={selectors.pages.deviceCapabilityMatrix}><CardHeader><CardTitle>Capabilities</CardTitle></CardHeader><CardContent><table className="w-full text-left"><tbody>{rows.map((capability) => <tr key={`${capability.name}-${capability.status}`}><td>{capability.name}</td><td data-testid={selectors.pages.capabilityDisposition}>{capability.status}</td><td data-testid={selectors.pages.capabilityReason}>{capability.status === "available" ? "Ready" : capability.prerequisite || capability.reason || capability.next_action || "Unavailable"}{capability.status === "unsupported" && <span data-testid={selectors.pages.unsupportedReason} className="sr-only">{capability.reason || "Unsupported"}</span>}</td></tr>)}</tbody></table></CardContent></Card>
    <Card data-testid={selectors.pages.deviceStrategySelection}><CardHeader><CardTitle>{device.strategy_id || "Selected strategy"}</CardTitle></CardHeader><CardContent data-testid={selectors.pages.deviceStrategyRationale}>Capabilities determine the control panels; transport health and prerequisites are shown below.</CardContent></Card>
    <Card data-testid={selectors.pages.deviceIdentityClaims}><CardHeader><CardTitle>Identity claims</CardTitle></CardHeader><CardContent>{device.identity_reason && <p data-testid={selectors.pages.deviceIdentityReason}>{device.identity_reason}</p>}{device.claims?.length ? <ul>{device.claims.map((claim) => <li key={`${claim.kind}-${claim.value}-${claim.evidence}`}>{claim.kind}={claim.value} · {claim.evidence} · {claim.strategy_id}</li>)}</ul> : <p>No hardware identity claim has been observed.</p>}</CardContent></Card>
    <div data-testid={selectors.pages.deviceTransport} className="flex flex-col gap-2"><p className="rounded-md border p-3 text-sm">Host node: {device.host_node_id || "host node unavailable"}</p>{(device.transports ?? [{ strategy_id: device.strategy_id, name: device.transport || "default", endpoint: device.endpoint, health: device.health || device.status, health_reason: device.health_reason, capabilities: Object.fromEntries(rows.map((item) => [item.name, item])) }]).map((transport) => <div key={`${transport.strategy_id}-${transport.name}`} className="rounded-md border p-3"><p>{transport.strategy_id} · {transport.name}</p><p className="text-sm">{transport.endpoint || "endpoint unavailable"} · {transport.health || "unknown"}</p></div>)}</div>
    {fixture === "stale" || fixture === "probing" ? <p role="status">Showing the last known snapshot while a probe is in flight.</p> : null}

  </section>;
}
