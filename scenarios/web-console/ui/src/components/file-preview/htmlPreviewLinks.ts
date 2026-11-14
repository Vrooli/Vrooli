import runtimeSource from "./htmlPreviewLinksRuntime.js?raw";

export function htmlPreviewLinksScript(): string {
  return `<script type="module" data-wc-html-links>${runtimeSource}\ninstallHtmlPreviewLinks(window);</script>`;
}

// Resolve links against the actual file, never the API/tunnel URL. Navigation
// is read-only and goes through the session's normal resolver and viewer stack;
// no file bytes or authority are returned to the authored iframe.
export function htmlPreviewFilePath(href: string, documentPath: string): string | null {
  const raw = href.trim();
  if (!raw || raw.length > 8192 || raw.startsWith("#") || raw.startsWith("//") || /[\0\r\n]/.test(raw)) return null;
  if (/^[a-z][a-z0-9+.-]*:/i.test(raw) && !/^file:\/\//i.test(raw)) return null;
  try {
    const base = new URL("file:///");
    base.pathname = documentPath;
    const target = new URL(raw, base);
    if (target.protocol !== "file:" || (target.hostname && target.hostname !== "localhost")) return null;
    const path = decodeURIComponent(target.pathname);
    return /[\0\r\n]/.test(path) ? null : path;
  } catch {
    return null;
  }
}

export function attachHtmlPreviewLinks(
  frame: HTMLIFrameElement,
  documentPath: string,
  navigate: (path: string) => void,
  activated = () => navigator.userActivation.isActive,
): () => void {
  const message = (event: MessageEvent<unknown>) => {
    if (event.source !== frame.contentWindow || typeof event.data !== "object" || event.data === null) return;
    if (!("type" in event.data) || event.data.type !== "wc-html-file-link" || !("href" in event.data) || typeof event.data.href !== "string" || !activated()) return;
    const path = htmlPreviewFilePath(event.data.href, documentPath);
    if (path) navigate(path);
  };
  window.addEventListener("message", message);
  return () => { window.removeEventListener("message", message); };
}
