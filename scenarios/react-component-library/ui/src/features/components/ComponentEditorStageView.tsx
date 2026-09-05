import { Columns2, GripVertical, Maximize2 } from "lucide-react";
import { useLayoutEffect, useRef, useState, type PointerEvent } from "react";
import { Button } from "@vrooli/react-component-library/Button/2";
import { IconButton } from "@vrooli/react-component-library/IconButton/2";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { API_BASE } from "../../api/client";
import type { PreviewFrameCandidate } from "../../api/components";
import { useTranslation } from "../../i18n";
import type { PreviewKit } from "./ThemeSwitcher";
import { EmulatorViewport } from "./EmulatorChrome";
import type { PreviewSpecimen, SpecimenIdentity } from "./ComponentEditorStage";

type StageViewModel = Record<string, any>;

export function ComponentEditorStageView({ model }: { model: StageViewModel }) {
  const { t } = useTranslation();
  const {
    emulator,
    filters,
    stageMode,
    comparisonActive,
    specimens,
    readyExamples,
    previewMessage,
    specimenErrors,
    specimenRetries,
    comparedSpecimens,
    activeSpecimen,
    previewKit,
    frameEnabled,
    frameOverride,
    id,
    baselineSha,
    previewReloadKey,
    selectedVersion,
    resolvedPreviewTheme,
    previewCanvasRef,
    tools,
    toolsDocked,
    toolsOpen,
    onClearComparison,
    onSelectAllComparison,
    onToggleComparison,
    onRetrySpecimen,
    onRegisterPreviewFrame,
    onPreviewLoad,
    onPreviewError,
    postToPreviewFrames,
    onCloseTools,
    onEnterSpecimen,
    canvasRef,
    canvasOffset,
    setCanvasOffset,
    dragRef,
    draggedCase,
    dropTarget,
    dropTargetRef,
    setDropTarget,
    setDraggedCase,
    caseOrder,
    updateCanvasDropTarget,
    clearCaseDrag,
    moveCase,
    moveCaseByKeyboard,
    orderedVisibleSpecimens,
    identityFor,
  } = model;
  const typedOrderedVisibleSpecimens = orderedVisibleSpecimens as Array<
    PreviewSpecimen | undefined
  >;
  return (
    <div id="component-preview-workbench" className="relative min-h-0 flex-1 pt-space-2xl">
      <div className="relative h-full min-h-0">
        {comparisonActive && (
          <div className="absolute bottom-space-xs left-space-xs z-10 flex max-w-inline-overlay items-center gap-space-2xs rounded-md border border-app-primary/30 bg-app-surface/95 px-space-xs py-space-2xs text-xs shadow-md backdrop-blur">
            <p
              data-testid={selectors.components.editor.comparisonToolbar}
              className="text-app-foreground"
            >
              {t(strings.components.editor.comparing)}
            </p>
            <Button
              data-testid={selectors.components.editor.comparisonClear}
              type="button"
              variant="secondary"
              className="h-control-compact shrink-0 px-space-2xs text-xs"
              onClick={onClearComparison}
            >
              {t(strings.components.editor.showAllStories)}
            </Button>
            <Button
              type="button"
              variant="secondary"
              className="h-control-compact shrink-0 px-space-2xs text-xs"
              onClick={onSelectAllComparison}
            >
              Story sheet (4 max)
            </Button>
          </div>
        )}
        <p
          data-testid={selectors.components.editor.galleryStatus}
          aria-live="polite"
          className="pointer-events-none absolute bottom-space-xs right-space-xs z-10 max-w-overlay-label truncate rounded-md bg-app-surface/90 px-space-2xs py-space-3xs text-xs text-app-muted-foreground shadow-sm backdrop-blur"
        >
          {previewMessage ||
            t(strings.components.editor.storyStatus, {
              ready: readyExamples.size,
              total: specimens.length,
            })}
        </p>
        <EmulatorViewport
          emulator={emulator}
          filters={filters}
          mode={stageMode ? "stage" : "gallery"}
        >
          <div
            ref={canvasRef}
            data-testid={selectors.components.editor.gallery}
            data-preview-stage-mode={stageMode ? "true" : "false"}
            data-preview-view={stageMode ? "focus" : "canvas"}
            onPointerDown={(event: PointerEvent<HTMLDivElement>) => {
              if (stageMode) return;
              const target = event.target as Element;
              if (target.closest(`[data-testid="${selectors.components.editor.exampleCard}"]`)) {
                return;
              }
              dragRef.current = {
                x: event.clientX,
                y: event.clientY,
                originX: canvasOffset.x,
                originY: canvasOffset.y,
              };
              event.currentTarget.setPointerCapture(event.pointerId);
            }}
            onPointerMove={(event) => {
              if (!dragRef.current) return;
              setCanvasOffset({
                x: dragRef.current.originX + event.clientX - dragRef.current.x,
                y: dragRef.current.originY + event.clientY - dragRef.current.y,
              });
            }}
            onPointerUp={(event) => {
              dragRef.current = null;
              event.currentTarget.releasePointerCapture?.(event.pointerId);
            }}
            className={
              stageMode
                ? "flex h-full min-h-0 items-center justify-center bg-app-background p-space-xs"
                : "h-full cursor-grab touch-none bg-app-background p-space-xs active:cursor-grabbing"
            }
            style={
              stageMode
                ? { width: emulator.displayWidth, height: emulator.displayHeight }
                : {
                    backgroundImage:
                      "linear-gradient(to right, color-mix(in srgb, var(--color-border) 48%, transparent) 1px, transparent 1px), linear-gradient(to bottom, color-mix(in srgb, var(--color-border) 48%, transparent) 1px, transparent 1px)",
                    backgroundSize: "24px 24px",
                  }
            }
          >
            <div
              data-canvas-surface={!stageMode ? "true" : undefined}
              aria-label={!stageMode ? "Preview canvas. Drag to pan, scroll to zoom." : undefined}
              onDragOver={!stageMode ? updateCanvasDropTarget : undefined}
              onDragLeave={
                !stageMode
                  ? (event) => {
                      const next = event.relatedTarget;
                      if (!(next instanceof Node) || !event.currentTarget.contains(next)) {
                        dropTargetRef.current = null;
                        setDropTarget(null);
                      }
                    }
                  : undefined
              }
              onDrop={
                !stageMode
                  ? (event) => {
                      event.preventDefault();
                      event.stopPropagation();
                      const moving =
                        (event.dataTransfer.getData(
                          "application/x-rcl-preview-case",
                        ) as SpecimenIdentity) || draggedCase;
                      const target = dropTargetRef.current;
                      if (moving && target) {
                        moveCase(moving, target.identity, target.placement);
                      }
                      clearCaseDrag();
                    }
                  : undefined
              }
              style={
                !stageMode
                  ? {
                      transform: `translate(${canvasOffset.x}px, ${canvasOffset.y}px) scale(${emulator.zoom})`,
                      transformOrigin: "top left",
                    }
                  : undefined
              }
              className={
                stageMode
                  ? "flex min-w-0 justify-center"
                  : comparisonActive
                    ? `relative grid min-h-full min-w-0 grid-cols-1 gap-space-xs rounded-md md:grid-cols-2 ${draggedCase ? "ring-1 ring-inset ring-app-primary/25" : ""}`
                    : `relative grid min-h-full min-w-0 grid-cols-[repeat(auto-fit,minmax(20rem,1fr))] gap-space-xs rounded-md ${draggedCase ? "ring-1 ring-inset ring-app-primary/25" : ""}`
              }
            >
              {typedOrderedVisibleSpecimens.map((example) => {
                const identity = identityFor(example);
                const title =
                  example?.displayName ||
                  example?.name ||
                  t(strings.components.editor.previewHeading);
                const error = specimenErrors[identity];
                const isActive = activeSpecimen === identity;
                return (
                  <section
                    ref={stageMode && isActive ? previewCanvasRef : undefined}
                    key={identity}
                    data-testid={selectors.components.editor.exampleCard}
                    data-specimen={identity}
                    data-story={example?.storyId ?? "__default__"}
                    data-story-ready={readyExamples.has(identity) ? "true" : "false"}
                    data-case-position={caseOrder.indexOf(identity) + 1}
                    data-drop-placement={
                      dropTarget?.identity === identity ? dropTarget.placement : undefined
                    }
                    className={`relative min-w-0 rounded-md border bg-app-surface transition-[border-color,box-shadow,opacity,transform] ${stageMode ? "w-fit max-w-full overflow-hidden" : "flex h-[clamp(22rem,60vh,48rem)] min-h-[18rem] max-h-[80vh] min-w-[18rem] max-w-full resize flex-col overflow-auto"} ${draggedCase === identity ? "scale-[0.98] border-dashed opacity-40 shadow-none" : "shadow-sm"} ${dropTarget?.identity === identity ? "border-app-primary ring-2 ring-app-primary/40" : isActive ? "border-app-primary ring-1 ring-app-primary/30" : "border-app-border"}`}
                  >
                    {dropTarget?.identity === identity && (
                      <span
                        aria-hidden
                        className={`pointer-events-none absolute bottom-space-2xs top-space-2xs z-30 w-1 rounded-full bg-app-primary shadow-lg ${dropTarget.placement === "before" ? "left-0 -translate-x-1/2" : "right-0 translate-x-1/2"}`}
                      />
                    )}
                    {draggedCase && !stageMode && (
                      <span
                        aria-hidden
                        className="absolute inset-0 z-20 cursor-grabbing bg-app-primary/[0.02]"
                      />
                    )}
                    {!stageMode && (
                      <>
                        <header className="flex items-center justify-between gap-space-2xs border-b border-app-border px-space-xs py-space-2xs">
                          <div className="flex min-w-0 items-baseline gap-space-2xs">
                            <h3
                              data-testid={selectors.components.editor.exampleTitle}
                              title={example?.description}
                              className="min-w-0 truncate text-sm font-semibold text-app-foreground"
                            >
                              {title}
                            </h3>
                            <CaseDimensions />
                          </div>
                          <div className="flex shrink-0 items-center gap-space-3xs">
                            <span
                              role="button"
                              tabIndex={0}
                              draggable
                              aria-label={`Move ${title}. Use arrow keys to reorder.`}
                              className="h-control-compact min-h-control-compact min-w-control-compact cursor-grab active:cursor-grabbing"
                              onClick={(event) => event.stopPropagation()}
                              onKeyDown={(event) => moveCaseByKeyboard(identity, event)}
                              onDragStart={(event) => {
                                event.stopPropagation();
                                event.dataTransfer.effectAllowed = "move";
                                event.dataTransfer.setData(
                                  "application/x-rcl-preview-case",
                                  identity,
                                );
                                const card = event.currentTarget.closest("section");
                                if (card && typeof event.dataTransfer.setDragImage === "function") {
                                  event.dataTransfer.setDragImage(card, 24, 18);
                                }
                                setDraggedCase(identity);
                              }}
                              onDragEnd={(event) => {
                                event.stopPropagation();
                                clearCaseDrag();
                              }}
                            >
                              <GripVertical aria-hidden className="h-icon-compact w-icon-compact" />
                            </span>
                            <IconButton
                              data-testid={selectors.components.editor.exampleFocus}
                              type="button"
                              density="compact"
                              variant="secondary"
                              aria-label={`Focus ${title}`}
                              className="h-control-compact min-h-control-compact min-w-control-compact"
                              onClick={(event) => {
                                event.stopPropagation();
                                onEnterSpecimen(identity);
                              }}
                            >
                              <Maximize2 aria-hidden className="h-icon-compact w-icon-compact" />
                            </IconButton>
                            {examplesCanCompare(specimens) && (
                              <IconButton
                                data-testid={selectors.components.editor.exampleCompare}
                                type="button"
                                variant={comparedSpecimens.has(identity) ? "primary" : "secondary"}
                                aria-label={
                                  comparedSpecimens.has(identity)
                                    ? `Remove ${title} from comparison`
                                    : `Compare ${title}`
                                }
                                aria-pressed={comparedSpecimens.has(identity)}
                                disabled={
                                  !comparedSpecimens.has(identity) && comparedSpecimens.size >= 4
                                }
                                density="compact"
                                className="h-control-compact min-h-control-compact min-w-control-compact shrink-0"
                                onClick={(event) => {
                                  event.stopPropagation();
                                  onToggleComparison(identity);
                                }}
                              >
                                <Columns2 aria-hidden className="h-icon-compact w-icon-compact" />
                              </IconButton>
                            )}
                          </div>
                        </header>
                      </>
                    )}
                    {/*
                     * The state's own description belongs to the specimen, not
                     * to the canvas chrome: focus mode drops the card header
                     * but still shows a single named state, and dropping the
                     * one sentence that says what that state *is* was the
                     * difference between a labelled preview and an unlabelled
                     * frame.
                     */}
                    {example?.description ? (
                      <p
                        data-testid={selectors.components.editor.storyDescription}
                        className="border-b border-app-border bg-app-muted/30 px-space-xs py-space-2xs text-xs leading-relaxed text-app-muted-foreground"
                      >
                        {example.description}
                      </p>
                    ) : null}
                    {error ? (
                      <div
                        data-testid={selectors.components.editor.specimenError}
                        className="flex min-h-stage-empty flex-col items-center justify-center gap-space-xs bg-app-danger/5 p-space-sm text-center"
                      >
                        <p className="text-xs text-app-danger">{error}</p>
                        <Button
                          data-testid={selectors.components.editor.specimenRetry}
                          type="button"
                          variant="secondary"
                          className="h-control-tight px-space-xs text-xs"
                          onClick={() => onRetrySpecimen(identity)}
                        >
                          {t(strings.components.editor.previewRetry)}
                        </Button>
                      </div>
                    ) : (
                      <iframe
                        data-testid={selectors.components.editor.previewFrame}
                        data-specimen={identity}
                        data-preview-kit={previewKit}
                        title={`${t(strings.components.editor.previewHeading)} - ${title}`}
                        src={harnessUrl(
                          id,
                          baselineSha,
                          previewReloadKey + (specimenRetries[identity] ?? 0),
                          example,
                          selectedVersion,
                          previewKit,
                          frameEnabled,
                          frameOverride,
                          stageMode,
                        )}
                        sandbox="allow-scripts allow-same-origin"
                        ref={(frame) => onRegisterPreviewFrame(identity, frame)}
                        onLoad={() => {
                          onPreviewLoad(identity);
                          postToPreviewFrames({
                            type: "rcl-resolved-theme",
                            theme: resolvedPreviewTheme,
                          });
                        }}
                        onError={() => onPreviewError(identity)}
                        style={{
                          width: stageMode ? emulator.displayWidth : "100%",
                          ...(stageMode ? { height: emulator.displayHeight } : { height: "100%" }),
                          backgroundColor: "var(--color-background)",
                        }}
                        className={
                          stageMode
                            ? "block border-0"
                            : "block min-h-[18rem] max-w-full flex-1 border-0"
                        }
                      />
                    )}
                  </section>
                );
              })}
            </div>
          </div>
        </EmulatorViewport>
      </div>
      {/*
       * Where the layout can dock them, the tools stay mounted and the toggle
       * only collapses them out of view. Unmounting on collapse threw away the
       * whole panel's state — a half-typed props override, the scroll position
       * in the event log, the inspector selection — every time the toggle was
       * used, and left the preview unable to report events until someone
       * expanded it. Narrow layouts still get no docked panel at all; they use
       * the mobile tools sheet instead.
       */}
      {activeSpecimen && toolsDocked && (
        <aside
          id="component-preview-tools"
          data-testid={selectors.components.editor.previewToolsPanel}
          aria-label={t("components.editor.showTools", { defaultValue: "Preview controls" })}
          className={`absolute bottom-space-xs right-space-xs top-48 z-30 w-stage-panel min-w-0 flex-col overflow-hidden rounded-panel border border-app-border bg-app-surface/98 shadow-2xl backdrop-blur ${toolsOpen ? "flex" : "hidden"}`}
        >
          <div className="flex shrink-0 items-center justify-between gap-space-xs border-b border-app-border px-space-xs py-space-2xs">
            <div>
              <p className="text-sm font-semibold text-app-foreground">
                {t("components.editor.previewControls", { defaultValue: "Preview controls" })}
              </p>
              <p className="text-xs text-app-muted-foreground">
                {activeSpecimen.split(":").slice(1).join(":")}
              </p>
            </div>
            <Button
              type="button"
              variant="secondary"
              className="h-control-tight shrink-0 px-space-2xs text-xs"
              onClick={onCloseTools}
            >
              {t("common.close", { defaultValue: "Close" })}
            </Button>
          </div>
          <div className="min-h-0 flex-1 overflow-y-auto p-space-xs">{tools}</div>
        </aside>
      )}
    </div>
  );
}
type CaseDropTarget = {
  identity: SpecimenIdentity;
  placement: "before" | "after";
};

