import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { strings } from "../../consts/strings";

const rpc = vi.hoisted(() => ({
  trim: vi.fn(),
  fade: vi.fn(),
  extractMetadata: vi.fn(),
}));

vi.mock("@connectrpc/connect", () => ({
  createClient: () => rpc,
}));

import { AudioOpsPanel } from "./AudioOpsPanel";

function uploadFile() {
  const file = new File(["x"], "in.wav", { type: "audio/wav" });
  Object.defineProperty(file, "arrayBuffer", { value: () => Promise.resolve(new Uint8Array([120]).buffer) });
  return file;
}

beforeEach(() => {
  Object.defineProperty(URL, "createObjectURL", { value: vi.fn(() => "blob:fake"), writable: true });
  Object.defineProperty(URL, "revokeObjectURL", { value: vi.fn(), writable: true });
  rpc.trim.mockResolvedValue({ audio: new Uint8Array([1]), contentType: "audio/wav" });
  rpc.fade.mockResolvedValue({ audio: new Uint8Array([2]), contentType: "audio/wav" });
  rpc.extractMetadata.mockResolvedValue({
    metadata: {
      durationSeconds: 1.25,
      sampleRate: 16000,
      channels: 1,
      bitrate: 128000,
      codec: "pcm_s16le",
      format: 1,
      tags: { title: "fixture" },
    },
  });
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("AudioOpsPanel", () => {
  it("runs trim with a file and numeric start/end fields", async () => {
    const user = userEvent.setup();
    renderWithProviders(<AudioOpsPanel />);
    await user.selectOptions(
      screen.getByRole("combobox", { name: strings.diagnostics.audioOps.operationLabel }),
      "trim",
    );
    await user.upload(screen.getByLabelText(strings.diagnostics.audioOps.fields.audio), uploadFile());

    const start = screen.getByRole("spinbutton", {
      name: new RegExp(strings.diagnostics.audioOps.fields.start_seconds),
    });
    const end = screen.getByRole("spinbutton", {
      name: new RegExp(strings.diagnostics.audioOps.fields.end_seconds),
    });
    await user.clear(start);
    await user.type(start, "1.5");
    await user.clear(end);
    await user.type(end, "3");

    await user.click(
      screen.getByRole("button", { name: new RegExp(strings.diagnostics.audioOps.runAction, "i") }),
    );

    await waitFor(() =>
      expect(rpc.trim).toHaveBeenCalledWith({
        audio: new Uint8Array([120]),
        format: 1,
        startSeconds: 1.5,
        endSeconds: 3,
      }),
    );
    expect(
      await screen.findByRole("link", { name: new RegExp(strings.diagnostics.audioOps.download, "i") }),
    ).toBeInTheDocument();
  });

  it("runs fade with both numeric windows", async () => {
    const user = userEvent.setup();
    renderWithProviders(<AudioOpsPanel />);
    await user.selectOptions(
      screen.getByRole("combobox", { name: strings.diagnostics.audioOps.operationLabel }),
      "fade",
    );
    await user.upload(screen.getByLabelText(strings.diagnostics.audioOps.fields.audio), uploadFile());

    const fadeIn = screen.getByRole("spinbutton", {
      name: new RegExp(strings.diagnostics.audioOps.fields.fade_in_seconds),
    });
    const fadeOut = screen.getByRole("spinbutton", {
      name: new RegExp(strings.diagnostics.audioOps.fields.fade_out_seconds),
    });
    await user.clear(fadeIn);
    await user.type(fadeIn, "0.5");
    await user.clear(fadeOut);
    await user.type(fadeOut, "1.5");

    await user.click(
      screen.getByRole("button", { name: new RegExp(strings.diagnostics.audioOps.runAction, "i") }),
    );

    await waitFor(() =>
      expect(rpc.fade).toHaveBeenCalledWith({
        audio: new Uint8Array([120]),
        format: 1,
        fadeInSeconds: 0.5,
        fadeOutSeconds: 1.5,
        outputFormat: 1,
      }),
    );
  });

  it("renders extractMetadata as readable JSON instead of a download", async () => {
    const user = userEvent.setup();
    renderWithProviders(<AudioOpsPanel />);
    await user.selectOptions(
      screen.getByRole("combobox", { name: strings.diagnostics.audioOps.operationLabel }),
      "extract_metadata",
    );
    await user.upload(screen.getByLabelText(strings.diagnostics.audioOps.fields.audio), uploadFile());

    await user.click(
      screen.getByRole("button", { name: new RegExp(strings.diagnostics.audioOps.runAction, "i") }),
    );

    await waitFor(() => expect(rpc.extractMetadata).toHaveBeenCalled());
    expect(await screen.findByText(/"codec": "pcm_s16le"/)).toBeInTheDocument();
    expect(screen.getByText(/"sampleRate": 16000/)).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: new RegExp(strings.diagnostics.audioOps.download, "i") })).toBeNull();
  });
});
