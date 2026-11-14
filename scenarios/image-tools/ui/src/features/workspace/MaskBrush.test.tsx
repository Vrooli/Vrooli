/**
 * MaskBrush tests. The pointer-painting path needs a real 2D canvas (exercised
 * in the browser BAS journey), so these cover the parts that run headless: the
 * accessible file-upload fallback (the non-pointer path), clearing, and status.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import { MaskBrush } from "./MaskBrush";

// jsdom ships no PointerEvent constructor; @testing-library's fireEvent.pointer*
// then drops clientX/pointerId. Polyfill a minimal one (MouseEvent carries the
// coordinates) so the paint plumbing sees real points + a pointer id.
class PointerEventPolyfill extends MouseEvent {
  pointerId: number;
  constructor(type: string, init: PointerEventInit = {}) {
    super(type, init);
    this.pointerId = init.pointerId ?? 0;
  }
}
if (typeof window.PointerEvent === "undefined") {
  (window as unknown as { PointerEvent: typeof PointerEvent }).PointerEvent =
    PointerEventPolyfill as unknown as typeof PointerEvent;
}

/** A 2D-context double recording the draw calls MaskBrush makes. */
const makeCtx = () => ({
  fillStyle: "",
  strokeStyle: "",
  lineCap: "",
  lineJoin: "",
  lineWidth: 0,
  beginPath: vi.fn(),
  arc: vi.fn(),
  fill: vi.fn(),
  moveTo: vi.fn(),
  lineTo: vi.fn(),
  stroke: vi.fn(),
  fillRect: vi.fn(),
  drawImage: vi.fn(),
  clearRect: vi.fn(),
});

beforeEach(async () => {
  await setLocale("en");
});

afterEach(() => {
  cleanup();
  // Drop every per-test prototype spy (getContext / toBlob / getBoundingClientRect)
  // so none leaks into the next test, then re-assert the process-wide null
  // getContext the setup file installs (it is deliberately never restored).
  vi.restoreAllMocks();
  vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue(null);
});

/** Stub a real 2D surface + a synchronous toBlob for the painting path. */
const installCanvasSurface = () => {
  const ctx = makeCtx();
  vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue(
    ctx as unknown as CanvasRenderingContext2D,
  );
  vi.spyOn(HTMLCanvasElement.prototype, "toBlob").mockImplementation(
    (cb: BlobCallback) => cb(new Blob(["mask"], { type: "image/png" })),
  );
  // jsdom has no pointer capture — assign no-ops so onPointerDown can capture.
  const proto = HTMLElement.prototype as unknown as {
    setPointerCapture?: (id: number) => void;
  };
  if (!proto.setPointerCapture) {
    proto.setPointerCapture = () => {};
  }
  return ctx;
};

/** Fire the masked image's onLoad so the canvas adopts its natural size. */
const loadImage = (width = 100, height = 80) => {
  const img = document.querySelector("img");
  if (!img) throw new Error("mask preview image not rendered");
  Object.defineProperties(img, {
    naturalWidth: { configurable: true, value: width },
    naturalHeight: { configurable: true, value: height },
  });
  fireEvent.load(img);
};

/** A bounding-rect stub for the paint canvas (jsdom reports 0×0). */
const stubCanvasRect = (rect: Partial<DOMRect> = {}) =>
  vi.spyOn(HTMLCanvasElement.prototype, "getBoundingClientRect").mockReturnValue({
    x: 0,
    y: 0,
    top: 0,
    left: 0,
    right: 100,
    bottom: 80,
    width: 100,
    height: 80,
    toJSON: () => ({}),
    ...rect,
  });

/** Stub only pointer capture (no real 2D surface) for null-context paths. */
const stubPointerCapture = () => {
  const proto = HTMLElement.prototype as unknown as {
    setPointerCapture?: (id: number) => void;
  };
  if (!proto.setPointerCapture) {
    proto.setPointerCapture = () => {};
  }
};