export function canvasDropTarget(
  surface: HTMLDivElement,
  moving: SpecimenIdentity,
  pointerX: number,
  pointerY: number,
): CaseDropTarget | null {
  const cards = Array.from(
    surface.querySelectorAll<HTMLElement>(
      `[data-testid="${selectors.components.editor.exampleCard}"]`,
    ),
  )
    .filter((card) => card.dataset.specimen !== moving)
    .map((card) => ({ card, bounds: card.getBoundingClientRect() }));
  if (cards.length === 0) return null;
  const first = cards[0]!;
  const last = cards[cards.length - 1]!;
  if (pointerY <= first.bounds.top)
    return { identity: first.card.dataset.specimen as SpecimenIdentity, placement: "before" };
  if (pointerY >= last.bounds.bottom)
    return { identity: last.card.dataset.specimen as SpecimenIdentity, placement: "after" };
  const nearest = cards.reduce((best, candidate) => {
    const bestCenterX = best.bounds.left + best.bounds.width / 2;
    const bestCenterY = best.bounds.top + best.bounds.height / 2;
    const candidateCenterX = candidate.bounds.left + candidate.bounds.width / 2;
    const candidateCenterY = candidate.bounds.top + candidate.bounds.height / 2;
    return Math.hypot(pointerX - candidateCenterX, pointerY - candidateCenterY) <
      Math.hypot(pointerX - bestCenterX, pointerY - bestCenterY)
      ? candidate
      : best;
  });
  return {
    identity: nearest.card.dataset.specimen as SpecimenIdentity,
    placement:
      pointerY < nearest.bounds.top ||
      (pointerY <= nearest.bounds.bottom &&
        pointerX < nearest.bounds.left + nearest.bounds.width / 2)
        ? "before"
        : "after",
  };
}

