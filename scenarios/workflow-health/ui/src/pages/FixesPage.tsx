import { Wrench } from "lucide-react";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { fixPreviews, uiText } from "./workflowData";

export function FixesPage() {
  const { t } = useTranslation();

  return (
    <section data-testid={selectors.pages.fixes} aria-labelledby="fixes-heading" className="flex flex-col gap-5">
      <div>
        <h2 id="fixes-heading" className="text-3xl font-semibold">
          {t(strings.pages.fixes.title)}
        </h2>
        <p className="mt-2 max-w-3xl text-sm text-app-muted-foreground">
          {t(strings.pages.fixes.description)}
        </p>
      </div>

      <section className="rounded-panel border border-app-border bg-app-surface p-4">
        <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
          <div>
            <h3 className="text-lg font-semibold">{uiText.fixes.preview}</h3>
            <p className="mt-1 text-sm text-app-muted-foreground">
              {uiText.fixes.previewDescription}
            </p>
          </div>
          <button
            type="button"
            className="inline-flex items-center justify-center gap-2 rounded-md bg-app-primary px-4 py-2 text-sm font-semibold text-app-primary-foreground"
          >
            <Wrench aria-hidden="true" className="h-4 w-4" />
            {uiText.fixes.applySelected}
          </button>
        </div>

        <div data-testid={selectors.workflow.fixPreview} className="mt-4 grid gap-3">
          {fixPreviews.map((fix) => (
            <article key={`${fix.rule}:${fix.file}`} className="rounded-md border border-app-border p-3">
              <div className="flex flex-col gap-2 md:flex-row md:items-start md:justify-between">
                <div>
                  <p className="text-sm font-semibold">{fix.rule}</p>
                  <p className="mt-1 font-mono text-xs text-app-muted-foreground">{fix.file}</p>
                </div>
                <span className="rounded-full bg-app-muted px-2.5 py-1 text-xs font-medium text-app-muted-foreground">
                  {fix.risk}
                </span>
              </div>
              <p className="mt-3 rounded-md bg-app-background p-3 text-sm">{fix.change}</p>
            </article>
          ))}
        </div>
      </section>
    </section>
  );
}
