import { useState } from "react";
import { useSearchParams } from "react-router-dom";

import { permittedSurfaces } from "../api/studio";
import { ExperienceSurface } from "../components/experience/ExperienceSurface";
import { MockupPreview } from "../components/studio/MockupPreview";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useObjectURL, useRender, useStyles, useSurfaces } from "../hooks/useStudio";
import { useTranslation } from "../i18n";

/**
 * One style at each placement it declares.
 *
 * A placement is not a crop — it is a claim about where copy sits and what the
 * image has to leave alone. Showing them together is the only way to see that a
 * style working full-bleed can fail in a split panel, which is a per-placement
 * property rather than a property of the style.
 */
export function PlacementsPage() {
  const { t } = useTranslation();
  const [params, setParams] = useSearchParams();
  const styles = useStyles({});
  const surfaces = useSurfaces();
  const styleId = params.get("style") ?? "";
  const [seed] = useState(7n);

  const style = styles.data?.find((candidate) => candidate.id === styleId);
  const permitted = style ? permittedSurfaces(style, surfaces.data ?? []) : [];

  return (
    <ExperienceSurface
      surfaceId="placements"
      state={styles.isLoading ? "loading" : styles.isError ? "error" : style ? "ready" : "empty"}
      statusMessage={styles.isLoading ? t(strings.pages.placements.loading) : undefined}
      data-testid={selectors.pages.placements}
      aria-labelledby="placements-heading"
      className="flex flex-col gap-6"
    >
      <header className="flex flex-col gap-2">
        <h2 id="placements-heading" className="text-2xl font-semibold">
          {t(strings.pages.placements.title)}
        </h2>
        <p className="max-w-3xl text-app-muted-foreground">
          {t(strings.pages.placements.description)}
        </p>
      </header>

      <label className="flex w-fit flex-col gap-1 text-sm font-medium" htmlFor="placements-style">
        {t(strings.pages.sweep.styleLabel)}
        <select
          id="placements-style"
          className="min-h-11 min-w-56 rounded-control border border-app-border bg-app-surface px-3"
          value={styleId}
          data-testid="placements-style-select"
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

      {!style ? (
        <EmptyState
          title={t(strings.pages.placements.title)}
          description={t(strings.pages.placements.empty)}
        />
      ) : (
        <div className="grid gap-6" data-testid="placements-grid">
          {style.placements.map((placement) => (
            <PlacementRow
              key={placement}
              placement={placement}
              styleId={style.id}
              surfaceId={permitted[0]?.id ?? ""}
              seed={seed}
              style={style}
            />
          ))}
        </div>
      )}
    </ExperienceSurface>
  );
}

function PlacementRow({
  placement,
  styleId,
  surfaceId,
  seed,
  style,
}: {
  placement: string;
  styleId: string;
  surfaceId: string;
  seed: bigint;
  style: Parameters<typeof MockupPreview>[0]["style"];
}) {
  const { t } = useTranslation();
  const render = useRender(surfaceId ? { styleId, surfaceId, placement, seed } : null);
  const imageURL = useObjectURL(render.data?.candidates[0]);
  return (
    <section className="flex flex-col gap-2" data-testid={`placement-${placement}`}>
      <div className="flex items-center gap-2">
        <h3 className="text-lg font-semibold">{placement}</h3>
        {render.isError ? <StatusBadge>{t(strings.pages.common.renderFailed)}</StatusBadge> : null}
      </div>
      <MockupPreview
        imageURL={imageURL}
        style={style}
        kind={placement.includes("device") || placement === "feature_graphic" ? "store" : "landing"}
      />
    </section>
  );
}
