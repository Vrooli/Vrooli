import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";

import { useTranslation } from "../../i18n";
import type { CatalogAsset, WorkbenchStory } from "../data";
import type { Template } from "../templates/spec";
import { parseCssColor, relativeLuminance } from "./instruments";
import { buildContextChecks, type ContextCheckRow, type Translate } from "./contextChecks";
import { InContextCanvas } from "./InContextCanvas";
import { InContextPlacementPanel } from "./InContextPlacementPanel";
import { InContextToolbar } from "./InContextToolbar";
import type { FixtureStatus } from "./FixtureFrame";
import type { FixturePaint } from "./fixtureBench";
import { componentProbeContrast, type ComponentProbe } from "./componentProbe";
import { placeComponent, type PlacementDecision } from "./placement";
import { useInContextMeasurements } from "./useInContextMeasurements";
import {
  SAFE_BOTTOM,
  SAFE_TOP,
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
  decision: PlacementDecision;
  device: Device;
  viewport: number;
}

export interface InContextStageProps {
  asset: CatalogAsset;
  story?: WorkbenchStory;
  version: string;
  template: Template;
  templates: Array<{ id: string; name: string; summary: string }>;
  onTemplateChange: (id: string) => void;
  device: Device;
  onDeviceChange: (device: Device) => void;
  /** Manual slot choice; null means "let the scorer decide". */
  slotChoice: string | null;
  onSlotChoice: (slotId: string | null) => void;
  onReport: (report: InContextReport) => void;
  /** State coverage from the experience contract, when the caller has it. */
  states?: {
    total: number;
    covered: number;
    status: string;
    captures?: number;
    latest?: string;
    message?: string;
  } | null;
}

