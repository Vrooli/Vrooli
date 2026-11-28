import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { strings } from "../../consts/strings";
import { audioOps } from "./audioOps.generated";

// jsdom's File shim does not implement Blob.arrayBuffer, while browsers do.
// userEvent may clone uploaded files, so an own-property stub on one File is
// not enough for multi-file journeys.
if (typeof File.prototype.arrayBuffer !== "function") {
  Object.defineProperty(File.prototype, "arrayBuffer", {
    configurable: true,
    value: function arrayBuffer(): Promise<ArrayBuffer> {
      const bytes = new ArrayBuffer(1);
      new Uint8Array(bytes)[0] = 120;
      return Promise.resolve(bytes);
    },
  });
}

const rpc = vi.hoisted(() => ({
  trim: vi.fn(),
  fade: vi.fn(),
  extractMetadata: vi.fn(),
  merge: vi.fn(),
  split: vi.fn(),
  volume: vi.fn(),
  normalize: vi.fn(),
}));

vi.mock("@connectrpc/connect", () => ({
  createClient: () => rpc,
}));

import { AudioOpsPanel } from "./AudioOpsPanel";

function uploadFile() {
  return new File(["x"], "in.wav", { type: "audio/wav" });
}

beforeEach(() => {
  Object.defineProperty(URL, "createObjectURL", { value: vi.fn(() => "blob:fake"), writable: true });
  Object.defineProperty(URL, "revokeObjectURL", { value: vi.fn(), writable: true });
  rpc.trim.mockResolvedValue({ audio: new Uint8Array([1]), contentType: "audio/wav" });
  rpc.fade.mockResolvedValue({ audio: new Uint8Array([2]), contentType: "audio/wav" });
  rpc.merge.mockResolvedValue({ audio: new Uint8Array([3]), contentType: "audio/wav" });
  rpc.split.mockResolvedValue({ chunks: [
    { audio: new Uint8Array([4]), contentType: "audio/wav", startSeconds: 0, durationSeconds: 1 },
    { audio: new Uint8Array([5]), contentType: "audio/wav", startSeconds: 1, durationSeconds: 1 },
  ] });
  rpc.volume.mockResolvedValue({ audio: new Uint8Array([6]), contentType: "audio/wav" });
  rpc.normalize.mockResolvedValue({ audio: new Uint8Array([7]), contentType: "audio/wav" });
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
  it("keeps every declared transform in the exercised UI surface", () => {
    expect(audioOps.map((op) => op.id)).toEqual([
      "transcode", "trim", "merge", "split", "fade", "volume", "normalize", "extract_metadata",
    ]);
  });

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

  it("runs merge with multiple uploaded sources and crossfade", async () => {
    const user = userEvent.setup();
    renderWithProviders(<AudioOpsPanel />);
    await user.selectOptions(screen.getByRole("combobox", { name: strings.diagnostics.audioOps.operationLabel }), "merge");
    const files = [uploadFile(), uploadFile()];
    await user.upload(screen.getByLabelText(strings.diagnostics.audioOps.fields.sources), files);
    const crossfade = screen.getByRole("spinbutton", { name: new RegExp(strings.diagnostics.audioOps.fields.crossfade_seconds) });
    await user.clear(crossfade);
    await user.type(crossfade, "0.25");
    await user.click(screen.getByRole("button", { name: new RegExp(strings.diagnostics.audioOps.runAction, "i") }));
    await waitFor(() => expect(rpc.merge).toHaveBeenCalledWith({
      sources: [{ audio: new Uint8Array([120]), format: 0 }, { audio: new Uint8Array([120]), format: 0 }],
      outputFormat: 1,
      crossfadeSeconds: 0.25,
    }));
    expect(await screen.findByRole("link", { name: new RegExp(strings.diagnostics.audioOps.download, "i") })).toBeInTheDocument();
  });

  it("runs split with boundaries and renders one download for each returned chunk", async () => {
    const user = userEvent.setup();
    renderWithProviders(<AudioOpsPanel />);
    await user.selectOptions(screen.getByRole("combobox", { name: strings.diagnostics.audioOps.operationLabel }), "split");
    await user.upload(screen.getByLabelText(strings.diagnostics.audioOps.fields.audio), uploadFile());
    await user.click(screen.getByText(strings.diagnostics.audioOps.advanced));
    await user.type(screen.getByLabelText(strings.diagnostics.audioOps.fields.boundaries_seconds), "1, 2");
    await user.click(screen.getByRole("button", { name: new RegExp(strings.diagnostics.audioOps.runAction, "i") }));
    await waitFor(() => expect(rpc.split).toHaveBeenCalledWith({
      audio: new Uint8Array([120]), format: 1, chunkSeconds: 0, boundariesSeconds: [1, 2], outputFormat: 1,
    }));
    expect(await screen.findAllByRole("link", { name: new RegExp(strings.diagnostics.audioOps.download, "i") })).toHaveLength(2);
  });

  it("requires a split strategy before enabling Run", async () => {
    const user = userEvent.setup();
    renderWithProviders(<AudioOpsPanel />);
    await user.selectOptions(screen.getByRole("combobox", { name: strings.diagnostics.audioOps.operationLabel }), "split");
    await user.upload(screen.getByLabelText(strings.diagnostics.audioOps.fields.audio), uploadFile());
    const run = screen.getByRole("button", { name: new RegExp(strings.diagnostics.audioOps.runAction, "i") });
    expect(run).toBeDisabled();

    await user.click(screen.getByText(strings.diagnostics.audioOps.advanced));
    await user.type(screen.getByLabelText(strings.diagnostics.audioOps.fields.boundaries_seconds), "1, 2");
    expect(run).toBeEnabled();
    expect(rpc.split).not.toHaveBeenCalled();
  });

  it("runs volume with the requested gain", async () => {
    const user = userEvent.setup();
    renderWithProviders(<AudioOpsPanel />);
    await user.selectOptions(screen.getByRole("combobox", { name: strings.diagnostics.audioOps.operationLabel }), "volume");
    await user.upload(screen.getByLabelText(strings.diagnostics.audioOps.fields.audio), uploadFile());
    await user.clear(screen.getByRole("spinbutton", { name: new RegExp(strings.diagnostics.audioOps.fields.gain_db) }));
    await user.type(screen.getByRole("spinbutton", { name: new RegExp(strings.diagnostics.audioOps.fields.gain_db) }), "3.5");
    await user.click(screen.getByRole("button", { name: new RegExp(strings.diagnostics.audioOps.runAction, "i") }));
    await waitFor(() => expect(rpc.volume).toHaveBeenCalledWith({
      audio: new Uint8Array([120]), format: 1, gainDb: 3.5, outputFormat: 1,
    }));
  });

  it("runs normalize with method and target LUFS", async () => {
    const user = userEvent.setup();
    renderWithProviders(<AudioOpsPanel />);
    await user.selectOptions(screen.getByRole("combobox", { name: strings.diagnostics.audioOps.operationLabel }), "normalize");
    await user.upload(screen.getByLabelText(strings.diagnostics.audioOps.fields.audio), uploadFile());
    await user.selectOptions(screen.getByLabelText(strings.diagnostics.audioOps.fields.method), "peak");
    await user.click(screen.getByText(strings.diagnostics.audioOps.advanced));
    const target = screen.getByRole("spinbutton", { name: new RegExp(strings.diagnostics.audioOps.fields.target_lufs) });
    await user.clear(target);
    await user.type(target, "-14");
    await user.click(screen.getByRole("button", { name: new RegExp(strings.diagnostics.audioOps.runAction, "i") }));
    await waitFor(() => expect(rpc.normalize).toHaveBeenCalledWith({
      audio: new Uint8Array([120]), format: 1, method: 3, targetLufs: -14, outputFormat: 1,
    }));
  });
});
