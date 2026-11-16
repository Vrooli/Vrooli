import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { strings } from "../../consts/strings";

const transcodeRpc = vi.fn();
vi.mock("@connectrpc/connect", () => ({
  createClient: () => ({ transcode: (request: unknown) => transcodeRpc(request) }),
}));

import { TranscodeTryIt } from "./TranscodeTryIt";

beforeEach(() => {
  Object.defineProperty(URL, "createObjectURL", { value: vi.fn(() => "blob:fake"), writable: true });
  Object.defineProperty(URL, "revokeObjectURL", { value: vi.fn(), writable: true });
  transcodeRpc.mockResolvedValue({
    audio: new Uint8Array([1, 2]),
    contentType: "audio/wav",
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

  it("sends file bytes through Connect and renders the download link", async () => {
    const user = userEvent.setup();
    renderWithProviders(<TranscodeTryIt />);
    const file = new File(["x"], "in.wav", { type: "audio/wav" });
    Object.defineProperty(file, "arrayBuffer", { value: async () => new Uint8Array([120]).buffer });
    await user.upload(screen.getByLabelText(strings.diagnostics.audioFileLabel), file);
    await user.click(
      screen.getByRole("button", { name: new RegExp(strings.diagnostics.transcodeAction, "i") }),
    );
    await waitFor(() => expect(transcodeRpc).toHaveBeenCalledWith({
      audio: new Uint8Array([120]), inputFormat: 0, outputFormat: 1, sampleRate: 0, channels: 0, bitrate: 0,
    }));
    expect(
      await screen.findByRole("link", { name: new RegExp(strings.diagnostics.transcodeDownload, "i") }),
    ).toBeInTheDocument();
  });

  it("on Connect failure: renders the failed copy key", async () => {
    transcodeRpc.mockRejectedValue(new Error("connect failed"));
    const user = userEvent.setup();
    renderWithProviders(<TranscodeTryIt />);
    const file = new File(["x"], "in.wav", { type: "audio/wav" });
    Object.defineProperty(file, "arrayBuffer", { value: async () => new Uint8Array([120]).buffer });
    await user.upload(screen.getByLabelText(strings.diagnostics.audioFileLabel), file);
    await user.click(
      screen.getByRole("button", { name: new RegExp(strings.diagnostics.transcodeAction, "i") }),
    );
    await waitFor(() =>
      expect(screen.getByText(new RegExp(strings.diagnostics.transcodeFailed))).toBeInTheDocument(),
    );
  });
});
