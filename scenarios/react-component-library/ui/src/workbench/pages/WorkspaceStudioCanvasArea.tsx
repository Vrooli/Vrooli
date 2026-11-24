import type { RefObject } from "react";
import { Image } from "lucide-react";
import { StateStatus, type WorkspaceDisplay, type WorkspaceState } from "../../api/workspaces";
import type { DesignDocument, DesignFrame } from "../design-types";
import { useTranslation } from "../../i18n";
import { cx } from "../cx";
import { StudioCanvas } from "../studio/StudioCanvas";
import { VisualDriftPanel } from "../studio/VisualDriftPanel";
import { DEVICE_NAME } from "../studio/model";
import { useFrameAudit } from "../studio/useFrameAudit";
import { useWorkspaceStudio } from "../studio/useWorkspaceStudio";
import { WorkspaceComponentSlot } from "./WorkspaceComponentSlot";
import { findSlot, slotPlacement, type TemplateNode } from "../templates/spec";
import type { LayerDropPosition } from "../studio/layerMovement";

interface WorkspaceStudioCanvasAreaProps {
  workspaceId: string;
  studio: ReturnType<typeof useWorkspaceStudio>;
  design: DesignDocument;
  frames: DesignFrame[];
  frame: DesignFrame;
  display: WorkspaceDisplay;
  state: WorkspaceState | null;
  selectedId: string | null;
  scales: Record<string, number>;
  guides: boolean;
  onToggleGuides: () => void;
  zoomOverride: number | null;
  onZoomChange: (zoom: number | null) => void;
  narrow: boolean;
  frameScale: number;
  fitScale: number;
  audits: ReturnType<typeof useFrameAudit>;
  componentId: string;
  storyId: string;
  componentVersion: string;
  selectFrame: (id: string) => void;
  selectNode: (frameId: string, id: string) => void;
  reparentLayer: (
    frameId: string,
    sourceId: string,
    targetId: string,
    position: LayerDropPosition,
  ) => void;
  handleDevice: (device: DesignFrame["device"]) => void;
  dirty: boolean;
  saving: boolean;
  readOnly: boolean;
  capturingState: boolean;
  captureSelectedState: () => Promise<void>;
  canvasRef: RefObject<HTMLDivElement>;
  updateURL: (next: Record<string, string | undefined>) => void;
}

export function WorkspaceStudioCanvasArea({
  workspaceId,
  studio,
  design,
  frames,
  frame,
  display,
  state,
  selectedId,
  scales,
  guides,
  onToggleGuides,
  zoomOverride,
  onZoomChange,
  narrow,
  frameScale,
  fitScale,
  audits,
  componentId,
  storyId,
  componentVersion,
  selectFrame,
  selectNode,
  reparentLayer,
  handleDevice,
  dirty,
  saving,
  readOnly,
  capturingState,
  captureSelectedState,
  canvasRef,
  updateURL,
}: WorkspaceStudioCanvasAreaProps) {
  const { t } = useTranslation();
  return (
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
        {display.states.map((entry) => (
          <button
            key={entry.id}
            type="button"
            aria-pressed={entry.id === state?.id}
            className={cx("studio-chip-btn", entry.id === state?.id && "is-active")}
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
        <button
          type="button"
          className="studio-chip-btn"
          disabled={capturingState || dirty || saving || readOnly || !studio.state}
          aria-busy={capturingState}
          onClick={() => void captureSelectedState()}
          title={t("workbench.workspace.captureRequiresSaved", {
            defaultValue: "Save the selected composition before capturing it.",
          })}
        >
          <Image size={14} aria-hidden="true" />{" "}
          {capturingState
            ? t("workbench.workspace.capturing", { defaultValue: "Capturing…" })
            : state?.status === StateStatus.CAPTURED
              ? t("workbench.workspace.captureAgain", { defaultValue: "Capture again" })
              : t("workbench.workspace.capture", { defaultValue: "Capture state" })}
        </button>
      </div>

      <StudioCanvas
        design={design}
        frames={frames}
        activeFrameId={frame.id}
        selectedId={selectedId}
        scales={scales}
        guides={guides}
        onToggleGuides={onToggleGuides}
        zoomOverride={zoomOverride}
        onZoomChange={onZoomChange}
        fitScale={narrow ? frameScale : fitScale}
        audit={audits}
        slot={
          componentId ? (
            <WorkspaceComponentSlot
              componentId={componentId}
              storyId={storyId}
              componentVersion={componentVersion || undefined}
              height={slotPlacement(findSlot(frame.spec))?.h ?? 64}
            />
          ) : undefined
        }
        slotEditable={Boolean(componentId)}
        renderComponentSlot={(node: TemplateNode) => {
          if (node.id === findSlot(frame.spec)?.id) return undefined;
          const boundComponentId =
            typeof node.props.componentId === "string" ? node.props.componentId : "";
          if (!boundComponentId) return undefined;
          return (
            <WorkspaceComponentSlot
              key={node.id}
              componentId={boundComponentId}
              storyId={typeof node.props.storyId === "string" ? node.props.storyId : ""}
              componentVersion={
                typeof node.props.componentVersion === "string"
                  ? node.props.componentVersion
                  : undefined
              }
              height={slotPlacement(node)?.h ?? 64}
            />
          );
        }}
        onPlacementChange={() => undefined}
        onSelectFrame={selectFrame}
        onSelectNode={selectNode}
        onMoveLayer={readOnly ? undefined : reparentLayer}
        onAddFrame={handleDevice}
        onRemoveFrame={(id) => studio.removeFrame(id)}
        scrollRef={canvasRef}
        saving={saving}
      />
      <VisualDriftPanel workspaceId={workspaceId} stateId={state?.id} />
    </div>
  );
}
