import { useEffect, useMemo, useRef, useState, type CSSProperties } from "react";
import { pluralDefault } from "../plural";
import { Link, useSearchParams } from "react-router-dom";
import {
  ArrowDown,
  ArrowUp,
  Check,
  ChevronRight,
  CircleAlert,
  Copy,
  Eye,
  FilePlus2,
  Layers,
  Monitor,
  Plus,
  RotateCcw,
  SlidersHorizontal,
  Smartphone,
  Sparkles,
  Tablet,
  Trash2,
  X,
} from "lucide-react";

import { useTranslation } from "../../i18n";
import { appRoutes } from "../../routes";
import { formatRelative, useComponentBundle, useDesignPages, useDesignScenarios } from "../data";
import { FixtureFrame } from "../component/FixtureFrame";
import { TemplateRenderer } from "../templates/TemplateRenderer";
import { DEFAULT_TEMPLATE } from "../templates/library";
import {
  TEMPLATE_PALETTES,
  blankTemplate,
  cloneTemplate,
  localTemplateId,
  readLocalTemplates,
  removeLocalTemplate,
  resolveTemplate,
  templateShelf,
  upsertLocalTemplate,
  useLocalTemplates,
  withPalette,
} from "../templates/local";
import { readWorkbenchPreferences } from "../preferences";
import {
  appendChild,
  clearSlotPlacement,
  duplicateNode,
  findSlot,
  layerLabel,
  layerRows,
  moveNode,
  node,
  removeNode,
  setProp,
  setSlotPlacement,
  slotPlacement,
  walk,
  type Template,
  type TemplateNode,
  type TemplateNodeType,
} from "../templates/spec";
import {
  Banner,
  Empty,
  Field,
  Segmented,
  Select,
  Skeleton,
  TextInput,
  Toggle,
  WbButton,
} from "../primitives";
import { cx } from "../cx";

type Device = "desktop" | "tablet" | "phone";
type InspectorTab = "props" | "theme" | "contract" | "compare";

const DEVICE_WIDTH: Record<Device, number> = { desktop: 1180, tablet: 834, phone: 390 };
const CONTAINERS = new Set<TemplateNode["type"]>([
  "page",
  "stack",
  "row",
  "grid",
  "card",
  "surface",
]);
const DEVICE_HEIGHT: Record<Device, number> = { desktop: 760, tablet: 900, phone: 760 };

interface Revision {
  at: number;
  hash: string;
  spec: TemplateNode;
}

/** Presentation fields a draft may customize without cloning the composition. */
type TemplatePresentation = Pick<
  Template,
  "name" | "tagline" | "slotHint" | "idea" | "palette" | "display" | "radius"
>;

function presentationOf(template: Template): TemplatePresentation {
  return {
    name: template.name,
    tagline: template.tagline,
    slotHint: template.slotHint,
    idea: template.idea,
    palette: template.palette,
    display: template.display,
    radius: template.radius,
  };
}

function storageKey(templateId: string): string {
  return `rcl.workbench.design.${templateId}.v1`;
}

function readDraft(
  templateId: string,
): { spec: TemplateNode; revisions: Revision[]; template?: TemplatePresentation } | null {
  try {
    const raw = window.localStorage.getItem(storageKey(templateId));
    if (!raw) return null;
    const parsed = JSON.parse(raw) as {
      spec?: TemplateNode;
      revisions?: Revision[];
      template?: TemplatePresentation;
    };
    if (!parsed.spec || !parsed.spec.id.startsWith(templateId)) return null;
    return { spec: parsed.spec, revisions: parsed.revisions ?? [], template: parsed.template };
  } catch {
    return null;
  }
}

function applyPresentation(base: Template, saved?: TemplatePresentation): Template {
  if (!saved) return base;
  return {
    ...base,
    ...saved,
    palette: { ...base.palette, ...saved.palette },
    spec: base.spec,
  };
}

interface AddableNode {
  type: TemplateNodeType;
  label: string;
  props: Record<string, string | number | boolean>;
}

const ADDABLE: AddableNode[] = [
  {
    type: "text",
    label: "Text",
    props: { text: "New line of copy", variant: "body", tone: "muted" },
  },
  { type: "text", label: "Heading", props: { text: "A stronger headline", variant: "display" } },
  { type: "card", label: "Card", props: { pad: 18, elevated: true } },
  { type: "stack", label: "Stack", props: { gap: 14 } },
  { type: "row", label: "Row", props: { gap: 10, wrap: true } },
  { type: "grid", label: "Grid", props: { gap: 14, cols: 2 } },
  { type: "button", label: "Action", props: { text: "Action", variant: "ghost", size: "sm" } },
  { type: "input", label: "Input", props: { placeholder: "Field" } },
  { type: "chip", label: "Chip", props: { text: "New" } },
  { type: "stat", label: "Stat", props: { label: "Measure", value: "0", detail: "sample" } },
  { type: "meter", label: "Meter", props: { label: "Progress", value: 42 } },
  { type: "divider", label: "Divider", props: {} },
  { type: "spacer", label: "Spacer", props: { h: 24 } },
  { type: "swatch", label: "Block", props: { h: 120, shape: "arch", tone: "accent" } },
];

function hashSpec(spec: TemplateNode): string {
  const text = JSON.stringify(spec);
  let hash = 0;
  for (let index = 0; index < text.length; index += 1) {
    hash = (hash * 31 + text.charCodeAt(index)) | 0;
  }
  return Math.abs(hash).toString(16).padStart(8, "0");
}

