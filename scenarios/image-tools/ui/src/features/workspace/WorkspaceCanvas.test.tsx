/**
 * WorkspaceCanvas tests. The canvas is the work surface: the empty-state upload
 * affordance (click / drop / file-input), the zoom-fit-actual toolbar, the
 * before↔after compare toggle (+ the auto-engage on an async result landing),
 * the live progress overlay, the result-meta line, the metadata read panel, and
 * the crop-overlay wiring driven off the image's natural size. jsdom has no
 * layout, so the image `onLoad` is fired with stubbed natural/client sizes.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import type { RunOpImageResult } from "../../api/ops";
import { WorkspaceCanvas, type WorkspaceCanvasProps } from "./WorkspaceCanvas";

const baseProps: WorkspaceCanvasProps = {
  baseUrl: null,
  previewUrl: null,
  hasSteps: false,
  currentResult: null,
  metadata: null,
  onFile: vi.fn(),
};

const render = (overrides: Partial<WorkspaceCanvasProps> = {}) =>
  renderWithProviders(<WorkspaceCanvas {...baseProps} {...overrides} />);

/** Fire the image's onLoad with stubbed natural/client sizes (jsdom has none). */
const loadImage = (natural = { width: 200, height: 100 }, client = { width: 200, height: 100 }) => {
  const img = screen.getByTestId(selectors.workspace.canvas.image);
  Object.defineProperties(img, {
    naturalWidth: { configurable: true, value: natural.width },
    naturalHeight: { configurable: true, value: natural.height },
    clientWidth: { configurable: true, value: client.width },
    clientHeight: { configurable: true, value: client.height },
  });
  fireEvent.load(img);
  return img;
};

const imageResult: RunOpImageResult = {
  kind: "image",
  url: "blob:result",
  width: 800,
  height: 600,
  format: "png",
  jobId: "job-1",
};

