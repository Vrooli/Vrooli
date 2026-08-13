import { useState } from "react";
import { useSearchParams } from "react-router-dom";

import { representativeSurface, type Style, type Surface } from "../api/studio";
import { ExperienceSurface } from "../components/experience/ExperienceSurface";
import { EmptyState } from "../components/ui/empty-state";
import { StatusBadge } from "../components/ui/status-badge";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useObjectURL, useRender, useStyles, useSurfaces } from "../hooks/useStudio";
import { useTranslation } from "../i18n";

/**
 * One style across a seed range.
 *
 * A seed is the whole difference between two renders of the same style, so a
 * grid of seeds is the only comparison that isolates *composition* from art
 * direction. It is also the cheapest way to answer the question an operator
 * actually has — "is this style good, or did I get a good seed?" — which a
 * single specimen cannot.
 */
function SweepCell({
  style,
  surface,
  seed,
  selected,
  onSelect,
}: {
  style: Style;
  surface: Surface;
  seed: bigint;
  selected: boolean;
  onSelect: () => void;
}) {
  const { t } = useTranslation();
  const render = useRender({
    styleId: style.id,
    surfaceId: surface.id,
    placement: style.placements[0],
    seed,
  });
  const imageURL = useObjectURL(render.data?.candidates[0]);

  return (
    <figure
      className={`flex flex-col gap-2 rounded-panel border p-2 ${selected ? "border-app-primary" : "border-app-border"}`}
      data-testid={`sweep-cell-${seed.toString()}`}
      data-selected={selected ? "true" : "false"}
    >
      <div
        className="overflow-hidden rounded-control bg-app-surface-muted"
        style={{ aspectRatio: `${surface.width} / ${surface.height}` }}
      >
        {imageURL ? (
          <img
            src={imageURL}
            alt={t(strings.pages.catalog.specimenAlt, { style: `${style.name} · ${seed}` })}
            className="h-full w-full object-cover"
          />
        ) : (
          <div role="status" className="flex h-full items-center justify-center text-xs text-app-muted-foreground">
            {render.isError
              ? t(strings.pages.catalog.specimenUnavailable)
              : t(strings.pages.sweep.loading)}
          </div>
        )}
      </div>
      <figcaption className="flex items-center justify-between gap-2 text-xs">
        <span className="font-mono">{seed.toString()}</span>
        <button
          type="button"
          className="min-h-11 rounded-control border border-app-border px-3 font-medium"
          onClick={onSelect}
        >
          {selected ? t(strings.pages.sweep.selected) : t(strings.pages.sweep.select)}
        </button>
      </figcaption>
    </figure>
  );
}

export function SweepPage() {
  const { t } = useTranslation();
  const [params, setParams] = useSearchParams();
  const styles = useStyles({});
  const surfaces = useSurfaces();

  const styleId = params.get("style") ?? "";
  const [start, setStart] = useState(7);
  const [count, setCount] = useState(6);
  const [selected, setSelected] = useState<string | null>(null);

  const style = styles.data?.find((s) => s.id === styleId);
  const surface = style ? representativeSurface(style, surfaces.data ?? []) : undefined;
  const seeds = Array.from({ length: count }, (_, i) => BigInt(start + i));

  return (
    <ExperienceSurface
      surfaceId="sweep"
      state={styles.isLoading ? "loading" : styles.isError ? "error" : style ? "ready" : "empty"}
      statusMessage={styles.isLoading ? t(strings.pages.common.loading) : undefined}
      data-testid={selectors.pages.sweep}
      aria-labelledby="sweep-heading"
      className="flex flex-col gap-6"
    >
      <header className="flex flex-col gap-2">
        <h2 id="sweep-heading" className="text-2xl font-semibold">
          {t(strings.pages.sweep.title)}
        </h2>
        <p className="max-w-3xl text-app-muted-foreground">{t(strings.pages.sweep.description)}</p>
      </header>

      <div className="flex flex-wrap items-end gap-3">
        <label className="flex flex-col gap-1 text-sm font-medium" htmlFor="sweep-style">
          {t(strings.pages.sweep.styleLabel)}
          <select
            id="sweep-style"
            className="min-h-11 min-w-56 rounded-control border border-app-border bg-app-surface px-3"
            value={styleId}
            data-testid="sweep-style-select"
            onChange={(event) => setParams({ style: event.target.value })}
          >
            <option value="">{t(strings.pages.catalog.axisAny)}</option>
            {(styles.data ?? []).map((option) => (
              <option key={option.id} value={option.id}>
                {option.name}
              </option>
            ))}
          </select>
        </label>
        <label className="flex flex-col gap-1 text-sm font-medium" htmlFor="sweep-seed">
          {t(strings.pages.sweep.seedStart)}
          <input
            id="sweep-seed"
            type="number"
            className="min-h-11 w-28 rounded-control border border-app-border bg-app-surface px-3"
            value={start}
            onChange={(event) => setStart(Number(event.target.value) || 0)}
          />
        </label>
        <label className="flex flex-col gap-1 text-sm font-medium" htmlFor="sweep-count">
          {t(strings.pages.sweep.count)}
          <input
            id="sweep-count"
            type="number"
            min={1}
            max={12}
            className="min-h-11 w-24 rounded-control border border-app-border bg-app-surface px-3"
            value={count}
            onChange={(event) => setCount(Math.min(12, Math.max(1, Number(event.target.value) || 1)))}
          />
        </label>
        {selected ? <StatusBadge>{`${t(strings.pages.sweep.selected)} · ${selected}`}</StatusBadge> : null}
      </div>

      {!style || !surface ? (
        <EmptyState title={t(strings.pages.sweep.title)} description={t(strings.pages.sweep.empty)} />
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3" data-testid="sweep-grid">
          {seeds.map((seed) => (
            <SweepCell
              key={seed.toString()}
              style={style}
              surface={surface}
              seed={seed}
              selected={selected === seed.toString()}
              onSelect={() => setSelected(seed.toString())}
            />
          ))}
        </div>
      )}
    </ExperienceSurface>
  );
}
