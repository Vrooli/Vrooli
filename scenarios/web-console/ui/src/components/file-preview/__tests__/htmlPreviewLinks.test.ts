import { afterEach, describe, expect, it, vi } from "vitest";
import { attachHtmlPreviewLinks, htmlPreviewFilePath } from "../htmlPreviewLinks";
import { installHtmlPreviewLinks } from "../htmlPreviewLinksRuntime.js";

afterEach(() => { vi.restoreAllMocks(); });

describe("HTML evidence link paths", () => {
  it.each([
    ["runs/episode/script.md", "/research/video/runs/episode/script.md"],
    ["../books/interior.html", "/research/books/interior.html"],
    ["./source%20notes.json#result", "/research/video/source notes.json"],
    ["/tmp/preview.mp4", "/tmp/preview.mp4"],
    ["file:///tmp/preview.mp4", "/tmp/preview.mp4"],
    ["/tmp/%E6%9B%B8%E9%A1%9E.md", "/tmp/書類.md"],
  ])("resolves %s against the actual document", (href, expected) => {
    expect(htmlPreviewFilePath(href, "/research/video/REVIEW.html")).toBe(expected);
  });
  it.each(["#review", "https://example.com/api", "//example.com/path", "javascript:alert(1)", "data:text/html,a", "file://other-host/path", "file:///tmp/%00file", "bad%escape", "", "x".repeat(8193)])("rejects non-file or malformed input %s", (href) => {
    expect(htmlPreviewFilePath(href, "/research/video/REVIEW.html")).toBeNull();
  });
});

describe("HTML link transport", () => {
  it("requires the active frame and a current user activation, then hands off only a path", () => {
    const frame = document.createElement("iframe");
    document.body.append(frame);
    const navigate = vi.fn(), activation = vi.fn(() => true);
    const dispose = attachHtmlPreviewLinks(frame, "/research/video/REVIEW.html", navigate, activation);
    const send = (href: unknown, source: MessageEventSource | null = frame.contentWindow) => {
      window.dispatchEvent(new MessageEvent("message", { source, data: { type: "wc-html-file-link", href } }));
    };
    try {
      send("runs/one/script.md", window);
      activation.mockReturnValueOnce(false); send("runs/one/script.md");
      send({ path: "script.md" }); send("https://example.com/api");
      expect(navigate).not.toHaveBeenCalled();
      send("runs/one/script.md");
      expect(navigate).toHaveBeenCalledTimes(1);
      expect(navigate).toHaveBeenCalledWith("/research/video/runs/one/script.md");
      dispose(); send("other.md");
      expect(navigate).toHaveBeenCalledTimes(1);
    } finally { dispose(); frame.remove(); }
  });

  it("scrolls fragments without navigating through the runtime base, including missing IDs", () => {
    const frame = document.createElement("iframe"); document.body.append(frame);
    const child = frame.contentWindow, doc = frame.contentDocument;
    if (!child || !doc) throw new Error("Expected frame document");
    doc.head.innerHTML = '<base href="https://console.test/api/preview/runtime/">';
    doc.body.innerHTML = '<a href="#review">Review</a><h2 id="review">Artifacts</h2><a href="#not-found">Missing</a>';
    const send = vi.spyOn(window, "postMessage").mockImplementation(() => {});
    const heading = doc.getElementById("review");
    if (!heading) throw new Error("Expected heading");
    const scroll = vi.fn(); heading.scrollIntoView = scroll;
    const dispose = installHtmlPreviewLinks(child);
    try {
      for (const anchor of doc.querySelectorAll("a")) {
        const event = new MouseEvent("click", { bubbles: true, cancelable: true });
        anchor.dispatchEvent(event); expect(event.defaultPrevented).toBe(true);
      }
      expect(scroll).toHaveBeenCalledTimes(1);
      expect(scroll).toHaveBeenCalledWith({ block: "start" });
      expect(send).not.toHaveBeenCalled();
    } finally { dispose(); frame.remove(); }
  });

  it("prevents authored scripts from opening local files with synthetic clicks", () => {
    const frame = document.createElement("iframe"); document.body.append(frame);
    const child = frame.contentWindow, doc = frame.contentDocument;
    if (!child || !doc) throw new Error("Expected frame document");
    doc.body.innerHTML = '<a href="../private.md">Read</a>';
    const send = vi.spyOn(window, "postMessage").mockImplementation(() => {});
    const dispose = installHtmlPreviewLinks(child);
    try {
      const event = new MouseEvent("click", { bubbles: true, cancelable: true });
      doc.querySelector("a")?.dispatchEvent(event);
      expect(event.defaultPrevented).toBe(true);
      expect(send).not.toHaveBeenCalled();
    } finally { dispose(); frame.remove(); }
  });
});