function CaseDimensions() {
  const markerRef = useRef<HTMLSpanElement>(null);
  const [dimensions, setDimensions] = useState<{ width: number; height: number } | null>(null);

  useLayoutEffect(() => {
    const card = markerRef.current?.closest<HTMLElement>(
      `[data-testid="${selectors.components.editor.exampleCard}"]`,
    );
    if (!card) return;
    const measure = () => {
      const bounds = card.getBoundingClientRect();
      const width = card.offsetWidth || Math.round(bounds.width);
      const height = card.offsetHeight || Math.round(bounds.height);
      if (width > 0 && height > 0) setDimensions({ width, height });
    };
    measure();
    const observer = new ResizeObserver(measure);
    observer.observe(card);
    return () => observer.disconnect();
  }, []);

  return (
    <span
      ref={markerRef}
      data-testid={selectors.components.editor.exampleDimensions}
      className="shrink-0 font-mono text-[0.6875rem] tabular-nums text-app-muted-foreground"
      aria-label="Case dimensions"
    >
      {dimensions ? `${dimensions.width} × ${dimensions.height}` : "— × —"}
    </span>
  );
}

function examplesCanCompare(specimens: Array<PreviewSpecimen | undefined>): boolean {
  return specimens.length > 1;
}

function harnessUrl(
  id: string,
  contentVersion: string,
  reloadKey: number,
  example: PreviewSpecimen | undefined,
  selectedVersion: string | undefined,
  kit: PreviewKit,
  frameEnabled: boolean,
  frameOverride: PreviewFrameCandidate | undefined,
  stageMode: boolean,
): string {
  const url = new URL(
    `${API_BASE.replace(/\/$/, "")}/preview/${encodeURIComponent(id)}/harness.html`,
  );
  url.searchParams.set("v", encodeURIComponent(contentVersion || "initial"));
  url.searchParams.set("r", String(reloadKey));
  url.searchParams.set("kit", kit);
  url.searchParams.set("frame", frameEnabled ? "on" : "off");
  if (frameOverride) {
    url.searchParams.set("frameAsset", frameOverride.asset);
    url.searchParams.set("frameVersion", frameOverride.version);
    url.searchParams.set("frameRegion", frameOverride.region);
    url.searchParams.set("frameCapability", frameOverride.capability);
    url.searchParams.set("frameFixture", frameOverride.fixture);
  }
  url.searchParams.set("view", stageMode ? "focus" : "canvas");
  if (selectedVersion) url.searchParams.set("version", selectedVersion);
  if (example) {
    if (example.storyId) url.searchParams.set("story", example.storyId);
    else url.searchParams.set("example", example.name);
    url.searchParams.set("version", selectedVersion || example.version);
  }
  return url.toString();
}
