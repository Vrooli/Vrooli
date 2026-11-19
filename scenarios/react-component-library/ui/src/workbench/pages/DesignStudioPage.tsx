import { useEffect, useMemo, useRef, useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router-dom";
import { Check, Image, Layers, Palette, SlidersHorizontal, X } from "lucide-react";

import { useTranslation } from "../../i18n";
import { designPath } from "../../routes";
import { useComponentBundle, useDesignPages, useDesignScenarios } from "../data";
import { FixtureFrame } from "../component/FixtureFrame";
import { TemplateRenderer } from "../templates/TemplateRenderer";
import { DEFAULT_TEMPLATE } from "../templates/library";
import {
  blankTemplate,
  localTemplateId,
  removeLocalTemplate,
  resolveTemplate,
  templateShelf,
  upsertLocalTemplate,
  useLocalTemplates,
} from "../templates/local";
import { readWorkbenchPreferences } from "../preferences";
import {
  appendChild,
  duplicateNode,
  findSlot,
  findNode,
  moveNode,
  node,
  removeNode,
  setProp,
  setSlotPlacement,
  slotPlacement,
  walk,
  type Template,
  type TemplateNode,
} from "../templates/spec";
import { Select, Skeleton, Tabs } from "../primitives";
import { cx } from "../cx";
import { useFocusTrap } from "../useFocusTrap";
import { AssetImportError, useStudio } from "../studio/useStudio";
import { CONTAINER_TYPES } from "../studio/model";
import { historyShortcut } from "../studio/shortcuts";
import { TEMPLATE_ERROR_EVENT } from "../templates/local";
import type { DesignAsset } from "../designs";
import { StudioTopBar } from "../studio/StudioTopBar";
import { HtmlImport } from "../studio/HtmlImport";
import { StudioLayers } from "../studio/StudioLayers";
import type { AddableNode } from "../studio/model";
import { StudioSystem } from "../studio/StudioSystem";
import { StudioAssets } from "../studio/StudioAssets";
import { StudioCanvas } from "../studio/StudioCanvas";
import {
  StudioInspector,
  type AiScope,
  type InspectorTab,
  type StudioClaim,
} from "../studio/StudioInspector";
import { StudioStatusBar } from "../studio/StudioStatusBar";
import {
  DEVICE_KEY,
  DEVICE_NAME,
  FRAME_SIZE,
  docSignature,
  frameTemplate,
  visibleLayerRows,
} from "../studio/model";
import { designCoverage, frameCoverage, type Coverage } from "../studio/rcl";
import { captureEvidenceVerdict, primaryActionVerdict, stateAudit } from "../studio/checks";
import { useElementWidth, useFrameAudit } from "../studio/useFrameAudit";
import type { DesignDevice } from "../designs";
import type { TemplatePalette } from "../templates/spec";
import "../studio/studio.css";

type LeftTab = "layers" | "system" | "assets";

const EMPTY_FRAMES: never[] = [];
const EMPTY_COVERAGE: Coverage = { rcl: 0, custom: 0, total: 0, ratio: 0 };

function isDevice(value: string | null): value is DesignDevice {
  return value === "desktop" || value === "tablet" || value === "phone";
}

/**
 * The Design studio shell: rail, design tabs, docked layers/system/assets,
 * device frames, inspector and status bar. State and history live in
 * `useStudio`; pure maths live beside it; this file composes them.
 */
export function DesignStudioPage() {
  const { t } = useTranslation();
  const [searchParams, setSearchParams] = useSearchParams();
  const routeParams = useParams<{ scenario?: string; page?: string }>();
  const navigate = useNavigate();
  const scenarios = useDesignScenarios();
  const scenario = routeParams.scenario ?? searchParams.get("scenario");
  const pages = useDesignPages(scenario);
  const page = routeParams.page ?? searchParams.get("page");
  const studio = useStudio({ scenario, page });
  const componentId = searchParams.get("component") ?? "";
  const storyId = searchParams.get("story") ?? "";
  const locals = useLocalTemplates();
  const shelf = useMemo(() => templateShelf(locals), [locals]);

  const [leftTab, setLeftTab] = useState<LeftTab>("layers");
  const [rightTab, setRightTab] = useState<InspectorTab>("inspect");
  const [sheet, setSheet] = useState<"left" | "right" | null>(null);
  const [presenting, setPresenting] = useState(false);
  const [notice, setNotice] = useState<string | null>(null);
  const [htmlImportOpen, setHtmlImportOpen] = useState(false);
  const [zoomOverride, setZoomOverride] = useState<number | null>(null);
  const [guides, setGuides] = useState(false);
  const [collapsed, setCollapsed] = useState<Set<string>>(() => new Set());
  const [token, setToken] = useState<keyof TemplatePalette | null>(null);
  const [aiScope, setAiScope] = useState<AiScope>("layer");
  const [templateId, setTemplateId] = useState(DEFAULT_TEMPLATE.id);
  const [viewportWidth, setViewportWidth] = useState(() =>
    typeof window === "undefined" ? 1440 : window.innerWidth,
  );
  const noticeTimer = useRef<number | null>(null);

  const design = studio.activeDoc;
  const frames = design?.frames ?? EMPTY_FRAMES;
  const frame = studio.activeFrame;
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
  const coverage = useMemo(
    () => (design ? designCoverage(design.frames) : EMPTY_COVERAGE),
    [design],
  );
  const frameCoverageValue = useMemo(
    () => (frame ? frameCoverage(frame.spec) : EMPTY_COVERAGE),
    [frame],
  );
  const selected = useMemo(
    () => (frame && studio.selectedId ? findNode(frame.spec, studio.selectedId) : undefined),
    [frame, studio.selectedId],
  );
  const stateGap = useMemo(() => (frame ? stateAudit(frame.spec) : { total: 0, covered: 0, missing: [] }), [frame]);
  const assets = design ? studio.assetsFor(design.id) : [];
  const slotNode = frame ? findSlot(frame.spec) : undefined;
  const placement = slotPlacement(slotNode);

  const updateURL = (next: Record<string, string | undefined>) => {
    const params = new URLSearchParams(searchParams);
    for (const [key, value] of Object.entries(next)) {
      if (value) params.set(key, value);
      else params.delete(key);
    }
    setSearchParams(params, { replace: true });
  };

  const selectScenario = (next: string) => {
    if (routeParams.scenario) {
      void navigate(designPath(next));
      return;
    }
    updateURL({ scenario: next, page: undefined });
  };

  const selectPage = (next: string) => {
    if (routeParams.page) {
      void navigate(designPath(scenario ?? undefined, next));
      return;
    }
    updateURL({ page: next });
  };

  const flash = (message: string) => {
    setNotice(message);
    if (noticeTimer.current) window.clearTimeout(noticeTimer.current);
    noticeTimer.current = window.setTimeout(() => setNotice(null), 2600);
  };

  const placeAsset = (asset: DesignAsset) => {
    if (!frame) return;
    const parentId =
      selected && CONTAINER_TYPES.has(selected.type) ? selected.id : frame.spec.id;
    const src = `/api/designs/${asset.designId}/assets/${asset.id}`;
    commitFrame(
      (spec) =>
        appendChild(
          spec,
          parentId,
          node(`image-${asset.id}`, "image", {
            src,
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

  useEffect(() => {
    return () => {
      if (noticeTimer.current) window.clearTimeout(noticeTimer.current);
    };
  }, []);

  // A shelf write that fell back to this browser must not look saved. The
  // template module emits the failure after the optimistic cache update.
  useEffect(() => {
    const onShelfError = (event: Event) => {
      const detail = (event as CustomEvent<{ message?: string } | undefined>).detail;
      flash(
        detail?.message
          ? t("workbench.studio.templateLocalOnly", {
              defaultValue: "{{message}} The template still exists in this browser only.",
              message: detail.message,
            })
          : t("workbench.studio.templateLocalOnlyShort", {
              defaultValue:
                "The template could not be saved to the server. It still exists in this browser only.",
            }),
      );
    };
    window.addEventListener(TEMPLATE_ERROR_EVENT, onShelfError);
    return () => window.removeEventListener(TEMPLATE_ERROR_EVENT, onShelfError);
    // `flash` only touches a ref and state setters; the listener is registered once.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // A design tool without Cmd/Ctrl+Z would be the only one. The shortcut never
  // fires while a field has focus, so text editing keeps its own undo.
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

  // A page owns at most one design. Opening a page selects that design, and
  // creates one from the current template when the page has none yet. The ref
  // guard makes the creation happen once per page even though the effect sees
  // every edit.
  const boundRequest = useRef<string | null>(null);
  useEffect(() => {
    if (!studio.ready || !scenario || !page) return;
    const bound = studio.docs.find((doc) => doc.scenario === scenario && doc.page === page);
    if (bound) {
      if (bound.id !== studio.activeDoc?.id) studio.selectDesign(bound.id);
      return;
    }
    const key = `${scenario}/${page}`;
    if (boundRequest.current === key) return;
    boundRequest.current = key;
    const title = pages.data?.find((entry) => entry.page === page)?.title || page;
    // A page-bound design starts blank: seeding it with the Aperture sample
    // would present an unrelated composition as this page's design.
    void studio
      .createFromTemplate(blankTemplate(title), title, { scenario, page })
      .then(() =>
        flash(
          t("workbench.studio.boundCreated", {
            defaultValue: "Created a design for {{page}}",
            page: title,
          }),
        ),
      )
      .catch(() => {
        boundRequest.current = null;
      });
    // `flash` is stable in behaviour; the ref guard is what prevents repeats.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [studio.ready, studio.docs, studio.activeDoc?.id, scenario, page, pages.data]);

  useEffect(() => {
    const onResize = () => setViewportWidth(window.innerWidth);
    window.addEventListener("resize", onResize);
    return () => window.removeEventListener("resize", onResize);
  }, []);

  useEffect(() => {
    if (!studio.ready || !design) return;
    const requested = searchParams.get("device") ?? readWorkbenchPreferences().defaultDevice;
    if (!isDevice(requested)) return;
    const target = design.frames.find((entry) => entry.device === requested);
    if (target && target.id !== frame?.id) studio.selectFrame(target.id);
    // Only the stored preference should move the first frame, not every
    // selection the person makes afterwards.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [design?.id, studio.ready]);

  const claims = useMemo<StudioClaim[]>(() => {
    if (!frame) return [];
    let primary = 0;
    let displays = 0;
    let titles = 0;
    let bodies = 0;
    let slots = 0;
    walk(frame.spec, (current) => {
      if (current.type === "button" && current.props.variant === "primary") primary += 1;
      if (current.type === "slot") slots += 1;
      if (current.type === "text" && current.props.variant === "display") displays += 1;
      if (current.type === "text" && current.props.variant === "title") titles += 1;
      if (current.type === "text" && current.props.variant === "body") bodies += 1;
    });
    return [
      {
        id: "slot",
        label: t("workbench.studio.checkSlot", {
          defaultValue: "Composition declares a component slot",
        }),
        verdict: slots > 0 ? "pass" : "warn",
        detail: componentId
          ? placement
            ? t("workbench.studio.slotPlaced", {
                defaultValue: "Live component placed · {{width}} × {{height}}",
                width: Math.round(placement.w),
                height: Math.round(placement.h),
              })
            : t("workbench.studio.slotInFlow", {
                defaultValue: "Live component sits in the designed slot",
              })
          : t("workbench.studio.slotCount", {
              count: slots,
            }),
      },
      {
        id: "primary",
        label: t("workbench.studio.checkPrimary", { defaultValue: "A primary action is declared" }),
        verdict: primaryActionVerdict(primary),
        detail: t("workbench.studio.primaryCount", {
          defaultValue: "{{buttons}} primary buttons · {{slots}} component slots",
          buttons: primary,
          slots,
        }),
      },
      {
        id: "hierarchy",
        label: t("workbench.studio.checkHierarchy", {
          defaultValue: "Display, title and body hierarchy exists",
        }),
        verdict: displays > 0 && titles > 0 && bodies > 0 ? "pass" : "warn",
        detail: t("workbench.studio.hierarchyCount", {
          defaultValue: "{{display}} display · {{title}} title · {{body}} body",
          display: displays,
          title: titles,
          body: bodies,
        }),
      },
      {
        id: "evidence",
        label: t("workbench.studio.checkEvidence", {
          defaultValue: "Capture evidence is attached",
        }),
        verdict: captureEvidenceVerdict(false),
        detail: componentId
          ? t("workbench.studio.evidenceLiveOnly", {
              defaultValue: "Live component attached · no capture for this revision",
            })
          : t("workbench.studio.evidenceNone", {
              defaultValue: "No captured evidence for this revision",
            }),
      },
      {
        id: "contrast",
        label: t("workbench.studio.checkContrast", { defaultValue: "Contrast is measured" }),
        verdict: "unmeasured",
        detail: t("workbench.studio.contrastNote", {
          defaultValue: "Not captured for this local composition",
        }),
      },
    ];
  }, [componentId, frame, placement, t]);

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
  const dirty = Boolean(design && studio.saved[design.id]?.signature !== docSignature(design));

  const leftSheetRef = useFocusTrap<HTMLElement>(sheet === "left", () => setSheet(null));
  const rightSheetRef = useFocusTrap<HTMLElement>(sheet === "right", () => setSheet(null));
  const presentRef = useFocusTrap<HTMLDivElement>(presenting, () => setPresenting(false));

  if (!design || !frame) {
    return (
      <div className="studio" data-experience-surface="design-overview">
        <div className="studio-loading">
          <Skeleton height={24} />
          <Skeleton height={240} />
        </div>
      </div>
    );
  }

  const selectedLayer = selected;
  const presentScale = Math.min(
    1,
    Math.max(0.1, (viewportWidth - 48) / FRAME_SIZE[frame.device].width),
  );

  const selectNode = (frameId: string, id: string) => {
    if (frameId !== frame.id) studio.selectFrame(frameId);
    studio.selectNode(id);
  };

  const selectFrame = (id: string) => {
    const target = frames.find((entry) => entry.id === id);
    studio.selectFrame(id);
    if (target) updateURL({ device: target.device });
  };

  const handleDevice = (next: DesignDevice) => {
    const target = frames.find((entry) => entry.device === next);
    if (target) {
      selectFrame(target.id);
      return;
    }
    studio.addFrame(
      next,
      DEVICE_NAME[next],
      t("workbench.studio.addFrameAction", {
        defaultValue: "Add {{device}} frame",
        device: t(DEVICE_KEY[next], { defaultValue: DEVICE_NAME[next] }),
      }),
    );
    updateURL({ device: next });
  };

  const commitFrame = (
    mutate: (spec: TemplateNode) => TemplateNode,
    label: string,
    mergeKey?: string,
  ) => {
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
    if (!selectedLayer || selectedLayer.id === frame.spec.id) return;
    commitFrame(
      (spec) => moveNode(spec, selectedLayer.id, direction),
      t("workbench.studio.moveLayer", { defaultValue: "Move layer" }),
    );
  };

  const duplicateLayer = () => {
    if (!selectedLayer || selectedLayer.id === frame.spec.id) return;
    commitFrame(
      (spec) => duplicateNode(spec, selectedLayer.id),
      t("workbench.studio.duplicateLayerAction", { defaultValue: "Duplicate layer" }),
    );
  };

  const removeLayer = () => {
    if (!selectedLayer || selectedLayer.id === frame.spec.id) return;
    commitFrame(
      (spec) => removeNode(spec, selectedLayer.id),
      t("workbench.studio.removeLayerAction", { defaultValue: "Remove layer" }),
    );
    studio.selectNode(null);
  };

  const applyTemplate = (id: string) => {
    const found = resolveTemplate(id, locals);
    if (!found) return;
    setTemplateId(id);
    updateURL({ template: id });
    studio.applyTemplate(
      found,
      t("workbench.studio.applyTemplate", {
        defaultValue: "Apply template {{name}}",
        name: found.name,
      }),
    );
    flash(
      t("workbench.studio.appliedTemplate", { defaultValue: "Applied {{name}}.", name: found.name }),
    );
  };

  const saveFrameAsTemplate = () => {
    const source = frameTemplate(design, frame);
    const name = t("workbench.studio.frameTemplateName", {
      defaultValue: "{{design}} · {{frame}}",
      design: design.name,
      frame: frame.name,
    });
    const copy: Template = {
      ...source,
      id: localTemplateId(name),
      name,
      tagline: design.name,
      local: true,
    };
    upsertLocalTemplate(copy);
    setTemplateId(copy.id);
    flash(
      t("workbench.studio.savedTemplate", {
        defaultValue: "Saved {{name}} to the template shelf.",
        name,
      }),
    );
  };

  const newTemplate = () => {
    const fresh = blankTemplate();
    upsertLocalTemplate(fresh);
    setTemplateId(fresh.id);
    void studio.createFromTemplate(fresh).then(() => {
      flash(t("workbench.studio.createdTemplate", { defaultValue: "Created a blank template." }));
    });
  };

  const deleteTemplate = () => {
    const found = resolveTemplate(templateId, locals);
    if (!found?.local) return;
    removeLocalTemplate(templateId);
    setTemplateId(DEFAULT_TEMPLATE.id);
    updateURL({ template: DEFAULT_TEMPLATE.id });
    flash(t("workbench.studio.deletedTemplate", { defaultValue: "Deleted the template." }));
  };

  const share = () => {
    const url = window.location.href;
    try {
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
    } catch {
      flash(
        t("workbench.studio.shareManual", {
          defaultValue: "Copy the address bar to share this design.",
        }),
      );
    }
  };

  const openAssets = () => {
    setLeftTab("assets");
    setSheet("left");
  };

  const openEvidence = () => {
    setRightTab("checks");
    setSheet("right");
  };

  const importHtmlDesign = (template: Template, name: string) => {
    setHtmlImportOpen(false);
    // An imported page is its own design. Leaving a page route first prevents
    // the binding effect from immediately selecting that page's design.
    if (scenario || page) void navigate(designPath());
    void studio
      .createFromTemplate(template, name, { scenario: null, page: null })
      .then(() =>
        flash(
          t("workbench.studio.htmlImported", {
            defaultValue: "Imported {{name}} as a new design.",
            name,
          }),
        ),
      )
      .catch(() =>
        flash(
          t("workbench.studio.htmlImportFailed", {
            defaultValue: "The HTML page could not be imported. Nothing was changed.",
          }),
        ),
      );
  };

  return (
    <div className={cx("studio", sheet && `is-sheet-${sheet}`)} data-experience-surface="design-overview">
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
                {t("workbench.studio.conflictTitle", {
                  defaultValue: "This design changed in another tab",
                })}
              </b>
              <span>
                {t("workbench.studio.conflictBody", {
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

        <StudioTopBar
          designs={studio.docs}
          openIds={studio.openDocs.map((entry) => entry.id)}
          activeId={design.id}
          onSelect={studio.selectDesign}
          onClose={studio.closeDesign}
          onNew={() => {
            void studio.createFromTemplate(blankTemplate()).then(() => {
              flash(t("workbench.studio.createdTemplate", { defaultValue: "Created a blank template." }));
            });
          }}
          onOpen={(id) => {
            studio.openDesign(id);
          }}
          onImportHtml={() => setHtmlImportOpen(true)}
          onDelete={() => void studio.deleteActive()}
          onDuplicate={() => {
            void studio.duplicateActive(
              t("workbench.studio.duplicateName", {
                defaultValue: "{{name}} copy",
                name: design.name,
              }),
            );
          }}
          canUndo={studio.history.index > 0}
          canRedo={studio.history.index < studio.history.entries.length}
          onUndo={studio.undo}
          onRedo={studio.redo}
          history={studio.history}
          onJump={studio.jump}
          device={frame.device}
          onDevice={handleDevice}
          onShare={share}
          onPresent={() => setPresenting(true)}
          dirty={dirty}
        />

        <div className="studio-body">
          <aside
            ref={leftSheetRef}
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
                  {
                    id: "system",
                    label: t("workbench.studio.system", { defaultValue: "System" }),
                  },
                  {
                    id: "assets",
                    label: t("workbench.studio.assets", { defaultValue: "Assets" }),
                    count: assets.length,
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
                  selectedId={selectedLayer?.id ?? null}
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
                    studio.renameDesign(
                      name,
                      t("workbench.studio.renameDesign", { defaultValue: "Rename design" }),
                    )
                  }
                  footer={
                    <div className="studio-shelf">
                      <span className="wb-caption">
                        {t("workbench.studio.templates", { defaultValue: "Templates" })}
                      </span>
                      <Select
                        aria-label={t("workbench.studio.templates", { defaultValue: "Templates" })}
                        value={templateId}
                        onChange={(event) => setTemplateId(event.target.value)}
                      >
                        {shelf.map((entry) => (
                          <option key={entry.id} value={entry.id}>
                            {entry.name}
                            {entry.local
                              ? t("workbench.studio.shelfDraftSuffix", { defaultValue: " · draft" })
                              : ""}{" "}
                            — {entry.tagline}
                          </option>
                        ))}
                      </Select>
                      <div className="studio-shelf-actions">
                        <button
                          type="button"
                          className="studio-chip-btn"
                          title={t("workbench.studio.applyTemplateHint", {
                            defaultValue:
                              "Replaces the frames in this design. Undo restores the previous composition.",
                          })}
                          onClick={() => applyTemplate(templateId)}
                        >
                          {t("workbench.studio.applyTemplateAction", { defaultValue: "Apply template" })}
                        </button>
                        <button type="button" className="studio-chip-btn" onClick={saveFrameAsTemplate}>
                          {t("workbench.studio.saveFrameTemplate", {
                            defaultValue: "Save frame as template",
                          })}
                        </button>
                        <button type="button" className="studio-chip-btn" onClick={newTemplate}>
                          {t("workbench.studio.newTemplate", { defaultValue: "New template" })}
                        </button>
                        {resolveTemplate(templateId, locals)?.local ? (
                          <button
                            type="button"
                            className="studio-chip-btn is-danger"
                            onClick={deleteTemplate}
                          >
                            {t("workbench.studio.deleteTemplate", { defaultValue: "Delete template" })}
                          </button>
                        ) : null}
                      </div>

                      <details className="studio-context-details">
                        <summary className="wb-caption">
                          {t("workbench.studio.context", { defaultValue: "Page context" })}
                        </summary>
                        <div className="studio-context">
                          <label>
                            <span className="wb-label">
                              {t("workbench.studio.scenario", { defaultValue: "Scenario" })}
                            </span>
                            <Select
                              aria-label={t("workbench.studio.scenario", { defaultValue: "Scenario" })}
                              value={scenario ?? ""}
                              onChange={(event) => selectScenario(event.target.value)}
                            >
                              <option value="">
                                {t("workbench.studio.noScenario", { defaultValue: "None" })}
                              </option>
                              {(scenarios.data ?? []).map((entry) => (
                                <option key={entry.scenario} value={entry.scenario}>
                                  {entry.scenario}
                                </option>
                              ))}
                              {scenario &&
                              !(scenarios.data ?? []).some((entry) => entry.scenario === scenario) ? (
                                <option value={scenario}>{scenario}</option>
                              ) : null}
                            </Select>
                          </label>
                          <label>
                            <span className="wb-label">
                              {t("workbench.studio.page", { defaultValue: "Page" })}
                            </span>
                            <Select
                              aria-label={t("workbench.studio.page", { defaultValue: "Page" })}
                              value={page ?? ""}
                              onChange={(event) => selectPage(event.target.value)}
                            >
                              <option value="">
                                {t("workbench.studio.noPageShort", { defaultValue: "None" })}
                              </option>
                              {(pages.data ?? []).map((entry) => (
                                <option key={entry.page} value={entry.page}>
                                  {entry.title || entry.page}
                                </option>
                              ))}
                              {page && !(pages.data ?? []).some((entry) => entry.page === page) ? (
                                <option value={page}>{page}</option>
                              ) : null}
                            </Select>
                          </label>
                          <p className="studio-note">
                            {page
                              ? t("workbench.studio.contextBound", {
                                  defaultValue:
                                    "Bound to {{page}}. Choosing another page opens or creates that page's design.",
                                  page,
                                })
                              : t("workbench.studio.contextNote", {
                                  defaultValue:
                                    "Choose a page to bind this design. Each page keeps its own design.",
                                })}
                          </p>
                        </div>
                      </details>
                    </div>
                  }
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

              {leftTab === "assets" ? (
                <StudioAssets
                  designId={design.id}
                  assets={assets}
                  onImport={(file, fidelity) => {
                    void studio
                      .importAsset(file, fidelity)
                      .catch((error: unknown) => {
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
                />
              ) : null}
            </div>
          </aside>

          <StudioCanvas
            design={design}
            frames={frames}
            activeFrameId={frame.id}
            selectedId={selectedLayer?.id ?? null}
            scales={scales}
            guides={guides}
            onToggleGuides={() => setGuides((value) => !value)}
            zoomOverride={zoomOverride}
            onZoomChange={setZoomOverride}
            fitScale={narrow ? frameScale : fitScale}
            audit={audits}
            slot={
              componentId ? (
                <StudioSlot
                  componentId={componentId}
                  storyId={storyId}
                  template={frameTemplate(design, frame)}
                  height={placement?.h ?? 56}
                />
              ) : undefined
            }
            slotEditable={Boolean(componentId)}
            onPlacementChange={(frameId, nodeId, next) =>
              studio.updateFrame(
                frameId,
                (spec) => setSlotPlacement(spec, nodeId, next),
                t("workbench.studio.moveSlot", { defaultValue: "Move component slot" }),
                `slot:${frameId}:${nodeId}`,
              )
            }
            onSelectFrame={selectFrame}
            onSelectNode={selectNode}
            onAddFrame={handleDevice}
            onRemoveFrame={(id) =>
              studio.removeFrame(id, t("workbench.studio.removeFrameAction", { defaultValue: "Remove frame" }))
            }
            scrollRef={canvasRef}
            saving={studio.saving}
          />

          <aside
            ref={rightSheetRef}
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
              selected={selectedLayer ?? null}
              onPropChange={(key, value) => {
                if (!selectedLayer) return;
                commitFrame(
                  (spec) => setProp(spec, selectedLayer.id, key, value),
                  t("workbench.studio.editProp", { defaultValue: "Edit {{key}}", key }),
                  `prop:${frame.id}:${selectedLayer.id}:${key}`,
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
          <button type="button" className="studio-mobile-btn" onClick={openAssets}>
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
          <button type="button" className="studio-mobile-btn" onClick={openEvidence}>
            <Check aria-hidden size={14} />
            {t("workbench.studio.checks", { defaultValue: "Checks" })}
          </button>
        </div>

        <StudioStatusBar
          doc={design}
          coverage={coverage}
          audits={audits}
          saved={studio.saved[design.id]}
          dirty={dirty}
          saving={studio.saving}
          saveFailed={studio.saveFailed}
          onRetrySave={studio.retrySave}
          unavailable={studio.unavailable}
          fallback={studio.fallback}
          openCount={studio.openDocs.length}
          issueCount={issueCount}
          zoom={zoomOverride ?? (narrow ? frameScale : fitScale)}
        />
      </div>

      {htmlImportOpen ? (
        <HtmlImport onClose={() => setHtmlImportOpen(false)} onImport={importHtmlDesign} />
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
            style={{
              width: FRAME_SIZE[frame.device].width,
              zoom: presentScale,
            }}
          >
            <TemplateRenderer
              template={frameTemplate(design, frame)}
              spec={frame.spec}
              mode="preview"
              slot={
                componentId ? (
                  <StudioSlot
                    componentId={componentId}
                    storyId={storyId}
                    template={frameTemplate(design, frame)}
                    height={placement?.h ?? 56}
                  />
                ) : undefined
              }
            />
          </div>
        </div>
      ) : null}
    </div>
  );
}

function StudioSlot({
  componentId,
  storyId,
  template,
  height,
}: {
  componentId: string;
  storyId: string;
  template: Template;
  height: number;
}) {
  const { t } = useTranslation();
  const bundle = useComponentBundle(componentId);
  if (bundle.isLoading) return <Skeleton height={height} />;
  if (!bundle.data) {
    return (
      <span className="tpl-slot-hint">
        {t("workbench.studio.slotUnavailable", { defaultValue: "Component unavailable" })}
      </span>
    );
  }
  const story = bundle.data.stories.find((entry) => entry.id === storyId) ?? bundle.data.stories[0];
  if (!story)
    return (
      <span className="tpl-slot-hint">
        {t("workbench.studio.slotNoStory", { defaultValue: "No story" })}
      </span>
    );
  return (
    <FixtureFrame
      assetId={componentId}
      storyId={story.id}
      version={bundle.data.asset.latestVersion || bundle.data.asset.version || ""}
      height={height}
      title={t("workbench.studio.slotTitle", {
        defaultValue: "{{name}} in {{template}}",
        name: story.name || story.id,
        template: template.name,
      })}
    />
  );
}
