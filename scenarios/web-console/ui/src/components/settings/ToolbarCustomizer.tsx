// DOC: docs/reference/configuration.md#mobile-toolbar-layout
/**
 * Mobile toolbar settings.
 *
 * The preview is not a drawing of the toolbar — it renders the same controls
 * through the same layout engine, differing only in the width it feeds in. A
 * preview with its own layout code is a preview that eventually disagrees with
 * the thing it previews.
 */
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { TriangleAlert } from "lucide-react";
import { Button } from "../ui/button";
import { strings } from "../../consts/strings";
import { cn } from "../../lib/classnames";
import { useWorkspaceStore } from "../../stores/useWorkspaceStore";
import {
  MIN_RECOMMENDED_TOUCH_TARGET_PX,
  TOOLBAR_CONTROLS,
  TOOLBAR_DENSITY_PX,
  layoutToolbar,
  type ToolbarArrowStyle,
  type ToolbarDensity,
  type ToolbarOverflowStyle,
  type ToolbarPresetId,
  type ToolbarRowBudget,
} from "../../lib/toolbarLayout";
import ToolbarSurface from "../toolbar/ToolbarSurface";
import type { ToolbarControlContext } from "../toolbar/toolbarControls";

/** Widths the preview offers besides the device's own. */
const SAMPLE_WIDTHS = [360, 430] as const;
/** Guard rails for the "this device" reading in unusual viewports. */
const MIN_PREVIEW_WIDTH = 280;
const MAX_PREVIEW_WIDTH = 768;

interface Option<T> {
  value: T;
  label: string;
  testId: string;
}

function Segmented<T extends string | number>({
  ariaLabel,
  value,
  options,
  onChange,
}: {
  ariaLabel: string;
  value: T;
  options: readonly Option<T>[];
  onChange: (next: T) => void;
}) {
  return (
    <div role="group" aria-label={ariaLabel} className="flex flex-wrap items-center gap-2">
      {options.map((option) => (
        <Button
          key={String(option.value)}
          data-testid={option.testId}
          aria-pressed={option.value === value}
          variant={option.value === value ? "default" : "outline"}
          size="sm"
          className="h-8 px-3"
          onClick={() => { onChange(option.value); }}
        >
          {option.label}
        </Button>
      ))}
    </div>
  );
}

