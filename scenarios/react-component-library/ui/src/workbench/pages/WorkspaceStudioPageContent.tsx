import { useEffect, useRef, useState, type HTMLAttributes } from "react";
import { useSearchParams } from "react-router-dom";

import { useGlobalKeydown } from "../../hooks/useGlobalKeydown";
import { useTranslation } from "../../i18n";
import { WorkspaceStudioLoadState } from "./WorkspaceStudioLoadState";
import { useWorkspaceStudioViewport } from "./useWorkspaceStudioViewport";
import type { DesignDevice } from "../design-types";
import { readWorkbenchPreferences } from "../preferences";
import { cx } from "../cx";
import { historyShortcut } from "../studio/shortcuts";
import { useFocusTrap } from "../useFocusTrap";
import { type AiScope, type InspectorTab } from "../studio/StudioInspector";
import { useWorkspaceStudio } from "../studio/useWorkspaceStudio";
import { useWorkspaceStudioCapture } from "./useWorkspaceStudioCapture";
import { WorkspaceStudioCanvasArea } from "./WorkspaceStudioCanvasArea";
import { WorkspaceStudioInspectorPane } from "./WorkspaceStudioInspectorPane";
import { WorkspaceStudioLeftPane, type WorkspaceStudioLeftTab } from "./WorkspaceStudioLeftPane";
import { WorkspaceStudioPageHeader } from "./WorkspaceStudioPageHeader";
import { WorkspaceStudioPageControls } from "./WorkspaceStudioPageControls";
import { WorkspaceStudioPageOverlays } from "./WorkspaceStudioPageOverlays";
import type { AppliedPatch } from "../studio/patch";
import { WorkspaceStudioConflictBanner } from "./WorkspaceStudioConflictBanner";
import type { LoadedWorkspaceStudio } from "./workspaceStudioPageTypes";
import {
  createWorkspaceStudioPageActions,
  type WorkspaceStudioPageTranslate,
} from "./workspaceStudioPageActions";
import { hasLegacyCompositionAdapter } from "../workspaces/legacyCompositionPreview";
import { findSlot, type TemplatePalette } from "../templates/spec";
import { useWorkspaceStudioPageMetrics } from "./useWorkspaceStudioPageMetrics";
import "../studio/studio.css";
import "./workspace-studio.css";

type LeftTab = WorkspaceStudioLeftTab;

function isDevice(value: string | null): value is DesignDevice {
  return value === "desktop" || value === "tablet" || value === "phone";
}

/**
 * The workspace studio. Pages are tabs, displays are frames, and states are the
 * named variants behind each frame. It renders the same mature studio surfaces
 * as the design studio and writes through the revision-guarded workspace
 * aggregate, so there is one composition owner.
 */
