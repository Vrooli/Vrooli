import { useEffect, useState } from "react";
import { readPreviewBytes } from "./readPreviewBytes";
import type { PreviewModel, PreviewTextContent } from "./types";

// HTML can contain embedded media and exceed the inline-text RPC budget.
// Read the inert blob with the same bound as non-media runtime resources;
// execution still happens only in HtmlPreview's opaque, CSP-limited iframe.
export const MAX_HTML_PREVIEW_BYTES = 16 * 1024 * 1024;

export function useHtmlPreview(model: PreviewModel, text: PreviewTextContent | null) {
  const [loaded, setLoaded] = useState<{ href: string; content: string } | null>(null);
  const [failure, setFailure] = useState<{ href: string; reason: "limit" | "fetch" } | null>(null);
  const href = model.blobHref;
  const needsBlob = text === null && !model.archiveMember;
  const overLimit = model.sizeBytes > MAX_HTML_PREVIEW_BYTES;

  useEffect(() => {
    if (!needsBlob || overLimit || !href) return;
    const request = new AbortController();
    const timer = setTimeout(() => { request.abort(); }, 30_000);
    let active = true;
    void readPreviewBytes(href, MAX_HTML_PREVIEW_BYTES, request.signal).then((bytes) => {
      const content = new TextDecoder("utf-8", { fatal: true }).decode(bytes);
      if (content.includes("\0")) throw new Error("invalid text");
      if (active) setLoaded({ href, content });
    }).catch((error: unknown) => {
      if (active) setFailure({ href, reason: error instanceof RangeError ? "limit" : "fetch" });
    }).finally(() => { clearTimeout(timer); });
    return () => { active = false; clearTimeout(timer); request.abort(); };
  }, [href, needsBlob, overLimit]);

  return {
    content: text?.content ?? (loaded?.href === href ? loaded.content : null),
    error: needsBlob && overLimit ? "limit" : failure?.href === href ? failure.reason : needsBlob && !href ? "fetch" : null,
  };
}
