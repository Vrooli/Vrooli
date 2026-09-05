import { afterEach, describe, expect, it, vi } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, renderHook } from "@testing-library/react";
import { create, toJson } from "@bufbuild/protobuf";
import {
  EventSchema,
  EventType,
} from "@vrooli/proto-types/device-sync-hub/v1/realtime/realtime_pb";
import type { ReactNode } from "react";

import { useRealtime } from "./useRealtime";
import { ITEMS_QUERY_KEY } from "../transfer/queries";

const eventJson = (init: Parameters<typeof create<typeof EventSchema>>[1]) =>
  JSON.stringify(toJson(EventSchema, create(EventSchema, init), { useProtoFieldName: true }));

interface MockSource {
  url: string;
  readyState: number;
  CLOSED: number;
  onopen: (() => void) | null;
  onerror: (() => void) | null;
  onmessage: ((ev: { data: string }) => void) | null;
  close: () => void;
  emit: (data: string) => void;
}

function lastSource(): MockSource {
  const instances = (globalThis.EventSource as unknown as { instances: MockSource[] }).instances;
  return instances[instances.length - 1]!;
}

function wrapper(client: QueryClient) {
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
}

describe("useRealtime", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("stays closed and opens no EventSource when unpaired", () => {
    const client = new QueryClient();
    const { result } = renderHook(() => useRealtime(null), { wrapper: wrapper(client) });
    expect(result.current.status).toBe("closed");
  });

  it("connects with the token in the query param and tracks open/error states", () => {
    const client = new QueryClient();
    const { result } = renderHook(() => useRealtime("tok-1"), { wrapper: wrapper(client) });

    const source = lastSource();
    expect(source.url).toContain("token=tok-1");
    expect(result.current.status).toBe("connecting");

    act(() => source.onopen?.());
    expect(result.current.status).toBe("open");

    // Auto-reconnecting (not CLOSED) surfaces as connecting.
    source.readyState = 0;
    act(() => source.onerror?.());
    expect(result.current.status).toBe("connecting");

    // Explicitly closed surfaces as closed.
    source.readyState = source.CLOSED;
    act(() => source.onerror?.());
    expect(result.current.status).toBe("closed");
  });

  it("invalidates the items query on ITEM_ARRIVED events", () => {
    const client = new QueryClient();
    const invalidate = vi.spyOn(client, "invalidateQueries");
    renderHook(() => useRealtime("tok-1"), { wrapper: wrapper(client) });

    act(() => lastSource().emit(eventJson({ type: EventType.ITEM_ARRIVED })));
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ITEMS_QUERY_KEY });
  });

  it("routes pairing requests into reducer state and supports dismissal", () => {
    const client = new QueryClient();
    const { result } = renderHook(() => useRealtime("tok-1"), { wrapper: wrapper(client) });

    act(() =>
      lastSource().emit(
        eventJson({ type: EventType.PAIRING_REQUESTED, pairing: { deviceId: "d9", name: "Phone" } }),
      ),
    );
    expect(result.current.pendingPairing?.deviceId).toBe("d9");

    act(() => result.current.dismissPairing());
    expect(result.current.pendingPairing).toBeNull();
  });

  it("ignores unparseable stream lines", () => {
    const client = new QueryClient();
    const invalidate = vi.spyOn(client, "invalidateQueries");
    const { result } = renderHook(() => useRealtime("tok-1"), { wrapper: wrapper(client) });

    act(() => lastSource().emit("not-json"));
    expect(result.current.pendingPairing).toBeNull();
    expect(invalidate).not.toHaveBeenCalled();
  });

  it("closes the source on unmount", () => {
    const client = new QueryClient();
    const { unmount } = renderHook(() => useRealtime("tok-1"), { wrapper: wrapper(client) });
    const source = lastSource();
    const closeSpy = vi.spyOn(source, "close");
    unmount();
    expect(closeSpy).toHaveBeenCalled();
  });
});
