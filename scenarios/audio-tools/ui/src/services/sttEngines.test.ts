import { describe, it, expect, vi, beforeEach } from "vitest";

const listEnginesRpc = vi.fn();
const getEngineSwitchImpactRpc = vi.fn();
const updateStreamConfigRpc = vi.fn();

vi.mock("@connectrpc/connect", () => ({
  createClient: () => ({
    listEngines: (req: unknown) => listEnginesRpc(req),
    getEngineSwitchImpact: (req: unknown) => getEngineSwitchImpactRpc(req),
    updateStreamConfig: (req: unknown) => updateStreamConfigRpc(req),
  }),
}));

import { listEngines, getEngineSwitchImpact, setEngine } from "./sttEngines";

beforeEach(() => {
  vi.clearAllMocks();
});

describe("listEngines", () => {
  it("decodes the engine list", async () => {
    listEnginesRpc.mockResolvedValue({
      engines: [
        {
          id: "whisper-local",
          displayName: "Whisper (local)",
          kind: "local_resource",
          available: true,
          nativeStreaming: false,
          isActive: true,
        },
      ],
    });
    const out = await listEngines();
    expect(listEnginesRpc).toHaveBeenCalledWith({});
    expect(out).toEqual([
      {
        id: "whisper-local",
        displayName: "Whisper (local)",
        kind: "local_resource",
        available: true,
        nativeStreaming: false,
        isActive: true,
      },
    ]);
  });

  it("returns an empty array when there are no engines", async () => {
    listEnginesRpc.mockResolvedValue({ engines: [] });
    expect(await listEngines()).toEqual([]);
  });
});

describe("getEngineSwitchImpact", () => {
  it("decodes resource impact and consumers", async () => {
    getEngineSwitchImpactRpc.mockResolvedValue({
      resource: "whisper",
      consumers: [{ scenario: "dictation", displayName: "Dictation", required: true }],
      safeToStop: false,
      stopCommand: "vrooli resource stop whisper",
      consumersKnown: true,
    });
    const out = await getEngineSwitchImpact("whisper-local");
    expect(getEngineSwitchImpactRpc).toHaveBeenCalledWith({ fromEngineId: "whisper-local" });
    expect(out.resource).toBe("whisper");
    expect(out.consumers[0]!.scenario).toBe("dictation");
    expect(out.safeToStop).toBe(false);
    expect(out.consumersKnown).toBe(true);
  });

  it("handles an engine with no backing resource", async () => {
    getEngineSwitchImpactRpc.mockResolvedValue({
      resource: "",
      consumers: [],
      safeToStop: true,
      stopCommand: "",
      consumersKnown: false,
    });
    const out = await getEngineSwitchImpact("byok-openai");
    expect(out.resource).toBe("");
    expect(out.consumers).toEqual([]);
    expect(out.safeToStop).toBe(true);
  });
});

describe("setEngine", () => {
  it("persists the engine id through a masked stream-config update", async () => {
    updateStreamConfigRpc.mockResolvedValue({});
    await setEngine("whisper-local");
    const req = updateStreamConfigRpc.mock.calls[0]![0];
    expect(req.updateMask.paths).toEqual(["engine_id"]);
    expect(req.config).toEqual({ engineId: "whisper-local" });
  });
});
