import { useMemo, useState } from "react";
import { ArrowDownWideNarrow, ArrowUpNarrowWide, Search } from "lucide-react";
import { Button } from "../components/ui/button";
import { Skeleton } from "../components/ui/LoadingStates";
import { HelpDialog } from "../components/ui/HelpDialog";

interface ResourceStatus {
  resource_name: string;
  secrets_total: number;
  secrets_found: number;
  secrets_missing: number;
  secrets_optional: number;
  health_status: string;
  last_checked: string;
}

interface ResourceTableProps {
  resourceStatuses: ResourceStatus[];
  isLoading: boolean;
  onOpenResource: (resourceName: string) => void;
}

type SortKey = "name" | "missing" | "total";

export const ResourceTable = ({ resourceStatuses, isLoading, onOpenResource }: ResourceTableProps) => {
  const [search, setSearch] = useState("");
  const [sortKey, setSortKey] = useState<SortKey>("missing");
  const [sortDir, setSortDir] = useState<"asc" | "desc">("desc");
  const [showActionableOnly, setShowActionableOnly] = useState(false);

  const filtered = useMemo(() => {
    const term = search.toLowerCase();
    return resourceStatuses
      .filter((status) => {
        if (!status.resource_name.toLowerCase().includes(term)) return false;
        if (showActionableOnly && status.secrets_missing === 0) return false;
        return true;
      })
      .sort((a, b) => {
        const dir = sortDir === "asc" ? 1 : -1;
        if (sortKey === "name") {
          return dir * a.resource_name.localeCompare(b.resource_name);
        }
        if (sortKey === "missing") {
          return dir * (a.secrets_missing - b.secrets_missing);
        }
        return dir * (a.secrets_total - b.secrets_total);
      });
  }, [resourceStatuses, search, sortDir, sortKey, showActionableOnly]);

  const toggleSort = (key: SortKey) => {
    if (sortKey === key) {
      setSortDir((dir) => (dir === "asc" ? "desc" : "asc"));
    } else {
      setSortKey(key);
      setSortDir("desc");
    }
  };

  const renderSortIcon = (key: SortKey) => {
    if (sortKey !== key) return <ArrowDownWideNarrow className="h-4 w-4 text-white/40" />;
    return sortDir === "asc" ? (
      <ArrowUpNarrowWide className="h-4 w-4 text-white/40" />
    ) : (
      <ArrowDownWideNarrow className="h-4 w-4 text-white/40" />
    );
  };

  return (
    <section id="anchor-resources" className="rounded-3xl border border-white/10 bg-white/5 p-6">
      <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
        <div>
          <p className="text-xs uppercase tracking-[0.3em] text-emerald-300">Per Resource</p>
          <div className="flex items-center gap-2">
            <p className="text-2xl font-semibold text-white">All resources, searchable and sortable</p>
            <HelpDialog title="How resource secrets are defined">
              <p>
                Each resource owns a <code>config/secrets.yaml</code> file that declares its secrets and the Vault paths
                they live at. The Vault CLI (<code>resource-vault secrets ...</code>) and this dashboard read that file
                to know what to validate and where to store values.
              </p>
              <p>
                To add secrets for a resource: <code>resource-vault secrets create-template &lt;resource&gt;</code>, edit
                the generated <code>config/secrets.yaml</code>, then run <code>resource-vault secrets init</code> to set
                them. secrets-manager will pick them up automatically via the Vault check/fallback scan.
              </p>
            </HelpDialog>
          </div>
          <p className="text-sm text-white/60">Includes healthy resources so you can confirm full coverage.</p>
        </div>
        <div className="flex flex-col gap-2 md:flex-row md:items-center md:gap-3">
          <div className="relative w-full max-w-xs">
            <input
              type="text"
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder="Search resources"
              className="w-full rounded-2xl border border-white/10 bg-black/30 px-3 py-2 pr-10 text-sm text-white placeholder:text-white/40"
            />
            <Search className="pointer-events-none absolute right-3 top-2.5 h-4 w-4 text-white/40" />
          </div>
          <Button variant="outline" size="sm" onClick={() => setShowActionableOnly((v) => !v)}>
            {showActionableOnly ? "Show all" : "Only action needed"}
          </Button>
        </div>
      </div>

      <div className="mt-4 overflow-hidden rounded-2xl border border-white/10">
        <table className="min-w-full divide-y divide-white/10 text-sm">
          <thead className="bg-black/30 text-white/60">
            <tr>
              <th className="px-3 py-2 text-left">
                <button className="flex items-center gap-2" onClick={() => toggleSort("name")}>
                  Resource {renderSortIcon("name")}
                </button>
              </th>
              <th className="px-3 py-2 text-left">
                <button className="flex items-center gap-2" onClick={() => toggleSort("missing")}>
                  Missing {renderSortIcon("missing")}
                </button>
              </th>
              <th className="px-3 py-2 text-left">
                <button className="flex items-center gap-2" onClick={() => toggleSort("total")}>
                  Total secrets {renderSortIcon("total")}
                </button>
              </th>
              <th className="px-3 py-2 text-left">Health</th>
              <th className="px-3 py-2 text-left">Last checked</th>
              <th className="px-3 py-2 text-left">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-white/10">
            {isLoading ? (
              <tr>
                <td colSpan={6} className="px-3 py-4">
                  <Skeleton className="h-10 w-full" variant="text" />
                </td>
              </tr>
            ) : filtered.length === 0 ? (
              <tr>
                <td colSpan={6} className="px-3 py-4 text-center text-sm text-white/60">
                  No resources found. If you expect more entries, ensure the API returns resources with zero secrets.
                </td>
              </tr>
            ) : (
              filtered.map((status) => {
                const healthy = status.secrets_missing === 0;
                return (
                  <tr key={status.resource_name} className="bg-black/20">
                    <td className="px-3 py-2 font-semibold text-white">{status.resource_name}</td>
                    <td className="px-3 py-2 text-amber-100">{status.secrets_missing}</td>
                    <td className="px-3 py-2 text-white/80">
                      {status.secrets_found}/{status.secrets_total}
                    </td>
                    <td className="px-3 py-2">
                      <span
                        className={`rounded-full border px-2 py-0.5 text-[11px] uppercase tracking-[0.2em] ${
                          healthy ? "border-emerald-400/50 text-emerald-100" : "border-amber-400/50 text-amber-100"
                        }`}
                      >
                        {healthy ? "Healthy" : "Action needed"}
                      </span>
                    </td>
                    <td className="px-3 py-2 text-xs text-white/60">
                      {status.last_checked ? new Date(status.last_checked).toLocaleString() : "—"}
                    </td>
                    <td className="px-3 py-2">
                      <Button variant="outline" size="sm" onClick={() => onOpenResource(status.resource_name)}>
                        Open
                      </Button>
                    </td>
                  </tr>
                );
              })
            )}
          </tbody>
        </table>
      </div>
    </section>
  );
};
