import { useMemo, useState } from "react";
import { Link, useParams } from "react-router-dom";

import { parseProvenance, parseQuality, permittedSurfaces, qualityTierOf } from "../api/studio";
import { qualityTierString } from "../consts/qualityTier";
import { ExperienceSurface } from "../components/experience/ExperienceSurface";
import { MockupPreview } from "../components/studio/MockupPreview";
import { QualityMeters } from "../components/studio/QualityMeters";
import { StatusBadge } from "../components/ui/status-badge";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useObjectURL, useRender, useStyles, useSurfaces } from "../hooks/useStudio";
import { useTranslation } from "../i18n";

/**
 * One style, in full: how it is made, what it measures, and how it reads behind
 * copy.
 *
 * The three sections answer the three questions an operator has before using a
 * style, and each was previously unanswerable. The chain and its resolved
 * parameters say *how* — including the pixel values a relative parameter became
 * at this surface, which is the only place that conversion is visible. The
 * perceptual meters say *how safely*, with margins rather than a bare verdict.
 * The mockups say *whether it works*, with real type rather than grey bars.
 */
export function StylePage() {
  const { t } = useTranslation();
  const { styleId = "" } = useParams();
  const styles = useStyles({});
  const surfaces = useSurfaces();
  const [seed, setSeed] = useState(7n);

  const style = styles.data?.find((candidate) => candidate.id === styleId);
  const permitted = useMemo(
    () => (style ? permittedSurfaces(style, surfaces.data ?? []) : []),
    [style, surfaces.data],
  );
  const [surfaceId, setSurfaceId] = useState<string>("");
  const surface = permitted.find((s) => s.id === surfaceId) ?? permitted[permitted.length - 1];

  const render = useRender(
    style && surface
      ? { styleId: style.id, surfaceId: surface.id, placement: style.placements[0], seed }
      : null,
  );
  const candidate = render.data?.candidates[0];
  const imageURL = useObjectURL(candidate);
  const quality = candidate ? parseQuality(candidate) : null;
  const provenance = candidate ? parseProvenance(candidate) : null;

  const state = styles.isLoading
    ? "loading"
    : styles.isError
      ? "error"
      : !style
        ? "empty"
        : "ready";

  if (state !== "ready" || !style) {
    return (
      <ExperienceSurface
        surfaceId="style"
        state={state}
        statusMessage={state === "loading" ? t(strings.pages.common.loading) : undefined}
        data-testid={selectors.pages.style}
        className="flex flex-col gap-4"
      >
        {state === "loading" ? <p role="status">{t(strings.pages.common.loading)}</p> : null}
        {state === "error" ? <p role="alert">{t(strings.pages.common.error)}</p> : null}
        {state === "empty" ? (
          <div className="flex flex-col gap-3">
            <p>{t(strings.pages.style.notFound)}</p>
            <Link to="/catalog" className="underline">
              {t(strings.pages.style.backToCatalog)}
            </Link>
          </div>
        ) : null}
      </ExperienceSurface>
    );
  }

  // A region fails when the render reported it: the perceptual verdict names
  // reserved_quiet when a reserved area is busier than the frame around it.
  const failingRegions =
    quality && !quality.passed && quality.failures?.some((f) => f.includes("reserved"))
      ? style.regions.map((_, index) => index)
      : [];

  return (
    <ExperienceSurface
      surfaceId="style"
      state={render.isLoading ? "loading" : render.isError ? "partial" : "ready"}
      statusMessage={render.isLoading ? t(strings.pages.common.loading) : undefined}
      data-testid={selectors.pages.style}
      data-style-id={style.id}
      aria-labelledby="style-heading"
      className="flex flex-col gap-6"
    >
      <header className="flex flex-col gap-2">
        <Link to="/catalog" className="text-sm underline">
          {t(strings.pages.style.backToCatalog)}
        </Link>
        <div className="flex flex-wrap items-center justify-between gap-3">
          <h2 id="style-heading" className="text-2xl font-semibold">
            {style.name}
          </h2>
          <div className="flex items-center gap-2">
            <StatusBadge>{style.strategy}</StatusBadge>
            <StatusBadge>{t(qualityTierString(qualityTierOf(style)))}</StatusBadge>
          </div>
        </div>
        <p className="text-app-muted-foreground">
          {style.role} · {style.subject} · {style.lineage}
        </p>
      </header>

      <section className="flex flex-wrap items-end gap-3">
        <label className="flex flex-col gap-1 text-sm font-medium" htmlFor="style-surface">
          {t(strings.pages.style.surfaceLabel)}
          <select
            id="style-surface"
            className="min-h-11 rounded-control border border-app-border bg-app-surface px-3"
            value={surface?.id ?? ""}
            data-testid="style-surface-select"
            onChange={(event) => setSurfaceId(event.target.value)}
          >
            {permitted.map((option) => (
              <option key={option.id} value={option.id}>
                {option.id} — {option.width}×{option.height}
              </option>
            ))}
          </select>
        </label>
        <label className="flex flex-col gap-1 text-sm font-medium" htmlFor="style-seed">
          {t(strings.pages.style.seedLabel)}
          <input
            id="style-seed"
            type="number"
            className="min-h-11 w-32 rounded-control border border-app-border bg-app-surface px-3"
            value={Number(seed)}
            data-testid="style-seed-input"
            onChange={(event) => setSeed(BigInt(event.target.value || "0"))}
          />
        </label>
        <Link
          to={`/sweep?style=${style.id}`}
          className="inline-flex min-h-11 items-center rounded-control border border-app-border px-4 text-sm font-medium"
        >
          {t(strings.pages.style.compareHeading)}
        </Link>
        <Link
          to={`/remix?parent=${style.id}`}
          className="inline-flex min-h-11 items-center rounded-control border border-app-border px-4 text-sm font-medium"
        >
          {t(strings.pages.style.remixAction)}
        </Link>
      </section>

      {render.isError ? (
        <p role="alert" className="rounded-panel border border-app-border p-4">
          {t(strings.pages.style.renderError)}
        </p>
      ) : null}

      <section aria-labelledby="mockup-heading" className="flex flex-col gap-3">
        <h3 id="mockup-heading" className="text-lg font-semibold">
          {t(strings.pages.style.mockupHeading)}
        </h3>
        <div className="grid gap-4 xl:grid-cols-2">
          <MockupPreview imageURL={imageURL} style={style} kind="landing" failingRegions={failingRegions} />
          <MockupPreview imageURL={imageURL} style={style} kind="store" failingRegions={failingRegions} />
        </div>
      </section>

      <section aria-labelledby="chain-heading" className="flex flex-col gap-3">
        <h3 id="chain-heading" className="text-lg font-semibold">
          {t(strings.pages.style.chainHeading)}
        </h3>
        {style.treatments.length === 0 ? (
          <p className="text-app-muted-foreground">{t(strings.pages.style.chainEmpty)}</p>
        ) : (
          <ol className="flex flex-wrap items-center gap-2" data-testid="style-chain">
            {style.treatments.map((treatment, index) => (
              <li key={`${treatment}-${index}`} className="flex items-center gap-2">
                <StatusBadge>{treatment}</StatusBadge>
                {index < style.treatments.length - 1 ? <span aria-hidden="true">→</span> : null}
              </li>
            ))}
          </ol>
        )}
        <h4 className="text-sm font-semibold">{t(strings.pages.style.parametersHeading)}</h4>
        <pre
          className="overflow-x-auto rounded-control border border-app-border bg-app-surface-muted p-3 text-xs"
          data-testid="style-parameters"
        >
          {JSON.stringify(style.treatmentParams, null, 2)}
        </pre>
        {style.scaffold ? (
          <>
            <h4 className="text-sm font-semibold">{t(strings.pages.style.generatorHeading)}</h4>
            <pre className="overflow-x-auto rounded-control border border-app-border bg-app-surface-muted p-3 text-xs">
              {JSON.stringify(style.scaffold, null, 2)}
            </pre>
          </>
        ) : null}
        {style.generation?.promptTemplate ? (
          <>
            <h4 className="text-sm font-semibold">{t(strings.pages.style.promptHeading)}</h4>
            <p className="rounded-control border border-app-border bg-app-surface-muted p-3 text-sm" data-testid="style-prompt">
              {style.generation.promptTemplate}
            </p>
          </>
        ) : null}
      </section>

      <section aria-labelledby="quality-heading" className="flex flex-col gap-3">
        <h3 id="quality-heading" className="text-lg font-semibold">
          {t(strings.pages.style.qualityHeading)}
        </h3>
        <QualityMeters verdict={quality} />
        {provenance ? (
          <details>
            <summary className="cursor-pointer text-sm font-medium">{t(strings.pages.common.provenanceHeading)}</summary>
            <pre className="overflow-x-auto rounded-control border border-app-border bg-app-surface-muted p-3 text-xs">
              {JSON.stringify(provenance, null, 2)}
            </pre>
          </details>
        ) : null}
      </section>

      <section aria-labelledby="regions-heading" className="flex flex-col gap-2">
        <h3 id="regions-heading" className="text-lg font-semibold">
          {t(strings.pages.style.regionsHeading)}
        </h3>
        {style.regions.length === 0 ? (
          <p className="text-app-muted-foreground">{t(strings.pages.style.regionsEmpty)}</p>
        ) : (
          <ul className="text-sm" data-testid="style-regions">
            {style.regions.map((region, index) => (
              <li key={`${region.x}-${region.y}-${index}`}>
                {region.kind} — {(region.width * 100).toFixed(0)}% × {(region.height * 100).toFixed(0)}%
                {region.textColor ? ` · ${region.textColor}` : ""}
              </li>
            ))}
          </ul>
        )}
      </section>
    </ExperienceSurface>
  );
}
