import { useEffect, useMemo, useRef, useState, type HTMLAttributes } from "react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import { Check, Image, Layers, Palette, SlidersHorizontal, X } from "lucide-react";

import { forkWorkspace, listAllWorkspaceAssets } from "../../api/workspaces";
import { useTranslation } from "../../i18n";
import { appRoutes, workspacePath } from "../../routes";
import type { DesignDevice } from "../designs";
import { readWorkbenchPreferences } from "../preferences";
import { Skeleton, Tabs } from "../primitives";
import { cx } from "../cx";
import { useFocusTrap } from "../useFocusTrap";
import { historyShortcut } from "../studio/shortcuts";
import { StudioCanvas } from "../studio/StudioCanvas";
import { StudioInspector, type AiScope, type InspectorTab } from "../studio/StudioInspector";
import { HtmlImport } from "../studio/HtmlImport";
import { StudioAssets } from "../studio/StudioAssets";
import { AssetImportError, type ReferenceUpload } from "../studio/assets";
import { StudioLayers } from "../studio/StudioLayers";
import { StudioStatusBar } from "../studio/StudioStatusBar";
import { StudioSystem } from "../studio/StudioSystem";
import { WorkspaceTopBar } from "../studio/WorkspaceTopBar";
import { useFrameAudit, useElementWidth } from "../studio/useFrameAudit";
import { useWorkspaceStudio } from "../studio/useWorkspaceStudio";
import { proposeWorkspacePatch } from "../studio/workspaceAssist";
import {
  DEVICE_NAME,
  CONTAINER_TYPES,
  FRAME_SIZE,
  frameTemplate,
  visibleLayerRows,
  type AddableNode,
} from "../studio/model";
import { stateAudit } from "../studio/checks";
import { claimsFor } from "../studio/claims";
import { designCoverage, frameCoverage, type Coverage } from "../studio/rcl";
import {
  appendChild,
  duplicateNode,
  findNode,
  moveNode,
  node,
  removeNode,
  setProp,
  type TemplateNode,
  type TemplatePalette,
} from "../templates/spec";
import { TemplateRenderer } from "../templates/TemplateRenderer";
import "../studio/studio.css";
import "./workspace-studio.css";

type LeftTab = "layers" | "system" | "assets";
const EMPTY_COVERAGE: Coverage = { rcl: 0, custom: 0, total: 0, ratio: 0 };

function isDevice(value: string | null): value is DesignDevice {
  return value === "desktop" || value === "tablet" || value === "phone";
}

/**
 * The workspace studio. Pages are tabs, displays are frames, and states are the
 * named variants behind each frame. It renders the same mature studio surfaces
 * as the design studio and writes through the revision-guarded workspace
 * aggregate, so there is one composition owner.
 */
