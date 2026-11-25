import { Link } from "react-router-dom";
import { ExternalLink } from "lucide-react";

import { useTranslation } from "../../i18n";
import { appRoutes } from "../../routes";
import type { CatalogAsset, WorkbenchStory } from "../data";
import { cx } from "../cx";
import { Chip } from "../primitives";
import { FixtureFrame, type FixtureStatus } from "./FixtureFrame";
import { fixtureViewportHeight, fixtureViewportWidth, type FixturePaint } from "./fixtureBench";
import { fixtureTheme, type FixtureSurface } from "./fixtureTheme";

function previewPath(asset: CatalogAsset, story: WorkbenchStory): string {
  return `${appRoutes.preview.replace(":id", encodeURIComponent(asset.id))}?story=${encodeURIComponent(story.id)}`;
}

type Translate = (key: string, options?: Record<string, unknown>) => string;

function statusLabel(status: FixtureStatus, t: Translate): string {
  switch (status) {
    case "loading":
      return t("workbench.component.stateLoading", { defaultValue: "loading" });
    case "error":
      return t("workbench.component.stateError", { defaultValue: "failed" });
    case "applied":
      return t("workbench.component.stateApplied", { defaultValue: "edited" });
    default:
      return t("workbench.component.stateReady", { defaultValue: "auto" });
  }
}

export function FixtureTile({
  asset,
  story,
  version,
  sourceRevision,
  selected,
  stage = false,
  override,
  reloadKey,
  status,
  paint,
  surface,
  measured,
  message,
  onSelect,
  onStatus,
  onMeasure,
  onRetry,
}: {
  asset: CatalogAsset;
  story: WorkbenchStory;
  version: string;
  sourceRevision: string;
  selected: boolean;
  stage?: boolean;
  override: Record<string, unknown> | null;
  reloadKey: number;
  status: FixtureStatus;
  paint: FixturePaint;
  surface: FixtureSurface;
  measured: boolean;
  message?: string;
  onSelect: (story: WorkbenchStory) => void;
  onStatus: (storyId: string, status: FixtureStatus, message?: string) => void;
  onMeasure: (storyId: string, paint: FixturePaint) => void;
  onRetry: (storyId: string) => void;
}) {
  const { t } = useTranslation();
  const tr = t as unknown as Translate;
  const theme = fixtureTheme(surface);
  const role = story.role || "story";
  const label = story.name || story.id;
  return (
    <article
      className={cx("wb-bench-tile", stage && "is-stage", selected && "is-selected")}
      data-story-id={story.id}
      data-selected={selected ? "true" : undefined}
      aria-current={selected ? "true" : undefined}
    >
      <div className="wb-bench-tile-head">
        <button
          type="button"
          className="wb-bench-select"
          onClick={() => onSelect(story)}
          aria-label={t("workbench.component.selectFixture", {
            defaultValue: "Select {{name}}",
            name: label,
          })}
        >
          <span className="wb-bench-name">{label}</span>
          <span className={`wb-bench-role is-${role}`}>{role}</span>
        </button>
        <Link
          className="wb-bench-open"
          to={previewPath(asset, story)}
          target="_blank"
          rel="noreferrer"
          aria-label={t("workbench.component.openFixture", {
            defaultValue: "Open {{name}} in the preview",
            name: label,
          })}
          title={t("workbench.component.openFixture", {
            defaultValue: "Open {{name}} in the preview",
            name: label,
          })}
        >
          <ExternalLink aria-hidden size={13} />
        </Link>
      </div>
      <div className="wb-bench-surface" style={{ width: paint.width, height: paint.height }}>
        {status === "loading" && !measured ? (
          <span className="wb-bench-paint-status" role="status">
            {t("workbench.component.fixtureLoading", { defaultValue: "Loading specimen…" })}
          </span>
        ) : null}
        {status === "error" ? (
          <span className="wb-bench-paint-status is-error" role="alert">
            {message ||
              t("workbench.component.fixtureError", {
                defaultValue: "The specimen did not render.",
              })}
            <button type="button" data-testid="bench-retry" onClick={() => onRetry(story.id)}>
              {t("workbench.component.fixtureRetry", { defaultValue: "Try again" })}
            </button>
          </span>
        ) : null}
        <div
          className="wb-bench-paint"
          style={{ transform: `translate(${-paint.x}px, ${-paint.y}px)` }}
        >
          <FixtureFrame
            assetId={asset.id}
            storyId={story.id}
            version={version}
            sourceRevision={sourceRevision}
            override={override}
            reloadKey={reloadKey}
            frameWidth={fixtureViewportWidth(story, asset)}
            frameHeight={Math.max(fixtureViewportHeight(story), paint.height)}
            theme={theme}
            hideState
            title={t("workbench.component.fixtureTitle", {
              defaultValue: "{{name}} specimen",
              name: label,
            })}
            onStatus={(next, message) => onStatus(story.id, next, message)}
            onMeasure={(next) => onMeasure(story.id, next)}
          />
        </div>
      </div>
      <div className="wb-bench-meta">
        <span className="wb-bench-size" data-testid="bench-size">{`${paint.width} × ${paint.height}px`}</span>
        <Chip tone={status === "error" ? "red" : status === "applied" ? "green" : "ice"}>
          {statusLabel(status, tr)}
        </Chip>
      </div>
    </article>
  );
}
