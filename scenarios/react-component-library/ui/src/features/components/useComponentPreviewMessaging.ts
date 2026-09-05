import {
  useCallback,
  useEffect,
  type Dispatch,
  type MutableRefObject,
  type SetStateAction,
} from "react";

import type { SpecimenIdentity } from "./ComponentEditorStage";

export type PreviewEvent = { story: string; name: string; args: unknown[]; ts: number };
type Setter<T> = Dispatch<SetStateAction<T>>;

type PreviewMessagingOptions = {
  id: string;
  previewFrameRef: MutableRefObject<HTMLIFrameElement | null>;
  previewFramesRef: MutableRefObject<Set<HTMLIFrameElement>>;
  specimenFramesRef: MutableRefObject<Map<SpecimenIdentity, HTMLIFrameElement>>;
  setActiveSpecimen: Setter<SpecimenIdentity | null>;
  onSelectedStoryChange?: (story: string) => void;
  setReadyExamples: Setter<ReadonlySet<string>>;
  setSpecimenErrors: Setter<Record<string, string>>;
  setSpecimenRetries: Setter<Record<string, number>>;
  setOverrideStatus: Setter<Record<string, "idle" | "applying" | "applied" | "error">>;
  setOverrideMessages: Setter<Record<string, string>>;
  setPreviewEvents: Setter<PreviewEvent[]>;
  previewFailedMessage: string;
  propsRejectedMessage: string;
};

export function useComponentPreviewMessaging({
  id,
  previewFrameRef,
  previewFramesRef,
  specimenFramesRef,
  setActiveSpecimen,
  onSelectedStoryChange,
  setReadyExamples,
  setSpecimenErrors,
  setSpecimenRetries,
  setOverrideStatus,
  setOverrideMessages,
  setPreviewEvents,
  previewFailedMessage,
  propsRejectedMessage,
}: PreviewMessagingOptions) {
  const postToPreviewFrames = useCallback(
    (message: unknown) => {
      for (const frame of previewFramesRef.current) frame.contentWindow?.postMessage(message, "*");
    },
    [previewFramesRef],
  );

  const postToSpecimen = useCallback(
    (identity: SpecimenIdentity, message: unknown) => {
      specimenFramesRef.current.get(identity)?.contentWindow?.postMessage(message, "*");
    },
    [specimenFramesRef],
  );

  const registerPreviewFrame = useCallback(
    (identity: SpecimenIdentity, frame: HTMLIFrameElement | null) => {
      const previous = specimenFramesRef.current.get(identity);
      if (previous) previewFramesRef.current.delete(previous);
      if (!frame) {
        specimenFramesRef.current.delete(identity);
        if (previewFrameRef.current === previous) previewFrameRef.current = null;
        return;
      }
      specimenFramesRef.current.set(identity, frame);
      previewFramesRef.current.add(frame);
      if (!previewFrameRef.current) previewFrameRef.current = frame;
    },
    [previewFrameRef, previewFramesRef, specimenFramesRef],
  );

  const activateSpecimen = useCallback(
    (identity: SpecimenIdentity) => {
      setActiveSpecimen(identity);
      previewFrameRef.current = specimenFramesRef.current.get(identity) ?? null;
      const story = identity.split(":").slice(1).join(":");
      if (story && story !== "__default__") onSelectedStoryChange?.(story);
    },
    [onSelectedStoryChange, previewFrameRef, setActiveSpecimen, specimenFramesRef],
  );

  const retrySpecimen = useCallback(
    (identity: SpecimenIdentity) => {
      setSpecimenErrors((current) => {
        const next = { ...current };
        delete next[identity];
        return next;
      });
      setSpecimenRetries((current) => ({ ...current, [identity]: (current[identity] ?? 0) + 1 }));
    },
    [setSpecimenErrors, setSpecimenRetries],
  );

  useEffect(() => {
    const handler = (ev: MessageEvent) => {
      const data = ev.data as {
        type?: string;
        id?: string;
        message?: string;
        story?: string;
        version?: string;
        name?: string;
        args?: unknown[];
        ts?: number;
        passed?: boolean;
        failures?: Array<{ message?: string }>;
      } | null;
      const identity: SpecimenIdentity = `${data?.version || "__current__"}:${data?.story || "__default__"}`;
      const frame = specimenFramesRef.current.get(identity);
      if (!data || data.id !== id || !frame || ev.source !== frame.contentWindow) return;
      if (data.type === "preview-ready") {
        setReadyExamples((current) => new Set([...current, identity]));
        setSpecimenErrors((current) => {
          if (!current[identity]) return current;
          const next = { ...current };
          delete next[identity];
          return next;
        });
      } else if (data.type === "preview-error") {
        setSpecimenErrors((current) => ({
          ...current,
          [identity]: data.message || previewFailedMessage,
        }));
      } else if (data.type === "rcl-preview-props-applied") {
        setOverrideStatus((current) => ({ ...current, [identity]: "applied" }));
      } else if (data.type === "rcl-preview-props-reset") {
        setOverrideStatus((current) => ({ ...current, [identity]: "idle" }));
        setOverrideMessages((current) => ({ ...current, [identity]: "" }));
      } else if (data.type === "rcl-preview-props-error") {
        setOverrideStatus((current) => ({ ...current, [identity]: "error" }));
        setOverrideMessages((current) => ({
          ...current,
          [identity]: data.message || propsRejectedMessage,
        }));
      } else if (data.type === "rcl-story-result" && data.passed === false) {
        const details = (data.failures ?? [])
          .map((failure) => failure.message)
          .filter(Boolean)
          .join(" ");
        setSpecimenErrors((current) => ({
          ...current,
          [identity]: details || "Story interactions or expectations failed.",
        }));
      } else if (data.type === "rcl-preview-event" && typeof data.name === "string") {
        setPreviewEvents((current) =>
          [
            {
              story: data.story ?? "",
              name: data.name!,
              args: Array.isArray(data.args) ? data.args : [],
              ts: typeof data.ts === "number" ? data.ts : Date.now(),
            },
            ...current,
          ].slice(0, 200),
        );
      }
    };
    window.addEventListener("message", handler);
    return () => window.removeEventListener("message", handler);
  }, [
    id,
    previewFailedMessage,
    propsRejectedMessage,
    setOverrideMessages,
    setOverrideStatus,
    setPreviewEvents,
    setReadyExamples,
    setSpecimenErrors,
    specimenFramesRef,
  ]);

  return {
    postToPreviewFrames,
    postToSpecimen,
    registerPreviewFrame,
    activateSpecimen,
    retrySpecimen,
  };
}
