import type { PreviewKind, PreviewModel, PreviewTextContent } from "../../api/filePreview";

export type { PreviewKind, PreviewModel, PreviewTextContent };

// PreviewStatus is the viewer state machine. Every open() walks
// idle → resolving → (loadingText →) ready | unsupported | error, and any
// renderer load failure (e.g. a 409 stale blob) transitions to error.
export type PreviewStatus =
  | "idle"
  | "resolving"
  | "loadingText"
  | "ready"
  | "unsupported"
  | "error";

export interface PreviewState {
  status: PreviewStatus;
  open: boolean;
  requestedPath: string | null;
  model: PreviewModel | null;
  text: PreviewTextContent | null;
  error: string | null;
}

export const IDLE_PREVIEW_STATE: PreviewState = {
  status: "idle",
  open: false,
  requestedPath: null,
  model: null,
  text: null,
  error: null,
};

// PreviewRendererProps is the uniform contract every renderer receives. text is
// non-null only for text-kind previews; blob kinds read model.blobHref.
export interface PreviewRendererProps {
  model: PreviewModel;
  text: PreviewTextContent | null;
  /** Renderers call this when native media/blob loading fails. */
  onError: (message: string) => void;
}

export type PreviewRenderer = (props: PreviewRendererProps) => JSX.Element;
