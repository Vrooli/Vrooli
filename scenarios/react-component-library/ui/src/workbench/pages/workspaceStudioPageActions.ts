import type { DesignDevice, DesignFrame } from "../design-types";
import { useWorkspaceStudio } from "../studio/useWorkspaceStudio";
import { CONTAINER_TYPES, type AddableNode } from "../studio/model";
import {
  reparentLayer as reparentLayerNode,
  type LayerDropPosition,
} from "../studio/layerMovement";
import {
  appendChild,
  duplicateNode,
  moveNode,
  node,
  removeNode,
  type TemplateNode,
} from "../templates/spec";
import type { ReferenceUpload } from "../studio/assets";

export type WorkspaceStudioPageTranslate = (
  key: string,
  options?: { defaultValue?: string; [name: string]: unknown },
) => string;
type WorkspaceStudioApi = ReturnType<typeof useWorkspaceStudio>;

interface WorkspaceStudioPageActionInput {
  studio: WorkspaceStudioApi;
  frames: DesignFrame[];
  frame: DesignFrame;
  selected: TemplateNode | undefined;
  translate: WorkspaceStudioPageTranslate;
  updateURL: (next: Record<string, string | undefined>) => void;
  flash: (message: string) => void;
}

/** Owns the page's user-triggered frame and layer operations. */
export function createWorkspaceStudioPageActions({
  studio,
  frames,
  frame,
  selected,
  translate: t,
  updateURL,
  flash,
}: WorkspaceStudioPageActionInput) {
  const commitFrame = (
    mutate: (spec: TemplateNode) => TemplateNode,
    label: string,
    mergeKey?: string,
  ) => studio.updateFrame(frame.id, mutate, label, mergeKey);

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

  const reparentLayer = (sourceId: string, targetId: string, position: LayerDropPosition) => {
    if (sourceId === frame.spec.id) return;
    commitFrame(
      (spec) => reparentLayerNode(spec, sourceId, targetId, position),
      t("workbench.studio.reparentLayer", { defaultValue: "Reparent layer" }),
    );
    studio.selectNode(sourceId);
  };

  const reparentLayerInFrame = (
    frameId: string,
    sourceId: string,
    targetId: string,
    position: LayerDropPosition,
  ) => {
    const targetFrame = frames.find((entry) => entry.id === frameId);
    if (!targetFrame || sourceId === targetFrame.spec.id) return;
    studio.updateFrame(
      frameId,
      (spec) => reparentLayerNode(spec, sourceId, targetId, position),
      t("workbench.studio.reparentLayer", { defaultValue: "Reparent layer" }),
    );
    if (frameId !== frame.id) studio.selectFrame(frameId);
    studio.selectNode(sourceId);
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

  return {
    addLayer,
    commitFrame,
    duplicateLayer,
    handleDevice,
    moveLayer,
    placeAsset,
    reparentLayer,
    reparentLayerInFrame,
    removeLayer,
    selectFrame,
    selectNode,
    share,
  };
}

export type WorkspaceStudioPageActionSet = ReturnType<typeof createWorkspaceStudioPageActions>;
