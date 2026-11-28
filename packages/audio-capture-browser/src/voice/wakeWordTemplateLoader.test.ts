import { describe, expect, it, vi } from "vitest";
import { loadWakeWordTemplate } from "./wakeWordTemplateLoader";
import type { VoiceCoreServices } from "./services";
import type { AudioFeatures, WakeWordEngine } from "./wakeword";

const feature: AudioFeatures = { kind: "mfcc-v1", data: [[1]], sampleRate: 16_000, durationSec: 0.1 };

function mockServices(configured = true): Pick<VoiceCoreServices, "getWakeWordConfig" | "createWakeWordEngine" | "bytesToFeatures"> {
  return {
    getWakeWordConfig: vi.fn(async () => ({
      configured,
      template: configured ? { samples: [{ audio: new Uint8Array([1]) }], label: "Hey Vrooli", updatedAt: "now" } : null,
    })),
    createWakeWordEngine: vi.fn(() => ({ calibrate: vi.fn(() => ({ kind: "mfcc-v1", mu: 0.1, sigma: 0.01 })) } as unknown as WakeWordEngine)),
    bytesToFeatures: vi.fn(async () => feature),
  };
}

describe("loadWakeWordTemplate", () => {
  it("derives features and calibration using the live threshold", async () => {
    const services = mockServices();
    const onEngine = vi.fn();
    const { engine, template } = await loadWakeWordTemplate(services, null, 0.82, onEngine);

    expect(engine).toBeTruthy();
    expect(onEngine).toHaveBeenCalledWith(engine);
    expect(template).toMatchObject({ samples: [feature], label: "Hey Vrooli", threshold: 0.82, updatedAt: "now" });
    expect(template?.calibration).toEqual({ kind: "mfcc-v1", mu: 0.1, sigma: 0.01 });
  });

  it("keeps the current engine and skips creation when no template is configured", async () => {
    const services = mockServices(false);
    const current = {} as WakeWordEngine;
    const result = await loadWakeWordTemplate(services, current, 0.7, vi.fn());

    expect(result).toEqual({ engine: current, template: null });
    expect(services.createWakeWordEngine).not.toHaveBeenCalled();
  });

  it("keeps the engine but declines an unusable template when every sample fails", async () => {
    const services = mockServices();
    services.bytesToFeatures = vi.fn(async () => { throw new Error("invalid sample"); });
    const result = await loadWakeWordTemplate(services, null, 0.7, vi.fn());

    expect(result.engine).toBeTruthy();
    expect(result.template).toBeNull();
  });
});
