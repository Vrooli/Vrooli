import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import type { DesignAsset } from "../designs";
import { renderWithProviders } from "../../test-utils";
import { StudioAssets } from "./StudioAssets";

const drawImage = vi.fn();
let restoreToBlob: () => void = () => undefined;

function makeAsset(overrides: Partial<DesignAsset> & { id: string }): DesignAsset {
  return {
    designId: "design-1",
    name: "hero.png",
    mime: "image/png",
    bytes: 4096,
    width: 800,
    height: 600,
    fidelity: "reference",
    addedAt: "2026-11-19T12:00:00.000Z",
    url: "/api/designs/design-1/assets/asset-1",
    ...overrides,
  };
}

function renderAssets(assets: DesignAsset[], onImport = vi.fn()) {
  renderWithProviders(
    <StudioAssets
      assets={assets}
      assetUrl={(asset) => asset.url}
      onImport={onImport}
      onRemove={() => undefined}
      onPlace={() => undefined}
    />,
  );
  return { onImport };
}

function openCrop(assetId: string): void {
  fireEvent.click(screen.getByTestId(`studio-asset-crop-${assetId}`));
}

function loadImage(width = 800, height = 600): HTMLImageElement {
  const image = screen.getByTestId("region-capture-image");
  expect(image).toBeInstanceOf(HTMLImageElement);
  const element = image as HTMLImageElement;
  Object.defineProperty(element, "naturalWidth", { configurable: true, value: width });
  Object.defineProperty(element, "naturalHeight", { configurable: true, value: height });
  fireEvent.load(element);
  return element;
}

function fieldInput(key: "x" | "y" | "w" | "h"): HTMLInputElement {
  return screen.getByTestId<HTMLInputElement>(`region-capture-${key}`);
}

function setField(key: "x" | "y" | "w" | "h", value: string): void {
  fireEvent.change(fieldInput(key), { target: { value } });
  fireEvent.blur(fieldInput(key));
}

beforeEach(() => {
  drawImage.mockReset();
  vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue({
    drawImage,
  } as unknown as CanvasRenderingContext2D);
  if (typeof HTMLCanvasElement.prototype.toBlob !== "function") {
    Object.defineProperty(HTMLCanvasElement.prototype, "toBlob", {
      configurable: true,
      writable: true,
      value: () => undefined,
    });
  }
  const toBlob = vi
    .spyOn(HTMLCanvasElement.prototype, "toBlob")
    .mockImplementation((callback: BlobCallback) =>
      callback(new Blob(["png"], { type: "image/png" })),
    );
  restoreToBlob = () => toBlob.mockRestore();
});

afterEach(() => {
  restoreToBlob();
  vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue(null);
});

describe("StudioAssets fidelity", () => {
  it("distinguishes reference and captured assets with accessible explanations", () => {
    const reference = makeAsset({ id: "asset-ref" });
    const captured = makeAsset({
      id: "asset-cap",
      name: "capture-hero.png",
      fidelity: "captured",
    });
    renderAssets([reference, captured]);

    expect(screen.getByTestId("studio-asset-fidelity-asset-ref")).toHaveTextContent(
      "workbench.studio.reference",
    );
    expect(screen.getByTestId("studio-asset-fidelity-asset-ref")).toHaveAttribute(
      "title",
      "workbench.studio.fidelityReferenceTitle",
    );
    expect(screen.getByTestId("studio-asset-fidelity-asset-ref")).toHaveAttribute(
      "aria-label",
      "workbench.studio.fidelityReferenceTitle",
    );

    expect(screen.getByTestId("studio-asset-fidelity-asset-cap")).toHaveTextContent(
      "workbench.studio.fidelityCaptured",
    );
    expect(screen.getByTestId("studio-asset-fidelity-asset-cap")).toHaveAttribute(
      "title",
      "workbench.studio.fidelityCapturedTitle",
    );
  });

  it("passes uploaded files through as references", () => {
    const { onImport } = renderAssets([makeAsset({ id: "asset-ref" })]);
    const input = document.querySelector<HTMLInputElement>(".studio-file-input");
    expect(input).not.toBeNull();
    if (!input) return;
    const file = new File(["pixels"], "reference.png", { type: "image/png" });

    fireEvent.change(input, { target: { files: [file] } });

    expect(onImport).toHaveBeenCalledWith(file, "reference");
  });
});

