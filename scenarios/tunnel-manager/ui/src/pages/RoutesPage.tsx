import { RouteTable } from "../components/RouteTable";
import { RouteManagement } from "../components/RouteForm";

export default function RoutesPage() {
  return (
    <div className="space-y-4 sm:space-y-6">
      <div>
        <h1 className="text-lg sm:text-xl font-semibold">Route Management</h1>
        <p className="text-sm text-slate-300">Configure subdomain-to-port mappings for the Cloudflare tunnel.</p>
      </div>
      <RouteTable />
      <RouteManagement />
    </div>
  );
}
