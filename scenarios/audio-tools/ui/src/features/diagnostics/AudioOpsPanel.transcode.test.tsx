import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { strings } from "../../consts/strings";

const rpc = vi.hoisted(() => ({
  transcode: vi.fn(),
}));

vi.mock("@connectrpc/connect", () => ({
  createClient: () => rpc,
}));

import { AudioOpsPanel } from "./AudioOpsPanel";

beforeEach(() => {
  Object.defineProperty(URL, "createObjectURL", { value: vi.fn(() => "blob:fake"), writable: true });
  Object.defineProperty(URL, "revokeObjectURL", { value: vi.fn(), writable: true });
  rpc.transcode.mockResolvedValue({
    audio: new Uint8Array([1, 2]),
    contentType: "audio/wav",
  });
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("AudioOpsPanel (transcode)", () => {
  it("starts on the transcode operation and disables Run until a file is selected", () => {
    renderWithProviders(<AudioOpsPanel initialOperationId="transcode" />);
    expect(
      screen.getByRole("combobox", { name: strings.diagnostics.audioOps.operationLabel }),
    ).toHaveValue("transcode");
    expect(
      screen.getByRole("button", { name: new RegExp(strings.diagnostics.audioOps.runAction, "i") }),
    ).toBeDisabled();
  });

  it("sends file bytes through the matching Connect method and renders the download link", async () => {
    const user = userEvent.setup();
    renderWithProviders(<AudioOpsPanel initialOperationId="transcode" />);
    const file = new File(["x"], "in.wav", { type: "audio/wav" });
    Object.defineProperty(file, "arrayBuffer", { value: () => Promise.resolve(new Uint8Array([120]).buffer) });
    await user.upload(screen.getByLabelText(strings.diagnostics.audioOps.fields.audio), file);
    await user.click(
      screen.getByRole("button", { name: new RegExp(strings.diagnostics.audioOps.runAction, "i") }),
    );
    await waitFor(() =>
      expect(rpc.transcode).toHaveBeenCalledWith({
        audio: new Uint8Array([120]),
        inputFormat: 0,
        outputFormat: 1,
        sampleRate: 0,
        channels: 0,
        bitrate: 0,
      }),
    );
    expect(
      await screen.findByRole("link", { name: new RegExp(strings.diagnostics.audioOps.download, "i") }),
    ).toBeInTheDocument();
  });

  it("on Connect failure: renders the error envelope", async () => {
    rpc.transcode.mockRejectedValue(new Error("connect failed"));
    const user = userEvent.setup();
    renderWithProviders(<AudioOpsPanel initialOperationId="transcode" />);
    const file = new File(["x"], "in.wav", { type: "audio/wav" });
    Object.defineProperty(file, "arrayBuffer", { value: () => Promise.resolve(new Uint8Array([120]).buffer) });
    await user.upload(screen.getByLabelText(strings.diagnostics.audioOps.fields.audio), file);
    await user.click(
      screen.getByRole("button", { name: new RegExp(strings.diagnostics.audioOps.runAction, "i") }),
    );
    expect(await screen.findByText(/connect failed/)).toBeInTheDocument();
  });
});
