import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { assets, findings, stats, timeline, uiText } from "./workflowData";

export function DashboardPage() {
  const { t } = useTranslation();

  return (
    <section
      data-testid={selectors.pages.overview}
      aria-labelledby="overview-heading"
      className="flex flex-col gap-5"
    >
      <div className="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <p className="text-xs font-semibold uppercase tracking-normal text-app-muted-foreground">
            {t(strings.app.eyebrow)}
          </p>
          <h2 id="overview-heading" className="mt-1 text-3xl font-semibold">
            {t(strings.pages.overview.title)}
          </h2>
          <p className="mt-2 max-w-3xl text-sm text-app-muted-foreground">
            {t(strings.app.description)}
          </p>
        </div>
        <label className="flex w-full flex-col gap-1 text-sm font-medium text-app-muted-foreground sm:w-72">
          {uiText.overview.scenarioLabel}
          <select
            data-testid={selectors.workflow.scenarioSelect}
            className="rounded-md border border-app-border bg-app-surface px-3 py-2 text-app-foreground"
            defaultValue="workflow-health"
          >
            {uiText.overview.scenarios.map((scenario) => (
              <option key={scenario} value={scenario}>
                {scenario}
              </option>
            ))}
          </select>
        </label>
      </div>

      <div data-testid={selectors.workflow.overview} className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
        {stats.map((stat) => {
          const Icon = stat.icon;
          return (
            <article
              key={stat.label}
              className="rounded-panel border border-app-border bg-app-surface p-4"
            >
              <div className="flex items-start justify-between gap-3">
                <div>
                  <p className="text-xs font-semibold uppercase tracking-normal text-app-muted-foreground">
                    {stat.label}
                  </p>
                  <p className="mt-2 text-2xl font-semibold">{stat.value}</p>
                </div>
                <Icon aria-hidden="true" className="h-5 w-5 text-app-primary" />
              </div>
              <p className="mt-3 text-sm text-app-muted-foreground">{stat.detail}</p>
            </article>
          );
        })}
      </div>

      <div className="grid gap-4 xl:grid-cols-[1.2fr_0.8fr]">
        <section className="rounded-panel border border-app-border bg-app-surface p-4">
          <div className="flex items-center justify-between gap-3">
            <h3 className="text-lg font-semibold">{uiText.overview.assetPosture}</h3>
            <span className="rounded-full bg-emerald-100 px-2.5 py-1 text-xs font-medium text-emerald-800">
              {assets.filter((asset) => asset.status === "Ready").length} {uiText.overview.readySuffix}
            </span>
          </div>
          <div className="mt-4 overflow-x-auto">
            <table className="min-w-full text-left text-sm">
              <thead className="text-xs uppercase text-app-muted-foreground">
                <tr>
                  {uiText.overview.assetHeaders.map((header) => (
                    <th key={header} className="px-2 py-2 font-semibold">
                      {header}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {assets.slice(0, 4).map((asset) => (
                  <tr key={asset.path} className="border-t border-app-border">
                    <td className="px-2 py-3">{asset.kind}</td>
                    <td className="px-2 py-3">
                      <p className="font-medium">{asset.name}</p>
                      <p className="text-xs text-app-muted-foreground">{asset.path}</p>
                    </td>
                    <td className="px-2 py-3">{asset.safety}</td>
                    <td className="px-2 py-3">{asset.status}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>

        <section className="rounded-panel border border-app-border bg-app-surface p-4">
          <h3 className="text-lg font-semibold">{uiText.overview.latestRun}</h3>
          <ol className="mt-4 space-y-3">
            {timeline.map((event) => (
              <li key={event.label} className="flex gap-3">
                <span className="mt-1 h-2.5 w-2.5 rounded-full bg-app-primary" />
                <div>
                  <p className="text-sm font-medium">
                    {event.time} - {event.label}
                  </p>
                  <p className="text-sm text-app-muted-foreground">{event.detail}</p>
                </div>
              </li>
            ))}
          </ol>
        </section>
      </div>

      <section className="rounded-panel border border-app-border bg-app-surface p-4">
        <h3 className="text-lg font-semibold">{uiText.overview.topFindings}</h3>
        <div className="mt-3 grid gap-3 md:grid-cols-3">
          {findings.map((finding) => (
            <article key={finding.id} className="rounded-md border border-app-border p-3">
              <p className="text-xs font-semibold uppercase tracking-normal text-app-muted-foreground">
                {finding.severity} - {finding.id}
              </p>
              <p className="mt-2 text-sm font-medium">{finding.summary}</p>
              <p className="mt-1 text-sm text-app-muted-foreground">{finding.remediation}</p>
            </article>
          ))}
        </div>
      </section>
    </section>
  );
}
