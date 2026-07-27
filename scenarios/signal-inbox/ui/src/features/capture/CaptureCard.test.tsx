import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { selectors } from "../../consts/selectors";
import { renderWithProviders } from "../../test-utils";

const api = vi.hoisted(() => ({
  captureSignal: vi.fn(),
  listSignals: vi.fn(),
  uploadSignalImage: vi.fn(),
}));

vi.mock("../../api/signals", () => ({
  signalsClient: {
    captureSignal: api.captureSignal,
    listSignals: api.listSignals,
  },
  uploadSignalImage: api.uploadSignalImage,
}));

import { CaptureCard } from "./CaptureCard";

describe("CaptureCard [REQ:SIG-P0-002]", () => {
  beforeEach(() => {
    api.captureSignal.mockResolvedValue({ duplicate: false });
    api.listSignals.mockResolvedValue({ signals: [] });
    api.uploadSignalImage.mockReset();
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("shows the empty immutable journal state", async () => {
    renderWithProviders(<CaptureCard />);

    expect(await screen.findByTestId(selectors.capture.empty)).toHaveTextContent("No signals captured yet.");
  });

  it("shows journal loading while the initial read is pending", () => {
    api.listSignals.mockReturnValue(new Promise(() => {}));
    renderWithProviders(<CaptureCard />);

    expect(screen.getByTestId(selectors.capture.loading)).toHaveTextContent("Loading journal…");
  });

  it("shows a recoverable error when the journal read fails", async () => {
    api.listSignals.mockRejectedValue(new Error("offline"));
    renderWithProviders(<CaptureCard />);

    expect(await screen.findByTestId(selectors.capture.error)).toHaveTextContent("Could not load the journal.");
  });

  it("captures text with its optional operator note", async () => {
    const user = userEvent.setup();
    renderWithProviders(<CaptureCard />);

    await user.type(screen.getByTestId(selectors.capture.source), "A useful idea");
    await user.type(screen.getByTestId(selectors.capture.note), "Review on Friday");
    await user.click(screen.getByTestId(selectors.capture.submit));

    await waitFor(() => {
      expect(api.captureSignal).toHaveBeenCalledWith({
        source: { case: "text", value: "A useful idea" },
        captureNote: "Review on Friday",
      });
    });
  });

  it("recognizes an HTTP URL without asking the operator to choose a source kind", async () => {
    const user = userEvent.setup();
    renderWithProviders(<CaptureCard />);

    await user.type(screen.getByTestId(selectors.capture.source), "https://example.com/article");
    await user.click(screen.getByTestId(selectors.capture.submit));

    await waitFor(() => {
      expect(api.captureSignal).toHaveBeenCalledWith({
        source: { case: "url", value: "https://example.com/article" },
        captureNote: "",
      });
    });
  });

  it("uploads an image and captures its payload reference", async () => {
    const user = userEvent.setup();
    api.uploadSignalImage.mockResolvedValue("signals/uploads/image-hash");
    renderWithProviders(<CaptureCard />);

    const image = new File(["image bytes"], "capture.png", { type: "image/png" });
    await user.upload(screen.getByTestId(selectors.capture.image), image);
    await user.click(screen.getByTestId(selectors.capture.submit));

    await waitFor(() => {
      expect(api.uploadSignalImage).toHaveBeenCalledWith(image);
      expect(api.captureSignal).toHaveBeenCalledWith({
        source: { case: "imagePayloadRef", value: "signals/uploads/image-hash" },
        captureNote: "",
      });
    });
  });

  it("surfaces signals that need later attention", async () => {
    api.listSignals.mockResolvedValue({
      signals: [{ id: "signal-1", sourceKind: "image", rawPayloadRef: "signals/uploads/image-hash", needsAttention: true }],
    });
    renderWithProviders(<CaptureCard />);

    expect(await screen.findByTestId(selectors.capture.needsAttention)).toHaveTextContent("Needs attention");
  });
});
