import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { AlertTriangle, CheckCircle2, CircleSlash, Clock, Eye, ShieldAlert } from "lucide-react";
import { useMemo, useState } from "react";
import {
  fetchIncidentObservations,
  fetchIncidents,
  type Incident,
  type IncidentSeverity,
  type IncidentStatus,
  type IncidentType,
  updateIncidentStatus,
} from "../../lib/api";
import { formatRelativeTime } from "../../lib/utils";
import { ErrorDisplay } from "../../shared/components";
import { Badge, Button, Card } from "../../shared/ui/primitives";

const STATUS_OPTIONS: Array<IncidentStatus | ""> = ["open", "acknowledged", "resolved", "ignored", ""];
const SEVERITY_OPTIONS: Array<IncidentSeverity | ""> = ["critical", "warning", "info", ""];
const TYPE_OPTIONS: Array<IncidentType | ""> = ["host_integrity", "unclean_boot", "resource_failure", "scenario_failure", "autoheal_failure", "manual", ""];

export function IncidentsPage() {
  const queryClient = useQueryClient();
  const [status, setStatus] = useState<IncidentStatus | "">("open");
  const [severity, setSeverity] = useState<IncidentSeverity | "">("");
  const [type, setType] = useState<IncidentType | "">("");
  const [selectedId, setSelectedId] = useState<string | null>(null);

  const query = useQuery({
    queryKey: ["incidents", status, severity, type],
    queryFn: () => fetchIncidents({ status, severity, type, limit: 100 }),
    refetchInterval: 60000,
  });

  const incidents = useMemo(() => query.data?.incidents ?? [], [query.data?.incidents]);
  const selectedIncident = useMemo(
    () => incidents.find((incident) => incident.id === selectedId) ?? incidents[0] ?? null,
    [incidents, selectedId],
  );

  const mutation = useMutation({
    mutationFn: ({ id, action }: { id: string; action: "acknowledge" | "resolve" | "ignore" }) =>
      updateIncidentStatus(id, action),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["incidents"] });
    },
  });

  if (query.error) {
    return <ErrorDisplay error={query.error} onRetry={() => query.refetch()} title="Unable to load incidents" />;
  }

  return (
    <div className="space-y-4" data-testid="autoheal-incidents-page">
      <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
        <div className="flex items-center gap-3">
          <ShieldAlert className="text-accent-primary" size={24} />
          <div>
            <h2 className="text-xl font-semibold">Incidents</h2>
            <p className="text-xs text-text-muted">Chronological host and recovery findings</p>
          </div>
        </div>
        <div className="flex flex-wrap gap-2">
          <select className="h-9 rounded-lg border border-border-default/70 bg-surface-overlay/40 px-3 text-sm" value={status} onChange={(event) => setStatus(event.target.value as IncidentStatus | "")}>
            {STATUS_OPTIONS.map((option) => <option key={option || "all"} value={option}>{option || "all statuses"}</option>)}
          </select>
          <select className="h-9 rounded-lg border border-border-default/70 bg-surface-overlay/40 px-3 text-sm" value={severity} onChange={(event) => setSeverity(event.target.value as IncidentSeverity | "")}>
            {SEVERITY_OPTIONS.map((option) => <option key={option || "all"} value={option}>{option || "all severities"}</option>)}
          </select>
          <select className="h-9 rounded-lg border border-border-default/70 bg-surface-overlay/40 px-3 text-sm" value={type} onChange={(event) => setType(event.target.value as IncidentType | "")}>
            {TYPE_OPTIONS.map((option) => <option key={option || "all"} value={option}>{option || "all types"}</option>)}
          </select>
        </div>
      </div>

      <div className="grid gap-4 lg:grid-cols-[minmax(0,0.9fr)_minmax(0,1.1fr)]">
        <Card className="p-3">
          <div className="mb-3 flex items-center justify-between">
            <h3 className="text-sm font-semibold">Incident Timeline</h3>
            <Badge tone="info">{query.data?.total ?? 0}</Badge>
          </div>
          <div className="max-h-[34rem] space-y-2 overflow-y-auto">
            {incidents.length === 0 ? (
              <p className="rounded-lg bg-surface-overlay/30 p-3 text-sm text-text-muted">No incidents match the selected filters.</p>
            ) : incidents.map((incident) => (
              <button
                key={incident.id}
                className={`w-full rounded-lg border p-3 text-left transition-colors ${selectedIncident?.id === incident.id ? "border-accent-primary bg-accent-primary/10" : "border-border-default/60 bg-surface-overlay/20 hover:bg-surface-overlay/40"}`}
                onClick={() => setSelectedId(incident.id)}
              >
                <div className="mb-2 flex items-center justify-between gap-2">
                  <span className="min-w-0 truncate text-sm font-medium">{incident.title}</span>
                  <SeverityBadge severity={incident.severity} />
                </div>
                <p className="line-clamp-2 text-xs text-text-muted">{incident.summary}</p>
                <div className="mt-2 flex flex-wrap gap-2 text-xs text-text-muted">
                  <span>{incident.status}</span>
                  <span>{incident.type}</span>
                  <span>{formatRelativeTime(incident.lastSeenAt)}</span>
                  <span>count {incident.occurrenceCount}</span>
                </div>
              </button>
            ))}
          </div>
        </Card>

        <Card className="p-4">
          {selectedIncident ? (
            <IncidentDetail
              incident={selectedIncident}
              busy={mutation.isPending}
              onAction={(action) => mutation.mutate({ id: selectedIncident.id, action })}
            />
          ) : (
            <p className="text-sm text-text-muted">Select an incident to inspect evidence.</p>
          )}
        </Card>
      </div>
    </div>
  );
}

