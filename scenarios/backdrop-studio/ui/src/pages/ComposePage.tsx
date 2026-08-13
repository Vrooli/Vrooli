import { useMemo, useState } from "react";

import { permittedSurfaces } from "../api/studio";
import { ExperienceSurface } from "../components/experience/ExperienceSurface";
import { MockupPreview } from "../components/studio/MockupPreview";
import { Button } from "../components/ui/button";
import { EmptyState } from "../components/ui/empty-state";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useObjectURL, useRender, useStyles, useSurfaces } from "../hooks/useStudio";
import { useTranslation } from "../i18n";

/**
 * Resolve a style and a surface into a plan, then render it.
 *
 * The plan is shown before the render because a model-backed style costs real
 * GPU time and, on a metered route, real money. An operator should be able to
 * see exactly what will be executed — which generator, which chain, which
 * geometry, which lane — while it still costs nothing to change their mind.
 */
export function ComposePage() {
  const { t } = useTranslation();
  const styles = useStyles({});
  const surfaces = useSurfaces();
  const [styleId, setStyleId] = useState("");
  const [surfaceId, setSurfaceId] = useState("");
  const [submitted, setSubmitted] = useState(false);
  const [copied, setCopied] = useState(false);

  const style = styles.data?.find((candidate) => candidate.id === styleId);
  const permitted = useMemo(
    () => (style ? permittedSurfaces(style, surfaces.data ?? []) : []),
    [style, surfaces.data],
  );
  const surface = permitted.find((s) => s.id === surfaceId) ?? permitted[0];

  const plan = useMemo(
    () =>
      style && surface
        ? JSON.stringify(
            {
              style_id: style.id,
              strategy: style.strategy,
              generator: style.scaffold?.preset ?? null,
              treatments: style.treatments,
              surface: { id: surface.id, width: surface.width, height: surface.height },
              placement: style.placements[0],
              execution_path:
                style.strategy === "guided" || style.strategy === "synthesized"
                  ? "scaffold → model → image-tools treatments"
                  : "scene → image-tools treatments",
            },
            null,
            2,
          )
        : "",
    [style, surface],
  );

  const render = useRender(
    submitted && style && surface
      ? { styleId: style.id, surfaceId: surface.id, placement: style.placements[0], seed: 7n }
      : null,
  );
  const imageURL = useObjectURL(render.data?.candidates[0]);

  return (
    <ExperienceSurface
      surfaceId="compose"
      state={styles.isLoading ? "loading" : styles.isError ? "error" : style ? "ready" : "empty"}
      statusMessage={styles.isLoading ? t(strings.pages.common.loading) : undefined}
      data-testid={selectors.pages.compose}
      aria-labelledby="compose-heading"
      className="flex flex-col gap-6"
    >
      <header className="flex flex-col gap-2">
        <h2 id="compose-heading" className="text-2xl font-semibold">
          {t(strings.pages.compose.title)}
        </h2>
        <p className="max-w-3xl text-app-muted-foreground">{t(strings.pages.compose.description)}</p>
      </header>

      <div className="flex flex-wrap items-end gap-3">
        <label className="flex flex-col gap-1 text-sm font-medium" htmlFor="compose-style">
          {t(strings.pages.compose.styleLabel)}
          <select
            id="compose-style"
            className="min-h-11 min-w-56 rounded-control border border-app-border bg-app-surface px-3"
            value={styleId}
            data-testid="compose-style-select"
            onChange={(event) => {
              setStyleId(event.target.value);
              setSurfaceId("");
              setSubmitted(false);
            }}
          >
            <option value="">{t(strings.pages.catalog.axisAny)}</option>
            {(styles.data ?? []).map((option) => (
              <option key={option.id} value={option.id}>
                {option.name}
              </option>
            ))}
          </select>
        </label>
        <label className="flex flex-col gap-1 text-sm font-medium" htmlFor="compose-surface">
          {t(strings.pages.compose.surfaceLabel)}
          <select
            id="compose-surface"
            className="min-h-11 min-w-56 rounded-control border border-app-border bg-app-surface px-3"
            value={surface?.id ?? ""}
            data-testid="compose-surface-select"
            onChange={(event) => {
              setSurfaceId(event.target.value);
              setSubmitted(false);
            }}
          >
            {permitted.map((option) => (
              <option key={option.id} value={option.id}>
                {option.id} — {option.width}×{option.height}
              </option>
            ))}
          </select>
        </label>
        <Button disabled={!style || !surface} onClick={() => setSubmitted(true)}>
          {t(strings.pages.compose.render)}
        </Button>
      </div>

      {!style ? (
        <EmptyState
          title={t(strings.pages.compose.title)}
          description={t(strings.pages.compose.chooseStyle)}
        />
      ) : (
        <>
          <section aria-labelledby="plan-heading" className="flex flex-col gap-2">
            <div className="flex items-center justify-between gap-3">
              <h3 id="plan-heading" className="text-lg font-semibold">
                {t(strings.pages.compose.planHeading)}
              </h3>
              <Button
                variant="secondary"
                size="sm"
                onClick={() => {
                  setCopied(true);
                  // `navigator.clipboard` is non-optional in lib.dom, so a
                  // truthiness guard here is dead code. The write can still
                  // reject — a document without focus, or a denied permission —
                  // and that is caught rather than left to become an unhandled
                  // rejection the test setup turns into a failure.
                  void navigator.clipboard.writeText(plan).catch(() => undefined);
                }}
              >
                {copied ? t(strings.pages.compose.planCopied) : t(strings.pages.compose.copyPlan)}
              </Button>
            </div>
            <pre
              className="overflow-x-auto rounded-control border border-app-border bg-app-surface-muted p-3 text-xs"
              data-testid="compose-plan"
            >
              {plan}
            </pre>
          </section>

          {submitted ? (
            <section aria-labelledby="compose-result-heading" className="flex flex-col gap-3">
              <h3 id="compose-result-heading" className="text-lg font-semibold">
                {t(strings.pages.style.mockupHeading)}
              </h3>
              {render.isError ? (
                <p role="alert">{t(strings.pages.style.renderError)}</p>
              ) : (
                <MockupPreview imageURL={imageURL} style={style} kind="landing" />
              )}
            </section>
          ) : null}
        </>
      )}
    </ExperienceSurface>
  );
}
