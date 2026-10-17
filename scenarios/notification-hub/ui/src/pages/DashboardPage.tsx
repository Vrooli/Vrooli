import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { HealthCard } from "../features/health/HealthCard";
import { useTranslation } from "../i18n";
import { useQuery } from "@tanstack/react-query";
import { deliveryClient } from "../api/notifications";

/**
 * Dashboard / home page. Composes the health card plus stat placeholders.
 * Replace the cards with real surfaces when the scenario grows them.
 */
export function DashboardPage() {
  const { t } = useTranslation();

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
      <div className="grid gap-4 lg:grid-cols-3">
        <HealthCard />
        <TimelineCard />
        <DeliverySummaryCard />
      </div>
    </section>
  );
}

function DeliverySummaryCard() {
  const analytics = useQuery({
    queryKey: ["notification-summary"],
    queryFn: () => deliveryClient.getTimeline({ limit: 100 }),
    refetchInterval: 15000,
  });
  const notifications = analytics.data?.notifications ?? [];
  const delivered = notifications.filter((item) => item.state.toString().endsWith("DELIVERED")).length;
  return (
    <Card>
      <CardHeader><CardTitle className="text-sm uppercase text-app-muted-foreground">Delivery analytics</CardTitle></CardHeader>
      <CardContent>
        {analytics.isError ? <p className="text-sm text-red-400">Delivery summary unavailable.</p> : <>
          <p className="text-2xl font-semibold">{notifications.length === 0 ? "--" : `${delivered}/${notifications.length}`}</p>
          <p className="text-sm text-app-muted-foreground">notifications delivered in the current view</p>
        </>}
      </CardContent>
    </Card>
  );
}

function TimelineCard() {
  const timeline = useQuery({
    queryKey: ["notification-timeline"],
    queryFn: () => deliveryClient.getTimeline({ limit: 20 }),
    refetchInterval: 5000,
  });
  const notifications = timeline.data?.notifications ?? [];
  return (
    <Card className="lg:col-span-2">
      <CardHeader><CardTitle className="text-sm uppercase text-app-muted-foreground">Delivery timeline</CardTitle></CardHeader>
      <CardContent>
        {timeline.isError && <p className="text-sm text-red-400">Timeline unavailable — check the API identity and scenario health.</p>}
        {!timeline.isError && notifications.length === 0 && <p className="text-sm text-app-muted-foreground">No notifications yet. Register this device, then send one from the CLI.</p>}
        <ul className="space-y-2" aria-label="Notification delivery timeline">
          {notifications.map((item) => <li key={item.id} className="flex items-center justify-between rounded border border-app-border px-3 py-2 text-sm"><span className="truncate">{item.title || item.body}</span><span className="ml-3 shrink-0 text-app-muted-foreground">{item.state.toString().replace("NOTIFICATION_STATE_", "").toLowerCase()}</span></li>)}
        </ul>
      </CardContent>
    </Card>
  );
}
