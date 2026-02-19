import { TunnelStatusPanel } from "./components/TunnelStatus";
import { RouteTable } from "./components/RouteTable";
import { ProbeResults } from "./components/ProbeResults";
import { AuditView } from "./components/AuditView";

export default function App() {
  return (
    <div className="min-h-screen bg-slate-950 text-slate-50">
      <header className="border-b border-white/10 bg-white/5 px-6 py-4">
        <h1 className="text-xl font-semibold">Tunnel Manager</h1>
        <p className="text-sm text-slate-400">Cloudflare tunnel monitoring and management</p>
      </header>

      <main className="mx-auto max-w-6xl space-y-6 p-6">
        <TunnelStatusPanel />
        <RouteTable />
        <div className="grid gap-6 lg:grid-cols-2">
          <ProbeResults />
          <AuditView />
        </div>
      </main>
    </div>
  );
}
