/* eslint-disable jsx-a11y/no-static-element-interactions, jsx-a11y/click-events-have-key-events --
 * A click anywhere on a frame is a pointer shortcut for "make this the active
 * frame". The frame label button beside it is the keyboard path, and the layer
 * tree is the complete keyboard path for selection. */
import { useState, type ReactNode, type RefObject } from "react";
import { Frame, Minus, Monitor, Plus, Ruler, Smartphone, Tablet, X } from "lucide-react";

import { useTranslation } from "../../i18n";
import type { DesignDevice, DesignDocument, DesignFrame } from "../designs";
import type { SlotPlacement } from "../templates/spec";
import { findSlot, slotPlacement } from "../templates/spec";
import { ErrorBoundary } from "../../components/ErrorBoundary";
import { DeviceFrame } from "../component/DeviceFrame";
import { TemplateRenderer } from "../templates/TemplateRenderer";
import { DEVICE_KEY, DEVICE_NAME, FRAME_SIZE, frameTemplate } from "./model";
import type { FrameAuditMap } from "./checks";
import { cx } from "../cx";

const DEVICE_ICON = {
  desktop: Monitor,
  tablet: Tablet,
  phone: Smartphone,
} as const;

export function StudioCanvas({
  design,
  frames,
  activeFrameId,
  selectedId,
  scales,
  guides,
  onToggleGuides,
  zoomOverride,
  onZoomChange,
  fitScale,
  audit,
  slot,
  slotEditable,
  onPlacementChange,
  onSelectFrame,
  onSelectNode,
  onAddFrame,
  onRemoveFrame,
  scrollRef,
  saving,
}: {
  design: DesignDocument;
  frames: DesignFrame[];
  activeFrameId: string;
  selectedId: string | null;
  scales: Record<string, number>;
  guides: boolean;
  onToggleGuides: () => void;
  zoomOverride: number | null;
  onZoomChange: (zoom: number | null) => void;
  fitScale: number;
  audit: FrameAuditMap;
  slot: ReactNode;
  slotEditable: boolean;
  onPlacementChange: (frameId: string, nodeId: string, placement: SlotPlacement) => void;
  onSelectFrame: (id: string) => void;
  onSelectNode: (frameId: string, id: string) => void;
  onAddFrame: (device: DesignDevice) => void;
  onRemoveFrame: (id: string) => void;
  scrollRef: RefObject<HTMLDivElement>;
  saving: boolean;
}) {
  const { t } = useTranslation();
  const [addOpen, setAddOpen] = useState(false);
  const zoom = zoomOverride ?? fitScale;

  return (
    <main className="studio-canvas" data-experience-surface="design-workspace">
      <div className="studio-canvas-bar">
        <div className="studio-zoom" role="group" aria-label={t("workbench.studio.zoom", { defaultValue: "Zoom" })}>
          <button
            type="button"
            className="studio-iconbtn"
            aria-label={t("workbench.studio.zoomOut", { defaultValue: "Zoom out" })}
            onClick={() => onZoomChange(Math.max(0.15, Math.round((zoom - 0.1) * 100) / 100))}
          >
            <Minus aria-hidden size={12} />
          </button>
          <span className="studio-zoom-value">{Math.round(zoom * 100)}%</span>
          <button
            type="button"
            className="studio-iconbtn"
            aria-label={t("workbench.studio.zoomIn", { defaultValue: "Zoom in" })}
            onClick={() => onZoomChange(Math.min(1, Math.round((zoom + 0.1) * 100) / 100))}
          >
            <Plus aria-hidden size={12} />
          </button>
          <button
            type="button"
            className="studio-chip-btn"
            aria-pressed={zoomOverride === null}
            onClick={() => onZoomChange(null)}
          >
            {t("workbench.studio.fit", { defaultValue: "Fit" })}
          </button>
        </div>

        <button
          type="button"
          className={cx("studio-chip-btn", guides && "is-active")}
          aria-pressed={guides}
          onClick={onToggleGuides}
        >
          <Ruler aria-hidden size={12} />
          {t("workbench.studio.guides", { defaultValue: "Guides" })}
        </button>

        <div className="studio-canvas-spacer" />

        {saving ? (
          <span className="studio-canvas-chip">
            {t("workbench.studio.saving", { defaultValue: "Saving…" })}
          </span>
        ) : null}
        <span className="studio-canvas-chip">
          {t("workbench.studio.frameCountLabel", {
            defaultValue: "{{count}} frames",
            count: frames.length,
          })}
        </span>

        <div className="studio-add-frame">
          <button
            type="button"
            className="studio-chip-btn"
            aria-expanded={addOpen}
            onClick={() => setAddOpen((value) => !value)}
          >
            <Frame aria-hidden size={12} />
            {t("workbench.studio.addFrame", { defaultValue: "Add frame" })}
          </button>
          {addOpen ? (
            <>
              <button
                type="button"
                className="studio-dismiss"
                aria-label={t("workbench.studio.closeMenu", { defaultValue: "Close menu" })}
                onClick={() => setAddOpen(false)}
              />
              <div
                className="studio-menu studio-menu-compact"
                role="dialog"
                aria-label={t("workbench.studio.addFrame", { defaultValue: "Add frame" })}
              >
                {(["desktop", "tablet", "phone"] as DesignDevice[]).map((device) => {
                  const Icon = DEVICE_ICON[device];
                  return (
                    <button
                      key={device}
                      type="button"
                      className="studio-menu-row"
                      onClick={() => {
                        setAddOpen(false);
                        onAddFrame(device);
                      }}
                    >
                      <Icon aria-hidden size={13} />
                      {t(DEVICE_KEY[device], { defaultValue: DEVICE_NAME[device] })}
                    </button>
                  );
                })}
              </div>
            </>
          ) : null}
        </div>
      </div>

      <div className="studio-canvas-scroll" ref={scrollRef}>
        <div className="studio-frames">
          {frames.map((frame) => {
            const active = frame.id === activeFrameId;
            const size = FRAME_SIZE[frame.device];
            const scale = scales[frame.id] ?? fitScale;
            const Icon = DEVICE_ICON[frame.device];
            const placement = slotPlacement(findSlot(frame.spec));
            const frameAudit = audit[frame.id];
            return (
              <div
                key={frame.id}
                className={cx("studio-frame", active && "is-selected")}
                data-studio-frame={frame.id}
                onClick={() => onSelectFrame(frame.id)}
              >
                <div className="studio-frame-label">
                  <button
                    type="button"
                    className="studio-frame-select"
                    aria-pressed={active}
                    onClick={() => onSelectFrame(frame.id)}
                  >
                    <Icon aria-hidden size={12} />
                    <b>{frame.name}</b>
                    <span>
                      {size.width} × {size.height}
                    </span>
                    <span>{Math.round(scale * 100)}%</span>
                    {frameAudit && frameAudit.measured && frameAudit.overflow > 0 ? (
                      <span className="studio-frame-warn">
                        {t("workbench.studio.overflow", { defaultValue: "overflow" })}
                      </span>
                    ) : null}
                  </button>
                  {frames.length > 1 ? (
                    <button
                      type="button"
                      className="studio-frame-remove"
                      aria-label={t("workbench.studio.removeFrame", {
                        defaultValue: "Remove {{name}} frame",
                        name: frame.name,
                      })}
                      onClick={() => onRemoveFrame(frame.id)}
                    >
                      <X aria-hidden size={11} />
                    </button>
                  ) : null}
                </div>
                <DeviceFrame
                  device={frame.device}
                  width={size.width}
                  height={size.height}
                  scale={scale}
                  selected={active}
                  guides={guides}
                  label={t("workbench.studio.frameLabel", {
                    defaultValue: "{{name}} at {{width}} by {{height}}",
                    name: frame.name,
                    width: size.width,
                    height: size.height,
                  })}
                >
                  <div className="studio-frame-page">
                    <ErrorBoundary
                      fallback={
                        <div className="studio-frame-error" role="alert">
                          <b>
                            {t("workbench.studio.frameCrashed", {
                              defaultValue: "This frame could not be rendered",
                            })}
                          </b>
                          <span>
                            {t("workbench.studio.frameCrashedHint", {
                              defaultValue:
                                "The rest of the studio still works. Undo the last change or pick another design.",
                            })}
                          </span>
                        </div>
                      }
                    >
                      <TemplateRenderer
                      template={frameTemplate(design, frame)}
                      spec={frame.spec}
                      mode="canvas"
                      selectedId={active ? (selectedId ?? undefined) : undefined}
                      onSelect={(id) => onSelectNode(frame.id, id)}
                      slot={slot}
                      slotEditable={slotEditable}
                      slotPlacement={placement}
                      onSlotPlacementChange={(nodeId, next) => onPlacementChange(frame.id, nodeId, next)}
                      />
                    </ErrorBoundary>
                  </div>
                </DeviceFrame>
              </div>
            );
          })}
        </div>
      </div>
    </main>
  );
}
