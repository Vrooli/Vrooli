/** @vrooliComponentSource manipulation.split-pane */
import { Columns2, GripVertical, Maximize2 } from "lucide-react";
import {
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
  type DragEvent,
  type KeyboardEvent,
  type PointerEvent,
  type ReactNode,
  type Ref,
} from "react";

import { Button } from "../../components/Button";
import { IconButton } from "../../components/IconButton";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { API_BASE } from "../../api/client";
import type { DeviceEmulationValue } from "../../hooks/useDeviceEmulation";
import type { DeviceFiltersValue } from "../../hooks/useDeviceFilters";
import { useTranslation } from "../../i18n";
import type { PreviewKit } from "./ThemeSwitcher";
import { EmulatorViewport } from "./EmulatorChrome";
import type { PreviewFrameCandidate } from "../../api/components";

export type SpecimenIdentity = `${string}:${string}`;
export type PreviewSpecimen = {
  id: string;
  componentId: string;
  libraryId: string;
  version: string;
  name: string;
  displayName: string;
  propsJson: string;
  environment: Record<string, string>;
  expectJson: string;
  sourcePath: string;
  storyId: string;
  description?: string;
};

type StageProps = {
  emulator: DeviceEmulationValue;
  filters: DeviceFiltersValue;
  stageMode: boolean;
  comparisonActive: boolean;
  specimens: Array<PreviewSpecimen | undefined>;
  visibleSpecimens: Array<PreviewSpecimen | undefined>;
  readyExamples: ReadonlySet<string>;
  previewMessage: string;
  specimenErrors: Record<string, string>;
  specimenRetries: Record<string, number>;
  comparedSpecimens: ReadonlySet<SpecimenIdentity>;
  activeSpecimen: SpecimenIdentity | null;
  previewKit: PreviewKit;
  frameEnabled: boolean;
  frameOverride?: PreviewFrameCandidate;
  id: string;
  baselineSha: string;
  previewReloadKey: number;
  selectedVersion?: string;
  resolvedPreviewTheme: string;
  previewCanvasRef: Ref<HTMLDivElement>;
  tools: ReactNode;
  /** The layout has room to dock the tools beside the preview at all. */
  toolsDocked: boolean;
  /** The docked tools are currently expanded rather than collapsed away. */
  toolsOpen: boolean;
  onClearComparison: () => void;
  onSelectAllComparison: () => void;
  onToggleComparison: (identity: SpecimenIdentity) => void;
  onRetrySpecimen: (identity: SpecimenIdentity) => void;
  onRegisterPreviewFrame: (identity: SpecimenIdentity, frame: HTMLIFrameElement | null) => void;
  onPreviewLoad: (identity: SpecimenIdentity) => void;
  onPreviewError: (identity: SpecimenIdentity) => void;
  postToPreviewFrames: (message: unknown) => void;
  onCloseTools: () => void;
  onEnterSpecimen: (identity: SpecimenIdentity) => void;
};

