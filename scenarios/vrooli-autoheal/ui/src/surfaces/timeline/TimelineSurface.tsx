import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { AlertTriangle, Clock, Filter, RefreshCw } from "lucide-react";
import { useMemo, useState } from "react";
import {
  fetchSystemEvents,
  refreshSystemEvents,
  type SystemEvent,
  type SystemEventSeverity,
  type SystemEventSourceStatus,
} from "../../lib/api";
import { formatRelativeTime } from "../../lib/utils";
import { ErrorDisplay } from "../../shared/components";
import { Badge, Button, Card } from "../../shared/ui/primitives";

const WINDOWS = [
  { label: "24h", value: "24h" },
  { label: "3d", value: "72h" },
  { label: "7d", value: "7d" },
  { label: "30d", value: "30d" },
];

const CATEGORIES = ["kernel", "driver", "firmware", "crash", "hardware", "boot", "display"];
const SEVERITIES: SystemEventSeverity[] = ["critical", "warning", "info"];

export function TimelineSurface() {
  const queryClient = useQueryClient();
  const [since, setSince] = useState("72h");
  const [category, setCategory] = useState("");
  const [severity, setSeverity] = useState("");

  const query = useQuery({
    queryKey: ["system-events", since, category, severity],
    queryFn: () => fetchSystemEvents({ since, category, severity, limit: 200, correlate: true }),
    refetchInterval: 60000,
  });

  const refreshMutation = useMutation({
    mutationFn: refreshSystemEvents,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["system-events"] }),
  });

  const grouped = useMemo(() => groupEventsByDay(query.data?.events ?? []), [query.data?.events]);

  if (query.isLoading) {
    return (
      <Card className="flex min-h-[18rem] items-center justify-center p-6">
        <div className="text-center">
          <RefreshCw className="mx-auto mb-3 h-5 w-5 animate-spin text-accent-primary" />
          <p className="text-sm text-text-muted">Loading system timeline...</p>
        </div>
      </Card>
    );
  }

  if (query.error) {
    return <ErrorDisplay error={query.error} onRetry={() => query.refetch()} title="Unable to load system timeline" />;
  }

  return (
    <div className="space-y-4" data-testid="autoheal-system-timeline">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 className="flex items-center gap-2 text-lg font-medium">
            <Clock size={20} className="text-accent-primary" />
            System Timeline
          </h2>
          <p className="text-sm text-text-muted">Chronological host events, updates, crashes, and deterministic correlation hints.</p>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={() => refreshMutation.mutate()}
          disabled={refreshMutation.isPending}
        >
          <RefreshCw className={`h-4 w-4 sm:mr-2 ${refreshMutation.isPending ? "animate-spin" : ""}`} />
          <span className="hidden sm:inline">Refresh</span>
        </Button>
      </div>

      <SourceStrip sources={query.data?.sources ?? []} />

      <Card className="p-3">
        <div className="flex flex-wrap items-center gap-2">
          <Filter className="h-4 w-4 text-text-muted" />
          <Segmented value={since} options={WINDOWS} onChange={setSince} />
          <select
            value={category}
            onChange={(event) => setCategory(event.target.value)}
            className="h-9 rounded-md border border-border-default bg-surface-overlay px-2 text-sm text-text-primary"
            aria-label="Filter category"
          >
            <option value="">All categories</option>
            {CATEGORIES.map((item) => <option key={item} value={item}>{item}</option>)}
          </select>
          <select
            value={severity}
            onChange={(event) => setSeverity(event.target.value)}
            className="h-9 rounded-md border border-border-default bg-surface-overlay px-2 text-sm text-text-primary"
            aria-label="Filter severity"
          >
            <option value="">All severities</option>
            {SEVERITIES.map((item) => <option key={item} value={item}>{item}</option>)}
          </select>
          <span className="ml-auto text-xs text-text-muted">{query.data?.count ?? 0} events</span>
        </div>
      </Card>

      {(query.data?.correlations?.length ?? 0) > 0 ? (
        <Card className="border-accent-warning/30 bg-accent-warning/10 p-4">
          <h3 className="mb-2 flex items-center gap-2 text-sm font-medium text-accent-warning">
            <AlertTriangle size={16} />
            Correlation Hints
          </h3>
          <div className="space-y-2">
            {query.data?.correlations?.map((hint) => (
              <div key={`${hint.title}-${hint.eventIds.join("-")}`} className="text-sm">
                <p className="font-medium text-text-primary">{hint.title}</p>
                <p className="text-text-muted">{hint.summary}</p>
                <p className="text-xs text-text-muted/80">{hint.rationale}</p>
              </div>
            ))}
          </div>
        </Card>
      ) : null}

      <div className="space-y-4">
        {grouped.length === 0 ? (
          <Card className="p-6 text-center text-sm text-text-muted">No system events matched the selected filters.</Card>
        ) : grouped.map((group) => (
          <section key={group.day} className="space-y-2">
            <h3 className="text-sm font-medium text-text-muted">{group.day}</h3>
            <div className="divide-y divide-border-default/50 rounded-lg border border-border-default/70 bg-surface-elevated/50">
              {group.events.map((event) => <SystemEventRow key={`${event.fingerprint}-${event.id}`} event={event} />)}
            </div>
          </section>
        ))}
      </div>
    </div>
  );
}

