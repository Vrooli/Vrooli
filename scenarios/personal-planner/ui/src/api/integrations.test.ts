import { afterEach, describe, expect, it, vi } from "vitest";

const client = vi.hoisted(() => ({ listConnections: vi.fn(), createFixtureConnection: vi.fn(), syncConnection: vi.fn(), disconnectConnection: vi.fn() }));
vi.mock("@connectrpc/connect", () => ({ createClient: () => client }));

import { createFixtureConnection, disconnectConnection, fetchConnections, syncConnection } from "./integrations";

afterEach(() => vi.clearAllMocks());

describe("integrations API transport", () => {
  it("reads connection state and sends revision-protected actions", async () => {
    const connection = { id: "fixture-1", displayName: "Sample", revision: 3n } as never;
    client.listConnections.mockResolvedValueOnce({ connections: [connection] });
    client.createFixtureConnection.mockResolvedValueOnce({ connection });
    client.syncConnection.mockResolvedValueOnce({ connection });
    client.disconnectConnection.mockResolvedValueOnce({ connection });

    expect(await fetchConnections()).toEqual([connection]);
    expect(await createFixtureConnection("Sample")).toBe(connection);
    expect(await syncConnection(connection)).toBe(connection);
    expect(await disconnectConnection(connection)).toBe(connection);
    expect(client.createFixtureConnection).toHaveBeenCalledWith({ displayName: "Sample" });
    expect(client.syncConnection).toHaveBeenCalledWith({ id: "fixture-1", expectedRevision: 3n });
    expect(client.disconnectConnection).toHaveBeenCalledWith({ id: "fixture-1", expectedRevision: 3n });
  });

  it("rejects mutation responses without a connection", async () => {
    client.createFixtureConnection.mockResolvedValueOnce({});
    client.syncConnection.mockResolvedValueOnce({});
    client.disconnectConnection.mockResolvedValueOnce({});
    const connection = { id: "fixture-1", revision: 1n } as never;
    await expect(createFixtureConnection()).rejects.toThrow("connection was not returned");
    await expect(syncConnection(connection)).rejects.toThrow("connection was not returned");
    await expect(disconnectConnection(connection)).rejects.toThrow("connection was not returned");
  });
});
