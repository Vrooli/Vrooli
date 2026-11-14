import { afterEach, describe, expect, it, vi } from "vitest";
import { loadPresentationFrame, presentationFrame } from "../presentationFrame";

afterEach(() => { vi.unstubAllGlobals(); });

describe("presentation startup isolation", () => {
  it("executes only nonce-authorized owned scripts without anonymous asset requests", () => {
    const source = presentationFrame("window.ownedRenderer = true;"), doc = new DOMParser().parseFromString(source, "text/html");
    const policy = doc.querySelector('meta[http-equiv="Content-Security-Policy"]')?.getAttribute("content") ?? "";
    const scripts = Array.from(doc.scripts), nonce = scripts[0]?.getAttribute("nonce");
    expect(nonce).toMatch(/^[a-f0-9]{32}$/);
    expect(scripts).toHaveLength(2);
    scripts.forEach(script => { expect(script.getAttribute("nonce")).toBe(nonce); expect(script.hasAttribute("src")).toBe(false); });
    expect(policy).toContain(`script-src 'nonce-${nonce ?? ""}'`);
    expect(policy).toContain("connect-src 'none'"); expect(policy).toContain("worker-src 'none'");
    expect(policy).not.toContain("script-src 'unsafe-inline'");
    expect(source).not.toBe(presentationFrame("window.ownedRenderer = true;"));
  });
  it("keeps script end tags inside the source and reports uncaught startup errors", () => {
    const doc = new DOMParser().parseFromString(presentationFrame('const text = "</ScRiPt><img src=x onerror=alert(1)>";'), "text/html");
    expect(doc.scripts).toHaveLength(2); expect(doc.querySelector("img")).toBeNull();
    expect(doc.scripts[1]?.textContent).toContain('<\\/script>');
    expect(doc.scripts[0]?.textContent).toContain('type:"error",error:"unavailable"');
    expect(doc.scripts[0]?.textContent).toContain('"unhandledrejection"');
  });
  it("fetches the owned runtime with host credentials and an abortable request", async () => {
    const fetch = vi.fn().mockResolvedValue(new Response("// owned renderer", { status: 200 }));
    vi.stubGlobal("fetch", fetch);
    const controller = new AbortController(), source = await loadPresentationFrame(controller.signal);
    expect(fetch).toHaveBeenCalledWith(expect.stringContaining("presentation-runtime.js"), { credentials: "same-origin", signal: controller.signal });
    expect(source).toContain("// owned renderer");
  });
  it("rejects failed and over-budget renderer downloads before producing a frame", async () => {
    const fetch = vi.fn().mockResolvedValueOnce(new Response("Login required", { status: 403 })).mockResolvedValueOnce(new Response("// oversized", { headers: { "Content-Length": String(4 * 1024 * 1024 + 1) } }));
    vi.stubGlobal("fetch", fetch);
    await expect(loadPresentationFrame(new AbortController().signal)).rejects.toThrow("fetch");
    await expect(loadPresentationFrame(new AbortController().signal)).rejects.toThrow(RangeError);
  });

});
