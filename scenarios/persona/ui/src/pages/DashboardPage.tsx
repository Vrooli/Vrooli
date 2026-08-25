import { Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";

import { listAllHandoffs, listPersonas } from "../api/persona";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { Button } from "@vrooli/react-component-library/Button/1.2.0";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import { HealthCard } from "../features/health/HealthCard";
import { useTranslation } from "../i18n";
import { HandoffState } from "@vrooli/proto-types/persona/v1/handoffs/handoffs_pb";

/**
 * Dashboard / home page. Composes the health card plus stat placeholders.
 * Replace the cards with real surfaces when the scenario grows them.
 */
export function DashboardPage() {
  const { t } = useTranslation();
  const personasQuery = useQuery({ queryKey: ["personas", "active"], queryFn: () => listPersonas(false) });
  const handoffsQuery = useQuery({
    queryKey: ["handoffs", "dashboard", personasQuery.data?.map((persona) => persona.id)],
    queryFn: () => listAllHandoffs(personasQuery.data ?? []),
    enabled: personasQuery.isSuccess,
  });
  const openHandoffs = (handoffsQuery.data ?? []).filter((handoff) => handoff.state === HandoffState.AWAITING_HUMAN);
  const loading = personasQuery.isPending || handoffsQuery.isPending;

  return (
    <section
      data-testid={selectors.pages.dashboard}
      aria-labelledby="dashboard-heading"
      className="flex flex-col gap-4"
    >
      <h2 id="dashboard-heading" className="text-2xl font-semibold">
        {t(strings.pages.dashboard.title)}
      </h2>
      <p className="max-w-3xl text-app-muted-foreground">{t(strings.pages.dashboard.description)}</p>
      <div className="grid gap-4 lg:grid-cols-3">
        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle>What needs you</CardTitle>
            <CardDescription>Human-only steps are durable, explicit, and never silently discarded.</CardDescription>
          </CardHeader>
          <CardContent>
            {loading ? <p className="text-sm text-app-muted-foreground">Loading the queue…</p> : null}
            {!loading && openHandoffs.length === 0 ? (
              <div className="flex flex-col items-start gap-3">
                <p className="text-sm text-app-muted-foreground">Nothing is waiting for a human right now.</p>
                <Link to="/personas"><Button size="sm">Inspect personas</Button></Link>
              </div>
            ) : null}
            {openHandoffs.length > 0 ? (
              <ul className="divide-y divide-app-border">
                {openHandoffs.slice(0, 3).map((handoff) => (
                  <li key={handoff.id} className="flex items-center justify-between gap-4 py-3">
                    <div>
                      <p className="font-medium">{handoff.title}</p>
                      <p className="text-sm text-app-muted-foreground">{handoff.humanAction}</p>
                    </div>
                    <Link className="text-sm font-semibold text-app-primary underline-offset-4 hover:underline" to={`/handoffs/${handoff.id}`}>Open</Link>
                  </li>
                ))}
              </ul>
            ) : null}
          </CardContent>
        </Card>
        <Card>
          <CardHeader><CardTitle>System posture</CardTitle><CardDescription>Current durable records</CardDescription></CardHeader>
          <CardContent className="grid grid-cols-2 gap-4">
            <Metric label="Personas" value={personasQuery.data?.length ?? 0} />
            <Metric label="Open handoffs" value={openHandoffs.length} />
            <Metric label="Queue status" value={handoffsQuery.isError ? "Stale" : "Ready"} />
            <Metric label="Authority" value="Fail-closed" />
          </CardContent>
        </Card>
        <HealthCard />
      </div>
    </section>
  );
}

function Metric({ label, value }: { label: string; value: string | number }) {
  return <div><p className="text-xs uppercase tracking-wide text-app-muted-foreground">{label}</p><p className="mt-1 text-xl font-semibold">{value}</p></div>;
}