describe("WorkspaceCanvas", () => {
  beforeEach(async () => {
    await setLocale("en");
  });
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  describe("empty state", () => {
    it("renders the dropzone and opens the file picker on click", async () => {
      const user = userEvent.setup();
      render();
      const dropzone = screen.getByTestId(selectors.workspace.dropzone);
      expect(dropzone).toBeInTheDocument();

      // Clicking the dropzone (and the empty-state) clicks the hidden input.
      const input = screen.getByTestId(selectors.workspace.fileInput);
      const click = vi.spyOn(input, "click").mockImplementation(() => {});
      await user.click(dropzone);
      expect(click).toHaveBeenCalled();
    });

    it("accepts a dropped file (onDrop branch)", () => {
      const onFile = vi.fn();
      render({ onFile });
      const file = new File(["x"], "drop.png", { type: "image/png" });
      const dropzone = screen.getByTestId(selectors.workspace.dropzone);

      fireEvent.drop(dropzone, { dataTransfer: { files: [file] } });
      expect(onFile).toHaveBeenCalledWith(file);
    });

    it("ignores a drop with no files (falsy file branch)", () => {
      const onFile = vi.fn();
      render({ onFile });
      fireEvent.drop(screen.getByTestId(selectors.workspace.dropzone), {
        dataTransfer: { files: [] },
      });
      expect(onFile).not.toHaveBeenCalled();
    });

    it("dragover is prevented to allow a drop", () => {
      render();
      const dropzone = screen.getByTestId(selectors.workspace.dropzone);
      const event = new Event("dragover", { bubbles: true, cancelable: true });
      fireEvent(dropzone, event);
      expect(event.defaultPrevented).toBe(true);
    });

    it("emits the selected file via the hidden input change handler", async () => {
      const user = userEvent.setup();
      const onFile = vi.fn();
      render({ onFile });
      const file = new File(["x"], "pick.png", { type: "image/png" });
      await user.upload(screen.getByTestId(selectors.workspace.fileInput), file);
      expect(onFile).toHaveBeenCalledWith(file);
    });

    it("disables the zoom controls and hides Replace when no image is loaded", () => {
      render();
      expect(screen.getByTestId(selectors.workspace.canvas.zoomIn)).toBeDisabled();
      expect(screen.getByTestId(selectors.workspace.canvas.zoomOut)).toBeDisabled();
      expect(screen.getByTestId(selectors.workspace.canvas.zoomActual)).toBeDisabled();
      expect(screen.getByTestId(selectors.workspace.canvas.zoomFit)).toBeDisabled();
      expect(screen.queryByTestId(selectors.workspace.canvas.replace)).not.toBeInTheDocument();
    });
  });

  describe("loaded image", () => {
    it("renders the image and a Replace button that re-opens the picker", async () => {
      const user = userEvent.setup();
      render({ baseUrl: "blob:base" });
      expect(screen.getByTestId(selectors.workspace.canvas.image)).toHaveAttribute(
        "src",
        "blob:base",
      );

      const input = screen.getByTestId(selectors.workspace.fileInput);
      const click = vi.spyOn(input, "click").mockImplementation(() => {});
      await user.click(screen.getByTestId(selectors.workspace.canvas.replace));
      expect(click).toHaveBeenCalled();
    });

    it("prefers the previewUrl over the baseUrl for the image src", () => {
      render({ baseUrl: "blob:base", previewUrl: "blob:preview", hasSteps: true });
      expect(screen.getByTestId(selectors.workspace.canvas.image)).toHaveAttribute(
        "src",
        "blob:preview",
      );
    });

    it("zooms in, out, to actual size, and back to fit", async () => {
      const user = userEvent.setup();
      render({ baseUrl: "blob:base" });
      const level = () => screen.getByText(/%$/).textContent;

      // Fit is the default → 100%.
      expect(level()).toContain("100%");

      await user.click(screen.getByTestId(selectors.workspace.canvas.zoomIn));
      expect(level()).toContain("125%"); // 1 * 1.25

      await user.click(screen.getByTestId(selectors.workspace.canvas.zoomActual));
      expect(level()).toContain("100%");

      await user.click(screen.getByTestId(selectors.workspace.canvas.zoomOut));
      expect(level()).toContain("80%"); // 1 / 1.25

      await user.click(screen.getByTestId(selectors.workspace.canvas.zoomFit));
      expect(level()).toContain("100%");
    });

    it("applies an inline width percentage when not fitting", async () => {
      const user = userEvent.setup();
      render({ baseUrl: "blob:base" });
      await user.click(screen.getByTestId(selectors.workspace.canvas.zoomActual));
      const img = screen.getByTestId(selectors.workspace.canvas.image);
      // actual size → fit off, width style set to 100%.
      expect(img.getAttribute("style")).toContain("width: 100%");
    });

    it("shows the result meta line when a current result exists", () => {
      render({ baseUrl: "blob:base", hasSteps: true, currentResult: imageResult });
      expect(screen.getByTestId(selectors.workspace.canvas.meta)).toBeInTheDocument();
    });

    it("renders the metadata read panel when metadata is present", () => {
      render({ baseUrl: "blob:base", metadata: '{"format":"png"}' });
      const out = screen.getByTestId(selectors.workspace.canvas.metadataOutput);
      expect(out).toHaveTextContent('{"format":"png"}');
    });
  });

  describe("compare", () => {
    it("toggles before/after and reflects aria-pressed", async () => {
      const user = userEvent.setup();
      render({ baseUrl: "blob:base", previewUrl: "blob:preview", hasSteps: true });

      const toggle = screen.getByTestId(selectors.workspace.canvas.compareToggle);
      expect(toggle).toHaveAttribute("aria-pressed", "false");

      await user.click(toggle);
      expect(toggle).toHaveAttribute("aria-pressed", "true");
      // The BeforeAfter compare surface mounts.
      expect(screen.getByTestId(selectors.workspace.compare.root)).toBeInTheDocument();

      await user.click(toggle);
      expect(toggle).toHaveAttribute("aria-pressed", "false");
      expect(screen.queryByTestId(selectors.workspace.compare.root)).not.toBeInTheDocument();
    });

    it("does not render the compare toggle without steps", () => {
      render({ baseUrl: "blob:base" });
      expect(
        screen.queryByTestId(selectors.workspace.canvas.compareToggle),
      ).not.toBeInTheDocument();
    });

    it("auto-engages compare when an async result lands (compareSignal effect)", () => {
      const { rerender } = render({
        baseUrl: "blob:base",
        previewUrl: "blob:preview",
        hasSteps: true,
        compareSignal: 0,
      });
      expect(screen.queryByTestId(selectors.workspace.compare.root)).not.toBeInTheDocument();

      // Bumping the signal flips comparing on so the change is visible.
      rerender(
        <WorkspaceCanvas
          {...baseProps}
          baseUrl="blob:base"
          previewUrl="blob:preview"
          hasSteps
          compareSignal={1}
        />,
      );
      expect(screen.getByTestId(selectors.workspace.compare.root)).toBeInTheDocument();
    });
  });

  describe("progress overlay", () => {
    it("shows the live progress with percent and message", () => {
      render({
        baseUrl: "blob:base",
        progress: { percent: 42, message: "Upscaling…" },
      });
      const overlay = screen.getByTestId(selectors.workspace.canvas.progress);
      expect(overlay).toHaveTextContent("42%");
      expect(overlay).toHaveTextContent("Upscaling…");
    });

    it("omits the message line when blank (message falsy branch)", () => {
      render({ baseUrl: "blob:base", progress: { percent: 10, message: "" } });
      const overlay = screen.getByTestId(selectors.workspace.canvas.progress);
      expect(overlay).toHaveTextContent("10%");
    });

    it("does not show progress when no image is loaded", () => {
      render({ baseUrl: null, progress: { percent: 50, message: "x" } });
      expect(screen.queryByTestId(selectors.workspace.canvas.progress)).not.toBeInTheDocument();
    });
  });

  describe("crop + detection overlays", () => {
    it("seeds the crop overlay from the image natural size on load", () => {
      const onNatural = vi.fn();
      const onChange = vi.fn();
      render({
        baseUrl: "blob:base",
        crop: { rect: { x: 0, y: 0, width: 50, height: 50 }, onChange, onNatural },
      });

      loadImage({ width: 200, height: 100 }, { width: 200, height: 100 });
      expect(onNatural).toHaveBeenCalledWith({ width: 200, height: 100 });
      // After load, the crop box overlay mounts (natural + client are known).
      expect(screen.getByTestId(selectors.workspace.crop.box)).toBeInTheDocument();
    });

    it("renders read-only detection boxes once the image size is known", () => {
      render({
        baseUrl: "blob:base",
        boxes: [{ id: "b0", label: "Hi", box: { x: 5, y: 5, width: 10, height: 10 } }],
      });
      loadImage({ width: 100, height: 100 }, { width: 100, height: 100 });
      expect(screen.getByTestId(selectors.workspace.analyze.overlay)).toBeInTheDocument();
    });
  });
});
