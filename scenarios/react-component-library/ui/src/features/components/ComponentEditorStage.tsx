/** @vrooliComponentSource patterns.split-pane-workspace */
import type { ReactNode, Ref } from "react";
import { Group, Panel, Separator } from "react-resizable-panels";

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
  desktopLayout: boolean;
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
  mobileTools: ReactNode;
  onClearComparison: () => void;
  onToggleComparison: (identity: SpecimenIdentity) => void;
  onRetrySpecimen: (identity: SpecimenIdentity) => void;
  onRegisterPreviewFrame: (identity: SpecimenIdentity, frame: HTMLIFrameElement | null) => void;
  onPreviewLoad: (identity: SpecimenIdentity) => void;
  onPreviewError: (identity: SpecimenIdentity) => void;
  postToPreviewFrames: (message: unknown) => void;
  previewToolsPanelRef: React.RefObject<
    import("react-resizable-panels").PanelImperativeHandle | null
  >;
  onPanelCollapsed: (collapsed: boolean) => void;
  onSetMobileTool: (tool: "props" | "inspector" | null) => void;
};

export function ComponentEditorStage({
  emulator,
  filters,
  desktopLayout,
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
  mobileTools,
  onClearComparison,
  onToggleComparison,
  onRetrySpecimen,
  onRegisterPreviewFrame,
  onPreviewLoad,
  onPreviewError,
  postToPreviewFrames,
  previewToolsPanelRef,
  onPanelCollapsed,
  onSetMobileTool,
}: StageProps) {
  const { t } = useTranslation();
  const identityFor = (example?: PreviewSpecimen): SpecimenIdentity =>
    `${example?.version || "__current__"}:${example?.name || "__default__"}`;
  return (
    <Group
      id="component-preview-workbench"
      orientation="vertical"
      defaultLayout={{ specimen: 1, tools: 280 }}
      className="min-h-0 flex-1"
    >
      <Panel id="specimen" minSize={220} className="min-h-0">
        <div className="relative h-full min-h-0">
          <EmulatorViewport
            emulator={emulator}
            filters={filters}
            mode={stageMode ? "stage" : "gallery"}
          >
            <div
              data-testid={selectors.components.editor.gallery}
              data-preview-stage-mode={stageMode ? "true" : "false"}
              className={
                stageMode
                  ? "flex h-full items-center justify-center overflow-auto bg-app-background p-space-xs"
                  : "h-full overflow-auto bg-app-background p-space-xs"
              }
              style={
                stageMode
                  ? { width: emulator.displayWidth, height: emulator.displayHeight }
                  : undefined
              }
            >
              {comparisonActive && (
                <div
                  data-testid={selectors.components.editor.comparisonToolbar}
                  className="mb-space-xs flex flex-wrap items-center justify-between gap-space-2xs rounded-md border border-app-primary/30 bg-app-primary/10 px-space-xs py-space-2xs"
                >
                  <p className="text-xs text-app-foreground">
                    {t(strings.components.editor.comparing)}
                  </p>
                  <Button
                    data-testid={selectors.components.editor.comparisonClear}
                    type="button"
                    variant="secondary"
                    className="h-7 px-space-2xs text-xs"
                    onClick={onClearComparison}
                  >
                    {t(strings.components.editor.showAllStories)}
                  </Button>
                </div>
              )}
              <p
                data-testid={selectors.components.editor.galleryStatus}
                aria-live="polite"
                className="mb-space-2xs text-xs text-app-muted-foreground"
              >
                {previewMessage ||
                  t(strings.components.editor.storyStatus, {
                    ready: readyExamples.size,
                    total: specimens.length,
                  })}
              </p>
              <div
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
                      className={`min-w-0 overflow-hidden rounded-md border bg-app-surface ${stageMode ? "w-fit max-w-full" : ""} ${isActive ? "border-app-primary ring-1 ring-app-primary/30" : "border-app-border"}`}
                    >
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
                            className="h-7 px-space-2xs text-xs"
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
                      {error ? (
                        <div
                          data-testid={selectors.components.editor.specimenError}
                          className="flex min-h-[18rem] flex-col items-center justify-center gap-space-xs bg-app-danger/5 p-space-sm text-center"
                        >
                          <p className="text-xs text-app-danger">{error}</p>
                          <Button
                            data-testid={selectors.components.editor.specimenRetry}
                            type="button"
                            variant="secondary"
                            className="h-8 px-space-xs text-xs"
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
                            ...(stageMode
                              ? { height: emulator.displayHeight }
                              : { aspectRatio: "16 / 9" }),
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
      </Panel>
      {activeSpecimen && desktopLayout && (
        <>
          <Separator className="hidden h-1 shrink-0 bg-app-border hover:bg-app-primary lg:block" />
          <Panel
            panelRef={previewToolsPanelRef}
            id="tools"
            defaultSize={280}
            minSize={160}
            maxSize="45%"
            collapsible
            collapsedSize={0}
            onResize={(size) => onPanelCollapsed(size.inPixels === 0)}
            className="hidden min-h-0 lg:block"
          >
            <div
              id="component-preview-tools"
              data-testid={selectors.components.editor.previewToolsPanel}
              className="h-full overflow-y-auto border-t border-app-border bg-app-surface p-space-xs"
            >
              {tools}
            </div>
          </Panel>
        </>
      )}
      {activeSpecimen && (
        <div className="flex shrink-0 gap-space-2xs border-t border-app-border bg-app-surface p-space-2xs lg:hidden">
          <Button
            type="button"
            className="h-9 flex-1 text-xs"
            onClick={() => onSetMobileTool("props")}
          >
            {t("components.editor.editProps", { defaultValue: "Edit props" })}
          </Button>
          <Button
            type="button"
            variant="secondary"
            className="h-9 flex-1 text-xs"
            onClick={() => onSetMobileTool("inspector")}
          >
            {t("components.inspector.title", { defaultValue: "Inspect" })}
          </Button>
          <Button
            type="button"
            variant="secondary"
            className="h-9 px-space-xs text-xs"
            onClick={() => onSetMobileTool(null)}
          >
            {t("components.editor.reset", { defaultValue: "Reset" })}
          </Button>
        </div>
      )}
      {mobileTools}
    </Group>
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
): string {
  const url = new URL(
    `${API_BASE.replace(/\/$/, "")}/preview/${encodeURIComponent(id)}/harness.html`,
  );
  url.searchParams.set("v", encodeURIComponent(contentVersion || "initial"));
  url.searchParams.set("r", String(reloadKey));
  url.searchParams.set("kit", kit);
  url.searchParams.set("frame", frameEnabled ? "on" : "off");
  if (selectedVersion) url.searchParams.set("version", selectedVersion);
  if (example) {
    if (example.storyId) url.searchParams.set("story", example.storyId);
    else url.searchParams.set("example", example.name);
    url.searchParams.set("version", selectedVersion || example.version);
  }
  return url.toString();
}
