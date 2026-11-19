import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";
import { Contrast, Eye, Ruler, ScanLine, Target } from "lucide-react";

import { useTranslation } from "../../i18n";
import type { CatalogAsset, WorkbenchStory } from "../data";
import { Chip, Segmented, Select } from "../primitives";
import { TemplateRenderer } from "../templates/TemplateRenderer";
import { DEFAULT_TEMPLATE } from "../templates/library";
import { resolveTemplate, templateShelf, useLocalTemplates } from "../templates/local";
import type { Template } from "../templates/spec";
import {
  contrastRatioFromCss,
  effectiveBackground,
  isTouchTargetSmall,
  measureTouchTargets,
  overflowAt,
  parseCssColor,
  pickTextProbe,
  relativeLuminance,
} from "./instruments";
import {
  buildContextChecks,
  countPassed,
  scoredRows,
  type ContextCheckRow,
  type Translate,
} from "./contextChecks";
import { DeviceFrame } from "./DeviceFrame";
import { FixtureFrame, type FixtureStatus } from "./FixtureFrame";
import {
  neutralizeSlots,
  placeComponent,
  type PlacementDecision,
} from "./placement";
import {
  REFERENCE_WIDTHS,
  SAFE_BOTTOM,
  SAFE_TOP,
  VIEWPORTS,
  presetFor,
  type DeviceKind,
  type ViewportPreset,
} from "./viewports";
import "./incontext.css";

export type Device = DeviceKind;

export type InstrumentId = "guides" | "overflow" | "targets" | "contrast" | "blur";

export type Instruments = Record<InstrumentId, boolean>;

export interface InContextReport {
  rows: ContextCheckRow[];
  passed: number;
  scored: number;
  decision: PlacementDecision;
  device: Device;
  viewport: number;
  status: FixtureStatus;
  instruments: Instruments;
  storyId: string | null;
  /** True when the frame is showing a story rather than the empty state. */
  hasStory: boolean;
}

export interface InContextStageProps {
  asset: CatalogAsset;
  story?: WorkbenchStory;
  version: string;
  templateId: string;
  onTemplateChange: (id: string) => void;
  device: Device;
  onDeviceChange: (device: Device) => void;
  /** Manual slot choice; null means "let the scorer decide". */
  slotChoice: string | null;
  onSlotChoice: (slotId: string | null) => void;
  onReport: (report: InContextReport) => void;
  /** State coverage from the experience contract, when the caller has it. */
  states?: { total: number; covered: number; status: string } | null;
}

const DEFAULT_INSTRUMENTS: Instruments = {
  guides: true,
  overflow: true,
  targets: true,
  contrast: true,
  blur: false,
};

interface Mark {
  x: number;
  y: number;
  w: number;
  h: number;
  tone: "red" | "amber";
}

interface ContrastReading {
  nodeId: string;
  label: string;
  foreground: string;
  background: string;
  ratio: number | null;
  rect: { x: number; y: number; w: number; h: number } | null;
}

function themeFor(template: Template): {
  id: string;
  tokens: Record<string, string>;
  resolved: "light" | "dark";
} {
  const p = template.palette;
  const tokens: Record<string, string> = {
    "--color-background": p.bg,
    "--color-surface": p.surface,
    "--color-surface-muted": p.surfaceAlt,
    "--color-surface-raised": p.surface,
    "--color-surface-alt": p.surfaceAlt,
    "--color-foreground": p.ink,
    "--color-muted-foreground": p.muted,
    "--color-border": p.line,
    "--color-border-strong": p.muted,
    "--color-primary": p.accent,
    "--color-primary-foreground": p.accentInk,
    "--color-primary-hover": p.accent,
    "--color-focus": p.accent,
    "--color-field": p.surfaceAlt,
  };
  const bg = parseCssColor(p.bg);
  const resolved = bg && relativeLuminance(bg) > 0.45 ? "light" : "dark";
  return { id: `template:${template.id}`, tokens, resolved };
}

