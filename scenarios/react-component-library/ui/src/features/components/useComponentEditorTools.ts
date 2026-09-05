import { useCallback } from "react";
import type { Dispatch, RefObject, SetStateAction } from "react";
import type { ComponentStory } from "../../api/components";
import type { PreviewSpecimen, SpecimenIdentity } from "./ComponentEditorStage";
import type { PreviewDiagnostics } from "./ComponentEditorTools";
import type { PreviewEvent } from "./useComponentPreviewMessaging";

type OverrideStatus = Record<SpecimenIdentity, "idle" | "applying" | "applied" | "error">;

type EditorToolOptions = {
  activeSpecimen: SpecimenIdentity | null;
  specimens: Array<PreviewSpecimen | undefined>;
  id: string;
  postToSpecimen: (identity: SpecimenIdentity, message: unknown) => void;
  setSpecimenOverrides: Dispatch<SetStateAction<Record<string, Record<string, unknown>>>>;
  setOverrideStatus: Dispatch<SetStateAction<OverrideStatus>>;
  setOverrideMessages: Dispatch<SetStateAction<Record<string, string>>>;
  setPreviewEvents: Dispatch<SetStateAction<PreviewEvent[]>>;
  activeExample: PreviewSpecimen | undefined;
  activeSpecimenLabel: string | undefined;
  storyContract: ComponentStory | undefined;
  inspector: unknown;
  specimenOverrides: Record<string, Record<string, unknown>>;
  overrideStatus: OverrideStatus;
  overrideMessages: Record<string, string>;
  previewEvents: PreviewEvent[];
  previewDiagnostics: PreviewDiagnostics;
  previewStageRef: RefObject<HTMLElement | null>;
  previewFullscreen: boolean;
  setPreviewFullscreen: Dispatch<SetStateAction<boolean>>;
};

export function useComponentEditorTools(options: EditorToolOptions) {
  const {
    activeSpecimen,
    specimens,
    id,
    postToSpecimen,
    setSpecimenOverrides,
    setOverrideStatus,
    setOverrideMessages,
    setPreviewEvents,
    previewStageRef,
    previewFullscreen,
    setPreviewFullscreen,
  } = options;

  const applyPropsOverride = useCallback(
    (props: Record<string, unknown>, environment?: Record<string, string>) => {
      if (!activeSpecimen) return;
      const example = specimens.find(
        (candidate) => candidate && `${candidate.version}:${candidate.storyId}` === activeSpecimen,
      );
      setSpecimenOverrides((current) => ({ ...current, [activeSpecimen]: props }));
      setOverrideStatus((current) => ({ ...current, [activeSpecimen]: "applying" }));
      setOverrideMessages((current) => ({ ...current, [activeSpecimen]: "" }));
      postToSpecimen(activeSpecimen, {
        type: "rcl-preview-props-override",
        componentId: id,
        story: example?.storyId || "",
        version: example?.version || "",
        props,
        environment: environment ?? example?.environment ?? {},
      });
    },
    [
      activeSpecimen,
      id,
      postToSpecimen,
      setOverrideMessages,
      setOverrideStatus,
      setSpecimenOverrides,
      specimens,
    ],
  );

  const resetPropsOverride = useCallback(() => {
    if (!activeSpecimen) return;
    const example = specimens.find(
      (candidate) => candidate && `${candidate.version}:${candidate.storyId}` === activeSpecimen,
    );
    setSpecimenOverrides((current) =>
      Object.fromEntries(Object.entries(current).filter(([key]) => key !== activeSpecimen)),
    );
    setOverrideStatus((current) => ({ ...current, [activeSpecimen]: "applying" }));
    postToSpecimen(activeSpecimen, {
      type: "rcl-preview-props-reset",
      componentId: id,
      story: example?.storyId || "",
      version: example?.version || "",
    });
  }, [activeSpecimen, id, postToSpecimen, setOverrideStatus, setSpecimenOverrides, specimens]);

  const togglePreviewFullscreen = useCallback(async () => {
    const stage = previewStageRef.current;
    if (!stage) return;
    if (previewFullscreen) {
      if (document.fullscreenElement === stage && typeof document.exitFullscreen === "function")
        await document.exitFullscreen();
      setPreviewFullscreen(false);
      return;
    }
    try {
      if (typeof stage.requestFullscreen === "function") await stage.requestFullscreen();
    } catch {
      // Embedded hosts may deny native fullscreen; the fixed preview remains usable.
    }
    setPreviewFullscreen(true);
  }, [previewFullscreen, previewStageRef, setPreviewFullscreen]);

  return {
    editorToolProps: {
      activeSpecimen: options.activeSpecimen,
      activeExample: options.activeExample,
      activeSpecimenLabel: options.activeSpecimenLabel,
      storyContract: options.storyContract,
      inspector: options.inspector,
      overrideStatus: options.overrideStatus,
      specimenOverrides: options.specimenOverrides,
      overrideMessages: options.overrideMessages,
      previewEvents: options.previewEvents,
      previewDiagnostics: options.previewDiagnostics,
      onApply: applyPropsOverride,
      onReset: resetPropsOverride,
      onClearEvents: () => setPreviewEvents([]),
    },
    togglePreviewFullscreen,
  };
}
