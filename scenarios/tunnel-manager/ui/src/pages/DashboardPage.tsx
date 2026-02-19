import { TunnelStatusPanel } from "../components/TunnelStatus";
import { RouteTable } from "../components/RouteTable";
import { ProbeResults } from "../components/ProbeResults";
import { AuditView } from "../components/AuditView";
import { RecoveryTimeline } from "../components/RecoveryTimeline";

export default function DashboardPage() {
  return (
    <div className="space-y-4 sm:space-y-6">
      <h1 className="sr-only">Dashboard Overview</h1>
      <TunnelStatusPanel />
      <RouteTable />
      <div className="grid gap-4 sm:gap-6 lg:grid-cols-2">
        <ProbeResults />
        <AuditView />
      </div>
      <RecoveryTimeline />
    </div>
  );
}