export function InContextStage({
  asset,
  story,
  version,
  templateId,
  onTemplateChange,
  device,
  onDeviceChange,
  slotChoice,
  onSlotChoice,
  onReport,
  states = null,
}: InContextStageProps) {
  const { t } = useTranslation();
  const locals = useLocalTemplates();
  const templates = templateShelf(locals);
  const template = resolveTemplate(templateId, locals) ?? DEFAULT_TEMPLATE;

  const [viewport, setViewport] = useState<ViewportPreset>(() => presetFor(device));
  const [instruments, setInstruments] = useState<Instruments>(DEFAULT_INSTRUMENTS);
  const [status, setStatus] = useState<FixtureStatus>("loading");
  const [reloadKey, setReloadKey] = useState(0);
  const [scale, setScale] = useState(1);
  const [overflow, setOverflow] = useState<
    Array<{ width: number; overflow: boolean; scrollWidth: number }>
  >([]);
  const [marks, setMarks] = useState<Mark[]>([]);
  const [targets, setTargets] = useState<{ total: number; small: number } | null>(null);
  const [typeInfo, setTypeInfo] = useState<{ inherited: boolean; fontFamily: string; fontSize: string } | null>(
    null,
  );
  const [pageType, setPageType] = useState<{
    fontFamily: string;
    fontSize: string;
    color: string;
  } | null>(null);
  const [safeArea, setSafeArea] = useState<{
    top: number;
    gap: number;
    height: number;
    safeTop: number;
    safeBottom: number;
  } | null>(null);
  const [contrastNodeId, setContrastNodeId] = useState<string | null>(null);
  const [contrast, setContrast] = useState<ContrastReading | null>(null);

  const canvasRef = useRef<HTMLDivElement>(null);
  const pageRef = useRef<HTMLDivElement>(null);
  const slotWrapRef = useRef<HTMLDivElement>(null);
  const probeRef = useRef<HTMLDivElement>(null);
  const lastDevice = useRef(device);
  const storyId = story?.id ?? null;

  // The frame is a physical choice; the viewport is what is tested. Switching
  // frames follows the device's natural viewport, but a person can then test
  // any width inside that frame.
  useEffect(() => {
    if (lastDevice.current === device) return;
    lastDevice.current = device;
    setViewport(presetFor(device));
  }, [device]);

  const chosen = useMemo(
    () =>
      placeComponent({
        spec: template.spec,
        asset,
        viewportWidth: viewport.width,
        preferredSlotId: slotChoice,
      }),
    [template.spec, asset, viewport.width, slotChoice],
  );

  const renderSpec = useMemo(
    () => neutralizeSlots(template.spec, chosen.chosen?.slotId),
    [template.spec, chosen.chosen?.slotId],
  );

  const theme = useMemo(() => themeFor(template), [template]);
  // A palette edit changes the tokens but not the spec. The contrast probe has
  // to re-read the page when the surface changes, so the tokens are a dep.
  const themeSignature = useMemo(
    () => `${theme.id}:${JSON.stringify(theme.tokens)}`,
    [theme],
  );

  const safeTop = SAFE_TOP[device];
  const safeBottom = SAFE_BOTTOM[device];
  const slotHeight = Math.max(chosen.chosen?.box.h ?? 40, chosen.profile.size.h);

  const toggle = (id: InstrumentId) =>
    setInstruments((current) => ({ ...current, [id]: !current[id] }));

  /* ── fit the frame into the canvas ─────────────────────────────────────── */
  useLayoutEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const fit = () => {
      const available = canvas.clientWidth - 44;
      const next = Math.max(0.28, Math.min(1, available / Math.max(280, viewport.width)));
      setScale((current) => (Math.abs(current - next) < 0.005 ? current : next));
    };
    fit();
    if (typeof ResizeObserver === "undefined") return;
    const observer = new ResizeObserver(fit);
    observer.observe(canvas);
    return () => observer.disconnect();
  }, [viewport.width]);

  /* ── page-only overflow at the three reference widths ──────────────────── */
  useLayoutEffect(() => {
    const root = probeRef.current;
    if (!root) return;
    const read = () => {
      const rows = Array.from(root.querySelectorAll<HTMLElement>("[data-ic-probe]")).map((frame) => {
        const width = Number(frame.dataset.icProbe);
        // Overflow is scrollWidth against the logical page width. The probe is
        // never scaled, and a screen rect would describe the probe, not the page.
        const scrollWidth = frame.scrollWidth;
        return { width, scrollWidth, overflow: overflowAt(width, scrollWidth) };
      });
      setOverflow((current) => {
        const same =
          current.length === rows.length &&
          current.every((row) => {
            const next = rows.find((entry) => entry.width === row.width);
            return next?.overflow === row.overflow && next.scrollWidth === row.scrollWidth;
          });
        return same ? current : rows;
      });
    };
    read();
    const timer = window.setTimeout(read, 120);
    return () => window.clearTimeout(timer);
  }, [renderSpec, template.id]);

  /* ── live measurements inside the frame ────────────────────────────────── */
  const measure = useCallback(() => {
    const host = pageRef.current;
    const root = host?.querySelector<HTMLElement>(".tpl");
    if (!host || !root) return;
    const hostRect = host.getBoundingClientRect();
    const rootRect = root.getBoundingClientRect();
    const factor = scale || 1;
    const toLocal = (value: number) => value / factor;

    const nextMarks: Mark[] = [];

    if (instruments.overflow) {
      const pageWidth = viewport.width;
      for (const element of Array.from(root.querySelectorAll<HTMLElement>("*"))) {
        const rect = element.getBoundingClientRect();
        if (rect.width === 0 || rect.height === 0) continue;
        const left = toLocal(rect.left - rootRect.left);
        const right = left + toLocal(rect.width);
        if (right > pageWidth + 1 || left < -1) {
          nextMarks.push({
            x: left,
            y: toLocal(rect.top - rootRect.top),
            w: toLocal(rect.width),
            h: toLocal(rect.height),
            tone: "red",
          });
        }
        if (nextMarks.length > 80) break;
      }
    }

    if (instruments.targets) {
      const elements = root.querySelectorAll<HTMLElement>(
        "button, a[href], input, select, textarea, [role='button'], [role='link'], [role='tab'], [tabindex='0']",
      );
      // The frame is zoomed, so the screen rect is not the target's size. The
      // scan divides by the frame's scale to report logical CSS pixels.
      const scan = measureTouchTargets(elements, factor);
      setTargets((current) => {
        if (!scan) return null;
        const next = { total: scan.total, small: scan.small };
        return current && current.total === next.total && current.small === next.small ? current : next;
      });
      for (const entry of scan?.targets ?? []) {
        if (!isTouchTargetSmall(entry.width, entry.height)) continue;
        nextMarks.push({
          x: toLocal(entry.rect.left - rootRect.left),
          y: toLocal(entry.rect.top - rootRect.top),
          w: entry.width,
          h: entry.height,
          tone: "amber",
        });
      }
    } else {
      setTargets(null);
    }

    setMarks((current) => {
      if (current.length !== nextMarks.length) return nextMarks;
      const same = current.every((mark, index) => {
        const next = nextMarks[index];
        return (
          next &&
          mark.tone === next.tone &&
          Math.round(mark.x) === Math.round(next.x) &&
          Math.round(mark.y) === Math.round(next.y) &&
          Math.round(mark.w) === Math.round(next.w) &&
          Math.round(mark.h) === Math.round(next.h)
        );
      });
      return same ? current : nextMarks;
    });

    // Type inheritance: the slot placeholder carries mono placeholder styles,
    // so the slot body is reset to the page's computed type before we compare.
    const rootStyle = getComputedStyle(root);
    setPageType((current) =>
      current &&
      current.fontFamily === rootStyle.fontFamily &&
      current.fontSize === rootStyle.fontSize &&
      current.color === rootStyle.color
        ? current
        : {
            fontFamily: rootStyle.fontFamily,
            fontSize: rootStyle.fontSize,
            color: rootStyle.color,
          },
    );

    const typeElement = slotWrapRef.current ?? host.querySelector<HTMLElement>(".tpl-slot");
    if (typeElement) {
      const slotStyle = getComputedStyle(typeElement);
      const family = slotStyle.fontFamily || "";
      const rootFamily = rootStyle.fontFamily || "";
      const inherited = family.length > 0 && family === rootFamily;
      const next = { inherited, fontFamily: family || "—", fontSize: slotStyle.fontSize || "—" };
      setTypeInfo((current) =>
        current &&
        current.inherited === next.inherited &&
        current.fontFamily === next.fontFamily &&
        current.fontSize === next.fontSize
          ? current
          : next,
      );

      // Safe area: the frame insets the page; the content must start after it.
      const page = root.querySelector<HTMLElement>(".tpl-page") ?? root;
      const first = page.firstElementChild as HTMLElement | null;
      const scroll = host.scrollTop || 0;
      const top = first ? toLocal(first.getBoundingClientRect().top - hostRect.top) + scroll : 0;
      const gap = Number.parseFloat(getComputedStyle(host).paddingBottom) || 0;
      const nextSafe = { top, gap, height: viewport.height, safeTop, safeBottom };
      setSafeArea((current) =>
        current &&
        Math.abs(current.top - nextSafe.top) < 0.5 &&
        Math.abs(current.gap - nextSafe.gap) < 0.5 &&
        current.height === nextSafe.height &&
        current.safeTop === nextSafe.safeTop
          ? current
          : nextSafe,
      );
    }
  }, [
    instruments.overflow,
    instruments.targets,
    safeBottom,
    safeTop,
    scale,
    viewport.height,
    viewport.width,
  ]);

  useLayoutEffect(() => {
    if (typeof window === "undefined") return;
    const raf = window.requestAnimationFrame(measure);
    return () => window.cancelAnimationFrame(raf);
  }, [measure, pageType, renderSpec, status, storyId, contrastNodeId]);

  useEffect(() => {
    const host = pageRef.current;
    if (!host || typeof ResizeObserver === "undefined") return;
    const observer = new ResizeObserver(() => measure());
    observer.observe(host);
    const root = host.querySelector<HTMLElement>(".tpl");
    if (root) observer.observe(root);
    return () => observer.disconnect();
  }, [measure, renderSpec]);

  /* ── contrast probe ─────────────────────────────────────────────────────── */
  const readContrast = useCallback(
    (node: HTMLElement | null) => {
      const host = pageRef.current;
      const root = host?.querySelector<HTMLElement>(".tpl");
      if (!host || !root || !node) {
        setContrast(null);
        return;
      }
      const style = getComputedStyle(node);
      const background =
        effectiveBackground((element) => getComputedStyle(element).backgroundColor, node) ?? "";
      const ratio = contrastRatioFromCss(style.color, background);
      const rect = node.getBoundingClientRect();
      const rootRect = root.getBoundingClientRect();
      const factor = scale || 1;
      const label = node.textContent.trim().replace(/\s+/g, " ").slice(0, 32);
      setContrast({
        nodeId: node.dataset.nodeId ?? "",
        label: label || node.dataset.nodeId || "text",
        foreground: style.color,
        background,
        ratio,
        rect: {
          x: (rect.left - rootRect.left) / factor,
          y: (rect.top - rootRect.top) / factor,
          w: rect.width / factor,
          h: rect.height / factor,
        },
      });
    },
    [scale],
  );

  useEffect(() => {
    if (!instruments.contrast) {
      setContrast(null);
      return;
    }
    const host = pageRef.current;
    if (!host) return;
    const explicit = contrastNodeId
      ? host.querySelector<HTMLElement>(`[data-node-id="${contrastNodeId}"]`)
      : null;
    // Only a node with text can produce a meaningful ratio. The old fallback
    // took the first `.tpl-text` even when it was empty.
    const fallback = explicit ?? pickTextProbe(host);
    if (!explicit && fallback?.dataset.nodeId) setContrastNodeId(fallback.dataset.nodeId);
    readContrast(fallback);
  }, [
    contrastNodeId,
    instruments.contrast,
    readContrast,
    renderSpec,
    status,
    storyId,
    template.id,
    themeSignature,
  ]);

  // A click on the page picks the text node under the pointer for the
  // contrast probe. A native listener keeps the page a document, not a
  // focusable control: the instruments are the interactive surface.
  useEffect(() => {
    const host = pageRef.current;
    if (!host || !instruments.contrast) return;
    const onPick = (event: Event) => {
      const target = event.target as HTMLElement | null;
      const node = target?.closest<HTMLElement>("[data-node-id]");
      if (!node) return;
      setContrastNodeId(node.dataset.nodeId ?? null);
      readContrast(node);
    };
    host.addEventListener("click", onPick);
    return () => host.removeEventListener("click", onPick);
  }, [instruments.contrast, readContrast, renderSpec]);

  /* ── checks + report ────────────────────────────────────────────────────── */
  const translate = useCallback<Translate>(
    (key, options) => t(key as never, options as never) as unknown as string,
    [t],
  );

  const rows = useMemo(() => {
    return buildContextChecks(
      {
        slot: chosen.chosen
          ? {
              label: chosen.chosen.label,
              role: chosen.chosen.role,
              accepts: chosen.chosen.accepts,
              compatible: chosen.chosen.compatible,
              box: chosen.chosen.box,
            }
          : undefined,
        component: { kind: chosen.profile.kind, size: chosen.profile.size },
        overflow,
        type: typeInfo,
        touch: instruments.targets && targets ? { on: true, ...targets } : { on: false, small: 0, total: 0 },
        contrast: {
          on: instruments.contrast,
          ratio: contrast?.ratio ?? null,
          label: contrast?.label ?? "",
          foreground: contrast?.foreground ?? "",
          background: contrast?.background ?? "",
        },
        states,
        safeArea:
          instruments.guides && safeArea
            ? { on: true, ...safeArea }
            : { on: false, top: 0, gap: 0, height: viewport.height, safeTop, safeBottom },
        squint: { on: instruments.blur },
      },
      translate,
    );
  }, [
    chosen,
    contrast,
    instruments.blur,
    instruments.contrast,
    instruments.guides,
    instruments.targets,
    overflow,
    safeArea,
    safeBottom,
    safeTop,
    states,
    targets,
    typeInfo,
    translate,
    viewport.height,
  ]);

  const passed = countPassed(rows);
  const scored = scoredRows(rows).length;

  useEffect(() => {
    onReport({
      rows,
      passed,
      scored,
      decision: chosen,
      device,
      viewport: viewport.width,
      status,
      instruments,
      storyId,
      hasStory: Boolean(story),
    });
  }, [chosen, device, instruments, onReport, passed, rows, scored, status, story, storyId, viewport.width]);

  /* ── placement copy ─────────────────────────────────────────────────────── */
  const best = chosen.auto;
  const why = chosen.manual && chosen.chosen
    ? chosen.chosen.slotId === best?.slotId
      ? t("workbench.component.ctxWhyPinned", {
          defaultValue: "{{slot}} is pinned by hand; it is also the best score at {{score}}%.",
          slot: chosen.chosen.label,
          score: chosen.chosen.score,
        })
      : t("workbench.component.ctxWhyManual", {
          defaultValue:
            "{{slot}} was chosen by hand; {{auto}} still scores highest at {{score}}%.",
          slot: chosen.chosen.label,
          auto: best?.label ?? "—",
          score: best?.score ?? 0,
        })
    : chosen.chosen
      ? t("workbench.component.ctxWhy", {
          defaultValue:
            "{{slot}} wins for a {{kind}}: role {{role}}, {{fit}} of {{width}} px available, slot {{index}} on the page.",
          slot: chosen.chosen.label,
          kind: chosen.profile.kind,
          role:
            chosen.chosen.role ??
            t("workbench.component.ctxRoleOpen", { defaultValue: "open" }),
          fit: `${chosen.profile.size.w} px`,
          width: chosen.chosen.box.w,
          index: chosen.chosen.index + 1,
        })
      : t("workbench.component.ctxWhyEmpty", {
          defaultValue: "This page declares no slots, so the component cannot be placed.",
        });

  return (
    <section
      className="ic"
      aria-label={t("workbench.component.inContext", { defaultValue: "In context" })}
      data-testid="in-context-stage"
    >
      <div className="ic-bar">
        <Segmented
          label={t("workbench.component.device", { defaultValue: "Frame" })}
          value={device}
          onChange={onDeviceChange}
          items={[
            { id: "desktop", label: t("workbench.component.desktop", { defaultValue: "Desktop" }) },
            { id: "tablet", label: t("workbench.component.tablet", { defaultValue: "Tablet" }) },
            { id: "phone", label: t("workbench.component.phone", { defaultValue: "Phone" }) },
          ]}
        />
        <Select
          aria-label={t("workbench.component.viewport", { defaultValue: "Viewport width" })}
          className="ic-viewport-select"
          value={viewport.id}
          onChange={(event) => {
            const next = VIEWPORTS.find((entry) => entry.id === event.target.value);
            if (next) setViewport(next);
          }}
        >
          {VIEWPORTS.map((entry) => (
            <option key={entry.id} value={entry.id}>
              {t("workbench.component.viewportValue", {
                defaultValue: "{{width}} px",
                width: entry.label,
              })}
            </option>
          ))}
        </Select>
        <div className="ic-tools" role="group" aria-label={t("workbench.component.instruments", { defaultValue: "Instruments" })}>
          <ToolButton
            id="guides"
            active={instruments.guides}
            icon={<Ruler aria-hidden size={12} />}
            label={t("workbench.component.instrumentGuides", { defaultValue: "Guides" })}
            onClick={toggle}
          />
          <ToolButton
            id="overflow"
            active={instruments.overflow}
            icon={<ScanLine aria-hidden size={12} />}
            label={t("workbench.component.instrumentOverflow", { defaultValue: "Overflow" })}
            onClick={toggle}
          />
          <ToolButton
            id="targets"
            active={instruments.targets}
            icon={<Target aria-hidden size={12} />}
            label={t("workbench.component.instrumentTargets", { defaultValue: "Targets" })}
            onClick={toggle}
          />
          <ToolButton
            id="contrast"
            active={instruments.contrast}
            icon={<Contrast aria-hidden size={12} />}
            label={t("workbench.component.instrumentContrast", { defaultValue: "Contrast" })}
            onClick={toggle}
          />
          <ToolButton
            id="blur"
            active={instruments.blur}
            icon={<Eye aria-hidden size={12} />}
            label={t("workbench.component.instrumentBlur", { defaultValue: "Blur" })}
            onClick={toggle}
          />
        </div>
        <div className="ic-bar-spacer" />
        <Select
          aria-label={t("workbench.component.template", { defaultValue: "Page template" })}
          className="ic-template-select"
          value={template.id}
          onChange={(event) => onTemplateChange(event.target.value)}
        >
          {templates.map((entry) => (
            <option key={entry.id} value={entry.id}>
              {entry.name} — {entry.tagline}
            </option>
          ))}
        </Select>
        <Chip tone={status === "error" ? "red" : status === "loading" ? "amber" : "green"}>
          {status === "loading"
            ? t("workbench.component.harnessConnecting", { defaultValue: "Connecting…" })
            : t("workbench.component.liveHarness", { defaultValue: "Live harness" })}
        </Chip>
        <button
          type="button"
          className="ic-reload"
          aria-label={t("workbench.component.reloadFixture", { defaultValue: "Reload specimen" })}
          onClick={() => setReloadKey((value) => value + 1)}
        >
          ↻
        </button>
      </div>

      <p
        className="rcl-visually-hidden"
        role="status"
        aria-live="polite"
        data-testid="in-context-announce"
      >
        {chosen.chosen
          ? t("workbench.component.ctxAnnounce", {
              defaultValue: "Placed in {{slot}}, {{score}}% affinity",
              slot: chosen.chosen.label,
              score: chosen.chosen.score,
            })
          : t("workbench.component.ctxAnnounceNone", {
              defaultValue: "No slot to place this component. Choose a page template with a component slot.",
            })}
      </p>

      <div className="ic-body">
        <div
          className="ic-side"
          role="group"
          aria-label={t("workbench.component.placement", { defaultValue: "Placement" })}
        >
          <span className="ic-eyebrow">{t("workbench.component.placement", { defaultValue: "Placement" })}</span>
          {chosen.chosen ? (
            <>
              <div className="ic-chosen">
                <div className="ic-chosen-head">
                  <b>{chosen.chosen.label}</b>
                  <Chip tone={chosen.chosen.compatible ? "green" : "amber"}>{chosen.chosen.score}%</Chip>
                </div>
                <p className="ic-why">{why}</p>
                <p className="ic-meta">
                  {t("workbench.component.ctxSlotMeta", {
                    defaultValue: "role {{role}} · {{width}} × {{height}} · accepts {{accepts}}",
                    role: chosen.chosen.role ?? t("workbench.component.ctxRoleOpen", { defaultValue: "open" }),
                    width: chosen.chosen.box.w,
                    height: Math.round(slotHeight),
                    accepts: chosen.chosen.accepts.length
                      ? chosen.chosen.accepts.join(", ")
                      : t("workbench.component.ctxAcceptsAny", { defaultValue: "any kind" }),
                  })}
                </p>
                <p className="ic-flow">
                  {t("workbench.component.ctxFlow", {
                    defaultValue:
                      "In flow: the page lays the specimen out inside {{slot}}, so the slot sets its width. Pin another slot below to compare.",
                    slot: chosen.chosen.label,
                  })}
                </p>
              </div>

              {!chosen.chosen.compatible ? (
                <p className="ic-note is-warn" role="status">
                  {t("workbench.component.ctxMismatch", {
                    defaultValue:
                      "No slot on this page declares {{kind}}. It is placed unproven — pick another slot or a different page template.",
                    kind: chosen.profile.kind,
                  })}
                </p>
              ) : null}

              <div className="ic-slots" role="group" aria-label={t("workbench.component.slotPicker", { defaultValue: "Slot picker" })}>
                {chosen.candidates.map((candidate) => (
                  <button
                    key={candidate.slotId}
                    type="button"
                    className="ic-slot-row"
                    aria-pressed={candidate.slotId === chosen.chosen?.slotId}
                    onClick={() => onSlotChoice(candidate.slotId)}
                  >
                    <span className={`ic-slot-mark${candidate.compatible ? "" : " is-mismatch"}`} aria-hidden />
                    <span className="ic-slot-label">
                      <span className="ic-slot-name">{candidate.label}</span>
                      {candidate.slotId === chosen.auto?.slotId ? (
                        <span className="ic-slot-auto">
                          {t("workbench.component.ctxAuto", { defaultValue: "auto" })}
                        </span>
                      ) : null}
                      <span className="ic-slot-kind">
                        {candidate.role ??
                          t("workbench.component.ctxRoleOpen", { defaultValue: "open" })}
                      </span>
                    </span>
                    <span className="ic-slot-score">{candidate.score}</span>
                    <span className="ic-affinity" aria-hidden>
                      <i style={{ width: `${candidate.score}%` }} />
                    </span>
                  </button>
                ))}
              </div>

              {chosen.manual ? (
                <button type="button" className="ic-reset" onClick={() => onSlotChoice(null)}>
                  {t("workbench.component.resetPlacement", { defaultValue: "Reset to authored slot" })}
                </button>
              ) : null}
            </>
          ) : (
            <div className="ic-note is-warn" role="status">
              <b>{t("workbench.component.ctxNoSlotTitle", { defaultValue: "No slot on this page" })}</b>
              <p>
                {t("workbench.component.ctxNoSlot", {
                  defaultValue:
                    "This page has no component slot to place into. Choose a page template that declares one, or add a slot in the studio.",
                })}
              </p>
            </div>
          )}
        </div>

        <div className="ic-canvas" ref={canvasRef}>
          <DeviceFrame
            device={device}
            width={viewport.width}
            height={viewport.height}
            scale={scale}
            label={t("workbench.component.ctxFrameLabel", {
              defaultValue: "{{template}} at {{width}} pixels in a {{device}} frame",
              template: template.name,
              width: viewport.width,
              device,
            })}
            url={`app.vrooli.local/${template.id}`}
            guides={instruments.guides}
            statusBar={device !== "desktop"}
          >
            <div
              className={`ic-page${instruments.blur ? " is-blurred" : ""}`}
              ref={pageRef}
              style={{
                paddingTop: safeTop,
                paddingBottom: safeBottom,
                background: template.palette.bg,
              }}
              data-testid="in-context-page"
            >
              <div className="ic-page-inner">
                <TemplateRenderer
                  template={template}
                  spec={renderSpec}
                  mode="preview"
                  slot={
                    story ? (
                      <div
                        className="ic-slot-body"
                        ref={slotWrapRef}
                        style={
                          pageType
                            ? {
                                fontFamily: pageType.fontFamily,
                                fontSize: pageType.fontSize,
                                color: pageType.color,
                              }
                            : undefined
                        }
                      >
                        <FixtureFrame
                          assetId={asset.id}
                          storyId={story.id}
                          version={version}
                          reloadKey={reloadKey}
                          height={slotHeight}
                          theme={theme}
                          title={t("workbench.component.contextTitle", {
                            defaultValue: "{{name}} in {{template}}",
                            name: story.name || story.id,
                            template: template.name,
                          })}
                          onStatus={setStatus}
                          onRetry={() => setReloadKey((value) => value + 1)}
                        />
                      </div>
                    ) : (
                      <div
                        className="ic-empty-slot"
                        ref={slotWrapRef}
                        style={
                          pageType
                            ? {
                                fontFamily: pageType.fontFamily,
                                fontSize: pageType.fontSize,
                                color: pageType.color,
                              }
                            : undefined
                        }
                      >
                        {t("workbench.component.ctxNoStory", {
                          defaultValue:
                            "No story selected. Pick a story in the toolbar above to render this component here.",
                        })}
                      </div>
                    )
                  }
                />

                {marks.length > 0 ? (
                  <div className="ic-marks" aria-hidden>
                    {marks.map((mark, index) => (
                      <span
                        key={`${mark.tone}-${index}-${Math.round(mark.x)}-${Math.round(mark.y)}`}
                        className={`ic-mark is-${mark.tone}`}
                        style={{
                          left: Math.round(mark.x),
                          top: Math.round(mark.y),
                          width: Math.max(1, Math.round(mark.w)),
                          height: Math.max(1, Math.round(mark.h)),
                        }}
                      />
                    ))}
                  </div>
                ) : null}

                {instruments.contrast && contrast?.rect && contrast.ratio !== null ? (
                  <div
                    className="ic-contrast"
                    style={{
                      left: Math.round(contrast.rect.x),
                      top: Math.round(contrast.rect.y),
                      width: Math.round(contrast.rect.w),
                      height: Math.round(contrast.rect.h),
                    }}
                  />
                ) : null}
              </div>
            </div>
          </DeviceFrame>

          <p className="ic-caption">
            {chosen.chosen
              ? t("workbench.component.ctxCaption", {
                  defaultValue:
                    "{{template}} · placed in {{slot}} · {{width}} × {{height}} at {{scale}}% scale",
                  template: template.name,
                  slot: chosen.chosen.label,
                  width: viewport.width,
                  height: viewport.height,
                  scale: Math.round(scale * 100),
                })
              : t("workbench.component.ctxCaptionEmpty", {
                  defaultValue: "{{template}} · no slot declared",
                  template: template.name,
                })}
          </p>

          {status === "error" ? (
            <p className="ic-caption is-error" role="status">
              <span>
                {t("workbench.component.ctxHarnessFailed", {
                  defaultValue: "The specimen could not load. Try another story, or reload it.",
                })}
              </span>{" "}
              <button
                type="button"
                className="ic-link"
                onClick={() => setReloadKey((value) => value + 1)}
              >
                {t("workbench.component.reloadFixture", { defaultValue: "Reload specimen" })}
              </button>
            </p>
          ) : null}

          {/* Page-only probes at the three reference widths. They render the
              composition without the harness iframe, which cannot change the
              page's width because the frame is width-constrained. `inert` keeps
              their controls out of the tab order and the accessibility tree. */}
          <div className="ic-probe" ref={probeRef} {...{ inert: "" }}>
            {REFERENCE_WIDTHS.map((width) => (
              <div key={width} className="ic-probe-frame" style={{ width }} data-ic-probe={width}>
                <TemplateRenderer template={template} spec={renderSpec} mode="preview" />
              </div>
            ))}
          </div>
        </div>
      </div>
    </section>
  );
}

function ToolButton({
  id,
  active,
  icon,
  label,
  onClick,
}: {
  id: InstrumentId;
  active: boolean;
  icon: ReactNode;
  label: string;
  onClick: (id: InstrumentId) => void;
}) {
  return (
    <button
      type="button"
      className={`ic-tool${active ? " is-on" : ""}`}
      aria-pressed={active}
      onClick={() => onClick(id)}
    >
      {icon}
      {label}
    </button>
  );
}
