import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  client: {
    listModels: vi.fn(),
    getModel: vi.fn(),
    listOperations: vi.fn(),
    selectModel: vi.fn(),
    setModelEnabled: vi.fn(),
    listBlocklist: vi.fn(),
  },
}));

vi.mock("@connectrpc/connect", () => ({
  createClient: vi.fn(() => mocks.client),
}));

describe("api/models", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    vi.clearAllMocks();
    vi.unstubAllGlobals();
  });

  it("exports the generated Connect client and forwards listModels", async () => {
    const { modelsClient } = await import("./models");

    await modelsClient.listModels({});

    expect(mocks.client.listModels).toHaveBeenCalledWith({});
  });

  it("forwards an operation filter to listModels", async () => {
    const { modelsClient } = await import("./models");

    await modelsClient.listModels({ operation: "upscale" });

    expect(mocks.client.listModels).toHaveBeenCalledWith({ operation: "upscale" });
  });

  it("forwards listOperations", async () => {
    const { modelsClient } = await import("./models");

    await modelsClient.listOperations({});

    expect(mocks.client.listOperations).toHaveBeenCalledWith({});
  });

  it("forwards setModelEnabled with id + enabled", async () => {
    const { modelsClient } = await import("./models");

    await modelsClient.setModelEnabled({ id: "m-1", enabled: false });

    expect(mocks.client.setModelEnabled).toHaveBeenCalledWith({ id: "m-1", enabled: false });
  });
});
