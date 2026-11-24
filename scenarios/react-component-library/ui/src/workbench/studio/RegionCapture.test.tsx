import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { setLocale } from "../../i18n";
import { renderWithProviders } from "../../test-utils";
import { RegionCapture } from "./RegionCapture";

const asset = { id: "reference-1", name: "hero.png", url: "/assets/hero.png" };

function renderCapture(onImport = vi.fn(), onClose = vi.fn()) {
  renderWithProviders(<RegionCapture asset={asset} onImport={onImport} onClose={onClose} />);
  const image = screen.getByTestId("region-capture-image");
  Object.defineProperties(image, {
    naturalWidth: { configurable: true, value: 800 },
    naturalHeight: { configurable: true, value: 600 },
  });
  fireEvent.load(image);
  return { image, onImport, onClose };
}

beforeEach(async () => {
  await setLocale("en");
});

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

describe("RegionCapture image validation", () => {
  it("initializes a cached image whose load event completed before the listener attached", () => {
    vi.spyOn(HTMLImageElement.prototype, "complete", "get").mockReturnValue(true);
    vi.spyOn(HTMLImageElement.prototype, "naturalWidth", "get").mockReturnValue(800);
    vi.spyOn(HTMLImageElement.prototype, "naturalHeight", "get").mockReturnValue(600);

    renderWithProviders(<RegionCapture asset={asset} onImport={vi.fn()} onClose={vi.fn()} />);

    expect(screen.getByTestId("region-capture-marquee")).toHaveAttribute(
      "aria-valuetext",
      "x 0, y 0, 800 by 600 pixels",
    );
  });

  it("explains an unreadable reference image and disables capture controls", () => {
    renderWithProviders(<RegionCapture asset={asset} onImport={vi.fn()} onClose={vi.fn()} />);

    fireEvent.error(screen.getByTestId("region-capture-image"));

    expect(screen.getByRole("alert")).toHaveTextContent(
      "This image could not be loaded for cropping.",
    );
    expect(screen.queryByTestId("region-capture-marquee")).toBeNull();
    expect(screen.queryByText(/Loading image/i)).toBeNull();
    expect(screen.getByTestId("region-capture-confirm")).toBeDisabled();
    expect(screen.getByTestId("region-capture-w")).toBeDisabled();
    expect(screen.getByTestId("region-capture-aspect-1:1")).toBeDisabled();
  });

  it("rejects an image load with zero natural dimensions", () => {
    renderWithProviders(<RegionCapture asset={asset} onImport={vi.fn()} onClose={vi.fn()} />);
    const image = screen.getByTestId("region-capture-image");
    Object.defineProperties(image, {
      naturalWidth: { configurable: true, value: 0 },
      naturalHeight: { configurable: true, value: 600 },
    });

    fireEvent.load(image);

    expect(screen.getByTestId("region-capture-image-error")).toBeInTheDocument();
    expect(screen.queryByTestId("region-capture-marquee")).toBeNull();
  });

  it("restores the crop controls when an unreadable image later loads", () => {
    renderWithProviders(<RegionCapture asset={asset} onImport={vi.fn()} onClose={vi.fn()} />);
    const image = screen.getByTestId("region-capture-image");
    fireEvent.error(image);

    Object.defineProperties(image, {
      naturalWidth: { configurable: true, value: 800 },
      naturalHeight: { configurable: true, value: 600 },
    });
    fireEvent.load(image);

    expect(screen.queryByTestId("region-capture-image-error")).toBeNull();
    expect(screen.getByTestId("region-capture-marquee")).toBeInTheDocument();
    expect(screen.getByTestId("region-capture-confirm")).toBeEnabled();
  });

  it("reports a missing canvas context, then allows a successful retry", async () => {
    const onImport = vi.fn();
    const onClose = vi.fn();
    renderCapture(onImport, onClose);
    const contextSpy = vi.spyOn(HTMLCanvasElement.prototype, "getContext");
    contextSpy.mockReturnValueOnce(null);

    fireEvent.click(screen.getByTestId("region-capture-confirm"));

    await waitFor(() => expect(screen.getByTestId("region-capture-error")).toBeInTheDocument());
    expect(screen.getByTestId("region-capture-confirm")).toBeEnabled();
    expect(onImport).not.toHaveBeenCalled();

    contextSpy.mockReturnValue({ drawImage: vi.fn() } as unknown as CanvasRenderingContext2D);
    vi.spyOn(HTMLCanvasElement.prototype, "toBlob").mockImplementation((callback) =>
      callback(new Blob(["png"], { type: "image/png" })),
    );
    fireEvent.click(screen.getByTestId("region-capture-confirm"));

    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1));
    expect(onImport).toHaveBeenCalledWith(expect.any(File), "captured");
  });

  it("recovers from a synchronous PNG encoder exception", async () => {
    const onImport = vi.fn();
    const onClose = vi.fn();
    renderCapture(onImport, onClose);
    vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue({
      drawImage: vi.fn(),
    } as unknown as CanvasRenderingContext2D);
    const toBlobSpy = vi.spyOn(HTMLCanvasElement.prototype, "toBlob").mockImplementation(() => {
      // Exercise the defensive conversion of arbitrary thrown values.
      // eslint-disable-next-line @typescript-eslint/only-throw-error
      throw "encoder unavailable";
    });

    fireEvent.click(screen.getByTestId("region-capture-confirm"));

    await waitFor(() => expect(screen.getByTestId("region-capture-error")).toBeInTheDocument());
    expect(screen.getByTestId("region-capture-confirm")).toBeEnabled();
    expect(onImport).not.toHaveBeenCalled();
    expect(onClose).not.toHaveBeenCalled();
    toBlobSpy.mockRestore();
  });

  it("reports an encoder that returns no PNG blob", async () => {
    const onImport = vi.fn();
    const onClose = vi.fn();
    renderCapture(onImport, onClose);
    vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue({
      drawImage: vi.fn(),
    } as unknown as CanvasRenderingContext2D);
    vi.spyOn(HTMLCanvasElement.prototype, "toBlob").mockImplementation((callback) =>
      callback(null),
    );

    fireEvent.click(screen.getByTestId("region-capture-confirm"));

    await waitFor(() => expect(screen.getByTestId("region-capture-error")).toBeInTheDocument());
    expect(onImport).not.toHaveBeenCalled();
    expect(onClose).not.toHaveBeenCalled();
  });

  it("recovers when the browser rejects drawing the selected image", async () => {
    const onImport = vi.fn();
    const onClose = vi.fn();
    renderCapture(onImport, onClose);
    vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue({
      drawImage: vi.fn(() => {
        throw new Error("image could not be drawn");
      }),
    } as unknown as CanvasRenderingContext2D);

    fireEvent.click(screen.getByTestId("region-capture-confirm"));

    await waitFor(() => expect(screen.getByTestId("region-capture-error")).toBeInTheDocument());
    expect(screen.getByTestId("region-capture-confirm")).toBeEnabled();
    expect(onImport).not.toHaveBeenCalled();
    expect(onClose).not.toHaveBeenCalled();
  });
});

