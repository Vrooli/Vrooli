import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { installHtmlPreviewPinch } from "../htmlPreviewPinchRuntime.js";
import { attachHtmlPreviewInspection } from "../htmlPreviewPinch";

describe("HTML inspection compositor", () => {
  let frame: HTMLIFrameElement;
  let surface: HTMLDivElement;
  let stage: HTMLDivElement;
  let dispose: () => void;
  let pending: FrameRequestCallback | null;
  let child: ReturnType<typeof vi.spyOn>;

  const send = (data: Record<string, unknown>, source: MessageEventSource | null = frame.contentWindow) => {
    window.dispatchEvent(new MessageEvent("message", { source, data: { type: "wc-html-gesture", ...data } }));
  };
  const flush = () => { const callback = pending; pending = null; callback?.(0); };
  const start = () => { send({ kind: "pinch-start", distance: 100, x: 150, y: 100, localX: 600, localY: 400 }); };
  const move = (distance: number, x = 150, y = 100) => { send({ kind: "pinch-move", distance, x, y }); };

  beforeEach(() => {
    frame = document.createElement("iframe"); surface = document.createElement("div"); stage = document.createElement("div");
    stage.append(surface); surface.append(frame); document.body.append(stage);
    if (!frame.contentWindow) throw new Error("Expected an iframe window");
    child = vi.spyOn(frame.contentWindow, "postMessage").mockImplementation(() => {});
    pending = null;
    vi.stubGlobal("requestAnimationFrame", vi.fn((callback: FrameRequestCallback) => { pending = callback; return 1; }));
    vi.stubGlobal("cancelAnimationFrame", vi.fn());
    dispose = attachHtmlPreviewInspection(frame, surface, stage, 0.25, 360, 600);
  });
  afterEach(() => { dispose(); stage.remove(); vi.restoreAllMocks(); vi.unstubAllGlobals(); });

  it("coalesces movement into one paint without measuring or mutating the document", () => {
    const body = frame.contentDocument?.body;
    if (!body) throw new Error("Expected authored content");
    body.style.cssText = "transform-origin:60% 30%;scale:1.5;background:navy";
    const original = body.style.cssText;
    const bounds = vi.spyOn(body, "getBoundingClientRect");
    start(); move(150); move(180); move(200);
    expect(requestAnimationFrame).toHaveBeenCalledTimes(1);
    flush();
    expect(surface.dataset.wcHtmlInspectionScale).toBe("2");
    expect(frame.style.transform).toBe("translate3d(-150px, -100px, 0) scale(0.5)");
    expect(body.style.cssText).toBe(original);
    expect(bounds).not.toHaveBeenCalled();
    expect(child).toHaveBeenCalledWith({ type: "wc-html-inspection", pan: true }, "*");
  });

  it("uses stable screen coordinates when the transformed iframe's local coordinates change", () => {
    start(); move(200, 170, 110); flush();
    expect(frame.style.transform).toBe("translate3d(-130px, -90px, 0) scale(0.5)");
    move(300, 170, 110); flush();
    expect(frame.style.transform).toBe("translate3d(-280px, -190px, 0) scale(0.75)");
  });

  it("clamps all edges and restores a fully covered viewport after repeated pinches", () => {
    for (let i = 0; i < 20; i++) {
      start(); move(1000, 10000, -10000); flush();
      expect(frame.style.transform).toBe("translate3d(0px, -2400px, 0) scale(1.25)");
      send({ kind: "end" }); start(); move(1, -10000, 10000); flush();
      expect(frame.style.transform).toBe("scale(0.25)");
      expect(surface.dataset.wcHtmlInspectionScale).toBe("1");
    }
  });

  it("pans inside bounds and only scrolls the document after reaching an edge", () => {
    start(); move(200); flush(); child.mockClear();
    send({ kind: "end" }); send({ kind: "pan-start", x: 200, y: 200 });
    send({ kind: "pan-move", x: 150, y: 150 }); flush();
    expect(frame.style.transform).toBe("translate3d(-200px, -150px, 0) scale(0.5)");
    expect(child).not.toHaveBeenCalled();
    send({ kind: "pan-move", x: 150, y: -650 }); flush();
    expect(frame.style.transform).toBe("translate3d(-200px, -600px, 0) scale(0.5)");
    expect(child).toHaveBeenCalledWith({ type: "wc-html-inspection", scrollX: 0, scrollY: 700 }, "*");
  });

  it("rejects foreign frames and malformed coordinates, and cancels gestures", () => {
    send({ kind: "pinch-start", distance: 100, x: 0, y: 0, localX: 0, localY: 0 }, window);
    move(200); expect(pending).toBeNull();
    start(); move(NaN); expect(pending).toBeNull();
    send({ kind: "cancel" }); move(200); expect(pending).toBeNull();
    start(); move(200); dispose();
    expect(cancelAnimationFrame).toHaveBeenCalled();
  });

  it("supports focal trackpad zoom and leaves the frame's layout dimensions alone", () => {
    frame.style.width = "1440px"; frame.style.height = "2400px";
    send({ kind: "wheel", delta: -Math.log(2) * 100, localX: 600, localY: 400 }); flush();
    expect(surface.dataset.wcHtmlInspectionScale).toBe("2");
    expect(frame.style.width).toBe("1440px"); expect(frame.style.height).toBe("2400px");
  });
});

describe("isolated HTML gesture transport", () => {
  it("forwards stable screen coordinates without changing authored body styles", () => {
    const frame = document.createElement("iframe"); document.body.append(frame);
    const preview = frame.contentWindow;
    if (!preview) throw new Error("Expected iframe window");
    const body = preview.document.body;
    body.style.cssText = "background:navy;scale:1.5;transform-origin:20% 40%";
    const original = body.style.cssText;
    const messages = vi.spyOn(preview.parent, "postMessage").mockImplementation(() => {});
    const dispose = installHtmlPreviewPinch(preview);
    const touch = (type: string, points: number[]) => {
      const event = new Event(type, { cancelable: true, bubbles: true });
      Object.defineProperty(event, "touches", { value: points.map((screenX) => ({ screenX, screenY: 200, clientX: screenX * 4, clientY: 800 })) });
      body.dispatchEvent(event); return event;
    };
    expect(touch("touchstart", [100]).defaultPrevented).toBe(false);
    expect(touch("touchmove", [150]).defaultPrevented).toBe(false);
    expect(touch("touchstart", [100, 200]).defaultPrevented).toBe(true);
    expect(touch("touchmove", [50, 250]).defaultPrevented).toBe(true);
    expect(messages).toHaveBeenCalledWith({ type: "wc-html-gesture", kind: "pinch-move", distance: 200, x: 150, y: 200 }, "*");
    expect(body.style.cssText).toBe(original);
    dispose(); messages.mockRestore(); frame.remove();
  });
});