export function WorkspaceStudioPage({ workspaceId }: { workspaceId: string }) {
  const { t } = useTranslation();
  const [searchParams, setSearchParams] = useSearchParams();
  const navigate = useNavigate();
  const studio = useWorkspaceStudio(workspaceId);

  const [leftTab, setLeftTab] = useState<LeftTab>("layers");
  const [rightTab, setRightTab] = useState<InspectorTab>("inspect");
  const [aiScope, setAiScope] = useState<AiScope>("layer");
  const [sheet, setSheet] = useState<"left" | "right" | null>(null);
  const [presenting, setPresenting] = useState(false);
  const [htmlImportOpen, setHtmlImportOpen] = useState(false);
  const [forking, setForking] = useState(false);
  const [forkError, setForkError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [zoomOverride, setZoomOverride] = useState<number | null>(null);
  const [guides, setGuides] = useState(false);
  const [collapsed, setCollapsed] = useState<Set<string>>(() => new Set());
  const [token, setToken] = useState<keyof TemplatePalette | null>(null);
  const [viewportWidth, setViewportWidth] = useState(() =>
    typeof window === "undefined" ? 1440 : window.innerWidth,
  );
  const noticeTimer = useRef<number | null>(null);

  const design = studio.doc;
  const frames = studio.frames;
  const frame = studio.display ? frames.find((entry) => entry.id === studio.display?.id) : undefined;
  const canvasRef = useRef<HTMLDivElement>(null);
  const canvasWidth = useElementWidth(canvasRef);
  const narrow = canvasWidth > 0 && canvasWidth < 420;
  const fitScale =
    canvasWidth > 0
      ? Math.min(
          1,
          Math.max(
            0.1,
            Math.min(
              (canvasWidth - 44) / FRAME_SIZE.desktop.width,
              (canvasWidth - 104) / (FRAME_SIZE.tablet.width + FRAME_SIZE.phone.width + 40),
            ),
          ),
        )
      : 0.4;

  const scales = useMemo(() => {
    const map: Record<string, number> = {};
    for (const entry of frames) {
      const base = narrow
        ? Math.min(1, Math.max(0.1, (canvasWidth - 28) / FRAME_SIZE[entry.device].width))
        : fitScale;
      map[entry.id] = Math.min(1, Math.max(0.1, zoomOverride ?? base));
    }
    return map;
  }, [canvasWidth, fitScale, frames, narrow, zoomOverride]);

  const audits = useFrameAudit(canvasRef, frames, scales);
  const frameScale = frame ? (scales[frame.id] ?? fitScale) : fitScale;

  const rows = useMemo(
    () => (frame ? visibleLayerRows(frame.spec, collapsed) : []),
    [collapsed, frame],
  );
  const coverage = useMemo(() => (design ? designCoverage(design.frames) : EMPTY_COVERAGE), [design]);
  const frameCoverageValue = useMemo(
    () => (frame ? frameCoverage(frame.spec) : EMPTY_COVERAGE),
    [frame],
  );
  const selected = useMemo(
    () => (frame && studio.selectedId ? findNode(frame.spec, studio.selectedId) : undefined),
    [frame, studio.selectedId],
  );
  const stateGap = useMemo(
    () => (frame ? stateAudit(frame.spec) : { total: 0, covered: 0, missing: [] }),
    [frame],
  );

  const flash = (message: string) => {
    setNotice(message);
    if (noticeTimer.current) window.clearTimeout(noticeTimer.current);
    noticeTimer.current = window.setTimeout(() => setNotice(null), 2600);
  };

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
    const onResize = () => setViewportWidth(window.innerWidth);
    window.addEventListener("resize", onResize);
    return () => window.removeEventListener("resize", onResize);
  }, []);

  useEffect(() => {
    return () => {
      if (noticeTimer.current) window.clearTimeout(noticeTimer.current);
    };
  }, []);

  const undoHistory = studio.undo;
  const redoHistory = studio.redo;
  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
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
      if (!action) return;
      event.preventDefault();
      if (action === "undo") undoHistory();
      else redoHistory();
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [undoHistory, redoHistory]);

  const saveNow = studio.save;
  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "s") {
        event.preventDefault();
        void saveNow();
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [saveNow]);

  const leftSheetRef = useFocusTrap<HTMLElement>(sheet === "left", () => setSheet(null));
  const rightSheetRef = useFocusTrap<HTMLElement>(sheet === "right", () => setSheet(null));
  const presentRef = useFocusTrap<HTMLDivElement>(presenting, () => setPresenting(false));

  if (studio.loadError === "not-found") {
    return (
      <div className="wb-page" data-testid="workspace-studio">
        <div className="wb-empty">
          <b className="wb-empty-title">
            {t("workbench.workspace.unavailable", { defaultValue: "Workspace unavailable" })}
          </b>
          <p>{t("workbench.workspace.notFound", { defaultValue: "Workspace not found." })}</p>
          <Link className="wb-btn quiet" to={appRoutes.design}>
            {t("workbench.workspace.back", { defaultValue: "Back to workspaces" })}
          </Link>
        </div>
      </div>
    );
  }

  if (studio.loadError) {
    return (
      <div className="wb-page" data-testid="workspace-studio">
        <div className="wb-empty">
          <b className="wb-empty-title">
            {t("workbench.workspace.loadFailed", { defaultValue: "The workspace could not load" })}
          </b>
          <p>{studio.loadError}</p>
          <Link className="wb-btn quiet" to={appRoutes.design}>
            {t("workbench.workspace.back", { defaultValue: "Back to workspaces" })}
          </Link>
        </div>
      </div>
    );
  }

  if (!studio.ready || !studio.workspace || !design || !frame || !studio.page || !studio.display) {
    return (
      <div className="studio" data-testid="workspace-studio" aria-busy="true">
        <div className="studio-loading">
          <Skeleton height={24} />
          <Skeleton height={240} />
        </div>
      </div>
    );
  }

  const workspace = studio.workspace;
  const page = studio.page;
  // The editing panes are inert while a template is previewed, so a keyboard
  // user cannot land on a control that would silently refuse. The canvas stays
  // live: zooming, switching pages/states and presenting are inspection, not
  // editing, and they are how a person judges whether to fork.
  const inertWhenReadOnly = (studio.readOnly ? { inert: "" } : {}) as HTMLAttributes<HTMLElement>;
  const claims = claimsFor(frame.spec, t);
  const claimIssues = claims.filter(
    (claim) => claim.verdict === "warn" || claim.verdict === "fail",
  ).length;
  const overflowFrames = frames.filter((entry) => {
    const audit = audits[entry.id];
    return Boolean(audit?.measured && audit.overflow > 0);
  }).length;
  const targetsUnder = frames.some((entry) => (audits[entry.id]?.buttons?.under ?? 0) > 0);
  const issueCount =
    claimIssues + (overflowFrames > 0 ? 1 : 0) + (targetsUnder ? 1 : 0) + (stateGap.missing.length > 0 ? 1 : 0);
  const presentScale = Math.min(1, Math.max(0.1, (viewportWidth - 48) / FRAME_SIZE[frame.device].width));

  const selectFrame = (id: string) => {
    const target = frames.find((entry) => entry.id === id);
    studio.selectFrame(id);
    if (target) updateURL({ device: target.device, display: target.id });
  };

  const selectNode = (frameId: string, id: string) => {
    if (frameId !== frame.id) selectFrame(frameId);
    studio.selectNode(id);
  };

  const placeAsset = (asset: ReferenceUpload) => {
    const parentId = selected && CONTAINER_TYPES.has(selected.type) ? selected.id : frame.spec.id;
    commitFrame(
      (spec) =>
        appendChild(
          spec,
          parentId,
          node(`image-${asset.id}`, "image", {
            src: asset.url,
            alt: asset.name,
            label: asset.name,
            fit: "cover",
            h: 220,
          }),
        ),
      t("workbench.studio.placeAssetAction", {
        defaultValue: "Place {{name}} on the page",
        name: asset.name,
      }),
    );
    flash(
      t("workbench.studio.placeAssetDone", {
        defaultValue: "Placed {{name}} as a reference layer.",
        name: asset.name,
      }),
    );
  };

  const handleDevice = (next: DesignDevice) => {
    const target = frames.find((entry) => entry.device === next);
    if (target) {
      selectFrame(target.id);
      return;
    }
    studio.addFrame(next);
    updateURL({ device: next });
  };

  const commitFrame = (mutate: (spec: TemplateNode) => TemplateNode, label: string, mergeKey?: string) => {
    studio.updateFrame(frame.id, mutate, label, mergeKey);
  };

  const addLayer = (entry: AddableNode, parentId: string) => {
    const id = `${parentId}.${entry.type}-${Date.now().toString(36)}`;
    const children =
      entry.type === "card"
        ? [
            node(`${id}.title`, "text", { text: "New card", variant: "title" }),
            node(`${id}.body`, "text", {
              text: "A short supporting line.",
              variant: "body",
              tone: "muted",
            }),
          ]
        : undefined;
    commitFrame(
      (spec) => appendChild(spec, parentId, node(id, entry.type, { ...entry.props }, children)),
      t("workbench.studio.addLayerAction", {
        defaultValue: "Add {{type}}",
        type: t(entry.labelKey, { defaultValue: entry.label }),
      }),
    );
  };

  const moveLayer = (direction: -1 | 1) => {
    if (!selected || selected.id === frame.spec.id) return;
    commitFrame(
      (spec) => moveNode(spec, selected.id, direction),
      t("workbench.studio.moveLayer", { defaultValue: "Move layer" }),
    );
  };

  const duplicateLayer = () => {
    if (!selected || selected.id === frame.spec.id) return;
    commitFrame(
      (spec) => duplicateNode(spec, selected.id),
      t("workbench.studio.duplicateLayerAction", { defaultValue: "Duplicate layer" }),
    );
  };

  const removeLayer = () => {
    if (!selected || selected.id === frame.spec.id) return;
    commitFrame(
      (spec) => removeNode(spec, selected.id),
      t("workbench.studio.removeLayerAction", { defaultValue: "Remove layer" }),
    );
    studio.selectNode(null);
  };

  const share = () => {
    const url = window.location.href;
    void navigator.clipboard
      .writeText(url)
      .then(() => flash(t("workbench.studio.shareCopied", { defaultValue: "Link copied." })))
      .catch(() =>
        flash(
          t("workbench.studio.shareManual", {
            defaultValue: "Copy the address bar to share this design.",
          }),
        ),
      );
  };

  return (
    <div className={cx("studio", sheet && `is-sheet-${sheet}`)} data-experience-surface="design-workspace">
      {sheet ? (
        <button
          type="button"
          className="studio-backdrop"
          aria-label={t("workbench.studio.closePanel", { defaultValue: "Close panel" })}
          onClick={() => setSheet(null)}
        />
      ) : null}

      <div className="studio-main">
        {studio.conflict ? (
          <div className="studio-conflict" role="alert">
            <div>
              <b>
                {t("workbench.workspace.conflictTitle", {
                  defaultValue: "This workspace changed elsewhere",
                })}
              </b>
              <span>
                {t("workbench.workspace.conflictBody", {
                  defaultValue:
                    "Saving is paused so neither version is overwritten. Choose which one to keep.",
                })}
              </span>
            </div>
            <div className="studio-conflict-actions">
              <button
                type="button"
                className="studio-conflict-btn"
                onClick={() => studio.resolveConflict("take-theirs")}
              >
                {t("workbench.studio.conflictTheirs", { defaultValue: "Load the other version" })}
              </button>
              <button
                type="button"
                className="studio-conflict-btn is-primary"
                onClick={() => studio.resolveConflict("keep-mine")}
              >
                {t("workbench.studio.conflictMine", { defaultValue: "Keep mine" })}
              </button>
            </div>
          </div>
        ) : null}

        <WorkspaceTopBar
          workspaceName={workspace.name}
          pages={workspace.pages}
          activePageId={studio.page.id}
          onSelectPage={(id) => {
            studio.selectPage(id);
            updateURL({ page: id, display: undefined, state: undefined });
          }}
          onAddPage={() => {
            studio.addPage();
            flash(t("workbench.workspace.pageAdded", { defaultValue: "Added a page." }));
          }}
          onDuplicatePage={() => studio.duplicatePage()}
          onImportHtml={() => setHtmlImportOpen(true)}
          onRemovePage={() => {
            studio.removePage();
            flash(t("workbench.workspace.pageRemoved", { defaultValue: "Removed the page." }));
          }}
          canUndo={studio.history.index > 0}
          canRedo={studio.history.index < studio.history.entries.length}
          onUndo={studio.undo}
          onRedo={studio.redo}
          dirty={studio.dirty}
          saving={studio.saving}
          saveFailed={studio.saveFailed}
          onSave={() => void studio.save()}
          device={frame.device}
          onDevice={handleDevice}
          onPresent={() => setPresenting(true)}
          onShare={share}
          readOnly={studio.readOnly}
        />

        {studio.readOnly ? (
          <div
            className="studio-template-bar"
            role="status"
            aria-label={t("workbench.workspace.template", { defaultValue: "Curated template" })}
          >
            <span className="studio-template-badge">
              {t("workbench.workspace.templateChip", { defaultValue: "Template" })}
            </span>
            <p>
              <b>{t("workbench.workspace.templateTitle", { defaultValue: "Curated template" })}</b>
              {" · "}
              {t("workbench.workspace.templateBodyShort", {
                defaultValue: "A reviewed snapshot. Fork it to make it yours.",
              })}
            </p>
            {forkError ? <span role="alert">{forkError}</span> : null}
            <button
              type="button"
              className="wb-btn primary sm"
              disabled={forking}
              onClick={() => {
                setForking(true);
                setForkError(null);
                void forkWorkspace(workspace.id)
                  .then((forked) => navigate(workspacePath(forked.id), { replace: true }))
                  .catch((cause: unknown) =>
                    setForkError(
                      cause instanceof Error
                        ? cause.message
                        : t("workbench.workspace.forkFailed", {
                            defaultValue: "The template could not be forked.",
                          }),
                    ),
                  )
                  .finally(() => setForking(false));
              }}
            >
              {forking
                ? t("workbench.workspace.forking", { defaultValue: "Forking…" })
                : t("workbench.workspace.forkAction", { defaultValue: "Fork to edit" })}
            </button>
          </div>
        ) : null}

        <div className="studio-body">
          <aside
            ref={leftSheetRef}
            {...inertWhenReadOnly}
            className="studio-pane studio-left"
            aria-label={t("workbench.studio.layersPanel", {
              defaultValue: "Layers, system and assets",
            })}
          >
            <button
              type="button"
              className="studio-sheet-close"
              aria-label={t("workbench.studio.closePanel", { defaultValue: "Close panel" })}
              onClick={() => setSheet(null)}
            >
              <X aria-hidden size={15} />
            </button>
            <div className="studio-pane-tabs">
              <Tabs
                label={t("workbench.studio.leftTabs", { defaultValue: "Design panes" })}
                value={leftTab}
                onChange={setLeftTab}
                items={[
                  {
                    id: "layers",
                    label: t("workbench.studio.layers", { defaultValue: "Layers" }),
                    count: rows.length,
                  },
                  { id: "system", label: t("workbench.studio.system", { defaultValue: "System" }) },
                  {
                    id: "assets",
                    label: t("workbench.studio.assets", { defaultValue: "Assets" }),
                    count: studio.assets.length,
                  },
                ]}
              />
            </div>

            <div className={cx("studio-pane-scroll", leftTab === "layers" && "is-tight")}>
              {leftTab === "layers" ? (
                <StudioLayers
                  design={design}
                  frame={frame}
                  rows={rows}
                  selectedId={selected?.id ?? null}
                  coverage={frameCoverageValue}
                  onToggleCollapse={(id) =>
                    setCollapsed((prev) => {
                      const next = new Set(prev);
                      if (next.has(id)) next.delete(id);
                      else next.add(id);
                      return next;
                    })
                  }
                  onSelect={(id) => selectNode(frame.id, id)}
                  onMove={moveLayer}
                  onDuplicate={duplicateLayer}
                  onRemove={removeLayer}
                  onAdd={addLayer}
                  onRename={(name) =>
                    studio.renamePage(
                      name,
                      t("workbench.workspace.renamePage", { defaultValue: "Rename page" }),
                    )
                  }
                  footer={
                    <PageFields
                      pageName={studio.page.name}
                      route={studio.page.route}
                      pageTypeId={studio.page.pageTypeId}
                      stateName={studio.state?.name ?? ""}
                      canRemoveState={studio.display.states.length > 1}
                      onPageName={(value) =>
                        studio.updatePage(
                          { name: value },
                          t("workbench.workspace.renamePage", { defaultValue: "Rename page" }),
                          `page:${studio.page?.id}:name`,
                        )
                      }
                      onRoute={(value) =>
                        studio.updatePage(
                          { route: value },
                          t("workbench.workspace.editRoute", { defaultValue: "Edit page route" }),
                          `page:${studio.page?.id}:route`,
                        )
                      }
                      onPageType={(value) =>
                        studio.updatePage(
                          { pageTypeId: value },
                          t("workbench.workspace.editPageType", { defaultValue: "Edit page type" }),
                          `page:${studio.page?.id}:type`,
                        )
                      }
                      onStateName={(value) => studio.renameState(value)}
                      onRemoveState={() => studio.removeState()}
                    />
                  }
                />
              ) : null}

              {leftTab === "assets" ? (
                <StudioAssets
                  assets={studio.assets}
                  assetUrl={studio.assetUrl}
                  onImport={(file, fidelity) => {
                    void studio.importAsset(file, fidelity).catch((error: unknown) => {
                      if (error instanceof AssetImportError) {
                        flash(
                          error.reason === "type"
                            ? t("workbench.studio.assetType", {
                                defaultValue: "Only image files can be imported as references.",
                              })
                            : t("workbench.studio.assetTooBig", {
                                defaultValue: "That image is too large. Import one under 8 MB.",
                              }),
                        );
                        return;
                      }
                      flash(
                        t("workbench.studio.assetFailed", {
                          defaultValue: "The image could not be uploaded. Try again.",
                        }),
                      );
                    });
                  }}
                  onPlace={placeAsset}
                  onRemove={(assetId) => {
                    void studio.removeAsset(assetId).catch(() =>
                      flash(
                        t("workbench.studio.assetRemoveFailed", {
                          defaultValue: "That reference could not be removed. Try again.",
                        }),
                      ),
                    );
                  }}
                  loadOther={async () => {
                    const all = await listAllWorkspaceAssets();
                    return all.filter((asset) => asset.workspaceId !== workspace.id);
                  }}
                />
              ) : null}

              {leftTab === "system" ? (
                <StudioSystem
                  theme={design.theme}
                  frames={frames}
                  activeFrame={frame}
                  token={token}
                  onSelectToken={setToken}
                  onTokenChange={(key, value) =>
                    studio.updateTheme(
                      { [key]: value },
                      t("workbench.studio.editTokenAction", {
                        defaultValue: "Edit {{token}} token",
                        token: key,
                      }),
                      `theme:${key}`,
                    )
                  }
                  onSelectNode={(id) => selectNode(frame.id, id)}
                  onSelectFrame={selectFrame}
                  coverage={frameCoverageValue}
                  options={design.options}
                  onOptionsChange={(patch, label) => studio.updateOptions(patch, label)}
                />
              ) : null}
            </div>
          </aside>

          <div className="studio-center">
            <div
              className="studio-statebar"
              role="group"
              aria-label={t("workbench.workspace.states", { defaultValue: "States" })}
            >
              <span className="studio-statebar-label">
                {DEVICE_NAME[frame.device]} ·{" "}
                {t("workbench.workspace.statesLabel", { defaultValue: "states" })}
              </span>
              {studio.display.states.map((entry) => (
                <button
                  key={entry.id}
                  type="button"
                  aria-pressed={entry.id === studio.state?.id}
                  className={cx("studio-chip-btn", entry.id === studio.state?.id && "is-active")}
                  onClick={() => {
                    studio.selectState(entry.id);
                    updateURL({ state: entry.id });
                  }}
                >
                  {entry.name}
                </button>
              ))}
              <button type="button" className="studio-chip-btn" onClick={() => studio.addState()}>
                {t("workbench.workspace.addState", { defaultValue: "Add state" })}
              </button>
            </div>

            <StudioCanvas
              design={design}
              frames={frames}
              activeFrameId={frame.id}
              selectedId={selected?.id ?? null}
              scales={scales}
              guides={guides}
              onToggleGuides={() => setGuides((value) => !value)}
              zoomOverride={zoomOverride}
              onZoomChange={setZoomOverride}
              fitScale={narrow ? frameScale : fitScale}
              audit={audits}
              slot={undefined}
              slotEditable={false}
              onPlacementChange={() => undefined}
              onSelectFrame={selectFrame}
              onSelectNode={selectNode}
              onAddFrame={handleDevice}
              onRemoveFrame={(id) => studio.removeFrame(id)}
              scrollRef={canvasRef}
              saving={studio.saving}
            />
          </div>

          <aside
            ref={rightSheetRef}
            {...inertWhenReadOnly}
            className="studio-pane studio-right"
            aria-label={t("workbench.studio.inspector", { defaultValue: "Inspector" })}
          >
            <button
              type="button"
              className="studio-sheet-close"
              aria-label={t("workbench.studio.closePanel", { defaultValue: "Close panel" })}
              onClick={() => setSheet(null)}
            >
              <X aria-hidden size={15} />
            </button>
            <StudioInspector
              tab={rightTab}
              onTab={setRightTab}
              design={design}
              frame={frame}
              theme={design.theme}
              selected={selected ?? null}
              onPropChange={(key, value) => {
                if (!selected) return;
                commitFrame(
                  (spec) => setProp(spec, selected.id, key, value),
                  t("workbench.studio.editProp", { defaultValue: "Edit {{key}}", key }),
                  `prop:${frame.id}:${selected.id}:${key}`,
                );
              }}
              coverage={coverage}
              claims={claims}
              frames={frames}
              audits={audits}
              stateGap={stateGap}
              aiScope={aiScope}
              onAiScope={setAiScope}
              onSelectFrame={selectFrame}
              onApplyProposal={studio.applyProposal}
              onApplied={(summary) =>
                flash(
                  t("workbench.studio.aiApplied", {
                    defaultValue: "Applied “{{summary}}” as one undoable step.",
                    summary,
                  }),
                )
              }
              propose={(input, signal) =>
                proposeWorkspacePatch(
                  workspace.id,
                  {
                    ...input,
                    pageId: page.id,
                    activeStates: page.displays.flatMap((entry) => {
                      const stateId = studio.stateByDisplay[entry.id] ?? entry.states[0]?.id ?? "";
                      return stateId ? [{ displayId: entry.id, stateId }] : [];
                    }),
                  },
                  signal,
                )
              }
            />
          </aside>
        </div>

        <div className="studio-mobile-bar">
          <button
            type="button"
            className="studio-mobile-btn"
            onClick={() => {
              setLeftTab("layers");
              setSheet("left");
            }}
          >
            <Layers aria-hidden size={14} />
            {t("workbench.studio.layers", { defaultValue: "Layers" })}
          </button>
          <button
            type="button"
            className="studio-mobile-btn"
            onClick={() => {
              setLeftTab("system");
              setSheet("left");
            }}
          >
            <Palette aria-hidden size={14} />
            {t("workbench.studio.system", { defaultValue: "System" })}
          </button>
          <button
            type="button"
            className="studio-mobile-btn"
            onClick={() => {
              setLeftTab("assets");
              setSheet("left");
            }}
          >
            <Image aria-hidden size={14} />
            {t("workbench.studio.assets", { defaultValue: "Assets" })}
          </button>
          <button
            type="button"
            className="studio-mobile-btn"
            onClick={() => {
              setRightTab("inspect");
              setSheet("right");
            }}
          >
            <SlidersHorizontal aria-hidden size={14} />
            {t("workbench.studio.inspect", { defaultValue: "Inspect" })}
          </button>
          <button
            type="button"
            className="studio-mobile-btn"
            onClick={() => {
              setRightTab("checks");
              setSheet("right");
            }}
          >
            <Check aria-hidden size={14} />
            {t("workbench.studio.checks", { defaultValue: "Checks" })}
          </button>
        </div>

        <StudioStatusBar
          doc={design}
          coverage={coverage}
          audits={audits}
          saved={studio.saved}
          dirty={studio.dirty}
          saving={studio.saving}
          saveFailed={studio.saveFailed}
          onRetrySave={() => void studio.save()}
          unavailable={0}
          fallback={false}
          openCount={workspace.pages.length}
          openUnit="page"
          readOnly={studio.readOnly}
          issueCount={issueCount}
          zoom={zoomOverride ?? (narrow ? frameScale : fitScale)}
        />
      </div>

      {htmlImportOpen ? (
        <HtmlImport
          onClose={() => setHtmlImportOpen(false)}
          onImport={(template, name) => {
            setHtmlImportOpen(false);
            studio.addPage(name, template.spec);
            flash(
              t("workbench.workspace.htmlImported", {
                defaultValue: "Imported {{name}} as a new page.",
                name,
              }),
            );
          }}
        />
      ) : null}

      {notice ? (
        <div className="studio-toast" role="status">
          <Check aria-hidden size={13} />
          {notice}
        </div>
      ) : null}

      {presenting ? (
        <div
          ref={presentRef}
          role="dialog"
          aria-modal="true"
          aria-label={t("workbench.studio.presentLabel", { defaultValue: "Present composition" })}
          className="studio-present"
        >
          <div className="studio-present-close">
            <button
              type="button"
              className="studio-iconbtn"
              aria-label={t("workbench.studio.closePresent", {
                defaultValue: "Close presentation",
              })}
              onClick={() => setPresenting(false)}
            >
              <X aria-hidden size={16} />
            </button>
          </div>
          <div
            className="studio-present-frame"
            style={{ width: FRAME_SIZE[frame.device].width, zoom: presentScale }}
          >
            <TemplateRenderer
              template={frameTemplate(design, frame)}
              spec={frame.spec}
              mode="preview"
            />
          </div>
        </div>
      ) : null}
    </div>
  );
}

