import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { PropsWithChildren } from "react";
import React from "react";

const disconnectDevice = vi.fn();
const giveControl = vi.fn();
const listDevices = vi.fn();
const renameOwnDevice = vi.fn();

vi.mock("../api/devices", () => ({
  disconnectDevice: (...args: unknown[]) => disconnectDevice(...args),
  giveControl: (...args: unknown[]) => giveControl(...args),
  listDevices: (...args: unknown[]) => listDevices(...args),
  renameOwnDevice: (...args: unknown[]) => renameOwnDevice(...args),
}));

import { DEVICES_QUERY_KEY, useDeviceMutations, useDevices } from "./useDevices";

function makeWrapper(): { wrapper: (props: PropsWithChildren) => React.ReactElement; client: QueryClient } {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  const wrapper = ({ children }: PropsWithChildren) =>
    React.createElement(QueryClientProvider, { client }, children);
  return { wrapper, client };
}

describe("useDevices", () => {
  beforeEach(() => {
    disconnectDevice.mockReset().mockResolvedValue(undefined);
    giveControl.mockReset().mockResolvedValue(undefined);
    listDevices.mockReset().mockResolvedValue([{ deviceId: "d1" }]);
    renameOwnDevice.mockReset();
  });

  afterEach(() => vi.clearAllMocks());

  it("queries the device roster only when enabled", async () => {
    const { wrapper } = makeWrapper();
    const { rerender, result } = renderHook(({ enabled }) => useDevices(enabled), {
      wrapper,
      initialProps: { enabled: false },
    });
    expect(listDevices).not.toHaveBeenCalled();
    expect(result.current.data).toBeUndefined();

    rerender({ enabled: true });
    await waitFor(() => expect(result.current.data).toEqual([{ deviceId: "d1" }]));
    expect(listDevices).toHaveBeenCalledTimes(1);
  });
});

describe("useDeviceMutations", () => {
  beforeEach(() => {
    disconnectDevice.mockReset().mockResolvedValue(undefined);
    giveControl.mockReset().mockResolvedValue(undefined);
    listDevices.mockReset().mockResolvedValue([]);
    renameOwnDevice.mockReset();
  });

  afterEach(() => vi.clearAllMocks());

  it("forwards disconnect, giveControl, and rename and invalidates the roster on success", async () => {
    const { wrapper, client } = makeWrapper();
    const invalidate = vi.spyOn(client, "invalidateQueries");
    const { result } = renderHook(() => useDeviceMutations(), { wrapper });

    await result.current.disconnect.mutateAsync({ deviceId: "d1", connectionId: "c1" });
    expect(disconnectDevice).toHaveBeenCalledWith("d1", "c1");

    await result.current.giveControl.mutateAsync({ deviceId: "d1", sessionId: "s1" });
    expect(giveControl).toHaveBeenCalledWith("d1", "s1");

    await result.current.rename.mutateAsync("Laptop");
    expect(renameOwnDevice).toHaveBeenCalledWith("Laptop");

    expect(invalidate).toHaveBeenCalledWith({ queryKey: DEVICES_QUERY_KEY });
    expect(invalidate).toHaveBeenCalledTimes(3);
  });
});
