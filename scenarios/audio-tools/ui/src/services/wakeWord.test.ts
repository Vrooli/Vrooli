import { describe, it, expect, vi, beforeEach } from "vitest";

import { AudioFormat } from "@vrooli/proto-types/audio-tools/v1/common/common_pb";

const getWakeWordConfigRpc = vi.fn();
const updateWakeWordTemplateRpc = vi.fn();
const deleteWakeWordTemplateRpc = vi.fn();

vi.mock("@connectrpc/connect", () => ({
  createClient: () => ({
    getWakeWordConfig: (req: unknown) => getWakeWordConfigRpc(req),
    updateWakeWordTemplate: (req: unknown) => updateWakeWordTemplateRpc(req),
    deleteWakeWordTemplate: (req: unknown) => deleteWakeWordTemplateRpc(req),
  }),
}));

import {
  getWakeWordConfig,
  saveWakeWordTemplate,
  deleteWakeWordTemplate,
} from "./wakeWord";

beforeEach(() => {
  vi.clearAllMocks();
});

describe("getWakeWordConfig", () => {
  it("decodes a configured template with samples", async () => {
    getWakeWordConfigRpc.mockResolvedValue({
      config: {
        configured: true,
        template: {
          label: "hey-vrooli",
          threshold: 0.8,
          samples: [
            { audio: new Uint8Array([1]), format: AudioFormat.WAV, sampleRateHz: 16_000 },
          ],
        },
      },
    });
    const out = await getWakeWordConfig();
    expect(out.configured).toBe(true);
    expect(out.template?.label).toBe("hey-vrooli");
    expect(out.template?.samples[0]!.sampleRateHz).toBe(16_000);
    expect(out.template?.samples[0]!.format).toBe(AudioFormat.WAV);
  });

  it("returns not-configured when the config is absent", async () => {
    getWakeWordConfigRpc.mockResolvedValue({});
    const out = await getWakeWordConfig();
    expect(out).toEqual({ configured: false });
  });

  it("tolerates a config with no template", async () => {
    getWakeWordConfigRpc.mockResolvedValue({ config: { configured: false } });
    const out = await getWakeWordConfig();
    expect(out.configured).toBe(false);
    expect(out.template).toBeUndefined();
  });

  it("defaults sample fields when partially populated", async () => {
    getWakeWordConfigRpc.mockResolvedValue({
      config: { configured: true, template: { samples: [{}] } },
    });
    const out = await getWakeWordConfig();
    expect(out.template?.label).toBe("");
    expect(out.template?.threshold).toBe(0);
    expect(Array.from(out.template!.samples[0]!.audio)).toEqual([]);
    expect(out.template?.samples[0]!.format).toBe(AudioFormat.UNSPECIFIED);
  });

  it("defaults to an empty sample list when template has none", async () => {
    getWakeWordConfigRpc.mockResolvedValue({
      config: { configured: true, template: { label: "x", threshold: 0.5 } },
    });
    const out = await getWakeWordConfig();
    expect(out.template?.samples).toEqual([]);
  });
});

describe("saveWakeWordTemplate", () => {
  it("encodes the template and returns the decoded config", async () => {
    updateWakeWordTemplateRpc.mockResolvedValue({
      config: { configured: true, template: { label: "hi", threshold: 0.7, samples: [] } },
    });
    const out = await saveWakeWordTemplate({
      label: "hi",
      threshold: 0.7,
      samples: [{ audio: new Uint8Array([9]), format: AudioFormat.PCM_S16LE, sampleRateHz: 8_000 }],
    });
    const req = updateWakeWordTemplateRpc.mock.calls[0]![0];
    expect(req.template.label).toBe("hi");
    expect(req.template.samples[0].sampleRateHz).toBe(8_000);
    expect(out.configured).toBe(true);
  });
});

describe("deleteWakeWordTemplate", () => {
  it("clears the template and returns the decoded config", async () => {
    deleteWakeWordTemplateRpc.mockResolvedValue({ config: { configured: false } });
    const out = await deleteWakeWordTemplate();
    expect(deleteWakeWordTemplateRpc).toHaveBeenCalledWith({});
    expect(out.configured).toBe(false);
  });
});
