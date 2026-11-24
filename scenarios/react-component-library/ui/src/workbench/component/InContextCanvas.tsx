import type { RefObject } from "react";

import { useTranslation } from "../../i18n";
import type { CatalogAsset, WorkbenchStory } from "../data";
import { TemplateRenderer } from "../templates/TemplateRenderer";
import type { Template } from "../templates/spec";
import { DeviceFrame } from "./DeviceFrame";
import { FixtureFrame, type FixtureStatus } from "./FixtureFrame";
import type { PlacementDecision } from "./placement";
import { REFERENCE_WIDTHS, type DeviceKind, type ViewportPreset } from "./viewports";

interface PreviewTheme {
  id: string;
  tokens: Record<string, string>;
  resolved: "light" | "dark";
}

interface CanvasMark {
  x: number;
  y: number;
  w: number;
  h: number;
  tone: "red" | "amber";
}

interface TypographyReading {
  fontFamily: string;
  fontSize: string;
  color: string;
}

interface ContrastReading {
  ratio: number | null;
  rect: { x: number; y: number; w: number; h: number } | null;
}

interface InContextCanvasProps {
  asset: CatalogAsset;
  story?: WorkbenchStory;
  version: string;
  template: Template;
  renderSpec: Template["spec"];
  decision: PlacementDecision;
  device: DeviceKind;
  viewport: ViewportPreset;
  scale: number;
  instruments: { guides: boolean; blur: boolean; contrast: boolean };
  status: FixtureStatus;
  reloadKey: number;
  slotHeight: number;
  safeTop: number;
  safeBottom: number;
  pageType: TypographyReading | null;
  marks: CanvasMark[];
  contrast: ContrastReading | null;
  theme: PreviewTheme;
  canvasRef: RefObject<HTMLDivElement>;
  pageRef: RefObject<HTMLDivElement>;
  slotWrapRef: RefObject<HTMLDivElement>;
  probeRef: RefObject<HTMLDivElement>;
  onStatus: (status: FixtureStatus) => void;
  onReload: () => void;
}

export function InContextCanvas({
  asset,
  story,
  version,
  template,
  renderSpec,
  decision,
  device,
  viewport,
  scale,
  instruments,
  status,
  reloadKey,
  slotHeight,
  safeTop,
  safeBottom,
  pageType,
  marks,
  contrast,
  theme,
  canvasRef,
  pageRef,
  slotWrapRef,
  probeRef,
  onStatus,
  onReload,
}: InContextCanvasProps) {
  const { t } = useTranslation();

  return (
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
                      onStatus={onStatus}
                      onRetry={onReload}
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
        {decision.chosen
          ? t("workbench.component.ctxCaption", {
              defaultValue:
                "{{template}} · placed in {{slot}} · {{width}} × {{height}} at {{scale}}% scale",
              template: template.name,
              slot: decision.chosen.label,
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
          <button type="button" className="ic-link" onClick={onReload}>
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
  );
}
