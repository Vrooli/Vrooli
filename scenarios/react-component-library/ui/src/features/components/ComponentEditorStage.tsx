/** @vrooliComponentSource manipulation.split-pane */
import {
  useEffect,
  useRef,
  useState,
  type DragEvent,
  type KeyboardEvent,
  type ReactNode,
  type Ref,
} from "react";

import type { DeviceEmulationValue } from "../../hooks/useDeviceEmulation";
import type { DeviceFiltersValue } from "../../hooks/useDeviceFilters";
import type { PreviewKit } from "./ThemeSwitcher";
import type { PreviewFrameCandidate } from "../../api/components";
import { canvasDropTarget, ComponentEditorStageView } from "./ComponentEditorStageView";

export type SpecimenIdentity = `${string}:${string}`;
export type PreviewSpecimen = {
  id: string;
  componentId: string;
  libraryId: string;
  version: string;
  name: string;
  displayName: string;
  propsJson: string;
  environment: Record<string, string>;
  expectJson: string;
  sourcePath: string;
  storyId: string;
  description?: string;
};

type StageProps = {
  emulator: DeviceEmulationValue;
  filters: DeviceFiltersValue;
  stageMode: boolean;
  comparisonActive: boolean;
  specimens: Array<PreviewSpecimen | undefined>;
  visibleSpecimens: Array<PreviewSpecimen | undefined>;
  readyExamples: ReadonlySet<string>;
  previewMessage: string;
  specimenErrors: Record<string, string>;
  specimenRetries: Record<string, number>;
  comparedSpecimens: ReadonlySet<SpecimenIdentity>;
  activeSpecimen: SpecimenIdentity | null;
  previewKit: PreviewKit;
  frameEnabled: boolean;
  frameOverride?: PreviewFrameCandidate;
  id: string;
  baselineSha: string;
  previewReloadKey: number;
  selectedVersion?: string;
  resolvedPreviewTheme: string;
  previewCanvasRef: Ref<HTMLDivElement>;
  tools: ReactNode;
  /** The layout has room to dock the tools beside the preview at all. */
  toolsDocked: boolean;
  /** The docked tools are currently expanded rather than collapsed away. */
  toolsOpen: boolean;
  onClearComparison: () => void;
  onSelectAllComparison: () => void;
  onToggleComparison: (identity: SpecimenIdentity) => void;
  onRetrySpecimen: (identity: SpecimenIdentity) => void;
  onRegisterPreviewFrame: (identity: SpecimenIdentity, frame: HTMLIFrameElement | null) => void;
  onPreviewLoad: (identity: SpecimenIdentity) => void;
  onPreviewError: (identity: SpecimenIdentity) => void;
  postToPreviewFrames: (message: unknown) => void;
  onCloseTools: () => void;
  onEnterSpecimen: (identity: SpecimenIdentity) => void;
};

