import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import { DataTable, type DataTableColumn } from "../components/ui/data-table";
import { StatusBadge } from "../components/ui/status-badge";
import { HealthCard } from "../features/health/HealthCard";
import { useTranslation } from "../i18n";

type ProviderRow = {
  id: string;
  tier: "safe" | "conditional";
  bytes: string;
  status: string;
};

const PROVIDERS: ProviderRow[] = [
  { id: "tmp", tier: "safe", bytes: "640 MB", status: "ready" },
  { id: "docker", tier: "conditional", bytes: "2.1 GB", status: "operator approval" },
  { id: "apt-metadata", tier: "conditional", bytes: "312 MB", status: "preview only" },
];

const AUDIT_EVENTS = [
  { id: "plan.created", detail: "conservative profile, preview only" },
  { id: "policy.saved", detail: "forbidden providers disabled" },
] as const;

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
      <div className="grid gap-4 xl:grid-cols-[1fr_2fr]">
        <HealthCard />
        <Card data-testid={selectors.cleanup.overview}>
          <CardHeader>
            <CardTitle>{t(strings.cleanup.overview.title)}</CardTitle>
            <CardDescription>{t(strings.cleanup.overview.description)}</CardDescription>
          </CardHeader>
          <CardContent>
            <dl className="grid gap-3 sm:grid-cols-3">
              <div>
                <dt className="text-xs uppercase text-app-muted-foreground">{t(strings.cleanup.overview.reclaimable)}</dt>
                <dd className="mt-1 text-2xl font-semibold">3.0 GB</dd>
              </div>
              <div>
                <dt className="text-xs uppercase text-app-muted-foreground">{t(strings.cleanup.overview.providers)}</dt>
                <dd className="mt-1 text-2xl font-semibold">3</dd>
              </div>
              <div>
                <dt className="text-xs uppercase text-app-muted-foreground">{t(strings.cleanup.overview.blocked)}</dt>
                <dd className="mt-1 text-2xl font-semibold">0</dd>
              </div>
            </dl>
          </CardContent>
        </Card>
      </div>

      <div className="grid gap-4 xl:grid-cols-[1.4fr_1fr]">
        <Card data-testid={selectors.cleanup.providers}>
          <CardHeader>
            <CardTitle>{t(strings.cleanup.providers.title)}</CardTitle>
            <CardDescription>{t(strings.cleanup.providers.description)}</CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              rows={PROVIDERS}
              columns={providerColumns({
                provider: t(strings.cleanup.providers.provider),
                tier: t(strings.cleanup.providers.tier),
                estimate: t(strings.cleanup.providers.estimate),
                status: t(strings.cleanup.providers.status),
              })}
              getRowKey={(provider) => provider.id}
              caption={t(strings.cleanup.providers.title)}
              searchLabel={t(strings.cleanup.providers.provider)}
              searchPlaceholder={t(strings.cleanup.providers.provider)}
              tableTestId={selectors.cleanup.providers}
              className="min-w-[34rem]"
            />
          </CardContent>
        </Card>

        <Card data-testid={selectors.cleanup.policy}>
          <CardHeader>
            <CardTitle>{t(strings.cleanup.policy.title)}</CardTitle>
            <CardDescription>{t(strings.cleanup.policy.description)}</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3 text-sm">
            <div className="flex items-center justify-between gap-3 border-b border-app-border pb-2">
              <span className="text-app-muted-foreground">{t(strings.cleanup.policy.profile)}</span>
              <span className="font-medium">conservative</span>
            </div>
            <div className="flex items-center justify-between gap-3 border-b border-app-border pb-2">
              <span className="text-app-muted-foreground">{t(strings.cleanup.policy.applyGate)}</span>
              <span className="font-medium">approval required</span>
            </div>
            <div className="flex items-center justify-between gap-3">
              <span className="text-app-muted-foreground">{t(strings.cleanup.policy.replay)}</span>
              <span className="font-medium">idempotency key required</span>
            </div>
          </CardContent>
        </Card>
      </div>

      <div className="grid gap-4 xl:grid-cols-2">
        <Card data-testid={selectors.cleanup.plan}>
          <CardHeader>
            <CardTitle>{t(strings.cleanup.plan.title)}</CardTitle>
            <CardDescription>{t(strings.cleanup.plan.description)}</CardDescription>
          </CardHeader>
          <CardContent className="space-y-2 text-sm">
            <p>{t(strings.cleanup.plan.preview)}</p>
            <p className="font-medium text-app-danger">{t(strings.cleanup.plan.applyDisabled)}</p>
          </CardContent>
        </Card>

        <Card data-testid={selectors.cleanup.audit}>
          <CardHeader>
            <CardTitle>{t(strings.cleanup.audit.title)}</CardTitle>
            <CardDescription>{t(strings.cleanup.audit.description)}</CardDescription>
          </CardHeader>
          <CardContent>
            <ul className="space-y-2 text-sm">
              {AUDIT_EVENTS.map((event) => (
                <li key={event.id} className="rounded-md border border-app-border p-3">
                  <span className="font-medium">{event.id}</span>
                  <span className="block text-app-muted-foreground">{event.detail}</span>
                </li>
              ))}
            </ul>
          </CardContent>
        </Card>
      </div>
    </section>
  );
}

function providerColumns(labels: {
  provider: string;
  tier: string;
  estimate: string;
  status: string;
}): Array<DataTableColumn<ProviderRow>> {
  return [
    {
      id: "provider",
      header: labels.provider,
      accessor: (provider) => <span className="font-medium">{provider.id}</span>,
      sortValue: (provider) => provider.id,
      searchValue: (provider) => provider.id,
    },
    {
      id: "tier",
      header: labels.tier,
      accessor: (provider) => (
        <StatusBadge tone={provider.tier === "safe" ? "success" : "warning"}>{provider.tier}</StatusBadge>
      ),
      sortValue: (provider) => provider.tier,
      searchValue: (provider) => provider.tier,
    },
    {
      id: "estimate",
      header: labels.estimate,
      accessor: (provider) => provider.bytes,
      sortValue: (provider) => provider.bytes,
      searchValue: (provider) => provider.bytes,
    },
    {
      id: "status",
      header: labels.status,
      accessor: (provider) => provider.status,
      sortValue: (provider) => provider.status,
      searchValue: (provider) => provider.status,
    },
  ];
}
