import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { strings } from "../../consts/strings";

// TranscodeTryIt dynamic-imports api/client.uploadFile. We need to mock
// that module so the call doesn't try a real fetch.
const uploadFile = vi.fn();
vi.mock("../../api/client", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/client")>();
  return { ...actual, uploadFile };
});

import { TranscodeTryIt } from "./TranscodeTryIt";

beforeEach(() => {
  Object.defineProperty(URL, "createObjectURL", { value: vi.fn(() => "blob:fake"), writable: true });
  Object.defineProperty(URL, "revokeObjectURL", { value: vi.fn(), writable: true });
  uploadFile.mockResolvedValue({
    ok: true,
    status: 200,
    blob: () => Promise.resolve(new Blob(["transcoded"], { type: "audio/wav" })),
  });
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("TranscodeTryIt", () => {
  it("disables the Transcode action until a file is selected", () => {
    renderWithProviders(<TranscodeTryIt />);
    expect(
      screen.getByRole("button", { name: new RegExp(strings.diagnostics.transcodeAction, "i") }),
    ).toBeDisabled();
  });

  it("on success: posts to /api/v1/audio/transcode and renders the download link", async () => {
    const user = userEvent.setup();
    renderWithProviders(<TranscodeTryIt />);
    const file = new File(["x"], "in.wav", { type: "audio/wav" });
    await user.upload(screen.getByLabelText(strings.diagnostics.audioFileLabel), file);
    await user.click(
      screen.getByRole("button", { name: new RegExp(strings.diagnostics.transcodeAction, "i") }),
    );
    await waitFor(() =>
      expect(uploadFile).toHaveBeenCalledWith("/api/v1/audio/transcode", expect.any(FormData)),
    );
    expect(
      await screen.findByRole("link", { name: new RegExp(strings.diagnostics.transcodeDownload, "i") }),
    ).toBeInTheDocument();
  });

  it("on non-2xx: renders the failed copy key", async () => {
    uploadFile.mockResolvedValue({
      ok: false,
      status: 500,
      blob: () => Promise.resolve(new Blob([])),
    });
    const user = userEvent.setup();
    renderWithProviders(<TranscodeTryIt />);
    const file = new File(["x"], "in.wav", { type: "audio/wav" });
    await user.upload(screen.getByLabelText(strings.diagnostics.audioFileLabel), file);
    await user.click(
      screen.getByRole("button", { name: new RegExp(strings.diagnostics.transcodeAction, "i") }),
    );
    await waitFor(() =>
      expect(screen.getByText(new RegExp(strings.diagnostics.transcodeFailed))).toBeInTheDocument(),
    );
  });
});