describe("RegionCapture dialog", () => {
  it("opens as a labelled modal and closes on Escape", () => {
    renderAssets([makeAsset({ id: "asset-ref" })]);
    openCrop("asset-ref");

    const dialog = screen.getByTestId("region-capture");
    expect(dialog).toHaveAttribute("aria-modal", "true");
    expect(screen.getByRole("dialog")).toBe(dialog);
    const titleId = dialog.getAttribute("aria-labelledby");
    expect(titleId).toBeTruthy();
    if (titleId) {
      expect(document.getElementById(titleId)?.textContent).toBe(
        "workbench.studio.cropRegionTitle",
      );
    }
    expect(dialog.contains(document.activeElement)).toBe(true);

    fireEvent.keyDown(document, { key: "Escape" });
    expect(screen.queryByTestId("region-capture")).toBeNull();
  });

  it("closes when the backdrop is pressed", () => {
    renderAssets([makeAsset({ id: "asset-ref" })]);
    openCrop("asset-ref");
    const backdrop = document.querySelector(".rc-backdrop");
    expect(backdrop).not.toBeNull();
    if (backdrop) fireEvent.pointerDown(backdrop);
    expect(screen.queryByTestId("region-capture")).toBeNull();
  });

  it("describes the image and exposes the marquee's position and size", () => {
    renderAssets([makeAsset({ id: "asset-ref" })]);
    openCrop("asset-ref");
    const image = loadImage();

    expect(image).toHaveAttribute("alt", "workbench.studio.cropImageAlt");
    const marquee = screen.getByTestId("region-capture-marquee");
    expect(marquee).toHaveAttribute("role", "slider");
    expect(marquee).toHaveAttribute("aria-label", "workbench.studio.regionMarquee");
    expect(marquee).toHaveAttribute("aria-valuetext", "workbench.studio.regionValue");
    expect(marquee).toHaveAttribute("tabindex", "0");
    expect(fieldInput("w")).toHaveValue(800);
    expect(fieldInput("h")).toHaveValue(600);
  });

  it("turns a pointer drag into a natural-pixel region", () => {
    renderAssets([makeAsset({ id: "asset-ref" })]);
    openCrop("asset-ref");
    const image = loadImage();
    const stage = screen.getByTestId("region-capture-stage");
    const rect = new DOMRect(0, 0, 400, 300);
    vi.spyOn(stage, "getBoundingClientRect").mockReturnValue(rect);
    vi.spyOn(image, "getBoundingClientRect").mockReturnValue(rect);

    fireEvent.pointerDown(stage, { clientX: 50, clientY: 50, pointerId: 1, button: 0 });
    fireEvent.pointerMove(stage, { clientX: 250, clientY: 200, pointerId: 1 });
    fireEvent.pointerUp(stage, { clientX: 250, clientY: 200, pointerId: 1 });

    expect(fieldInput("x")).toHaveValue(100);
    expect(fieldInput("y")).toHaveValue(100);
    expect(fieldInput("w")).toHaveValue(400);
    expect(fieldInput("h")).toHaveValue(300);
  });

  it("moves with arrows and resizes with Shift + arrows", () => {
    renderAssets([makeAsset({ id: "asset-ref" })]);
    openCrop("asset-ref");
    loadImage();
    setField("w", "400");
    setField("h", "300");
    const marquee = screen.getByTestId("region-capture-marquee");
    marquee.focus();

    fireEvent.keyDown(marquee, { key: "ArrowRight" });
    expect(fieldInput("x")).toHaveValue(2);
    fireEvent.keyDown(marquee, { key: "ArrowDown", shiftKey: true });
    expect(fieldInput("h")).toHaveValue(302);
  });

  it("applies the 1:1 aspect preset", () => {
    renderAssets([makeAsset({ id: "asset-ref" })]);
    openCrop("asset-ref");
    loadImage();

    fireEvent.click(screen.getByTestId("region-capture-aspect-1:1"));

    expect(screen.getByTestId("region-capture-aspect-1:1")).toHaveAttribute(
      "aria-pressed",
      "true",
    );
    expect(fieldInput("w")).toHaveValue(600);
    expect(fieldInput("h")).toHaveValue(600);
  });

  it("renders exactly the selected rectangle and imports it as captured", async () => {
    let finishBlob: ((blob: Blob | null) => void) | null = null;
    vi.spyOn(HTMLCanvasElement.prototype, "toBlob").mockImplementation((callback: BlobCallback) => {
      finishBlob = (blob) => callback(blob);
    });
    let finishImport: () => void = () => undefined;
    const onImport = vi.fn<(file: File, fidelity: string) => Promise<void>>(
      () =>
        new Promise<void>((resolve) => {
          finishImport = resolve;
        }),
    );
    renderAssets([makeAsset({ id: "asset-ref", name: "hero.png" })], onImport);
    openCrop("asset-ref");
    const image = loadImage();
    setField("w", "400");
    setField("h", "300");
    setField("x", "100");
    setField("y", "50");

    fireEvent.click(screen.getByTestId("region-capture-confirm"));

    await waitFor(() =>
      expect(screen.getByTestId("region-capture-confirm")).toHaveTextContent(
        "workbench.studio.regionCapturing",
      ),
    );
    expect(screen.getByTestId("region-capture-confirm")).toBeDisabled();

    act(() => finishBlob?.(new Blob(["png"], { type: "image/png" })));

    await waitFor(() => expect(onImport).toHaveBeenCalledTimes(1));
    expect(screen.getByTestId("region-capture-confirm")).toHaveTextContent(
      "workbench.studio.regionSaving",
    );
    const call = onImport.mock.calls[0];
    expect(call).toBeDefined();
    if (!call) return;
    const [file, fidelity] = call;
    expect(file).toBeInstanceOf(File);
    expect(file.name).toBe("capture-hero.png");
    expect(file.type).toBe("image/png");
    expect(fidelity).toBe("captured");
    expect(drawImage).toHaveBeenCalledWith(image, 100, 50, 400, 300, 0, 0, 400, 300);

    act(() => finishImport());

    await waitFor(() => expect(screen.queryByTestId("region-capture")).toBeNull());
  });

  it("keeps the dialog open with a retry path when PNG encoding fails", async () => {
    vi.spyOn(HTMLCanvasElement.prototype, "toBlob").mockImplementation((callback: BlobCallback) =>
      callback(null),
    );
    const onImport = vi.fn();
    renderAssets([makeAsset({ id: "asset-ref" })], onImport);
    openCrop("asset-ref");
    loadImage();

    fireEvent.click(screen.getByTestId("region-capture-confirm"));

    await waitFor(() =>
      expect(screen.getByTestId("region-capture-error")).toHaveTextContent(
        "workbench.studio.regionCaptureFailed",
      ),
    );
    expect(onImport).not.toHaveBeenCalled();
    expect(screen.getByTestId("region-capture")).toBeInTheDocument();
    expect(screen.getByTestId("region-capture-confirm")).toBeEnabled();
    expect(screen.getByTestId("region-capture-confirm")).toHaveTextContent(
      "workbench.studio.captureRegion",
    );
  });

  it("keeps the dialog open with an error when the capture upload fails", async () => {
    const onImport = vi.fn(() => Promise.reject(new Error("upload failed")));
    renderAssets([makeAsset({ id: "asset-ref" })], onImport);
    openCrop("asset-ref");
    loadImage();

    fireEvent.click(screen.getByTestId("region-capture-confirm"));

    await waitFor(() =>
      expect(screen.getByTestId("region-capture-error")).toHaveTextContent(
        "workbench.studio.regionCaptureFailed",
      ),
    );
    expect(onImport).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId("region-capture")).toBeInTheDocument();
    expect(screen.getByTestId("region-capture-confirm")).toBeEnabled();
  });
});
