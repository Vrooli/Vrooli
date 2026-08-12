import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { strings } from "../../../consts/strings";
import { CenteredPreview, PreviewActions, PreviewMetaLine, PreviewNotice } from "./shared";
// ImagePreview renders raster + SVG images from the blob href. SVG is served
// with image/svg+xml and rendered via the blob URL (never injected into the
// DOM), keeping it XSS-safe.
export function ImagePreview({ model }) {
    const { t } = useTranslation();
    const [failed, setFailed] = useState(false);
    return (_jsxs(CenteredPreview, { checkerboard: true, testId: "file-preview-image", children: [failed ? (_jsx(PreviewNotice, { message: t(strings.messagesFileViewer.mediaLoadError) })) : (_jsx("img", { src: model.blobHref, alt: model.basename, onError: () => setFailed(true), className: "max-h-full max-w-full rounded-lg border border-wc-default bg-white/95 object-contain p-2 shadow-lg" })), _jsx(PreviewMetaLine, { model: model }), _jsx(PreviewActions, { model: model })] }));
}
// AudioPreview renders native audio controls. Seeking relies on the server's
// HTTP Range support.
export function AudioPreview({ model }) {
    const { t } = useTranslation();
    const [failed, setFailed] = useState(false);
    return (_jsxs(CenteredPreview, { testId: "file-preview-audio", children: [_jsx("audio", { controls: true, preload: "metadata", src: model.blobHref, onError: () => setFailed(true), "aria-label": t(strings.messagesFileViewer.audioPreview), className: "w-full max-w-xl" }), failed && _jsx(PreviewNotice, { message: t(strings.messagesFileViewer.codecHint) }), _jsx(PreviewMetaLine, { model: model }), _jsx(PreviewActions, { model: model })] }));
}
// VideoPreview renders native video controls with playsInline for mobile.
export function VideoPreview({ model }) {
    const { t } = useTranslation();
    const [failed, setFailed] = useState(false);
    return (_jsxs(CenteredPreview, { checkerboard: true, testId: "file-preview-video", children: [_jsx("video", { controls: true, playsInline: true, preload: "metadata", src: model.blobHref, onError: () => setFailed(true), "aria-label": t(strings.messagesFileViewer.videoPreview), className: "max-h-[70vh] max-w-full rounded-lg border border-wc-default bg-black shadow-lg" }), failed && _jsx(PreviewNotice, { message: t(strings.messagesFileViewer.codecHint) }), _jsx(PreviewMetaLine, { model: model }), _jsx(PreviewActions, { model: model })] }));
}
// PdfPreview embeds the PDF via a native browser viewer (iframe at the blob
// href) with a download/open fallback. PDF.js is intentionally deferred.
export function PdfPreview({ model }) {
    const { t } = useTranslation();
    return (_jsxs("div", { className: "flex h-full flex-col", "data-testid": "file-preview-pdf", children: [_jsx("div", { className: "min-h-0 flex-1", children: _jsx("iframe", { src: model.blobHref, title: t(strings.messagesFileViewer.pdfPreview), className: "h-full w-full border-0 bg-wc-surface-base" }) }), _jsxs("div", { className: "shrink-0 border-t border-wc-default bg-wc-surface-base px-4 py-3", children: [_jsx("p", { className: "mb-2 text-xs text-wc-text-muted", children: t(strings.messagesFileViewer.pdfFallback) }), _jsx(PreviewActions, { model: model })] })] }));
}
