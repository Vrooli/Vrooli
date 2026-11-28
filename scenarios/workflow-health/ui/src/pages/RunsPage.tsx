import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { timeline, uiText } from "./workflowData";

export function RunsPage() {
  const { t } = useTranslation();

  return (
    <section data-testid={selectors.pages.runs} aria-labelledby="runs-heading" className="flex flex-col gap-5">
      <div>
        <h2 id="runs-heading" className="text-3xl font-semibold">
          {t(strings.pages.runs.title)}
        </h2>
        <p className="mt-2 max-w-3xl text-sm text-app-muted-foreground">
          {t(strings.pages.runs.description)}
        </p>
      </div>

      <div className="grid gap-4 xl:grid-cols-[0.75fr_1.25fr]">
        <section className="rounded-panel border border-app-border bg-app-surface p-4">
          <h3 className="text-lg font-semibold">{uiText.runs.queue}</h3>
          <div className="mt-4 space-y-3">
            {uiText.runs.runNames.map((name, index) => (
              <article key={name} className="rounded-md border border-app-border p-3">
                <p className="text-sm font-medium">{name}</p>
                <p className="mt-1 text-xs text-app-muted-foreground">
                  {`${uiText.runs.runPrefix}${index + 1}${uiText.runs.scenarioSuffix}`}
                </p>
              </article>
            ))}
          </div>
        </section>

        <section className="rounded-panel border border-app-border bg-app-surface p-4">
          <h3 className="text-lg font-semibold">{uiText.runs.timeline}</h3>
          <ol data-testid={selectors.workflow.runTimeline} className="mt-4 space-y-4">
            {timeline.map((event) => (
              <li key={event.label} className="grid gap-2 rounded-md border border-app-border p-3 md:grid-cols-[80px_1fr_120px]">
                <span className="text-sm font-semibold">{event.time}</span>
                <div>
                  <p className="text-sm font-medium">{event.label}</p>
                  <p className="text-sm text-app-muted-foreground">{event.detail}</p>
                </div>
                <span className="text-sm capitalize text-app-muted-foreground">{event.status}</span>
              </li>
            ))}
          </ol>
        </section>
      </div>
    </section>
  );
}
