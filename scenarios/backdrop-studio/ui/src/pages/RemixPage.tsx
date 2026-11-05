import { useState } from "react";
import { useSearchParams } from "react-router-dom";

import { createStyle, representativeSurface, type Style } from "../api/studio";
import { ExperienceSurface } from "../components/experience/ExperienceSurface";
import { Button } from "@vrooli/react-component-library/Button/2";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { Input } from "../components/ui/input";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { axisValues, useObjectURL, useRender, useStyles, useSurfaces } from "../hooks/useStudio";
import { useTranslation } from "../i18n";

/** The axes a fork may change. One at a time, which is what makes it a comparison. */
const FORKABLE = ["lineage", "subject", "role"] as const;
type ForkAxis = (typeof FORKABLE)[number];

function Preview({ style, heading }: { style: Style; heading: string }) {
  const { t } = useTranslation();
  const surfaces = useSurfaces();
  const surface = representativeSurface(style, surfaces.data ?? []);
  const render = useRender(
    surface
      ? { styleId: style.id, surfaceId: surface.id, placement: style.placements[0], seed: 7n }
      : null,
  );
  const imageURL = useObjectURL(render.data?.candidates[0]);
  return (
    <figure className="flex flex-col gap-2" data-testid={`remix-preview-${style.id}`}>
      <figcaption className="text-xs font-medium uppercase tracking-wide text-app-muted-foreground">
        {heading} — {style.id}
      </figcaption>
      <div
        className="overflow-hidden rounded-panel border border-app-border bg-app-surface-muted"
        style={{ aspectRatio: surface ? `${surface.width} / ${surface.height}` : "2 / 1" }}
      >
        {imageURL ? (
          <img
            src={imageURL}
            alt={t(strings.pages.catalog.specimenAlt, { style: style.name })}
            className="h-full w-full object-cover"
          />
        ) : (
          <div role="status" className="flex h-full items-center justify-center text-xs text-app-muted-foreground">
            {render.isError
              ? t(strings.pages.catalog.specimenUnavailable)
              : t(strings.pages.catalog.specimenLoading)}
          </div>
        )}
      </div>
    </figure>
  );
}

/**
 * Fork a style, change one axis, and see the result beside its parent.
 *
 * Only one axis at a time, and the axes offered are the classification ones —
 * not the treatment chain. That is a deliberate narrowing rather than a
 * shortcut: a fork that changed three things at once answers no question, and
 * an arbitrary parameter editor would let an operator author a style the wire
 * contract rejects, which is the failure the catalog's write-time validation
 * exists to prevent. Changing the chain is a catalog authoring job.
 */
export function RemixPage() {
  const { t } = useTranslation();
  const [params] = useSearchParams();
  const styles = useStyles({});
  const parentId = params.get("parent") ?? "";

  const [axis, setAxis] = useState<ForkAxis>("lineage");
  const [value, setValue] = useState("");
  const [newId, setNewId] = useState("");
  const [saved, setSaved] = useState<string | null>(null);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [fork, setFork] = useState<Style | null>(null);

  const parent = styles.data?.find((style) => style.id === parentId);
  const values = axisValues(styles.data ?? []);
  const options = axis === "lineage" ? values.lineage : axis === "subject" ? values.subject : values.role;

  const buildFork = (): Style | null => {
    if (!parent || !value || !newId) {
      return null;
    }
    return { ...parent, id: newId, name: `${parent.name} (${value})`, parentId: parent.id, [axis]: value };
  };

  return (
    <ExperienceSurface
      surfaceId="compose"
      state={styles.isLoading ? "loading" : styles.isError ? "error" : parent ? "ready" : "empty"}
      statusMessage={styles.isLoading ? t(strings.pages.common.loading) : undefined}
      data-testid={selectors.pages.remix}
      aria-labelledby="remix-heading"
      className="flex flex-col gap-6"
    >
      <header className="flex flex-col gap-2">
        <h2 id="remix-heading" className="text-2xl font-semibold">
          {t(strings.pages.remix.title)}
        </h2>
        <p className="max-w-3xl text-app-muted-foreground">{t(strings.pages.remix.description)}</p>
      </header>

      {!parent ? (
        <EmptyState title={t(strings.pages.remix.title)} description={t(strings.pages.remix.chooseStyle)} />
      ) : (
        <>
          <div className="flex flex-wrap items-end gap-3">
            <p className="text-sm">
              {t(strings.pages.remix.parentLabel)}: <span className="font-mono">{parent.id}</span>
            </p>
            <label className="flex flex-col gap-1 text-sm font-medium" htmlFor="remix-axis">
              {t(strings.pages.remix.axisLabel)}
              <select
                id="remix-axis"
                className="min-h-11 rounded-control border border-app-border bg-app-surface px-3"
                value={axis}
                data-testid="remix-axis-select"
                onChange={(event) => {
                  setAxis(event.target.value as ForkAxis);
                  setValue("");
                }}
              >
                {FORKABLE.map((option) => (
                  <option key={option} value={option}>
                    {option}
                  </option>
                ))}
              </select>
            </label>
            <label className="flex flex-col gap-1 text-sm font-medium" htmlFor="remix-value">
              {t(strings.pages.remix.valueLabel)}
              <select
                id="remix-value"
                className="min-h-11 min-w-48 rounded-control border border-app-border bg-app-surface px-3"
                value={value}
                data-testid="remix-value-select"
                onChange={(event) => setValue(event.target.value)}
              >
                <option value="">{t(strings.pages.catalog.axisAny)}</option>
                {options
                  .filter((option) => option !== (parent as unknown as Record<string, string>)[axis])
                  .map((option) => (
                    <option key={option} value={option}>
                      {option}
                    </option>
                  ))}
              </select>
            </label>
            <label className="flex flex-col gap-1 text-sm font-medium" htmlFor="remix-id">
              {t(strings.pages.remix.newIdLabel)}
              <Input
                id="remix-id"
                value={newId}
                data-testid="remix-id-input"
                onChange={(event) => setNewId(event.target.value)}
              />
            </label>
            <Button
              variant="secondary"
              disabled={!value || !newId}
              onClick={() => setFork(buildFork())}
            >
              {t(strings.pages.remix.renderBoth)}
            </Button>
            <Button
              disabled={!fork}
              onClick={() => {
                if (!fork) return;
                setSaveError(null);
                // Deliberately not an async handler: React expects a void
                // return here, and an async one hands it a promise nobody
                // awaits — so a rejection becomes an unhandled rejection rather
                // than the error message this page is supposed to show.
                void createStyle(fork)
                  .then((created) => setSaved(created.id))
                  .catch((error: unknown) =>
                    setSaveError(error instanceof Error ? error.message : String(error)),
                  );
              }}
            >
              {t(strings.pages.remix.save)}
            </Button>
            {saved ? <StatusBadge>{`${t(strings.pages.remix.saved)} · ${saved}`}</StatusBadge> : null}
          </div>

          {saveError ? (
            <p role="alert" className="rounded-panel border border-app-border p-3 text-sm">
              {t(strings.pages.remix.saveError)} {saveError}
            </p>
          ) : null}

          <div className="grid gap-4 xl:grid-cols-2" data-testid="remix-comparison">
            <Preview style={parent} heading={t(strings.pages.remix.parentHeading)} />
            {fork ? <Preview style={fork} heading={t(strings.pages.remix.childHeading)} /> : null}
          </div>
        </>
      )}
    </ExperienceSurface>
  );
}