const DEFAULT_INSTRUMENTS: Instruments = {
  guides: true,
  overflow: true,
  targets: true,
  contrast: true,
  blur: false,
};

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
  template,
  templates,
  onTemplateChange,
  device,
  onDeviceChange,
  slotChoice,
  onSlotChoice,
  onReport,
  states = null,
}: InContextStageProps) {
  const { t } = useTranslation();

  const [viewport, setViewport] = useState<ViewportPreset>(() => presetFor(device));
  const [instruments, setInstruments] = useState<Instruments>(DEFAULT_INSTRUMENTS);
  const [status, setStatus] = useState<FixtureStatus>("loading");
  const [reloadKey, setReloadKey] = useState(0);
  const [scale, setScale] = useState(1);
  const [paint, setPaint] = useState<FixturePaint | null>(null);
  const [probe, setProbe] = useState<ComponentProbe | null>(null);

  const canvasRef = useRef<HTMLDivElement>(null);
  const pageRef = useRef<HTMLDivElement>(null);
  const slotWrapRef = useRef<HTMLDivElement>(null);
  const probeRef = useRef<HTMLDivElement>(null);
  const lastDevice = useRef(device);
  const storyId = story?.id ?? null;

  useEffect(() => {
    setPaint(null);
    setProbe(null);
  }, [storyId, template.id]);

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

  const renderSpec = template.spec;

  const theme = useMemo(() => themeFor(template), [template]);
  // A palette edit changes the tokens but not the spec. The contrast probe has
  // to re-read the page when the surface changes, so the tokens are a dep.
  const themeSignature = useMemo(() => `${theme.id}:${JSON.stringify(theme.tokens)}`, [theme]);

  const measurements = useInContextMeasurements({
    pageRef,
    slotWrapRef,
    probeRef,
    scale,
    viewport,
    safeTop: SAFE_TOP[device],
    safeBottom: SAFE_BOTTOM[device],
    overflowEnabled: instruments.overflow,
    targetsEnabled: instruments.targets,
    contrastEnabled: instruments.contrast,
    renderSpec,
    templateId: template.id,
    themeSignature,
    status,
    storyId,
  });
  const { overflow, marks, targets, typeInfo, pageType, safeArea, contrast } = measurements;

  // A real page can be taller than the device viewport. Bring the chosen slot
  // into view instead of leaving the operator looking at the page header while
  // the component sits below the fold.
  useEffect(() => {
    const page = pageRef.current;
    const slot = slotWrapRef.current;
    if (
      !page ||
      !slot ||
      typeof window.requestAnimationFrame !== "function" ||
      typeof page.scrollTo !== "function"
    )
      return;
    const frame = window.requestAnimationFrame(() => {
      const pageRect = page.getBoundingClientRect();
      const slotRect = slot.getBoundingClientRect();
      const pageScale = page.offsetWidth > 0 ? pageRect.width / page.offsetWidth : 1;
      const target =
        page.scrollTop +
        (slotRect.top - pageRect.top) / pageScale -
        page.clientHeight / 2 +
        slotRect.height / pageScale / 2;
      page.scrollTo({ top: Math.max(0, target), behavior: "smooth" });
    });
    return () => window.cancelAnimationFrame(frame);
  }, [chosen.chosen?.slotId, storyId, paint?.height]);

  const safeTop = SAFE_TOP[device];
  const safeBottom = SAFE_BOTTOM[device];
  const slotHeight = Math.max(
    chosen.chosen?.box.h ?? 40,
    chosen.profile.size.h,
    paint?.height ?? 0,
  );

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

  /* ── checks + report ────────────────────────────────────────────────────── */
  const translate = useCallback<Translate>(
    (key, options) => t(key as never, options as never) as unknown as string,
    [t],
  );

  const componentContrast = useMemo(() => {
    if (!instruments.contrast) {
      return { on: false, ratio: null, label: "", background: "" };
    }
    if (probe?.measured) {
      return componentProbeContrast(probe);
    }
    return {
      on: true,
      ratio: contrast?.ratio ?? null,
      label: contrast?.label ?? "",
      background: contrast?.background ?? "",
    };
  }, [contrast, instruments.contrast, probe]);

  const componentType = useMemo(() => {
    if (probe?.measured) {
      const type = probe.typography;
      if (!type) return { fontFamily: "—", fontSize: "—" };
      return { fontFamily: type.fontFamily, fontSize: type.fontSize };
    }
    return typeInfo ? { fontFamily: typeInfo.fontFamily, fontSize: typeInfo.fontSize } : null;
  }, [probe, typeInfo]);

  const rows = useMemo(() => {
    return buildContextChecks(
      {
        slot: chosen.chosen
          ? {
              label: chosen.chosen.label,
              role: chosen.chosen.role,
              accepts: chosen.chosen.accepts,
              compatible: chosen.chosen.compatible,
              roleMatch: chosen.chosen.roleMatch,
              box: chosen.chosen.box,
            }
          : undefined,
        // The declared profile is the placement contract; the frame is what
        // actually rendered. Prefer the measured story so the row changes when
        // a story's geometry changes.
        component: {
          kind: chosen.profile.kind,
          size: paint
            ? { w: Math.round(paint.width), h: Math.round(paint.height) }
            : chosen.profile.size,
        },
        overflow,
        type: componentType,
        touch:
          instruments.targets && probe?.measured
            ? { on: true, ...probe.targets }
            : instruments.targets && targets
              ? { on: true, ...targets }
              : { on: false, small: 0, total: 0 },
        contrast: componentContrast,
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
    componentContrast,
    instruments.blur,
    instruments.guides,
    instruments.targets,
    overflow,
    safeArea,
    safeBottom,
    safeTop,
    componentType,
    paint,
    probe,
    states,
    targets,
    translate,
    viewport.height,
  ]);

  useEffect(() => {
    onReport({
      rows,
      decision: chosen,
      device,
      viewport: viewport.width,
    });
  }, [chosen, device, onReport, rows, viewport.width]);

  return (
    <section
      className="ic"
      aria-label={t("workbench.component.inContext", { defaultValue: "In context" })}
      data-testid="in-context-stage"
    >
      <InContextToolbar
        device={device}
        onDeviceChange={onDeviceChange}
        viewport={viewport}
        onViewportChange={setViewport}
        instruments={instruments}
        onInstrumentToggle={toggle}
        template={template}
        templates={templates}
        onTemplateChange={onTemplateChange}
        status={status}
        onReload={() => setReloadKey((value) => value + 1)}
      />

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
              defaultValue:
                "No slot to place this component. Choose a page template with a component slot.",
            })}
      </p>

      <div className="ic-body">
        <InContextPlacementPanel
          decision={chosen}
          slotHeight={slotHeight}
          componentSize={
            paint ? { w: Math.round(paint.width), h: Math.round(paint.height) } : undefined
          }
          onSlotChoice={onSlotChoice}
        />

        <InContextCanvas
          asset={asset}
          story={story}
          version={version}
          template={template}
          renderSpec={renderSpec}
          activeSlotId={chosen.chosen?.slotId}
          decision={chosen}
          device={device}
          viewport={viewport}
          scale={scale}
          instruments={instruments}
          status={status}
          reloadKey={reloadKey}
          slotHeight={slotHeight}
          safeTop={safeTop}
          safeBottom={safeBottom}
          pageType={pageType}
          marks={marks}
          contrast={contrast}
          theme={theme}
          canvasRef={canvasRef}
          pageRef={pageRef}
          slotWrapRef={slotWrapRef}
          probeRef={probeRef}
          onStatus={setStatus}
          onMeasure={setPaint}
          onProbe={setProbe}
          onReload={() => setReloadKey((value) => value + 1)}
        />
      </div>
    </section>
  );
}