export function DesignStudioPage() {
  const { t } = useTranslation();
  const [searchParams, setSearchParams] = useSearchParams();
  const scenarios = useDesignScenarios();
  const initialScenario = searchParams.get("scenario");
  const [scenario, setScenario] = useState<string | null>(initialScenario);
  const pages = useDesignPages(scenario);
  const [page, setPage] = useState<string | null>(searchParams.get("page"));
  const [templateId, setTemplateId] = useState(searchParams.get("template") ?? DEFAULT_TEMPLATE.id);
  const [device, setDevice] = useState<Device>(() => {
    const requested = searchParams.get("device");
    if (requested === "desktop" || requested === "tablet" || requested === "phone")
      return requested;
    return readWorkbenchPreferences().defaultDevice;
  });
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [inspectorTab, setInspectorTab] = useState<InspectorTab>("props");
  const [presenting, setPresenting] = useState(false);
  const [mobilePanel, setMobilePanel] = useState<"layers" | "inspector" | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  const locals = useLocalTemplates();
  const shelf = useMemo(() => templateShelf(locals), [locals]);
  const base = resolveTemplate(templateId, locals) ?? DEFAULT_TEMPLATE;
  const [template, setTemplate] = useState<Template>(() =>
    applyPresentation(base, readDraft(base.id)?.template),
  );
  const [spec, setSpec] = useState<TemplateNode>(() => readDraft(base.id)?.spec ?? base.spec);
  const [revisions, setRevisions] = useState<Revision[]>(() => readDraft(base.id)?.revisions ?? []);

  useEffect(() => {
    const found = resolveTemplate(templateId, readLocalTemplates()) ?? DEFAULT_TEMPLATE;
    const saved = readDraft(found.id);
    setSpec(saved?.spec ?? found.spec);
    setRevisions(saved?.revisions ?? []);
    setTemplate(applyPresentation(found, saved?.template));
    setSelectedId(null);
  }, [templateId]);

  useEffect(() => {
    if (!scenario && scenarios.data && scenarios.data.length > 0) {
      const preferred = scenarios.data.find(
        (entry) => entry.scenario === "react-component-library",
      );
      const first = scenarios.data[0];
      if (first) setScenario(preferred?.scenario ?? first.scenario);
    }
  }, [scenario, scenarios.data]);

  useEffect(() => {
    const first = pages.data?.[0];
    if (!page && first) setPage(first.page);
  }, [page, pages.data]);

  const rows = useMemo(() => layerRows(spec), [spec]);
  const selected = useMemo(
    () => rows.find((row) => row.node.id === selectedId)?.node,
    [rows, selectedId],
  );
  const componentId = searchParams.get("component") ?? "";
  const storyId = searchParams.get("story") ?? "";
  const currentPage = pages.data?.find((entry) => entry.page === page);
  const slotNode = useMemo(() => findSlot(spec), [spec]);
  const placement = slotPlacement(slotNode);

  const checks = useMemo(() => {
    let primary = 0;
    let displays = 0;
    let titles = 0;
    let bodies = 0;
    let slots = 0;
    walk(spec, (current) => {
      if (current.type === "button" && current.props.variant === "primary") primary += 1;
      if (current.type === "slot") slots += 1;
      if (current.type === "text" && current.props.variant === "display") displays += 1;
      if (current.type === "text" && current.props.variant === "title") titles += 1;
      if (current.type === "text" && current.props.variant === "body") bodies += 1;
    });
    return [
      {
        id: "route",
        label: t("workbench.studio.checkRoute", { defaultValue: "Page route is declared" }),
        verdict: currentPage?.route ? "pass" : "unmeasured",
        detail:
          currentPage?.route ??
          t("workbench.studio.noPage", { defaultValue: "No design page selected" }),
      },
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
              defaultValue: pluralDefault(slots, "{{count}} slot", "{{count}} slots"),
              count: slots,
            }),
      },
      {
        id: "primary",
        label: t("workbench.studio.checkPrimary", { defaultValue: "A primary action is declared" }),
        verdict: primary + slots > 0 ? "pass" : "warn",
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
        detail: `${displays} display · ${titles} title · ${bodies} body`,
      },
      {
        id: "evidence",
        label: t("workbench.studio.checkEvidence", {
          defaultValue: "Component evidence is attached",
        }),
        verdict: componentId ? "pass" : "unmeasured",
        detail: componentId
          ? t("workbench.studio.evidenceAttached", { defaultValue: "Live component attached" })
          : t("workbench.studio.noComponent", { defaultValue: "No component attached" }),
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
  }, [componentId, currentPage?.route, placement, spec, t]);

  const updateURL = (next: Record<string, string | undefined>) => {
    const params = new URLSearchParams(searchParams);
    for (const [key, value] of Object.entries(next)) {
      if (value) params.set(key, value);
      else params.delete(key);
    }
    setSearchParams(params, { replace: true });
  };

  const flash = (message: string) => {
    setNotice(message);
    window.setTimeout(() => setNotice(null), 2400);
  };

  const save = (specToSave: TemplateNode, message: string) => {
    const revision: Revision = { at: Date.now(), hash: hashSpec(specToSave), spec: specToSave };
    const nextRevisions = [revision, ...revisions].slice(0, 10);
    setRevisions(nextRevisions);
    try {
      window.localStorage.setItem(
        storageKey(template.id),
        JSON.stringify({
          spec: specToSave,
          revisions: nextRevisions,
          template: presentationOf(template),
        }),
      );
    } catch {
      /* local draft only */
    }
    if (template.local) {
      upsertLocalTemplate({ ...template, spec: specToSave });
    }
    flash(message);
  };

  const reset = () => {
    const found = resolveTemplate(templateId, readLocalTemplates()) ?? DEFAULT_TEMPLATE;
    setSpec(found.spec);
    setRevisions([]);
    setTemplate(found);
    try {
      window.localStorage.removeItem(storageKey(found.id));
    } catch {
      /* ignore */
    }
    flash(t("workbench.studio.resetDone", { defaultValue: "Restored the template baseline." }));
  };

  const selectTemplate = (id: string) => {
    setTemplateId(id);
    updateURL({ template: id });
  };

  const duplicateTemplate = () => {
    const copy = cloneTemplate(template, `${template.name} copy`);
    upsertLocalTemplate(copy);
    selectTemplate(copy.id);
    flash(
      t("workbench.studio.duplicated", {
        defaultValue: "Created {{name}} as a browser-local template.",
        name: copy.name,
      }),
    );
  };

  const createTemplate = () => {
    const fresh = blankTemplate();
    upsertLocalTemplate(fresh);
    selectTemplate(fresh.id);
    flash(t("workbench.studio.created", { defaultValue: "Created a blank template." }));
  };

  const deleteTemplate = () => {
    if (!template.local) return;
    removeLocalTemplate(template.id);
    selectTemplate(DEFAULT_TEMPLATE.id);
    flash(t("workbench.studio.deleted", { defaultValue: "Deleted the local template." }));
  };

  const saveAsTemplate = () => {
    const name = t("workbench.studio.copyName", {
      defaultValue: "{{name}} study",
      name: template.name,
    });
    const copy: Template = {
      ...template,
      id: localTemplateId(name),
      name,
      spec,
      local: true,
    };
    upsertLocalTemplate(copy);
    selectTemplate(copy.id);
    flash(
      t("workbench.studio.savedAs", {
        defaultValue: "Saved {{name}} as a new template.",
        name,
      }),
    );
  };

  const updateTemplate = (patch: Partial<Template>) => {
    setTemplate((current) => ({ ...current, ...patch }));
  };

  const addNodeOfType = (parentId: string, entry: AddableNode) => {
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
    setSpec(appendChild(spec, parentId, node(id, entry.type, { ...entry.props }, children)));
  };

  const width = DEVICE_WIDTH[device];
  const height = DEVICE_HEIGHT[device];
  const areaRef = useRef<HTMLDivElement>(null);
  const artboardRef = useRef<HTMLDivElement>(null);
  const [fit, setFit] = useState(1);
  const [contentHeight, setContentHeight] = useState(height);

  useEffect(() => {
    const area = areaRef.current;
    if (!area) return;
    const update = () => {
      const available = Math.max(240, area.clientWidth - 56);
      setFit(Math.min(1, available / width));
    };
    update();
    const observer = new ResizeObserver(update);
    observer.observe(area);
    return () => observer.disconnect();
  }, [width]);

  useEffect(() => {
    const artboard = artboardRef.current;
    if (!artboard) return;
    const update = () => setContentHeight(Math.max(height, artboard.scrollHeight));
    update();
    const observer = new ResizeObserver(update);
    observer.observe(artboard);
    return () => observer.disconnect();
  }, [height, spec, width]);

  return (
    <div
      className={mobilePanel ? `wb-studio is-mobile-${mobilePanel}` : "wb-studio"}
      data-experience-surface="design-overview"
    >
      {mobilePanel ? (
        <button
          type="button"
          className="wb-studio-backdrop"
          aria-label={t("workbench.studio.closePanel", { defaultValue: "Close panel" })}
          onClick={() => setMobilePanel(null)}
        />
      ) : null}
      <aside className="wb-studio-left">
        <button
          type="button"
          className="wb-studio-sheet-close"
          aria-label={t("workbench.studio.closePanel", { defaultValue: "Close panel" })}
          onClick={() => setMobilePanel(null)}
        >
          <X aria-hidden size={15} />
        </button>
        <div className="wb-studio-section">
          <span className="wb-caption">
            {t("workbench.studio.scenario", { defaultValue: "Scenario" })}
          </span>
          {scenarios.isLoading ? <Skeleton height={34} /> : null}
          {scenarios.isError ? (
            <Banner tone="amber" icon={<CircleAlert aria-hidden size={14} />}>
              {t("workbench.studio.scenariosDown", { defaultValue: "Page inventory unavailable." })}
            </Banner>
          ) : null}
          <Select
            aria-label={t("workbench.studio.scenario", { defaultValue: "Scenario" })}
            value={scenario ?? ""}
            onChange={(event) => {
              setScenario(event.target.value);
              setPage(null);
              updateURL({ scenario: event.target.value, page: undefined });
            }}
          >
            {(scenarios.data ?? []).map((entry) => (
              <option key={entry.scenario} value={entry.scenario}>
                {entry.scenario} {entry.pageCount ? `(${entry.pageCount})` : ""}
              </option>
            ))}
          </Select>
        </div>

        <div className="wb-studio-section">
          <span className="wb-caption">
            {t("workbench.studio.pages", { defaultValue: "Pages" })}
          </span>
          {pages.isLoading ? <Skeleton height={80} /> : null}
          <div className="wb-page-list">
            {(pages.data ?? []).map((entry) => (
              <button
                key={entry.page}
                type="button"
                className="wb-page-item"
                aria-current={page === entry.page ? "true" : undefined}
                onClick={() => {
                  setPage(entry.page);
                  updateURL({ page: entry.page });
                }}
              >
                <ChevronRight aria-hidden size={12} />
                <span
                  style={{
                    minWidth: 0,
                    flex: 1,
                    overflow: "hidden",
                    textOverflow: "ellipsis",
                    whiteSpace: "nowrap",
                  }}
                >
                  {entry.title || entry.page}
                </span>
                <span className="wb-count">{entry.regionCount}</span>
              </button>
            ))}
            {(pages.data ?? []).length === 0 && !pages.isLoading ? (
              <p className="wb-muted-2 wb-small">
                {t("workbench.studio.noPages", {
                  defaultValue: "No declared pages for this scenario.",
                })}
              </p>
            ) : null}
          </div>
        </div>

        <div className="wb-studio-section" style={{ flex: 1 }}>
          <span className="wb-caption">
            {t("workbench.studio.layers", { defaultValue: "Layers" })}
          </span>
          <div className="wb-page-list">
            {rows.map((row) => (
              <button
                key={row.node.id}
                type="button"
                className="wb-layer"
                aria-current={selectedId === row.node.id ? "true" : undefined}
                onClick={() => {
                  setSelectedId(row.node.id);
                  // On a phone the tree and inspector are separate sheets.
                  // Selecting a layer is a request to edit it, so hand the
                  // person straight to its properties.
                  if (mobilePanel === "layers") setMobilePanel("inspector");
                }}
              >
                <span
                  className="wb-layer-depth"
                  style={{ "--depth": row.depth } as CSSProperties}
                />
                <span
                  style={{
                    minWidth: 0,
                    overflow: "hidden",
                    textOverflow: "ellipsis",
                    whiteSpace: "nowrap",
                  }}
                >
                  {layerLabel(row.node)}
                </span>
                <span className="wb-layer-type">{row.node.type}</span>
              </button>
            ))}
          </div>
        </div>

        <div className="wb-studio-section">
          <span className="wb-caption">
            {t("workbench.studio.template", { defaultValue: "Template" })}
          </span>
          <Select
            aria-label={t("workbench.studio.template", { defaultValue: "Template" })}
            value={template.id}
            onChange={(event) => selectTemplate(event.target.value)}
          >
            {shelf.map((entry) => (
              <option key={entry.id} value={entry.id}>
                {entry.name}
                {entry.local ? " · local" : ""} — {entry.tagline}
              </option>
            ))}
          </Select>
          <div className="wb-row" style={{ marginTop: 6, flexWrap: "wrap", gap: 4 }}>
            <button type="button" className="wb-btn quiet sm" onClick={duplicateTemplate}>
              <Copy aria-hidden size={12} />
              {t("workbench.studio.duplicate", { defaultValue: "Duplicate template" })}
            </button>
            <button type="button" className="wb-btn quiet sm" onClick={createTemplate}>
              <FilePlus2 aria-hidden size={12} />
              {t("workbench.studio.newTemplate", { defaultValue: "New template" })}
            </button>
            {template.local ? (
              <button type="button" className="wb-btn quiet sm" onClick={deleteTemplate}>
                <Trash2 aria-hidden size={12} />
                {t("workbench.studio.deleteTemplate", { defaultValue: "Delete template" })}
              </button>
            ) : null}
          </div>
          <p className="wb-muted-2 wb-small" style={{ marginTop: 6 }}>
            {t("workbench.studio.templateNote", {
              defaultValue:
                "Local templates stay in this browser until promoted through the governed asset flow.",
            })}
          </p>
        </div>
      </aside>

      <div className="wb-studio-center" data-experience-surface="design-workspace">
        <div className="wb-studio-top">
          <div className="wb-studio-title">
            <h1>{template.name}</h1>
            <small>
              {scenario ?? t("workbench.studio.noScenario", { defaultValue: "no scenario" })} ·{" "}
              {currentPage?.page ??
                t("workbench.studio.workingDraft", { defaultValue: "working draft" })}
            </small>
          </div>
          <div className="wb-row" style={{ flexWrap: "wrap" }}>
            <Segmented
              label={t("workbench.studio.device", { defaultValue: "Device" })}
              value={device}
              onChange={(next) => {
                setDevice(next);
                updateURL({ device: next });
              }}
              items={[
                { id: "desktop", label: <Monitor aria-hidden size={13} /> },
                { id: "tablet", label: <Tablet aria-hidden size={13} /> },
                { id: "phone", label: <Smartphone aria-hidden size={13} /> },
              ]}
            />
            <WbButton variant="ghost" size="sm" onClick={() => setPresenting(true)}>
              <Eye aria-hidden size={13} />
              {t("workbench.studio.present", { defaultValue: "Present" })}
            </WbButton>
          </div>
        </div>

        <div className="wb-studio-mobilebar">
          <button
            type="button"
            className="wb-btn ghost sm"
            onClick={() => setMobilePanel("layers")}
          >
            <Layers aria-hidden size={13} />
            {t("workbench.studio.layers", { defaultValue: "Layers" })}
          </button>
          <button
            type="button"
            className="wb-btn ghost sm"
            onClick={() => setMobilePanel("inspector")}
          >
            <SlidersHorizontal aria-hidden size={13} />
            {selected
              ? t("workbench.studio.inspect", { defaultValue: "Inspect" })
              : t("workbench.studio.openInspector", { defaultValue: "Inspect page" })}
          </button>
        </div>

        <div className="wb-brief">
          <Sparkles
            aria-hidden
            size={14}
            style={{ color: "var(--wb-signal)", flex: "none", marginTop: 2 }}
          />
          <span>
            <b>{t("workbench.studio.brief", { defaultValue: "Brief" })}:</b>{" "}
            {t("workbench.studio.briefBody", {
              defaultValue:
                "Make this page feel considered at a glance: one clear action, editorial hierarchy, and evidence the composition can actually hold.",
            })}
          </span>
        </div>

        {notice ? (
          <div style={{ padding: "8px 14px" }}>
            <Banner tone="ice" icon={<Check aria-hidden size={14} />}>
              {notice}
            </Banner>
          </div>
        ) : null}

        <div className="wb-artboard-area" ref={areaRef}>
          <div className="wb-artboard-bar">
            <span>{`${template.name} · ${device} · ${width}px · ${Math.round(fit * 100)}%`}</span>
            <span>
              {t("workbench.studio.revision", {
                defaultValue: "Revision {{hash}}",
                hash: hashSpec(spec),
              })}
            </span>
          </div>
          <div style={{ width: width * fit, height: contentHeight * fit, margin: "0 auto" }}>
            <div
              ref={artboardRef}
              className="wb-artboard"
              style={{
                width,
                minHeight: height,
                margin: 0,
                transform: `scale(${fit})`,
                transformOrigin: "top left",
                overflow: "visible",
              }}
            >
              <TemplateRenderer
                template={template}
                spec={spec}
                mode="canvas"
                selectedId={selectedId ?? undefined}
                onSelect={setSelectedId}
                slotEditable={Boolean(componentId)}
                slotPlacement={placement}
                onSlotPlacementChange={(nodeId, next) =>
                  setSpec(setSlotPlacement(spec, nodeId, next))
                }
                slot={
                  componentId ? (
                    <StudioSlot
                      componentId={componentId}
                      storyId={storyId}
                      template={template}
                      height={placement?.h ?? 56}
                    />
                  ) : undefined
                }
              />
            </div>
          </div>
          <div style={{ height: 24 }} />
        </div>

        <div className="wb-studio-foot">
          <span>
            {t("workbench.studio.localDraft", {
              defaultValue: pluralDefault(
                revisions.length,
                "Local draft · single user · {{count}} revision",
                "Local draft · single user · {{count}} revisions",
              ),
              count: revisions.length,
            })}
          </span>
          <span className="wb-row">
            <button type="button" className="wb-btn quiet sm" onClick={reset}>
              <RotateCcw aria-hidden size={12} />
              {t("workbench.studio.reset", { defaultValue: "Reset" })}
            </button>
            <Link to={appRoutes.review} className="wb-btn ghost sm">
              {t("workbench.studio.requestReview", { defaultValue: "Request review" })}
            </Link>
          </span>
        </div>
      </div>

      <aside className="wb-studio-right">
        <button
          type="button"
          className="wb-studio-sheet-close"
          aria-label={t("workbench.studio.closePanel", { defaultValue: "Close panel" })}
          onClick={() => setMobilePanel(null)}
        >
          <X aria-hidden size={15} />
        </button>
        <div className="wb-inspector-section">
          <span className="wb-caption">
            {t("workbench.studio.selection", { defaultValue: "Selection" })}
          </span>
          <div className="wb-row-between">
            <b className="wb-small">
              {selected
                ? layerLabel(selected)
                : t("workbench.studio.nothing", { defaultValue: "Nothing selected" })}
            </b>
            {selected ? <span className="wb-tag">{selected.type}</span> : null}
          </div>
          {selected ? (
            <div className="wb-stack tight" style={{ marginTop: 8 }}>
              {CONTAINERS.has(selected.type) ? (
                <div className="wb-row" style={{ flexWrap: "wrap", gap: 4 }}>
                  <Select
                    aria-label={t("workbench.studio.addLayer", { defaultValue: "Add a layer" })}
                    defaultValue=""
                    onChange={(event) => {
                      const entry = ADDABLE.find((item) => item.label === event.target.value);
                      if (entry) addNodeOfType(selected.id, entry);
                      event.target.value = "";
                    }}
                    style={{ width: "auto", minWidth: 130 }}
                  >
                    <option value="" disabled>
                      {t("workbench.studio.addLayer", { defaultValue: "Add a layer" })}
                    </option>
                    {ADDABLE.map((entry) => (
                      <option key={entry.label} value={entry.label}>
                        {entry.label}
                      </option>
                    ))}
                  </Select>
                  <span className="wb-tag">
                    <Plus aria-hidden size={11} /> {selected.type}
                  </span>
                </div>
              ) : (
                <p className="wb-muted-2 wb-small">
                  {t("workbench.studio.containerOnly", {
                    defaultValue: "Select a layout, card or the page to add children.",
                  })}
                </p>
              )}
              {selected.type === "slot" ? (
                <div className="wb-row" style={{ flexWrap: "wrap", gap: 4 }}>
                  <WbButton
                    variant="ghost"
                    size="sm"
                    disabled={!placement}
                    onClick={() => setSpec(clearSlotPlacement(spec, selected.id))}
                  >
                    <RotateCcw aria-hidden size={12} />
                    {t("workbench.studio.resetSlot", { defaultValue: "Reset slot position" })}
                  </WbButton>
                  <span className="wb-small wb-muted-2">
                    {placement
                      ? t("workbench.studio.slotPlacedAt", {
                          defaultValue: "x {{x}} · y {{y}} · {{w}} × {{h}}",
                          x: Math.round(placement.x),
                          y: Math.round(placement.y),
                          w: Math.round(placement.w),
                          h: Math.round(placement.h),
                        })
                      : t("workbench.studio.slotInFlowShort", { defaultValue: "Designed slot" })}
                  </span>
                </div>
              ) : null}
              {selected.id !== spec.id ? (
                <div className="wb-row" style={{ flexWrap: "wrap", gap: 4 }}>
                  <WbButton
                    variant="ghost"
                    size="sm"
                    aria-label={t("workbench.studio.moveUp", { defaultValue: "Move up" })}
                    onClick={() => setSpec(moveNode(spec, selected.id, -1))}
                  >
                    <ArrowUp aria-hidden size={12} />
                  </WbButton>
                  <WbButton
                    variant="ghost"
                    size="sm"
                    aria-label={t("workbench.studio.moveDown", { defaultValue: "Move down" })}
                    onClick={() => setSpec(moveNode(spec, selected.id, 1))}
                  >
                    <ArrowDown aria-hidden size={12} />
                  </WbButton>
                  <WbButton
                    variant="ghost"
                    size="sm"
                    aria-label={t("workbench.studio.duplicateLayer", {
                      defaultValue: "Duplicate layer",
                    })}
                    onClick={() => {
                      const next = duplicateNode(spec, selected.id);
                      setSpec(next);
                    }}
                  >
                    <Copy aria-hidden size={12} />
                  </WbButton>
                  <WbButton
                    variant="danger"
                    size="sm"
                    onClick={() => {
                      const next = removeNode(spec, selected.id);
                      setSpec(next);
                      setSelectedId(null);
                    }}
                  >
                    <Trash2 aria-hidden size={12} />
                    {t("workbench.studio.remove", { defaultValue: "Remove" })}
                  </WbButton>
                </div>
              ) : null}
            </div>
          ) : null}
        </div>

        <div className="wb-inspector-tabs">
          <button
            type="button"
            role="tab"
            aria-selected={inspectorTab === "props"}
            className="wb-tab"
            onClick={() => setInspectorTab("props")}
          >
            {t("workbench.studio.props", { defaultValue: "Props" })}
          </button>
          <button
            type="button"
            role="tab"
            aria-selected={inspectorTab === "theme"}
            className="wb-tab"
            onClick={() => setInspectorTab("theme")}
          >
            {t("workbench.studio.theme", { defaultValue: "Theme" })}
          </button>
          <button
            type="button"
            role="tab"
            aria-selected={inspectorTab === "contract"}
            className="wb-tab"
            onClick={() => setInspectorTab("contract")}
          >
            {t("workbench.studio.contract", { defaultValue: "Contract" })}
          </button>
          <button
            type="button"
            role="tab"
            aria-selected={inspectorTab === "compare"}
            className="wb-tab"
            onClick={() => setInspectorTab("compare")}
          >
            {t("workbench.studio.compare", { defaultValue: "Compare" })}
          </button>
        </div>

        <div className="wb-inspector-section">
          {inspectorTab === "props" ? (
            selected ? (
              <NodeInspector
                node={selected}
                onChange={(key, value) => setSpec(setProp(spec, selected.id, key, value))}
              />
            ) : (
              <Empty title={t("workbench.studio.selectLayer", { defaultValue: "Select a layer" })}>
                {t("workbench.studio.selectLayerBody", {
                  defaultValue: "Choose a layer in the tree or click an element in the artboard.",
                })}
              </Empty>
            )
          ) : null}

          {inspectorTab === "theme" ? (
            <ThemePanel template={template} onChange={updateTemplate} onSaveAs={saveAsTemplate} />
          ) : null}

          {inspectorTab === "contract" ? (
            <div>
              {checks.map((check) => (
                <div
                  key={check.id}
                  className={cx(
                    "wb-contract-row",
                    check.verdict === "pass" && "is-pass",
                    check.verdict === "warn" && "is-warn",
                    check.verdict === "fail" && "is-fail",
                  )}
                >
                  <span
                    aria-hidden
                    style={{
                      color:
                        check.verdict === "pass"
                          ? "var(--wb-green)"
                          : check.verdict === "fail"
                            ? "var(--wb-red)"
                            : check.verdict === "warn"
                              ? "var(--wb-amber)"
                              : "var(--wb-muted-2)",
                    }}
                  >
                    {check.verdict === "pass" ? <Check size={11} /> : <CircleAlert size={11} />}
                  </span>
                  <span>
                    {check.label}
                    <small className="wb-muted-2 wb-small" style={{ display: "block" }}>
                      {check.detail}
                    </small>
                  </span>
                  <b>{check.verdict.toUpperCase()}</b>
                </div>
              ))}
              <p className="wb-muted-2 wb-small" style={{ marginTop: 10 }}>
                {t("workbench.studio.contractNote", {
                  defaultValue:
                    "Contract checks are computed from this composition. Contrast and capture evidence stay unmeasured until a real capture exists.",
                })}
              </p>
            </div>
          ) : null}

          {inspectorTab === "compare" ? (
            <div className="wb-stack tight">
              <span className="wb-caption">
                {t("workbench.studio.baseline", { defaultValue: "Baseline vs draft" })}
              </span>
              <div className="wb-compare">
                <div>
                  <span className="wb-tag">
                    {t("workbench.studio.baseline", { defaultValue: "Baseline" })}
                  </span>
                  <div
                    style={{
                      marginTop: 6,
                      height: 120,
                      overflow: "hidden",
                      borderRadius: 6,
                      border: "1px solid var(--wb-line)",
                    }}
                  >
                    <TemplateRenderer template={template} spec={template.spec} mode="thumb" />
                  </div>
                </div>
                <div>
                  <span className="wb-tag">
                    {t("workbench.studio.draft", { defaultValue: "Draft" })}
                  </span>
                  <div
                    style={{
                      marginTop: 6,
                      height: 120,
                      overflow: "hidden",
                      borderRadius: 6,
                      border: "1px solid var(--wb-line)",
                    }}
                  >
                    <TemplateRenderer template={template} spec={spec} mode="thumb" />
                  </div>
                </div>
              </div>
              <div className="wb-divider" />
              <span className="wb-caption">
                {t("workbench.studio.history", { defaultValue: "Revisions" })}
              </span>
              {revisions.length === 0 ? (
                <p className="wb-muted-2 wb-small">
                  {t("workbench.studio.noHistory", { defaultValue: "No accepted revisions yet." })}
                </p>
              ) : (
                revisions.map((revision) => (
                  <button
                    key={revision.at}
                    type="button"
                    className="wb-row-between"
                    style={{ background: "none", border: 0, padding: "4px 0", cursor: "pointer" }}
                    onClick={() => setSpec(revision.spec)}
                  >
                    <span className="wb-mono wb-small">{revision.hash}</span>
                    <span className="wb-small wb-muted-2">
                      {formatRelative(new Date(revision.at))}
                    </span>
                  </button>
                ))
              )}
            </div>
          ) : null}
        </div>

        <div className="wb-inspector-section">
          <span className="wb-caption">
            {t("workbench.studio.actions", { defaultValue: "Actions" })}
          </span>
          <div className="wb-stack tight">
            <WbButton
              variant="primary"
              size="md"
              onClick={() =>
                save(
                  spec,
                  t("workbench.studio.accepted", {
                    defaultValue: "Composition accepted as a local revision.",
                  }),
                )
              }
            >
              <Check aria-hidden size={13} />
              {t("workbench.studio.accept", { defaultValue: "Accept composition" })}
            </WbButton>
            <Link to={appRoutes.review} className="wb-btn ghost sm wide">
              {t("workbench.studio.requestReview", { defaultValue: "Request review" })}
            </Link>
            <WbButton variant="ghost" size="sm" onClick={reset}>
              <RotateCcw aria-hidden size={12} />
              {t("workbench.studio.reset", { defaultValue: "Reset" })}
            </WbButton>
          </div>
        </div>
      </aside>

      {presenting ? (
        <div
          role="dialog"
          aria-modal="true"
          aria-label={t("workbench.studio.presentLabel", { defaultValue: "Present composition" })}
          style={{
            position: "fixed",
            inset: 0,
            zIndex: 90,
            display: "grid",
            placeItems: "center",
            padding: 24,
            background: "rgba(4,3,7,.92)",
          }}
        >
          <div style={{ position: "absolute", top: 16, right: 16 }}>
            <button
              type="button"
              className="wb-iconbtn"
              aria-label={t("workbench.studio.closePresent", {
                defaultValue: "Close presentation",
              })}
              onClick={() => setPresenting(false)}
            >
              <X aria-hidden size={16} />
            </button>
          </div>
          <div
            style={{
              width,
              maxHeight: "92dvh",
              overflow: "auto",
              borderRadius: 10,
              boxShadow: "0 40px 120px rgba(0,0,0,.6)",
            }}
          >
            <TemplateRenderer
              template={template}
              spec={spec}
              mode="preview"
              slot={
                componentId ? (
                  <StudioSlot
                    componentId={componentId}
                    storyId={storyId}
                    template={template}
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

function ThemePanel({
  template,
  onChange,
  onSaveAs,
}: {
  template: Template;
  onChange: (patch: Partial<Template>) => void;
  onSaveAs: () => void;
}) {
  const { t } = useTranslation();
  return (
    <div className="wb-stack tight">
      <Field label={t("workbench.studio.name", { defaultValue: "Name" })}>
        <TextInput
          value={template.name}
          onChange={(event) => onChange({ name: event.target.value })}
        />
      </Field>
      <Field label={t("workbench.studio.tagline", { defaultValue: "Tagline" })}>
        <TextInput
          value={template.tagline}
          onChange={(event) => onChange({ tagline: event.target.value })}
        />
      </Field>
      <Field label={t("workbench.studio.slotHint", { defaultValue: "Component slot hint" })}>
        <TextInput
          value={template.slotHint}
          onChange={(event) => onChange({ slotHint: event.target.value })}
        />
      </Field>
      <div className="wb-field">
        <span className="wb-label">
          {t("workbench.studio.palette", { defaultValue: "Palette" })}
        </span>
        <div className="wb-swatches">
          {TEMPLATE_PALETTES.map((entry) => {
            const selected = entry.palette.bg === template.palette.bg;
            return (
              <button
                key={entry.id}
                type="button"
                className={cx("wb-swatch", selected && "is-selected")}
                aria-pressed={selected}
                title={entry.label}
                onClick={() => onChange(withPalette(template, entry.palette))}
              >
                <span style={{ background: entry.palette.bg }} />
                <span style={{ background: entry.palette.accent }} />
                <span style={{ background: entry.palette.ink }} />
              </button>
            );
          })}
        </div>
      </div>
      <div className="wb-row" style={{ gap: 8 }}>
        <Field label={t("workbench.studio.accent", { defaultValue: "Accent" })}>
          <input
            type="color"
            className="wb-color"
            value={template.palette.accent}
            onChange={(event) =>
              onChange({ palette: { ...template.palette, accent: event.target.value } })
            }
            aria-label={t("workbench.studio.accent", { defaultValue: "Accent" })}
          />
        </Field>
        <Field label={t("workbench.studio.background", { defaultValue: "Background" })}>
          <input
            type="color"
            className="wb-color"
            value={template.palette.bg}
            onChange={(event) =>
              onChange({ palette: { ...template.palette, bg: event.target.value } })
            }
            aria-label={t("workbench.studio.background", { defaultValue: "Background" })}
          />
        </Field>
      </div>
      <div className="wb-field">
        <span className="wb-label">
          {t("workbench.studio.display", { defaultValue: "Display type" })}
        </span>
        <Segmented
          label={t("workbench.studio.display", { defaultValue: "Display type" })}
          value={template.display}
          onChange={(next) => onChange({ display: next })}
          items={[
            { id: "serif", label: t("workbench.studio.serif", { defaultValue: "Serif" }) },
            { id: "sans", label: t("workbench.studio.sans", { defaultValue: "Sans" }) },
          ]}
        />
      </div>
      <Field label={t("workbench.studio.radius", { defaultValue: "Corner radius" })}>
        <input
          type="range"
          min={0}
          max={28}
          value={template.radius}
          onChange={(event) => onChange({ radius: Number(event.target.value) })}
          aria-label={t("workbench.studio.radius", { defaultValue: "Corner radius" })}
        />
      </Field>
      <WbButton variant="ghost" size="sm" onClick={onSaveAs}>
        <Copy aria-hidden size={12} />
        {t("workbench.studio.saveAsTemplate", { defaultValue: "Save as new template" })}
      </WbButton>
      <p className="wb-muted-2 wb-small">
        {t("workbench.studio.themeNote", {
          defaultValue:
            "Theme edits are a browser-local draft until Accept composition. Duplicate the template to keep a named copy.",
        })}
      </p>
    </div>
  );
}

function NodeInspector({
  node: selectedNode,
  onChange,
}: {
  node: TemplateNode;
  onChange: (key: string, value: string | number | boolean) => void;
}) {
  const { t } = useTranslation();
  const props = selectedNode.props;
  const stringValue = (key: string): string => (typeof props[key] === "string" ? props[key] : "");
  const numberValue = (key: string, fallback = 0): number =>
    typeof props[key] === "number" ? props[key] : fallback;
  const has = (key: string): boolean => key in props;
  const isButton = selectedNode.type === "button";
  const textVariants = [
    "display",
    "title",
    "subtitle",
    "body",
    "label",
    "caption",
    "eyebrow",
    "mono",
    "price",
  ];

  return (
    <div className="wb-stack tight">
      <span className="wb-caption">{selectedNode.type}</span>

      {has("text") ? (
        <Field label={t("workbench.studio.text", { defaultValue: "Text" })}>
          <TextInput
            value={stringValue("text")}
            onChange={(event) => onChange("text", event.target.value)}
          />
        </Field>
      ) : null}

      {has("placeholder") ? (
        <Field label={t("workbench.studio.placeholder", { defaultValue: "Placeholder" })}>
          <TextInput
            value={stringValue("placeholder")}
            onChange={(event) => onChange("placeholder", event.target.value)}
          />
        </Field>
      ) : null}

      {has("name") ? (
        <Field label={t("workbench.studio.name", { defaultValue: "Name" })}>
          <TextInput
            value={stringValue("name")}
            onChange={(event) => onChange("name", event.target.value)}
          />
        </Field>
      ) : null}

      {has("label") ? (
        <Field label={t("workbench.studio.label", { defaultValue: "Label" })}>
          <TextInput
            value={stringValue("label")}
            onChange={(event) => onChange("label", event.target.value)}
          />
        </Field>
      ) : null}

      {has("hint") ? (
        <Field label={t("workbench.studio.hint", { defaultValue: "Slot hint" })}>
          <TextInput
            value={stringValue("hint")}
            onChange={(event) => onChange("hint", event.target.value)}
          />
        </Field>
      ) : null}

      {selectedNode.type === "stat" ? (
        <>
          <Field label={t("workbench.studio.value", { defaultValue: "Value" })}>
            <TextInput
              value={stringValue("value")}
              onChange={(event) => onChange("value", event.target.value)}
            />
          </Field>
          <Field label={t("workbench.studio.detail", { defaultValue: "Detail" })}>
            <TextInput
              value={stringValue("detail")}
              onChange={(event) => onChange("detail", event.target.value)}
            />
          </Field>
        </>
      ) : null}

      {selectedNode.type === "meter" ? (
        <Field label={t("workbench.studio.value", { defaultValue: "Value" })}>
          <TextInput
            type="number"
            min={0}
            max={100}
            value={numberValue("value")}
            onChange={(event) =>
              onChange("value", Math.max(0, Math.min(100, Number(event.target.value))))
            }
          />
        </Field>
      ) : null}

      {has("variant") ? (
        <Field label={t("workbench.studio.variant", { defaultValue: "Variant" })}>
          <Select
            value={stringValue("variant")}
            onChange={(event) => onChange("variant", event.target.value)}
          >
            {(isButton ? ["primary", "secondary", "ghost"] : textVariants).map((value) => (
              <option key={value} value={value}>
                {value}
              </option>
            ))}
          </Select>
        </Field>
      ) : null}

      {has("tone") ? (
        <Field label={t("workbench.studio.tone", { defaultValue: "Tone" })}>
          <Select
            value={stringValue("tone")}
            onChange={(event) => onChange("tone", event.target.value)}
          >
            {["ink", "muted", "accent", "secondary", "green", "amber", "red", "ice"].map(
              (value) => (
                <option key={value} value={value}>
                  {value}
                </option>
              ),
            )}
          </Select>
        </Field>
      ) : null}

      {has("shape") ? (
        <Field label={t("workbench.studio.shape", { defaultValue: "Art shape" })}>
          <Select
            value={stringValue("shape")}
            onChange={(event) => onChange("shape", event.target.value)}
          >
            {["none", "orb", "vase", "arch", "rings", "wave", "grid", "portrait"].map((value) => (
              <option key={value} value={value}>
                {value}
              </option>
            ))}
          </Select>
        </Field>
      ) : null}

      {has("align") ? (
        <Field label={t("workbench.studio.align", { defaultValue: "Align" })}>
          <Select
            value={stringValue("align")}
            onChange={(event) => onChange("align", event.target.value)}
          >
            {["left", "center", "right", "start", "end"].map((value) => (
              <option key={value} value={value}>
                {value}
              </option>
            ))}
          </Select>
        </Field>
      ) : null}

      {has("gap") ? (
        <Field label={t("workbench.studio.gap", { defaultValue: "Gap" })}>
          <TextInput
            type="number"
            value={numberValue("gap")}
            onChange={(event) => onChange("gap", Number(event.target.value))}
          />
        </Field>
      ) : null}

      {has("cols") ? (
        <Field label={t("workbench.studio.cols", { defaultValue: "Columns" })}>
          <TextInput
            type="number"
            min={1}
            max={6}
            value={numberValue("cols", 1)}
            onChange={(event) =>
              onChange("cols", Math.max(1, Math.min(6, Number(event.target.value))))
            }
          />
        </Field>
      ) : null}

      {has("pad") ? (
        <Field label={t("workbench.studio.pad", { defaultValue: "Padding" })}>
          <TextInput
            type="number"
            value={numberValue("pad")}
            onChange={(event) => onChange("pad", Number(event.target.value))}
          />
        </Field>
      ) : null}

      {has("h") ? (
        <Field label={t("workbench.studio.height", { defaultValue: "Height" })}>
          <TextInput
            type="number"
            value={numberValue("h")}
            onChange={(event) => onChange("h", Number(event.target.value))}
          />
        </Field>
      ) : null}

      {has("elevated") ? (
        <div className="wb-row-between">
          <span className="wb-label">
            {t("workbench.studio.elevated", { defaultValue: "Elevated" })}
          </span>
          <Toggle
            checked={Boolean(props.elevated)}
            onChange={(next) => onChange("elevated", next)}
            label={t("workbench.studio.elevated", { defaultValue: "Elevated" })}
          />
        </div>
      ) : null}

      <p className="wb-muted-2 wb-small">
        {t("workbench.studio.editNote", {
          defaultValue:
            "Edits update the local composition immediately. Accept creates a revision; it does not publish a library release.",
        })}
      </p>
    </div>
  );
}