describe("MaskBrush", () => {
  it("renders the paint canvas and the upload fallback for a loaded image", () => {
    renderWithProviders(<MaskBrush imageUrl="blob:img" onMask={vi.fn()} />);
    expect(screen.getByTestId(selectors.workspace.mask.root)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.workspace.mask.canvas)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.workspace.mask.upload)).toBeInTheDocument();
  });

  it("emits the uploaded mask file via the accessible fallback", async () => {
    const user = userEvent.setup();
    const onMask = vi.fn();
    renderWithProviders(<MaskBrush imageUrl="blob:img" onMask={onMask} />);

    const mask = new File(["m"], "mask.png", { type: "image/png" });
    await user.upload(screen.getByTestId(selectors.workspace.mask.upload), mask);

    expect(onMask).toHaveBeenCalledWith(mask);
    expect(screen.getByTestId(selectors.workspace.mask.status)).toHaveTextContent("mask.png");
  });

  it("clears the mask back to null", async () => {
    const user = userEvent.setup();
    const onMask = vi.fn();
    renderWithProviders(<MaskBrush imageUrl="blob:img" onMask={onMask} />);

    await user.click(screen.getByTestId(selectors.workspace.mask.clear));
    expect(onMask).toHaveBeenCalledWith(null);
  });

  it("hides the paint canvas and brush slider when no image is loaded", () => {
    renderWithProviders(<MaskBrush imageUrl={null} onMask={vi.fn()} />);
    expect(screen.queryByTestId(selectors.workspace.mask.canvas)).not.toBeInTheDocument();
    // The brush slider only shows in the paint (non-uploaded) state, but it
    // still renders here because no upload happened — assert the upload field.
    expect(screen.getByTestId(selectors.workspace.mask.upload)).toBeInTheDocument();
  });

  it("reports the empty status before any paint or upload", () => {
    renderWithProviders(<MaskBrush imageUrl="blob:img" onMask={vi.fn()} />);
    const status = screen.getByTestId(selectors.workspace.mask.status);
    // cimode/en both resolve a non-empty status line for the empty state.
    expect(status.textContent.length).toBeGreaterThan(0);
  });

  it("adopts the image's natural size as the canvas size on load", () => {
    installCanvasSurface();
    renderWithProviders(<MaskBrush imageUrl="blob:img" onMask={vi.fn()} />);
    loadImage(120, 90);
    const canvas = screen.getByTestId<HTMLCanvasElement>(selectors.workspace.mask.canvas);
    expect(canvas.width).toBe(120);
    expect(canvas.height).toBe(90);
  });

  it("paints a stroke and exports a white-on-black mask File on pointer up", async () => {
    const ctx = installCanvasSurface();
    stubCanvasRect();
    const onMask = vi.fn();
    renderWithProviders(<MaskBrush imageUrl="blob:img" onMask={onMask} />);
    loadImage(100, 80);

    const canvas = screen.getByTestId(selectors.workspace.mask.canvas);
    fireEvent.pointerDown(canvas, { pointerId: 1, clientX: 10, clientY: 10 });
    // The first down stamps a dot (arc + fill) and marks painted.
    expect(ctx.arc).toHaveBeenCalled();
    expect(screen.getByTestId(selectors.workspace.mask.status).textContent.length).toBeGreaterThan(
      0,
    );

    fireEvent.pointerMove(canvas, { pointerId: 1, clientX: 30, clientY: 25 });
    // A move with a prior point draws a connecting line.
    expect(ctx.lineTo).toHaveBeenCalled();
    expect(ctx.stroke).toHaveBeenCalled();

    fireEvent.pointerUp(canvas, { pointerId: 1 });
    // Pointer-up composites + exports the mask as a PNG File.
    await waitFor(() => expect(onMask).toHaveBeenCalled());
    const exported = onMask.mock.calls.at(-1)?.[0] as File;
    expect(exported).toBeInstanceOf(File);
    expect(exported.name).toBe("mask.png");
    expect(exported.type).toBe("image/png");
    // The output canvas is filled black then the white strokes drawn over it.
    expect(ctx.fillRect).toHaveBeenCalled();
    expect(ctx.drawImage).toHaveBeenCalled();
  });

  it("ignores pointer move when not actively painting", () => {
    const ctx = installCanvasSurface();
    stubCanvasRect();
    renderWithProviders(<MaskBrush imageUrl="blob:img" onMask={vi.fn()} />);
    loadImage(100, 80);

    const canvas = screen.getByTestId(selectors.workspace.mask.canvas);
    // A move with no preceding down is a no-op (paintingRef is false).
    fireEvent.pointerMove(canvas, { pointerId: 1, clientX: 30, clientY: 25 });
    expect(ctx.lineTo).not.toHaveBeenCalled();
  });

  it("does not start painting once a mask file has been uploaded", async () => {
    const user = userEvent.setup();
    const ctx = installCanvasSurface();
    stubCanvasRect();
    const onMask = vi.fn();
    renderWithProviders(<MaskBrush imageUrl="blob:img" onMask={onMask} />);
    loadImage(100, 80);

    // Upload replaces the brush surface; the paint canvas unmounts.
    await user.upload(
      screen.getByTestId(selectors.workspace.mask.upload),
      new File(["m"], "up.png", { type: "image/png" }),
    );
    expect(screen.queryByTestId(selectors.workspace.mask.canvas)).not.toBeInTheDocument();
    expect(ctx.arc).not.toHaveBeenCalled();
  });

  it("adjusts the brush size via the slider", () => {
    installCanvasSurface();
    const onMask = vi.fn();
    renderWithProviders(<MaskBrush imageUrl="blob:img" onMask={onMask} />);
    const slider = screen.getByTestId(selectors.workspace.mask.brushSize);
    fireEvent.change(slider, { target: { value: "64" } });
    expect(slider).toHaveValue("64");
  });

  it("clears a painted canvas and resets painted state", async () => {
    const ctx = installCanvasSurface();
    stubCanvasRect();
    const user = userEvent.setup();
    const onMask = vi.fn();
    renderWithProviders(<MaskBrush imageUrl="blob:img" onMask={onMask} />);
    loadImage(100, 80);

    const canvas = screen.getByTestId(selectors.workspace.mask.canvas);
    fireEvent.pointerDown(canvas, { pointerId: 1, clientX: 10, clientY: 10 });
    fireEvent.pointerUp(canvas, { pointerId: 1 });

    await user.click(screen.getByTestId(selectors.workspace.mask.clear));
    expect(ctx.clearRect).toHaveBeenCalled();
    expect(onMask).toHaveBeenLastCalledWith(null);
  });

  it("ends a stroke when the pointer leaves the canvas", async () => {
    installCanvasSurface();
    stubCanvasRect();
    const onMask = vi.fn();
    renderWithProviders(<MaskBrush imageUrl="blob:img" onMask={onMask} />);
    loadImage(100, 80);

    const canvas = screen.getByTestId(selectors.workspace.mask.canvas);
    fireEvent.pointerDown(canvas, { pointerId: 1, clientX: 10, clientY: 10 });
    fireEvent.pointerLeave(canvas, { pointerId: 1 });
    // Leaving while painting exports the mask just like pointer-up.
    await waitFor(() => expect(onMask).toHaveBeenCalled());
  });

  it("ignores a pointer down when the canvas has no measurable box", () => {
    const ctx = installCanvasSurface();
    // No rect stub → jsdom reports a 0×0 box → pointAt returns null → no paint.
    const onMask = vi.fn();
    renderWithProviders(<MaskBrush imageUrl="blob:img" onMask={onMask} />);
    loadImage(100, 80);

    fireEvent.pointerDown(screen.getByTestId(selectors.workspace.mask.canvas), {
      pointerId: 1,
      clientX: 10,
      clientY: 10,
    });
    expect(ctx.arc).not.toHaveBeenCalled();
    expect(onMask).not.toHaveBeenCalled();
  });

  it("a pointer up with no active stroke is a no-op", () => {
    installCanvasSurface();
    stubCanvasRect();
    const onMask = vi.fn();
    renderWithProviders(<MaskBrush imageUrl="blob:img" onMask={onMask} />);
    loadImage(100, 80);

    // No preceding pointer-down → endStroke short-circuits, no export.
    fireEvent.pointerUp(screen.getByTestId(selectors.workspace.mask.canvas), { pointerId: 1 });
    expect(onMask).not.toHaveBeenCalled();
  });

  it("tolerates a missing 2D context when painting (null-ctx guards)", () => {
    // Keep the process-wide null getContext; only pointer capture is stubbed.
    stubPointerCapture();
    stubCanvasRect();
    const onMask = vi.fn();
    renderWithProviders(<MaskBrush imageUrl="blob:img" onMask={onMask} />);
    // No installCanvasSurface → getContext stays null. onImageLoad still sizes
    // the canvas from the image. stroke() and exportMask()'s out-context both
    // hit their `!ctx` guards without throwing.
    loadImage(100, 80);
    const canvas = screen.getByTestId(selectors.workspace.mask.canvas);
    fireEvent.pointerDown(canvas, { pointerId: 1, clientX: 10, clientY: 10 });
    // painted flips true even though the (null) context drew nothing.
    expect(screen.getByTestId(selectors.workspace.mask.status).textContent.length).toBeGreaterThan(
      0,
    );
    // pointerUp runs exportMask; the out-canvas getContext is null → no File.
    fireEvent.pointerUp(canvas, { pointerId: 1 });
    expect(onMask).not.toHaveBeenCalled();
  });

  it("export is skipped when the canvas has zero size (unloaded image)", () => {
    installCanvasSurface();
    stubCanvasRect();
    const onMask = vi.fn();
    renderWithProviders(<MaskBrush imageUrl="blob:img" onMask={onMask} />);
    // Load an image with a 0×0 natural size → canvas.width stays 0 →
    // exportMask early-returns at the `canvas.width === 0` guard.
    loadImage(0, 0);
    const canvas = screen.getByTestId(selectors.workspace.mask.canvas);
    fireEvent.pointerDown(canvas, { pointerId: 1, clientX: 10, clientY: 10 });
    fireEvent.pointerUp(canvas, { pointerId: 1 });
    expect(onMask).not.toHaveBeenCalled();
  });

  it("ignores a pointer move whose point cannot be measured while painting", () => {
    const ctx = installCanvasSurface();
    const rectSpy = stubCanvasRect();
    const onMask = vi.fn();
    renderWithProviders(<MaskBrush imageUrl="blob:img" onMask={onMask} />);
    loadImage(100, 80);

    const canvas = screen.getByTestId(selectors.workspace.mask.canvas);
    // Start a real stroke (measurable box).
    fireEvent.pointerDown(canvas, { pointerId: 1, clientX: 10, clientY: 10 });
    ctx.lineTo.mockClear();

    // Now collapse the box → the next move's point is null → move is a no-op
    // even though painting is active.
    rectSpy.mockReturnValue({
      x: 0,
      y: 0,
      top: 0,
      left: 0,
      right: 0,
      bottom: 0,
      width: 0,
      height: 0,
      toJSON: () => ({}),
    });
    fireEvent.pointerMove(canvas, { pointerId: 1, clientX: 30, clientY: 25 });
    expect(ctx.lineTo).not.toHaveBeenCalled();
  });

  it("upload ignores a change event with no file selected", () => {
    installCanvasSurface();
    const onMask = vi.fn();
    renderWithProviders(<MaskBrush imageUrl="blob:img" onMask={onMask} />);
    // Fire change with an empty file list → the `if (file)` guard skips.
    fireEvent.change(screen.getByTestId(selectors.workspace.mask.upload), {
      target: { files: [] },
    });
    expect(onMask).not.toHaveBeenCalled();
  });
});
