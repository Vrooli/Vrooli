/** @vrooliComponentSource manipulation.split-pane */
import { Fragment, type ReactNode } from "react";
import { Group, Panel, Separator } from "react-resizable-panels";
import {
  ArrowLeft,
  Eye,
  Info,
  Maximize2,
  Menu,
  Minimize2,
  PanelsLeftRight,
  SlidersHorizontal,
} from "lucide-react";

import { Button } from "@vrooli/react-component-library/Button/2";
import { IconButton } from "@vrooli/react-component-library/IconButton/2";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import { ExperienceSurface } from "@vrooli/react-component-library/ExperienceSurface/1";
import { WorkspaceHeader } from "@vrooli/react-component-library/WorkspaceHeader/1";
import { AssetWorkspace } from "../assets/AssetWorkspace";
import { selectors } from "../../consts/selectors";
import { ComponentEditorStage } from "./ComponentEditorStage";
import { ComponentEditorSource } from "./ComponentEditorSource";
import { ComponentEditorMobileTools, ComponentEditorTools } from "./ComponentEditorTools";
import { ThemeSwitcher } from "./ThemeSwitcher";
import { EmulatorToolbar } from "./EmulatorChrome";
import { errorMessage } from "../../lib/errorMessage";

function SourceLoadingSkeleton() {
  return (
    <div
      data-testid={selectors.components.editor.loading}
      role="status"
      aria-label="Loading source"
      className="flex h-full min-h-0 flex-col bg-app-background"
    >
      <div className="flex h-control-md shrink-0 items-center border-b border-app-border bg-app-surface px-space-2xs">
        <span className="h-3 w-16 animate-pulse rounded bg-app-surface-muted" />
      </div>
      <div className="flex shrink-0 gap-space-3xs border-b border-app-border bg-app-surface px-space-2xs py-space-2xs">
        <span className="h-control-compact w-8 animate-pulse rounded bg-app-surface-muted" />
        <span className="h-control-compact w-36 animate-pulse rounded bg-app-surface-muted" />
        <span className="h-control-compact w-28 animate-pulse rounded bg-app-surface-muted" />
      </div>
      <div className="flex min-h-0 flex-1 flex-col gap-space-2xs p-space-sm" aria-hidden="true">
        {["w-11/12", "w-8/12", "w-10/12", "w-6/12", "w-9/12", "w-7/12"].map((width) => (
          <span key={width} className={`h-3 animate-pulse rounded bg-app-surface-muted ${width}`} />
        ))}
      </div>
    </div>
  );
}

type EditorViewModel = { [key: string]: any };
type WorkspacePane = "details" | "files" | "preview";

function PaneHeader({
  pane,
  index,
  label,
  icon,
  availablePanes,
  paneLabels,
  onSelect,
}: {
  pane: WorkspacePane;
  index: number;
  label: string;
  icon: ReactNode;
  availablePanes: WorkspacePane[];
  paneLabels: Record<WorkspacePane, string>;
  onSelect: (index: number, pane: WorkspacePane) => void;
}) {
  return (
    <header className="flex h-control-md shrink-0 items-center justify-between gap-space-2xs border-b border-app-border bg-app-surface px-space-2xs">
      <details className="relative min-w-0">
        <summary
          data-testid="components-editor-split-pane-switcher"
          data-pane={pane}
          className="flex cursor-pointer list-none items-center gap-space-2xs rounded-control px-space-3xs py-space-3xs text-xs font-semibold text-app-foreground hover:bg-app-surface-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50"
        >
          {icon}
          <span className="truncate">{label}</span>
        </summary>
        <div className="absolute left-0 z-20 mt-space-3xs w-field rounded-control border border-app-border bg-app-surface p-space-3xs shadow-lg">
          {availablePanes.map((candidate) => (
            <Button
              key={candidate}
              type="button"
              variant="secondary"
              className="h-control-tight w-full justify-start px-space-2xs text-xs"
              onClick={() => onSelect(index, candidate)}
            >
              {paneLabels[candidate]}
            </Button>
          ))}
        </div>
      </details>
    </header>
  );
}

