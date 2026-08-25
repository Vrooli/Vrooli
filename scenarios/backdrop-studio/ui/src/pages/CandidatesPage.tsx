import { useState } from "react";

import { parseQuality, representativeSurface, type Candidate } from "../api/studio";
import { ExperienceSurface } from "../components/experience/ExperienceSurface";
import { QualityMeters } from "../components/studio/QualityMeters";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1.1.0";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useObjectURL, useRender, useStyles, useSurfaces } from "../hooks/useStudio";
import { useTranslation } from "../i18n";

/**
 * Candidates from one submission, with the verdict each one carries.
 *
 * Candidates are per-render rather than a durable collection — the render store
 * holds them in memory for the process that made them — so this page renders a
 * batch and shows what came back, rather than pretending to list a history the
 * API does not keep. Saying so is better than an empty table that looks broken.
 */
export function CandidatesPage() {
  const { t } = useTranslation();
  const styles = useStyles({});
  const surfaces = useSurfaces();
  const [styleId, setStyleId] = useState("");
  const [count, setCount] = useState(3);

  const style = styles.data?.find((candidate) => candidate.id === styleId);
  const surface = style ? representativeSurface(style, surfaces.data ?? []) : undefined;
  const render = useRender(
    style && surface
      ? {
          styleId: style.id,
          surfaceId: surface.id,
          placement: style.placements[0],
          seed: 7n,
          candidateCount: count,
        }
      : null,
  );

  return (
    <ExperienceSurface
      surfaceId="candidates"
      state={render.isLoading ? "loading" : render.isError ? "error" : style ? "ready" : "empty"}
      statusMessage={render.isLoading ? t(strings.pages.candidates.loading) : undefined}
      data-testid={selectors.pages.candidates}
      aria-labelledby="candidates-heading"
      className="flex flex-col gap-6"
    >
      <header className="flex flex-col gap-2">
        <h2 id="candidates-heading" className="text-2xl font-semibold">
          {t(strings.pages.candidates.title)}
        </h2>
        <p className="max-w-3xl text-app-muted-foreground">
          {t(strings.pages.candidates.description)}
        </p>
      </header>

      <div className="flex flex-wrap items-end gap-3">
        <label className="flex flex-col gap-1 text-sm font-medium" htmlFor="candidates-style">
          {t(strings.pages.sweep.styleLabel)}
          <select
            id="candidates-style"
            className="min-h-11 min-w-56 rounded-control border border-app-border bg-app-surface px-3"
            value={styleId}
            data-testid="candidates-style-select"
            onChange={(event) => setStyleId(event.target.value)}
          >
            <option value="">{t(strings.pages.catalog.axisAny)}</option>
            {(styles.data ?? []).map((option) => (
              <option key={option.id} value={option.id}>
                {option.name}
              </option>
            ))}
          </select>
        </label>
        <label className="flex flex-col gap-1 text-sm font-medium" htmlFor="candidates-count">
          {t(strings.pages.sweep.count)}
          <input
            id="candidates-count"
            type="number"
            min={1}
            max={8}
            className="min-h-11 w-24 rounded-control border border-app-border bg-app-surface px-3"
            value={count}
            onChange={(event) => setCount(Math.min(8, Math.max(1, Number(event.target.value) || 1)))}
          />
        </label>
      </div>

      {!style ? (
        <EmptyState
          title={t(strings.pages.candidates.title)}
          description={t(strings.pages.candidates.empty)}
        />
      ) : (
        <div className="grid gap-4 xl:grid-cols-2" data-testid="candidates-grid">
          {(render.data?.candidates ?? []).map((candidate, index) => (
            <CandidateCard key={candidate.id} index={index} candidate={candidate} styleName={style.name} />
          ))}
        </div>
      )}
    </ExperienceSurface>
  );
}

function CandidateCard({
  candidate,
  index,
  styleName,
}: {
  candidate: Candidate;
  index: number;
  styleName: string;
}) {
  const { t } = useTranslation();
  const imageURL = useObjectURL(candidate);
  const quality = parseQuality(candidate);
  return (
    <article
      className="flex flex-col gap-3 rounded-panel border border-app-border p-3"
      data-testid={`candidate-${index}`}
    >
      {imageURL ? (
        <img
          src={imageURL}
          alt={t(strings.pages.catalog.specimenAlt, { style: `${styleName} · ${candidate.seed}` })}
          className="w-full rounded-control"
        />
      ) : null}
      <p className="font-mono text-xs text-app-muted-foreground">
        {`${candidate.width}×${candidate.height} · ${t(strings.pages.common.seedLabel)} ${candidate.seed.toString()}`}
      </p>
      <QualityMeters verdict={quality} />
    </article>
  );
}
