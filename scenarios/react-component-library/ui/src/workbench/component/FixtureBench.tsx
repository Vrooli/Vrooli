import { useEffect, useMemo, useState } from "react";

import { useTranslation } from "../../i18n";
import type { CatalogAsset, WorkbenchStory } from "../data";
import { cx } from "../cx";
import { Empty } from "../primitives";
import type { FixtureStatus } from "./FixtureFrame";
import {
  buildFixtureBench,
  initialPaint,
  type FixturePaint,
} from "./fixtureBench";
import { FixtureTile } from "./FixtureTile";
import type { FixtureSurface } from "./fixtureTheme";

export interface FixtureBenchProps {
  asset: CatalogAsset;
  stories: WorkbenchStory[];
  version: string;
  sourceRevision?: string;
  selectedId: string | null;
  onSelect: (story: WorkbenchStory) => void;
  overrides: Record<string, Record<string, unknown> | null>;
  reloads: Record<string, number>;
  statuses: Record<string, FixtureStatus>;
  onStatus: (storyId: string, status: FixtureStatus, message?: string) => void;
  onRetry: (storyId: string) => void;
}

export function FixtureBench({
  asset,
  stories,
  version,
  sourceRevision = "",
  selectedId,
  onSelect,
  overrides,
  reloads,
  statuses,
  onStatus,
  onRetry,
}: FixtureBenchProps) {
  const { t } = useTranslation();
  const [surface, setSurface] = useState<FixtureSurface>("dark");
  const [showHidden, setShowHidden] = useState(false);
  const [measurements, setMeasurements] = useState<Record<string, FixturePaint>>({});
  const [messages, setMessages] = useState<Record<string, string>>({});
  const model = useMemo(() => buildFixtureBench(stories), [stories]);
  const selectedIsHidden = Boolean(selectedId && model.hidden.some((story) => story.id === selectedId));

  useEffect(() => {
    if (selectedIsHidden) setShowHidden(true);
  }, [selectedIsHidden]);

  if (stories.length === 0) {
    return (
      <div className="wb-bench-wrap" data-testid="bench-empty">
        <Empty title={t("workbench.component.noStories", { defaultValue: "No declared stories" })}>
          {t("workbench.component.noStoriesBody", {
            defaultValue:
              "This asset has no story contract yet. Add a story before claiming it renders in isolation.",
          })}
        </Empty>
      </div>
    );
  }

  const paintFor = (story: WorkbenchStory) =>
    measurements[story.id] ?? initialPaint(story, asset);

  const recordMeasure = (storyId: string, next: FixturePaint) => {
    setMeasurements((current) => {
      const previous = current[storyId];
      if (
        previous &&
        previous.x === next.x &&
        previous.y === next.y &&
        previous.width === next.width &&
        previous.height === next.height
      ) {
        return current;
      }
      return { ...current, [storyId]: next };
    });
  };

  const renderTile = (story: WorkbenchStory, stage = false) => (
    <FixtureTile
      key={story.id}
      asset={asset}
      story={story}
      version={version}
      sourceRevision={sourceRevision}
      selected={selectedId === story.id}
      stage={stage}
      override={overrides[story.id] ?? null}
      reloadKey={reloads[story.id] ?? 0}
      status={statuses[story.id] ?? "loading"}
      paint={paintFor(story)}
      surface={surface}
      measured={Boolean(measurements[story.id])}
      message={messages[story.id]}
      onSelect={onSelect}
      onStatus={(storyId, status, message) => {
        setMessages((current) =>
          message ? { ...current, [storyId]: message } : { ...current, [storyId]: "" },
        );
        onStatus(storyId, status, message);
      }}
      onMeasure={recordMeasure}
      onRetry={onRetry}
    />
  );

  const visibleStories = [
    ...(model.anatomy ? [model.anatomy] : []),
    ...model.groups.flatMap((group) => group.stories),
    ...(showHidden ? model.hidden : []),
  ];
  const allReady = visibleStories.every((story) => {
    const status = statuses[story.id];
    return status === "ready" || status === "applied";
  });

  return (
    <div className="wb-bench-wrap">
      <div
        className="wb-bench"
        data-fixtures-ready={allReady ? "true" : undefined}
        data-fixture-surface={surface}
      >
        <div className="wb-bench-intro">
          <p>
            {t("workbench.component.benchSummary", {
              defaultValue:
                "{{visible}} frames grouped by the question they answer. {{hidden}} redundant frames are hidden.",
              visible: model.visibleCount,
              hidden: model.hidden.length,
            })}
          </p>
          <div
            className="wb-bench-surface-toggle"
            role="group"
            aria-label={t("workbench.component.fixtureSurface", {
              defaultValue: "Fixture surface",
            })}
          >
            <button
              type="button"
              className={cx(surface === "dark" && "is-on")}
              aria-pressed={surface === "dark"}
              data-testid="fixture-surface-dark"
              onClick={() => setSurface("dark")}
            >
              {t("workbench.component.surfaceDark", { defaultValue: "Dark" })}
            </button>
            <button
              type="button"
              className={cx(surface === "light" && "is-on")}
              aria-pressed={surface === "light"}
              data-testid="fixture-surface-light"
              onClick={() => setSurface("light")}
            >
              {t("workbench.component.surfaceLight", { defaultValue: "Light" })}
            </button>
          </div>
        </div>

        {model.anatomy ? (
          <section className="wb-bench-section" aria-labelledby="wb-bench-anatomy">
            <div className="wb-bench-section-head">
              <h2 id="wb-bench-anatomy">
                {t("workbench.component.anatomyFrames", { defaultValue: "Anatomy" })}
              </h2>
              <p>{t("workbench.component.anatomyHint", { defaultValue: "one stable subject" })}</p>
            </div>
            <div className="wb-bench-stage">{renderTile(model.anatomy, true)}</div>
          </section>
        ) : (
          <p className="wb-bench-note" role="status">
            {t("workbench.component.noAnatomy", {
              defaultValue: "This contract has no anatomy frame. The axes and boundaries are below.",
            })}
          </p>
        )}

        {model.groups.map((group) => (
          <section
            key={group.id}
            className="wb-bench-section"
            aria-labelledby={`wb-bench-${group.id}`}
          >
            <div className="wb-bench-section-head">
              <h2 id={`wb-bench-${group.id}`}>{group.label}</h2>
              <p>
                {group.role === "axis"
                  ? t("workbench.component.axisHint", {
                      defaultValue: "axis · {{count}} frames · {{values}}",
                      count: group.stories.length,
                      values: group.values?.join(" / ") ?? "",
                    })
                  : t("workbench.component.boundaryHint", {
                      defaultValue: "{{count}} boundary frames",
                      count: group.stories.length,
                    })}
              </p>
            </div>
            <div className="wb-bench-grid">{group.stories.map((story) => renderTile(story))}</div>
          </section>
        ))}

        {model.hidden.length > 0 ? (
          <section className="wb-bench-hidden">
            <button
              type="button"
              className="wb-bench-toggle"
              aria-expanded={showHidden}
              onClick={() => setShowHidden((current) => !current)}
            >
              {showHidden
                ? t("workbench.component.hideRedundant", {
                    defaultValue: "Hide {{count}} redundant frames",
                    count: model.hidden.length,
                  })
                : t("workbench.component.showRedundant", {
                    defaultValue: "Show {{count}} redundant frames",
                    count: model.hidden.length,
                  })}
            </button>
            <p>
              {t("workbench.component.redundantHint", {
                defaultValue:
                  "These frames mount the same specimen and args as a frame above. They stay available so no state evidence is lost.",
              })}
            </p>
            {showHidden ? (
              <div className="wb-bench-grid is-redundant">
                {model.hidden.map((story) => renderTile(story))}
              </div>
            ) : null}
          </section>
        ) : null}
      </div>
    </div>
  );
}