describe("RegionCapture selection controls", () => {
  it("moves and resizes the selection with keyboard controls and aspect presets", () => {
    renderCapture();
    const marquee = screen.getByTestId("region-capture-marquee");

    fireEvent.keyDown(marquee, { key: "ArrowRight" });
    expect(marquee).toHaveAttribute("aria-valuetext", "x 0, y 0, 800 by 600 pixels");
    fireEvent.keyDown(marquee, { key: "ArrowDown", shiftKey: true });
    expect(marquee).toHaveAttribute("aria-valuetext", "x 0, y 0, 800 by 600 pixels");
    fireEvent.keyDown(marquee, { key: "Escape" });
    expect(marquee).toHaveAttribute("aria-valuetext", "x 0, y 0, 800 by 600 pixels");

    fireEvent.click(screen.getByTestId("region-capture-aspect-1:1"));
    expect(screen.getByTestId("region-capture-aspect-1:1")).toHaveAttribute("aria-pressed", "true");
    expect(marquee).toHaveAttribute("aria-valuetext", "x 100, y 0, 600 by 600 pixels");
    fireEvent.click(screen.getByTestId("region-capture-aspect-free"));
    expect(screen.getByTestId("region-capture-aspect-free")).toHaveAttribute(
      "aria-pressed",
      "true",
    );
  });

  it("applies bounded numeric edits and commits invalid drafts without changing the region", () => {
    renderCapture();
    const marquee = screen.getByTestId("region-capture-marquee");
    const xField = screen.getByTestId("region-capture-x");
    const widthField = screen.getByTestId("region-capture-w");

    fireEvent.change(xField, { target: { value: "900" } });
    expect(marquee).toHaveAttribute("aria-valuetext", "x 0, y 0, 800 by 600 pixels");
    expect(xField).toHaveValue(900);
    fireEvent.blur(xField);
    expect(xField).toHaveValue(0);

    fireEvent.change(widthField, { target: { value: "" } });
    expect(widthField).toHaveValue(null);
    fireEvent.blur(widthField);
    expect(widthField).toHaveValue(800);

    fireEvent.change(widthField, { target: { value: "200" } });
    fireEvent.keyDown(widthField, { key: "Enter" });
    expect(widthField).toHaveValue(200);
    expect(marquee).toHaveAttribute("aria-valuetext", "x 0, y 0, 200 by 600 pixels");

    fireEvent.change(screen.getByTestId("region-capture-h"), { target: { value: "100" } });
    fireEvent.change(screen.getByTestId("region-capture-y"), { target: { value: "40" } });
    expect(marquee).toHaveAttribute("aria-valuetext", "x 0, y 40, 200 by 100 pixels");
  });

  it("recovers when saving the captured image fails", async () => {
    const onImport = vi.fn().mockRejectedValueOnce(new Error("storage unavailable"));
    const onClose = vi.fn();
    renderCapture(onImport, onClose);
    vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue({
      drawImage: vi.fn(),
    } as unknown as CanvasRenderingContext2D);
    vi.spyOn(HTMLCanvasElement.prototype, "toBlob").mockImplementation((callback) =>
      callback(new Blob(["png"], { type: "image/png" })),
    );

    fireEvent.click(screen.getByTestId("region-capture-confirm"));

    await waitFor(() => expect(screen.getByTestId("region-capture-error")).toBeInTheDocument());
    expect(screen.getByTestId("region-capture-confirm")).toBeEnabled();
    expect(onImport).toHaveBeenCalledWith(expect.any(File), "captured");
    expect(onClose).not.toHaveBeenCalled();
  });

  it("creates and moves a selection through pointer gestures", () => {
    renderCapture();
    const stage = screen.getByTestId("region-capture-stage");
    vi.spyOn(HTMLElement.prototype, "getBoundingClientRect").mockReturnValue({
      left: 0,
      top: 0,
      right: 800,
      bottom: 600,
      width: 800,
      height: 600,
      x: 0,
      y: 0,
      toJSON: () => ({}),
    });
    stage.setPointerCapture = vi.fn();
    stage.hasPointerCapture = vi.fn(() => true);
    const releaseCapture = vi.fn();
    stage.releasePointerCapture = releaseCapture;

    fireEvent.pointerDown(stage, { pointerId: 7, button: 0, clientX: 100, clientY: 80 });
    fireEvent.pointerMove(stage, { pointerId: 7, clientX: 300, clientY: 240 });
    const marquee = screen.getByTestId("region-capture-marquee");
    expect(marquee).toHaveAttribute("aria-valuetext", "x 100, y 80, 200 by 160 pixels");
    fireEvent.pointerUp(stage, { pointerId: 7 });
    expect(releaseCapture).toHaveBeenCalledWith(7);

    fireEvent.pointerDown(marquee, { pointerId: 8, button: 0, clientX: 150, clientY: 120 });
    fireEvent.pointerMove(stage, { pointerId: 8, clientX: 170, clientY: 130 });
    expect(screen.getByTestId("region-capture-marquee")).toHaveAttribute(
      "aria-valuetext",
      "x 120, y 90, 200 by 160 pixels",
    );
    fireEvent.pointerCancel(stage, { pointerId: 8 });

    fireEvent.click(screen.getByTestId("region-capture-aspect-16:9"));
    const resizedMarquee = screen.getByTestId("region-capture-marquee");
    fireEvent.pointerDown(screen.getByTestId("region-capture-handle-se"), {
      pointerId: 9,
      button: 0,
      clientX: 320,
      clientY: 250,
    });
    fireEvent.pointerMove(stage, { pointerId: 9, clientX: 360, clientY: 280 });
    expect(resizedMarquee).toHaveAttribute("aria-valuetext", expect.stringContaining(" by "));
    fireEvent.lostPointerCapture(stage, { pointerId: 9 });
    fireEvent.pointerMove(stage, { pointerId: 9, clientX: 400, clientY: 300 });
    expect(screen.getByTestId("region-capture-marquee")).toBe(resizedMarquee);
  });

  it("restarts a full-image selection and ignores secondary or unrelated pointer input", () => {
    renderCapture();
    const stage = screen.getByTestId("region-capture-stage");
    const image = screen.getByTestId("region-capture-image");
    vi.spyOn(HTMLElement.prototype, "getBoundingClientRect").mockReturnValue({
      left: 0,
      top: 0,
      right: 800,
      bottom: 600,
      width: 800,
      height: 600,
      x: 0,
      y: 0,
      toJSON: () => ({}),
    });
    stage.setPointerCapture = vi.fn();
    stage.hasPointerCapture = vi.fn(() => false);
    const releaseCapture = vi.fn();
    stage.releasePointerCapture = releaseCapture;

    fireEvent.pointerDown(stage, { pointerId: 20, button: 2, clientX: 40, clientY: 30 });
    fireEvent.pointerMove(stage, { pointerId: 21, clientX: 200, clientY: 150 });
    expect(screen.getByTestId("region-capture-marquee")).toHaveAttribute(
      "aria-valuetext",
      "x 0, y 0, 800 by 600 pixels",
    );

    fireEvent.pointerDown(screen.getByTestId("region-capture-marquee"), {
      pointerId: 22,
      button: 0,
      clientX: 100,
      clientY: 100,
    });
    fireEvent.pointerMove(stage, { pointerId: 22, clientX: 250, clientY: 220 });
    expect(screen.getByTestId("region-capture-marquee")).toHaveAttribute(
      "aria-valuetext",
      "x 100, y 100, 150 by 120 pixels",
    );

    fireEvent.pointerUp(stage, { pointerId: 22 });
    expect(releaseCapture).not.toHaveBeenCalled();
    // A zero-size rendered image has a defined origin, so gestures remain bounded.
    vi.spyOn(image, "getBoundingClientRect").mockReturnValue({
      left: 0,
      top: 0,
      right: 0,
      bottom: 0,
      width: 0,
      height: 0,
      x: 0,
      y: 0,
      toJSON: () => ({}),
    });
    fireEvent.pointerDown(stage, { pointerId: 23, button: 0, clientX: 10, clientY: 10 });
    fireEvent.pointerMove(stage, { pointerId: 23, clientX: 20, clientY: 20 });
    expect(screen.getByTestId("region-capture-marquee")).toHaveAttribute(
      "aria-valuetext",
      "x 0, y 0, 16 by 16 pixels",
    );
  });

  it("moves the selection toward the image origin with the left and up keys", () => {
    renderCapture();
    const marquee = screen.getByTestId("region-capture-marquee");

    fireEvent.keyDown(marquee, { key: "ArrowLeft" });
    fireEvent.keyDown(marquee, { key: "ArrowUp" });

    expect(marquee).toHaveAttribute("aria-valuetext", "x 0, y 0, 800 by 600 pixels");
  });
});