export function ComponentEditorView({ model }: { model: EditorViewModel }) {
  const {
    t,
    selectors,
    strings,
    chromeless,
    storiesQuery,
    libraryId,
    shellNavigation,
    dirty,
    previewReady,
    desktopLayout,
    currentPane,
    availablePanes,
    splitView,
    toggleSplitView,
    renderable,
    onClose,
    navigationSlot,
    contentQuery,
    activeVersionFiles,
    selectedVersion,
    splitLayout,
    saveDesktopPanelLayout,
    visiblePanes,
    visibleSpecimens,
    readyExamples,
    previewMessage,
    specimenErrors,
    specimenRetries,
    comparedSpecimens,
    frameEnabled,
    baselineSha,
    previewReloadKey,
    resolvedPreviewTheme,
    previewCanvasRef,
    id,
    activeVersion,
    releasedVersion,
    selectedFile,
    selectedTemplate,
    filesView,
    comparison,
    buffer,
    appResolvedTheme,
    fontSize,
    wordWrap,
    readOnly,
    saveMutation,
    handleBeforeMount,
    handleMount,
    selectFile,
    setSelectedTemplate,
    setFilesView,
    setBuffer,
    setWordWrap,
    setFontSize,
    onCloseComparison,
    previewStageRef,
    previewFullscreen,
    showSaved,
    previewExperienceState,
    paneLabels,
    selectSplitPane,
    specimens,
    activeSpecimen,
    specimenIdentity,
    activateSpecimen,
    framePickerEnabled,
    frameCandidatesQuery,
    frameOverride,
    compatibleFrameCandidates,
    setFrameSaveMessage,
    setFrameOverride,
    frameSaveMessage,
    persistFrameMutation,
    filters,
    previewKit,
    setPreviewKit,
    stageMode,
    setStageMode,
    setPreviewToolsCollapsed,
    setComparedSpecimens,
    comparisonActive,
    toggleComparison,
    selectAllComparison,
    emulator,
    togglePreviewTools,
    togglePreviewFullscreen,
    previewToolsCollapsed,
    editorToolProps,
    setActiveSpecimen,
    setSpecimenErrors,
    retrySpecimen,
    registerPreviewFrame,
    postToPreviewFrames,
    metadataSlot,
  } = model;
  return (
    <AssetWorkspace
      testId={selectors.components.editor.panel}
      label={t(strings.components.editor.title, { libraryId })}
    >
      {!chromeless && (
        <WorkspaceHeader
          title={<span data-testid={selectors.components.editor.title}>{libraryId}</span>}
          description={t(strings.components.editor.subtitle)}
          leading={
            shellNavigation.sidebarCollapsed ? (
              <button
                type="button"
                onClick={shellNavigation.openSidebar}
                aria-label={t("nav.openDrawer", { defaultValue: "Open navigation" })}
                data-testid="workspace-header-open-sidebar"
                className="touch-target inline-flex items-center justify-center rounded-control text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-foreground"
              >
                <Menu aria-hidden className="h-icon-md w-icon-md" />
              </button>
            ) : undefined
          }
          actions={
            <div className="flex items-center gap-space-2xs">
              {dirty && (
                <StatusBadge data-testid={selectors.components.editor.dirtyBadge} tone="warning">
                  {t(strings.components.editor.dirty)}
                </StatusBadge>
              )}
              {previewReady && (desktopLayout || currentPane === "preview") && (
                <StatusBadge data-testid={selectors.components.editor.previewBadge} tone="success">
                  {t(strings.components.editor.previewReady)}
                </StatusBadge>
              )}
              {availablePanes.length > 1 && (
                <IconButton
                  data-testid="components-editor-split-view-toggle"
                  aria-label={t("components.editor.splitView", { defaultValue: "Split view" })}
                  aria-pressed={splitView}
                  onClick={toggleSplitView}
                  className={`hidden h-control-tight min-h-control-tight min-w-control-tight lg:inline-flex ${splitView ? "bg-app-primary text-app-primary-foreground" : "border border-app-border bg-app-surface"}`}
                >
                  <PanelsLeftRight aria-hidden className="h-icon-compact w-icon-compact" />
                </IconButton>
              )}
              {renderable && (
                <IconButton
                  data-testid={selectors.components.editor.closeButton}
                  aria-label={t(strings.components.editor.close)}
                  onClick={onClose}
                  className="h-touch min-h-touch min-w-touch border border-app-border bg-app-surface"
                >
                  <ArrowLeft aria-hidden className="h-icon-compact w-icon-compact" />
                </IconButton>
              )}
            </div>
          }
        />
      )}

      {!chromeless && navigationSlot && (
        <div className="shrink-0 border-b border-app-border bg-app-surface px-space-sm">
          {navigationSlot}
        </div>
      )}

      {contentQuery.isLoading && <SourceLoadingSkeleton />}

      {contentQuery.error && (
        <p data-testid={selectors.components.editor.error} className="p-space-sm text-app-danger">
          {errorMessage(contentQuery.error, t)}
        </p>
      )}

      {saveMutation.error && (
        <p data-testid={selectors.components.editor.error} className="p-space-sm text-app-danger">
          {errorMessage(saveMutation.error, t)}
        </p>
      )}

      {(storiesQuery.data?.warnings?.length ?? 0) > 0 && (
        <aside
          role="status"
          aria-label="Story contract warnings"
          className="border-b border-app-warning/30 bg-app-warning/10 px-space-sm py-space-xs text-xs text-app-warning"
        >
          <p className="font-medium">
            {storiesQuery.data?.warnings?.length} story contract warning(s); indexing remains
            available.
          </p>
          <ul className="mt-space-2xs list-disc space-y-space-3xs pl-space-md">
            {storiesQuery.data?.warnings?.map((warning: string) => (
              <li key={warning}>{warning}</li>
            ))}
          </ul>
        </aside>
      )}

      {contentQuery.data && (
        <div className="min-h-0 flex-1">
          {selectedVersion && (
            <p className="border-b border-app-border bg-app-warning/10 px-space-sm py-space-2xs text-xs text-app-warning">
              {t(strings.components.editor.viewingVersion, { version: selectedVersion })}
            </p>
          )}
          <Group
            id="component-editor-panels"
            orientation={splitView && desktopLayout ? "horizontal" : "vertical"}
            defaultLayout={splitView && desktopLayout ? splitLayout : { primary: 100 }}
            onLayoutChanged={saveDesktopPanelLayout}
            className="h-full min-h-0"
          >
            {visiblePanes.map((pane: WorkspacePane, index: number) => (
              <Fragment key={pane}>
                <Panel
                  id={index === 0 ? "primary" : "secondary"}
                  minSize="15%"
                  defaultSize={splitView ? splitLayout[index === 0 ? "primary" : "secondary"] : 100}
                >
                  <div className="h-full min-h-0">
                    {pane === "files" && (
                      <ComponentEditorSource
                        id={id}
                        libraryId={libraryId}
                        renderable={renderable}
                        splitView={splitView}
                        version={activeVersion}
                        releasedVersion={releasedVersion}
                        activeVersionFiles={activeVersionFiles}
                        selectedFile={selectedFile}
                        selectedTemplate={selectedTemplate}
                        filesView={filesView}
                        comparison={comparison}
                        buffer={buffer}
                        appResolvedTheme={appResolvedTheme}
                        fontSize={fontSize}
                        wordWrap={wordWrap}
                        readOnly={readOnly}
                        dirty={dirty}
                        savePending={saveMutation.isPending}
                        contentLoading={contentQuery.isLoading}
                        handleBeforeMount={handleBeforeMount}
                        handleMount={handleMount}
                        onSelectFile={selectFile}
                        onSelectTemplate={setSelectedTemplate}
                        onFilesViewChange={setFilesView}
                        onBufferChange={setBuffer}
                        onSave={() => saveMutation.mutate()}
                        onRevert={() => setBuffer(contentQuery.data.content)}
                        onToggleWordWrap={() =>
                          setWordWrap((current: "on" | "off") => (current === "on" ? "off" : "on"))
                        }
                        onDecreaseFont={() =>
                          setFontSize((current: number) => Math.max(11, current - 1))
                        }
                        onIncreaseFont={() =>
                          setFontSize((current: number) => Math.min(20, current + 1))
                        }
                        onCloseComparison={() => {
                          onCloseComparison?.();
                          setFilesView("source");
                        }}
                      />
                    )}
                    {pane === "preview" && (
                      <div
                        ref={previewStageRef}
                        data-testid={selectors.components.editor.previewStage}
                        data-preview-fullscreen={previewFullscreen ? "true" : "false"}
                        className={
                          previewFullscreen
                            ? "fixed inset-0 z-50 h-full w-full bg-app-background"
                            : "h-full min-h-0"
                        }
                      >
                        <ExperienceSurface
                          surfaceId="component-preview"
                          state={previewExperienceState}
                          data-testid={selectors.components.editor.workspacePane}
                          data-preview-state={previewExperienceState}
                          data-pane="preview"
                          className="relative flex h-full min-h-0 min-w-0 w-full max-w-full flex-col overflow-hidden bg-app-background"
                        >
                          {splitView && (
                            <PaneHeader
                              pane="preview"
                              index={index}
                              label={paneLabels.preview}
                              icon={<Eye aria-hidden className="h-icon-compact w-icon-compact" />}
                              availablePanes={availablePanes}
                              paneLabels={paneLabels}
                              onSelect={selectSplitPane}
                            />
                          )}
                          <div
                            data-preview-floating-dock
                            role="toolbar"
                            aria-label="Preview controls"
                            className="absolute left-space-xs right-space-xs top-space-xs z-20 flex min-w-0 max-w-inline-overlay flex-wrap items-center gap-space-2xs rounded-panel border border-app-border/80 bg-app-surface/95 px-space-2xs py-space-2xs shadow-xl backdrop-blur"
                          >
                            {stageMode && (
                              <nav
                                className="order-last min-w-0 max-w-full basis-full overflow-x-auto border-t border-app-border/70 pt-space-2xs"
                                aria-label={t(strings.components.editor.storiesLabel)}
                              >
                                <div className="flex w-max gap-space-3xs">
                                  {specimens.map((example: any) => {
                                    const identity = specimenIdentity(example);
                                    const selected = identity === activeSpecimen;
                                    return (
                                      <Button
                                        key={identity}
                                        data-testid={selectors.components.editor.storyPickerItem}
                                        type="button"
                                        variant={selected ? "primary" : "secondary"}
                                        className="h-control-tight min-w-touch shrink-0 px-space-2xs text-xs"
                                        aria-current={selected ? "true" : undefined}
                                        title={example?.description}
                                        onClick={() => activateSpecimen(identity)}
                                      >
                                        {example?.displayName || example?.name || "Default"}
                                      </Button>
                                    );
                                  })}
                                </div>
                              </nav>
                            )}
                            {framePickerEnabled && (
                              <div className="flex min-w-0 items-center gap-space-3xs">
                                <label className="flex h-control items-center gap-space-3xs rounded-control border border-app-border/80 bg-app-surface px-space-2xs text-xs text-app-muted-foreground">
                                  <span className="sr-only">Preview frame</span>
                                  <select
                                    aria-label="Preview frame"
                                    data-testid="components-editor-frame-picker"
                                    className="h-full min-h-touch max-w-[12rem] bg-transparent text-app-foreground outline-none"
                                    value={frameOverride ? frameOverride.asset : ""}
                                    disabled={
                                      frameCandidatesQuery.isPending || frameCandidatesQuery.isError
                                    }
                                    onChange={(event) => {
                                      const next = compatibleFrameCandidates.find(
                                        (candidate: any) => candidate.asset === event.target.value,
                                      );
                                      setFrameSaveMessage("");
                                      setFrameOverride(next);
                                    }}
                                  >
                                    <option value="">
                                      {frameCandidatesQuery.isPending
                                        ? "Loading frames…"
                                        : frameCandidatesQuery.isError
                                          ? "Frames unavailable"
                                          : compatibleFrameCandidates.length === 0
                                            ? "No compatible frames"
                                            : "Story frame"}
                                    </option>
                                    {compatibleFrameCandidates.map((candidate: any) => (
                                      <option
                                        key={`${candidate.asset}:${candidate.region}`}
                                        value={candidate.asset}
                                      >
                                        {candidate.label} · {candidate.region}
                                      </option>
                                    ))}
                                  </select>
                                </label>
                                {frameOverride && (
                                  <Button
                                    type="button"
                                    variant="secondary"
                                    className="h-control-tight px-space-2xs text-xs"
                                    data-testid="components-editor-frame-save"
                                    disabled={persistFrameMutation.isPending}
                                    onClick={() => persistFrameMutation.mutate()}
                                  >
                                    {persistFrameMutation.isPending ? "Saving…" : "Save frame"}
                                  </Button>
                                )}
                                {frameSaveMessage && (
                                  <span
                                    role={persistFrameMutation.isError ? "alert" : "status"}
                                    className="max-w-[14rem] truncate text-xs text-app-muted-foreground"
                                    title={frameSaveMessage}
                                  >
                                    {frameSaveMessage}
                                  </span>
                                )}
                              </div>
                            )}
                            <IconButton
                              data-testid={selectors.components.editor.previewStageFullscreen}
                              type="button"
                              aria-label={
                                previewFullscreen
                                  ? t("components.editor.exitFullscreen", {
                                      defaultValue: "Exit full screen",
                                    })
                                  : t("components.editor.enterFullscreen", {
                                      defaultValue: "View preview full screen",
                                    })
                              }
                              aria-pressed={previewFullscreen}
                              onClick={() => {
                                void togglePreviewFullscreen();
                              }}
                              className="h-touch min-h-touch min-w-touch border border-app-border bg-app-surface"
                            >
                              {previewFullscreen ? (
                                <Minimize2 aria-hidden className="h-icon-compact w-icon-compact" />
                              ) : (
                                <Maximize2 aria-hidden className="h-icon-compact w-icon-compact" />
                              )}
                            </IconButton>
                            <ThemeSwitcher
                              previewReady={previewReady}
                              colorScheme={filters.colorScheme}
                              setColorScheme={filters.setColorScheme}
                              kit={previewKit}
                              setKit={setPreviewKit}
                              filters={filters}
                              compactOnMobile
                            />
                            <Button
                              type="button"
                              variant={stageMode ? "primary" : "secondary"}
                              className="h-control-tight px-space-2xs text-xs"
                              aria-pressed={stageMode}
                              data-testid="components-editor-stage-mode"
                              onClick={() => {
                                setStageMode((enabled: boolean) => !enabled);
                                setPreviewToolsCollapsed(true);
                                setComparedSpecimens(new Set());
                              }}
                            >
                              {stageMode ? "Focus" : "Canvas"}
                            </Button>
                            {!stageMode && specimens.length > 1 && (
                              <Button
                                type="button"
                                variant={comparisonActive ? "primary" : "secondary"}
                                className="h-control-tight px-space-2xs text-xs"
                                data-testid={selectors.components.editor.storySheetAll}
                                aria-pressed={comparisonActive}
                                onClick={selectAllComparison}
                              >
                                Story sheet ({Math.min(specimens.length, 4)})
                              </Button>
                            )}
                            <EmulatorToolbar emulator={emulator} compactOnMobile />
                            {activeSpecimen && (
                              <IconButton
                                data-testid={selectors.components.editor.previewToolsToggle}
                                aria-label={
                                  previewToolsCollapsed
                                    ? t("components.editor.showTools", {
                                        defaultValue: "Show controls",
                                      })
                                    : t("components.editor.hideTools", {
                                        defaultValue: "Hide controls",
                                      })
                                }
                                className="ml-auto h-touch min-h-touch min-w-touch border border-app-border bg-app-surface"
                                aria-expanded={!previewToolsCollapsed}
                                aria-controls="component-preview-tools"
                                onClick={togglePreviewTools}
                              >
                                <SlidersHorizontal
                                  aria-hidden
                                  className="h-icon-compact w-icon-compact"
                                />
                              </IconButton>
                            )}
                          </div>
                          <ComponentEditorStage
                            emulator={emulator}
                            filters={filters}
                            stageMode={stageMode}
                            comparisonActive={comparisonActive}
                            specimens={specimens}
                            visibleSpecimens={visibleSpecimens}
                            readyExamples={readyExamples}
                            previewMessage={previewMessage}
                            specimenErrors={specimenErrors}
                            specimenRetries={specimenRetries}
                            comparedSpecimens={comparedSpecimens}
                            activeSpecimen={activeSpecimen}
                            previewKit={previewKit}
                            frameEnabled={frameEnabled}
                            frameOverride={frameOverride}
                            id={id}
                            baselineSha={baselineSha}
                            previewReloadKey={previewReloadKey}
                            selectedVersion={selectedVersion}
                            resolvedPreviewTheme={resolvedPreviewTheme}
                            previewCanvasRef={previewCanvasRef}
                            toolsDocked={desktopLayout}
                            toolsOpen={desktopLayout && !previewToolsCollapsed}
                            onClearComparison={() => setComparedSpecimens(new Set())}
                            onSelectAllComparison={selectAllComparison}
                            onToggleComparison={toggleComparison}
                            onRetrySpecimen={retrySpecimen}
                            onRegisterPreviewFrame={registerPreviewFrame}
                            onPreviewLoad={(identity: string) => {
                              setActiveSpecimen((current: any) => current ?? identity);
                            }}
                            onPreviewError={(identity: string) =>
                              setSpecimenErrors((current: Record<string, string>) => ({
                                ...current,
                                [identity]: t(strings.components.editor.previewFailed),
                              }))
                            }
                            postToPreviewFrames={postToPreviewFrames}
                            onCloseTools={() => setPreviewToolsCollapsed(true)}
                            onEnterSpecimen={(identity) => {
                              setActiveSpecimen(identity);
                              setStageMode(true);
                              setComparedSpecimens(new Set());
                            }}
                            tools={<ComponentEditorTools {...editorToolProps} />}
                          />
                          {!desktopLayout && (
                            <ComponentEditorMobileTools
                              tool={previewToolsCollapsed ? null : "props"}
                              onClose={() => setPreviewToolsCollapsed(true)}
                              props={editorToolProps}
                            />
                          )}
                        </ExperienceSurface>
                      </div>
                    )}
                    {pane === "details" && (
                      <aside
                        data-testid={selectors.components.editor.workspacePane}
                        data-pane="details"
                        className="flex h-full min-h-0 flex-col overflow-hidden bg-app-surface"
                      >
                        {splitView && (
                          <PaneHeader
                            pane="details"
                            index={index}
                            label={paneLabels.details}
                            icon={<Info aria-hidden className="h-icon-compact w-icon-compact" />}
                            availablePanes={availablePanes}
                            paneLabels={paneLabels}
                            onSelect={selectSplitPane}
                          />
                        )}
                        <div
                          data-testid={selectors.components.editor.infoDialog}
                          className="min-h-0 flex-1 overflow-auto p-space-sm"
                        >
                          {metadataSlot ?? (
                            <p className="text-sm text-app-muted-foreground">
                              {t(strings.components.editor.noInfo)}
                            </p>
                          )}
                        </div>
                      </aside>
                    )}
                  </div>
                </Panel>
                {index < visiblePanes.length - 1 && (
                  <Separator className="w-separator shrink-0 bg-app-border hover:bg-app-primary" />
                )}
              </Fragment>
            ))}
          </Group>
        </div>
      )}

      {showSaved && (
        <p
          data-testid={selectors.components.editor.savedToast}
          className="absolute bottom-3 right-3 rounded-md bg-app-success/10 px-space-xs py-space-2xs text-xs text-app-success shadow-lg"
        >
          {t(strings.components.editor.saved)}
        </p>
      )}
    </AssetWorkspace>
  );
}
