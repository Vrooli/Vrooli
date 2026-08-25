import { useMemo, useState } from "react";
import { useQueries } from "@tanstack/react-query";
import { AlertTriangle, ArrowDown, CheckCircle2, ChevronRight, CircleHelp, ClipboardCheck, LayoutDashboard, List, MapPin, RefreshCw } from "lucide-react";

import {
  fetchAdoption,
  fetchCensusHistory,
  fetchInfraHealth,
  fetchInventory,
  fetchPlacement,
  fetchPlacementAudit,
  fetchRetentionOwners,
  type CensusReport,
  type StorageOwner,
} from "../api/storage";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { Button } from "@vrooli/react-component-library/Button/1.2.0";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1.1.0";
import { BottomNav } from "@vrooli/react-component-library/BottomNav/1.2.0";
import { DataTable } from "@vrooli/react-component-library/DataTable/1.2.0";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1.1.0";
import { HealthCard } from "../components/HealthCard";
import { Metric } from "../components/Metric";
import { QueryError } from "../components/QueryError";
import { SectionNav } from "../components/SectionNav";

const formatBytes = (bytes: number | undefined) => {
  if (bytes === undefined || !Number.isFinite(bytes)) return "—";
  if (bytes < 1024) return `${bytes} B`;
  const units = ["KB", "MB", "GB", "TB"];
  let value = bytes;
  let unit = -1;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit += 1;
  }
  return `${value.toFixed(value >= 10 ? 0 : 1)} ${units[unit]}`;
};

const formatPercent = (value: number | undefined) =>
  value === undefined || !Number.isFinite(value) ? "—" : `${Math.round(value * 100)}%`;

const formatTime = (value: string | undefined) => {
  if (!value) return "No scan recorded";
  const date = new Date(value);
  return Number.isNaN(date.valueOf()) ? "Unknown time" : date.toLocaleString();
};

function confidenceTone(confidence: string | undefined): "neutral" | "success" | "warning" | "danger" {
  if (confidence === "high") return "success";
  if (confidence === "degraded") return "warning";
  return "neutral";
}

function ownerBytes(owner: StorageOwner, snapshot: CensusReport | undefined) {
  return (snapshot?.entries ?? [])
    .filter((entry) => entry.owner === owner.id)
    .reduce((total, entry) => total + entry.bytes, 0);
}

function ownerStatus(owner: StorageOwner, bytes: number, hasFinding: boolean) {
  if (hasFinding) return { label: "Needs review", tone: "warning" as const };
  if (owner.storage_entries?.length) return { label: "Declared", tone: "success" as const };
  if (owner.storage_declared) return { label: "No local surface", tone: "neutral" as const };
  if (bytes > 0) return { label: "Adoption gap", tone: "warning" as const };
  return { label: "Unmeasured", tone: "neutral" as const };
}

