import runtimeUrl from "virtual:presentation-runtime-url";
import { readPreviewBytes } from "./readPreviewBytes";

// No document bytes or publisher code enter this document; source is the owned
// renderer bundle, fetched as data by the host with same-origin credentials.
export function presentationFrame(source: string): string {
  const nonce = Array.from(crypto.getRandomValues(new Uint8Array(16)), byte => byte.toString(16).padStart(2, "0")).join("");
  // A trusted bundle can still contain HTML end tags in string literals.
  const script = source.replace(/<\/script/gi, "<\\/script");
  return `<!doctype html><html lang="en"><head><title>Presentation</title><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta http-equiv="Content-Security-Policy" content="default-src 'none'; script-src 'nonce-${nonce}'; style-src 'unsafe-inline'; img-src data: blob:; font-src data: blob:; media-src 'none'; connect-src 'none'; worker-src 'none'; base-uri 'none'; form-action 'none'"><style>
  *{box-sizing:border-box}html,body,#presentation{margin:0;width:100%;height:100%;overflow:hidden}body{font-family:system-ui,sans-serif}#presentation{display:flex;background:var(--background,#111821);color:var(--text,#e5eaf0)}.stage{flex:1;min-width:0;overflow:auto;display:grid;padding:12px;outline:none}.stage:focus-visible{box-shadow:inset 0 0 0 2px var(--accent)}.canvas{margin:auto;position:relative;flex:none;overflow:hidden;box-shadow:0 2px 8px #0002}.slides{width:184px;flex:none;overflow:auto;padding:12px;background:var(--surface);border-inline-end:1px solid var(--border)}button{display:block;width:100%;border:0;border-radius:5px;background:transparent;color:var(--muted);padding:6px;margin:0 0 12px;text-align:start;font:inherit;cursor:pointer}button[aria-current]{background:var(--background);outline:2px solid var(--accent);color:var(--text)}button:focus-visible{outline:2px solid var(--accent)}.thumbnail{display:block;width:100%;position:relative;overflow:hidden;background:var(--background)}.caption{display:block;font-size:11px;line-height:1.6;margin-top:6px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.grid-view .slides{width:100%;display:grid;grid-template-columns:repeat(auto-fill,minmax(130px,1fr));gap:12px;align-content:start;border:0}.grid-view button{margin:0;min-height:44px}.thumbnail *{pointer-events:none}audio,video{pointer-events:none}
  </style></head><body><div id="presentation"></div><script nonce="${nonce}">
  const fail = () => parent.postMessage({channel:"wc-presentation",type:"error",error:"unavailable"},"*");
  window.addEventListener("error",fail);window.addEventListener("unhandledrejection",fail);
  </script><script nonce="${nonce}">${script}</script></body></html>`;
}
export async function loadPresentationFrame(signal: AbortSignal): Promise<string> {
  const source = await readPreviewBytes(new URL(runtimeUrl, window.location.href).href, 4 * 1024 * 1024, signal);
  return presentationFrame(new TextDecoder("utf-8", { fatal: true }).decode(source));
}