function PageFields({
  pageName,
  route,
  pageTypeId,
  stateName,
  canRemoveState,
  onPageName,
  onRoute,
  onPageType,
  onStateName,
  onRemoveState,
}: {
  pageName: string;
  route: string;
  pageTypeId: string;
  stateName: string;
  canRemoveState: boolean;
  onPageName: (value: string) => void;
  onRoute: (value: string) => void;
  onPageType: (value: string) => void;
  onStateName: (value: string) => void;
  onRemoveState: () => void;
}) {
  const { t } = useTranslation();
  return (
    <div className="studio-page-fields">
      <span className="wb-caption">{t("workbench.workspace.pageContext", { defaultValue: "Page" })}</span>
      <label>
        <span className="wb-label">{t("workbench.workspace.pageName", { defaultValue: "Name" })}</span>
        <input
          className="wb-input"
          value={pageName}
          onChange={(event) => onPageName(event.target.value)}
        />
      </label>
      <label>
        <span className="wb-label">{t("workbench.workspace.pageRoute", { defaultValue: "Route" })}</span>
        <input
          className="wb-input"
          value={route}
          placeholder={t("workbench.workspace.routePlaceholder", {
            defaultValue: "/onboarding/welcome",
          })}
          onChange={(event) => onRoute(event.target.value)}
        />
      </label>
      <label>
        <span className="wb-label">{t("workbench.workspace.pageType", { defaultValue: "Page type" })}</span>
        <input
          className="wb-input"
          value={pageTypeId}
          placeholder={t("workbench.workspace.pageTypePlaceholder", {
            defaultValue: "templates.landing-page",
          })}
          onChange={(event) => onPageType(event.target.value)}
        />
      </label>
      <span className="wb-caption">
        {t("workbench.workspace.stateSection", { defaultValue: "State" })}
      </span>
      <label>
        <span className="wb-label">{t("workbench.workspace.stateName", { defaultValue: "Name" })}</span>
        <input
          className="wb-input"
          value={stateName}
          onChange={(event) => onStateName(event.target.value)}
        />
      </label>
      <div className="studio-shelf-actions">
        <button
          type="button"
          className="studio-chip-btn is-danger"
          onClick={onRemoveState}
          disabled={!canRemoveState}
        >
          {t("workbench.workspace.removeState", { defaultValue: "Remove state" })}
        </button>
      </div>
    </div>
  );
}