export function WorkspaceStudioPageContent({ workspaceId }: { workspaceId: string }) {
  const { t } = useTranslation();
  const [searchParams, setSearchParams] = useSearchParams();
  const studio = useWorkspaceStudio(workspaceId);

  const [leftTab, setLeftTab] = useState<LeftTab>("layers");
  const [rightTab, setRightTab] = useState<InspectorTab>("inspect");
  const [aiScope, setAiScope] = useState<AiScope>("layer");
  const [sheet, setSheet] = useState<"left" | "right" | null>(null);
  const [presenting, setPresenting] = useState(false);
  const [htmlImportOpen, setHtmlImportOpen] = useState(false);
  const [capturePatchOpen, setCapturePatchOpen] = useState(false);
  const [notice, setNotice] = useState<string | null>(null);
  const [zoomOverride, setZoomOverride] = useState<number | null>(null);
  const [guides, setGuides] = useState(false);
  const [collapsed, setCollapsed] = useState<Set<string>>(() => new Set());
  const [token, setToken] = useState<keyof TemplatePalette | null>(null);
  const noticeTimer = useRef<number | null>(null);

  const design = studio.doc;
  const frames = studio.frames;
  const frame = studio.display
    ? frames.find((entry) => entry.id === studio.display?.id)
    : undefined;
  const componentSlot = frame ? findSlot(frame.spec) : undefined;
  const componentId =
    typeof componentSlot?.props.componentId === "string" ? componentSlot.props.componentId : "";
  const storyId =
    typeof componentSlot?.props.storyId === "string" ? componentSlot.props.storyId : "";
  const componentVersion =
    typeof componentSlot?.props.componentVersion === "string"
      ? componentSlot.props.componentVersion
      : "";
  const viewport = useWorkspaceStudioViewport(frames, frame, zoomOverride);
  const { canvasRef, narrow, fitScale, scales, audits, frameScale, presentScale } = viewport;

  const metrics = useWorkspaceStudioPageMetrics({
    design,
    frames,
    frame,
    selectedId: studio.selectedId,
    collapsed,
    audits,
    translate: t,
  });

  const flash = (message: string) => {
    setNotice(message);
    if (noticeTimer.current) window.clearTimeout(noticeTimer.current);
    noticeTimer.current = window.setTimeout(() => setNotice(null), 2600);
  };

  const { capturingState, captureSelectedState } = useWorkspaceStudioCapture({
    workspaceId,
    studio,
    translate: t as unknown as WorkspaceStudioPageTranslate,
    flash,
  });

  const updateURL = (next: Record<string, string | undefined>) => {
    const params = new URLSearchParams(searchParams);
    for (const [key, value] of Object.entries(next)) {
      if (value) params.set(key, value);
      else params.delete(key);
    }
    setSearchParams(params, { replace: true });
  };

  // The URL is the durable address of a page/display/state. Applying it once
  // after load keeps a shared link landing on the same composition.
  const appliedURL = useRef("");
  useEffect(() => {
    if (!studio.ready || !studio.workspace) return;
    // Apply the deep link once per workspace, not once per revision: autosave
    // bumps the revision and would otherwise snap the selection back.
    if (appliedURL.current === workspaceId) return;
    appliedURL.current = workspaceId;
    studio.selectAddress(
      searchParams.get("page") ?? "",
      searchParams.get("display") ?? "",
      searchParams.get("state") ?? "",
    );
    // Only the first load should move the selection; later edits are local.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [studio.ready, studio.workspace]);

  useEffect(() => {
    if (!studio.ready || !design) return;
    const requested = searchParams.get("device") ?? readWorkbenchPreferences().defaultDevice;
    if (!isDevice(requested)) return;
    const target = frames.find((entry) => entry.device === requested);
    if (target && studio.display?.id !== target.id) studio.selectDisplay(target.id);
    // Preference applies on open, not on every later selection.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [design?.id, studio.ready]);

  useEffect(() => {
    return () => {
      if (noticeTimer.current) window.clearTimeout(noticeTimer.current);
    };
  }, []);

  const undoHistory = studio.undo;
  const redoHistory = studio.redo;
  const saveNow = studio.save;
  useGlobalKeydown((event) => {
    const target = event.target;
    const editable =
      target instanceof HTMLElement &&
      Boolean(target.closest("input, textarea, select, [contenteditable=true]"));
    const action = historyShortcut({
      key: event.key,
      meta: event.metaKey,
      ctrl: event.ctrlKey,
      shift: event.shiftKey,
      editable,
    });
    if (action) {
      event.preventDefault();
      if (action === "undo") undoHistory();
      else redoHistory();
      return;
    }
    if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "s") {
      event.preventDefault();
      void saveNow();
    }
  });

  const leftSheetRef = useFocusTrap<HTMLElement>(sheet === "left", () => setSheet(null));
  const rightSheetRef = useFocusTrap<HTMLElement>(sheet === "right", () => setSheet(null));

  if (
    studio.loadError ||
    !studio.ready ||
    !studio.workspace ||
    !design ||
    !frame ||
    !studio.page ||
    !studio.display
  ) {
    return (
      <WorkspaceStudioLoadState
        loadError={studio.loadError}
        ready={Boolean(
          studio.ready && studio.workspace && design && frame && studio.page && studio.display,
        )}
      />
    );
  }

  const workspace = studio.workspace;
  const page = studio.page;
  // The editing panes are inert while a template is previewed, so a keyboard
  // user cannot land on a control that would silently refuse. The canvas stays
  // live: zooming, switching pages/states and presenting are inspection, not
  // editing, and they are how a person judges whether to fork.
  const inertWhenReadOnly = (studio.readOnly ? { inert: "" } : {}) as HTMLAttributes<HTMLElement>;
  const { rows, coverage, frameCoverageValue, selected, stateGap, claims, issueCount } = metrics;
  const pageActions = createWorkspaceStudioPageActions({
    studio,
    frames,
    frame,
    selected,
    translate: t as unknown as WorkspaceStudioPageTranslate,
    updateURL,
    flash,
  });

  return (
    <div
      className={cx("studio", sheet && `is-sheet-${sheet}`)}
      data-experience-surface="design-workspace"
    >
      {sheet ? (
        <button
          type="button"
          className="studio-backdrop"
          aria-label={t("workbench.studio.closePanel", { defaultValue: "Close panel" })}
          onClick={() => setSheet(null)}
        />
      ) : null}

      <div className="studio-main">
        <WorkspaceStudioConflictBanner studio={studio} />

        <WorkspaceStudioPageHeader
          studio={studio as LoadedWorkspaceStudio}
          workspace={workspace}
          frame={frame}
          actions={pageActions}
          onImportHtml={() => setHtmlImportOpen(true)}
          onImportPatch={() => setCapturePatchOpen(true)}
          updateURL={updateURL}
          onPresent={() => setPresenting(true)}
          flash={flash}
        />

        <div className="studio-body">
          <WorkspaceStudioLeftPane
            studio={studio as LoadedWorkspaceStudio}
            workspace={workspace}
            design={design}
            frames={frames}
            frame={frame}
            rows={rows}
            selected={selected}
            coverage={frameCoverageValue}
            leftTab={leftTab}
            onLeftTab={setLeftTab}
            leftSheetRef={leftSheetRef}
            inertWhenReadOnly={inertWhenReadOnly}
            onCloseSheet={() => setSheet(null)}
            setCollapsed={setCollapsed}
            actions={pageActions}
            flash={flash}
            token={token}
            onSelectToken={setToken}
          />

          <WorkspaceStudioCanvasArea
            workspaceId={workspaceId}
            studio={studio}
            design={design}
            frames={frames}
            frame={frame}
            display={studio.display}
            state={studio.state}
            selectedId={selected?.id ?? null}
            scales={scales}
            guides={guides}
            onToggleGuides={() => setGuides((value) => !value)}
            zoomOverride={zoomOverride}
            onZoomChange={setZoomOverride}
            narrow={narrow}
            frameScale={frameScale}
            fitScale={fitScale}
            audits={audits}
            componentId={componentId}
            storyId={storyId}
            componentVersion={componentVersion}
            selectFrame={pageActions.selectFrame}
            selectNode={pageActions.selectNode}
            reparentLayer={pageActions.reparentLayerInFrame}
            handleDevice={pageActions.handleDevice}
            dirty={studio.dirty}
            saving={studio.saving}
            readOnly={studio.readOnly}
            capturingState={capturingState}
            captureSelectedState={captureSelectedState}
            canvasRef={canvasRef}
            updateURL={updateURL}
          />

          <WorkspaceStudioInspectorPane
            studio={studio}
            workspace={workspace}
            page={page}
            design={design}
            frame={frame}
            frames={frames}
            selected={selected}
            commitFrame={pageActions.commitFrame}
            coverage={coverage}
            claims={claims}
            audits={audits}
            stateGap={stateGap}
            rightTab={rightTab}
            onRightTab={setRightTab}
            aiScope={aiScope}
            onAiScope={setAiScope}
            selectFrame={pageActions.selectFrame}
            flash={flash}
            rightSheetRef={rightSheetRef}
            inertWhenReadOnly={inertWhenReadOnly}
            onCloseSheet={() => setSheet(null)}
          />
        </div>

        <WorkspaceStudioPageControls
          doc={design}
          coverage={coverage}
          audits={audits}
          saved={studio.saved}
          dirty={studio.dirty}
          saving={studio.saving}
          saveFailed={studio.saveFailed}
          onRetrySave={() => void studio.save()}
          openCount={workspace.pages.length}
          readOnly={studio.readOnly}
          issueCount={issueCount}
          zoom={zoomOverride ?? (narrow ? frameScale : fitScale)}
          onOpenLeft={(tab) => {
            setLeftTab(tab);
            setSheet("left");
          }}
          onOpenRight={(tab) => {
            setRightTab(tab);
            setSheet("right");
          }}
        />
      </div>

      <WorkspaceStudioPageOverlays
        htmlImportOpen={htmlImportOpen}
        capturePatchOpen={capturePatchOpen}
        onCloseHtmlImport={() => setHtmlImportOpen(false)}
        onCloseCapturePatch={() => setCapturePatchOpen(false)}
        onImportHtml={(name, template) => {
          studio.addPage(name, template.spec);
          flash(
            t("workbench.workspace.htmlImported", {
              defaultValue: "Imported {{name}} as a new page.",
              name,
            }),
          );
        }}
        onApplyCapturePatch={(applied: AppliedPatch, patchId, summary) => {
          const patchedFrame = applied.doc.frames.find((entry) => entry.id === frame.id);
          if (!patchedFrame) {
            flash(t("workbench.studio.patchFrameMissing", { defaultValue: "The target frame is no longer open." }));
            return;
          }
          studio.updateFrame(frame.id, () => patchedFrame.spec, `Apply capture patch ${patchId}`);
          setCapturePatchOpen(false);
          flash(t("workbench.studio.patchApplied", { defaultValue: "Applied region patch: {{summary}}", summary }));
        }}
        notice={notice}
        design={design}
        frame={frame}
        readOnly={studio.readOnly}
        presenting={presenting}
        onClosePresentation={() => setPresenting(false)}
        presentScale={presentScale}
        legacyPreviewArgs={
          studio.state && hasLegacyCompositionAdapter(studio.state.argsJson)
            ? studio.state.argsJson
            : undefined
        }
      />
    </div>
  );
}
