import { describe, it, expect, vi, beforeEach } from "vitest";

const listLocalProvidersRpc = vi.fn();
const startProviderRpc = vi.fn();
const stopProviderRpc = vi.fn();
const restartProviderRpc = vi.fn();
const pullModelRpc = vi.fn();
const getProviderLogsRpc = vi.fn();

vi.mock("../api/client", () => ({ transport: {} }));
vi.mock("@connectrpc/connect", () => ({
  createClient: () => ({
    listLocalProviders: (req: unknown) => listLocalProvidersRpc(req),
    startProvider: (req: unknown, opts: unknown) => startProviderRpc(req, opts),
    stopProvider: (req: unknown, opts: unknown) => stopProviderRpc(req, opts),
    restartProvider: (req: unknown, opts: unknown) => restartProviderRpc(req, opts),
    pullModel: (req: unknown, opts: unknown) => pullModelRpc(req, opts),
    getProviderLogs: (req: unknown, opts: unknown) => getProviderLogsRpc(req, opts),
  }),
}));

import {
  listLocalProviders,
  startProvider,
  stopProvider,
  restartProvider,
  pullModel,
  streamProviderLogs,
} from "./providerLifecycle";

beforeEach(() => {
  vi.clearAllMocks();
});

describe("listLocalProviders", () => {
  it("calls RPC with empty request and returns response", async () => {
    const fake = { providers: [] };
    listLocalProvidersRpc.mockResolvedValueOnce(fake);
    const result = await listLocalProviders();
    expect(listLocalProvidersRpc).toHaveBeenCalledWith({});
    expect(result).toBe(fake);
  });
});

describe("startProvider", () => {
  it("calls RPC without dry-run header by default", async () => {
    startProviderRpc.mockResolvedValueOnce({});
    await startProvider("ollama");
    expect(startProviderRpc).toHaveBeenCalledWith(
      { providerId: "ollama" },
      { headers: undefined },
    );
  });

  it("passes X-Dry-Run header when dryRun=true", async () => {
    startProviderRpc.mockResolvedValueOnce({});
    await startProvider("ollama", true);
    expect(startProviderRpc).toHaveBeenCalledWith(
      { providerId: "ollama" },
      { headers: { "X-Dry-Run": "true" } },
    );
  });

  it("does NOT pass dry-run header when dryRun=false", async () => {
    startProviderRpc.mockResolvedValueOnce({});
    await startProvider("ollama", false);
    expect(startProviderRpc).toHaveBeenCalledWith(
      { providerId: "ollama" },
      { headers: undefined },
    );
  });
});

describe("stopProvider", () => {
  it("passes providerId and no header by default", async () => {
    stopProviderRpc.mockResolvedValueOnce({});
    await stopProvider("whisper");
    expect(stopProviderRpc).toHaveBeenCalledWith(
      { providerId: "whisper" },
      { headers: undefined },
    );
  });

  it("passes dry-run header when requested", async () => {
    stopProviderRpc.mockResolvedValueOnce({});
    await stopProvider("whisper", true);
    expect(stopProviderRpc).toHaveBeenCalledWith(
      { providerId: "whisper" },
      { headers: { "X-Dry-Run": "true" } },
    );
  });
});

describe("restartProvider", () => {
  it("passes providerId without dry-run", async () => {
    restartProviderRpc.mockResolvedValueOnce({});
    await restartProvider("piper");
    expect(restartProviderRpc).toHaveBeenCalledWith(
      { providerId: "piper" },
      { headers: undefined },
    );
  });

  it("passes dry-run header when requested", async () => {
    restartProviderRpc.mockResolvedValueOnce({});
    await restartProvider("piper", true);
    expect(restartProviderRpc).toHaveBeenCalledWith(
      { providerId: "piper" },
      { headers: { "X-Dry-Run": "true" } },
    );
  });
});

describe("pullModel", () => {
  it("calls RPC with ollama providerId and the given model name", async () => {
    pullModelRpc.mockResolvedValueOnce({});
    await pullModel("whisper:large");
    expect(pullModelRpc).toHaveBeenCalledWith(
      { providerId: "ollama", modelName: "whisper:large" },
      { headers: undefined },
    );
  });

  it("includes dry-run header when dryRun=true", async () => {
    pullModelRpc.mockResolvedValueOnce({});
    await pullModel("whisper:large", true);
    expect(pullModelRpc).toHaveBeenCalledWith(
      { providerId: "ollama", modelName: "whisper:large" },
      { headers: { "X-Dry-Run": "true" } },
    );
  });
});

describe("streamProviderLogs", () => {
  it("passes required fields with default follow=false and tailLines=0", () => {
    const fakeIterable = { [Symbol.asyncIterator]: vi.fn() };
    getProviderLogsRpc.mockReturnValueOnce(fakeIterable);
    const ctrl = new AbortController();
    const result = streamProviderLogs({ providerId: "ollama" }, ctrl.signal);
    expect(getProviderLogsRpc).toHaveBeenCalledWith(
      { providerId: "ollama", follow: false, tailLines: 0 },
      { signal: ctrl.signal },
    );
    expect(result).toBe(fakeIterable);
  });

  it("passes explicit follow and tailLines when provided", () => {
    const fakeIterable = { [Symbol.asyncIterator]: vi.fn() };
    getProviderLogsRpc.mockReturnValueOnce(fakeIterable);
    const ctrl = new AbortController();
    streamProviderLogs({ providerId: "piper", follow: true, tailLines: 50 }, ctrl.signal);
    expect(getProviderLogsRpc).toHaveBeenCalledWith(
      { providerId: "piper", follow: true, tailLines: 50 },
      { signal: ctrl.signal },
    );
  });
});
