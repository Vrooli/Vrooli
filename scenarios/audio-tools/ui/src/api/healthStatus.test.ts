import { describe, it, expect, vi, beforeEach } from "vitest";

const getProviderHealthRpc = vi.fn();
const refreshProviderHealthRpc = vi.fn();
const streamProviderHealthRpc = vi.fn();

vi.mock("../api/client", () => ({ transport: {} }));
vi.mock("@connectrpc/connect", () => ({
  createClient: () => ({
    getProviderHealth: (req: unknown) => getProviderHealthRpc(req),
    refreshProviderHealth: (req: unknown) => refreshProviderHealthRpc(req),
    streamProviderHealth: (req: unknown, opts: unknown) => streamProviderHealthRpc(req, opts),
  }),
}));

import {
  getProviderHealth,
  refreshProviderHealth,
  streamProviderHealth,
} from "./healthStatus";

beforeEach(() => {
  vi.clearAllMocks();
});

describe("getProviderHealth", () => {
  it("calls the RPC with an empty request and returns the response", async () => {
    const fake = { providers: [] };
    getProviderHealthRpc.mockResolvedValueOnce(fake);
    const result = await getProviderHealth();
    expect(getProviderHealthRpc).toHaveBeenCalledWith({});
    expect(result).toBe(fake);
  });

  it("propagates RPC errors", async () => {
    getProviderHealthRpc.mockRejectedValueOnce(new Error("rpc-fail"));
    await expect(getProviderHealth()).rejects.toThrow("rpc-fail");
  });
});

describe("refreshProviderHealth", () => {
  it("calls the RPC and returns the refresh response", async () => {
    const fake = { refreshed: true };
    refreshProviderHealthRpc.mockResolvedValueOnce(fake);
    const result = await refreshProviderHealth();
    expect(refreshProviderHealthRpc).toHaveBeenCalledWith({});
    expect(result).toBe(fake);
  });
});

describe("streamProviderHealth", () => {
  it("passes the AbortSignal through and returns the iterable from the RPC", () => {
    const fakeIterable = { [Symbol.asyncIterator]: vi.fn() };
    streamProviderHealthRpc.mockReturnValueOnce(fakeIterable);
    const ctrl = new AbortController();
    const result = streamProviderHealth(ctrl.signal);
    expect(streamProviderHealthRpc).toHaveBeenCalledWith({}, { signal: ctrl.signal });
    expect(result).toBe(fakeIterable);
  });
});