export function DashboardPage() {
  const { t } = useTranslation();
  const [activeSection, setActiveSection] = useState("overview");
  const queries = useQueries({
    queries: [
      { queryKey: ["storage", "inventory"], queryFn: fetchInventory },
      { queryKey: ["storage", "history"], queryFn: fetchCensusHistory },
      { queryKey: ["storage", "adoption"], queryFn: fetchAdoption },
      { queryKey: ["storage", "infra-health"], queryFn: fetchInfraHealth },
      { queryKey: ["storage", "placement"], queryFn: fetchPlacement },
      { queryKey: ["storage", "retention"], queryFn: fetchRetentionOwners },
      { queryKey: ["storage", "audit"], queryFn: fetchPlacementAudit },
    ],
  });
  const [inventoryQuery, historyQuery, adoptionQuery, infraQuery, placementQuery, retentionQuery, auditQuery] = queries;
  const inventory = inventoryQuery.data;
  const snapshot = infraQuery.data?.latest_snapshot ?? historyQuery.data?.[0];
  const findings = inventory?.findings ?? [];
  const isRefreshing = queries.some((query) => query.isFetching);
  const hasError = queries.some((query) => query.isError);
  const ownerRows = useMemo(() => {
    if (!inventory) return [];
    return inventory.owners
      .map((owner) => {
        const bytes = ownerBytes(owner, snapshot);
        const hasFinding = findings.some((finding) => finding.owner_id === owner.id);
        return { owner, bytes, status: ownerStatus(owner, bytes, hasFinding) };
      })
      .sort((a, b) => b.bytes - a.bytes || a.owner.id.localeCompare(b.owner.id));
  }, [findings, inventory, snapshot]);

  const refresh = () => queries.forEach((query) => void query.refetch());
  return (
    <section data-testid={selectors.pages.dashboard} aria-labelledby="dashboard-heading" className="mx-auto flex w-full max-w-7xl flex-col gap-6">
      <header className="flex flex-col gap-4 border-b border-app-border pb-5 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <p className="text-xs font-semibold uppercase tracking-[0.18em] text-app-primary">{t(strings.app.eyebrow)}</p>
          <h2 id="dashboard-heading" className="mt-2 text-3xl font-semibold tracking-tight">{t(strings.pages.dashboard.title)}</h2>
          <p className="mt-2 max-w-2xl text-sm text-app-muted-foreground">{t(strings.pages.dashboard.description)}</p>
        </div>
        <Button data-testid={selectors.console.refresh} variant="secondary" onClick={refresh} disabled={isRefreshing}>
          <RefreshCw aria-hidden="true" className={`h-4 w-4 ${isRefreshing ? "animate-spin" : ""}`} />
          {isRefreshing ? "Refreshing…" : "Refresh ledger"}
        </Button>
      </header>

      <SectionNav
        activeId={activeSection}
        items={[
          { id: "overview", label: "Overview", icon: <LayoutDashboard className="h-4 w-4" /> },
          { id: "coverage", label: "Coverage", icon: <List className="h-4 w-4" /> },
          { id: "placement", label: "Placement", icon: <MapPin className="h-4 w-4" /> },
          { id: "audit", label: "Audit", icon: <ClipboardCheck className="h-4 w-4" /> },
        ]}
        onSelect={setActiveSection}
      />
      <BottomNav
        label="Mobile storage console sections"
        testId="storage-mobile-nav"
        items={[
          { id: "overview", label: "Overview", href: "#storage-overview", icon: <LayoutDashboard className="h-4 w-4" />, active: activeSection === "overview" },
          { id: "coverage", label: "Owners", href: "#storage-coverage", icon: <List className="h-4 w-4" />, active: activeSection === "coverage" },
          { id: "placement", label: "Placement", href: "#storage-placement", icon: <MapPin className="h-4 w-4" />, active: activeSection === "placement" },
          { id: "audit", label: "Audit", href: "#storage-audit", icon: <ClipboardCheck className="h-4 w-4" />, active: activeSection === "audit" },
        ]}
        onItemSelect={(item) => setActiveSection(item.id)}
      />

      {hasError && <QueryError testId={selectors.console.error} message="Some operational surfaces are unavailable. Values below are labeled by source and may be incomplete." />}

      <div id="storage-overview" className="scroll-mt-20 space-y-4">
        <div className="flex items-center justify-between gap-3">
          <div>
            <h3 className="text-xl font-semibold">Accounting overview</h3>
            <p className="text-sm text-app-muted-foreground">{snapshot ? `Last observation: ${formatTime(snapshot.observed_at)}` : "A census has not been recorded yet."}</p>
          </div>
          <StatusBadge data-testid={selectors.console.confidence} tone={confidenceTone(snapshot?.confidence)}>
            {snapshot?.confidence ? `${snapshot.confidence} confidence` : "No confidence signal"}
          </StatusBadge>
        </div>
        {snapshot ? (
          <div data-testid={selectors.console.summary} className="grid gap-3 sm:grid-cols-2 xl:grid-cols-5">
            <Metric label="Measured" value={formatBytes(snapshot.measured_bytes)} detail="Observed in selected root" />
            <Metric label="Attributed" value={formatBytes(snapshot.attributed_bytes)} detail={formatPercent(snapshot.measured_bytes ? snapshot.attributed_bytes / snapshot.measured_bytes : undefined) + " of measured"} />
            <Metric label="Undeclared" value={snapshot.unattributed_bytes == null ? "Unknown" : formatBytes(snapshot.unattributed_bytes)} detail={snapshot.unattributed_bytes == null ? "Unreadable coverage prevents attribution" : snapshot.unattributed_bytes ? "Requires adoption review" : "Nothing outstanding"} tone={snapshot.unattributed_bytes == null || snapshot.unattributed_bytes ? "warning" : "success"} />
            <Metric label="Owners" value={String(inventory?.owners.length ?? infraQuery.data?.owner_count ?? 0)} detail={`${infraQuery.data?.owners_with_declared_ceiling ?? 0} with a declared ceiling`} />
            <Metric label="Accounting" value={snapshot.closed ? "Closed" : "Open"} detail={snapshot.accounting_identity ? "Identity verified" : "Identity unresolved"} tone={snapshot.closed ? "success" : "warning"} />
          </div>
        ) : (
          <div data-testid={selectors.console.empty}><EmptyState title="No census snapshot yet" description="The console does not infer storage totals. Run a read-only census from the CLI or API to populate this ledger." /></div>
        )}
        {snapshot && !snapshot.closed && (
          <div className="flex gap-3 rounded-panel border border-app-warning/40 bg-app-warning/10 p-4 text-sm">
            <AlertTriangle aria-hidden="true" className="mt-0.5 h-5 w-5 shrink-0 text-app-warning" />
            <p><span className="font-semibold">Accounting is open.</span> {snapshot.findings?.length ?? 0} finding(s) or unreadable path(s) prevent a closed claim. Review coverage before any reclaim decision.</p>
          </div>
        )}
      </div>

      <div id="storage-coverage" className="scroll-mt-20 grid gap-4 xl:grid-cols-[1.5fr_1fr]">
        <Card data-testid={selectors.console.ownerLedger}>
          <CardHeader>
            <CardTitle>Owner ledger</CardTitle>
            <CardDescription>Every discovered scenario, resource, tool, and safeguard remains visible, including adoption gaps.</CardDescription>
          </CardHeader>
          <CardContent>
            {inventoryQuery.isLoading ? <p className="text-sm text-app-muted-foreground">Loading owner inventory…</p> : ownerRows.length === 0 ? <EmptyState title="No owners discovered" description="The inventory endpoint returned no owner surfaces." /> : (
              <DataTable
                rows={ownerRows}
                caption="Storage owner ledger"
                tableTestId={`${selectors.console.ownerLedger}-table`}
                searchPlaceholder="Search owners, kinds, or manifest paths"
                emptyMessage="No owners match this search or filter."
                filters={[
                  { id: "all", label: "All", predicate: () => true },
                  { id: "scenario", label: "Scenarios", predicate: (row) => row.owner.kind === "scenario" },
                  { id: "resource", label: "Resources", predicate: (row) => row.owner.kind === "resource" },
                  { id: "attention", label: "Needs review", predicate: (row) => row.status.tone === "warning" },
                ]}
                columns={[
                  { id: "owner", header: "Owner", accessor: (row) => <><span className="block truncate font-medium" title={row.owner.id}>{row.owner.id}</span><span className="block truncate text-xs text-app-muted-foreground" title={row.owner.manifest_path}>{row.owner.manifest_path}</span></>, searchValue: (row) => `${row.owner.id} ${row.owner.manifest_path}`, sortValue: (row) => row.owner.id, className: "w-[42%]" },
                  { id: "kind", header: "Kind", accessor: (row) => <span className="text-app-muted-foreground">{row.owner.kind}</span>, sortValue: (row) => row.owner.kind },
                  { id: "observed", header: "Observed", accessor: (row) => <span className="font-medium">{formatBytes(row.bytes)}</span>, sortValue: (row) => row.bytes },
                  { id: "entries", header: "Entries", accessor: (row) => <span className="text-app-muted-foreground">{row.owner.storage_entries?.length ?? 0}</span>, sortValue: (row) => row.owner.storage_entries?.length ?? 0 },
                  { id: "state", header: "State", accessor: (row) => <StatusBadge tone={row.status.tone}>{row.status.label}</StatusBadge>, sortValue: (row) => row.status.label },
                ]}
                getRowKey={(row) => `${row.owner.kind}/${row.owner.id}`}
              />
            )}
          </CardContent>
        </Card>

        <div className="space-y-4">
          <Card data-testid={selectors.console.adoption}>
            <CardHeader><CardTitle>Adoption path</CardTitle><CardDescription>Suggestions are generated from the typed inventory, not from a static provider list.</CardDescription></CardHeader>
            <CardContent>
              {adoptionQuery.isLoading ? <p className="text-sm text-app-muted-foreground">Measuring bounded owner roots…</p> : adoptionQuery.data ? <>
                <div className="grid grid-cols-3 gap-2 text-center"><Metric label="Owners" value={String(adoptionQuery.data.total_owners)} detail="discovered" /><Metric label="Findings" value={String(adoptionQuery.data.findings)} detail="inventory gaps" tone={adoptionQuery.data.findings ? "warning" : "success"} /><Metric label="Next" value={String(adoptionQuery.data.suggestions?.length ?? 0)} detail="suggestions" /></div>
                <ul className="mt-4 space-y-2">{(adoptionQuery.data.suggestions ?? []).slice(0, 5).map((suggestion) => <li key={`${suggestion.kind}/${suggestion.owner}`} className="flex items-start gap-2 rounded-control border border-app-border p-3 text-sm"><CircleHelp aria-hidden="true" className="mt-0.5 h-4 w-4 shrink-0 text-app-warning" /><span className="min-w-0"><span className="block truncate font-medium">{suggestion.owner}</span><span className="block text-xs text-app-muted-foreground">{suggestion.kind} · {suggestion.observed_bytes ? formatBytes(suggestion.observed_bytes) : "measurement not available"}</span></span></li>)}</ul>
                {(adoptionQuery.data.suggestions ?? []).length === 0 && <p className="mt-3 text-sm text-app-success">All discovered owners have at least one storage declaration.</p>}
              </> : null}
            </CardContent>
          </Card>
          <Card>
            <CardHeader><CardTitle>Infra-health signal</CardTitle><CardDescription>Persisted census telemetry only; opening this page does not scan the host.</CardDescription></CardHeader>
            <CardContent>{infraQuery.data ? <dl className="grid grid-cols-2 gap-4 text-sm"><div><dt className="text-app-muted-foreground">Ceiling coverage</dt><dd className="mt-1 text-xl font-semibold">{formatPercent(infraQuery.data.declared_ceiling_coverage)}</dd></div><div><dt className="text-app-muted-foreground">Enforced bytes</dt><dd className="mt-1 text-xl font-semibold">{formatBytes(infraQuery.data.measured_bytes_under_enforced_ceiling)}</dd><dd className="text-xs text-app-muted-foreground">{formatPercent(infraQuery.data.enforced_ceiling_coverage)} of measured storage</dd></div><div><dt className="text-app-muted-foreground">Snapshots</dt><dd className="mt-1 text-xl font-semibold">{infraQuery.data.snapshot_count}</dd></div><div><dt className="text-app-muted-foreground">Retention owners</dt><dd className="mt-1 font-medium">{retentionQuery.data?.owners.filter((owner) => (owner.budgets?.length ?? 0) > 0).length ?? "—"}</dd></div><div><dt className="text-app-muted-foreground">Growth slope</dt><dd className="mt-1 font-medium">{infraQuery.data.growth_slope_bytes_per_hour === undefined ? "Not enough history" : `${formatBytes(Math.abs(infraQuery.data.growth_slope_bytes_per_hour))}/hr`}</dd></div><div><dt className="text-app-muted-foreground">Signal</dt><dd className="mt-1 font-medium">{infraQuery.data.confidence}</dd></div></dl> : <p className="text-sm text-app-muted-foreground">Loading health signal…</p>}</CardContent>
          </Card>
        </div>
      </div>

      <div id="storage-placement" className="scroll-mt-20 grid gap-4 xl:grid-cols-[1.5fr_1fr]">
        <Card data-testid={selectors.console.placement}>
          <CardHeader><CardTitle>Placement and levers</CardTitle><CardDescription>{placementQuery.data ? `Resolved for ${placementQuery.data.platform}. Preview is required before any migration.` : "Portable placement declarations and supported relocation levers."}</CardDescription></CardHeader>
          <CardContent>{placementQuery.data ? <>
            {(placementQuery.data.lever_error || placementQuery.data.lever_warnings?.length) && <div className="mb-3 rounded-control border border-app-warning/40 bg-app-warning/10 p-3 text-sm">{placementQuery.data.lever_error ?? `${placementQuery.data.lever_warnings?.length} lever warning(s)`}</div>}
            <div className="space-y-2">{placementQuery.data.owners.slice(0, 8).map((owner) => <div key={`${owner.owner}/${owner.entry}`} className="flex items-center justify-between gap-3 rounded-control border border-app-border p-3 text-sm"><span className="min-w-0"><span className="block truncate font-medium">{owner.owner} / {owner.entry}</span><span className="block truncate text-xs text-app-muted-foreground">{owner.path ?? owner.error ?? "No portable path for this platform"}</span></span><StatusBadge tone={owner.applicable ? "success" : "warning"}>{owner.applicable ? "Resolved" : "Review"}</StatusBadge></div>)}</div>
          </> : <p className="text-sm text-app-muted-foreground">Loading placement…</p>}</CardContent>
        </Card>
        <HealthCard />
      </div>

      <div id="storage-audit" className="scroll-mt-20 grid gap-4 xl:grid-cols-2">
        <Card data-testid={selectors.console.findings}><CardHeader><CardTitle>Findings and confidence</CardTitle><CardDescription>Typed gaps remain actionable and block false certainty.</CardDescription></CardHeader><CardContent>{snapshot?.findings?.length ? <ul className="space-y-2">{snapshot.findings.slice(0, 8).map((finding, index) => <li key={`${finding.code}/${finding.path}/${index}`} className="flex gap-3 rounded-control border border-app-border p-3 text-sm"><AlertTriangle aria-hidden="true" className="mt-0.5 h-4 w-4 shrink-0 text-app-warning" /><span><span className="font-medium">{finding.code}</span><span className="block text-xs text-app-muted-foreground">{finding.message}</span></span></li>)}</ul> : <div className="flex items-center gap-2 text-sm text-app-success"><CheckCircle2 aria-hidden="true" className="h-5 w-5" />No census findings in the latest snapshot.</div>}</CardContent></Card>
        <Card data-testid={selectors.console.audit}><CardHeader><CardTitle>Migration audit</CardTitle><CardDescription>Copy, verify, remove, and audit are separate states. No action is applied from this view.</CardDescription></CardHeader><CardContent>{auditQuery.data?.length ? <ul className="space-y-2">{auditQuery.data.slice(0, 8).map((event) => <li key={event.id} className="flex items-center justify-between gap-3 rounded-control border border-app-border p-3 text-sm"><span className="min-w-0"><span className="block truncate font-medium">{event.entry}</span><span className="block truncate text-xs text-app-muted-foreground">{event.status} · {event.source_preserved ? "source preserved" : "source removed after verification"}</span></span><ChevronRight aria-hidden="true" className="h-4 w-4 shrink-0 text-app-muted-foreground" /></li>)}</ul> : <EmptyState title="No migrations recorded" description="Preview plans and explicit approval are required before an audit event can exist." />}</CardContent></Card>
      </div>

      <div className="flex items-center gap-2 text-xs text-app-muted-foreground"><ArrowDown aria-hidden="true" className="h-4 w-4" />Data is read from the storage-manager APIs. Empty, failed, and degraded states are intentional.</div>
    </section>
  );
}
