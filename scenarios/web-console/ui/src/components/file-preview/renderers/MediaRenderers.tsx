import { useState } from "react";
import { useTranslation } from "react-i18next";

import { strings } from "../../../consts/strings";
import { CenteredPreview, PreviewActions, PreviewMetaLine, PreviewNotice } from "./shared";
import type { PreviewRendererProps } from "../types";

// ImagePreview renders raster + SVG images from the blob href. SVG is served
// with image/svg+xml and rendered via the blob URL (never injected into the
// DOM), keeping it XSS-safe.
export function ImagePreview({ model }: PreviewRendererProps) {
  const { t } = useTranslation();
  const [failed, setFailed] = useState(false);
  return (
    <CenteredPreview checkerboard testId="file-preview-image">
      {failed ? (
        <PreviewNotice message={t(strings.messagesFileViewer.mediaLoadError)} />
      ) : (
        <img
          src={model.blobHref}
          alt={model.basename}
          onError={() => setFailed(true)}
          className="max-h-full max-w-full rounded-lg border border-wc-default bg-white/95 object-contain p-2 shadow-lg"
        />
      )}
      <PreviewMetaLine model={model} />
      <PreviewActions model={model} />
    </CenteredPreview>
  );
}

// AudioPreview renders native audio controls. Seeking relies on the server's
// HTTP Range support.
export function AudioPreview({ model }: PreviewRendererProps) {
  const { t } = useTranslation();
  const [failed, setFailed] = useState(false);
  return (
    <CenteredPreview testId="file-preview-audio">
      <audio
        controls
        preload="metadata"
        src={model.blobHref}
        onError={() => setFailed(true)}
        aria-label={t(strings.messagesFileViewer.audioPreview)}
        className="w-full max-w-xl"
      />
      {failed && <PreviewNotice message={t(strings.messagesFileViewer.codecHint)} />}
      <PreviewMetaLine model={model} />
      <PreviewActions model={model} />
    </CenteredPreview>
  );
}

// VideoPreview renders native video controls with playsInline for mobile.
export function VideoPreview({ model }: PreviewRendererProps) {
  const { t } = useTranslation();
  const [failed, setFailed] = useState(false);
  return (
    <CenteredPreview checkerboard testId="file-preview-video">
      <video
        controls
        playsInline
        preload="metadata"
        src={model.blobHref}
        onError={() => setFailed(true)}
        aria-label={t(strings.messagesFileViewer.videoPreview)}
        className="max-h-[70vh] max-w-full rounded-lg border border-wc-default bg-black shadow-lg"
      />
      {failed && <PreviewNotice message={t(strings.messagesFileViewer.codecHint)} />}
      <PreviewMetaLine model={model} />
      <PreviewActions model={model} />
    </CenteredPreview>
  );
}

// PdfPreview embeds the PDF via a native browser viewer (iframe at the blob
// href) with a download/open fallback. PDF.js is intentionally deferred.
export function PdfPreview({ model }: PreviewRendererProps) {
  const { t } = useTranslation();
  return (
    <div className="flex h-full flex-col" data-testid="file-preview-pdf">
      <div className="min-h-0 flex-1">
        <iframe
          src={model.blobHref}
          title={t(strings.messagesFileViewer.pdfPreview)}
          className="h-full w-full border-0 bg-wc-surface-base"
        />
      </div>
      <div className="shrink-0 border-t border-wc-default bg-wc-surface-base px-4 py-3">
        <p className="mb-2 text-xs text-wc-text-muted">{t(strings.messagesFileViewer.pdfFallback)}</p>
        <PreviewActions model={model} />
      </div>
    </div>
  );
}
