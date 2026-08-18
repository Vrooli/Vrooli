/** @vrooliComponentSource react-component-library:SplitPane */
import { useRef, useState, type PointerEvent, type ReactNode, type Ref } from "react";

import { Button } from "../../components/Button";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { API_BASE } from "../../api/client";
import type { DeviceEmulationValue } from "../../hooks/useDeviceEmulation";
import type { DeviceFiltersValue } from "../../hooks/useDeviceFilters";
import { useTranslation } from "../../i18n";
import type { PreviewKit } from "./ThemeSwitcher";
import { EmulatorViewport } from "./EmulatorChrome";

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
  id: string;
  baselineSha: string;
  previewReloadKey: number;
  selectedVersion?: string;
  resolvedPreviewTheme: string;
  previewCanvasRef: Ref<HTMLDivElement>;
  tools: ReactNode;
  toolsOpen: boolean;
  onClearComparison: () => void;
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
  id,
  baselineSha,
  previewReloadKey,
  selectedVersion,
  resolvedPreviewTheme,
  previewCanvasRef,
  tools,
  toolsOpen,
  onClearComparison,
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
  const dragRef = useRef<{ x: number; y: number; originX: number; originY: number } | null>(null);
  const identityFor = (example?: PreviewSpecimen): SpecimenIdentity =>
    `${example?.version || "__current__"}:${example?.name || "__default__"}`;
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
            onWheel={(event) => {
              if (stageMode) return;
              event.preventDefault();
              const direction = event.deltaY > 0 ? -1 : 1;
              emulator.setZoom(emulator.zoom + direction * 0.1);
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
                    ? "grid min-w-0 grid-cols-1 gap-space-xs md:grid-cols-2"
                    : "grid min-w-0 grid-cols-[repeat(auto-fit,minmax(20rem,1fr))] gap-space-xs"
              }
            >
              {visibleSpecimens.map((example) => {
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
                    onClick={() => {
                      if (!stageMode) onEnterSpecimen(identity);
                    }}
                    className={`min-w-0 overflow-hidden rounded-md border bg-app-surface ${stageMode ? "w-fit max-w-full" : "h-[24rem]"} ${isActive ? "border-app-primary ring-1 ring-app-primary/30" : "border-app-border"}`}
                  >
                    {!stageMode && (
                      <>
                        <header className="flex items-center justify-between gap-space-2xs border-b border-app-border px-space-xs py-space-2xs">
                          <h3
                            data-testid={selectors.components.editor.exampleTitle}
                            title={example?.description}
                            className="min-w-0 truncate text-sm font-semibold text-app-foreground"
                          >
                            {title}
                          </h3>
                          {examplesCanCompare(specimens) && (
                            <Button
                              data-testid={selectors.components.editor.exampleCompare}
                              type="button"
                              variant={comparedSpecimens.has(identity) ? "primary" : "secondary"}
                              aria-pressed={comparedSpecimens.has(identity)}
                              disabled={
                                !comparedSpecimens.has(identity) && comparedSpecimens.size >= 2
                              }
                              className="h-control-compact px-space-2xs text-xs"
                              onClick={() => onToggleComparison(identity)}
                            >
                              {t(strings.components.editor.compareStory)}
                            </Button>
                          )}
                        </header>
                        {example?.description ? (
                          <p
                            data-testid={selectors.components.editor.storyDescription}
                            className="border-b border-app-border bg-app-muted/30 px-space-xs py-space-2xs text-xs leading-relaxed text-app-muted-foreground"
                          >
                            {example.description}
                          </p>
                        ) : null}
                      </>
                    )}
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
                        className={stageMode ? "block border-0" : "block max-w-full border-0"}
                      />
                    )}
                  </section>
                );
              })}
            </div>
          </div>
        </EmulatorViewport>
      </div>
      {activeSpecimen && toolsOpen && (
        <aside
          id="component-preview-tools"
          data-testid={selectors.components.editor.previewToolsPanel}
          aria-label={t("components.editor.showTools", { defaultValue: "Preview controls" })}
          className="absolute bottom-space-xs right-space-xs top-48 z-30 flex w-stage-panel min-w-0 flex-col overflow-hidden rounded-panel border border-app-border bg-app-surface/98 shadow-2xl backdrop-blur"
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
  stageMode: boolean,
): string {
  const url = new URL(
    `${API_BASE.replace(/\/$/, "")}/preview/${encodeURIComponent(id)}/harness.html`,
  );
  url.searchParams.set("v", encodeURIComponent(contentVersion || "initial"));
  url.searchParams.set("r", String(reloadKey));
  url.searchParams.set("kit", kit);
  url.searchParams.set("frame", frameEnabled ? "on" : "off");
  url.searchParams.set("view", stageMode ? "focus" : "canvas");
  if (selectedVersion) url.searchParams.set("version", selectedVersion);
  if (example) {
    if (example.storyId) url.searchParams.set("story", example.storyId);
    else url.searchParams.set("example", example.name);
    url.searchParams.set("version", selectedVersion || example.version);
  }
  return url.toString();
}