export function ComponentEditorStage({
  emulator,
  filters,
  stageMode,
  comparisonActive,
  specimens,
  visibleSpecimens,
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
}: StageProps) {
  const { t } = useTranslation();
  const [canvasOffset, setCanvasOffset] = useState({ x: 0, y: 0 });
  const canvasRef = useRef<HTMLDivElement>(null);
  const dragRef = useRef<{ x: number; y: number; originX: number; originY: number } | null>(null);
  const identityFor = (example?: PreviewSpecimen): SpecimenIdentity =>
    `${example?.version || "__current__"}:${example?.name || "__default__"}`;
  const specimenOrderKey = specimens.map(identityFor).join("\u0000");
  const [caseOrder, setCaseOrder] = useState<SpecimenIdentity[]>(() => specimens.map(identityFor));
  const [draggedCase, setDraggedCase] = useState<SpecimenIdentity | null>(null);
  const [dropTarget, setDropTarget] = useState<{
    identity: SpecimenIdentity;
    placement: "before" | "after";
  } | null>(null);
  const dropTargetRef = useRef<typeof dropTarget>(null);

  const clearCaseDrag = () => {
    setDraggedCase(null);
    dropTargetRef.current = null;
    setDropTarget(null);
  };

  const updateCanvasDropTarget = (event: DragEvent<HTMLDivElement>) => {
    if (!draggedCase) return;
    event.preventDefault();
    event.stopPropagation();
    event.dataTransfer.dropEffect = "move";
    const nextTarget = canvasDropTarget(
      event.currentTarget,
      draggedCase,
      event.clientX,
      event.clientY,
    );
    dropTargetRef.current = nextTarget;
    setDropTarget(nextTarget);
  };

  useEffect(() => {
    const sourceOrder = specimenOrderKey
      ? (specimenOrderKey.split("\u0000") as SpecimenIdentity[])
      : [];
    setCaseOrder((current) => {
      const available = new Set(sourceOrder);
      const next = current.filter((identity) => available.has(identity));
      for (const identity of sourceOrder) {
        if (!next.includes(identity)) next.push(identity);
      }
      return next.length === current.length &&
        next.every((identity, index) => identity === current[index])
        ? current
        : next;
    });
  }, [specimenOrderKey]);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const onWheel = (event: WheelEvent) => {
      if (stageMode) return;
      event.preventDefault();
      if (event.ctrlKey) {
        const currentIndex = emulator.zoomLevels.reduce(
          (closest, level, index) =>
            Math.abs(level - emulator.zoom) <
            Math.abs((emulator.zoomLevels[closest] ?? emulator.zoom) - emulator.zoom)
              ? index
              : closest,
          0,
        );
        const direction = event.deltaY > 0 ? -1 : 1;
        const nextIndex = Math.max(
          0,
          Math.min(emulator.zoomLevels.length - 1, currentIndex + direction),
        );
        emulator.setZoom(emulator.zoomLevels[nextIndex] ?? emulator.zoom);
        return;
      }
      setCanvasOffset((current) => ({
        x: current.x - event.deltaX,
        y: current.y - event.deltaY,
      }));
    };
    canvas.addEventListener("wheel", onWheel, { passive: false });
    return () => canvas.removeEventListener("wheel", onWheel);
  }, [emulator, stageMode]);

  const orderIndex = new Map(caseOrder.map((identity, index) => [identity, index]));
  const orderedVisibleSpecimens = [...visibleSpecimens].sort(
    (left, right) =>
      (orderIndex.get(identityFor(left)) ?? Number.MAX_SAFE_INTEGER) -
      (orderIndex.get(identityFor(right)) ?? Number.MAX_SAFE_INTEGER),
  );

  const moveCase = (
    moving: SpecimenIdentity,
    target: SpecimenIdentity,
    placement: "before" | "after",
  ) => {
    if (moving === target) return;
    setCaseOrder((current) => {
      const next = current.filter((identity) => identity !== moving);
      const targetIndex = next.indexOf(target);
      if (targetIndex < 0) return current;
      next.splice(targetIndex + (placement === "after" ? 1 : 0), 0, moving);
      return next;
    });
  };

  const moveCaseByKeyboard = (identity: SpecimenIdentity, event: KeyboardEvent) => {
    const currentIndex = caseOrder.indexOf(identity);
    if (currentIndex < 0) return;
    let nextIndex = currentIndex;
    if (event.key === "ArrowLeft" || event.key === "ArrowUp") nextIndex--;
    else if (event.key === "ArrowRight" || event.key === "ArrowDown") nextIndex++;
    else if (event.key === "Home") nextIndex = 0;
    else if (event.key === "End") nextIndex = caseOrder.length - 1;
    else return;
    event.preventDefault();
    event.stopPropagation();
    nextIndex = Math.max(0, Math.min(caseOrder.length - 1, nextIndex));
    if (nextIndex === currentIndex) return;
    setCaseOrder((current) => {
      const next = [...current];
      next.splice(currentIndex, 1);
      next.splice(nextIndex, 0, identity);
      return next;
    });
  };
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
              {orderedVisibleSpecimens.map((example) => {
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

function canvasDropTarget(
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
  if (pointerY <= first.bounds.top) {
    return { identity: first.card.dataset.specimen as SpecimenIdentity, placement: "before" };
  }
  if (pointerY >= last.bounds.bottom) {
    return { identity: last.card.dataset.specimen as SpecimenIdentity, placement: "after" };
  }

  const nearest = cards.reduce((best, candidate) => {
    const bestCenterX = best.bounds.left + best.bounds.width / 2;
    const bestCenterY = best.bounds.top + best.bounds.height / 2;
    const candidateCenterX = candidate.bounds.left + candidate.bounds.width / 2;
    const candidateCenterY = candidate.bounds.top + candidate.bounds.height / 2;
    const bestDistance = Math.hypot(pointerX - bestCenterX, pointerY - bestCenterY);
    const candidateDistance = Math.hypot(pointerX - candidateCenterX, pointerY - candidateCenterY);
    return candidateDistance < bestDistance ? candidate : best;
  });
  const centerX = nearest.bounds.left + nearest.bounds.width / 2;
  const placement =
    pointerY < nearest.bounds.top || (pointerY <= nearest.bounds.bottom && pointerX < centerX)
      ? "before"
      : "after";
  return {
    identity: nearest.card.dataset.specimen as SpecimenIdentity,
    placement,
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