function Segmented({ value, options, onChange }: { value: string; options: Array<{ label: string; value: string }>; onChange: (value: string) => void }) {
  return (
    <div className="flex rounded-md border border-border-default bg-surface-overlay p-0.5">
      {options.map((option) => (
        <button
          key={option.value}
          type="button"
          onClick={() => onChange(option.value)}
          className={`h-8 px-2 text-xs transition-colors ${value === option.value ? "rounded bg-accent-primary/25 text-text-primary" : "text-text-muted hover:text-text-primary"}`}
        >
          {option.label}
        </button>
      ))}
    </div>
  );
}

function SourceStrip({ sources }: { sources: SystemEventSourceStatus[] }) {
  if (sources.length === 0) {
    return null;
  }
  return (
    <div className="flex flex-wrap gap-2">
      {sources.map((source) => (
        <Badge key={source.source} tone={source.status === "ok" ? "success" : source.status === "unsupported" ? "neutral" : "warning"} size="sm" title={source.lastError || source.source}>
          {source.source}: {source.status}
        </Badge>
      ))}
    </div>
  );
}

function SystemEventRow({ event }: { event: SystemEvent }) {
  return (
    <article className="grid gap-2 p-3 sm:grid-cols-[10rem_1fr]">
      <div className="text-xs text-text-muted" title={new Date(event.occurredAt).toLocaleString()}>
        {formatRelativeTime(event.occurredAt)}
      </div>
      <div className="min-w-0">
        <div className="flex flex-wrap items-center gap-2">
          <Badge tone={severityTone(event.severity)} size="sm">{event.severity}</Badge>
          <Badge tone="neutral" size="sm">{event.category}</Badge>
          <span className="text-xs text-text-muted">{event.source}</span>
        </div>
        <h4 className="mt-1 break-words text-sm font-medium text-text-primary">{event.title}</h4>
        <p className="break-words text-sm text-text-muted">{event.summary}</p>
        {event.bootId ? <p className="mt-1 break-all font-mono text-xs text-text-muted/80">boot {event.bootId}</p> : null}
      </div>
    </article>
  );
}

function severityTone(severity: SystemEventSeverity): "neutral" | "warning" | "danger" {
  if (severity === "critical") return "danger";
  if (severity === "warning") return "warning";
  return "neutral";
}

function groupEventsByDay(events: SystemEvent[]) {
  const groups = new Map<string, SystemEvent[]>();
  for (const event of events) {
    const day = new Date(event.occurredAt).toLocaleDateString(undefined, { year: "numeric", month: "short", day: "numeric" });
    groups.set(day, [...(groups.get(day) ?? []), event]);
  }
  return Array.from(groups.entries()).map(([day, items]) => ({ day, events: items }));
}