export function ComponentEditorStage({
  emulator,
  filters,
  stageMode,
  comparisonActive,
  specimens,
  visibleSpecimens,
  readyExamples,
  previewMessage,
  specimenErrors,
  specimenRetries,
  comparedSpecimens,
  activeSpecimen,
  previewKit,
  frameEnabled,
  frameOverride,
  id,
  baselineSha,
  previewReloadKey,
  selectedVersion,
  resolvedPreviewTheme,
  previewCanvasRef,
  tools,
  toolsDocked,
  toolsOpen,
  onClearComparison,
  onSelectAllComparison,
  onToggleComparison,
  onRetrySpecimen,
  onRegisterPreviewFrame,
  onPreviewLoad,
  onPreviewError,
  postToPreviewFrames,
  onCloseTools,
  onEnterSpecimen,
}: StageProps) {
  const [canvasOffset, setCanvasOffset] = useState({ x: 0, y: 0 });
  const canvasRef = useRef<HTMLDivElement>(null);
  const dragRef = useRef<{ x: number; y: number; originX: number; originY: number } | null>(null);
  const identityFor = (example?: PreviewSpecimen): SpecimenIdentity =>
    `${example?.version || "__current__"}:${example?.name || "__default__"}`;
  const specimenOrderKey = specimens.map(identityFor).join("\u0000");
  const [caseOrder, setCaseOrder] = useState<SpecimenIdentity[]>(() => specimens.map(identityFor));
  const [draggedCase, setDraggedCase] = useState<SpecimenIdentity | null>(null);
  const [dropTarget, setDropTarget] = useState<{
    identity: SpecimenIdentity;
    placement: "before" | "after";
  } | null>(null);
  const dropTargetRef = useRef<typeof dropTarget>(null);

  const clearCaseDrag = () => {
    setDraggedCase(null);
    dropTargetRef.current = null;
    setDropTarget(null);
  };

  const updateCanvasDropTarget = (event: DragEvent<HTMLDivElement>) => {
    if (!draggedCase) return;
    event.preventDefault();
    event.stopPropagation();
    event.dataTransfer.dropEffect = "move";
    const nextTarget = canvasDropTarget(
      event.currentTarget,
      draggedCase,
      event.clientX,
      event.clientY,
    );
    dropTargetRef.current = nextTarget;
    setDropTarget(nextTarget);
  };

  useEffect(() => {
    const sourceOrder = specimenOrderKey
      ? (specimenOrderKey.split("\u0000") as SpecimenIdentity[])
      : [];
    setCaseOrder((current) => {
      const available = new Set(sourceOrder);
      const next = current.filter((identity) => available.has(identity));
      for (const identity of sourceOrder) {
        if (!next.includes(identity)) next.push(identity);
      }
      return next.length === current.length &&
        next.every((identity, index) => identity === current[index])
        ? current
        : next;
    });
  }, [specimenOrderKey]);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const onWheel = (event: WheelEvent) => {
      if (stageMode) return;
      event.preventDefault();
      if (event.ctrlKey) {
        const currentIndex = emulator.zoomLevels.reduce(
          (closest, level, index) =>
            Math.abs(level - emulator.zoom) <
            Math.abs((emulator.zoomLevels[closest] ?? emulator.zoom) - emulator.zoom)
              ? index
              : closest,
          0,
        );
        const direction = event.deltaY > 0 ? -1 : 1;
        const nextIndex = Math.max(
          0,
          Math.min(emulator.zoomLevels.length - 1, currentIndex + direction),
        );
        emulator.setZoom(emulator.zoomLevels[nextIndex] ?? emulator.zoom);
        return;
      }
      setCanvasOffset((current) => ({
        x: current.x - event.deltaX,
        y: current.y - event.deltaY,
      }));
    };
    canvas.addEventListener("wheel", onWheel, { passive: false });
    return () => canvas.removeEventListener("wheel", onWheel);
  }, [emulator, stageMode]);

  const orderIndex = new Map(caseOrder.map((identity, index) => [identity, index]));
  const orderedVisibleSpecimens = [...visibleSpecimens].sort(
    (left, right) =>
      (orderIndex.get(identityFor(left)) ?? Number.MAX_SAFE_INTEGER) -
      (orderIndex.get(identityFor(right)) ?? Number.MAX_SAFE_INTEGER),
  );

  const moveCase = (
    moving: SpecimenIdentity,
    target: SpecimenIdentity,
    placement: "before" | "after",
  ) => {
    if (moving === target) return;
    setCaseOrder((current) => {
      const next = current.filter((identity) => identity !== moving);
      const targetIndex = next.indexOf(target);
      if (targetIndex < 0) return current;
      next.splice(targetIndex + (placement === "after" ? 1 : 0), 0, moving);
      return next;
    });
  };

  const moveCaseByKeyboard = (identity: SpecimenIdentity, event: KeyboardEvent) => {
    const currentIndex = caseOrder.indexOf(identity);
    if (currentIndex < 0) return;
    let nextIndex = currentIndex;
    if (event.key === "ArrowLeft" || event.key === "ArrowUp") nextIndex--;
    else if (event.key === "ArrowRight" || event.key === "ArrowDown") nextIndex++;
    else if (event.key === "Home") nextIndex = 0;
    else if (event.key === "End") nextIndex = caseOrder.length - 1;
    else return;
    event.preventDefault();
    event.stopPropagation();
    nextIndex = Math.max(0, Math.min(caseOrder.length - 1, nextIndex));
    if (nextIndex === currentIndex) return;
    setCaseOrder((current) => {
      const next = [...current];
      next.splice(currentIndex, 1);
      next.splice(nextIndex, 0, identity);
      return next;
    });
  };
  return (
    <ComponentEditorStageView
      model={{
        emulator,
        filters,
        stageMode,
        comparisonActive,
        specimens,
        visibleSpecimens,
        readyExamples,
        previewMessage,
        specimenErrors,
        specimenRetries,
        comparedSpecimens,
        activeSpecimen,
        previewKit,
        frameEnabled,
        frameOverride,
        id,
        baselineSha,
        previewReloadKey,
        selectedVersion,
        resolvedPreviewTheme,
        previewCanvasRef,
        tools,
        toolsDocked,
        toolsOpen,
        onClearComparison,
        onSelectAllComparison,
        onToggleComparison,
        onRetrySpecimen,
        onRegisterPreviewFrame,
        onPreviewLoad,
        onPreviewError,
        postToPreviewFrames,
        onCloseTools,
        onEnterSpecimen,
        canvasRef,
        canvasOffset,
        setCanvasOffset,
        dragRef,
        draggedCase,
        dropTarget,
        dropTargetRef,
        setDropTarget,
        setDraggedCase,
        caseOrder,
        updateCanvasDropTarget,
        clearCaseDrag,
        moveCase,
        moveCaseByKeyboard,
        orderedVisibleSpecimens,
        identityFor,
      }}
    />
  );
}
