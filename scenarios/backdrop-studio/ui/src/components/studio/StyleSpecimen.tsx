import { useState } from "react";
import { Link } from "react-router-dom";

import {
  qualityTierOf,
  rendersOnRequest,
  representativeSurface,
  type Style,
  type Surface,
} from "../../api/studio";
import { qualityTierString } from "../../consts/qualityTier";
import { strings } from "../../consts/strings";
import { useRender, useObjectURL } from "../../hooks/useStudio";
import { useTranslation } from "../../i18n";
import { Button } from "@vrooli/react-component-library/Button/2";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";

/**
 * One style, shown as what it looks like.
 *
 * The tile renders a real specimen through the real render path rather than a
 * CSS gradient standing in for one. That is the whole difference between a
 * catalog and a list of names: every style here is an implicit claim about what
 * good looks like, and a claim nobody can see is not reviewable.
 *
 * It renders at the style's representative surface — the landing-page hero when
 * the style permits it — because a hero judged cropped to a phone is being
 * judged for something it never claimed to be.
 *
 * A model-backed style is the exception to rendering on sight. Its tile shows a
 * specimen already rendered this session if there is one, and otherwise waits
 * for the operator to ask: opening the catalog must not start minutes of
 * diffusion or a metered charge per tile.
 */
export function StyleSpecimen({
  style,
  surfaces,
  seed,
}: {
  style: Style;
  surfaces: Surface[];
  seed: bigint;
}) {
  const { t } = useTranslation();
  const surface = representativeSurface(style, surfaces);
  const tier = qualityTierOf(style);
  const request = surface
    ? { styleId: style.id, surfaceId: surface.id, placement: style.placements[0], seed }
    : null;
  const onRequest = rendersOnRequest(style);
  const [requested, setRequested] = useState(false);
  const render = useRender(request, { enabled: !onRequest || requested, retain: onRequest });
  const candidate = render.data?.candidates[0];
  const awaitingRequest = onRequest && !render.data && !render.isFetching && !render.isError;
  const imageURL = useObjectURL(candidate);

  const aspect = surface ? `${surface.width} / ${surface.height}` : "16 / 9";

  return (
    <article
      className="flex flex-col gap-3 rounded-panel border border-app-border bg-app-surface p-3"
      data-testid={`style-specimen-${style.id}`}
      data-style-id={style.id}
      aria-labelledby={`${style.id}-name`}
    >
      <div
        className="overflow-hidden rounded-control border border-app-border bg-app-surface-muted"
        style={{ aspectRatio: aspect }}
      >
        {imageURL ? (
          <img
            src={imageURL}
            alt={t(strings.pages.catalog.specimenAlt, { style: style.name })}
            className="h-full w-full object-cover"
            loading="lazy"
          />
        ) : (
          // The placeholder holds the surface's aspect so the grid does not
          // reflow as specimens arrive — a catalog that jumps while it loads is
          // unusable for exactly the comparison it exists to support.
          <div
            role="status"
            aria-live="polite"
            className="flex h-full w-full flex-col items-center justify-center gap-2 px-3 text-center text-xs text-app-muted-foreground"
          >
            {awaitingRequest ? (
              <>
                <span>{t(strings.pages.catalog.specimenOnRequest)}</span>
                <Button
                  variant="secondary"
                  size="sm"
                  disabled={!request}
                  onClick={() => setRequested(true)}
                  aria-label={t(strings.pages.catalog.renderSpecimenFor, { style: style.name })}
                >
                  {t(strings.pages.catalog.renderSpecimen)}
                </Button>
              </>
            ) : render.isError ? (
              <>
                <span>{t(strings.pages.catalog.specimenUnavailable)}</span>
                {onRequest ? (
                  <Button variant="secondary" size="sm" onClick={() => void render.refetch()}>
                    {t(strings.pages.catalog.retry)}
                  </Button>
                ) : null}
              </>
            ) : onRequest ? (
              t(strings.pages.catalog.specimenRenderingModel)
            ) : (
              t(strings.pages.catalog.specimenLoading)
            )}
          </div>
        )}
      </div>

      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <h3 id={`${style.id}-name`} className="truncate font-semibold">
            <Link to={`/styles/${style.id}`} className="hover:underline">
              {style.name}
            </Link>
          </h3>
          <p className="truncate text-xs text-app-muted-foreground">
            {style.subject} · {style.lineage}
          </p>
        </div>
        <StatusBadge>
          {t(qualityTierString(tier))}
        </StatusBadge>
      </div>

      <p className="text-xs text-app-muted-foreground">
        {style.treatments.length > 0
          ? style.treatments.join(" → ")
          : t(strings.pages.catalog.untreated)}
      </p>
    </article>
  );
}