function IncidentDetail({ incident, busy, onAction }: { incident: Incident; busy: boolean; onAction: (action: "acknowledge" | "resolve" | "ignore") => void }) {
  const observations = useQuery({
    queryKey: ["incident-observations", incident.id],
    queryFn: () => fetchIncidentObservations(incident.id),
  });
  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h3 className="text-lg font-semibold">{incident.title}</h3>
          <p className="text-sm text-text-muted">{incident.summary}</p>
        </div>
        <div className="flex flex-wrap gap-2">
          <SeverityBadge severity={incident.severity} />
          <Badge tone="info">{incident.status}</Badge>
        </div>
      </div>
      <div className="grid gap-2 text-sm sm:grid-cols-2">
        <DetailLine label="Detected" value={formatRelativeTime(incident.detectedAt)} />
        <DetailLine label="Last Seen" value={formatRelativeTime(incident.lastSeenAt)} />
        <DetailLine label="Type" value={incident.type} />
        <DetailLine label="Occurrences" value={String(incident.occurrenceCount)} />
      </div>
      <div className="flex flex-wrap gap-2">
        {incident.status === "open" ? (
          <Button size="sm" variant="outline" disabled={busy} onClick={() => onAction("acknowledge")}>
            <Eye className="mr-2 h-4 w-4" /> Acknowledge
          </Button>
        ) : null}
        {incident.status !== "resolved" ? (
          <Button size="sm" variant="outline" disabled={busy} onClick={() => onAction("resolve")}>
            <CheckCircle2 className="mr-2 h-4 w-4" /> Resolve
          </Button>
        ) : null}
        {incident.status !== "ignored" ? (
          <Button size="sm" variant="outline" disabled={busy} onClick={() => onAction("ignore")}>
            <CircleSlash className="mr-2 h-4 w-4" /> Ignore
          </Button>
        ) : null}
      </div>
      <section>
        <h4 className="mb-2 text-sm font-semibold">Source Checks</h4>
        <div className="flex flex-wrap gap-2">
          {(incident.sourceCheckIds ?? []).map((checkId) => <Badge key={checkId} tone="info">{checkId}</Badge>)}
        </div>
      </section>
      <section>
        <h4 className="mb-2 text-sm font-semibold">Recommendations</h4>
        <ul className="space-y-1 text-sm text-text-muted">
          {(incident.recommendations ?? []).map((item) => <li key={item}>{item}</li>)}
          {(incident.recommendations ?? []).length === 0 ? <li>No recommendations were attached.</li> : null}
        </ul>
      </section>
      <section>
        <h4 className="mb-2 text-sm font-semibold">Observations</h4>
        <div className="max-h-48 space-y-2 overflow-y-auto">
          {(observations.data?.observations ?? []).map((observation) => (
            <div key={observation.id} className="rounded-lg bg-surface-overlay/25 p-2 text-xs">
              <div className="mb-1 flex items-center gap-2 text-text-muted">
                <Clock className="h-3 w-3" />
                <span>{formatRelativeTime(observation.observedAt)}</span>
                <span>{observation.severity}</span>
              </div>
              <p className="text-text-primary">{observation.message}</p>
            </div>
          ))}
          {observations.isLoading ? <p className="text-sm text-text-muted">Loading observations...</p> : null}
        </div>
      </section>
    </div>
  );
}

function DetailLine({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg bg-surface-overlay/25 p-2">
      <p className="text-xs text-text-muted">{label}</p>
      <p className="break-words text-sm text-text-primary">{value}</p>
    </div>
  );
}

function SeverityBadge({ severity }: { severity: IncidentSeverity }) {
  if (severity === "critical") {
    return <Badge tone="danger"><AlertTriangle className="mr-1 h-3 w-3" /> critical</Badge>;
  }
  if (severity === "warning") {
    return <Badge tone="warning"><AlertTriangle className="mr-1 h-3 w-3" /> warning</Badge>;
  }
  return <Badge tone="info">{severity}</Badge>;
}
