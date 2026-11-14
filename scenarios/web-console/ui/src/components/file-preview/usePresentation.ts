import { useCallback, useEffect, useRef, useState } from "react";
import { loadPresentationFrame } from "./presentationFrame";
import { MAX_PUBLICATION_BYTES, isPresentationEvent, type PresentationMetadata, type PublicationError, type PublicationResult } from "./publicationData";

export function usePresentation(href: string, size: number, legacy: boolean, attempt: number, onSlide: (index: number) => void) {
  const frame = useRef<HTMLIFrameElement>(null), [metadata, setMetadata] = useState<PresentationMetadata | null>(null), [error, setError] = useState<PublicationError | null>(null), [warning, setWarning] = useState(false);
  const [source, setSource] = useState<string | null>(null);
  const latestSlide = useRef(onSlide); latestSlide.current = onSlide;
  const post = useCallback((message: object, transfer: Transferable[] = []) => { frame.current?.contentWindow?.postMessage({ channel: "wc-presentation", ...message }, "*", transfer); }, []);
  useEffect(() => {
    setMetadata(null); setError(null); setWarning(false); setSource(null);
    if (legacy || size > MAX_PUBLICATION_BYTES) { setError(legacy ? "legacy" : "limit"); return; }
    let worker: Worker;
    try { worker = new Worker(new URL("./presentation.worker.ts", import.meta.url), { type: "module" }); }
    catch { setError("unavailable"); return; }
    const controller = new AbortController();
    let active = true, failed = false, boot = false, pending: PublicationResult | null = null, opened = false, slideCount = 0;
    const fail = (reason: PublicationError) => {
      if (!active || failed) return;
      failed = true; controller.abort(); pending = null; setSource(null); setError(reason);
      window.clearTimeout(timeout); worker.terminate();
    };
    const timeout = window.setTimeout(() => { fail(opened ? "limit" : "unavailable"); }, 25000);
    void loadPresentationFrame(controller.signal).then(document => {
      if (active && !failed) setSource(document);
    }).catch(() => {
      fail("unavailable");
    });
    const open = () => {
      if (active && !failed && boot && pending && "bytes" in pending && !opened) { opened = true; post({ type: "open", ...pending }, [pending.bytes]); pending = null; }
    };
    const listener = (event: MessageEvent<unknown>) => {
      if (!active || failed || event.source !== frame.current?.contentWindow || !(event.data && typeof event.data === "object" && "channel" in event.data && event.data.channel === "wc-presentation") || !isPresentationEvent(event.data)) return;
      if (event.data.type === "boot") { boot = true; open(); }
      else if (event.data.type === "ready") { slideCount = event.data.metadata.slides.length; setMetadata(event.data.metadata); window.clearTimeout(timeout); }
      else if (event.data.type === "error") fail(event.data.error);
      else if (event.data.type === "warning") setWarning(true);
      else if (event.data.index < slideCount) latestSlide.current(event.data.index);
    };
    window.addEventListener("message", listener);
    // A renewed URL gets a new document and its own boot/onLoad handshake.
    worker.onmessage = (event: MessageEvent<PublicationResult>) => {
      if (!active || failed) return;
      worker.terminate(); pending = event.data;
      if ("error" in pending) fail(pending.error); else open();
    };
    worker.onerror = event => { event.preventDefault(); fail("unavailable"); };
    try { worker.postMessage({ href }); } catch { fail("unavailable"); }
    return () => { active = false; controller.abort(); window.clearTimeout(timeout); worker.terminate(); window.removeEventListener("message", listener); post({ type: "dispose" }); };
  }, [href, size, legacy, attempt, post]);
  return { frame, source, metadata, error, warning, post };
}
