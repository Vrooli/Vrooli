/* eslint-disable no-restricted-syntax, @typescript-eslint/use-unknown-in-catch-callback-variable */
import { useEffect, useState } from "react";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { HealthCard } from "../features/health/HealthCard";
import { useTranslation } from "../i18n";
import { killSession, listDevices, listSessions, listStrategies, type Device, type Session, type Strategy } from "../api/deviceControl";

/**
 * Dashboard / home page. Composes the health card plus stat placeholders.
 * Replace the cards with real surfaces when the scenario grows them.
 */
export function DashboardPage() {
  const { t } = useTranslation();
  const [devices, setDevices] = useState<Device[]>([]); const [strategies, setStrategies] = useState<Strategy[]>([]); const [sessions, setSessions] = useState<Session[]>([]); const [error, setError] = useState("");
  const refresh = () => Promise.all([listDevices(), listStrategies(), listSessions()]).then(([d,s,l]) => { setDevices(d.devices); setStrategies(s.strategies); setSessions(l.sessions); setError(""); }).catch((e: Error) => setError(e.message));
  useEffect(() => { let mounted = true; Promise.all([listDevices(), listStrategies(), listSessions()]).then(([d,s,l]) => { if (!mounted) return; setDevices(d.devices); setStrategies(s.strategies); setSessions(l.sessions); setError(""); }).catch((e: Error) => { if (mounted) setError(e.message); }); return () => { mounted = false; }; }, []);
  const active = devices.filter((device) => device.status === "available").length;
  const unavailable = devices.length - active;
  const stop = async (id: string) => { await killSession(id); await refresh(); };

  return (
    <section
      data-testid={selectors.pages.dashboard}
      aria-labelledby="dashboard-heading"
      className="flex flex-col gap-4"
    >
      <h2 id="dashboard-heading" className="text-2xl font-semibold">
        {t(strings.pages.dashboard.title)}
      </h2>
      <p className="text-app-muted-foreground">{t(strings.pages.dashboard.description)}</p>
      {error && <p role="alert" className="rounded-md border border-app-destructive/40 p-3 text-app-destructive">{error}</p>}
      <div className="grid gap-4 md:grid-cols-3"><HealthCard /><Metric label="Devices available" value={`${active}/${devices.length}`} /><Metric label="Unavailable prerequisites" value={String(unavailable)} /><Metric label="Live leases" value={String(sessions.length)} /></div>
      <div className="grid gap-4 xl:grid-cols-2"><Card><CardHeader><CardTitle>Fleet capability snapshot</CardTitle></CardHeader><CardContent><div className="flex flex-col gap-3">{devices.map((device) => <div key={device.id} className="rounded-md border p-3"><div className="flex items-center justify-between gap-3"><div><p className="font-medium">{device.id}</p><p className="text-sm text-app-muted-foreground">{device.name}</p></div><span className="rounded-full border px-2 py-1 text-xs">{device.status}</span></div><p className="mt-2 text-sm text-app-muted-foreground">{device.health_reason || "Probe passed; no health warning."}</p><div className="mt-2 flex flex-wrap gap-2">{device.capabilities.map((cap) => <span key={cap.name} className="rounded border px-2 py-1 text-xs">{cap.name}: {cap.status}</span>)}</div></div>)}{devices.length===0&&<p className="text-app-muted-foreground">No strategies have been probed yet.</p>}</div></CardContent></Card><Card><CardHeader><CardTitle>Strategy conformance matrix</CardTitle></CardHeader><CardContent><div className="overflow-x-auto"><table className="w-full text-left text-sm"><thead><tr><th className="p-2">Strategy</th><th className="p-2">Status</th><th className="p-2">Tiers</th><th className="p-2">Promotable</th></tr></thead><tbody>{strategies.map((item) => <tr key={item.id} className="border-t"><td className="p-2 font-medium">{item.id}</td><td className="p-2">{item.status}</td><td className="p-2">{item.tiers.join(", ") || "—"}</td><td className="p-2">{item.promotable ? "yes" : "no"}</td></tr>)}</tbody></table></div></CardContent></Card></div>
      <Card><CardHeader><CardTitle>Live session controls</CardTitle></CardHeader><CardContent>{sessions.length===0?<p className="text-app-muted-foreground">No live sessions. A lease is required before any device verb.</p>:<div className="flex flex-col gap-3">{sessions.map((session)=><div key={session.id} className="flex flex-wrap items-center justify-between gap-3 rounded-md border p-3"><div><p className="font-medium">{session.device_id} · {session.actor}</p><p className="text-sm text-app-muted-foreground">Lease expires {session.expires_at}</p></div><Button onClick={()=>void stop(session.id)} aria-label={`Kill session ${session.id}`}>Kill immediately</Button></div>)}</div>}</CardContent></Card>
    </section>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm uppercase text-app-muted-foreground">{label}</CardTitle>
      </CardHeader>
      <CardContent>
        <p className="text-2xl font-semibold">{value}</p>
      </CardContent>
    </Card>
  );
}
