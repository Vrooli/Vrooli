import { useCallback, useRef, type KeyboardEvent, type PointerEvent } from "react";

import { useTranslation } from "../../i18n";
import type { CatalogAsset, WorkbenchStory } from "../data";
import { cx } from "../cx";
import { Chip, Empty } from "../primitives";
import { FixtureFrame, type FixtureStatus } from "./FixtureFrame";

export interface FixturePosition {
  x: number;
  y: number;
}

export interface FixtureCanvasProps {
  asset: CatalogAsset;
  stories: WorkbenchStory[];
  version: string;
  selectedId: string | null;
  onSelect: (story: WorkbenchStory) => void;
  positions: Record<string, FixturePosition>;
  onMove: (storyId: string, position: FixturePosition) => void;
  overrides: Record<string, Record<string, unknown> | null>;
  reloads: Record<string, number>;
  statuses: Record<string, FixtureStatus>;
  onStatus: (storyId: string, status: FixtureStatus, message?: string) => void;
}

function defaultPosition(index: number): FixturePosition {
  return { x: 28 + (index % 2) * 352, y: 28 + Math.floor(index / 2) * 300 };
}

export function FixtureCanvas({
  asset,
  stories,
  version,
  selectedId,
  onSelect,
  positions,
  onMove,
  overrides,
  reloads,
  statuses,
  onStatus,
}: FixtureCanvasProps) {
  const { t } = useTranslation();
  const drag = useRef<{
    id: string;
    startX: number;
    startY: number;
    originX: number;
    originY: number;
    moved: boolean;
  } | null>(null);

  const beginDrag = useCallback(
    (event: PointerEvent<HTMLDivElement>, story: WorkbenchStory) => {
      if (event.button !== 0) return;
      const current = positions[story.id] ?? defaultPosition(stories.indexOf(story));
      drag.current = {
        id: story.id,
        startX: event.clientX,
        startY: event.clientY,
        originX: current.x,
        originY: current.y,
        moved: false,
      };
      event.currentTarget.setPointerCapture(event.pointerId);
      onSelect(story);
    },
    [onSelect, positions, stories],
  );

  const moveDrag = useCallback(
    (event: PointerEvent<HTMLDivElement>) => {
      const active = drag.current;
      if (!active) return;
      const dx = event.clientX - active.startX;
      const dy = event.clientY - active.startY;
      if (!active.moved && Math.hypot(dx, dy) < 4) return;
      active.moved = true;
      onMove(active.id, {
        x: Math.max(8, active.originX + dx),
        y: Math.max(8, active.originY + dy),
      });
    },
    [onMove],
  );

  const endDrag = useCallback((event: PointerEvent<HTMLDivElement>) => {
    if (!drag.current) return;
    drag.current = null;
    if (event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId);
    }
  }, []);

  const keyboardMove = useCallback(
    (event: KeyboardEvent<HTMLDivElement>, story: WorkbenchStory) => {
      const step = event.shiftKey ? 24 : 8;
      const current = positions[story.id] ?? defaultPosition(stories.indexOf(story));
      const deltas: Record<string, FixturePosition> = {
        ArrowLeft: { x: -step, y: 0 },
        ArrowRight: { x: step, y: 0 },
        ArrowUp: { x: 0, y: -step },
        ArrowDown: { x: 0, y: step },
      };
      const delta = deltas[event.key];
      if (!delta) return;
      event.preventDefault();
      onMove(story.id, {
        x: Math.max(8, current.x + delta.x),
        y: Math.max(8, current.y + delta.y),
      });
    },
    [onMove, positions, stories],
  );

  if (stories.length === 0) {
    return (
      <div className="wb-canvas-wrap">
        <Empty title={t("workbench.component.noStories", { defaultValue: "No declared stories" })}>
          {t("workbench.component.noStoriesBody", {
            defaultValue:
              "This asset has no story contract yet. Add a story before claiming it renders in isolation.",
          })}
        </Empty>
      </div>
    );
  }

  return (
    <div className="wb-canvas-wrap">
      <div className="wb-fixture-canvas">
        {stories.map((story, index) => {
          const position = positions[story.id] ?? defaultPosition(index);
          const selected = selectedId === story.id;
          const status = statuses[story.id] ?? "loading";
          return (
            <div
              key={story.id}
              className={cx("wb-fixture", selected && "is-selected")}
              data-selected={selected ? "true" : undefined}
              style={{ left: position.x, top: position.y }}
              onPointerDown={() => onSelect(story)}
            >
              <div
                className="wb-fixture-head"
                role="button"
                tabIndex={0}
                aria-label={t("workbench.component.fixtureHandle", {
                  defaultValue: "Select and move {{name}}",
                  name: story.name || story.id,
                })}
                onPointerDown={(event) => beginDrag(event, story)}
                onPointerMove={moveDrag}
                onPointerUp={endDrag}
                onPointerCancel={endDrag}
                onKeyDown={(event) => {
                  if (event.key === "Enter" || event.key === " ") {
                    event.preventDefault();
                    onSelect(story);
                    return;
                  }
                  keyboardMove(event, story);
                }}
              >
                <span className="wb-fixture-name">{story.name || story.id}</span>
                <span className="wb-fixture-role">{story.role || "story"}</span>
              </div>
              <div className="wb-fixture-frame">
                <FixtureFrame
                  assetId={asset.id}
                  storyId={story.id}
                  version={version}
                  override={overrides[story.id] ?? null}
                  reloadKey={reloads[story.id] ?? 0}
                  height={220}
                  title={t("workbench.component.fixtureTitle", {
                    defaultValue: "{{name}} specimen",
                    name: story.name || story.id,
                  })}
                  onStatus={(next, message) => onStatus(story.id, next, message)}
                />
              </div>
              <div className="wb-fixture-foot">
                <span>{story.version}</span>
                <Chip tone={status === "error" ? "red" : status === "applied" ? "green" : "ice"}>
                  {status === "loading"
                    ? t("workbench.component.stateLoading", { defaultValue: "loading" })
                    : status === "error"
                      ? t("workbench.component.stateError", { defaultValue: "failed" })
                      : status === "applied"
                        ? t("workbench.component.stateApplied", { defaultValue: "edited" })
                        : t("workbench.component.stateReady", { defaultValue: "auto" })}
                </Chip>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
