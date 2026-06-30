import { useCallback, useRef, useState } from "react";
import { useTranslation } from "react-i18next";

import { getFilePreviewText, resolveFilePreview, type PreviewSourceContext } from "../../api/filePreview";
import { APIError } from "../../lib/errors";
import { strings } from "../../consts/strings";
import { IDLE_PREVIEW_STATE, type PreviewState } from "./types";

// useFilePreviewController owns the file-preview viewer's state machine. It is
// the single seam MessagesPane (and future surfaces) call to open a path; the
// component only renders the returned state. Resolves are request-id guarded so
// a fast second open never lands a stale first response.
export function useFilePreviewController(sessionId: string) {
  const { t } = useTranslation();
  const [state, setState] = useState<PreviewState>(IDLE_PREVIEW_STATE);
  const reqIdRef = useRef(0);

  const messageFromError = useCallback(
    (err: unknown): string => {
      if (err instanceof APIError) return err.message;
      if (err instanceof Error && err.message) return err.message;
      return t(strings.messagesPane.fileOpenFailed);
    },
    [t],
  );

  const openPreview = useCallback(
    async (path: string, source?: PreviewSourceContext) => {
      const reqId = ++reqIdRef.current;
      setState({
        status: "resolving",
        open: true,
        requestedPath: path,
        model: null,
        text: null,
        error: null,
      });

      try {
        const model = await resolveFilePreview(sessionId, path, source);
        if (reqId !== reqIdRef.current) return;

        if (!model.canPreview && !model.textContentAvailable) {
          // Resolvable but no dedicated renderer — show metadata + download.
          setState((s) => ({ ...s, status: "unsupported", model }));
          return;
        }

        if (model.textContentAvailable) {
          setState((s) => ({ ...s, status: "loadingText", model }));
          const text = await getFilePreviewText(sessionId, model.previewId);
          if (reqId !== reqIdRef.current) return;
          setState((s) => ({ ...s, status: "ready", model, text }));
          return;
        }

        // Blob-rendered kind (svg/image/pdf/audio/video) — bytes load lazily in
        // the renderer via model.blobHref.
        setState((s) => ({
          ...s,
          status: model.canPreview ? "ready" : "unsupported",
          model,
          text: null,
        }));
      } catch (err) {
        if (reqId !== reqIdRef.current) return;
        setState((s) => ({ ...s, status: "error", error: messageFromError(err) }));
      }
    },
    [messageFromError, sessionId],
  );

  const reopen = useCallback(() => {
    if (state.requestedPath) void openPreview(state.requestedPath);
  }, [openPreview, state.requestedPath]);

  const reportError = useCallback((message: string) => {
    setState((s) => ({ ...s, status: "error", error: message }));
  }, []);

  const close = useCallback(() => {
    reqIdRef.current++;
    setState(IDLE_PREVIEW_STATE);
  }, []);

  return { state, openPreview, reopen, reportError, close };
}