export default function ToolbarCustomizer() {
  const { t } = useTranslation();
  const prefs = useWorkspaceStore((s) => s.toolbarPrefs);
  const setToolbarPreset = useWorkspaceStore((s) => s.setToolbarPreset);
  const updateToolbarPrefs = useWorkspaceStore((s) => s.updateToolbarPrefs);
  const setToolbarControlEnabled = useWorkspaceStore((s) => s.setToolbarControlEnabled);

  const [open, setOpen] = useState(false);

  const deviceWidth = useMemo(() => {
    const raw = typeof window === "undefined" ? MIN_PREVIEW_WIDTH : window.innerWidth;
    return Math.min(MAX_PREVIEW_WIDTH, Math.max(MIN_PREVIEW_WIDTH, Math.round(raw)));
  }, []);
  const [previewWidth, setPreviewWidth] = useState<number>(deviceWidth);

  const layout = useMemo(() => layoutToolbar(prefs, previewWidth), [prefs, previewWidth]);

  const controlLabels = useMemo<Record<string, string>>(() => ({
    more: t(strings.mobileToolbar.controls.more),
    modifiers: t(strings.mobileToolbar.controls.modifiers),
    special: t(strings.mobileToolbar.controls.special),
    arrows: t(strings.mobileToolbar.controls.arrows),
    mic: t(strings.mobileToolbar.controls.mic),
    image: t(strings.mobileToolbar.uploadImageTitle),
    ai: t(strings.mobileToolbar.aiCommandTitle),
    snippets: t(strings.snippets.picker.title),
  }), [t]);

  /**
   * Inert copies: no handlers, no focus, no assistive-tech presence. The
   * visuals come from the same renderer the live toolbar uses.
   */
  const previewContext = useMemo<ToolbarControlContext>(() => ({
    inert: true,
    onKey: () => {},
    modifiers: { ctrl: false, alt: false, shift: false },
    toggleModifier: () => {},
    aiSuggestActive: false,
    labels: controlLabels,
    voice: {
      supported: true,
      isPreparing: false,
      isRecording: false,
      isTranscribing: false,
      error: null,
      onStart: () => {},
      onStop: () => {},
    },
  }), [controlLabels]);

  const belowRecommendedTarget = TOOLBAR_DENSITY_PX[prefs.density] < MIN_RECOMMENDED_TOUCH_TARGET_PX;

  const presetOptions: Option<Exclude<ToolbarPresetId, "custom">>[] = [
    { value: "dense", label: t(strings.settings.workspaceSection.presetDense), testId: "toolbar-preset-dense" },
    { value: "balanced", label: t(strings.settings.workspaceSection.presetBalanced), testId: "toolbar-preset-balanced" },
    { value: "essential", label: t(strings.settings.workspaceSection.presetEssential), testId: "toolbar-preset-essential" },
  ];

  return (
    <div className="space-y-3" data-testid="toolbar-customizer">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex flex-wrap items-center gap-2">
          <Segmented
            ariaLabel={t(strings.settings.workspaceSection.mobileToolbarLabel)}
            value={prefs.preset as Exclude<ToolbarPresetId, "custom">}
            options={presetOptions}
            onChange={setToolbarPreset}
          />
          {prefs.preset === "custom" && (
            <span
              data-testid="toolbar-preset-custom"
              className="rounded-full border border-wc-default px-2 py-0.5 text-[11px] text-wc-text-muted"
            >
              {t(strings.settings.workspaceSection.presetCustom)}
            </span>
          )}
        </div>
        <Button
          data-testid="toolbar-customize-toggle"
          variant="outline"
          size="sm"
          className="h-8 px-3"
          aria-expanded={open}
          onClick={() => { setOpen((v) => !v); }}
        >
          {open
            ? t(strings.settings.workspaceSection.toolbarCustomizeClose)
            : t(strings.settings.workspaceSection.toolbarCustomize)}
        </Button>
      </div>

      {open && (
        <div data-testid="toolbar-customizer-panel" className="space-y-4 rounded-xl border border-wc-default bg-wc-surface-base/60 p-3">
          {/* ── Preview ────────────────────────────────────────────────── */}
          <div className="space-y-2">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <span className="text-[11px] font-semibold uppercase tracking-[0.18em] text-wc-text-muted">
                {t(strings.settings.workspaceSection.toolbarPreviewLabel)}
              </span>
              <Segmented
                ariaLabel={t(strings.settings.workspaceSection.toolbarPreviewLabel)}
                value={previewWidth}
                options={[
                  {
                    value: deviceWidth,
                    label: `${t(strings.settings.workspaceSection.toolbarPreviewWidthDevice)} · ${String(deviceWidth)}`,
                    testId: "toolbar-preview-width-device",
                  },
                  ...SAMPLE_WIDTHS.filter((w) => w !== deviceWidth).map((w) => ({
                    value: w as number,
                    label: String(w),
                    testId: `toolbar-preview-width-${String(w)}`,
                  })),
                ]}
                onChange={setPreviewWidth}
              />
            </div>

            {/* The device is rendered at its true pixel width so the readout
                below describes exactly what is on screen. */}
            <div className="overflow-x-auto">
              <div
                data-testid="toolbar-preview-device"
                className="wc-chrome-surface-raised rounded-lg border border-wc-default"
                style={{ width: previewWidth }}
              >
                <ToolbarSurface
                  testId="toolbar-preview-surface"
                  layout={layout}
                  ctx={previewContext}
                  // Decorative: the live toolbar is the real control.
                  className="pointer-events-none"
                />
              </div>
            </div>

            <p data-testid="toolbar-preview-readout" className="text-[11px] text-wc-text-muted" role="status">
              {t(strings.settings.workspaceSection.toolbarRowsReadout, { count: layout.rowCount })}
              {" · "}
              {t(strings.settings.workspaceSection.toolbarHeightReadout, { height: layout.keysHeightPx })}
              {" · "}
              {layout.overflow.length === 0
                ? t(strings.settings.workspaceSection.toolbarAllPlaced)
                : t(strings.settings.workspaceSection.toolbarInMore, { count: layout.overflow.length })}
            </p>
          </div>

          {/* ── Button size ────────────────────────────────────────────── */}
          <div className="space-y-2">
            <div className="text-sm font-medium text-wc-text-secondary">
              {t(strings.settings.workspaceSection.toolbarDensityLabel)}
            </div>
            <Segmented<ToolbarDensity>
              ariaLabel={t(strings.settings.workspaceSection.toolbarDensityLabel)}
              value={prefs.density}
              options={[
                { value: "compact", label: t(strings.settings.workspaceSection.toolbarDensityCompact), testId: "toolbar-density-compact" },
                { value: "standard", label: t(strings.settings.workspaceSection.toolbarDensityStandard), testId: "toolbar-density-standard" },
                { value: "large", label: t(strings.settings.workspaceSection.toolbarDensityLarge), testId: "toolbar-density-large" },
              ]}
              onChange={(density) => { updateToolbarPrefs({ density }); }}
            />
            {/* Warn, never block: trading target size for more controls per row
                is a legitimate preference, and the person making it knows. */}
            {belowRecommendedTarget && (
              <p data-testid="toolbar-density-warning" className="flex items-start gap-1.5 text-[11px] text-yellow-500">
                <TriangleAlert aria-hidden className="mt-0.5 h-3 w-3 shrink-0" />
                <span>{t(strings.settings.workspaceSection.toolbarDensityWarning)}</span>
              </p>
            )}
          </div>

          {/* ── Arrow keys ─────────────────────────────────────────────── */}
          <div className="space-y-2">
            <div className="text-sm font-medium text-wc-text-secondary">
              {t(strings.settings.workspaceSection.toolbarArrowsLabel)}
            </div>
            <Segmented<ToolbarArrowStyle | "off">
              ariaLabel={t(strings.settings.workspaceSection.toolbarArrowsLabel)}
              value={prefs.enabled.arrows === false ? "off" : prefs.arrows}
              options={[
                { value: "dpad", label: t(strings.settings.workspaceSection.toolbarArrowsDpad), testId: "toolbar-arrows-dpad" },
                { value: "inline", label: t(strings.settings.workspaceSection.toolbarArrowsInline), testId: "toolbar-arrows-inline" },
                { value: "off", label: t(strings.settings.workspaceSection.toolbarArrowsHidden), testId: "toolbar-arrows-off" },
              ]}
              onChange={(next) => {
                if (next === "off") setToolbarControlEnabled("arrows", false);
                else updateToolbarPrefs({ arrows: next, enabled: { ...prefs.enabled, arrows: true } });
              }}
            />
          </div>

          {/* ── Row budget ─────────────────────────────────────────────── */}
          <div className="space-y-2">
            <div className="text-sm font-medium text-wc-text-secondary">
              {t(strings.settings.workspaceSection.toolbarRowBudgetLabel)}
            </div>
            <Segmented<ToolbarRowBudget>
              ariaLabel={t(strings.settings.workspaceSection.toolbarRowBudgetLabel)}
              value={prefs.maxRows}
              options={[1, 2, 3].map((n) => ({
                value: n as ToolbarRowBudget,
                label: t(strings.settings.workspaceSection.toolbarRowsReadout, { count: n }),
                testId: `toolbar-rows-${String(n)}`,
              }))}
              onChange={(maxRows) => { updateToolbarPrefs({ maxRows }); }}
            />
            <p className="text-[11px] text-wc-text-muted">
              {t(strings.settings.workspaceSection.toolbarRowBudgetHint)}
            </p>
          </div>

          {/* ── Overflow ───────────────────────────────────────────────── */}
          <div className="space-y-2">
            <div className="text-sm font-medium text-wc-text-secondary">
              {t(strings.settings.workspaceSection.toolbarOverflowLabel)}
            </div>
            <Segmented<ToolbarOverflowStyle>
              ariaLabel={t(strings.settings.workspaceSection.toolbarOverflowLabel)}
              value={prefs.overflow}
              options={[
                { value: "strip", label: t(strings.settings.workspaceSection.toolbarOverflowStrip), testId: "toolbar-overflow-strip" },
                { value: "more", label: t(strings.settings.workspaceSection.toolbarOverflowMore), testId: "toolbar-overflow-more" },
              ]}
              onChange={(overflow) => { updateToolbarPrefs({ overflow }); }}
            />
            <p className="text-[11px] text-wc-text-muted">
              {t(strings.settings.workspaceSection.toolbarOverflowHint)}
            </p>
          </div>

          {/* ── Which controls ─────────────────────────────────────────── */}
          <div className="space-y-2">
            <div className="text-sm font-medium text-wc-text-secondary">
              {t(strings.settings.workspaceSection.toolbarShowLabel)}
            </div>
            <div className="flex flex-col">
              {TOOLBAR_CONTROLS.map((spec) => {
                const id = String(spec.id);
                const checked = spec.pinned || prefs.enabled[id] !== false;
                return (
                  <label
                    key={id}
                    className={cn(
                      "flex items-center gap-3 rounded px-1 py-1.5 text-sm",
                      spec.pinned ? "cursor-default" : "cursor-pointer hover:bg-wc-surface-input",
                    )}
                  >
                    <input
                      data-testid={`toolbar-control-${id}`}
                      type="checkbox"
                      checked={checked}
                      // The overflow host is where hidden controls live. A
                      // toggle that can strand its own contents is a trap, so
                      // it is pinned rather than merely defaulted on.
                      disabled={spec.pinned}
                      onChange={(e) => { setToolbarControlEnabled(id, e.currentTarget.checked); }}
                      className="h-4 w-4 shrink-0 accent-wc-accent"
                    />
                    <span className={cn("min-w-0 truncate", checked ? "text-wc-text-primary" : "text-wc-text-muted")}>
                      {controlLabels[id] ?? id}
                    </span>
                    {spec.pinned && (
                      <span className="ms-auto shrink-0 text-[11px] text-wc-text-muted">
                        {t(strings.settings.workspaceSection.toolbarControlPinned)}
                      </span>
                    )}
                  </label>
                );
              })}
            </div>
            <p className="text-[11px] text-wc-text-muted">
              {t(strings.settings.workspaceSection.toolbarShowHint)}
            </p>
          </div>
        </div>
      )}
    </div>
  );
}
